package domain

import (
	"errors"
	"testing"
	"time"
)

func at(s string) time.Time {
	t, e := time.Parse(time.RFC3339, s)
	if e != nil {
		panic(e)
	}
	return t
}

func TestNextDue(t *testing.T) {
	cases := []struct {
		name string
		item Item
		now  string
		want string
	}{
		{"daily early completion advances one step", Item{DueAt: "2026-11-12T09:00:00Z", Timezone: "Europe/Amsterdam", Recurrence: &Recurrence{Every: "daily", Interval: 1}}, "2026-11-12T07:00:00Z", "2026-11-13T09:00:00Z"},
		{"weekly overdue skips to the first future slot", Item{DueAt: "2026-11-01T09:00:00Z", Timezone: "Europe/Amsterdam", Recurrence: &Recurrence{Every: "weekly", Interval: 1}}, "2026-11-12T12:00:00Z", "2026-11-15T09:00:00Z"},
		{"monthly clamps to month end", Item{DueAt: "2027-01-31T09:00:00Z", Timezone: "Europe/Amsterdam", Recurrence: &Recurrence{Every: "monthly", Interval: 1}}, "2027-01-31T10:00:00Z", "2027-02-28T09:00:00Z"},
		{"yearly keeps the date", Item{DueAt: "2026-11-12T09:00:00Z", Timezone: "Europe/Amsterdam", Recurrence: &Recurrence{Every: "yearly", Interval: 1}}, "2026-11-12T10:00:00Z", "2027-11-12T09:00:00Z"},
		{"daily keeps wall-clock time across the DST change", Item{DueAt: "2027-03-27T08:00:00Z", Timezone: "Europe/Amsterdam", Recurrence: &Recurrence{Every: "daily", Interval: 1}}, "2027-03-27T07:00:00Z", "2027-03-28T07:00:00Z"},
		{"after completion counts from now at the same time of day", Item{DueAt: "2026-11-01T09:00:00Z", Timezone: "Europe/Amsterdam", Recurrence: &Recurrence{Every: "weekly", Interval: 2, AfterCompletion: true}}, "2026-11-12T20:00:00Z", "2026-11-26T09:00:00Z"},
	}
	for _, c := range cases {
		got, e := NextDue(c.item, at(c.now))
		if e != nil || got != c.want {
			t.Errorf("%s: got %q, %v; want %q", c.name, got, e, c.want)
		}
	}
}

func TestValidateRecurrenceListAndProfile(t *testing.T) {
	base := func() Item { return Item{Title: "x", Status: "open"} }
	bad := []Item{}
	i := base()
	i.Recurrence = &Recurrence{Every: "daily", Interval: 1}
	bad = append(bad, i) // no due time
	i = base()
	i.DueAt = "2026-11-12T09:00:00Z"
	i.Recurrence = &Recurrence{Every: "hourly", Interval: 1}
	bad = append(bad, i)
	i = base()
	i.DueAt = "2026-11-12T09:00:00Z"
	i.Recurrence = &Recurrence{Every: "daily", Interval: 0}
	bad = append(bad, i)
	i = base()
	i.List = string(make([]byte, 61))
	bad = append(bad, i)
	for n, item := range bad {
		if Validate(&item) == nil {
			t.Errorf("case %d accepted", n)
		}
	}
	ok := base()
	ok.List = "  Groceries "
	ok.DueAt = "2026-11-12T09:00:00Z"
	ok.Recurrence = &Recurrence{Every: "weekly", Interval: 1}
	if e := Validate(&ok); e != nil || ok.List != "Groceries" {
		t.Fatalf("valid item rejected: %v %q", e, ok.List)
	}
}

func TestValidateProfile(t *testing.T) {
	for _, p := range []Profile{{Name: ""}, {Name: "x", NagMinutes: -1}, {Name: "x", NagMinutes: 1441}, {Name: "x", RemindBeforeMinutes: -1}, {Name: "x", RemindBeforeMinutes: 20161}} {
		if ValidateProfile(&p) == nil {
			t.Errorf("accepted %+v", p)
		}
	}
	p := Profile{Name: " Chore ", RemindBeforeMinutes: 60}
	if e := ValidateProfile(&p); e != nil || p.Name != "Chore" {
		t.Fatalf("%v %q", e, p.Name)
	}
}

func TestReminderFromProfile(t *testing.T) {
	p := &Profile{ID: "daily-task", RemindBeforeMinutes: 30, NagMinutes: 15}
	i := Item{DueAt: "2026-11-12T09:00:00Z", Status: "open", ProfileID: "daily-task"}
	at, nag := EffectiveReminder(i, p)
	if at != "2026-11-12T08:30:00Z" || nag != 15 {
		t.Fatalf("%q %d", at, nag)
	}
	i.RemindAt, i.NagMinutes = "2026-11-12T07:00:00Z", 5
	if at, nag = EffectiveReminder(i, p); at != "2026-11-12T07:00:00Z" || nag != 5 {
		t.Fatalf("item reminder must override: %q %d", at, nag)
	}
	i = Item{DueAt: "2026-11-12T09:00:00Z", Status: "open"}
	if at, _ = EffectiveReminder(i, nil); at != "" {
		t.Fatal("no profile and no reminder must give none")
	}
	if at, _ = EffectiveReminder(Item{ProfileID: "x", Status: "open"}, p); at != "" {
		t.Fatal("no due time must give none")
	}
}

func TestValidateStartDurationPriority(t *testing.T) {
	base := func() Item { return Item{Title: "x", Status: "open"} }
	for n, mutate := range []func(*Item){
		func(i *Item) { i.StartAt = "tomorrow" },
		func(i *Item) { i.DurationMinutes = -1 },
		func(i *Item) { i.DurationMinutes = 10081 },
		func(i *Item) { i.Priority = 6 },
		func(i *Item) { i.Priority = -1 },
		func(i *Item) { i.StartAt, i.DueAt = "2026-11-13T10:00:00Z", "2026-11-13T09:00:00Z" },
	} {
		i := base()
		mutate(&i)
		if Validate(&i) == nil {
			t.Errorf("case %d accepted", n)
		}
	}
	i := base()
	i.StartAt, i.DueAt, i.DurationMinutes, i.Priority = "2026-11-13T10:00:00+01:00", "2026-11-13T12:00:00Z", 30, 1
	if e := Validate(&i); e != nil || i.StartAt != "2026-11-13T09:00:00Z" {
		t.Fatalf("%v %q", e, i.StartAt)
	}
}

func TestReminderAnchorsOnDeadlineThenStart(t *testing.T) {
	p := &Profile{ID: "day-event", RemindBeforeMinutes: 60}
	event := Item{StartAt: "2026-11-13T09:00:00Z", Status: "open", ProfileID: "day-event"}
	if at, _ := EffectiveReminder(event, p); at != "2026-11-13T08:00:00Z" {
		t.Fatalf("start anchor: %q", at)
	}
	both := event
	both.DueAt = "2026-11-13T17:00:00Z"
	if at, _ := EffectiveReminder(both, p); at != "2026-11-13T16:00:00Z" {
		t.Fatalf("deadline must win: %q", at)
	}
}

func TestAdvanceMovesStartDeadlineAndReminderTogether(t *testing.T) {
	i := Item{StartAt: "2026-11-13T09:00:00Z", DueAt: "2026-11-13T17:00:00Z", RemindAt: "2026-11-13T08:30:00Z", Timezone: "Europe/Amsterdam", Recurrence: &Recurrence{Every: "weekly", Interval: 1}}
	if e := Advance(&i, at("2026-11-13T07:00:00Z")); e != nil {
		t.Fatal(e)
	}
	if i.StartAt != "2026-11-20T09:00:00Z" || i.DueAt != "2026-11-20T17:00:00Z" || i.RemindAt != "2026-11-20T08:30:00Z" {
		t.Fatalf("%+v", i)
	}
	only := Item{StartAt: "2026-11-13T09:00:00Z", Timezone: "Europe/Amsterdam", Recurrence: &Recurrence{Every: "daily", Interval: 1}}
	if e := Advance(&only, at("2026-11-13T07:00:00Z")); e != nil || only.StartAt != "2026-11-14T09:00:00Z" || only.DueAt != "" {
		t.Fatalf("start-only: %+v %v", only, e)
	}
}

func TestCompleteFinishesAnItemOrRollsARepeatingOne(t *testing.T) {
	now := at("2026-11-13T07:00:00Z")
	plain := Item{Status: "open", DueAt: "2026-11-13T09:00:00Z", Timezone: "Europe/Amsterdam"}
	if e := Complete(&plain, now); e != nil || plain.Status != "done" || plain.CompletedAt == "" || plain.DueAt != "2026-11-13T09:00:00Z" {
		t.Fatalf("plain: %v %+v", e, plain)
	}
	repeating := Item{Status: "open", DueAt: "2026-11-13T09:00:00Z", Timezone: "Europe/Amsterdam", Recurrence: &Recurrence{Every: "weekly", Interval: 1}}
	if e := Complete(&repeating, now); e != nil || repeating.Status != "open" || repeating.CompletedAt == "" || repeating.DueAt != "2026-11-20T09:00:00Z" {
		t.Fatalf("repeating: %v %+v", e, repeating)
	}
	done := Item{Status: "done"}
	if e := Complete(&done, now); !errors.Is(e, ErrConflict) {
		t.Fatalf("finished item completed again: %v", e)
	}
}

func TestValidateNamesTheBadStatus(t *testing.T) {
	i := Item{Title: "x", Status: "bogus"}
	var invalid *Invalid
	if e := Validate(&i); !errors.As(e, &invalid) || invalid.Field != "status" {
		t.Fatalf("got %v", e)
	}
}
