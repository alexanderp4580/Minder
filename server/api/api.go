package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/alexanderp4580/Minder/server/conversation"
	"github.com/alexanderp4580/Minder/server/domain"
)

type Authenticator interface {
	Authenticate(*http.Request) error
}

type Queries interface {
	History(int64, int) ([]domain.Entry, error)
	Events(int64, int) ([]domain.Entry, error)
	ItemsBetween(time.Time, time.Time) ([]domain.Item, error)
}
type Commands interface {
	Confirm(string) (domain.Entry, error)
	Refresh(string) (domain.Entry, error)
	Discard(string) (domain.Entry, error)
	Undo(string) (domain.Entry, error)
}
type Messages interface {
	Accept(string) (conversation.Accepted, error)
}
type ReminderActions interface {
	Act(string, string, time.Time) (domain.Entry, error)
	Active() ([]domain.Entry, error)
	RegisterDevice(string, string) error
}
type Server struct {
	// Branding is the folder that holds the brand guide. Empty turns /branding off.
	Branding     string
	Reminders    ReminderActions
	PushConfig   *PushConfig
	PingInterval time.Duration
	queries      Queries
	commands     Commands
	messages     Messages
	verifier     Authenticator
	web          http.Handler
	public       *http.ServeMux // reachable without sign-in
	private      *http.ServeMux // reachable after sign-in
}

func New(s Queries, c Commands, messages Messages, reminders ReminderActions, v Authenticator, web string) *Server {
	srv := &Server{queries: s, commands: c, messages: messages, Reminders: reminders, verifier: v, PingInterval: 20 * time.Second}
	if web != "" {
		srv.web = http.FileServer(http.Dir(web))
	}
	srv.public = http.NewServeMux()
	srv.public.HandleFunc("GET /v1/health", func(w http.ResponseWriter, r *http.Request) {
		write(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	srv.public.HandleFunc("GET "+logoPath, serveLogo)
	m := http.NewServeMux()
	m.HandleFunc("GET /v1/chat", srv.chat)
	m.HandleFunc("GET /v1/items", srv.items)
	m.HandleFunc("GET /v1/events", srv.events)
	m.HandleFunc("POST /v1/chat/messages", srv.message)
	m.HandleFunc("POST /v1/changes/{id}/confirm", srv.confirm)
	m.HandleFunc("POST /v1/changes/{id}/discard", srv.discard)
	m.HandleFunc("POST /v1/undo/{id}", srv.undo)
	m.HandleFunc("GET /v1/reminders", srv.reminders)
	m.HandleFunc("POST /v1/reminders/{id}/{action}", srv.reminderAction)
	m.HandleFunc("GET /v1/push/config", srv.pushConfig)
	m.HandleFunc("POST /v1/push/devices", srv.registerDevice)
	m.HandleFunc("GET /branding", srv.serveBranding)
	m.HandleFunc("GET /branding/", srv.serveBranding)
	m.HandleFunc("/v1/", http.NotFound) // an unknown API path never falls through to the web app
	m.HandleFunc("/", srv.serveWeb)
	srv.private = m
	return srv
}

func write(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
func fail(w http.ResponseWriter, err error) {
	status := 500
	code := domain.ErrorCode(err)
	noteError(w, err)
	note(w, "error_code", code)
	switch code {
	case "invalid_field":
		status = 400
	case "conflict":
		status = 409
	case "not_found":
		status = 404
	case "limit_reached", "busy":
		status = 429
	case "model_error":
		status = 502
	}
	body := map[string]any{"code": code, "message": domain.Describe(err)}
	var invalid *domain.Invalid
	if errors.As(err, &invalid) {
		body["field"] = invalid.Field
	}
	write(w, status, body)
}
func (s *Server) serve(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "same-origin")
	if h, pattern := s.public.Handler(r); pattern != "" {
		noteAuth(w, "public")
		h.ServeHTTP(w, r)
		return
	}
	if !s.authorized(w, r) {
		return
	}
	s.private.ServeHTTP(w, r)
}

// authorized checks the owner's token, and for writes that the request comes from this site.
func (s *Server) authorized(w http.ResponseWriter, r *http.Request) bool {
	if s.verifier != nil {
		if e := s.verifier.Authenticate(r); e != nil {
			noteAuth(w, "unauthorized")
			write(w, http.StatusUnauthorized, map[string]string{"code": "unauthorized", "message": "Sign in to continue."})
			return false
		}
	}
	if strings.HasPrefix(r.URL.Path, "/v1/") && r.Method != http.MethodGet {
		if origin := r.Header.Get("Origin"); origin != "" {
			scheme := "https"
			if r.TLS == nil && r.Header.Get("X-Forwarded-Proto") != "https" {
				scheme = "http"
			}
			if origin != scheme+"://"+r.Host {
				noteAuth(w, "forbidden_origin")
				note(w, "origin", truncate(origin, 200), "expected_origin", scheme+"://"+r.Host)
				write(w, http.StatusForbidden, map[string]string{"code": "unauthorized", "message": "Use the app to send this request."})
				return false
			}
		}
	}
	if s.verifier != nil {
		noteAuth(w, "ok")
	}
	return true
}
func (s *Server) serveWeb(w http.ResponseWriter, r *http.Request) {
	if s.web == nil {
		http.NotFound(w, r)
		return
	}
	s.web.ServeHTTP(w, r)
}
func (s *Server) chat(w http.ResponseWriter, r *http.Request) {
	after, _ := strconv.ParseInt(r.URL.Query().Get("after"), 10, 64)
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	entries, e := s.queries.History(after, limit)
	if e != nil {
		fail(w, e)
		return
	}
	note(w, "after", after, "limit", limit, "entries", len(entries))
	write(w, http.StatusOK, entries)
}
func (s *Server) items(w http.ResponseWriter, r *http.Request) {
	from, e1 := time.Parse(time.RFC3339, r.URL.Query().Get("from"))
	to, e2 := time.Parse(time.RFC3339, r.URL.Query().Get("to"))
	if e1 != nil || e2 != nil || !to.After(from) || to.Sub(from) > 62*24*time.Hour {
		note(w, "from_raw", truncate(r.URL.Query().Get("from"), 64), "to_raw", truncate(r.URL.Query().Get("to"), 64))
		fail(w, &domain.Invalid{Field: "range", Message: "Send from and to as times, at most 62 days apart."})
		return
	}
	if name := r.URL.Query().Get("timezone"); name != "" {
		zone, err := time.LoadLocation(name)
		if err != nil {
			fail(w, &domain.Invalid{Field: "timezone", Message: "Choose a valid timezone."})
			return
		}
		from = from.In(zone)
		to = to.In(zone)
	}
	items, e := s.queries.ItemsBetween(from, to)
	if e != nil {
		fail(w, e)
		return
	}
	note(w, "from", from.Format(time.RFC3339), "to", to.Format(time.RFC3339), "items", len(items))
	write(w, http.StatusOK, items)
}

// confirm applies a preview. If the item changed since, it answers 409 with a fresh preview.
func (s *Server) confirm(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	note(w, "change_id", id)
	entry, e := s.commands.Confirm(id)
	if errors.Is(e, domain.ErrConflict) {
		if replacement, re := s.commands.Refresh(id); re == nil {
			noteError(w, e)
			note(w, "refreshed_entry_id", replacement.ID, "outcome", "conflict_refreshed")
			write(w, http.StatusConflict, map[string]any{"code": "conflict", "message": e.Error(), "entry": replacement})
			return
		}
	}
	if e != nil {
		fail(w, e)
		return
	}
	note(w, "entry_id", entry.ID, "outcome", "confirmed")
	write(w, http.StatusOK, entry)
}
func (s *Server) discard(w http.ResponseWriter, r *http.Request) {
	note(w, "change_id", r.PathValue("id"))
	entry, e := s.commands.Discard(r.PathValue("id"))
	if e != nil {
		fail(w, e)
		return
	}
	note(w, "entry_id", entry.ID, "outcome", "discarded")
	write(w, http.StatusOK, entry)
}
func (s *Server) undo(w http.ResponseWriter, r *http.Request) {
	note(w, "audit_id", r.PathValue("id"))
	entry, e := s.commands.Undo(r.PathValue("id"))
	if e != nil {
		fail(w, e)
		return
	}
	note(w, "entry_id", entry.ID, "outcome", "undone")
	write(w, http.StatusOK, entry)
}
func (s *Server) message(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Text string `json:"text"`
	}
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 20000))
	d.DisallowUnknownFields()
	if e := d.Decode(&input); e != nil {
		noteError(w, e)
		fail(w, &domain.Invalid{Field: "text", Message: "Send a valid text message."})
		return
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		fail(w, &domain.Invalid{Field: "text", Message: "Send one text message."})
		return
	}
	note(w, "text_len", utf8.RuneCountInString(input.Text))
	accepted, err := s.messages.Accept(input.Text)
	if err != nil {
		fail(w, err)
		return
	}
	note(w, "entry_id", accepted.Entry.ID, "cursor", accepted.Entry.Cursor, "reply_id", accepted.ReplyID)
	write(w, http.StatusAccepted, accepted)
}

func (s *Server) events(w http.ResponseWriter, r *http.Request) {
	f, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming is unavailable.", 500)
		return
	}
	after, _ := strconv.ParseInt(r.URL.Query().Get("after"), 10, 64)
	if raw := r.Header.Get("Last-Event-ID"); raw != "" {
		if n, e := strconv.ParseInt(raw, 10, 64); e == nil && n > after {
			after = n
		}
	}
	opened := time.Now()
	sent, pings := 0, 0
	reason := "client_gone"
	slog.Info("event stream opened", "after", after, "last_event_id", r.Header.Get("Last-Event-ID"), "remote_addr", r.RemoteAddr)
	defer func() {
		slog.Info("event stream closed", "reason", reason, "entries_sent", sent, "pings", pings, "last_cursor", after, "duration_ms", time.Since(opened).Milliseconds())
		note(w, "entries_sent", sent, "pings", pings, "reason", reason)
	}()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(200)
	f.Flush()
	poll := time.NewTicker(200 * time.Millisecond)
	defer poll.Stop()
	ping := time.NewTicker(s.PingInterval)
	defer ping.Stop()
	send := func() bool {
		entries, e := s.queries.Events(after, 200)
		if e != nil {
			reason = "query_error"
			noteError(w, e)
			slog.Error("event stream query failed", "after", after, "error", e)
			return false
		}
		for _, entry := range entries {
			b, _ := json.Marshal(entry)
			http.NewResponseController(w).SetWriteDeadline(time.Now().Add(10 * time.Second))
			if _, e = fmt.Fprintf(w, "id: %d\nevent: entry\ndata: %s\n\n", entry.Cursor, b); e != nil {
				reason = "write_error"
				noteError(w, e)
				return false
			}
			after = entry.Cursor
			sent++
			slog.Debug("event stream entry sent", "cursor", entry.Cursor, "kind", entry.Kind)
		}
		f.Flush()
		return true
	}
	if !send() {
		return
	}
	for {
		select {
		case <-r.Context().Done():
			return
		case <-poll.C:
			if !send() {
				return
			}
		case <-ping.C:
			http.NewResponseController(w).SetWriteDeadline(time.Now().Add(10 * time.Second))
			if _, e := fmt.Fprint(w, "event: ping\ndata: {}\n\n"); e != nil {
				reason = "write_error"
				noteError(w, e)
				return
			}
			pings++
			f.Flush()
		}
	}
}
