package api

import (
	"bufio"
	"context"
	"encoding/json"
	"github.com/alexanderp4580/Minder/server/command"
	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/store"
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
