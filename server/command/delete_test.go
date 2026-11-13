package command

import (
	"errors"
	"testing"

	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/store"
)

// savedItem creates an item through the normal preview and Confirm flow and returns it with the IDs of its two cards.
func savedItem(t *testing.T, c *Service, title string) (domain.Item, []string) {
	t.Helper()
	p, e := c.Preview(domain.Proposal{Action: "create_item", Title: title, DueAt: "2099-01-01T10:00:00Z", Timezone: "Europe/Amsterdam", RemindAt: "2099-01-01T09:00:00Z", NagMinutes: 15})
	if e != nil {
		t.Fatal(e)
	}
	r, e := c.Confirm(p.Change.ID)
	if e != nil {
		t.Fatal(e)
	}
	items, _ := c.repository.Items()
	for _, i := range items {
		if i.Title == title {
			return i, []string{p.ID, r.ID}
		}
	}
	t.Fatal("item not saved")
	return domain.Item{}, nil
}

func TestDeletePreviewChangesNothingAndNamesTheItem(t *testing.T) {
	c, s := setup(t)
	item, _ := savedItem(t, c, "Dentist")
	p, e := c.Preview(domain.Proposal{Action: "delete_item", ItemID: item.ID, ExpectedVersion: item.Version})
	if e != nil || p.Kind != "preview" || p.Change.Action != "delete_item" || p.Change.State != "pending" || p.Change.ItemTitle != "Dentist" {
		t.Fatalf("%+v %v", p, e)
	}
	if items, _ := s.Items(); len(items) != 1 {
		t.Fatal("a preview deleted the item")
	}
}

func TestDeleteWithAStaleVersionIsAConflict(t *testing.T) {
	c, s := setup(t)
	item, _ := savedItem(t, c, "Dentist")
	before, _ := s.History(0, 50)
	_, e := c.Preview(domain.Proposal{Action: "delete_item", ItemID: item.ID, ExpectedVersion: item.Version + 1})
	if !errors.Is(e, domain.ErrConflict) {
		t.Fatalf("want conflict, got %v", e)
	}
	after, _ := s.History(0, 50)
	if len(after) != len(before) {
		t.Fatal("a rejected delete left a card behind")
	}
}

func TestConfirmedDeleteRemovesTheItemItsCardsAndAnUnusedSource(t *testing.T) {
	c, s := setup(t)
	item, cards := savedItem(t, c, "Dentist")
	other, _ := savedItem(t, c, "Other")
	if e := s.Write(func(tx *store.Tx) error {
		for _, id := range []string{"only", "shared"} {
			if e := tx.PutCapture(domain.Capture{ID: id, Status: "reviewed", Revision: 1}); e != nil {
				return e
			}
		}
		for _, l := range [][2]string{{item.ID, "only"}, {item.ID, "shared"}, {other.ID, "shared"}} {
			if e := tx.LinkSource(l[0], l[1]); e != nil {
				return e
			}
		}
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	p, e := c.Preview(domain.Proposal{Action: "delete_item", ItemID: item.ID, ExpectedVersion: item.Version})
	if e != nil {
		t.Fatal(e)
	}
	r, e := c.Confirm(p.Change.ID)
	if e != nil || r.Kind != "result" || r.Change.UndoID == "" {
		t.Fatalf("%+v %v", r, e)
	}
	if items, _ := s.Items(); len(items) != 1 || items[0].ID != other.ID {
		t.Fatalf("items left: %+v", items)
	}
	history, _ := s.History(0, 50)
	have := map[string]bool{}
	for _, h := range history {
		have[h.ID] = true
	}
	for _, id := range cards {
		if have[id] {
			t.Fatalf("card %s of the deleted item survived", id)
		}
		found := false
		for _, removed := range r.Change.RemovedEntryIDs {
			found = found || removed == id
		}
		if !found {
			t.Fatalf("result does not list removed card %s: %v", id, r.Change.RemovedEntryIDs)
		}
	}
	if !have[p.ID] || !have[r.ID] {
		t.Fatal("the delete's own preview and result must stay")
	}
	for id, deleted := range map[string]bool{"only": true, "shared": false} {
		var got domain.Capture
		if e := s.Write(func(tx *store.Tx) (e error) { got, e = tx.Capture(id); return }); e != nil || got.Deleted != deleted {
			t.Fatalf("source %s deleted=%v err=%v, want %v", id, got.Deleted, e, deleted)
		}
	}
	if e := s.Write(func(tx *store.Tx) error {
		if _, e := tx.Schedule(item.ID); !errors.Is(e, domain.ErrNotFound) {
			t.Fatalf("reminder schedule survived: %v", e)
		}
		return nil
	}); e != nil {
		t.Fatal(e)
	}
}

func TestUndoOfADeleteRestoresTheItemAndItsReminder(t *testing.T) {
	c, s := setup(t)
	item, _ := savedItem(t, c, "Dentist")
	p, _ := c.Preview(domain.Proposal{Action: "delete_item", ItemID: item.ID, ExpectedVersion: item.Version})
	r, e := c.Confirm(p.Change.ID)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = c.Undo(r.Change.UndoID); e != nil {
		t.Fatal(e)
	}
	items, _ := s.Items()
	if len(items) != 1 || items[0].ID != item.ID || items[0].Title != "Dentist" {
		t.Fatalf("items after undo: %+v", items)
	}
	if e := s.Write(func(tx *store.Tx) error {
		if _, e := tx.Schedule(item.ID); e != nil {
			t.Fatalf("reminder not restored: %v", e)
		}
		return nil
	}); e != nil {
		t.Fatal(e)
	}
}

func TestDiscardedDeleteKeepsTheItem(t *testing.T) {
	c, s := setup(t)
	item, _ := savedItem(t, c, "Dentist")
	p, _ := c.Preview(domain.Proposal{Action: "delete_item", ItemID: item.ID, ExpectedVersion: item.Version})
	if _, e := c.Discard(p.Change.ID); e != nil {
		t.Fatal(e)
	}
	if items, _ := s.Items(); len(items) != 1 {
		t.Fatal("a discarded delete removed the item")
	}
}

func TestDeletePreviewOfARepeatingItemSaysTheWholeSeriesGoes(t *testing.T) {
	c, _ := setup(t)
	p, e := c.Preview(domain.Proposal{Action: "create_item", Title: "Water plants", DueAt: "2099-01-05T10:00:00Z", Timezone: "Europe/Amsterdam", Recurrence: &domain.Recurrence{Every: "weekly", Interval: 1}})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = c.Confirm(p.Change.ID); e != nil {
		t.Fatal(e)
	}
	items, _ := c.repository.Items()
	d, e := c.Preview(domain.Proposal{Action: "delete_item", ItemID: items[0].ID, ExpectedVersion: items[0].Version})
	if e != nil {
		t.Fatal(e)
	}
	var row *domain.Field
	for i := range d.Change.Fields {
		if d.Change.Fields[i].Name == "recurrence" {
			row = &d.Change.Fields[i]
		}
	}
	if row == nil || row.Old != "Repeats weekly" || row.New != "All occurrences removed" {
		t.Fatalf("fields: %+v", d.Change.Fields)
	}
}
