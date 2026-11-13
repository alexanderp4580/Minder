package reminder_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/port"
	"github.com/alexanderp4580/Minder/server/reminder"
	"github.com/alexanderp4580/Minder/server/store"
)

func TestScheduleAdvancesThroughOffsets(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	start := time.Now().UTC().Add(30 * time.Hour).Truncate(time.Second)
	item := domain.Item{ID: "appointment", Title: "Fitting", Status: "open", Version: 1, Timezone: "Europe/Amsterdam", Kind: "event",
		StartAt: start.Format(time.RFC3339), ReminderOffsets: []int{1440, 60}}
	if err = db.Transact(func(tx port.Transaction) error { return tx.PutItem(item) }); err != nil {
		t.Fatal(err)
	}
	svc := reminder.New(db)
	at := func() time.Time {
		var s domain.ReminderSchedule
		if err := db.Transact(func(tx port.Transaction) (e error) { s, e = tx.Schedule("appointment"); return }); err != nil {
			t.Fatal(err)
		}
		if s.State != "active" {
			t.Fatal("schedule not active", s)
		}
		return s.At
	}
	if first := at(); !first.Equal(start.Add(-24 * time.Hour)) {
		t.Fatal("first offset", first)
	}
	if err = svc.ScheduleDue(start.Add(-24*time.Hour + time.Second)); err != nil {
		t.Fatal(err)
	}
	if second := at(); !second.Equal(start.Add(-time.Hour)) {
		t.Fatal("did not advance to the second offset", second)
	}
	if err = svc.ScheduleDue(start.Add(-time.Hour + time.Second)); err != nil {
		t.Fatal(err)
	}
	var s domain.ReminderSchedule
	_ = db.Transact(func(tx port.Transaction) (e error) { s, e = tx.Schedule("appointment"); return })
	if s.State != "waiting" {
		t.Fatal("schedule after the last offset", s)
	}
	active, _ := svc.Active()
	if len(active) != 1 {
		t.Fatal("expected the second occurrence to supersede the first", len(active))
	}
}
