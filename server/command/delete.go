package command

import (
	"errors"
	"fmt"

	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/port"
)

// previewItemDeletion shows what a delete takes away. It changes no item.
func previewItemDeletion(t port.Transaction, p domain.Proposal) (domain.Entry, error) {
	item, e := t.Item(p.ItemID)
	if e != nil {
		return domain.Entry{}, e
	}
	if p.ExpectedVersion < 1 || p.ExpectedVersion != item.Version {
		return domain.Entry{}, domain.ErrConflict
	}
	fields := []domain.Field{{Name: "item", Old: item.Title, New: "Deleted"}}
	if item.RemindAt != "" || len(item.ReminderOffsets) > 0 {
		fields = append(fields, domain.Field{Name: "reminders", Old: "Scheduled", New: "Removed"})
	}
	if r := item.Recurrence; r != nil {
		repeats := "Repeats " + r.Every
		if r.Interval > 1 {
			repeats += fmt.Sprintf(", every %d", r.Interval)
		}
		fields = append(fields, domain.Field{Name: "recurrence", Old: repeats, New: "All occurrences removed"})
	}
	if sources, e := t.SourcesForItem(item.ID); e != nil {
		return domain.Entry{}, e
	} else if len(sources) > 0 {
		fields = append(fields, domain.Field{Name: "email source", Old: "Original email and attachments", New: "Deleted unless another item uses it"})
	}
	id := domain.NewID()
	entry, e := t.PutEntry(domain.Entry{Kind: "preview", Text: "Delete this item and its data. Undo restores the item and its reminders, not its chat cards or email source.", Change: &domain.Change{ID: id, Action: "delete_item", ItemTitle: item.Title, Fields: fields, State: "pending"}})
	if e != nil {
		return entry, e
	}
	return entry, t.PutPending(domain.Pending{ID: id, Proposal: p, Before: &item, After: item, State: "pending", EntryID: entry.ID})
}

// confirmItemDeletion removes the item and its data in the caller's transaction.
func confirmItemDeletion(t port.Transaction, p *domain.Pending, out *domain.Entry) error {
	current, e := t.Item(p.Before.ID)
	if e != nil {
		if errors.Is(e, domain.ErrNotFound) {
			return domain.ErrConflict
		}
		return e
	}
	if current.Version != p.Before.Version {
		return domain.ErrConflict
	}
	purged, e := t.PurgeItem(current.ID, p.ID)
	if e != nil {
		return e
	}
	for _, id := range purged.OrphanSources {
		c, e := t.Capture(id)
		if e != nil {
			return e
		}
		if c.Deleted {
			continue
		}
		if e = t.PutCapture(domain.Capture{ID: c.ID, Deleted: true, Revision: c.Revision + 1, Status: "reviewed", EntryID: c.EntryID}); e != nil {
			return e
		}
		if e = t.RedactCapture(c.ID); e != nil {
			return e
		}
	}
	audit := domain.Audit{ID: domain.NewID(), Before: p.Before, After: *p.Before, Actor: "agent", ItemDeleted: true}
	if e = t.PutAudit(audit); e != nil {
		return e
	}
	preview, e := t.Entry(p.EntryID)
	if e != nil {
		return e
	}
	preview.Change.State = "confirmed"
	if _, e = t.PutEntry(preview); e != nil {
		return e
	}
	*out, e = t.PutEntry(domain.Entry{Kind: "result", Text: "Deleted: " + current.Title + ".", Change: &domain.Change{ID: p.ID, Action: "delete_item", ItemTitle: current.Title, State: "confirmed", UndoID: audit.ID, RemovedEntryIDs: purged.EntryIDs}})
	if e != nil {
		return e
	}
	p.State = "confirmed"
	p.ResultID = out.ID
	return t.PutPending(*p)
}

// undoItemDeletion puts the deleted item back, which re-arms its reminder.
func undoItemDeletion(t port.Transaction, a *domain.Audit, out *domain.Entry) error {
	if _, e := t.Item(a.Before.ID); e == nil {
		return domain.ErrConflict
	} else if !errors.Is(e, domain.ErrNotFound) {
		return e
	}
	if e := t.PutItem(*a.Before); e != nil {
		return e
	}
	a.Undone = true
	if e := t.PutAudit(*a); e != nil {
		return e
	}
	var e error
	*out, e = t.PutEntry(domain.Entry{Kind: "result", Text: "Undone: " + a.Before.Title + "."})
	return e
}
