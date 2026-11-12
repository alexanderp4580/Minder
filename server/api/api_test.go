package api

import (
	"bufio"
	"context"
	"encoding/json"
	"github.com/alexanderp4580/Minder/server/command"
	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/store"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type replier struct{ s *store.Store }

func (r replier) Reply(ctx context.Context, text string) ([]domain.Entry, error) {
	e, err := r.s.AddEntry(domain.Entry{Kind: "minder", Text: "Which item do you mean?"})
	return []domain.Entry{e}, err
}
func TestChatHistoryAndStreamReconnect(t *testing.T) {
	s, e := store.Open(filepath.Join(t.TempDir(), "db"))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	h := New(s, command.New(s), replier{s}, nil, "")
	h.PingInterval = 20 * time.Millisecond
	srv := httptest.NewServer(h)
	defer srv.Close()
	resp, e := http.Post(srv.URL+"/v1/chat/messages", "application/json", strings.NewReader(`{"text":"Move it"}`))
	if e != nil {
		t.Fatal(e)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 202 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatal(resp.Status, string(b))
	}
	var accepted map[string]any
	json.NewDecoder(resp.Body).Decode(&accepted)
	if accepted["reply_id"] == nil {
		t.Fatal("reply id missing")
	}
	deadline := time.Now().Add(time.Second)
	var entries []domain.Entry
	for time.Now().Before(deadline) {
		entries, _ = s.History(0, 50)
		if len(entries) >= 2 {
			break
		}
		time.Sleep(time.Millisecond)
	}
	if len(entries) != 2 {
		t.Fatalf("entries %v", entries)
	}
	req, _ := http.NewRequest("GET", srv.URL+"/v1/events?after=1", nil)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	resp, e = http.DefaultClient.Do(req.WithContext(ctx))
	if e != nil {
		t.Fatal(e)
	}
	defer resp.Body.Close()
	scanner := bufio.NewScanner(resp.Body)
	seen, ping := false, false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") && strings.Contains(line, "Which item") {
			seen = true
		}
		if line == "event: ping" {
			ping = true
			break
		}
	}
	if !seen || !ping {
		t.Fatalf("replay %v ping %v", seen, ping)
	}
}
func TestInvalidRequests(t *testing.T) {
	s, _ := store.Open(filepath.Join(t.TempDir(), "db"))
	defer s.Close()
	h := New(s, command.New(s), replier{s}, nil, "")
	for _, input := range []string{`{`, `{"text":""}`, `{"text":"ok","unknown":true}`} {
		r := httptest.NewRequest("POST", "/v1/chat/messages", strings.NewReader(input))
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 400 {
			t.Fatalf("%s: %d", input, w.Code)
		}
	}
}
func TestDevProductionGuard(t *testing.T) {
	if ValidateMode("dev", "production") == nil {
		t.Fatal("production accepted dev")
	}
	if ValidateMode("dev", "development") != nil {
		t.Fatal("local dev refused")
	}
}

type blockedReplier struct{}

func (blockedReplier) Preflight() error { return domain.ErrLimit }
func (blockedReplier) Reply(context.Context, string) ([]domain.Entry, error) {
	panic("budget block called model")
}
func TestBudgetErrorIsReturnedBeforeAcceptingMessage(t *testing.T) {
	s, e := store.Open(filepath.Join(t.TempDir(), "db"))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	h := New(s, command.New(s), blockedReplier{}, nil, "")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("POST", "/v1/chat/messages", strings.NewReader(`{"text":"Call plumber"}`)))
	if w.Code != 429 || !strings.Contains(w.Body.String(), "limit_reached") {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
	entries, _ := s.History(0, 50)
	if len(entries) != 0 {
		t.Fatal("blocked request entered chat")
	}
}

func TestPublicLogoDoesNotExposePrivateRoutes(t *testing.T) {
	s, e := store.Open(filepath.Join(t.TempDir(), "db"))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	h := New(s, command.New(s), replier{s}, &Verifier{}, "")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/brand/minder-icon.png", nil))
	if w.Code != 200 || w.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("logo: %d %s", w.Code, w.Header().Get("Content-Type"))
	}
	icon, e := png.Decode(w.Body)
	if e != nil || icon.Bounds().Dx() != 192 || icon.Bounds().Dy() != 192 {
		t.Fatal("invalid public logo", e)
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("HEAD", "/brand/minder-icon.png", nil))
	if w.Code != 200 || w.Body.Len() != 0 {
		t.Fatal("HEAD logo returned a body or failed")
	}
	for _, path := range []string{"/", "/main.dart.js", "/v1/chat", "/v1/events", "/brand/minder-icon.png/extra", "/brand/minder-icon.png/../v1/chat"} {
		w = httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != 401 {
			t.Fatalf("private route %s returned %d", path, w.Code)
		}
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("POST", "/brand/minder-icon.png", nil))
	if w.Code != 401 {
		t.Fatal("public logo accepted an unauthenticated write")
	}
}

func TestReminderRoutesNeedAuthAndRejectInvalidBodies(t *testing.T) {
	s, e := store.Open(filepath.Join(t.TempDir(), "reminders"))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	locked := New(s, command.New(s), replier{s}, &Verifier{}, "")
	for _, path := range []string{"/v1/reminders", "/v1/push/config", "/v1/push/devices", "/v1/reminders/id/done"} {
		w := httptest.NewRecorder()
		locked.ServeHTTP(w, httptest.NewRequest("POST", path, strings.NewReader(`{}`)))
		if w.Code != 401 {
			t.Fatalf("unauthorized %s: %d", path, w.Code)
		}
	}
	h := New(s, command.New(s), replier{s}, nil, "")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("POST", "/v1/push/devices", strings.NewReader(`{"device_id":"phone","token":"test-token"}`)))
	if w.Code != 200 {
		t.Fatalf("device: %d", w.Code)
	}
	for _, body := range []string{`{}`, `{"device_id":"p","token":"t","extra":true}`, `{"device_id":"p","token":"t"} {}`} {
		w = httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("POST", "/v1/push/devices", strings.NewReader(body)))
		if w.Code != 400 {
			t.Fatalf("invalid body accepted: %d", w.Code)
		}
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/v1/push/config", nil))
	if w.Code != 503 {
		t.Fatalf("missing config: %d", w.Code)
	}
}

func TestItemsRangeReturnsPlacedItemsOnly(t *testing.T) {
	s, e := store.Open(filepath.Join(t.TempDir(), "items"))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	for _, i := range []domain.Item{
		{ID: "start", Title: "Dentist", Status: "open", Version: 1, Timezone: "Europe/Amsterdam", StartAt: "2026-11-13T09:00:00Z", DueAt: "2026-12-30T09:00:00Z"},
		{ID: "due", Title: "Taxes", Status: "open", Version: 1, Timezone: "Europe/Amsterdam", DueAt: "2026-11-20T09:00:00Z"},
		{ID: "edge", Title: "Edge", Status: "open", Version: 1, Timezone: "Europe/Amsterdam", DueAt: "2026-12-01T00:00:00Z"},
		{ID: "undated", Title: "Someday", Status: "open", Version: 1, Timezone: "Europe/Amsterdam"},
	} {
		i := i
		if e = s.Write(func(tx *store.Tx) error { return tx.PutItem(i) }); e != nil {
			t.Fatal(e)
		}
	}
	h := New(s, command.New(s), replier{s}, nil, "")
	get := func(query string) (int, []domain.Item) {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", "/v1/items"+query, nil))
		var items []domain.Item
		json.NewDecoder(w.Body).Decode(&items)
		return w.Code, items
	}
	code, items := get("?from=2026-11-01T00:00:00Z&to=2026-12-01T00:00:00Z")
	if code != 200 || len(items) != 2 || items[0].ID != "start" || items[1].ID != "due" {
		t.Fatalf("%d %+v", code, items)
	}
	for _, q := range []string{"", "?from=x&to=y", "?from=2026-12-01T00:00:00Z&to=2026-11-01T00:00:00Z", "?from=2026-01-01T00:00:00Z&to=2028-01-01T00:00:00Z"} {
		if code, _ = get(q); code != 400 {
			t.Fatalf("query %q gave %d", q, code)
		}
	}
	locked := New(s, command.New(s), replier{s}, &Verifier{}, "")
	w := httptest.NewRecorder()
	locked.ServeHTTP(w, httptest.NewRequest("GET", "/v1/items?from=2026-11-01T00:00:00Z&to=2026-12-01T00:00:00Z", nil))
	if w.Code != 401 {
		t.Fatalf("unauthorized: %d", w.Code)
	}
}
