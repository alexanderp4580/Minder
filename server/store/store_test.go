package store

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/alexanderp4580/Minder/server/domain"
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
