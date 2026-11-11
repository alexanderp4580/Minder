package store

import (
	"database/sql"
	"encoding/json"
	"github.com/alexanderp4580/Minder/server/domain"
	"strings"
	"time"
)

type Delivery struct {
	OccurrenceID, DeviceID, Token, Title, ScheduledAt string
	Late                                              bool
}

func (t *Tx) closeReminders(itemID, state string) error {
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
type reminderSpec struct {
	at     string
	nag    int
	active bool
}

func specOf(i domain.Item, p *domain.Profile) reminderSpec {
	at, nag := domain.EffectiveReminder(i, p)
	return reminderSpec{at, nag, at != "" && (i.Status == "open" || i.Status == "in_progress")}
}

// applyReminder replaces the schedule when the effective reminder changed. It cancels open occurrences first.
func (t *Tx) applyReminder(itemID string, old, now reminderSpec) error {
	if old == now {
		return nil
	}
	if e := t.closeReminders(itemID, "cancelled"); e != nil {
		return e
	}
	if _, e := t.tx.Exec("DELETE FROM reminder_schedule WHERE item_id=?", itemID); e != nil {
		return e
	}
	if !now.active {
		return nil
	}
	at, e := time.Parse(time.RFC3339, now.at)
	if e != nil {
		return e
	}
	_, e = t.tx.Exec("INSERT INTO reminder_schedule VALUES(?,?,?,?,?)", itemID, ID(), at.Unix(), now.nag, "active")
	return e
}
func (t *Tx) syncReminder(old *domain.Item, i domain.Item) error {
	var before reminderSpec
	if old != nil {
		p, e := t.profileOf(old.ProfileID)
		if e != nil {
			return e
		}
		before = specOf(*old, p)
	}
	p, e := t.profileOf(i.ProfileID)
	if e != nil {
		return e
	}
	return t.applyReminder(i.ID, before, specOf(i, p))
}
func (s *Store) RegisterDevice(id, token string) error {
	if len(id) < 1 || len(id) > 128 || len(token) < 1 || len(token) > 4096 || strings.TrimSpace(token) != token {
		return &domain.Invalid{Field: "device", Message: "Send a valid device ID and push token."}
	}
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
func (s *Store) DueReminders(now time.Time) error {
	return s.Write(func(t *Tx) error {
		rows, e := t.tx.Query("SELECT item_id,generation,next_at,nag_minutes FROM reminder_schedule WHERE state='active' AND next_at<=? ORDER BY next_at LIMIT 100", now.Unix())
		if e != nil {
			return e
		}
		type schedule struct {
			id, gen string
			at      int64
			nag     int
		}
		var due []schedule
		for rows.Next() {
			var r schedule
			if e = rows.Scan(&r.id, &r.gen, &r.at, &r.nag); e != nil {
				rows.Close()
				return e
			}
			due = append(due, r)
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return e
		}
		for _, r := range due {
			i, e := t.Item(r.id)
			if e != nil {
				return e
			}
			if e = t.closeReminders(i.ID, "superseded"); e != nil {
				return e
			}
			id := ID()
			rem := &domain.Reminder{ID: id, ItemID: i.ID, Title: i.Title, ScheduledAt: time.Unix(r.at, 0).UTC().Format(time.RFC3339), State: "active", Late: now.Unix()-r.at > 60}
			text := "Reminder: " + i.Title + "."
			if rem.Late {
				text = "Late reminder: " + i.Title + "."
			}
			entry, e := t.PutEntry(domain.Entry{Kind: "reminder", Text: text, Reminder: rem})
			if e != nil {
				return e
			}
			if _, e = t.tx.Exec("INSERT INTO reminder_occurrence(id,item_id,generation,state,entry_id) VALUES(?,?,?,'active',?)", id, i.ID, r.gen, entry.ID); e != nil {
				return e
			}
			if _, e = t.tx.Exec("INSERT INTO push_delivery(occurrence_id,device_id,next_at) SELECT ?,id,? FROM push_device", id, now.Unix()); e != nil {
				return e
			}
			state, next := "waiting", int64(0)
			if r.nag > 0 {
				state = "active"
				next = now.Add(time.Duration(r.nag) * time.Minute).Unix()
			}
			if _, e = t.tx.Exec("UPDATE reminder_schedule SET next_at=?,state=? WHERE item_id=? AND generation=?", next, state, i.ID, r.gen); e != nil {
				return e
			}
		}
		return nil
	})
}
func (s *Store) ActiveReminders() ([]domain.Entry, error) {
	rows, e := s.db.Query("SELECT e.body FROM reminder_occurrence r JOIN chat_entry e ON e.id=r.entry_id WHERE r.state='active' ORDER BY e.seq")
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []domain.Entry{}
	for rows.Next() {
		var raw string
		if e = rows.Scan(&raw); e != nil {
			return nil, e
		}
		var entry domain.Entry
		if e = json.Unmarshal([]byte(raw), &entry); e != nil {
			return nil, e
		}
		out = append(out, entry)
	}
	return out, rows.Err()
}
func (s *Store) ReminderAction(id, action string, now time.Time) (out domain.Entry, err error) {
	switch action {
	case "ack", "done", "snooze15", "snooze60", "tonight", "tomorrow":
	default:
		return out, &domain.Invalid{Field: "action", Message: "Choose Done, Open or a snooze time."}
	}
	err = s.Write(func(t *Tx) error {
		var itemID, gen, state, entryID, resultID string
		e := t.tx.QueryRow("SELECT item_id,generation,state,entry_id,result_id FROM reminder_occurrence WHERE id=?", id).Scan(&itemID, &gen, &state, &entryID, &resultID)
		if e == sql.ErrNoRows {
			return domain.ErrNotFound
		}
		if e != nil {
			return e
		}
		if state == action && resultID != "" {
			out, e = t.Entry(resultID)
			return e
		}
		if state != "active" {
			return domain.ErrConflict
		}
		var current string
		if e = t.tx.QueryRow("SELECT generation FROM reminder_schedule WHERE item_id=?", itemID).Scan(&current); e != nil || current != gen {
			return domain.ErrConflict
		}
		i, e := t.Item(itemID)
		if e != nil {
			return e
		}
		undoID := ""
		text := "Reminder acknowledged: " + i.Title + "."
		if action == "done" {
			before := i
			i.Status = "done"
			i.CompletedAt = now.UTC().Format(time.RFC3339)
			i.Version++
			i.UpdatedAt = domain.Now()
			if e = t.PutItem(i); e != nil {
				return e
			}
			undoID = ID()
			if e = t.PutAudit(domain.Audit{ID: undoID, Actor: "owner", Before: &before, After: i}); e != nil {
				return e
			}
			text = "Done: " + i.Title + "."
		} else {
			if e = t.closeReminders(itemID, action); e != nil {
				return e
			}
			state, next := "acknowledged", int64(0)
			if action != "ack" {
				state = "active"
				at := now.Add(15 * time.Minute)
				if action == "snooze60" {
					at = now.Add(time.Hour)
				}
				if action == "tonight" || action == "tomorrow" {
					zone, e := time.LoadLocation(i.Timezone)
					if e != nil {
						return e
					}
					local := now.In(zone)
					hour := 20
					day := local.Day()
					if action == "tomorrow" {
						hour = 9
						day++
					}
					at = time.Date(local.Year(), local.Month(), day, hour, 0, 0, 0, zone)
					if !at.After(now) {
						at = at.AddDate(0, 0, 1)
					}
				}
				next = at.Unix()
				text = "Snoozed: " + i.Title + "."
			}
			if _, e = t.tx.Exec("UPDATE reminder_schedule SET state=?,next_at=? WHERE item_id=?", state, next, itemID); e != nil {
				return e
			}
		}
		// Completion cancelled the active occurrence; replace its visible state with the chosen action.
		entry, e := t.Entry(entryID)
		if e != nil {
			return e
		}
		entry.Reminder.State = action
		if _, e = t.PutEntry(entry); e != nil {
			return e
		}
		var change *domain.Change
		if undoID != "" {
			change = &domain.Change{Action: "complete_item", ItemTitle: i.Title, State: "confirmed", UndoID: undoID}
		}
		out, e = t.PutEntry(domain.Entry{Kind: "result", Text: text, Change: change})
		if e != nil {
			return e
		}
		_, e = t.tx.Exec("UPDATE reminder_occurrence SET state=?,result_id=? WHERE id=?", action, out.ID, id)
		return e
	})
	return
}
func (s *Store) Deliveries(now time.Time) ([]Delivery, error) {
	rows, e := s.db.Query("SELECT p.occurrence_id,p.device_id,d.token,e.body FROM push_delivery p JOIN push_device d ON d.id=p.device_id JOIN reminder_occurrence r ON r.id=p.occurrence_id JOIN chat_entry e ON e.id=r.entry_id WHERE p.sent=0 AND p.next_at<=? AND r.state='active' ORDER BY p.next_at LIMIT 100", now.Unix())
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []Delivery{}
	for rows.Next() {
		var d Delivery
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
func (s *Store) FinishDelivery(d Delivery, now time.Time, sent, invalid bool) error {
	return s.Write(func(t *Tx) error {
		if invalid {
			_, e := t.tx.Exec("DELETE FROM push_device WHERE id=? AND token=?", d.DeviceID, d.Token)
			return e
		}
		// A rotated token must not inherit success for the previous token.
		_, e := t.tx.Exec("UPDATE push_delivery SET sent=?,next_at=? WHERE occurrence_id=? AND device_id=? AND EXISTS(SELECT 1 FROM push_device WHERE id=? AND token=?)", sent, now.Add(time.Minute).Unix(), d.OccurrenceID, d.DeviceID, d.DeviceID, d.Token)
		return e
	})
}
func (s *Store) DeliveryCurrent(d Delivery) (bool, error) {
	var n int
	e := s.db.QueryRow("SELECT count(*) FROM reminder_occurrence r JOIN push_device p ON p.id=? WHERE r.id=? AND r.state='active' AND p.token=?", d.DeviceID, d.OccurrenceID, d.Token).Scan(&n)
	return n == 1, e
}
