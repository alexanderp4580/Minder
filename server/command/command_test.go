package command

import (
	"errors"
	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/store"
	"path/filepath"
	"sync"
	"testing"
)

func setup(t *testing.T) (*Service, *store.Store) {
	t.Helper()
	s, e := store.Open(filepath.Join(t.TempDir(), "db"))
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { s.Close() })
	return New(s), s
}
func TestPreviewConfirmExactlyOnceUndo(t *testing.T) {
	c, s := setup(t)
	p, e := c.Preview(domain.Proposal{Action: "create_item", Title: "Call plumber", DueAt: "2026-11-12T10:00:00+01:00"})
	if e != nil {
		t.Fatal(e)
	}
	items, _ := s.Items()
	if len(items) != 0 {
		t.Fatal("preview wrote item")
	}
	r, e := c.Confirm(p.Change.ID)
	if e != nil {
		t.Fatal(e)
	}
	again, e := c.Confirm(p.Change.ID)
	if e != nil || again.ID != r.ID {
		t.Fatal("confirm is not idempotent", e)
	}
	items, _ = s.Items()
	if len(items) != 1 || items[0].DueAt != "2026-11-12T09:00:00Z" || items[0].Timezone != "Europe/Amsterdam" {
		t.Fatalf("items: %+v", items)
	}
	n, _ := s.AuditCount()
	if n != 1 {
		t.Fatalf("audit count %d", n)
	}
	_, e = c.Undo(r.Change.UndoID)
	if e != nil {
		t.Fatal(e)
	}
	items, _ = s.Items()
	if len(items) != 0 {
		t.Fatal("undo did not remove item")
	}
	_, e = c.Undo(r.Change.UndoID)
	if !errors.Is(e, domain.ErrConflict) {
		t.Fatalf("second undo: %v", e)
	}
}
func TestUpdateConflictDiscardAndCompleteUndo(t *testing.T) {
	c, s := setup(t)
	p, _ := c.Preview(domain.Proposal{Action: "create_item", Title: "Dentist"})
	c.Confirm(p.Change.ID)
	items, _ := s.Items()
	item := items[0]
	a, _ := c.Preview(domain.Proposal{Action: "update_item", ItemID: item.ID, ExpectedVersion: item.Version, Title: "Call dentist"})
	b, _ := c.Preview(domain.Proposal{Action: "update_item", ItemID: item.ID, ExpectedVersion: item.Version, Notes: "Phone first"})
	r, e := c.Confirm(a.Change.ID)
	if e != nil {
		t.Fatal(e)
	}
	_, e = c.Confirm(b.Change.ID)
	if !errors.Is(e, domain.ErrConflict) {
		t.Fatalf("stale preview: %v", e)
	}
	fresh, e := c.Refresh(b.Change.ID)
	if e != nil || fresh.Change.State != "pending" {
		t.Fatal("no replacement", e)
	}
	if _, e = c.Discard(fresh.Change.ID); e != nil {
		t.Fatal(e)
	}
	if _, e = c.Confirm(fresh.Change.ID); !errors.Is(e, domain.ErrConflict) {
		t.Fatal("discard applied")
	}
	items, _ = s.Items()
	p, _ = c.Preview(domain.Proposal{Action: "complete_item", ItemID: item.ID, ExpectedVersion: items[0].Version})
	done, e := c.Confirm(p.Change.ID)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = c.Undo(r.Change.UndoID); !errors.Is(e, domain.ErrConflict) {
		t.Fatal("undo overwrote later change")
	}
	if _, e = c.Undo(done.Change.UndoID); e != nil {
		t.Fatal(e)
	}
	items, _ = s.Items()
	if items[0].Status != "open" || items[0].CompletedAt != "" || items[0].Title != "Call dentist" {
		t.Fatalf("undo: %+v", items)
	}
}
func TestConcurrentConfirmCreatesOneItem(t *testing.T) {
	c, s := setup(t)
	p, _ := c.Preview(domain.Proposal{Action: "create_item", Title: "One"})
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			if _, e := c.Confirm(p.Change.ID); e != nil {
				t.Error(e)
			}
		})
	}
	wg.Wait()
	items, _ := s.Items()
	n, _ := s.AuditCount()
	if len(items) != 1 || n != 1 {
		t.Fatalf("items %d audits %d", len(items), n)
	}
}
func TestValidation(t *testing.T) {
	c, _ := setup(t)
	for _, p := range []domain.Proposal{{Action: "create_item"}, {Action: "create_item", Title: "x", Timezone: "not/a/zone"}, {Action: "create_item", Title: "x", DueAt: "tomorrow"}, {Action: "delete_item"}, {Action: "update_item", ItemID: "invented", ExpectedVersion: 1, Title: "x"}} {
		if _, e := c.Preview(p); e == nil {
			t.Fatalf("accepted %+v", p)
		}
	}
}
