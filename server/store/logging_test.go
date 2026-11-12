package store

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alexanderp4580/Minder/server/domain"
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
func count(records []map[string]any, msg string) int {
	n := 0
	for _, r := range records {
		if r["msg"] == msg {
			n++
		}
	}
	return n
}

func TestOpeningLogsPathMigrationsAndSeeding(t *testing.T) {
	logs := capture(t)
	path := filepath.Join(t.TempDir(), "minder.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if count(logs(), "migration applied") != len(steps()) {
		t.Fatalf("want %d migration logs, got %d", len(steps()), count(logs(), "migration applied"))
	}
	if m := first(logs(), "migration applied"); m["name"] == nil || m["from_version"] != float64(0) || m["duration_ms"] == nil {
		t.Fatalf("migration log incomplete: %v", m)
	}
	o := first(logs(), "database opened")
	if o == nil || o["path"] != path || o["schema_version"] != float64(len(steps())) || o["migrations_applied"] != float64(len(steps())) {
		t.Fatalf("open log incomplete: %v", o)
	}
	s.Close()
	logs2 := capture(t)
	s, _ = Open(path)
	if o := first(logs2(), "database opened"); o["migrations_applied"] != float64(0) {
		t.Fatalf("reopen should apply none: %v", o)
	}
}

func TestItemAndReminderChangesAreLoggedWithoutTitles(t *testing.T) {
	logs := capture(t)
	s, _ := Open(filepath.Join(t.TempDir(), "db"))
	defer s.Close()
	at := time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
	s.Write(func(tx *Tx) error {
		return tx.PutItem(domain.Item{ID: "i1", Title: "PRIVATEDENTIST", Status: "open", Version: 1, DueAt: at, RemindAt: at, NagMinutes: 10})
	})
	saved := first(logs(), "item saved")
	if saved == nil || saved["item_id"] != "i1" || saved["version"] != float64(1) || saved["status"] != "open" || saved["created"] != true || saved["reminder_active"] != true || saved["reminder_at"] != at || saved["nag_minutes"] != float64(10) || saved["has_due"] != true {
		t.Fatalf("item log incomplete: %v", saved)
	}
	if r := first(logs(), "reminder schedule set"); r == nil || r["item_id"] != "i1" || r["at"] != at {
		t.Fatalf("schedule not logged: %v", r)
	}
	s.Write(func(tx *Tx) error {
		return tx.PutItem(domain.Item{ID: "i1", Title: "PRIVATEDENTIST", Status: "done", Version: 2, DueAt: at})
	})
	if r := first(logs(), "reminder schedule cancelled"); r == nil || r["item_id"] != "i1" {
		t.Fatalf("cancel not logged: %v", r)
	}
	s.Write(func(tx *Tx) error { return tx.DeleteItem("i1") })
	if d := first(logs(), "item deleted"); d == nil || d["item_id"] != "i1" {
		t.Fatalf("delete not logged: %v", d)
	}
	for _, l := range logs() {
		b, _ := json.Marshal(l)
		if strings.Contains(string(b), "PRIVATEDENTIST") {
			t.Fatalf("title reached the log: %s", b)
		}
	}
}

func TestDeviceRegistrationAndQueueingAreLogged(t *testing.T) {
	logs := capture(t)
	s, _ := Open(filepath.Join(t.TempDir(), "db"))
	defer s.Close()
	now := time.Now().UTC().Format(time.RFC3339)
	s.Write(func(tx *Tx) error {
		return tx.PutItem(domain.Item{ID: "i1", Title: "T", Status: "open", Version: 1, RemindAt: now})
	})
	if e := s.RegisterDevice("d1", "token-0123456789"); e != nil {
		t.Fatal(e)
	}
	r := first(logs(), "push device registered")
	if r == nil || r["device_id"] != "d1" || r["token_len"] != float64(16) || r["queued_deliveries"] != float64(0) || r["devices_total"] != float64(1) {
		t.Fatalf("registration not logged: %v", r)
	}
	s.RegisterDevice("d2", "token-0123456789")
	if r := first(logs(), "push device replaced"); r == nil || r["device_id"] != "d2" || r["removed_devices"] != float64(1) {
		t.Fatalf("token reassignment not logged: %v", r)
	}
}

func TestSlowTransactionsAndRollbacksAreLogged(t *testing.T) {
	logs := capture(t)
	s, _ := Open(filepath.Join(t.TempDir(), "db"))
	defer s.Close()
	old := slowTransaction
	slowTransaction = 0
	defer func() { slowTransaction = old }()
	s.Write(func(tx *Tx) error { return nil })
	if r := first(logs(), "slow transaction"); r == nil || r["level"] != "WARN" || r["duration_ms"] == nil {
		t.Fatalf("slow transaction not logged: %v", r)
	}
	s.Write(func(tx *Tx) error { return domain.ErrConflict })
	if r := first(logs(), "transaction rolled back"); r == nil || r["level"] != "DEBUG" || r["error"] == nil {
		t.Fatalf("rollback not logged: %v", r)
	}
}

func TestReminderStatsCountThePipeline(t *testing.T) {
	s, _ := Open(filepath.Join(t.TempDir(), "db"))
	defer s.Close()
	now := time.Now().UTC().Truncate(time.Second)
	s.Write(func(tx *Tx) error {
		return tx.PutItem(domain.Item{ID: "i1", Title: "T", Status: "open", Version: 1, RemindAt: now.Add(-time.Minute).Format(time.RFC3339), NagMinutes: 5})
	})
	s.RegisterDevice("d1", "token-0123456789")
	st, err := s.ReminderStats(now)
	if err != nil {
		t.Fatal(err)
	}
	if st.Items != 1 || st.Devices != 1 || st.SchedulesActive != 1 || st.NextScheduleAt == "" {
		t.Fatalf("%+v", st)
	}
}
