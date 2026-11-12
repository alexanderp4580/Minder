package domain

import (
	"strings"
	"time"
	"unicode/utf8"
)

// Recurrence repeats an item. Fixed repeats keep the schedule; AfterCompletion counts from the day you finish.
type Recurrence struct {
	Every           string `json:"every"`
	Interval        int    `json:"interval"`
	AfterCompletion bool   `json:"after_completion,omitempty"`
}

// Profile holds reminder defaults for a kind of item. The reminder time is the due time minus RemindBeforeMinutes.
type Profile struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	RemindBeforeMinutes int    `json:"remind_before_minutes"`
	NagMinutes          int    `json:"nag_minutes"`
	Builtin             bool   `json:"builtin,omitempty"`
	Version             int    `json:"version"`
}

func ValidateProfile(p *Profile) error {
	p.Name = strings.TrimSpace(p.Name)
	if n := utf8.RuneCountInString(p.Name); n < 1 || n > 60 {
		return &Invalid{"name", "Use a profile name with 1 to 60 characters."}
	}
	if p.RemindBeforeMinutes < 0 || p.RemindBeforeMinutes > 20160 {
		return &Invalid{"remind_before_minutes", "Remind between 0 minutes and 2 weeks before the due time."}
	}
	if p.NagMinutes < 0 || p.NagMinutes > 1440 {
		return &Invalid{"nag_minutes", "Use a nag interval from 0 to 1,440 minutes."}
	}
	return nil
}

func validateRecurrence(i *Item) error {
	r := i.Recurrence
	if r == nil {
		return nil
	}
	switch r.Every {
	case "daily", "weekly", "monthly", "yearly":
	default:
		return &Invalid{"recurrence", "Repeat daily, weekly, monthly or yearly."}
	}
	if r.Interval < 1 || r.Interval > 365 {
		return &Invalid{"recurrence", "Repeat every 1 to 365 periods."}
	}
	if i.DueAt == "" && i.StartAt == "" {
		return &Invalid{"recurrence", "Set a start or a deadline for a repeating item."}
	}
	return nil
}

// Anchor is the time that reminders and repeats count from: the deadline, else the start.
func Anchor(i Item) string {
	if i.DueAt != "" {
		return i.DueAt
	}
	return i.StartAt
}

// PlacedAt is the time an item shows on a calendar: the start, else the deadline.
func PlacedAt(i Item) string {
	if i.StartAt != "" {
		return i.StartAt
	}
	return i.DueAt
}

// EffectiveReminder returns the reminder an item uses. A reminder set on the item overrides its profile.
func EffectiveReminder(i Item, p *Profile) (string, int) {
	if i.RemindAt != "" {
		return i.RemindAt, i.NagMinutes
	}
	anchor := Anchor(i)
	if p == nil || i.ProfileID == "" || anchor == "" {
		return "", 0
	}
	due, e := time.Parse(time.RFC3339, anchor)
	if e != nil {
		return "", 0
	}
	return due.Add(-time.Duration(p.RemindBeforeMinutes) * time.Minute).UTC().Format(time.RFC3339), p.NagMinutes
}

func addMonths(t time.Time, n int) time.Time {
	y, m, d := t.Date()
	first := time.Date(y, m, 1, t.Hour(), t.Minute(), t.Second(), 0, t.Location()).AddDate(0, n, 0)
	last := first.AddDate(0, 1, -1).Day()
	if d > last {
		d = last
	}
	return time.Date(first.Year(), first.Month(), d, t.Hour(), t.Minute(), t.Second(), 0, t.Location())
}

func step(t time.Time, r Recurrence, k int) time.Time {
	n := r.Interval * k
	switch r.Every {
	case "daily":
		return t.AddDate(0, 0, n)
	case "weekly":
		return t.AddDate(0, 0, 7*n)
	case "monthly":
		return addMonths(t, n)
	default:
		return addMonths(t, 12*n)
	}
}

// NextDue returns the next anchor time in UTC. Times keep their wall-clock hour in the item's timezone.
func NextDue(i Item, now time.Time) (string, error) {
	loc, e := time.LoadLocation(i.Timezone)
	if e != nil {
		return "", &Invalid{"timezone", "Choose a valid timezone."}
	}
	due, e := time.Parse(time.RFC3339, Anchor(i))
	if e != nil || i.Recurrence == nil {
		return "", &Invalid{"recurrence", "Set a start or a deadline for a repeating item."}
	}
	due = due.In(loc)
	r := *i.Recurrence
	if r.AfterCompletion {
		base := now.In(loc)
		base = time.Date(base.Year(), base.Month(), base.Day(), due.Hour(), due.Minute(), due.Second(), 0, loc)
		return step(base, r, 1).UTC().Format(time.RFC3339), nil
	}
	next := step(due, r, 1)
	for k := 2; !next.After(now); k++ {
		next = step(due, r, k)
	}
	return next.UTC().Format(time.RFC3339), nil
}

// Advance moves a repeating item to its next occurrence. The start, the deadline and the reminder move by the same distance.
func Advance(i *Item, now time.Time) error {
	next, e := NextDue(*i, now)
	if e != nil {
		return e
	}
	old, _ := time.Parse(time.RFC3339, Anchor(*i))
	nt, _ := time.Parse(time.RFC3339, next)
	delta := nt.Sub(old)
	for _, field := range []*string{&i.StartAt, &i.DueAt, &i.RemindAt} {
		if *field == "" {
			continue
		}
		t, e := time.Parse(time.RFC3339, *field)
		if e != nil {
			return e
		}
		*field = t.Add(delta).UTC().Format(time.RFC3339)
	}
	return nil
}
