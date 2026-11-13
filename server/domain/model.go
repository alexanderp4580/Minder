package domain

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

var ErrConflict = errors.New("The item changed. Request a new preview.")
var ErrNotFound = errors.New("The item was not found.")
var ErrLimit = errors.New("The monthly limit is reached.")

// ModelError is a failure of the language model or its provider.
type ModelError struct{ Message string }

func (e *ModelError) Error() string { return e.Message }

type Invalid struct {
	Field   string
	Message string
}

func (e *Invalid) Error() string { return e.Message }

type Item struct {
	Phone            string `json:"phone,omitempty"`
	Email            string `json:"email,omitempty"`
	Website          string `json:"website,omitempty"`
	BookingReference string `json:"booking_reference,omitempty"`

	SourceIDs     []string `json:"source_ids,omitempty"`
	Kind          string   `json:"kind,omitempty"`
	EndAt         string   `json:"end_at,omitempty"`
	StartTimezone string   `json:"start_timezone,omitempty"`
	EndTimezone   string   `json:"end_timezone,omitempty"`
	StartDate     string   `json:"start_date,omitempty"`
	EndDate       string   `json:"end_date,omitempty"`
	DueDate       string   `json:"due_date,omitempty"`

	StartAt         string      `json:"start_at,omitempty"`
	DurationMinutes int         `json:"duration_minutes,omitempty"`
	Priority        int         `json:"priority,omitempty"`
	Location        string      `json:"location,omitempty"`
	List            string      `json:"list,omitempty"`
	ProfileID       string      `json:"profile,omitempty"`
	Recurrence      *Recurrence `json:"recurrence,omitempty"`
	RemindAt        string      `json:"remind_at,omitempty"`
	NagMinutes      int         `json:"nag_minutes,omitempty"`
	ReminderOffsets []int       `json:"reminder_offsets_minutes,omitempty"`
	ID              string      `json:"id"`
	Title           string      `json:"title"`
	Status          string      `json:"status"`
	DueAt           string      `json:"due_at,omitempty"`
	Timezone        string      `json:"timezone"`
	Notes           string      `json:"notes,omitempty"`
	Version         int         `json:"version"`
	CreatedAt       string      `json:"created_at"`
	UpdatedAt       string      `json:"updated_at"`
	CompletedAt     string      `json:"completed_at,omitempty"`
}
type Proposal struct {
	Phone            string `json:"phone,omitempty"`
	Email            string `json:"email,omitempty"`
	Website          string `json:"website,omitempty"`
	BookingReference string `json:"booking_reference,omitempty"`

	ClearPhone            bool `json:"clear_phone,omitempty"`
	ClearEmail            bool `json:"clear_email,omitempty"`
	ClearWebsite          bool `json:"clear_website,omitempty"`
	ClearBookingReference bool `json:"clear_booking_reference,omitempty"`

	CaptureID       string `json:"capture_id,omitempty"`
	CandidateID     string `json:"candidate_id,omitempty"`
	CaptureRevision int    `json:"capture_revision,omitempty"`

	Kind          string `json:"kind,omitempty"`
	EndAt         string `json:"end_at,omitempty"`
	StartTimezone string `json:"start_timezone,omitempty"`
	EndTimezone   string `json:"end_timezone,omitempty"`
	StartDate     string `json:"start_date,omitempty"`
	EndDate       string `json:"end_date,omitempty"`
	DueDate       string `json:"due_date,omitempty"`

	ClearEnd           bool `json:"clear_end,omitempty"`
	ClearStartDate     bool `json:"clear_start_date,omitempty"`
	ClearEndDate       bool `json:"clear_end_date,omitempty"`
	ClearDueDate       bool `json:"clear_due_date,omitempty"`
	ClearStartTimezone bool `json:"clear_start_timezone,omitempty"`
	ClearEndTimezone   bool `json:"clear_end_timezone,omitempty"`

	StartAt              string      `json:"start_at,omitempty"`
	ClearStart           bool        `json:"clear_start,omitempty"`
	DurationMinutes      int         `json:"duration_minutes,omitempty"`
	ClearDuration        bool        `json:"clear_duration,omitempty"`
	Priority             int         `json:"priority,omitempty"`
	ClearPriority        bool        `json:"clear_priority,omitempty"`
	Location             string      `json:"location,omitempty"`
	ClearLocation        bool        `json:"clear_location,omitempty"`
	List                 string      `json:"list,omitempty"`
	ClearList            bool        `json:"clear_list,omitempty"`
	Profile              string      `json:"profile,omitempty"`
	ClearProfile         bool        `json:"clear_profile,omitempty"`
	Recurrence           *Recurrence `json:"recurrence,omitempty"`
	ClearRecurrence      bool        `json:"clear_recurrence,omitempty"`
	ProfileID            string      `json:"profile_id,omitempty"`
	ProfileName          string      `json:"profile_name,omitempty"`
	ProfileRemindBefore  *int        `json:"profile_remind_before_minutes,omitempty"`
	ProfileNag           *int        `json:"profile_nag_minutes,omitempty"`
	RemindAt             string      `json:"remind_at,omitempty"`
	NagMinutes           int         `json:"nag_minutes,omitempty"`
	ClearReminder        bool        `json:"clear_reminder,omitempty"`
	ReminderOffsets      []int       `json:"reminder_offsets_minutes,omitempty"`
	ClearReminderOffsets bool        `json:"clear_reminder_offsets,omitempty"`
	ClearNag             bool        `json:"clear_nag,omitempty"`
	Action               string      `json:"action"`
	ItemID               string      `json:"item_id,omitempty"`
	ExpectedVersion      int         `json:"expected_version,omitempty"`
	Title                string      `json:"title,omitempty"`
	DueAt                string      `json:"due_at,omitempty"`
	Timezone             string      `json:"timezone,omitempty"`
	Notes                string      `json:"notes,omitempty"`
	ClearDue             bool        `json:"clear_due,omitempty"`
	ClearNotes           bool        `json:"clear_notes,omitempty"`
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

	ReminderOffsets []int `json:"reminder_offsets_minutes,omitempty"`
	ReminderChoices []int `json:"reminder_choices_minutes,omitempty"`

	// RemovedEntryIDs lists chat entries a delete took away, so clients can drop them.
	RemovedEntryIDs []string `json:"removed_entry_ids,omitempty"`
}
type Entry struct {
	CaptureID string   `json:"capture_id,omitempty"`
	Capture   *Capture `json:"capture,omitempty"`

	CostEUR   float64   `json:"cost_eur,omitempty"`
	Reminder  *Reminder `json:"reminder,omitempty"`
	ID        string    `json:"id"`
	Cursor    int64     `json:"cursor"`
	Kind      string    `json:"kind"`
	Text      string    `json:"text"`
	CreatedAt string    `json:"created_at"`
	Change    *Change   `json:"change,omitempty"`
}
type Pending struct {
	BeforeCapture *Capture

	BeforeProfile *Profile
	AfterProfile  *Profile
	ID            string
	Proposal      Proposal
	Before        *Item
	After         Item
	State         string
	EntryID       string
	ResultID      string
}
type Audit struct {
	DeletedCaptureID string
	ItemDeleted      bool

	BeforeProfile *Profile
	AfterProfile  *Profile
	ID            string
	Before        *Item
	After         Item
	Undone        bool
	Actor         string
}

func Now() string { return time.Now().UTC().Format(time.RFC3339Nano) }
func Validate(i *Item) error {
	NormalizeBookingDetails(i)
	i.Title = strings.TrimSpace(i.Title)
	if n := utf8.RuneCountInString(i.Title); n < 1 || n > 200 {
		return &Invalid{"title", "Use a title with 1 to 200 characters."}
	}
	if utf8.RuneCountInString(i.Notes) > 4000 {
		return &Invalid{"notes", "Use notes with at most 4,000 characters."}
	}
	for _, field := range []struct {
		name  string
		value *string
		limit int
	}{
		{"phone", &i.Phone, 200}, {"email", &i.Email, 200},
		{"website", &i.Website, 2048}, {"booking_reference", &i.BookingReference, 200},
	} {
		*field.value = strings.TrimSpace(*field.value)
		if utf8.RuneCountInString(*field.value) > field.limit {
			return &Invalid{field.name, "The booking detail is too long."}
		}
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
	if i.StartAt != "" {
		t, e := time.Parse(time.RFC3339, i.StartAt)
		if e != nil {
			return &Invalid{"start_at", "Use a start date and time with a timezone."}
		}
		i.StartAt = t.UTC().Format(time.RFC3339)
	}
	if i.StartAt != "" && i.DueAt != "" && i.StartAt > i.DueAt {
		return &Invalid{"start_at", "The start must not be after the deadline."}
	}
	if err := validateEvent(i); err != nil {
		return err
	}
	if i.DurationMinutes < 0 || (i.DurationMinutes > 10080 && i.EndAt == "") {
		return &Invalid{"duration_minutes", "Use a duration from 1 minute to 1 week."}
	}
	if i.Priority < 0 || i.Priority > 5 {
		return &Invalid{"priority", "Use a priority from 1 (highest) to 5."}
	}
	offsets, e := NormalizeOffsets(i.ReminderOffsets)
	if e != nil {
		return e
	}
	i.ReminderOffsets = offsets
	if i.NagMinutes < 0 || i.NagMinutes > 1440 || (i.NagMinutes != 0 && i.RemindAt == "" && len(offsets) == 0) {
		return &Invalid{"nag_minutes", "Set a reminder and a nag interval from 0 to 1,440 minutes."}
	}
	if i.RemindAt != "" {
		t, e := time.Parse(time.RFC3339, i.RemindAt)
		if e != nil {
			return &Invalid{"remind_at", "Use a reminder date and time with a timezone."}
		}
		i.RemindAt = t.UTC().Format(time.RFC3339)
	}
	i.List = strings.TrimSpace(i.List)
	if utf8.RuneCountInString(i.List) > 60 {
		return &Invalid{"list", "Use a list name with at most 60 characters."}
	}
	i.Location = strings.TrimSpace(i.Location)
	if utf8.RuneCountInString(i.Location) > 200 {
		return &Invalid{"location", "Use a location with at most 200 characters."}
	}
	if e := validateRecurrence(i); e != nil {
		return e
	}
	switch i.Status {
	case "open", "in_progress", "done", "cancelled":
	default:
		return &Invalid{"status", "Use open, in progress, done or cancelled."}
	}
	return nil
}

// Reminder identifies one firing, so stale notification actions cannot alter a newer firing.
type Reminder struct {
	ID          string `json:"id"`
	ItemID      string `json:"item_id"`
	Title       string `json:"title"`
	ScheduledAt string `json:"scheduled_at"`
	State       string `json:"state"`
	Late        bool   `json:"late"`
}
