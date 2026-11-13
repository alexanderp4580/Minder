package domain

import (
	"reflect"
	"testing"
	"time"
)

func TestNormalizeOffsets(t *testing.T) {
	got, err := NormalizeOffsets([]int{10, 1440, 60, 10, 10080})
	if err != nil || !reflect.DeepEqual(got, []int{10080, 1440, 60, 10}) {
		t.Fatal(got, err)
	}
	if got, err = NormalizeOffsets(nil); err != nil || got != nil {
		t.Fatal("empty offsets", got, err)
	}
	if _, err = NormalizeOffsets([]int{7}); err == nil {
		t.Fatal("unknown offset accepted")
	}
	if got, err = NormalizeOffsets([]int{10, 30, 60, 120, 1440, 2880, 4320, 10080, 10}); err != nil || len(got) != 8 {
		t.Fatal("eight choices plus a duplicate", got, err)
	}
}

func TestNextReminderDropsPastOffsets(t *testing.T) {
	start := time.Date(2026, 11, 20, 14, 30, 0, 0, time.UTC)
	item := Item{StartAt: start.Format(time.RFC3339), Timezone: "Europe/Amsterdam", ReminderOffsets: []int{1440, 60, 10}}
	now := start.Add(-3 * time.Hour)
	first, ok := NextReminder(item, now)
	if !ok || !first.Equal(start.Add(-time.Hour)) {
		t.Fatal("past one-day offset must be dropped", first, ok)
	}
	second, ok := NextReminder(item, first)
	if !ok || !second.Equal(start.Add(-10*time.Minute)) {
		t.Fatal(second, ok)
	}
	if _, ok = NextReminder(item, second); ok {
		t.Fatal("reminder after the last offset")
	}
	if _, ok = NextReminder(Item{StartAt: item.StartAt}, now); ok {
		t.Fatal("no offsets must not schedule")
	}
}

func TestNextReminderAnchorsDateOnlyAt0900Local(t *testing.T) {
	item := Item{StartDate: "2026-11-20", Timezone: "Europe/Amsterdam", ReminderOffsets: []int{1440}}
	at, ok := NextReminder(item, time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC))
	if !ok || at.UTC().Format(time.RFC3339) != "2026-11-19T08:00:00Z" {
		t.Fatal(at, ok)
	}
}
