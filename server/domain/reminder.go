package domain

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type ReminderSchedule struct {
	ItemID, Generation, State string
	At                        time.Time
	NagMinutes                int
}

type Occurrence struct {
	ID, ItemID, Generation, State, EntryID, ResultID string
}

type Delivery struct {
	OccurrenceID, DeviceID, Token, Title, ScheduledAt string
	Late                                              bool
}

func ValidateDevice(id, token string) error {
	if len(id) < 1 || len(id) > 128 || len(token) < 1 || len(token) > 4096 || strings.TrimSpace(token) != token {
		return &Invalid{Field: "device", Message: "Send a valid device ID and push token."}
	}
	return nil
}

func SnoozeAt(action, timezone string, now time.Time) (time.Time, error) {
	switch action {
	case "snooze15":
		return now.Add(15 * time.Minute), nil
	case "snooze60":
		return now.Add(time.Hour), nil
	case "tonight", "tomorrow":
		zone, err := time.LoadLocation(timezone)
		if err != nil {
			return time.Time{}, &Invalid{Field: "timezone", Message: "Choose a valid timezone."}
		}
		local := now.In(zone)
		day, hour := local.Day(), 20
		if action == "tomorrow" {
			day++
			hour = 9
		}
		at := time.Date(local.Year(), local.Month(), day, hour, 0, 0, 0, zone)
		if !at.After(now) {
			at = at.AddDate(0, 0, 1)
		}
		return at, nil
	default:
		return time.Time{}, &Invalid{Field: "action", Message: "Choose Done, Open or a snooze time."}
	}
}

// ReminderProjection is the schedule derived from an item and its profile.
// SQLite stores this projection atomically with the item or profile write.
type ReminderProjection struct {
	At         string
	NagMinutes int
	Active     bool
	Offsets    string
}

func ProjectReminder(item Item, profile *Profile) ReminderProjection {
	at, nag := EffectiveReminder(item, profile)
	offsets := ""
	if next, ok := NextReminder(item, time.Now()); ok {
		at, nag = next.UTC().Format(time.RFC3339), item.NagMinutes
		offsets = fmt.Sprint(item.ReminderOffsets)
	}
	return ReminderProjection{at, nag, at != "" && (item.Status == "open" || item.Status == "in_progress"), offsets}
}

// ReminderStats counts what the reminder pipeline holds. The worker logs it so a missing notification can be traced.
type ReminderStats struct {
	Items, Devices, SchedulesActive, OccurrencesActive int
	DeliveriesPending, DeliveriesDue                   int
	OldestDueSeconds                                   int64
	NextScheduleAt                                     string
}

// ReminderOffsetChoices are the minutes before an item's start or deadline an owner can pick.
var ReminderOffsetChoices = []int{10, 30, 60, 120, 1440, 2880, 4320, 10080}

// NormalizeOffsets sorts offsets longest first, removes duplicates and rejects values outside the choices.
func NormalizeOffsets(in []int) ([]int, error) {
	seen := map[int]bool{}
	for _, minutes := range in {
		allowed := false
		for _, choice := range ReminderOffsetChoices {
			allowed = allowed || choice == minutes
		}
		if !allowed {
			return nil, &Invalid{Field: "reminder_offsets_minutes", Message: "Choose reminder times of 10 or 30 minutes, 1 or 2 hours, 1, 2 or 3 days, or 1 week."}
		}
		seen[minutes] = true
	}
	var out []int
	for minutes := range seen {
		out = append(out, minutes)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(out)))
	return out, nil
}

// reminderAnchor is the time offsets count back from: the deadline, else the start.
// A date-only item anchors at 09:00 local time on its first date.
func reminderAnchor(i Item) (time.Time, bool) {
	if anchor := Anchor(i); anchor != "" {
		t, e := time.Parse(time.RFC3339, anchor)
		return t, e == nil
	}
	date := i.StartDate
	if date == "" {
		date = i.DueDate
	}
	zone, e := time.LoadLocation(i.Timezone)
	if date == "" || e != nil {
		return time.Time{}, false
	}
	day, e := time.ParseInLocation("2006-01-02", date, zone)
	if e != nil {
		return time.Time{}, false
	}
	return day.Add(9 * time.Hour), true
}

// NextReminder returns the earliest offset time strictly after the given time.
// Offsets already past are dropped, so saving a soon item never fires a late reminder.
func NextReminder(i Item, after time.Time) (time.Time, bool) {
	anchor, ok := reminderAnchor(i)
	if !ok {
		return time.Time{}, false
	}
	var next time.Time
	found := false
	for _, minutes := range i.ReminderOffsets {
		at := anchor.Add(-time.Duration(minutes) * time.Minute)
		if at.After(after) && (!found || at.Before(next)) {
			next, found = at, true
		}
	}
	return next, found
}
