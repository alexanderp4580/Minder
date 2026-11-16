package domain

import (
	"sort"
	"time"
)

// BoardDoneDays and BoardDoneLimit bound the Done column to recent work.
const (
	BoardDoneDays  = 14
	BoardDoneLimit = 30
)

// Board holds the board's three columns. Each encodes as a list, never null.
type Board struct {
	Open       []Item `json:"open"`
	InProgress []Item `json:"in_progress"`
	Done       []Item `json:"done"`
}

// BoardStatuses are the statuses a task can take on the board.
var BoardStatuses = []string{"open", "in_progress", "done"}

func ValidBoardStatus(status string) bool {
	for _, s := range BoardStatuses {
		if s == status {
			return true
		}
	}
	return false
}

// OnBoard reports whether an item belongs on the board: a live task or reminder
// with no date and no recurrence. Items with a date belong to the calendar.
// A reminder time is not a date.
func OnBoard(i Item) bool {
	if i.Kind == "event" || i.Recurrence != nil {
		return false
	}
	if i.StartAt != "" || i.DueAt != "" || i.StartDate != "" || i.DueDate != "" {
		return false
	}
	return ValidBoardStatus(i.Status)
}

// Placement names where a live item appears: the Board, the Calendar, or "" for a cancelled item.
func Placement(i Item) string {
	if i.Status == "cancelled" {
		return ""
	}
	if OnBoard(i) {
		return "Board"
	}
	return "Calendar"
}

// doneAt is when a done task finished: its completion time, else its last update.
func doneAt(i Item) time.Time {
	for _, s := range []string{i.CompletedAt, i.UpdatedAt} {
		if t, e := time.Parse(time.RFC3339Nano, s); e == nil {
			return t
		}
	}
	return time.Time{}
}

// createdAt parses the creation time. RFC3339Nano drops trailing zeros, so the text does not sort by time.
func createdAt(i Item) time.Time {
	t, _ := time.Parse(time.RFC3339Nano, i.CreatedAt)
	return t
}

// BoardColumns groups the items on the board. Open and in-progress tasks sort by
// priority (1 first, unset last), then oldest first. Done tasks show the newest
// BoardDoneLimit finished within BoardDoneDays.
func BoardColumns(items []Item, now time.Time) Board {
	b := Board{Open: []Item{}, InProgress: []Item{}, Done: []Item{}}
	cutoff := now.AddDate(0, 0, -BoardDoneDays)
	for _, i := range items {
		if !OnBoard(i) {
			continue
		}
		switch i.Status {
		case "open":
			b.Open = append(b.Open, i)
		case "in_progress":
			b.InProgress = append(b.InProgress, i)
		case "done":
			if doneAt(i).After(cutoff) {
				b.Done = append(b.Done, i)
			}
		}
	}
	rank := func(i Item) int {
		if i.Priority == 0 {
			return 6
		}
		return i.Priority
	}
	for _, col := range [][]Item{b.Open, b.InProgress} {
		sort.SliceStable(col, func(a, c int) bool {
			if rank(col[a]) != rank(col[c]) {
				return rank(col[a]) < rank(col[c])
			}
			return createdAt(col[a]).Before(createdAt(col[c]))
		})
	}
	sort.SliceStable(b.Done, func(a, c int) bool { return doneAt(b.Done[a]).After(doneAt(b.Done[c])) })
	if len(b.Done) > BoardDoneLimit {
		b.Done = b.Done[:BoardDoneLimit]
	}
	return b
}
