package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/store"
)

// captureLogs sends every log record to a buffer for the test and restores the old logger after it.
// syncBuffer lets the test read log output while background goroutines still write it.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}
func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func captureLogs(t *testing.T) func() []map[string]any {
	t.Helper()
	var buf syncBuffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(previous) })
	return func() []map[string]any {
		var out []map[string]any
		for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
			if line == "" {
				continue
			}
			var record map[string]any
			if json.Unmarshal([]byte(line), &record) == nil {
				out = append(out, record)
			}
		}
		return out
	}
}
func find(records []map[string]any, msg string) map[string]any {
	for _, r := range records {
		if r["msg"] == msg {
			return r
		}
	}
	return nil
}

type failingQueries struct{ Queries }

func (failingQueries) History(int64, int) ([]domain.Entry, error) {
	return nil, errors.New("disk exploded")
}

type panicQueries struct{ Queries }

func (panicQueries) History(int64, int) ([]domain.Entry, error) { panic("boom") }

func loggedServer(t *testing.T, auth Authenticator) (*Server, *store.Store) {
	t.Helper()
	s, e := store.Open(filepath.Join(t.TempDir(), "db"))
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { s.Close() })
	return testServer(t, s, nil, replier{s}, auth, ""), s
}

func TestEveryRequestIsLoggedWithOutcome(t *testing.T) {
	logs := captureLogs(t)
	h, _ := loggedServer(t, nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/v1/chat?after=0&limit=5", nil)
	req.Header.Set("User-Agent", "TestAgent/1.0")
	h.ServeHTTP(rec, req)
	r := find(logs(), "http request")
	if r == nil {
		t.Fatal("no request log")
	}
	if r["method"] != "GET" || r["path"] != "/v1/chat" || r["status"] != float64(200) || r["user_agent"] != "TestAgent/1.0" {
		t.Fatalf("incomplete request log: %v", r)
	}
	if r["request_id"] == "" || r["request_id"] != rec.Header().Get("X-Request-Id") {
		t.Fatalf("request id %v does not match header %q", r["request_id"], rec.Header().Get("X-Request-Id"))
	}
	for _, key := range []string{"duration_ms", "bytes", "remote_addr", "route", "auth"} {
		if _, ok := r[key]; !ok {
			t.Fatalf("missing %s in %v", key, r)
		}
	}
	if r["entries"] == nil {
		t.Fatalf("handler output count missing: %v", r)
	}
}

func TestRejectedRequestsAreLoggedAsWarnings(t *testing.T) {
	logs := captureLogs(t)
	h, _ := loggedServer(t, rejectingAuthenticator{})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/v1/chat", nil))
	r := find(logs(), "http request")
	if r == nil || r["level"] != "WARN" || r["status"] != float64(401) || r["auth"] != "unauthorized" {
		t.Fatalf("unauthorized request not logged as a warning: %v", r)
	}
	rec = httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/v1/chat/messages", strings.NewReader(`{"text":"x"}`))
	req.Header.Set("Origin", "https://evil.example")
	h2, _ := loggedServer(t, nil)
	h2.ServeHTTP(rec, req)
	var forbidden map[string]any
	for _, line := range logs() {
		if line["status"] == float64(403) {
			forbidden = line
		}
	}
	if forbidden == nil || forbidden["auth"] != "forbidden_origin" || forbidden["origin"] != "https://evil.example" {
		t.Fatalf("cross-origin write not logged: %v", forbidden)
	}
}

func TestServerErrorLogsTheRealCauseButNotTheResponse(t *testing.T) {
	logs := captureLogs(t)
	s, _ := store.Open(filepath.Join(t.TempDir(), "db"))
	t.Cleanup(func() { s.Close() })
	h := testServer(t, s, nil, replier{s}, nil, "")
	h.queries = failingQueries{h.queries}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/v1/chat", nil))
	if rec.Code != 500 || strings.Contains(rec.Body.String(), "disk exploded") {
		t.Fatalf("response leaked or wrong: %d %s", rec.Code, rec.Body.String())
	}
	r := find(logs(), "http request")
	if r["level"] != "ERROR" || r["error"] != "disk exploded" || r["error_code"] != "internal_error" {
		t.Fatalf("cause not logged: %v", r)
	}
}

func TestPanicIsRecoveredAndLoggedWithStack(t *testing.T) {
	logs := captureLogs(t)
	s, _ := store.Open(filepath.Join(t.TempDir(), "db"))
	t.Cleanup(func() { s.Close() })
	h := testServer(t, s, nil, replier{s}, nil, "")
	h.queries = panicQueries{h.queries}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/v1/chat", nil))
	if rec.Code != 500 {
		t.Fatalf("status %d", rec.Code)
	}
	r := find(logs(), "http panic")
	if r == nil || r["panic"] != "boom" || !strings.Contains(r["stack"].(string), "goroutine") {
		t.Fatalf("panic not logged: %v", r)
	}
}

func TestMessageLogsLengthNotText(t *testing.T) {
	logs := captureLogs(t)
	h, _ := loggedServer(t, nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/v1/chat/messages", strings.NewReader(`{"text":"my secret dentist plan"}`))
	h.ServeHTTP(rec, req)
	if rec.Code != 202 {
		t.Fatalf("status %d %s", rec.Code, rec.Body.String())
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) && find(logs(), "reply finished") == nil {
		time.Sleep(10 * time.Millisecond)
	}
	r := find(logs(), "http request")
	if r["text_len"] != float64(len("my secret dentist plan")) || r["reply_id"] == nil {
		t.Fatalf("input size or reply id missing: %v", r)
	}
	for _, line := range logs() {
		b, _ := json.Marshal(line)
		if strings.Contains(string(b), "dentist") {
			t.Fatalf("message text reached the log: %s", b)
		}
	}
}

func TestDeviceRegistrationLogsFingerprintNotToken(t *testing.T) {
	logs := captureLogs(t)
	h, _ := loggedServer(t, nil)
	rec := httptest.NewRecorder()
	token := strings.Repeat("t", 40) + "SECRETTOKEN"
	req := httptest.NewRequest("POST", "/v1/push/devices", strings.NewReader(`{"device_id":"phone-1","token":"`+token+`"}`))
	h.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("status %d %s", rec.Code, rec.Body.String())
	}
	r := find(logs(), "http request")
	if r["device_id"] != "phone-1" || r["token_len"] != float64(len(token)) || len(r["token_fp"].(string)) != 8 {
		t.Fatalf("registration details missing: %v", r)
	}
	for _, line := range logs() {
		b, _ := json.Marshal(line)
		if strings.Contains(string(b), "SECRETTOKEN") {
			t.Fatalf("push token reached the log: %s", b)
		}
	}
}

func TestHealthChecksStayOutOfTheInfoLog(t *testing.T) {
	logs := captureLogs(t)
	h, _ := loggedServer(t, nil)
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/v1/health", nil))
	r := find(logs(), "http request")
	if r == nil || r["level"] != "DEBUG" {
		t.Fatalf("health check should log at debug: %v", r)
	}
}

func TestEventStreamLogsOpenAndClose(t *testing.T) {
	logs := captureLogs(t)
	h, s := loggedServer(t, nil)
	h.PingInterval = 10 * time.Millisecond
	s.AddEntry(domain.Entry{Kind: "minder", Text: "hello"})
	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest("GET", "/v1/events?after=0", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	done := make(chan struct{})
	go func() { h.ServeHTTP(rec, req); close(done) }()
	time.Sleep(80 * time.Millisecond)
	cancel()
	<-done
	if find(logs(), "event stream opened") == nil {
		t.Fatal("stream open not logged")
	}
	closed := find(logs(), "event stream closed")
	if closed == nil || closed["reason"] != "client_gone" || closed["entries_sent"].(float64) < 1 || closed["pings"].(float64) < 1 {
		t.Fatalf("stream close incomplete: %v", closed)
	}
}

var _ = http.StatusOK
