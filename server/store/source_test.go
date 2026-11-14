package store

import (
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/port"
)

func openTemp(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func putSource(t *testing.T, s *Store, ref domain.SourceRef) {
	t.Helper()
	if err := s.Transact(func(tx port.Transaction) error { return tx.PutSource(ref) }); err != nil {
		t.Fatal(err)
	}
}

func ids(refs []domain.SourceRef) string {
	out := []string{}
	for _, r := range refs {
		out = append(out, r.ID)
	}
	return strings.Join(out, ",")
}

func TestASourceRoundTripsAndAMissingOneIsNotFound(t *testing.T) {
	s := openTemp(t)
	want := domain.SourceRef{ID: "m1", Kind: domain.SourceEmail, CaptureID: "m1", Label: "Dentist", Sender: "owner@example.com"}
	putSource(t, s, want)
	var got domain.SourceRef
	err := s.Transact(func(tx port.Transaction) (e error) { got, e = tx.Source("m1"); return })
	if err != nil || got != want {
		t.Fatalf("%+v %v", got, err)
	}
	err = s.Transact(func(tx port.Transaction) error { _, e := tx.Source("none"); return e })
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatal(err)
	}
}

func TestFindSourcesMatchesLabelSenderAndChatTextBeyondRecentHistory(t *testing.T) {
	s := openTemp(t)
	putSource(t, s, domain.SourceRef{ID: "m1", Kind: domain.SourceEmail, CaptureID: "m1", Label: "Dentist checkup", Sender: "clinic@example.com"})
	putSource(t, s, domain.SourceRef{ID: "m2", Kind: domain.SourceEmail, CaptureID: "m2", Label: "Flight", Sender: "air@example.com"})
	s.AddEntry(domain.Entry{ID: "c1", Kind: "user", Text: "Remember the plumber on 3 March"})
	for n := range 60 { // pushes every earlier entry out of the recent history window
		s.AddEntry(domain.Entry{Kind: "minder", Text: fmt.Sprint("filler ", n)})
	}
	for query, want := range map[string]string{"dentist": "m1", "AIR@": "m2", "plumber": "c1", "nothing": ""} {
		got, err := s.FindSources(query)
		if err != nil || ids(got) != want {
			t.Fatalf("%q: %q %v", query, ids(got), err)
		}
	}
	got, _ := s.FindSources("plumber")
	if got[0].Kind != domain.SourceChat || got[0].EntryID != "c1" || got[0].Label != "Remember the plumber on 3 March" {
		t.Fatalf("%+v", got[0])
	}
}

func TestFindSourcesTreatsPercentAndUnderscoreLiterally(t *testing.T) {
	s := openTemp(t)
	putSource(t, s, domain.SourceRef{ID: "m1", Kind: domain.SourceEmail, CaptureID: "m1", Label: "100% sure"})
	putSource(t, s, domain.SourceRef{ID: "m2", Kind: domain.SourceEmail, CaptureID: "m2", Label: "Other"})
	if got, _ := s.FindSources("%"); ids(got) != "m1" {
		t.Fatalf("%q", ids(got))
	}
}

func TestFindSourcesSkipsDeletedSourcesAndListsAPublishedChatSourceOnce(t *testing.T) {
	s := openTemp(t)
	putSource(t, s, domain.SourceRef{ID: "m1", Kind: domain.SourceEmail, CaptureID: "m1", Label: "Dentist", Deleted: true})
	s.AddEntry(domain.Entry{ID: "c1", Kind: "user", Text: "Dentist on Friday"})
	putSource(t, s, domain.SourceRef{ID: "c1", Kind: domain.SourceChat, EntryID: "c1", Label: "Dentist on Friday"})
	got, err := s.FindSources("dentist")
	if err != nil || ids(got) != "c1" {
		t.Fatalf("%q %v", ids(got), err)
	}
}

func TestFindSourcesAsksForANarrowerQueryInsteadOfDroppingMatches(t *testing.T) {
	s := openTemp(t)
	for n := range maxSources + 1 {
		putSource(t, s, domain.SourceRef{ID: fmt.Sprint("m", n), Kind: domain.SourceEmail, CaptureID: fmt.Sprint("m", n), Label: "Booking"})
	}
	var invalid *domain.Invalid
	if got, err := s.FindSources("booking"); !errors.As(err, &invalid) || got != nil {
		t.Fatalf("%v %v", got, err)
	}
	if _, err := s.FindSources("  "); !errors.As(err, &invalid) {
		t.Fatalf("a blank query must be refused: %v", err)
	}
}

// A schema-5 database keeps its captures and chat, and its emails become sources.
func TestUpgradingSchemaFiveKeepsDataAndBackfillsEmailSources(t *testing.T) {
	path := filepath.Join(t.TempDir(), "five.db")
	old, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range steps()[:5] {
		script, _ := migrations.ReadFile(name)
		if _, err = old.Exec(string(script)); err != nil {
			t.Fatal(name, err)
		}
	}
	if _, err = old.Exec(`INSERT INTO chat_entry(id,body) VALUES('e1','{"id":"e1","kind":"user","text":"Kept"}');
INSERT INTO event(entry_id,body) VALUES('e1','{}');
INSERT INTO capture(id,body) VALUES('live','{"id":"live","subject":"Dentist","sender":"clinic@example.com","received_at":"2026-11-01T10:00:00Z","entry_id":"card","status":"reviewed"}'),
 ('gone','{"id":"gone","deleted":true,"status":"reviewed"}');
PRAGMA user_version=5`); err != nil {
		t.Fatal(err)
	}
	old.Close()
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if entries, _ := s.History(0, 5); len(entries) != 1 || entries[0].Text != "Kept" {
		t.Fatalf("chat lost: %+v", entries)
	}
	if c, err := s.Capture("live"); err != nil || c.Subject != "Dentist" {
		t.Fatalf("capture lost: %+v %v", c, err)
	}
	var live, gone domain.SourceRef
	err = s.Transact(func(tx port.Transaction) (e error) {
		if live, e = tx.Source("live"); e != nil {
			return e
		}
		gone, e = tx.Source("gone")
		return e
	})
	if err != nil || live.Kind != domain.SourceEmail || live.CaptureID != "live" || live.Label != "Dentist" || live.Sender != "clinic@example.com" || live.EntryID != "card" || live.Deleted || !gone.Deleted {
		t.Fatalf("%+v %+v %v", live, gone, err)
	}
	if found, _ := s.FindSources("dentist"); ids(found) != "live" {
		t.Fatalf("%q", ids(found))
	}
}

func TestRedactingACaptureDeletesItsSourceAndForgetsItsDetails(t *testing.T) {
	s := openTemp(t)
	putSource(t, s, domain.SourceRef{ID: "m1", Kind: domain.SourceEmail, CaptureID: "m1", Label: "Dentist", Sender: "clinic@example.com"})
	s.AddEntry(domain.Entry{ID: "c1", Kind: "user", Text: "Dentist on Friday, SECRETDETAIL"})
	putSource(t, s, domain.SourceRef{ID: "c1", Kind: domain.SourceChat, EntryID: "c1", CaptureID: "c1", Label: "Dentist on Friday"})
	err := s.Transact(func(tx port.Transaction) error {
		if err := tx.RedactCapture("m1"); err != nil {
			return err
		}
		return tx.RedactCapture("c1")
	})
	if err != nil {
		t.Fatal(err)
	}
	var email, chat domain.SourceRef
	s.Transact(func(tx port.Transaction) (e error) { email, e = tx.Source("m1"); chat, e = tx.Source("c1"); return })
	if !email.Deleted || email.Label != "" || email.Sender != "" || !chat.Deleted || chat.Label != "" {
		t.Fatalf("%+v %+v", email, chat)
	}
	if found, _ := s.FindSources("dentist"); len(found) != 0 {
		t.Fatalf("%+v", found)
	}
	history, _ := s.History(0, 10)
	for _, entry := range history {
		if strings.Contains(entry.Text, "SECRETDETAIL") {
			t.Fatal("the pasted text outlived its source")
		}
	}
}
