package store

import (
	"encoding/json"
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
		return nil
	}
	if e := t.CloseReminders(itemID, "cancelled"); e != nil {
		return e
	}
	if _, e := t.tx.Exec("DELETE FROM reminder_schedule WHERE item_id=?", itemID); e != nil {
		return e
	}
	if !now.Active {
		return nil
	}
	at, e := time.Parse(time.RFC3339, now.At)
	if e != nil {
		return e
	}
	_, e = t.tx.Exec("INSERT INTO reminder_schedule VALUES(?,?,?,?,?)", itemID, domain.NewID(), at.Unix(), now.NagMinutes, "active")
	return e
}
func (t *Tx) syncReminder(old *domain.Item, i domain.Item) error {
	var before domain.ReminderProjection
	if old != nil {
		p, e := t.profileOf(old.ProfileID)
		if e != nil {
			return e
		}
		before = domain.ProjectReminder(*old, p)
	}
	p, e := t.profileOf(i.ProfileID)
	if e != nil {
		return e
	}
	return t.applyReminder(i.ID, before, domain.ProjectReminder(i, p))
}
func (s *Store) RegisterDevice(id, token string) error {
	return s.Write(func(t *Tx) error {
		// Reinstallation may assign an existing token to a new local device ID.
		if _, e := t.tx.Exec("DELETE FROM push_device WHERE token=? AND id<>?", token, id); e != nil {
			return e
		}
		if _, e := t.tx.Exec("INSERT INTO push_device VALUES(?,?) ON CONFLICT(id) DO UPDATE SET token=excluded.token", id, token); e != nil {
			return e
		}
		_, e := t.tx.Exec("INSERT OR IGNORE INTO push_delivery(occurrence_id,device_id,next_at) SELECT id,?,0 FROM reminder_occurrence WHERE state='active'", id)
		return e
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
