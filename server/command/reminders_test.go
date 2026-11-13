package command

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/alexanderp4580/Minder/server/domain"
)

func appointment(offsets ...int) domain.Proposal {
	start := time.Now().UTC().Add(72 * time.Hour).Truncate(time.Minute)
	return domain.Proposal{Action: "create_item", Title: "Fitting", Kind: "event", StartAt: start.Format(time.RFC3339), ReminderOffsets: offsets}
}

func field(entry domain.Entry, name string) (domain.Field, bool) {
	for _, f := range entry.Change.Fields {
		if f.Name == name {
			return f, true
		}
	}
	return domain.Field{}, false
}

func TestPreviewListsRemindersAndChoices(t *testing.T) {
	c, _ := setup(t)
	entry, err := c.Preview(appointment(60, 1440))
	if err != nil {
		t.Fatal(err)
	}
	if f, ok := field(entry, "reminders"); !ok || f.New != "1 day, 1 hour" {
		t.Fatal("reminders field", f, ok)
	}
	if !reflect.DeepEqual(entry.Change.ReminderOffsets, []int{1440, 60}) || !reflect.DeepEqual(entry.Change.ReminderChoices, domain.ReminderOffsetChoices) {
		t.Fatal(entry.Change.ReminderOffsets, entry.Change.ReminderChoices)
	}
	noAnchor, err := c.Preview(domain.Proposal{Action: "create_item", Title: "Buy milk"})
	if err != nil || len(noAnchor.Change.ReminderChoices) != 0 {
		t.Fatal("an item without a start or deadline offers no reminder choices", noAnchor.Change.ReminderChoices, err)
	}
}

func TestSetRemindersUpdatesThePendingProposalAndPreview(t *testing.T) {
	c, s := setup(t)
	entry, _ := c.Preview(appointment(60))
	updated, err := c.SetReminders(entry.Change.ID, []int{10, 10080, 10})
	if err != nil {
		t.Fatal(err)
	}
	if f, ok := field(updated, "reminders"); !ok || f.New != "1 week, 10 minutes" || updated.ID != entry.ID {
		t.Fatal(f, ok, updated.ID)
	}
	if _, err = c.Confirm(entry.Change.ID); err != nil {
		t.Fatal(err)
	}
	items, _ := s.Items()
	if len(items) != 1 || !reflect.DeepEqual(items[0].ReminderOffsets, []int{10080, 10}) {
		t.Fatal("confirmed item reminders", items)
	}
	cleared, _ := c.Preview(appointment(60))
	after, err := c.SetReminders(cleared.Change.ID, nil)
	if err != nil || len(after.Change.ReminderOffsets) != 0 {
		t.Fatal("clearing all reminders", after.Change, err)
	}
}

func TestSetRemindersOnConfirmedChangeConflicts(t *testing.T) {
	c, _ := setup(t)
	entry, _ := c.Preview(appointment(60))
	if _, err := c.Confirm(entry.Change.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := c.SetReminders(entry.Change.ID, []int{10}); !errors.Is(err, domain.ErrConflict) {
		t.Fatal("confirmed change edited", err)
	}
	discarded, _ := c.Preview(appointment())
	_, _ = c.Discard(discarded.Change.ID)
	if _, err := c.SetReminders(discarded.Change.ID, []int{10}); !errors.Is(err, domain.ErrConflict) {
		t.Fatal("discarded change edited", err)
	}
}

func TestSetRemindersRejectsUnknownOffsets(t *testing.T) {
	c, _ := setup(t)
	entry, _ := c.Preview(appointment())
	var invalid *domain.Invalid
	if _, err := c.SetReminders(entry.Change.ID, []int{7}); !errors.As(err, &invalid) {
		t.Fatal("unknown offset accepted", err)
	}
}

func TestPreviewTimesAreShownInTheirOwnZones(t *testing.T) {
	c, _ := setup(t)
	start := time.Now().UTC().Add(96 * time.Hour).Truncate(time.Hour)
	entry, err := c.Preview(domain.Proposal{Action: "create_item", Title: "Flight", Kind: "event",
		StartAt: start.Format(time.RFC3339), StartTimezone: "Europe/Amsterdam",
		EndAt: start.Add(8 * time.Hour).Format(time.RFC3339), EndTimezone: "America/New_York"})
	if err != nil {
		t.Fatal(err)
	}
	startField, _ := field(entry, "start_at")
	endField, _ := field(entry, "end_at")
	wantStart := start.In(time.FixedZone("", 0)).Format("2006-01-02T15:04")
	if len(startField.New) < 20 || startField.New[:16] == wantStart && startField.New[len(startField.New)-1] == 'Z' {
		t.Fatal("start must carry the Amsterdam offset, not Z", startField.New)
	}
	startTime, _ := time.Parse(time.RFC3339, startField.New)
	endTime, _ := time.Parse(time.RFC3339, endField.New)
	_, startOffset := startTime.Zone()
	_, endOffset := endTime.Zone()
	amsterdam, _ := time.LoadLocation("Europe/Amsterdam")
	newYork, _ := time.LoadLocation("America/New_York")
	_, wantStartOffset := start.In(amsterdam).Zone()
	_, wantEndOffset := start.Add(8 * time.Hour).In(newYork).Zone()
	if startOffset != wantStartOffset || endOffset != wantEndOffset || !startTime.Equal(start) {
		t.Fatal("zones", startField.New, endField.New)
	}
}
