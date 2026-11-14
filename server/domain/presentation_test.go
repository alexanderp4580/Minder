package domain

import (
	"testing"
	"time"
)

func TestPresentationValidation(t *testing.T) {
	for _, c := range []string{"blue", "teal", "green", "amber", "lilac", "slate", ""} {
		i := Item{Title: "Event", Status: "open", Color: c}
		if err := Validate(&i); err != nil {
			t.Fatal(c, err)
		}
		if i.Color == "" {
			t.Fatal("missing fallback colour")
		}
	}
	for _, i := range []Item{{Color: "red"}, {LocationKind: "guess"}, {DateEnd: "maybe"}} {
		i.Title = "Event"
		i.Status = "open"
		if err := Validate(&i); err == nil {
			t.Fatal("invalid presentation accepted", i)
		}
	}
}
func TestInclusiveDateRangeAndCheckout(t *testing.T) {
	zone, _ := time.LoadLocation("Europe/Amsterdam")
	from := time.Date(2026, 11, 20, 0, 0, 0, 0, zone)
	to := from.AddDate(0, 0, 1)
	for _, tc := range []struct {
		mode string
		want bool
	}{{"inclusive", true}, {"checkout", false}, {"", false}} {
		i := Item{StartDate: "2026-11-17", EndDate: "2026-11-20", DateEnd: tc.mode}
		if got := Overlaps(i, from, to, zone); got != tc.want {
			t.Fatal(tc, got)
		}
	}
	i := Item{Title: "One day", Status: "open", StartDate: "2026-11-17", EndDate: "2026-11-17", DateEnd: "inclusive"}
	if err := Validate(&i); err != nil {
		t.Fatal(err)
	}
}
