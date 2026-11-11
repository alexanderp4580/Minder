package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

var ErrConflict = errors.New("The item changed. Request a new preview.")
var ErrNotFound = errors.New("The item was not found.")
var ErrLimit = errors.New("The monthly limit is reached.")

type Invalid struct {
	Field   string
	Message string
}

func (e *Invalid) Error() string { return e.Message }

type Item struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Status      string `json:"status"`
	DueAt       string `json:"due_at,omitempty"`
	Timezone    string `json:"timezone"`
	Notes       string `json:"notes,omitempty"`
	Version     int    `json:"version"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
	CompletedAt string `json:"completed_at,omitempty"`
}
type Proposal struct {
	Action          string `json:"action"`
	ItemID          string `json:"item_id,omitempty"`
	ExpectedVersion int    `json:"expected_version,omitempty"`
	Title           string `json:"title,omitempty"`
	DueAt           string `json:"due_at,omitempty"`
	Timezone        string `json:"timezone,omitempty"`
	Notes           string `json:"notes,omitempty"`
	ClearDue        bool   `json:"clear_due,omitempty"`
	ClearNotes      bool   `json:"clear_notes,omitempty"`
}
type Field struct {
	Name string `json:"name"`
	Old  string `json:"old,omitempty"`
	New  string `json:"new,omitempty"`
}
type Change struct {
	ID        string  `json:"id"`
	Action    string  `json:"action"`
	ItemTitle string  `json:"item_title"`
	Fields    []Field `json:"fields"`
	State     string  `json:"state"`
	UndoID    string  `json:"undo_id,omitempty"`
}
type Entry struct {
	ID        string  `json:"id"`
	Cursor    int64   `json:"cursor"`
	Kind      string  `json:"kind"`
	Text      string  `json:"text"`
	CreatedAt string  `json:"created_at"`
	Change    *Change `json:"change,omitempty"`
}
type Pending struct {
	ID       string
	Proposal Proposal
	Before   *Item
	After    Item
	State    string
	EntryID  string
	ResultID string
}
type Audit struct {
	ID     string
	Before *Item
	After  Item
	Undone bool
	Actor  string
}

func Now() string { return time.Now().UTC().Format(time.RFC3339Nano) }
func Validate(i *Item) error {
	i.Title = strings.TrimSpace(i.Title)
	if n := utf8.RuneCountInString(i.Title); n < 1 || n > 200 {
		return &Invalid{"title", "Use a title with 1 to 200 characters."}
	}
	if utf8.RuneCountInString(i.Notes) > 4000 {
		return &Invalid{"notes", "Use notes with at most 4,000 characters."}
	}
	if i.Timezone == "" {
		i.Timezone = "Europe/Amsterdam"
	}
	if _, e := time.LoadLocation(i.Timezone); e != nil {
		return &Invalid{"timezone", "Choose a valid timezone."}
	}
	if i.DueAt != "" {
		t, e := time.Parse(time.RFC3339, i.DueAt)
		if e != nil {
			return &Invalid{"due_at", "Use a date and time with a timezone."}
		}
		i.DueAt = t.UTC().Format(time.RFC3339)
	}
	switch i.Status {
	case "open", "in_progress", "done", "cancelled":
	default:
		return fmt.Errorf("invalid status")
	}
	return nil
}
