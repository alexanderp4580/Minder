package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/alexanderp4580/Minder/server/command"
	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/store"
)

type Authenticator interface {
	Authenticate(*http.Request) error
}

type Replier interface {
	Reply(context.Context, string) ([]domain.Entry, error)
}
type Server struct {
	// Branding is the folder that holds the brand guide. Empty turns /branding off.
	Branding     string
	PushConfig   *PushConfig
	PingInterval time.Duration
	store        *store.Store
	commands     *command.Service
	assistant    Replier
	verifier     Authenticator
	web          http.Handler
	public       *http.ServeMux // reachable without sign-in
	private      *http.ServeMux // reachable after sign-in
	processing   sync.Mutex
}

func New(s *store.Store, c *command.Service, a Replier, v Authenticator, web string) *Server {
	srv := &Server{store: s, commands: c, assistant: a, verifier: v, PingInterval: 20 * time.Second}
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
	code := command.ErrorCode(err)
	switch code {
	case "invalid_field":
		status = 400
	case "conflict":
		status = 409
	case "not_found":
		status = 404
	case "limit_reached":
		status = 429
	case "model_error":
		status = 502
	}
	body := map[string]any{"code": code, "message": command.Describe(err)}
	var invalid *domain.Invalid
	if errors.As(err, &invalid) {
		body["field"] = invalid.Field
	}
	write(w, status, body)
}
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "same-origin")
	if h, pattern := s.public.Handler(r); pattern != "" {
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
				write(w, http.StatusForbidden, map[string]string{"code": "unauthorized", "message": "Use the app to send this request."})
				return false
			}
		}
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
	entries, e := s.store.History(after, limit)
	if e != nil {
		fail(w, e)
		return
	}
	write(w, http.StatusOK, entries)
}
func (s *Server) items(w http.ResponseWriter, r *http.Request) {
	from, e1 := time.Parse(time.RFC3339, r.URL.Query().Get("from"))
	to, e2 := time.Parse(time.RFC3339, r.URL.Query().Get("to"))
	if e1 != nil || e2 != nil || !to.After(from) || to.Sub(from) > 62*24*time.Hour {
		fail(w, &domain.Invalid{Field: "range", Message: "Send from and to as times, at most 62 days apart."})
		return
	}
	items, e := s.store.ItemsBetween(from, to)
	if e != nil {
		fail(w, e)
		return
	}
	write(w, http.StatusOK, items)
}

// confirm applies a preview. If the item changed since, it answers 409 with a fresh preview.
func (s *Server) confirm(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	entry, e := s.commands.Confirm(id)
	if errors.Is(e, domain.ErrConflict) {
		if replacement, re := s.commands.Refresh(id); re == nil {
			write(w, http.StatusConflict, map[string]any{"code": "conflict", "message": e.Error(), "entry": replacement})
			return
		}
	}
	if e != nil {
		fail(w, e)
		return
	}
	write(w, http.StatusOK, entry)
}
func (s *Server) discard(w http.ResponseWriter, r *http.Request) {
	entry, e := s.commands.Discard(r.PathValue("id"))
	if e != nil {
		fail(w, e)
		return
	}
	write(w, http.StatusOK, entry)
}
func (s *Server) undo(w http.ResponseWriter, r *http.Request) {
	entry, e := s.commands.Undo(r.PathValue("id"))
	if e != nil {
		fail(w, e)
		return
	}
	write(w, http.StatusOK, entry)
}
func (s *Server) message(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Text string `json:"text"`
	}
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 20000))
	d.DisallowUnknownFields()
	if e := d.Decode(&input); e != nil {
		fail(w, &domain.Invalid{Field: "text", Message: "Send a valid text message."})
		return
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		fail(w, &domain.Invalid{Field: "text", Message: "Send one text message."})
		return
	}
	input.Text = strings.TrimSpace(input.Text)
	if n := utf8.RuneCountInString(input.Text); n < 1 || n > 4000 {
		fail(w, &domain.Invalid{Field: "text", Message: "Use 1 to 4,000 characters."})
		return
	}
	if ready, ok := s.assistant.(interface{ Preflight() error }); ok {
		if e := ready.Preflight(); e != nil {
			fail(w, e)
			return
		}
	}
	if !s.processing.TryLock() {
		write(w, 429, map[string]string{"code": "busy", "message": "Wait for Minder to finish."})
		return
	}
	entry, e := s.store.AddEntry(domain.Entry{Kind: "user", Text: input.Text})
	if e != nil {
		s.processing.Unlock()
		fail(w, e)
		return
	}
	replyID := store.ID()
	write(w, 202, map[string]any{"entry": entry, "reply_id": replyID})
	go func() {
		defer s.processing.Unlock()
		defer func() {
			if recover() != nil {
				log.Print("The assistant stopped with an internal error")
				s.store.AddEntry(domain.Entry{ID: replyID, Kind: "error", Text: command.Describe(errors.New("assistant panic"))})
			}
		}()
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		_, err := s.assistant.Reply(ctx, input.Text)
		if err != nil {
			s.store.AddEntry(domain.Entry{ID: replyID, Kind: "error", Text: command.Describe(err)})
		}
	}()
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
		entries, e := s.store.Events(after, 200)
		if e != nil {
			return false
		}
		for _, entry := range entries {
			b, _ := json.Marshal(entry)
			http.NewResponseController(w).SetWriteDeadline(time.Now().Add(10 * time.Second))
			if _, e = fmt.Fprintf(w, "id: %d\nevent: entry\ndata: %s\n\n", entry.Cursor, b); e != nil {
				return false
			}
			after = entry.Cursor
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
				return
			}
			f.Flush()
		}
	}
}
