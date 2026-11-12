package openrouter

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/alexanderp4580/Minder/server/assistant"
)

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

func capture(t *testing.T) func() []map[string]any {
	t.Helper()
	var buf syncBuffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(previous) })
	return func() []map[string]any {
		var out []map[string]any
		for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
			var r map[string]any
			if json.Unmarshal([]byte(line), &r) == nil {
				out = append(out, r)
			}
		}
		return out
	}
}
func first(records []map[string]any, msg string) map[string]any {
	for _, r := range records {
		if r["msg"] == msg {
			return r
		}
	}
	return nil
}

func TestModelCallLogsRequestAndResponseWithoutContentOrKey(t *testing.T) {
	logs := capture(t)
	m := provider(t, `{"choices":[{"finish_reason":"tool_calls","message":{"content":null,"tool_calls":[{"id":"c1","function":{"name":"propose_create","arguments":"{\"title\":\"x\"}"}},{"id":"c2","function":{"name":"list_items","arguments":"{}"}}]}}],"usage":{"cost":0.0123,"prompt_tokens":900,"completion_tokens":40,"total_tokens":940}}`, nil)
	m.Key = "sk-SECRETKEY"
	m.Model = "test/model"
	if _, e := m.Complete(context.Background(), []assistant.Message{{Role: "system", Content: "rules"}, {Role: "user", Content: "buy PRIVATEMILK"}}); e != nil {
		t.Fatal(e)
	}
	req := first(logs(), "model request")
	if req == nil || req["model"] != "test/model" || req["messages"] != float64(2) || req["tools"] == nil || req["request_bytes"] == nil || req["max_tokens"] != float64(maxOutputTokens) {
		t.Fatalf("request log incomplete: %v", req)
	}
	res := first(logs(), "model response")
	if res == nil || res["status"] != float64(200) || res["finish_reason"] != "tool_calls" || res["cost_usd"] != 0.0123 || res["prompt_tokens"] != float64(900) || res["completion_tokens"] != float64(40) {
		t.Fatalf("response log incomplete: %v", res)
	}
	if res["tool_calls"] != "propose_create,list_items" || res["duration_ms"] == nil {
		t.Fatalf("tool calls or timing missing: %v", res)
	}
	for _, line := range logs() {
		b, _ := json.Marshal(line)
		if strings.Contains(string(b), "SECRETKEY") || strings.Contains(string(b), "PRIVATEMILK") {
			t.Fatalf("key or message text reached the log: %s", b)
		}
	}
}

func TestModelFailuresLogTheCause(t *testing.T) {
	logs := capture(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(402)
		w.Write([]byte(`{"error":{"message":"insufficient credits","code":402}}`))
	}))
	defer srv.Close()
	m := &Client{Key: "k", URL: srv.URL}
	if _, e := m.Complete(context.Background(), []assistant.Message{{Role: "user", Content: "hi"}}); e == nil {
		t.Fatal("expected failure")
	}
	r := first(logs(), "model request failed")
	if r == nil || r["level"] != "ERROR" || r["status"] != float64(402) || !strings.Contains(r["body"].(string), "insufficient credits") {
		t.Fatalf("provider failure not logged: %v", r)
	}
	closed := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := closed.URL
	closed.Close()
	m = &Client{Key: "k", URL: url}
	if _, e := m.Complete(context.Background(), []assistant.Message{{Role: "user", Content: "hi"}}); e == nil {
		t.Fatal("expected connection failure")
	}
	var connect map[string]any
	for _, l := range logs() {
		if l["msg"] == "model request failed" && l["stage"] == "connect" {
			connect = l
		}
	}
	if connect == nil || !strings.Contains(connect["error"].(string), "connection refused") {
		t.Fatalf("connection error cause not logged: %v", connect)
	}
}

func TestUnreadableAndTruncatedRepliesAreLogged(t *testing.T) {
	logs := capture(t)
	m := provider(t, `not json`, nil)
	m.Complete(context.Background(), []assistant.Message{{Role: "user", Content: "hi"}})
	if r := first(logs(), "model reply rejected"); r == nil || r["reason"] != "unreadable" {
		t.Fatalf("unreadable reply not logged: %v", r)
	}
	m = provider(t, `{"choices":[{"finish_reason":"length","message":{"content":"x"}}],"usage":{"cost":0.1}}`, nil)
	m.Complete(context.Background(), []assistant.Message{{Role: "user", Content: "hi"}})
	var cut map[string]any
	for _, l := range logs() {
		if l["msg"] == "model reply rejected" && l["reason"] == "truncated" {
			cut = l
		}
	}
	if cut == nil {
		t.Fatal("truncated reply not logged")
	}
}
