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
