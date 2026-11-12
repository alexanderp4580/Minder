package store

import (
	"database/sql"
	"log/slog"
	"time"

	"github.com/alexanderp4580/Minder/server/domain"
)

func (t *Tx) DueSchedules(now time.Time) ([]domain.ReminderSchedule, error) {
	rows, err := t.tx.Query("SELECT item_id,generation,next_at,nag_minutes,state FROM reminder_schedule WHERE state='active' AND next_at<=? ORDER BY next_at LIMIT 100", now.Unix())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.ReminderSchedule{}
	for rows.Next() {
		var s domain.ReminderSchedule
		var at int64
		if err = rows.Scan(&s.ItemID, &s.Generation, &at, &s.NagMinutes, &s.State); err != nil {
			return nil, err
		}
		s.At = time.Unix(at, 0).UTC()
		out = append(out, s)
	}
	return out, rows.Err()
}
func (t *Tx) Schedule(id string) (domain.ReminderSchedule, error) {
	var s domain.ReminderSchedule
	var at int64
	err := t.tx.QueryRow("SELECT item_id,generation,next_at,nag_minutes,state FROM reminder_schedule WHERE item_id=?", id).Scan(&s.ItemID, &s.Generation, &at, &s.NagMinutes, &s.State)
	s.At = time.Unix(at, 0).UTC()
	return s, rowError(err)
}
func (t *Tx) UpdateSchedule(s domain.ReminderSchedule) error {
	_, err := t.tx.Exec("UPDATE reminder_schedule SET state=?,next_at=? WHERE item_id=? AND generation=?", s.State, s.At.Unix(), s.ItemID, s.Generation)
	return err
}
func (t *Tx) Occurrence(id string) (domain.Occurrence, error) {
	var o domain.Occurrence
	err := t.tx.QueryRow("SELECT id,item_id,generation,state,entry_id,result_id FROM reminder_occurrence WHERE id=?", id).Scan(&o.ID, &o.ItemID, &o.Generation, &o.State, &o.EntryID, &o.ResultID)
	return o, rowError(err)
}
func (t *Tx) PutOccurrence(o domain.Occurrence) error {
	_, err := t.tx.Exec("INSERT INTO reminder_occurrence(id,item_id,generation,state,entry_id,result_id) VALUES(?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET state=excluded.state,result_id=excluded.result_id", o.ID, o.ItemID, o.Generation, o.State, o.EntryID, o.ResultID)
	return err
}
func (t *Tx) QueueDeliveries(id string, at time.Time) error {
	res, err := t.tx.Exec("INSERT INTO push_delivery(occurrence_id,device_id,next_at) SELECT ?,id,? FROM push_device", id, at.Unix())
	if err == nil {
		devices, _ := res.RowsAffected()
		slog.Info("deliveries queued", "occurrence_id", id, "devices", devices, "due_at", at.UTC().Format(time.RFC3339))
		if devices == 0 {
			slog.Warn("no push devices registered; reminder stays in chat only", "occurrence_id", id)
		}
	}
	return err
}

// ReminderStats counts the reminder pipeline for the worker heartbeat.
func (s *Store) ReminderStats(now time.Time) (st domain.ReminderStats, err error) {
	count := func(dest *int, query string, args ...any) {
		if err == nil {
			err = s.db.QueryRow(query, args...).Scan(dest)
		}
	}
	count(&st.Items, "SELECT COUNT(*) FROM item")
	count(&st.Devices, "SELECT COUNT(*) FROM push_device")
	count(&st.SchedulesActive, "SELECT COUNT(*) FROM reminder_schedule WHERE state='active'")
	count(&st.OccurrencesActive, "SELECT COUNT(*) FROM reminder_occurrence WHERE state='active'")
	count(&st.DeliveriesPending, "SELECT COUNT(*) FROM push_delivery p JOIN reminder_occurrence r ON r.id=p.occurrence_id WHERE p.sent=0 AND r.state='active'")
	count(&st.DeliveriesDue, "SELECT COUNT(*) FROM push_delivery p JOIN reminder_occurrence r ON r.id=p.occurrence_id WHERE p.sent=0 AND r.state='active' AND p.next_at<=?", now.Unix())
	if err != nil {
		return
	}
	var oldest, next sql.NullInt64
	err = s.db.QueryRow("SELECT MIN(p.next_at) FROM push_delivery p JOIN reminder_occurrence r ON r.id=p.occurrence_id WHERE p.sent=0 AND r.state='active' AND p.next_at<=?", now.Unix()).Scan(&oldest)
	if err == nil {
		err = s.db.QueryRow("SELECT MIN(next_at) FROM reminder_schedule WHERE state='active'").Scan(&next)
	}
	if oldest.Valid {
		st.OldestDueSeconds = now.Unix() - oldest.Int64
	}
	if next.Valid {
		st.NextScheduleAt = time.Unix(next.Int64, 0).UTC().Format(time.RFC3339)
	}
	return
}
