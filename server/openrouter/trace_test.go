package openrouter

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alexanderp4580/Minder/server/assistant"
	"github.com/alexanderp4580/Minder/server/domain"
)

func readTraces(t *testing.T, dir string) []map[string]any {
	t.Helper()
	files, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var records []map[string]any
	for _, file := range files {
		info, err := file.Info()
		if err != nil || info.Mode().Perm() != 0600 {
			t.Fatalf("trace permissions: %v %v", info, err)
		}
		b, err := os.ReadFile(filepath.Join(dir, file.Name()))
		if err != nil {
			t.Fatal(err)
		}
		var record map[string]any
		if err = json.Unmarshal(b, &record); err != nil {
			t.Fatal(err)
		}
		records = append(records, record)
	}
	return records
}

func TestTraceCapturesCompleteExchangePrivately(t *testing.T) {
	logs := capture(t)
	dir := filepath.Join(t.TempDir(), "private")
	trace, err := NewTrace(dir, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("trace directory missing: %v", err)
	}
	if info.Mode().Perm() != 0700 {
		t.Fatal("trace directory must be private")
	}
	reply := `{"choices":[{"finish_reason":"tool_calls","message":{"content":"PRIVATE_REPLY","tool_calls":[{"id":"proposal1","function":{"name":"propose_review","arguments":"{\"candidates\":[]}"}}]}}],"usage":{"cost":0.001}}`
	m := provider(t, reply, nil)
	m.Key, m.Trace = "SECRET_API_KEY", trace
	ctx := domain.WithTrace(context.Background(), "turn/../../unsafe")
	messages := []assistant.Message{
		{Role: "system", Content: "PRIVATE_RULES"},
		{Role: "tool", ToolCallID: "read1", Content: "PRIVATE_EMAIL" + strings.Repeat("x", 400000)},
		{Role: "user", Parts: []assistant.ContentPart{{Type: "image_url", ImageURL: &assistant.ImageURL{URL: "data:image/png;base64,AA=="}}}},
	}
	if _, err = m.Complete(ctx, messages); err != nil {
		t.Fatal(err)
	}
	records := readTraces(t, dir)
	request, response := first(records, "request"), first(records, "response")
	if request == nil || response == nil {
		t.Fatalf("exchange missing: %d records", len(records))
	}
	if request["trace_id"] != "turn/../../unsafe" || request["call_id"] == "" || response["call_id"] != request["call_id"] {
		t.Fatal("correlation missing")
	}
	body := request["data"].(map[string]any)["body"].(string)
	var payload map[string]any
	if err = json.Unmarshal([]byte(body), &payload); err != nil {
		t.Fatal(err)
	}
	wireMessages := payload["messages"].([]any)
	if wireMessages[1].(map[string]any)["content"] != messages[1].Content || !strings.Contains(body, "PRIVATE_RULES") || !strings.Contains(body, "data:image/png;base64,AA==") || payload["tools"] == nil {
		t.Fatal("request incomplete")
	}
	if response["data"].(map[string]any)["body"] != reply {
		t.Fatal("response changed")
	}
	all, _ := json.Marshal(records)
	if strings.Contains(string(all), "SECRET_API_KEY") || strings.Contains(string(all), "Authorization") {
		t.Fatal("authentication reached trace")
	}
	ordinary, _ := json.Marshal(logs())
	if strings.Contains(string(ordinary), "PRIVATE_") || strings.Contains(string(ordinary), "SECRET_API_KEY") {
		t.Fatal("private content reached ordinary logs")
	}
}

func TestTraceDisabledAndExpiredCaptureNothing(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		dir := t.TempDir()
		m := provider(t, `{"choices":[{"finish_reason":"stop","message":{"content":"ok"}}],"usage":{"cost":0.001}}`, nil)
		if enabled {
			var err error
			m.Trace, err = NewTrace(dir, time.Now().Add(-time.Second))
			if err != nil {
				t.Fatal(err)
			}
		}
		if _, err := m.Complete(context.Background(), []assistant.Message{{Role: "user", Content: "private"}}); err != nil {
			t.Fatal(err)
		}
		m.RecordDiagnostic(context.Background(), "outcome", map[string]string{"text": "private"})
		if records := readTraces(t, dir); len(records) != 0 {
			t.Fatal("disabled or expired capture wrote content")
		}
	}
}

func TestTraceCapturesRejectedResponses(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
	}{
		{"unreadable", 200, "not json PRIVATE_REPLY"},
		{"provider", 429, `{"error":"PRIVATE_ERROR"}`},
		{"cutoff", 200, `{"choices":[{"finish_reason":"length","message":{"content":"PRIVATE_PARTIAL"}}],"usage":{"cost":0.001}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			trace, err := NewTrace(dir, time.Now().Add(time.Hour))
			if err != nil {
				t.Fatal(err)
			}
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(tc.status); w.Write([]byte(tc.body)) }))
			defer srv.Close()
			m := &Client{Key: "secret", URL: srv.URL, Trace: trace}
			if _, err = m.Complete(context.Background(), nil); err == nil {
				t.Fatal("expected rejection")
			}
			records := readTraces(t, dir)
			r := first(records, "response")
			if r == nil || r["data"].(map[string]any)["body"] != tc.body || r["data"].(map[string]any)["status"] != float64(tc.status) {
				t.Fatal("rejected response missing")
			}
			if first(records, "model_error") == nil {
				t.Fatal("rejection missing")
			}
		})
	}
}

func TestTraceExpiryDuringRequestStopsResponseCapture(t *testing.T) {
	dir := t.TempDir()
	trace, err := NewTrace(dir, time.Now().Add(150*time.Millisecond))
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(time.Until(trace.until) + time.Millisecond)
		w.Write([]byte(`{"choices":[{"finish_reason":"stop","message":{"content":"ok"}}],"usage":{"cost":0.001}}`))
	}))
	defer srv.Close()
	m := &Client{Key: "k", URL: srv.URL, Trace: trace}
	if _, err = m.Complete(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	records := readTraces(t, dir)
	if len(records) != 1 || records[0]["msg"] != "request" {
		t.Fatal("capture continued after expiry")
	}
}

func TestTraceWriteFailureDoesNotFailModelCallOrLeakContent(t *testing.T) {
	logs := capture(t)
	dir := filepath.Join(t.TempDir(), "private")
	trace, err := NewTrace(dir, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Remove(dir); err != nil {
		t.Fatal(err)
	}
	m := provider(t, `{"choices":[{"finish_reason":"stop","message":{"content":"PRIVATE_REPLY"}}],"usage":{"cost":0.001}}`, nil)
	m.Trace = trace
	if _, err = m.Complete(context.Background(), []assistant.Message{{Role: "user", Content: "PRIVATE_EMAIL"}}); err != nil {
		t.Fatal(err)
	}
	if first(logs(), "agent trace write failed") == nil {
		t.Fatal("missing safe write error")
	}
	b, _ := json.Marshal(logs())
	if strings.Contains(string(b), "PRIVATE_") {
		t.Fatal("content leaked on write error")
	}
}

func TestTraceRejectsSymlinkDirectoryAndSecuresExistingDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "private")
	if err := os.Mkdir(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := NewTrace(dir, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(dir)
	if err != nil || info.Mode().Perm() != 0700 {
		t.Fatal("existing directory stayed public")
	}
	link := filepath.Join(t.TempDir(), "link")
	if err = os.Symlink(dir, link); err != nil {
		t.Fatal(err)
	}
	if _, err = NewTrace(link, time.Now().Add(time.Hour)); err == nil {
		t.Fatal("symlink accepted")
	}
}
