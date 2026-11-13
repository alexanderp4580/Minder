package api

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alexanderp4580/Minder/server/command"
	"github.com/alexanderp4580/Minder/server/conversation"
	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/reminder"
	"github.com/alexanderp4580/Minder/server/store"
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
	t.Cleanup(func() { s.Close() })
	h := testServer(t, s, command.New(s), replier{s}, nil, "")
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
	t.Cleanup(func() { s.Close() })
	h := testServer(t, s, command.New(s), replier{s}, nil, "")
	for _, input := range []string{`{`, `{"text":""}`, `{"text":"ok","unknown":true}`} {
		r := httptest.NewRequest("POST", "/v1/chat/messages", strings.NewReader(input))
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 400 {
			t.Fatalf("%s: %d", input, w.Code)
		}
	}
}

type rejectingAuthenticator struct{}

func (rejectingAuthenticator) Authenticate(*http.Request) error {
	return errors.New("invalid identity")
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
	t.Cleanup(func() { s.Close() })
	h := testServer(t, s, command.New(s), blockedReplier{}, nil, "")
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
	t.Cleanup(func() { s.Close() })
	h := testServer(t, s, command.New(s), replier{s}, rejectingAuthenticator{}, "")
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
	t.Cleanup(func() { s.Close() })
	locked := testServer(t, s, command.New(s), replier{s}, rejectingAuthenticator{}, "")
	for _, path := range []string{"/v1/reminders", "/v1/push/config", "/v1/push/devices", "/v1/reminders/id/done"} {
		w := httptest.NewRecorder()
		locked.ServeHTTP(w, httptest.NewRequest("POST", path, strings.NewReader(`{}`)))
		if w.Code != 401 {
			t.Fatalf("unauthorized %s: %d", path, w.Code)
		}
	}
	h := testServer(t, s, command.New(s), replier{s}, nil, "")
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
	t.Cleanup(func() { s.Close() })
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
	h := testServer(t, s, command.New(s), replier{s}, nil, "")
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
	locked := testServer(t, s, command.New(s), replier{s}, rejectingAuthenticator{}, "")
	w := httptest.NewRecorder()
	locked.ServeHTTP(w, httptest.NewRequest("GET", "/v1/items?from=2026-11-01T00:00:00Z&to=2026-12-01T00:00:00Z", nil))
	if w.Code != 401 {
		t.Fatalf("unauthorized: %d", w.Code)
	}
}

func TestBrandingServesOnlyTheGuideFilesAfterSignIn(t *testing.T) {
	s, e := store.Open(filepath.Join(t.TempDir(), "brand"))
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { s.Close() })
	dir := t.TempDir()
	for name, body := range map[string]string{"index.html": "<h1>Guide</h1>", "app-icon.svg": "<svg/>", "app-icon.png": "png", "AGENTS.md": "private notes"} {
		if e = os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); e != nil {
			t.Fatal(e)
		}
	}
	h := testServer(t, s, command.New(s), replier{s}, nil, "")
	h.Branding = dir
	get := func(h *Server, method, path string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(method, path, nil))
		return w
	}
	if w := get(h, "GET", "/branding"); w.Code != 301 || w.Header().Get("Location") != "/branding/" {
		t.Fatalf("redirect: %d %q", w.Code, w.Header().Get("Location"))
	}
	if w := get(h, "GET", "/branding/"); w.Code != 200 || !strings.Contains(w.Body.String(), "Guide") || !strings.HasPrefix(w.Header().Get("Content-Type"), "text/html") {
		t.Fatalf("index: %d %q", w.Code, w.Header().Get("Content-Type"))
	}
	if w := get(h, "GET", "/branding/app-icon.svg"); w.Code != 200 || w.Header().Get("Content-Type") != "image/svg+xml" {
		t.Fatalf("svg: %d %q", w.Code, w.Header().Get("Content-Type"))
	}
	if w := get(h, "HEAD", "/branding/app-icon.png"); w.Code != 200 {
		t.Fatalf("png head: %d", w.Code)
	}
	for _, path := range []string{"/branding/AGENTS.md", "/branding/../AGENTS.md", "/branding/%2e%2e/AGENTS.md", "/branding/index.html/x", "/branding/other"} {
		if w := get(h, "GET", path); w.Code == 200 || strings.Contains(w.Body.String(), "private notes") {
			t.Fatalf("%s: %d", path, w.Code)
		}
	}
	if w := get(h, "POST", "/branding/"); w.Code == 200 {
		t.Fatal("POST accepted")
	}
	locked := testServer(t, s, command.New(s), replier{s}, rejectingAuthenticator{}, "")
	locked.Branding = dir
	if w := get(locked, "GET", "/branding/"); w.Code != 401 {
		t.Fatalf("unauthenticated: %d", w.Code)
	}
	off := testServer(t, s, command.New(s), replier{s}, nil, "")
	if w := get(off, "GET", "/branding/"); w.Code != 404 {
		t.Fatalf("disabled: %d", w.Code)
	}
}

type panicReplier struct{}

func (panicReplier) Reply(context.Context, string) ([]domain.Entry, error) { panic("boom") }

func TestAssistantPanicBecomesAnErrorEntryAndFreesTheServer(t *testing.T) {
	s, e := store.Open(filepath.Join(t.TempDir(), "panic"))
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { s.Close() })
	h := testServer(t, s, command.New(s), panicReplier{}, nil, "")
	for range 2 {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("POST", "/v1/chat/messages", strings.NewReader(`{"text":"Hello"}`)))
		if w.Code != 202 {
			t.Fatalf("message not accepted: %d %s", w.Code, w.Body.String())
		}
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			if entries, _ := s.History(0, 50); len(entries) > 0 && entries[len(entries)-1].Kind == "error" {
				break
			}
			time.Sleep(20 * time.Millisecond)
		}

	}
	entries, _ := s.History(0, 50)
	errs := 0
	for _, en := range entries {
		if en.Kind == "error" && en.Text == "Minder could not finish. Try again." {
			errs++
		}
	}
	if errs != 2 {
		t.Fatalf("want two error entries, got %d in %+v", errs, entries)
	}
}

func testServer(t *testing.T, database *store.Store, commands Commands, replier conversation.Replier, auth Authenticator, web string) *Server {
	t.Helper()
	messages := conversation.New(context.Background(), database, replier)
	t.Cleanup(messages.Close)
	return New(database, commands, messages, reminder.New(database), auth, web)
}

func TestSetRemindersRouteUpdatesPendingChangesOnly(t *testing.T) {
	s, e := store.Open(filepath.Join(t.TempDir(), "db"))
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { s.Close() })
	commands := command.New(s)
	h := testServer(t, s, commands, replier{s}, nil, "")
	entry, e := commands.Preview(domain.Proposal{Action: "create_item", Title: "Fitting", Kind: "event", StartAt: time.Now().UTC().Add(72 * time.Hour).Format(time.RFC3339)})
	if e != nil {
		t.Fatal(e)
	}
	post := func(body string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/v1/changes/"+entry.Change.ID+"/reminders", strings.NewReader(body)))
		return w
	}
	if w := post(`{"offsets":[1440,60]}`); w.Code != 200 || !strings.Contains(w.Body.String(), `"reminder_offsets_minutes":[1440,60]`) {
		t.Fatal(w.Code, w.Body.String())
	}
	if w := post(`{"offsets":[7]}`); w.Code != 400 {
		t.Fatal("unknown offset", w.Code)
	}
	if w := post(`{"offsets":[10],"extra":1}`); w.Code != 400 {
		t.Fatal("unknown field", w.Code)
	}
	if _, e = commands.Confirm(entry.Change.ID); e != nil {
		t.Fatal(e)
	}
	if w := post(`{"offsets":[10]}`); w.Code != 409 {
		t.Fatal("confirmed change", w.Code)
	}
}

func TestDeleteItemRouteGivesAPreviewThatConfirmsThroughTheChangeRoutes(t *testing.T) {
	s, e := store.Open(filepath.Join(t.TempDir(), "db"))
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { s.Close() })
	commands := command.New(s)
	h := testServer(t, s, commands, replier{s}, nil, "")
	created, e := commands.Preview(domain.Proposal{Action: "create_item", Title: "Fitting", DueAt: "2099-01-01T10:00:00Z", Timezone: "Europe/Amsterdam"})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = commands.Confirm(created.Change.ID); e != nil {
		t.Fatal(e)
	}
	items, _ := s.Items()
	id := items[0].ID
	post := func(path, body string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, path, strings.NewReader(body)))
		return w
	}
	if w := post("/v1/items/missing/delete", `{"expected_version":1}`); w.Code != 404 {
		t.Fatal("unknown item", w.Code, w.Body.String())
	}
	if w := post("/v1/items/"+id+"/delete", `{"expected_version":9}`); w.Code != 409 {
		t.Fatal("stale version", w.Code, w.Body.String())
	}
	if w := post("/v1/items/"+id+"/delete", `{"expected_version":1,"extra":1}`); w.Code != 400 {
		t.Fatal("unknown field", w.Code)
	}
	w := post("/v1/items/"+id+"/delete", `{"expected_version":1}`)
	var preview domain.Entry
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &preview) != nil || preview.Change == nil || preview.Change.Action != "delete_item" || preview.Change.State != "pending" {
		t.Fatal(w.Code, w.Body.String())
	}
	if left, _ := s.Items(); len(left) != 1 {
		t.Fatal("the preview deleted the item")
	}
	if w := post("/v1/changes/"+preview.Change.ID+"/confirm", ""); w.Code != 200 || !strings.Contains(w.Body.String(), `"undo_id"`) {
		t.Fatal("confirm", w.Code, w.Body.String())
	}
	if left, _ := s.Items(); len(left) != 0 {
		t.Fatal("confirm kept the item")
	}
}
