package reminder

import (
	"errors"
	"log/slog"
	"time"

	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/port"
)

type Repository interface {
	port.UnitOfWork
	ActiveReminders() ([]domain.Entry, error)
	RegisterDevice(string, string) error
}
type Service struct{ repository Repository }

func New(repository Repository) *Service { return &Service{repository: repository} }

func (s *Service) Active() ([]domain.Entry, error) { return s.repository.ActiveReminders() }
func (s *Service) RegisterDevice(id, token string) error {
	if err := domain.ValidateDevice(id, token); err != nil {
		slog.Warn("push device rejected", "reason", "invalid device or token", "device_id", id, "token_len", len(token), "error", err)
		return err
	}
	return s.repository.RegisterDevice(id, token)
}

func (s *Service) ScheduleDue(now time.Time) error {
	return s.repository.Transact(func(tx port.Transaction) error {
		schedules, err := tx.DueSchedules(now)
		if err != nil {
			return err
		}
		for _, schedule := range schedules {
			item, err := tx.Item(schedule.ItemID)
			if err != nil {
				return err
			}
			if err = tx.CloseReminders(item.ID, "superseded"); err != nil {
				return err
			}
			id := domain.NewID()
			rem := &domain.Reminder{ID: id, ItemID: item.ID, Title: item.Title, ScheduledAt: schedule.At.UTC().Format(time.RFC3339), State: "active", Late: now.Unix()-schedule.At.Unix() > 60}
			text := "Reminder: " + item.Title + "."
			if rem.Late {
				text = "Late reminder: " + item.Title + "."
			}
			entry, err := tx.PutEntry(domain.Entry{Kind: "reminder", Text: text, Reminder: rem})
			if err != nil {
				return err
			}
			if err = tx.PutOccurrence(domain.Occurrence{ID: id, ItemID: item.ID, Generation: schedule.Generation, State: "active", EntryID: entry.ID}); err != nil {
				return err
			}
			if err = tx.QueueDeliveries(id, now); err != nil {
				return err
			}
			fired := schedule.At
			scheduleAt := fired.Unix()
			schedule.State = "waiting"
			schedule.At = time.Unix(0, 0)
			after := fired
			if now.After(after) {
				after = now
			}
			if later, ok := domain.NextReminder(item, after); ok {
				schedule.State = "active"
				schedule.At = later
			} else if schedule.NagMinutes > 0 {
				schedule.State = "active"
				schedule.At = now.Add(time.Duration(schedule.NagMinutes) * time.Minute)
			}
			if err = tx.UpdateSchedule(schedule); err != nil {
				return err
			}
			next := "none"
			if schedule.State == "active" {
				next = schedule.At.UTC().Format(time.RFC3339)
			}
			slog.Info("reminder fired", "item_id", item.ID, "occurrence_id", id, "entry_id", entry.ID, "generation", schedule.Generation, "scheduled_at", rem.ScheduledAt, "late", rem.Late, "late_seconds", now.Unix()-scheduleAt, "nag_minutes", schedule.NagMinutes, "next", next)
		}
		return nil
	})
}

func (s *Service) Act(id, action string, now time.Time) (out domain.Entry, err error) {
	switch action {
	case "ack", "done", "snooze15", "snooze60", "tonight", "tomorrow":
	default:
		slog.Warn("reminder action rejected", "reason", "unknown action", "action", action, "occurrence_id", id)
		return out, &domain.Invalid{Field: "action", Message: "Choose Done, Open or a snooze time."}
	}
	next := "none"
	itemID := ""
	err = s.repository.Transact(func(tx port.Transaction) error {
		occurrence, err := tx.Occurrence(id)
		if err != nil {
			return err
		}
		itemID = occurrence.ItemID
		if occurrence.State == action && occurrence.ResultID != "" {
			slog.Info("reminder action repeated", "action", action, "occurrence_id", id, "item_id", itemID)
			out, err = tx.Entry(occurrence.ResultID)
			return err
		}
		if occurrence.State != "active" {
			slog.Warn("reminder action rejected", "reason", "occurrence is "+occurrence.State, "action", action, "occurrence_id", id, "item_id", itemID)
			return domain.ErrConflict
		}
		schedule, err := tx.Schedule(occurrence.ItemID)
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrConflict
		}
		if err != nil {
			return err
		}
		if schedule.Generation != occurrence.Generation {
			slog.Warn("reminder action rejected", "reason", "schedule changed since the reminder fired", "action", action, "occurrence_id", id, "item_id", itemID, "occurrence_generation", occurrence.Generation, "schedule_generation", schedule.Generation)
			return domain.ErrConflict
		}
		item, err := tx.Item(occurrence.ItemID)
		if err != nil {
			return err
		}
		text := "Reminder acknowledged: " + item.Title + "."
		var change *domain.Change
		if action == "done" {
			before := item
			if err = domain.Complete(&item, now); err != nil {
				return err
			}
			item.Version++
			item.UpdatedAt = domain.Now()
			if err = tx.PutItem(item); err != nil {
				return err
			}
			undoID := domain.NewID()
			if err = tx.PutAudit(domain.Audit{ID: undoID, Actor: "owner", Before: &before, After: item}); err != nil {
				return err
			}
			text = "Done: " + item.Title + "."
			change = &domain.Change{Action: "complete_item", ItemTitle: item.Title, State: "confirmed", UndoID: undoID}
		} else {
			if err = tx.CloseReminders(item.ID, action); err != nil {
				return err
			}
			schedule.State = "acknowledged"
			schedule.At = time.Unix(0, 0)
			if action != "ack" {
				schedule.At, err = domain.SnoozeAt(action, item.Timezone, now)
				if err != nil {
					return err
				}
				schedule.State = "active"
				text = "Snoozed: " + item.Title + "."
				next = schedule.At.UTC().Format(time.RFC3339)
			}
			if err = tx.UpdateSchedule(schedule); err != nil {
				return err
			}
		}
		entry, err := tx.Entry(occurrence.EntryID)
		if err != nil {
			return err
		}
		entry.Reminder.State = action
		if _, err = tx.PutEntry(entry); err != nil {
			return err
		}
		out, err = tx.PutEntry(domain.Entry{Kind: "result", Text: text, Change: change})
		if err != nil {
			return err
		}
		occurrence.State = action
		occurrence.ResultID = out.ID
		return tx.PutOccurrence(occurrence)
	})
	if err != nil {
		slog.Warn("reminder action failed", "action", action, "occurrence_id", id, "item_id", itemID, "error", err, "error_code", domain.ErrorCode(err))
		return
	}
	slog.Info("reminder action", "action", action, "occurrence_id", id, "item_id", itemID, "entry_id", out.ID, "next", next)
	return
}
