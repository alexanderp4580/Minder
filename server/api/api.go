package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/alexanderp4580/Minder/server/command"
	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/store"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

type Replier interface {
	Reply(context.Context, string) ([]domain.Entry, error)
}
type Server struct {
	store        *store.Store
	commands     *command.Service
	assistant    Replier
	verifier     *Verifier
	web          string
	PingInterval time.Duration
	processing   sync.Mutex
}

func New(s *store.Store, c *command.Service, a Replier, v *Verifier, web string) *Server {
	return &Server{store: s, commands: c, assistant: a, verifier: v, web: web, PingInterval: 20 * time.Second}
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
	if r.URL.Path == "/v1/health" && r.Method == "GET" {
		write(w, 200, map[string]string{"status": "ok", "version": "m0"})
		return
	}
	if s.verifier != nil {
		if e := s.verifier.Verify(r.Header.Get("Cf-Access-Jwt-Assertion")); e != nil {
			write(w, 401, map[string]string{"code": "unauthorized", "message": "Sign in to continue."})
			return
		}
	}
	if strings.HasPrefix(r.URL.Path, "/v1/") && r.Method != "GET" {
		if origin := r.Header.Get("Origin"); origin != "" {
			scheme := "https"
			if r.TLS == nil && r.Header.Get("X-Forwarded-Proto") != "https" {
				scheme = "http"
			}
			if origin != scheme+"://"+r.Host {
				write(w, 403, map[string]string{"code": "unauthorized", "message": "Use the app to send this request."})
				return
			}
		}
	}
	switch {
	case r.Method == "GET" && r.URL.Path == "/v1/chat":
		after, _ := strconv.ParseInt(r.URL.Query().Get("after"), 10, 64)
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		entries, e := s.store.History(after, limit)
		if e != nil {
			fail(w, e)
			return
		}
		write(w, 200, entries)
	case r.Method == "GET" && r.URL.Path == "/v1/events":
		s.events(w, r)
	case r.Method == "POST" && r.URL.Path == "/v1/chat/messages":
		s.message(w, r)
	case r.Method == "POST" && strings.HasPrefix(r.URL.Path, "/v1/changes/"):
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/v1/changes/"), "/")
		if len(parts) != 2 {
			http.NotFound(w, r)
			return
		}
		var entry domain.Entry
		var e error
		switch parts[1] {
		case "confirm":
			entry, e = s.commands.Confirm(parts[0])
		case "discard":
			entry, e = s.commands.Discard(parts[0])
		default:
			http.NotFound(w, r)
			return
		}
		if e != nil {
			if errors.Is(e, domain.ErrConflict) && parts[1] == "confirm" {
				replacement, re := s.commands.Refresh(parts[0])
				if re == nil {
					write(w, 409, map[string]any{"code": "conflict", "message": e.Error(), "entry": replacement})
					return
				}
			}
			fail(w, e)
			return
		}
		write(w, 200, entry)
	case r.Method == "POST" && strings.HasPrefix(r.URL.Path, "/v1/undo/"):
		entry, e := s.commands.Undo(strings.TrimPrefix(r.URL.Path, "/v1/undo/"))
		if e != nil {
			fail(w, e)
			return
		}
		write(w, 200, entry)
	default:
		if strings.HasPrefix(r.URL.Path, "/v1/") {
			http.NotFound(w, r)
			return
		}
		if s.web == "" {
			http.NotFound(w, r)
			return
		}
		http.FileServer(http.Dir(s.web)).ServeHTTP(w, r)
	}
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
