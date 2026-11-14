package assistant

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/store"
)

type diagnosticModel struct {
	Model
	records map[string]json.RawMessage
}

func (m *diagnosticModel) RecordDiagnostic(_ context.Context, event string, data any) {
	b, _ := json.Marshal(data)
	m.records[event] = b
}

func TestDiagnosticsIncludeReviewRejectionAndFinalOutcome(t *testing.T) {
	a, s, f := newAssistant(t, Response{Text: "PRIVATE_FINAL", CostUSD: 0.001})
	f.seq = []Response{{Calls: []Call{call("bad", "propose_review", `{"candidates":[]}`)}, CostUSD: 0.001}}
	m := &diagnosticModel{Model: f, records: map[string]json.RawMessage{}}
	a.model = m
	if _, err := reply(t, a, s, "Read my email"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(m.records["review_rejected"]), "source_id is required") {
		t.Fatal("missing rejection")
	}
	if !strings.Contains(string(m.records["outcome"]), "PRIVATE_FINAL") {
		t.Fatal("missing final outcome")
	}
}

func TestDiagnosticsRecordFailedTurn(t *testing.T) {
	a, s, f := newAssistant(t, Response{Calls: []Call{call("bad", "unknown_tool", `{}`)}, CostUSD: 0.001})
	m := &diagnosticModel{Model: f, records: map[string]json.RawMessage{}}
	a.model = m
	if _, err := reply(t, a, s, "Read my email"); err == nil {
		t.Fatal("expected failure")
	}
	if !strings.Contains(string(m.records["outcome"]), "unknown tool") {
		t.Fatal("missing final failure")
	}
}

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

func TestReplyLogsEachStageAndTheOutcome(t *testing.T) {
	logs := capture(t)
	s, _ := store.Open(filepath.Join(t.TempDir(), "db"))
	defer s.Close()
	f := &fakeModel{r: Response{Calls: []Call{{ID: "c", Name: "propose_create", Arguments: `{"title":"PRIVATEPLUMBER","due_at":"2026-11-12T09:00:00Z"}`}}, CostUSD: 0.002}}
	a := New(s, f, 5, 1.1)
	if _, e := replyIn(t, domain.WithTrace(context.Background(), "reply-1"), a, s, "Call PRIVATEPLUMBER"); e != nil {
		t.Fatal(e)
	}
	start := first(logs(), "assistant reply started")
	if start == nil || start["trace_id"] != "reply-1" || start["kind"] != domain.TurnOwner || start["items_total"] != float64(0) || start["prompt_bytes"] == nil || start["month_cost_eur"] == nil {
		t.Fatalf("start log incomplete: %v", start)
	}
	if r := first(logs(), "usage reserved"); r == nil || r["reservation_id"] == "" || r["amount_eur"] == nil {
		t.Fatalf("reservation not logged: %v", r)
	}
	if r := first(logs(), "usage settled"); r == nil || r["cost_eur"].(float64) < 0.0021 || r["cost_usd"] != 0.002 || r["usd_to_eur"] != 1.1 {
		t.Fatalf("settlement not logged: %v", r)
	}
	done := first(logs(), "assistant reply finished")
	if done == nil || done["proposals"] != float64(1) || done["rounds"] != float64(1) || done["kind"] != domain.TurnOwner || done["duration_ms"] == nil || done["trace_id"] != "reply-1" {
		t.Fatalf("outcome not logged: %v", done)
	}
	for _, l := range logs() {
		b, _ := json.Marshal(l)
		if strings.Contains(string(b), "PRIVATEPLUMBER") {
			t.Fatalf("message text or title reached the log: %s", b)
		}
	}
}

func TestBudgetBlockIsLoggedWithNumbers(t *testing.T) {
	logs := capture(t)
	s, _ := store.Open(filepath.Join(t.TempDir(), "db"))
	defer s.Close()
	s.RecordUsage(5, map[string]any{})
	a := New(s, &fakeModel{}, 5, 1)
	reply(t, a, s, "Hi")
	r := first(logs(), "assistant reply blocked")
	if r == nil || r["reason"] != "monthly_limit" || r["month_cost_eur"] != float64(5) || r["limit_eur"] != float64(5) {
		t.Fatalf("block not logged: %v", r)
	}
}

func TestModelFailureAndBadProposalAreLogged(t *testing.T) {
	logs := capture(t)
	s, _ := store.Open(filepath.Join(t.TempDir(), "db"))
	defer s.Close()
	f := &fakeModel{r: Response{Calls: []Call{{ID: "c", Name: "mystery_tool", Arguments: `{}`}}, CostUSD: 0.001}}
	a := New(s, f, 5, 1)
	if _, e := reply(t, a, s, "x"); e == nil {
		t.Fatal("unknown tool accepted")
	}
	r := first(logs(), "assistant reply failed")
	if r == nil || r["stage"] == nil || !strings.Contains(r["error"].(string), "unknown tool") {
		t.Fatalf("failure not logged: %v", r)
	}
}
