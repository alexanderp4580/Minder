package domain

import (
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
}

func ProjectReminder(item Item, profile *Profile) ReminderProjection {
	at, nag := EffectiveReminder(item, profile)
	return ReminderProjection{at, nag, at != "" && (item.Status == "open" || item.Status == "in_progress")}
}

// ReminderStats counts what the reminder pipeline holds. The worker logs it so a missing notification can be traced.
type ReminderStats struct {
	Items, Devices, SchedulesActive, OccurrencesActive int
	DeliveriesPending, DeliveriesDue                   int
	OldestDueSeconds                                   int64
	NextScheduleAt                                     string
}
