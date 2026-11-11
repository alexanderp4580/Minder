package command

import (
	"errors"
	"fmt"
	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/store"
)

type Service struct{ Store *store.Store }

func New(s *store.Store) *Service { return &Service{s} }
func (c *Service) Preview(p domain.Proposal) (out domain.Entry, err error) {
	err = c.Store.Write(func(t *store.Tx) error {
		var before *domain.Item
		var after domain.Item
		switch p.Action {
		case "create_item":
			after = domain.Item{ID: store.ID(), Title: p.Title, DueAt: p.DueAt, Timezone: p.Timezone, Notes: p.Notes, RemindAt: p.RemindAt, NagMinutes: p.NagMinutes, Status: "open", Version: 1, CreatedAt: domain.Now(), UpdatedAt: domain.Now()}
		case "update_item", "complete_item":
			i, e := t.Item(p.ItemID)
			if e != nil {
				return e
			}
			if p.ExpectedVersion < 1 || p.ExpectedVersion != i.Version {
				return domain.ErrConflict
			}
			before = &i
			after = i
			after.Version++
			after.UpdatedAt = domain.Now()
			if p.Action == "complete_item" {
				if i.Status != "open" && i.Status != "in_progress" {
					return domain.ErrConflict
				}
				after.Status = "done"
				after.CompletedAt = domain.Now()
			} else {
				if p.RemindAt != "" {
					after.RemindAt = p.RemindAt
					after.NagMinutes = p.NagMinutes
				}
				if p.NagMinutes != 0 {
					after.NagMinutes = p.NagMinutes
				}
				if p.ClearNag {
					after.NagMinutes = 0
				}
				if p.ClearReminder {
					after.RemindAt = ""
					after.NagMinutes = 0
				}
				if p.Title != "" {
					after.Title = p.Title
				}
				if p.DueAt != "" || p.ClearDue {
					after.DueAt = p.DueAt
				}
				if p.Notes != "" || p.ClearNotes {
					after.Notes = p.Notes
				}
				if p.Timezone != "" {
					after.Timezone = p.Timezone
				}
			}
		default:
			return &domain.Invalid{Field: "action", Message: "Choose create, change or complete."}
		}
		if e := domain.Validate(&after); e != nil {
			return e
		}
		id := store.ID()
		fields := []domain.Field{}
		old := domain.Item{}
		if before != nil {
			old = *before
		}
		for _, f := range []domain.Field{{Name: "remind_at", Old: old.RemindAt, New: after.RemindAt}, {Name: "nag_minutes", Old: fmt.Sprint(old.NagMinutes), New: fmt.Sprint(after.NagMinutes)}, {Name: "title", Old: old.Title, New: after.Title}, {Name: "due_at", Old: old.DueAt, New: after.DueAt}, {Name: "notes", Old: old.Notes, New: after.Notes}, {Name: "status", Old: old.Status, New: after.Status}, {Name: "timezone", Old: old.Timezone, New: after.Timezone}} {
			if f.Old != f.New {
				fields = append(fields, f)
			}
		}
		var e error
		out, e = t.PutEntry(domain.Entry{Kind: "preview", Text: "Review this change.", Change: &domain.Change{ID: id, Action: p.Action, ItemTitle: after.Title, Fields: fields, State: "pending"}})
		if e != nil {
			return e
		}
		return t.PutPending(domain.Pending{ID: id, Proposal: p, Before: before, After: after, State: "pending", EntryID: out.ID})
	})
	return
}
func (c *Service) Confirm(id string) (out domain.Entry, err error) {
	err = c.Store.Write(func(t *store.Tx) error {
		p, e := t.Pending(id)
		if e != nil {
			return e
		}
		if p.State == "confirmed" {
			out, e = t.Entry(p.ResultID)
			return e
		}
		if p.State != "pending" {
			return domain.ErrConflict
		}
		if p.Before != nil {
			i, e := t.Item(p.Before.ID)
			if e != nil || i.Version != p.Before.Version {
				return domain.ErrConflict
			}
		}
		if e = t.PutItem(p.After); e != nil {
			return e
		}
		audit := domain.Audit{ID: store.ID(), Before: p.Before, After: p.After, Actor: "agent"}
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
		out, e = t.PutEntry(domain.Entry{Kind: "result", Text: "Saved: " + p.After.Title + ".", Change: &domain.Change{ID: p.ID, Action: p.Proposal.Action, ItemTitle: p.After.Title, Fields: preview.Change.Fields, State: "confirmed", UndoID: audit.ID}})
		if e != nil {
			return e
		}
		p.State = "confirmed"
		p.ResultID = out.ID
		return t.PutPending(p)
	})
	return
}
func (c *Service) Discard(id string) (out domain.Entry, err error) {
	err = c.Store.Write(func(t *store.Tx) error {
		p, e := t.Pending(id)
		if e != nil {
			return e
		}
		if p.State != "pending" && p.State != "discarded" {
			return domain.ErrConflict
		}
		out, e = t.Entry(p.EntryID)
		if e != nil {
			return e
		}
		if p.State == "discarded" {
			return nil
		}
		out.Change.State = "discarded"
		out.Text = "Change discarded."
		out, e = t.PutEntry(out)
		if e != nil {
			return e
		}
		p.State = "discarded"
		return t.PutPending(p)
	})
	return
}
func (c *Service) Undo(id string) (out domain.Entry, err error) {
	err = c.Store.Write(func(t *store.Tx) error {
		a, e := t.Audit(id)
		if e != nil {
			return e
		}
		if a.Undone {
			return domain.ErrConflict
		}
		current, e := t.Item(a.After.ID)
		if e != nil || current.Version != a.After.Version {
			return domain.ErrConflict
		}
		if a.Before == nil {
			e = t.DeleteItem(a.After.ID)
		} else {
			i := *a.Before
			i.Version = current.Version + 1
			i.UpdatedAt = domain.Now()
			e = t.PutItem(i)
		}
		if e != nil {
			return e
		}
		a.Undone = true
		if e = t.PutAudit(a); e != nil {
			return e
		}
		undoAudit := domain.Audit{ID: store.ID(), Before: &current, After: current, Actor: "owner", Undone: true}
		if e = t.PutAudit(undoAudit); e != nil {
			return e
		}
		out, e = t.PutEntry(domain.Entry{Kind: "result", Text: "Undone: " + a.After.Title + "."})
		return e
	})
	return
}
func (c *Service) Refresh(id string) (domain.Entry, error) {
	var p domain.Pending
	err := c.Store.Write(func(t *store.Tx) error {
		var e error
		p, e = t.Pending(id)
		if e != nil {
			return e
		}
		if p.State != "pending" {
			return domain.ErrConflict
		}
		entry, e := t.Entry(p.EntryID)
		if e != nil {
			return e
		}
		entry.Change.State = "failed"
		if _, e = t.PutEntry(entry); e != nil {
			return e
		}
		p.State = "failed"
		return t.PutPending(p)
	})
	if err != nil {
		return domain.Entry{}, err
	}
	if p.Before != nil {
		items, e := c.Store.Items()
		if e != nil {
			return domain.Entry{}, e
		}
		found := false
		for _, i := range items {
			if i.ID == p.Before.ID {
				p.Proposal.ExpectedVersion = i.Version
				found = true
			}
		}
		if !found {
			return domain.Entry{}, domain.ErrNotFound
		}
	}
	return c.Preview(p.Proposal)
}
func ErrorCode(e error) string {
	var invalid *domain.Invalid
	switch {
	case errors.As(e, &invalid):
		return "invalid_field"
	case errors.Is(e, domain.ErrConflict):
		return "conflict"
	case errors.Is(e, domain.ErrNotFound):
		return "not_found"
	case errors.Is(e, domain.ErrLimit):
		return "limit_reached"
	default:
		return "model_error"
	}
}
func Describe(e error) string {
	if ErrorCode(e) == "model_error" {
		return "Minder could not finish. Try again."
	}
	return fmt.Sprint(e)
}
