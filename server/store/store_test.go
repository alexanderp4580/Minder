package store

import (
	"github.com/alexanderp4580/Minder/server/domain"
	"path/filepath"
	"testing"
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
		if _, e = s.db.Exec("INSERT INTO item(id,body) VALUES(?,?)", ID(), b); e == nil {
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
