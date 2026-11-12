package store

import (
	"errors"
	"github.com/alexanderp4580/Minder/server/domain"
	"path/filepath"
	"testing"
	"time"
)

func reminderStore(t *testing.T) (*Store, time.Time, domain.Item) {
	t.Helper()
	s, e := Open(filepath.Join(t.TempDir(), "db"))
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { s.Close() })
	now := time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)
	i := domain.Item{ID: "item", Title: "Bins", Status: "open", Version: 1, Timezone: "Europe/Amsterdam", RemindAt: now.Format(time.RFC3339), NagMinutes: 15}
	return s, now, i
}
func putReminder(t *testing.T, s *Store, i domain.Item) {
	t.Helper()
	if e := s.Write(func(tx *Tx) error { return tx.PutItem(i) }); e != nil {
		t.Fatal(e)
	}
}
func TestReminderCatchUpNagAndStaleActions(t *testing.T) {
	s, now, i := reminderStore(t)
	putReminder(t, s, i)
	if e := s.RegisterDevice("phone", "token"); e != nil {
		t.Fatal(e)
	}
	if e := s.DueReminders(now.Add(-time.Second)); e != nil {
		t.Fatal(e)
	}
	h, _ := s.History(0, 20)
	if len(h) != 0 {
		t.Fatal("early reminder")
	}
	if e := s.DueReminders(now.Add(time.Hour)); e != nil {
		t.Fatal(e)
	}
	h, _ = s.History(0, 20)
	if len(h) != 1 || h[0].Reminder == nil || !h[0].Reminder.Late {
		t.Fatalf("catchup %+v", h)
	}
	first := h[0].Reminder.ID
	s.DueReminders(now.Add(time.Hour))
	h, _ = s.History(0, 20)
	if len(h) != 1 {
		t.Fatal("duplicate firing")
	}
	d, e := s.Deliveries(now.Add(time.Hour))
	if e != nil || len(d) != 1 {
		t.Fatal("delivery missing", e)
	}
	s.DueReminders(now.Add(75 * time.Minute))
	h, _ = s.History(0, 20)
	if len(h) != 2 || h[0].Reminder.State != "superseded" {
		t.Fatalf("nag %+v", h)
	}
	if _, e = s.ReminderAction(first, "done", now.Add(75*time.Minute)); !errors.Is(e, domain.ErrConflict) {
		t.Fatal("old action accepted", e)
	}
	id := h[1].Reminder.ID
	r, e := s.ReminderAction(id, "snooze15", now.Add(75*time.Minute))
	if e != nil {
		t.Fatal(e)
	}
	again, e := s.ReminderAction(id, "snooze15", now.Add(76*time.Minute))
	if e != nil || again.ID != r.ID {
		t.Fatal("not idempotent", e)
	}
	s.DueReminders(now.Add(89 * time.Minute))
	h, _ = s.ActiveReminders()
	if len(h) != 0 {
		t.Fatal("snooze fired early")
	}
	s.DueReminders(now.Add(90 * time.Minute))
	h, _ = s.ActiveReminders()
	if len(h) != 1 {
		t.Fatal("snooze lost")
	}
	if _, e = s.ReminderAction(h[0].Reminder.ID, "done", now.Add(90*time.Minute)); e != nil {
		t.Fatal(e)
	}
	items, _ := s.Items()
	if items[0].Status != "done" {
		t.Fatal("Done did not complete")
	}
	d, _ = s.Deliveries(now.Add(2 * time.Hour))
	if len(d) != 0 {
		t.Fatal("completed delivery remains")
	}
}
func TestReminderScheduleChangesAndUndoShape(t *testing.T) {
	s, now, i := reminderStore(t)
	putReminder(t, s, i)
	s.DueReminders(now)
	old, _ := s.ActiveReminders()
	i.RemindAt = now.Add(time.Hour).Format(time.RFC3339)
	i.Version++
	putReminder(t, s, i)
	if _, e := s.ReminderAction(old[0].Reminder.ID, "ack", now); !errors.Is(e, domain.ErrConflict) {
		t.Fatal("rescheduled action accepted")
	}
	s.DueReminders(now)
	active, _ := s.ActiveReminders()
	if len(active) != 0 {
		t.Fatal("reschedule fired early")
	}
	i.RemindAt = ""
	i.NagMinutes = 0
	i.Version++
	putReminder(t, s, i)
	s.DueReminders(now.Add(2 * time.Hour))
	active, _ = s.ActiveReminders()
	if len(active) != 0 {
		t.Fatal("cleared reminder fired")
	}
	i.RemindAt = now.Format(time.RFC3339)
	i.NagMinutes = 15
	i.Version++
	putReminder(t, s, i)
	s.DueReminders(now.Add(2 * time.Hour))
	active, _ = s.ActiveReminders()
	if len(active) != 1 {
		t.Fatal("restored reminder not caught up")
	}
	s.ReminderAction(active[0].Reminder.ID, "ack", now.Add(2*time.Hour))
	i.Title = "New title"
	i.Version++
	putReminder(t, s, i)
	s.DueReminders(now.Add(3 * time.Hour))
	active, _ = s.ActiveReminders()
	if len(active) != 0 {
		t.Fatal("title edit rearmed acknowledged reminder")
	}
}
func TestReminderDeliveryRetriesAndDeviceRotation(t *testing.T) {
	s, now, i := reminderStore(t)
	putReminder(t, s, i)
	s.RegisterDevice("phone", "old")
	s.DueReminders(now)
	d, _ := s.Deliveries(now)
	if len(d) != 1 {
		t.Fatal("missing delivery")
	}
	s.FinishDelivery(d[0], now, false, false)
	d, _ = s.Deliveries(now.Add(59 * time.Second))
	if len(d) != 0 {
		t.Fatal("retry too early")
	}
	s.RegisterDevice("phone", "new")
	d, _ = s.Deliveries(now.Add(time.Minute))
	if len(d) != 1 || d[0].Token != "new" {
		t.Fatal("token did not rotate")
	}
	s.FinishDelivery(d[0], now.Add(time.Minute), true, false)
	d, _ = s.Deliveries(now.Add(2 * time.Minute))
	if len(d) != 0 {
		t.Fatal("sent again")
	}
	s.RegisterDevice("tablet", "tablet-token")
	d, _ = s.Deliveries(now)
	if len(d) != 1 || d[0].DeviceID != "tablet" {
		t.Fatal("new device missed active reminder")
	}
}
func TestReminderRestartPersistsAndDueOnlyDoesNotSchedule(t *testing.T) {
	s, now, i := reminderStore(t)
	path := filepath.Join(t.TempDir(), "persist")
	p, e := Open(path)
	if e != nil {
		t.Fatal(e)
	}
	putReminder(t, p, i)
	p.Close()
	p, e = Open(path)
	if e != nil {
		t.Fatal(e)
	}
	defer p.Close()
	p.DueReminders(now.Add(time.Hour))
	h, _ := p.ActiveReminders()
	if len(h) != 1 {
		t.Fatal("restart lost schedule")
	}
	i.RemindAt = ""
	i.NagMinutes = 0
	i.DueAt = now.Format(time.RFC3339)
	putReminder(t, s, i)
	s.DueReminders(now.Add(time.Hour))
	h, _ = s.ActiveReminders()
	if len(h) != 0 {
		t.Fatal("due-only item notified")
	}
}

func TestDoneOnARepeatingItemMovesItForwardAndUndoRestoresIt(t *testing.T) {
	s, now, i := reminderStore(t)
	i.DueAt = now.Format(time.RFC3339)
	i.Recurrence = &domain.Recurrence{Every: "weekly", Interval: 1}
	putReminder(t, s, i)
	s.DueReminders(now)
	active, _ := s.ActiveReminders()
	result, e := s.ReminderAction(active[0].Reminder.ID, "done", now)
	if e != nil {
		t.Fatal(e)
	}
	items, _ := s.Items()
	if items[0].Status != "open" || items[0].DueAt != now.AddDate(0, 0, 7).Format(time.RFC3339) || items[0].CompletedAt == "" {
		t.Fatalf("a repeating item must roll forward: %+v", items[0])
	}
	if result.Change == nil || result.Change.UndoID == "" {
		t.Fatal("Done must offer Undo")
	}
}
