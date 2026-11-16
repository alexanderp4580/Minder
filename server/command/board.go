package command

import (
	"log/slog"
	"strings"
	"time"

	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/port"
)

// AddTask creates an open, undated task for the board. It needs no preview and writes no chat entry.
func (c *Service) AddTask(title string) (domain.Item, error) {
	now := domain.Now()
	item := domain.Item{ID: domain.NewID(), Title: strings.TrimSpace(title), Status: "open", Kind: "task", Version: 1, CreatedAt: now, UpdatedAt: now}
	err := c.repository.Transact(func(t port.Transaction) error {
		if e := domain.Validate(&item); e != nil {
			return e
		}
		_, e := saveItem(t, nil, item, "owner")
		return e
	})
	if err != nil {
		slog.Warn("board item add rejected", "error", err, "error_code", domain.ErrorCode(err))
		return domain.Item{}, err
	}
	slog.Info("board item added", "item_id", item.ID)
	return item, nil
}

// MoveTask sets the status of a task on the board. It needs the current version, writes an
// audit row and no chat entry, and leaves a task with a date to the calendar.
func (c *Service) MoveTask(id string, version int, status string) (domain.Item, error) {
	if !domain.ValidBoardStatus(status) {
		return domain.Item{}, &domain.Invalid{Field: "status", Message: "Use open, in progress or done."}
	}
	var after domain.Item
	from := ""
	err := c.repository.Transact(func(t port.Transaction) error {
		i, e := t.Item(id)
		if e != nil {
			return e
		}
		if version < 1 || version != i.Version {
			return domain.ErrConflict
		}
		if !domain.OnBoard(i) {
			if i.Status == "cancelled" {
				return &domain.Invalid{Field: "status", Message: "This task was cancelled."}
			}
			return &domain.Invalid{Field: "status", Message: "This task is on the calendar."}
		}
		from, after = i.Status, i
		if from == status {
			return nil
		}
		after.Status = status
		after.Version++
		after.UpdatedAt = domain.Now()
		if status == "done" {
			after.CompletedAt = after.UpdatedAt
		} else {
			after.CompletedAt = ""
			if from == "done" {
				clearPastReminder(&after)
			}
		}
		if e = domain.Validate(&after); e != nil {
			return e
		}
		_, e = saveItem(t, &i, after, "owner")
		return e
	})
	if err != nil {
		slog.Warn("board item move rejected", "item_id", id, "to", status, "error", err, "error_code", domain.ErrorCode(err))
		return domain.Item{}, err
	}
	slog.Info("board item moved", "item_id", id, "from", from, "to", status, "version", after.Version)
	return after, nil
}

// clearPastReminder drops a reminder time that has passed. A reopened task must not notify about it again.
func clearPastReminder(i *domain.Item) {
	at, err := time.Parse(time.RFC3339, i.RemindAt)
	if err == nil && !at.After(time.Now()) {
		i.RemindAt, i.NagMinutes = "", 0
	}
}

// movedField tells the owner when an edit moves a task between the Board and the Calendar,
// so a task never changes screens without a word in its preview.
func movedField(before, after domain.Item) []domain.Field {
	was, now := domain.Placement(before), domain.Placement(after)
	if was == "" || now == "" || was == now {
		return nil
	}
	return []domain.Field{{Name: "shown_on", Old: was, New: now}}
}

// saveItem is the one place an item change is written and audited: Confirm uses it for agent
// changes and the board uses it for the owner's own taps. The item write runs the reminder
// projection, so every path keeps schedules in step.
func saveItem(t port.Transaction, before *domain.Item, after domain.Item, actor string) (domain.Audit, error) {
	if err := t.PutItem(after); err != nil {
		return domain.Audit{}, err
	}
	audit := domain.Audit{ID: domain.NewID(), Before: before, After: after, Actor: actor}
	return audit, t.PutAudit(audit)
}
