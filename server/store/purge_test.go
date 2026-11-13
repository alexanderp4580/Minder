package store_test

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/port"
	"github.com/alexanderp4580/Minder/server/store"
)

func TestPurgeItemRemovesItsDataAndKeepsOthers(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	one := domain.Item{ID: "i1", Title: "Dentist", Status: "open", Version: 1, Timezone: "Europe/Amsterdam", DueAt: "2099-01-01T09:00:00Z", RemindAt: "2099-01-01T08:00:00Z", NagMinutes: 15}
	two := domain.Item{ID: "i2", Title: "Other", Status: "open", Version: 1}
	pending := func(id, item, entry, result string) domain.Pending {
		return domain.Pending{ID: id, Proposal: domain.Proposal{Action: "update_item", ItemID: item}, After: domain.Item{ID: item}, EntryID: entry, ResultID: result, State: "confirmed"}
	}
	err = db.Transact(func(tx port.Transaction) error {
		for _, i := range []domain.Item{one, two} {
			if e := tx.PutItem(i); e != nil {
				return e
			}
		}
		for _, c := range []string{"s1", "s2"} {
			if e := tx.PutCapture(domain.Capture{ID: c, Status: "reviewed", Revision: 1}); e != nil {
				return e
			}
		}
		for _, l := range [][2]string{{"i1", "s1"}, {"i1", "s2"}, {"i2", "s2"}} {
			if e := tx.LinkSource(l[0], l[1]); e != nil {
				return e
			}
		}
		for _, id := range []string{"e1", "e2", "r1", "x", "k"} {
			if _, e := tx.PutEntry(domain.Entry{ID: id, Kind: "result", Text: id, Reminder: &domain.Reminder{State: "active"}}); e != nil {
				return e
			}
		}
		for _, p := range []domain.Pending{pending("p1", "i1", "e1", "e2"), pending("p2", "i2", "x", ""), pending("keep", "i1", "k", "")} {
			if e := tx.PutPending(p); e != nil {
				return e
			}
		}
		if e := tx.PutOccurrence(domain.Occurrence{ID: "o1", ItemID: "i1", Generation: "g", State: "active", EntryID: "r1"}); e != nil {
			return e
		}
		for _, a := range []domain.Audit{{ID: "a1", Actor: "agent", After: domain.Item{ID: "i1"}}, {ID: "a2", Actor: "agent", After: domain.Item{ID: "i2"}}} {
			if e := tx.PutAudit(a); e != nil {
				return e
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	var purged port.Purged
	err = db.Transact(func(tx port.Transaction) (e error) {
		purged, e = tx.PurgeItem("i1", "keep")
		return
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(purged.OrphanSources) != 1 || purged.OrphanSources[0] != "s1" {
		t.Fatalf("a source shared with another item must stay: %v", purged.OrphanSources)
	}
	err = db.Transact(func(tx port.Transaction) error {
		if _, e := tx.Item("i1"); !errors.Is(e, domain.ErrNotFound) {
			t.Fatalf("item survived: %v", e)
		}
		if _, e := tx.Item("i2"); e != nil {
			t.Fatalf("other item lost: %v", e)
		}
		if _, e := tx.Schedule("i1"); !errors.Is(e, domain.ErrNotFound) {
			t.Fatalf("schedule survived: %v", e)
		}
		if _, e := tx.Occurrence("o1"); !errors.Is(e, domain.ErrNotFound) {
			t.Fatalf("occurrence survived: %v", e)
		}
		if _, e := tx.Pending("p1"); !errors.Is(e, domain.ErrNotFound) {
			t.Fatalf("pending change survived: %v", e)
		}
		for _, id := range []string{"keep", "p2"} {
			if _, e := tx.Pending(id); e != nil {
				t.Fatalf("pending %s lost: %v", id, e)
			}
		}
		for _, id := range []string{"e1", "e2", "r1"} {
			if _, e := tx.Entry(id); !errors.Is(e, domain.ErrNotFound) {
				t.Fatalf("entry %s survived: %v", id, e)
			}
		}
		for _, id := range []string{"x", "k"} {
			if _, e := tx.Entry(id); e != nil {
				t.Fatalf("entry %s lost: %v", id, e)
			}
		}
		sources, e := tx.SourcesForItem("i2")
		if e != nil || len(sources) != 1 || sources[0] != "s2" {
			t.Fatalf("other item lost its source: %v %v", sources, e)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if n, _ := db.AuditCount(); n != 1 {
		t.Fatalf("audit rows left: %d", n)
	}
	history, _ := db.History(0, 20)
	for _, e := range history {
		if e.ID == "e1" || e.ID == "e2" || e.ID == "r1" {
			t.Fatalf("history still holds %s", e.ID)
		}
	}
	want := map[string]bool{"e1": true, "e2": true, "r1": true}
	for _, id := range purged.EntryIDs {
		delete(want, id)
	}
	if len(want) != 0 {
		t.Fatalf("removed entry IDs incomplete: %v missing %v", purged.EntryIDs, want)
	}
}
