package reminder

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/store"
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
func all(records []map[string]any, msg string) []map[string]any {
	var out []map[string]any
	for _, r := range records {
		if r["msg"] == msg {
			out = append(out, r)
		}
	}
	return out
}
func fixture(t *testing.T) (*store.Store, time.Time) {
	t.Helper()
	s, e := store.Open(filepath.Join(t.TempDir(), "db"))
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { s.Close() })
	now := time.Now().UTC().Truncate(time.Second)
	s.Write(func(tx *store.Tx) error {
		return tx.PutItem(domain.Item{ID: "x", Title: "PRIVATEBINS", Status: "open", Version: 1, RemindAt: now.Add(-2 * time.Minute).Format(time.RFC3339), NagMinutes: 15})
	})
	return s, now
}

func TestDeviceRegistrationIsLoggedWithFingerprint(t *testing.T) {
	logs := capture(t)
	s, _ := fixture(t)
	if e := New(s).RegisterDevice("phone-1", "TOKENSECRET-0123456789"); e != nil {
		t.Fatal(e)
	}
	r := first(logs(), "push device registered")
	if r == nil || r["device_id"] != "phone-1" || r["token_len"] != float64(22) || r["token_fp"] == nil {
		t.Fatalf("registration not logged: %v", r)
	}
	if r["queued_deliveries"] == nil {
		t.Fatalf("backfilled deliveries not reported: %v", r)
	}
	if e := New(s).RegisterDevice("", "x"); e == nil {
		t.Fatal("invalid device accepted")
	}
	if w := first(logs(), "push device rejected"); w == nil || w["level"] != "WARN" {
		t.Fatalf("invalid registration not logged: %v", w)
	}
	for _, l := range logs() {
		b, _ := json.Marshal(l)
		if strings.Contains(string(b), "TOKENSECRET") {
			t.Fatalf("token reached the log: %s", b)
		}
	}
}

func TestFiredReminderIsLoggedWithLatenessAndQueuedDeliveries(t *testing.T) {
	logs := capture(t)
	s, now := fixture(t)
	s.RegisterDevice("p1", "token-one-0123456789")
	s.RegisterDevice("p2", "token-two-0123456789")
	if e := New(s).ScheduleDue(now); e != nil {
		t.Fatal(e)
	}
	r := first(logs(), "reminder fired")
	if r == nil || r["item_id"] != "x" || r["late"] != true || r["late_seconds"].(float64) < 119 || r["occurrence_id"] == "" || r["next"] == nil {
		t.Fatalf("fired reminder not logged: %v", r)
	}
	if q := first(logs(), "deliveries queued"); q == nil || q["devices"] != float64(2) {
		t.Fatalf("queued deliveries not logged: %v", q)
	}
	for _, l := range logs() {
		b, _ := json.Marshal(l)
		if strings.Contains(string(b), "PRIVATEBINS") {
			t.Fatalf("item title reached the log: %s", b)
		}
	}
}

func TestReminderActionsAreLogged(t *testing.T) {
	logs := capture(t)
	s, now := fixture(t)
	svc := New(s)
	svc.ScheduleDue(now)
	active, _ := s.ActiveReminders()
	if _, e := svc.Act(active[0].Reminder.ID, "snooze15", now); e != nil {
		t.Fatal(e)
	}
	r := first(logs(), "reminder action")
	if r == nil || r["action"] != "snooze15" || r["occurrence_id"] != active[0].Reminder.ID || r["item_id"] != "x" || r["next"] == nil {
		t.Fatalf("action not logged: %v", r)
	}
	if _, e := svc.Act(active[0].Reminder.ID, "bogus", now); e == nil {
		t.Fatal("bad action accepted")
	}
	if w := first(logs(), "reminder action rejected"); w == nil || w["reason"] != "unknown action" {
		t.Fatalf("rejection not logged: %v", w)
	}
}

type recordingSender struct {
	invalid bool
	err     error
}

func (r recordingSender) Send(context.Context, string, map[string]string) (bool, error) {
	return r.invalid, r.err
}

func TestWorkerLogsEveryDeliveryOutcome(t *testing.T) {
	logs := capture(t)
	s, now := fixture(t)
	s.RegisterDevice("p1", "token-one-0123456789")
	w := Worker{Repository: s, Scheduler: New(s), Sender: recordingSender{err: errors.New("Firebase did not accept delivery")}}
	if e := w.Tick(context.Background(), now); e != nil {
		t.Fatal(e)
	}
	r := first(logs(), "push delivery failed")
	if r == nil || r["level"] != "WARN" || r["device_id"] != "p1" || r["error"] != "Firebase did not accept delivery" || r["retry_at"] == nil || r["attempt_ms"] == nil {
		t.Fatalf("failed delivery not logged: %v", r)
	}
	w.Sender = recordingSender{}
	w.Tick(context.Background(), now.Add(time.Minute))
	if ok := first(logs(), "push delivered"); ok == nil || ok["device_id"] != "p1" || ok["occurrence_id"] == "" {
		t.Fatalf("delivered not logged: %v", ok)
	}
	s.Write(func(tx *store.Tx) error {
		return tx.PutItem(domain.Item{ID: "y", Title: "B", Status: "open", Version: 1, RemindAt: now.Add(-time.Minute).Format(time.RFC3339)})
	})
	w.Sender = recordingSender{invalid: true, err: errors.New("unregistered")}
	w.Tick(context.Background(), now.Add(2*time.Minute))
	if g := first(logs(), "push device removed"); g == nil || g["device_id"] != "p1" || g["reason"] != "token rejected by Firebase" {
		t.Fatalf("token removal not logged: %v", g)
	}
	if len(all(logs(), "reminder tick")) == 0 {
		t.Fatal("tick summary missing")
	}
}

func TestWorkerLogsTickFailuresWithCauseAndMissingSender(t *testing.T) {
	logs := capture(t)
	s, now := fixture(t)
	w := Worker{Repository: s, Scheduler: failingScheduler{}, Sender: nil}
	w.runOnce(context.Background())
	r := first(logs(), "reminder worker tick failed")
	if r == nil || r["error"] != "schedule exploded" || r["level"] != "ERROR" {
		t.Fatalf("tick failure cause not logged: %v", r)
	}
	logs2 := capture(t)
	w = Worker{Repository: s, Scheduler: New(s)}
	w.Tick(context.Background(), now)
	if m := first(logs2(), "reminder tick"); m == nil || m["push_sender"] != false {
		t.Fatalf("missing sender not reported: %v", m)
	}
}

func TestHeartbeatReportsQueueState(t *testing.T) {
	logs := capture(t)
	s, now := fixture(t)
	s.RegisterDevice("p1", "token-one-0123456789")
	w := Worker{Repository: s, Scheduler: New(s), Sender: recordingSender{}, Heartbeat: time.Nanosecond}
	w.Tick(context.Background(), now)
	w.Tick(context.Background(), now.Add(time.Second))
	h := first(logs(), "reminder worker heartbeat")
	if h == nil || h["devices"] != float64(1) || h["items"] != float64(1) || h["push_sender"] != true || h["deliveries_pending"] == nil || h["schedules_active"] == nil {
		t.Fatalf("heartbeat incomplete: %v", h)
	}
}

type failingScheduler struct{}

func (failingScheduler) ScheduleDue(time.Time) error { return errors.New("schedule exploded") }
