package api

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/alexanderp4580/Minder/server/domain"
)

// recorder wraps the response writer so the request log can report status, size, outcome and handler details.
type recorder struct {
	http.ResponseWriter
	status int
	bytes  int
	err    error
	auth   string
	attrs  []any
}

func (r *recorder) WriteHeader(status int) {
	if r.status == 0 {
		r.status = status
	}
	r.ResponseWriter.WriteHeader(status)
}
func (r *recorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(b)
	r.bytes += n
	return n, err
}
func (r *recorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Unwrap lets http.ResponseController reach the real connection.
func (r *recorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }

// note adds handler details (counts, identifiers, sizes) to the request log. It never takes message text or secrets.
func note(w http.ResponseWriter, kv ...any) {
	if r, ok := w.(*recorder); ok {
		r.attrs = append(r.attrs, kv...)
	}
}
func noteError(w http.ResponseWriter, err error) {
	if r, ok := w.(*recorder); ok && r.err == nil {
		r.err = err
	}
}
func noteAuth(w http.ResponseWriter, outcome string) {
	if r, ok := w.(*recorder); ok {
		r.auth = outcome
	}
}

// ServeHTTP logs one line for every request: who, what, outcome, size and time. It also recovers panics.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	id := requestID(r)
	w.Header().Set("X-Request-Id", id)
	rec := &recorder{ResponseWriter: w, auth: "none"}
	defer func() {
		if p := recover(); p != nil {
			if p == http.ErrAbortHandler {
				panic(p)
			}
			slog.Error("http panic", "request_id", id, "method", r.Method, "path", r.URL.Path, "panic", p, "stack", string(debug.Stack()))
			if rec.status == 0 {
				write(rec, http.StatusInternalServerError, map[string]string{"code": "internal_error", "message": "Minder could not finish. Try again."})
			}
		}
		s.logRequest(rec, r, id, start)
	}()
	s.serve(rec, r)
}
func requestID(r *http.Request) string {
	if id := r.Header.Get("X-Request-Id"); id != "" && len(id) <= 64 && !strings.ContainsAny(id, " \t\r\n\"") {
		return id
	}
	return domain.NewID()[:12]
}
func (s *Server) route(r *http.Request) string {
	if _, pattern := s.public.Handler(r); pattern != "" {
		return pattern
	}
	_, pattern := s.private.Handler(r)
	return pattern
}
func (s *Server) logRequest(rec *recorder, r *http.Request, id string, start time.Time) {
	status := rec.status
	if status == 0 {
		status = http.StatusOK
	}
	level := slog.LevelInfo
	switch {
	case status >= 500:
		level = slog.LevelError
	case status >= 400:
		level = slog.LevelWarn
	case r.URL.Path == "/v1/health":
		level = slog.LevelDebug
	}
	attrs := []any{
		"request_id", id, "method", r.Method, "path", r.URL.Path, "route", s.route(r), "status", status,
		"duration_ms", time.Since(start).Milliseconds(), "bytes", rec.bytes, "auth", rec.auth,
		"remote_addr", r.RemoteAddr, "user_agent", truncate(r.UserAgent(), 200),
	}
	if ip := r.Header.Get("Cf-Connecting-Ip"); ip != "" {
		attrs = append(attrs, "client_ip", ip)
	}
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		attrs = append(attrs, "forwarded_for", truncate(forwarded, 200))
	}
	if r.URL.RawQuery != "" {
		attrs = append(attrs, "query", truncate(r.URL.RawQuery, 200))
	}
	if r.ContentLength > 0 {
		attrs = append(attrs, "request_bytes", r.ContentLength)
	}
	if rec.err != nil {
		attrs = append(attrs, "error", rec.err.Error())
	}
	slog.Log(r.Context(), level, "http request", append(attrs, rec.attrs...)...)
}
func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n] + "…"
	}
	return s
}
