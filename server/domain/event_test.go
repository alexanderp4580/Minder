package domain

import (
	"encoding/json"
	"testing"
	"time"
)

func TestEventDates(t *testing.T) {
	for _, tc := range []struct {
		name, fields string
		invalid      bool
	}{
		{"stay", `"kind":"event","start_date":"2026-11-01","end_date":"2026-11-15"`, false},
		{"bad date", `"start_date":"2026-02-30"`, true},
		{"mixed", `"start_date":"2026-11-01","start_at":"2026-11-01T12:00:00Z"`, true},
		{"orphan end", `"end_at":"2026-11-01T12:00:00Z"`, true},
		{"reversed", `"start_date":"2026-11-03","end_date":"2026-11-01"`, true},
		{"date deadline", `"due_date":"2026-11-20"`, false},
		{"invalid kind", `"kind":"unknown"`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var i Item
			if err := json.Unmarshal([]byte(`{"title":"Booking","status":"open",`+tc.fields+`}`), &i); err != nil {
				t.Fatal(err)
			}
			err := Validate(&i)
			if (err != nil) != tc.invalid {
				t.Fatalf("Validate error=%v, want invalid=%v", err, tc.invalid)
			}
		})
	}
}
func TestFlightZones(t *testing.T) {
	var i Item
	json.Unmarshal([]byte(`{"title":"Flight","status":"open","kind":"event","start_at":"2026-11-01T12:00:00+01:00","end_at":"2026-11-01T14:00:00-05:00","start_timezone":"Europe/Amsterdam","end_timezone":"America/New_York"}`), &i)
	if err := Validate(&i); err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(i)
	var out map[string]any
	json.Unmarshal(b, &out)
	if out["end_at"] != "2026-11-01T19:00:00Z" || i.DurationMinutes != 480 {
		t.Fatalf("flight %#v", out)
	}
}
func TestCalendarOverlap(t *testing.T) {
	var i Item
	json.Unmarshal([]byte(`{"title":"Stay","status":"open","start_date":"2026-10-28","end_date":"2026-11-03"}`), &i)
	from, _ := time.Parse(time.RFC3339, "2026-11-01T00:00:00Z")
	if !Overlaps(i, from, from.Add(24*time.Hour), time.UTC) {
		t.Fatal("stay missing on occupied day")
	}
	if Overlaps(i, from.Add(48*time.Hour), from.Add(72*time.Hour), time.UTC) {
		t.Fatal("checkout occupied")
	}
}

func TestLegacyItems(t *testing.T) {
	i := Item{Title: "Legacy", Status: "open", StartAt: "2026-11-12T09:00:00Z", Recurrence: &Recurrence{Every: "daily", Interval: 1}}
	if err := Validate(&i); err != nil {
		t.Fatal(err)
	}
	if err := Advance(&i, time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	if i.StartAt != "2026-11-13T09:00:00Z" {
		t.Fatal(i.StartAt)
	}
}
func TestEventRecurrence(t *testing.T) {
	i := Item{Title: "Stay", Status: "open", StartDate: "2026-11-01", EndDate: "2026-11-03", Recurrence: &Recurrence{Every: "monthly", Interval: 1}}
	if err := Validate(&i); err != nil {
		t.Fatal(err)
	}
	if err := Advance(&i, time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	if i.StartDate != "2026-12-01" || i.EndDate != "2026-12-03" {
		t.Fatalf("%+v", i)
	}
	flight := Item{Title: "Flight", Status: "open", StartAt: "2026-11-12T09:00:00Z", EndAt: "2026-11-12T11:00:00Z", Recurrence: &Recurrence{Every: "daily", Interval: 1}}
	if err := Advance(&flight, time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	if flight.EndAt != "2026-11-13T11:00:00Z" {
		t.Fatal("end not advanced", flight.EndAt)
	}
}
