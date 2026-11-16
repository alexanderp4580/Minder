package store

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/port"
)

func TestMigrationAndDurableHistory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "minder.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	entry, err := s.AddEntry(domain.Entry{Kind: "user", Text: "Call dentist"})
	if err != nil {
		t.Fatal(err)
	}
	if entry.ID == "" || entry.Cursor < 1 {
		t.Fatal("entry lacks cursor")
	}
	s.Close()
	s, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	entries, err := s.History(0, 50)
	if err != nil || len(entries) != 1 || entries[0].Text != "Call dentist" {
		t.Fatalf("history: %v %v", entries, err)
	}
}

// Invalid status or negative versions must fail even outside the command layer.
func TestSchemaRejectsInvalidItemState(t *testing.T) {
	s, e := Open(filepath.Join(t.TempDir(), "schema.db"))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	for _, b := range []string{`{"title":"Bad","status":"typo","version":1}`, `{"title":"Bad","status":"open","version":0}`, `{"title":"","status":"open","version":1}`} {
		if _, e = s.db.Exec("INSERT INTO item(id,body) VALUES(?,?)", domain.NewID(), b); e == nil {
			t.Fatalf("invalid item accepted: %s", b)
		}
	}
}

func TestUsageReservationBlocksAndSettles(t *testing.T) {
	s, e := Open(filepath.Join(t.TempDir(), "usage.db"))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	id, e := s.ReserveUsage(0.03, 0.02)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.ReserveUsage(0.03, 0.02); e != domain.ErrLimit {
		t.Fatalf("budget allowed overlapping request: %v", e)
	}
	if e = s.SettleUsage(id, 0.001, map[string]any{"cost_usd": 0.001}); e != nil {
		t.Fatal(e)
	}
	cost, _ := s.MonthCost()
	if cost != 0.001 {
		t.Fatalf("cost %v", cost)
	}
	if _, e = s.ReserveUsage(0.03, 0.02); e != nil {
		t.Fatal(e)
	}
}

func userVersion(t *testing.T, s *Store) (v int) {
	t.Helper()
	if e := s.db.QueryRow("PRAGMA user_version").Scan(&v); e != nil {
		t.Fatal(e)
	}
	return v
}

// A database written by an older release upgrades in place and keeps its data.
func TestOpenUpgradesAnOlderDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old.db")
	old, e := sql.Open("sqlite3", path)
	if e != nil {
		t.Fatal(e)
	}
	first, e := migrations.ReadFile("migrations/001_items_and_chat.sql")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = old.Exec(string(first)); e != nil {
		t.Fatal(e)
	}
	if _, e = old.Exec(`INSERT INTO chat_entry(id,body) VALUES('e1','{"id":"e1","kind":"user","text":"Kept"}'); INSERT INTO event(entry_id,body) VALUES('e1','{}'); PRAGMA user_version=1`); e != nil {
		t.Fatal(e)
	}
	old.Close()
	s, e := Open(path)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	if v := userVersion(t, s); v != len(steps()) {
		t.Fatalf("user_version %d, want %d", v, len(steps()))
	}
	if entries, e := s.History(0, 10); e != nil || len(entries) != 1 || entries[0].Text != "Kept" {
		t.Fatalf("data lost: %v %v", entries, e)
	}
	if ps, _ := s.Profiles(); len(ps) != 5 {
		t.Fatalf("built-in profiles missing: %d", len(ps))
	}
}

func TestOpenRefusesADatabaseFromANewerRelease(t *testing.T) {
	path := filepath.Join(t.TempDir(), "new.db")
	s, e := Open(path)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.db.Exec("PRAGMA user_version=99"); e != nil {
		t.Fatal(e)
	}
	s.Close()
	if s, e = Open(path); e == nil {
		s.Close()
		t.Fatal("a newer database was opened")
	}
}

func TestBoardGroupsUndatedTasksAndLeavesDatedItemsOut(t *testing.T) {
	s, e := Open(filepath.Join(t.TempDir(), "db"))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	now := domain.Now()
	for _, i := range []domain.Item{
		{ID: "a", Title: "Call plumber", Status: "open", Kind: "task", Timezone: "UTC", Version: 1, CreatedAt: now, UpdatedAt: now},
		{ID: "b", Title: "Renew passport", Status: "in_progress", Kind: "task", Timezone: "UTC", Version: 1, CreatedAt: now, UpdatedAt: now},
		{ID: "c", Title: "Dentist", Status: "open", Kind: "task", Timezone: "UTC", DueAt: "2026-12-01T10:00:00Z", Version: 1, CreatedAt: now, UpdatedAt: now},
	} {
		if e = s.Transact(func(tx port.Transaction) error { return tx.PutItem(i) }); e != nil {
			t.Fatal(e)
		}
	}
	b, e := s.Board(time.Now())
	if e != nil || len(b.Open) != 1 || b.Open[0].ID != "a" || len(b.InProgress) != 1 || len(b.Done) != 0 {
		t.Fatal(b, e)
	}
}

func TestBoardReadMatchesFilteringEveryItemInGo(t *testing.T) {
	s, e := Open(filepath.Join(t.TempDir(), "db"))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	now := time.Now().UTC()
	stamp := func(days int) string { return now.AddDate(0, 0, days).Format(time.RFC3339Nano) }
	base := func(id, status string, edit func(*domain.Item)) domain.Item {
		i := domain.Item{ID: id, Title: id, Status: status, Kind: "task", Timezone: "UTC", Version: 1, CreatedAt: stamp(-30), UpdatedAt: stamp(-30)}
		if edit != nil {
			edit(&i)
		}
		return i
	}
	items := []domain.Item{
		base("open", "open", nil),
		base("doing", "in_progress", func(i *domain.Item) { i.Priority = 2 }),
		base("done-new", "done", func(i *domain.Item) { i.CompletedAt = stamp(-2) }),
		base("done-old", "done", func(i *domain.Item) { i.CompletedAt = stamp(-20) }),
		base("done-unstamped-old", "done", nil),
		base("cancelled", "cancelled", nil),
		base("dated", "open", func(i *domain.Item) { i.DueAt = "2026-12-01T10:00:00Z" }),
		base("start-date", "open", func(i *domain.Item) { i.StartDate = "2026-12-01" }),
		base("event", "open", func(i *domain.Item) { i.Kind = "event" }),
		base("recurring", "open", func(i *domain.Item) { i.Recurrence = &domain.Recurrence{Every: "daily", Interval: 1} }),
		base("reminder-only", "open", func(i *domain.Item) { i.RemindAt = stamp(3) }),
	}
	for _, i := range items {
		if e = s.Transact(func(tx port.Transaction) error { return tx.PutItem(i) }); e != nil {
			t.Fatal(e)
		}
	}
	all, e := s.Items()
	if e != nil {
		t.Fatal(e)
	}
	want, got := domain.BoardColumns(all, now), mustBoard(t, s, now)
	ids := func(col []domain.Item) string {
		out := ""
		for _, i := range col {
			out += i.ID + " "
		}
		return out
	}
	for name, pair := range map[string][2][]domain.Item{"open": {want.Open, got.Open}, "in progress": {want.InProgress, got.InProgress}, "done": {want.Done, got.Done}} {
		if ids(pair[0]) != ids(pair[1]) {
			t.Fatalf("%s: SQL read %q, Go filter %q", name, ids(pair[1]), ids(pair[0]))
		}
	}
	if ids(got.Open) != "open reminder-only " || ids(got.InProgress) != "doing " || ids(got.Done) != "done-new " {
		t.Fatalf("unexpected board: %q %q %q", ids(got.Open), ids(got.InProgress), ids(got.Done))
	}
}

func mustBoard(t *testing.T, s *Store, now time.Time) domain.Board {
	t.Helper()
	b, e := s.Board(now)
	if e != nil {
		t.Fatal(e)
	}
	return b
}
