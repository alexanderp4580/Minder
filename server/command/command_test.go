package command

import (
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/reminder"
	"github.com/alexanderp4580/Minder/server/store"
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

func TestReminderPreviewRequiresConfirmationAndValidSchedule(t *testing.T) {
	c, s := setup(t)
	p, e := c.Preview(domain.Proposal{Action: "create_item", Title: "Take bins out", RemindAt: "2026-11-12T18:00:00+01:00", NagMinutes: 15})
	if e != nil {
		t.Fatal(e)
	}
	items, _ := s.Items()
	if len(items) != 0 {
		t.Fatal("preview wrote reminder item")
	}
	if _, e = c.Confirm(p.Change.ID); e != nil {
		t.Fatal(e)
	}
	items, _ = s.Items()
	if items[0].RemindAt != "2026-11-12T17:00:00Z" || items[0].NagMinutes != 15 {
		t.Fatalf("schedule %+v", items)
	}
	found := false
	for _, f := range p.Change.Fields {
		if f.Name == "remind_at" {
			found = true
		}
	}
	if !found {
		t.Fatal("reminder omitted from preview")
	}
	for _, bad := range []domain.Proposal{{Action: "create_item", Title: "x", RemindAt: "tomorrow"}, {Action: "create_item", Title: "x", NagMinutes: 15}, {Action: "create_item", Title: "x", RemindAt: "2026-11-12T17:00:00Z", NagMinutes: -1}, {Action: "create_item", Title: "x", RemindAt: "2026-11-12T17:00:00Z", NagMinutes: 1441}} {
		if _, e = c.Preview(bad); e == nil {
			t.Fatalf("accepted %+v", bad)
		}
	}
}

func TestNotificationDoneCanUndoWithoutRearmingOtherChanges(t *testing.T) {
	c, s := setup(t)
	now := time.Now().UTC().Truncate(time.Second)
	p, e := c.Preview(domain.Proposal{Action: "create_item", Title: "Bins", RemindAt: now.Format(time.RFC3339), NagMinutes: 15})
	if e != nil {
		t.Fatal(e)
	}
	c.Confirm(p.Change.ID)
	reminder.New(s).ScheduleDue(now)
	active, _ := s.ActiveReminders()
	result, e := reminder.New(s).Act(active[0].Reminder.ID, "done", now)
	if e != nil {
		t.Fatal(e)
	}
	if result.Change == nil || result.Change.UndoID == "" {
		t.Fatal("notification Done has no Undo")
	}
	if _, e = c.Undo(result.Change.UndoID); e != nil {
		t.Fatal(e)
	}
	reminder.New(s).ScheduleDue(now)
	active, _ = s.ActiveReminders()
	if len(active) != 1 {
		t.Fatal("Undo lost reminder")
	}
}
