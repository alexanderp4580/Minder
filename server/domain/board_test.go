package domain

import (
	"fmt"
	"testing"
	"time"
)

var boardNow = time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)

func task(id, status string, edit func(*Item)) Item {
	i := Item{ID: id, Title: id, Status: status, Kind: "task", CreatedAt: "2026-11-01T10:00:00Z", UpdatedAt: "2026-11-01T10:00:00Z"}
	if edit != nil {
		edit(&i)
	}
	return i
}

func TestOnBoardKeepsUndatedLiveTasksOnly(t *testing.T) {
	for name, tc := range map[string]struct {
		item Item
		want bool
	}{
		"plain task":             {task("a", "open", nil), true},
		"empty kind":             {task("a", "open", func(i *Item) { i.Kind = "" }), true},
		"reminder kind":          {task("a", "open", func(i *Item) { i.Kind = "reminder" }), true},
		"in progress":            {task("a", "in_progress", nil), true},
		"done":                   {task("a", "done", nil), true},
		"cancelled":              {task("a", "cancelled", nil), false},
		"event":                  {task("a", "open", func(i *Item) { i.Kind = "event" }), false},
		"due time":               {task("a", "open", func(i *Item) { i.DueAt = "2026-11-20T10:00:00Z" }), false},
		"start time":             {task("a", "open", func(i *Item) { i.StartAt = "2026-11-20T10:00:00Z" }), false},
		"start date":             {task("a", "open", func(i *Item) { i.StartDate = "2026-11-20" }), false},
		"due date":               {task("a", "open", func(i *Item) { i.DueDate = "2026-11-20" }), false},
		"recurring":              {task("a", "open", func(i *Item) { i.Recurrence = &Recurrence{} }), false},
		"reminder is not a date": {task("a", "open", func(i *Item) { i.RemindAt = "2026-11-20T10:00:00Z" }), true},
		"end only":               {task("a", "open", func(i *Item) { i.EndAt = "2026-11-20T10:00:00Z" }), true},
	} {
		t.Run(name, func(t *testing.T) {
			if got := OnBoard(tc.item); got != tc.want {
				t.Fatalf("OnBoard = %v, want %v", got, tc.want)
			}
		})
	}
}

func ids(items []Item) string {
	out := ""
	for _, i := range items {
		out += i.ID + " "
	}
	return out
}

func TestBoardColumnsSortByPriorityThenAgeAndDoneByCompletion(t *testing.T) {
	items := []Item{
		task("late-none", "open", func(i *Item) { i.CreatedAt = "2026-11-10T00:00:00Z" }),
		task("old-none", "open", func(i *Item) { i.CreatedAt = "2026-11-02T00:00:00Z" }),
		task("p2", "open", func(i *Item) { i.Priority = 2; i.CreatedAt = "2026-11-12T00:00:00Z" }),
		task("p1", "open", func(i *Item) { i.Priority = 1; i.CreatedAt = "2026-11-14T00:00:00Z" }),
		task("doing", "in_progress", nil),
		task("done-old", "done", func(i *Item) { i.CompletedAt = "2026-11-10T09:00:00Z" }),
		task("done-new", "done", func(i *Item) { i.CompletedAt = "2026-11-15T09:00:00Z" }),
		task("dated", "open", func(i *Item) { i.DueAt = "2026-11-20T10:00:00Z" }),
		task("gone", "cancelled", nil),
	}
	b := BoardColumns(items, boardNow)
	if got := ids(b.Open); got != "p1 p2 old-none late-none " {
		t.Fatalf("open = %q", got)
	}
	if got := ids(b.InProgress); got != "doing " {
		t.Fatalf("in progress = %q", got)
	}
	if got := ids(b.Done); got != "done-new done-old " {
		t.Fatalf("done = %q", got)
	}
}

func TestBoardDoneWindowAndCap(t *testing.T) {
	var items []Item
	items = append(items,
		task("stale", "done", func(i *Item) { i.CompletedAt = "2026-11-01T00:00:00Z" }),
		task("no-stamp-recent", "done", func(i *Item) { i.UpdatedAt = "2026-11-15T00:00:00Z" }),
		task("no-stamp-stale", "done", func(i *Item) { i.UpdatedAt = "2026-10-01T00:00:00Z" }),
	)
	for n := 0; n < 40; n++ {
		items = append(items, task(fmt.Sprintf("d%02d", n), "done", func(i *Item) {
			i.CompletedAt = boardNow.Add(-time.Duration(n+1) * time.Hour).Format(time.RFC3339)
		}))
	}
	done := BoardColumns(items, boardNow).Done
	if len(done) != BoardDoneLimit {
		t.Fatalf("done = %d, want %d", len(done), BoardDoneLimit)
	}
	for _, i := range done {
		if i.ID == "stale" || i.ID == "no-stamp-stale" {
			t.Fatalf("old done task %s shown", i.ID)
		}
	}
	if done[0].ID != "d00" {
		t.Fatalf("first = %s", done[0].ID)
	}
}

func TestBoardColumnsAreNeverNil(t *testing.T) {
	b := BoardColumns(nil, boardNow)
	if b.Open == nil || b.InProgress == nil || b.Done == nil {
		t.Fatal("empty columns must encode as []")
	}
}

func TestBoardOrdersByRealTimeNotTimestampText(t *testing.T) {
	// RFC3339Nano trims zeros, so "05.5Z" would sort after "05.50001Z" as text.
	items := []Item{
		task("newer", "open", func(i *Item) { i.CreatedAt = "2026-11-10T10:00:05.50001Z" }),
		task("older", "open", func(i *Item) { i.CreatedAt = "2026-11-10T10:00:05.5Z" }),
		task("whole", "open", func(i *Item) { i.CreatedAt = "2026-11-10T10:00:05Z" }),
	}
	if got := ids(BoardColumns(items, boardNow).Open); got != "whole older newer " {
		t.Fatalf("order = %q", got)
	}
}

func TestPlacementNamesWhereAnItemAppears(t *testing.T) {
	for name, tc := range map[string]struct {
		item Item
		want string
	}{
		"undated task":  {task("a", "open", nil), "Board"},
		"done task":     {task("a", "done", nil), "Board"},
		"due time":      {task("a", "open", func(i *Item) { i.DueAt = "2026-11-20T10:00:00Z" }), "Calendar"},
		"start date":    {task("a", "in_progress", func(i *Item) { i.StartDate = "2026-11-20" }), "Calendar"},
		"recurring":     {task("a", "open", func(i *Item) { i.Recurrence = &Recurrence{} }), "Calendar"},
		"event":         {task("a", "open", func(i *Item) { i.Kind = "event" }), "Calendar"},
		"cancelled":     {task("a", "cancelled", nil), ""},
		"cancelled due": {task("a", "cancelled", func(i *Item) { i.DueAt = "2026-11-20T10:00:00Z" }), ""},
	} {
		t.Run(name, func(t *testing.T) {
			if got := Placement(tc.item); got != tc.want {
				t.Fatalf("Placement = %q, want %q", got, tc.want)
			}
		})
	}
}
