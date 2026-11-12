package domain

import (
	"testing"
	"time"
)

func TestSnoozeUsesTheItemZoneAndKeepsTheNextLocalDay(t *testing.T) {
	for _, tc := range []struct{ action, now, want string }{
		{"snooze15", "2026-11-12T10:00:00Z", "2026-11-12T10:15:00Z"},
		{"snooze60", "2026-11-12T10:00:00Z", "2026-11-12T11:00:00Z"},
		{"tonight", "2026-11-12T10:00:00Z", "2026-11-12T19:00:00Z"},
		{"tonight", "2026-11-12T20:00:00Z", "2026-11-13T19:00:00Z"},
		{"tomorrow", "2026-10-24T10:00:00Z", "2026-10-25T08:00:00Z"},
	} {
		t.Run(tc.action+tc.now, func(t *testing.T) {
			now, _ := time.Parse(time.RFC3339, tc.now)
			at, err := SnoozeAt(tc.action, "Europe/Amsterdam", now)
			if err != nil || at.UTC().Format(time.RFC3339) != tc.want {
				t.Fatalf("want %s, got %s %v", tc.want, at, err)
			}
		})
	}
	if _, err := SnoozeAt("other", "Europe/Amsterdam", time.Now()); err == nil {
		t.Fatal("invalid action accepted")
	}
	if _, err := SnoozeAt("tomorrow", "invalid/zone", time.Now()); err == nil {
		t.Fatal("invalid zone accepted")
	}
}
