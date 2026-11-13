package command

import (
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/port"
)

type Repository interface {
	port.UnitOfWork
	Items() ([]domain.Item, error)
	Profiles() ([]domain.Profile, error)
}
type Service struct{ repository Repository }

func New(s Repository) *Service { return &Service{s} }
func (c *Service) Preview(p domain.Proposal) (out domain.Entry, err error) {
	err = c.repository.Transact(func(t port.Transaction) error { return previewIn(t, p, &out) })
	logPreview(p, out, err)
	return
}

// fieldNames lists which fields a change touches. Values stay out of the log.
func fieldNames(e domain.Entry) string {
	if e.Change == nil {
		return ""
	}
	names := make([]string, 0, len(e.Change.Fields))
	for _, f := range e.Change.Fields {
		names = append(names, f.Name)
	}
	return strings.Join(names, ",")
}
func logPreview(p domain.Proposal, out domain.Entry, err error) {
	if err != nil {
		slog.Warn("change preview rejected", "action", p.Action, "item_id", p.ItemID, "profile_id", p.ProfileID, "error", err, "error_code", domain.ErrorCode(err))
		return
	}
	slog.Info("change previewed", "action", p.Action, "item_id", p.ItemID, "profile_id", p.ProfileID, "change_id", out.Change.ID, "entry_id", out.ID, "fields", fieldNames(out), "expected_version", p.ExpectedVersion)
}

// PreviewAll previews several proposals in one transaction. One invalid proposal leaves no preview behind.
func (c *Service) PreviewAll(ps []domain.Proposal) (out []domain.Entry, err error) {
	err = c.repository.Transact(func(t port.Transaction) error {
		for _, p := range ps {
			var entry domain.Entry
			if e := previewIn(t, p, &entry); e != nil {
				return e
			}
			out = append(out, entry)
		}
		return nil
	})
	if err != nil {
		slog.Warn("changes preview rejected", "count", len(ps), "error", err, "error_code", domain.ErrorCode(err))
		return
	}
	for i, entry := range out {
		logPreview(ps[i], entry, nil)
	}
	slog.Info("changes previewed", "count", len(out))
	return
}
func previewIn(t port.Transaction, p domain.Proposal, out *domain.Entry) error {
	var before *domain.Item
	var after domain.Item
	switch p.Action {
	case "delete_capture":
		entry, e := previewCaptureDeletion(t, p)
		*out = entry
		return e
	case "create_item":
		after = domain.Item{ID: domain.NewID(), Title: p.Title, DueAt: p.DueAt, Timezone: p.Timezone, Notes: p.Notes, RemindAt: p.RemindAt, NagMinutes: p.NagMinutes, Status: "open", Version: 1, CreatedAt: domain.Now(), UpdatedAt: domain.Now()}
		if e := setAttributes(t, p, &after); e != nil {
			return e
		}
	case "update_item", "complete_item", "cancel_item":
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
		if p.Action == "cancel_item" {
			after.Status = "cancelled"
		} else if p.Action == "complete_item" {
			if e := domain.Complete(&after, time.Now()); e != nil {
				return e
			}
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
			if e := setAttributes(t, p, &after); e != nil {
				return e
			}
		}
	case "create_profile", "update_profile":
		entry, e := previewProfile(t, p)
		*out = entry
		return e
	default:
		return &domain.Invalid{Field: "action", Message: "Choose create, change or complete."}
	}
	if e := domain.Validate(&after); e != nil {
		return e
	}
	id := domain.NewID()
	old := domain.Item{}
	if before != nil {
		old = *before
	}
	fields := itemFields(t, old, after)
	var e error
	change := &domain.Change{ID: id, Action: p.Action, ItemTitle: after.Title, Fields: fields, State: "pending"}
	setReminderChoices(change, after)
	*out, e = t.PutEntry(domain.Entry{CaptureID: p.CaptureID, Kind: "preview", Text: "Review this change.", Change: change})
	if e != nil {
		return e
	}
	return t.PutPending(domain.Pending{ID: id, Proposal: p, Before: before, After: after, State: "pending", EntryID: out.ID})
}
func (c *Service) Confirm(id string) (out domain.Entry, err error) {
	repeated, itemID, version := false, "", 0
	err = c.repository.Transact(func(t port.Transaction) error {
		p, e := t.Pending(id)
		if e != nil {
			return e
		}
		itemID, version = p.After.ID, p.After.Version
		if p.State == "confirmed" {
			repeated = true
			out, e = t.Entry(p.ResultID)
			return e
		}
		if p.State != "pending" {
			slog.Warn("change confirm blocked", "change_id", id, "state", p.State)
			return domain.ErrConflict
		}
		if p.BeforeCapture != nil {
			return confirmCaptureDeletion(t, &p, &out)
		}
		if e = validateCaptureRevision(t, p.Proposal); e != nil {
			return e
		}
		if p.AfterProfile != nil {
			return confirmProfile(t, &p, &out)
		}
		if p.Before != nil {
			i, e := t.Item(p.Before.ID)
			if e != nil {
				if errors.Is(e, domain.ErrNotFound) {
					return domain.ErrConflict
				}
				return e
			}
			if i.Version != p.Before.Version {
				return domain.ErrConflict
			}
		}
		if e = t.PutItem(p.After); e != nil {
			return e
		}
		audit := domain.Audit{ID: domain.NewID(), Before: p.Before, After: p.After, Actor: "agent"}
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
		out, e = t.PutEntry(domain.Entry{CaptureID: p.Proposal.CaptureID, Kind: "result", Text: "Saved: " + p.After.Title + ".", Change: &domain.Change{ID: p.ID, Action: p.Proposal.Action, ItemTitle: p.After.Title, Fields: preview.Change.Fields, State: "confirmed", UndoID: audit.ID}})
		if e != nil {
			return e
		}
		p.State = "confirmed"
		p.ResultID = out.ID
		if e = updateCaptureCandidate(t, p, "confirmed"); e != nil {
			return e
		}
		return t.PutPending(p)
	})
	switch {
	case err != nil:
		slog.Warn("change confirm rejected", "change_id", id, "item_id", itemID, "error", err, "error_code", domain.ErrorCode(err))
	case repeated:
		slog.Info("change confirm repeated", "change_id", id, "item_id", itemID, "entry_id", out.ID)
	default:
		action := ""
		undo := ""
		if out.Change != nil {
			action, undo = out.Change.Action, out.Change.UndoID
		}
		slog.Info("change confirmed", "change_id", id, "action", action, "item_id", itemID, "version", version, "entry_id", out.ID, "undo_id", undo)
	}
	return
}
func (c *Service) Discard(id string) (out domain.Entry, err error) {
	err = c.repository.Transact(func(t port.Transaction) error {
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
		if e = updateCaptureCandidate(t, p, "discarded"); e != nil {
			return e
		}
		return t.PutPending(p)
	})
	if err != nil {
		slog.Warn("change discard rejected", "change_id", id, "error", err, "error_code", domain.ErrorCode(err))
	} else {
		slog.Info("change discarded", "change_id", id, "entry_id", out.ID)
	}
	return
}
func (c *Service) Undo(id string) (out domain.Entry, err error) {
	itemID := ""
	err = c.repository.Transact(func(t port.Transaction) error {
		a, e := t.Audit(id)
		if e != nil {
			return e
		}
		itemID = a.After.ID
		if a.Undone {
			slog.Warn("undo blocked", "audit_id", id, "reason", "already undone")
			return domain.ErrConflict
		}
		if a.AfterProfile != nil {
			return undoProfile(t, &a, &out)
		}
		current, e := t.Item(a.After.ID)
		if e != nil {
			if errors.Is(e, domain.ErrNotFound) {
				return domain.ErrConflict
			}
			return e
		}
		if current.Version != a.After.Version {
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
		undoAudit := domain.Audit{ID: domain.NewID(), Before: &current, After: current, Actor: "owner", Undone: true}
		if e = t.PutAudit(undoAudit); e != nil {
			return e
		}
		out, e = t.PutEntry(domain.Entry{Kind: "result", Text: "Undone: " + a.After.Title + "."})
		return e
	})
	if err != nil {
		slog.Warn("undo rejected", "audit_id", id, "item_id", itemID, "error", err, "error_code", domain.ErrorCode(err))
	} else {
		slog.Info("change undone", "audit_id", id, "item_id", itemID, "entry_id", out.ID)
	}
	return
}
func (c *Service) Refresh(id string) (domain.Entry, error) {
	var p domain.Pending
	err := c.repository.Transact(func(t port.Transaction) error {
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
	if p.BeforeProfile != nil {
		profiles, e := c.repository.Profiles()
		if e != nil {
			return domain.Entry{}, e
		}
		found := false
		for _, v := range profiles {
			if v.ID == p.BeforeProfile.ID {
				p.Proposal.ExpectedVersion = v.Version
				found = true
			}
		}
		if !found {
			return domain.Entry{}, domain.ErrNotFound
		}
	}
	if p.Before != nil {
		items, e := c.repository.Items()
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
	fresh, err := c.Preview(p.Proposal)
	if err == nil && fresh.Change != nil {
		slog.Info("change refreshed", "old_change_id", id, "new_change_id", fresh.Change.ID, "expected_version", p.Proposal.ExpectedVersion)
	}
	return fresh, err
}

// setReminderChoices offers reminder offsets on a pending item change that has a start or deadline.
func setReminderChoices(change *domain.Change, item domain.Item) {
	change.ReminderOffsets, change.ReminderChoices = nil, nil
	if change.Action != "create_item" && change.Action != "update_item" {
		return
	}
	if _, ok := domain.NextReminder(domain.Item{StartAt: item.StartAt, DueAt: item.DueAt, StartDate: item.StartDate, DueDate: item.DueDate, Timezone: item.Timezone, ReminderOffsets: []int{10}}, time.Time{}); !ok {
		return
	}
	change.ReminderOffsets = item.ReminderOffsets
	change.ReminderChoices = domain.ReminderOffsetChoices
}

// SetReminders changes the reminder offsets of a pending item change before it is confirmed.
func (c *Service) SetReminders(id string, offsets []int) (out domain.Entry, err error) {
	err = c.repository.Transact(func(t port.Transaction) error {
		p, e := t.Pending(id)
		if e != nil {
			return e
		}
		if p.State != "pending" || p.AfterProfile != nil || p.BeforeCapture != nil {
			return domain.ErrConflict
		}
		entry, e := t.Entry(p.EntryID)
		if e != nil {
			return e
		}
		if len(entry.Change.ReminderChoices) == 0 {
			return domain.ErrConflict
		}
		p.After.ReminderOffsets = offsets
		if e = domain.Validate(&p.After); e != nil {
			return e
		}
		p.Proposal.ReminderOffsets = p.After.ReminderOffsets
		p.Proposal.ClearReminderOffsets = len(p.After.ReminderOffsets) == 0
		old := domain.Item{}
		if p.Before != nil {
			old = *p.Before
		}
		entry.Change.Fields = itemFields(t, old, p.After)
		setReminderChoices(entry.Change, p.After)
		if out, e = t.PutEntry(entry); e != nil {
			return e
		}
		return t.PutPending(p)
	})
	return
}
