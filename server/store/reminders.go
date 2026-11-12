package store

import (
	"encoding/json"
	"log/slog"
	"time"

	"github.com/alexanderp4580/Minder/server/domain"
)

func (t *Tx) CloseReminders(itemID, state string) error {
	rows, e := t.tx.Query("SELECT entry_id FROM reminder_occurrence WHERE item_id=? AND state='active'", itemID)
	if e != nil {
		return e
	}
	var ids []string
	for rows.Next() {
		var id string
		if e = rows.Scan(&id); e != nil {
			rows.Close()
			return e
		}
		ids = append(ids, id)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	for _, id := range ids {
		entry, e := t.Entry(id)
		if e != nil {
			return e
		}
		entry.Reminder.State = state
		if _, e = t.PutEntry(entry); e != nil {
			return e
		}
	}
	if _, e = t.tx.Exec("UPDATE reminder_occurrence SET state=? WHERE item_id=? AND state='active'", state, itemID); e != nil {
		return e
	}
	_, e = t.tx.Exec("DELETE FROM push_delivery WHERE occurrence_id IN (SELECT id FROM reminder_occurrence WHERE item_id=?)", itemID)
	return e
}

// applyReminder replaces the schedule when the effective reminder changed. It cancels open occurrences first.
func (t *Tx) applyReminder(itemID string, old, now domain.ReminderProjection) error {
	if old == now {
		slog.Debug("reminder schedule unchanged", "item_id", itemID, "active", now.Active)
		return nil
	}
	if e := t.CloseReminders(itemID, "cancelled"); e != nil {
		return e
	}
	if _, e := t.tx.Exec("DELETE FROM reminder_schedule WHERE item_id=?", itemID); e != nil {
		return e
	}
	if !now.Active {
		slog.Info("reminder schedule cancelled", "item_id", itemID, "was_active", old.Active, "previous_at", old.At)
		return nil
	}
	at, e := time.Parse(time.RFC3339, now.At)
	if e != nil {
		return e
	}
	generation := domain.NewID()
	_, e = t.tx.Exec("INSERT INTO reminder_schedule VALUES(?,?,?,?,?)", itemID, generation, at.Unix(), now.NagMinutes, "active")
	if e == nil {
		slog.Info("reminder schedule set", "item_id", itemID, "generation", generation, "at", now.At, "nag_minutes", now.NagMinutes, "replaced", old.Active)
	}
	return e
}
func (t *Tx) syncReminder(old *domain.Item, i domain.Item) (domain.ReminderProjection, error) {
	var before domain.ReminderProjection
	if old != nil {
		p, e := t.profileOf(old.ProfileID)
		if e != nil {
			return before, e
		}
		before = domain.ProjectReminder(*old, p)
	}
	p, e := t.profileOf(i.ProfileID)
	if e != nil {
		return before, e
	}
	now := domain.ProjectReminder(i, p)
	return now, t.applyReminder(i.ID, before, now)
}
func (s *Store) RegisterDevice(id, token string) error {
	return s.Write(func(t *Tx) error {
		// Reinstallation may assign an existing token to a new local device ID.
		removed, e := t.tx.Exec("DELETE FROM push_device WHERE token=? AND id<>?", token, id)
		if e != nil {
			return e
		}
		replaced, _ := removed.RowsAffected()
		if replaced > 0 {
			slog.Info("push device replaced", "device_id", id, "removed_devices", replaced, "token_fp", domain.Fingerprint(token))
		}
		if _, e := t.tx.Exec("INSERT INTO push_device VALUES(?,?) ON CONFLICT(id) DO UPDATE SET token=excluded.token", id, token); e != nil {
			return e
		}
		queued, e := t.tx.Exec("INSERT OR IGNORE INTO push_delivery(occurrence_id,device_id,next_at) SELECT id,?,0 FROM reminder_occurrence WHERE state='active'", id)
		if e != nil {
			return e
		}
		backfilled, _ := queued.RowsAffected()
		var total int
		t.tx.QueryRow("SELECT COUNT(*) FROM push_device").Scan(&total)
		slog.Info("push device registered", "device_id", id, "token_len", len(token), "token_fp", domain.Fingerprint(token), "queued_deliveries", backfilled, "devices_total", total)
		return nil
	})
}
func (s *Store) ActiveReminders() ([]domain.Entry, error) {
	return queryAll[domain.Entry](s.db, "SELECT e.body FROM reminder_occurrence r JOIN chat_entry e ON e.id=r.entry_id WHERE r.state='active' ORDER BY e.seq")
}
func (s *Store) Deliveries(now time.Time) ([]domain.Delivery, error) {
	rows, e := s.db.Query("SELECT p.occurrence_id,p.device_id,d.token,e.body FROM push_delivery p JOIN push_device d ON d.id=p.device_id JOIN reminder_occurrence r ON r.id=p.occurrence_id JOIN chat_entry e ON e.id=r.entry_id WHERE p.sent=0 AND p.next_at<=? AND r.state='active' ORDER BY p.next_at LIMIT 100", now.Unix())
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []domain.Delivery{}
	for rows.Next() {
		var d domain.Delivery
		var raw string
		if e = rows.Scan(&d.OccurrenceID, &d.DeviceID, &d.Token, &raw); e != nil {
			return nil, e
		}
		var entry domain.Entry
		if e = json.Unmarshal([]byte(raw), &entry); e != nil {
			return nil, e
		}
		d.Title = entry.Reminder.Title
		d.ScheduledAt = entry.Reminder.ScheduledAt
		d.Late = entry.Reminder.Late
		out = append(out, d)
	}
	return out, rows.Err()
}
func (s *Store) FinishDelivery(d domain.Delivery, retryAt time.Time, sent, invalid bool) error {
	return s.Write(func(t *Tx) error {
		if invalid {
			_, e := t.tx.Exec("DELETE FROM push_device WHERE id=? AND token=?", d.DeviceID, d.Token)
			slog.Debug("push device deleted", "device_id", d.DeviceID, "token_fp", domain.Fingerprint(d.Token))
			return e
		}
		// A rotated token must not inherit success for the previous token.
		_, e := t.tx.Exec("UPDATE push_delivery SET sent=?,next_at=? WHERE occurrence_id=? AND device_id=? AND EXISTS(SELECT 1 FROM push_device WHERE id=? AND token=?)", sent, retryAt.Unix(), d.OccurrenceID, d.DeviceID, d.DeviceID, d.Token)
		return e
	})
}
func (s *Store) DeliveryCurrent(d domain.Delivery) (bool, error) {
	var n int
	e := s.db.QueryRow("SELECT count(*) FROM reminder_occurrence r JOIN push_device p ON p.id=? WHERE r.id=? AND r.state='active' AND p.token=?", d.DeviceID, d.OccurrenceID, d.Token).Scan(&n)
	return n == 1, e
}
