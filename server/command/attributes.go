package command

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/port"
)

// setAttributes applies optional item fields to a proposal preview.
func setAttributes(t port.Transaction, p domain.Proposal, i *domain.Item) error {
	// A duration previously derived from the event range is not a new constraint.
	if i.EndAt != "" && (p.StartAt != "" || p.EndAt != "" || p.ClearEnd) && p.DurationMinutes == 0 {
		i.DurationMinutes = 0
	}

	if p.Kind != "" {
		i.Kind = p.Kind
	}
	for _, f := range []struct {
		dst   *string
		value string
		clear bool
	}{
		{&i.EndAt, p.EndAt, p.ClearEnd}, {&i.StartDate, p.StartDate, p.ClearStartDate},
		{&i.EndDate, p.EndDate, p.ClearEndDate}, {&i.DueDate, p.DueDate, p.ClearDueDate},
		{&i.StartTimezone, p.StartTimezone, p.ClearStartTimezone}, {&i.EndTimezone, p.EndTimezone, p.ClearEndTimezone},
	} {
		if f.clear {
			*f.dst = ""
		}
		if f.value != "" {
			*f.dst = f.value
		}
	}

	if p.ClearStart {
		i.StartAt = ""
	}
	if p.StartAt != "" {
		i.StartAt = p.StartAt
	}
	if p.ClearDuration {
		i.DurationMinutes = 0
	}
	if p.DurationMinutes != 0 {
		i.DurationMinutes = p.DurationMinutes
	}
	if p.ClearPriority {
		i.Priority = 0
	}
	if p.Priority != 0 {
		i.Priority = p.Priority
	}
	if p.ClearLocation {
		i.Location = ""
	}
	if p.Location != "" {
		i.Location = p.Location
	}
	if p.ClearReminderOffsets {
		i.ReminderOffsets = nil
	}
	if len(p.ReminderOffsets) > 0 {
		i.ReminderOffsets = p.ReminderOffsets
	}
	if p.ClearList {
		i.List = ""
	}
	if name := strings.TrimSpace(p.List); name != "" {
		names, e := t.ListNames()
		if e != nil {
			return e
		}
		i.List = name
		for _, n := range names {
			if strings.EqualFold(n, name) {
				i.List = n
			}
		}
	}
	if p.ClearProfile {
		i.ProfileID = ""
	}
	if p.Profile != "" {
		if _, e := t.Profile(p.Profile); e != nil {
			if e == domain.ErrNotFound {
				return &domain.Invalid{Field: "profile", Message: "Choose an existing profile."}
			}
			return e
		}
		i.ProfileID = p.Profile
	}
	if p.ClearRecurrence {
		i.Recurrence = nil
	}
	if p.Recurrence != nil {
		r := *p.Recurrence
		i.Recurrence = &r
	}
	return nil
}

func recurrenceText(r *domain.Recurrence) string {
	if r == nil {
		return ""
	}
	unit := map[string]string{"daily": "day", "weekly": "week", "monthly": "month", "yearly": "year"}[r.Every]
	text := "every " + unit
	if r.Interval > 1 {
		text = fmt.Sprintf("every %d %ss", r.Interval, unit)
	}
	if r.AfterCompletion {
		text += " after completion"
	}
	return text
}

// itemFields lists what differs between two versions of an item, for the preview.
func itemFields(t port.Transaction, old, now domain.Item) []domain.Field {
	all := []domain.Field{
		{Name: "title", Old: old.Title, New: now.Title},
		{Name: "start_at", Old: old.StartAt, New: now.StartAt},
		{Name: "due_at", Old: old.DueAt, New: now.DueAt},
		{Name: "duration_minutes", Old: zeroEmpty(old.DurationMinutes), New: zeroEmpty(now.DurationMinutes)},
		{Name: "remind_at", Old: old.RemindAt, New: now.RemindAt},
		{Name: "reminders", Old: offsetsText(old.ReminderOffsets), New: offsetsText(now.ReminderOffsets)},
		{Name: "nag_minutes", Old: fmt.Sprint(old.NagMinutes), New: fmt.Sprint(now.NagMinutes)},
		{Name: "priority", Old: zeroEmpty(old.Priority), New: zeroEmpty(now.Priority)},
		{Name: "location", Old: old.Location, New: now.Location},
		{Name: "list", Old: old.List, New: now.List},
		{Name: "profile", Old: profileName(t, old.ProfileID), New: profileName(t, now.ProfileID)},
		{Name: "recurrence", Old: recurrenceText(old.Recurrence), New: recurrenceText(now.Recurrence)},
		{Name: "notes", Old: old.Notes, New: now.Notes},
		{Name: "status", Old: old.Status, New: now.Status},
		{Name: "timezone", Old: old.Timezone, New: now.Timezone},
	}
	for _, f := range []domain.Field{
		{Name: "kind", Old: old.Kind, New: now.Kind},
		{Name: "end_at", Old: old.EndAt, New: now.EndAt},
		{Name: "start_date", Old: old.StartDate, New: now.StartDate},
		{Name: "end_date", Old: old.EndDate, New: now.EndDate},
		{Name: "due_date", Old: old.DueDate, New: now.DueDate},
		{Name: "start_timezone", Old: old.StartTimezone, New: now.StartTimezone},
		{Name: "end_timezone", Old: old.EndTimezone, New: now.EndTimezone},
	} {
		if f.Name == "kind" && f.Old == "" && f.New == "task" {
			continue
		}
		all = append(all, f)
	}
	changed := []domain.Field{}
	for _, f := range all {
		if f.Old != f.New {
			if zone := displayZone(f.Name, now); zone != "" {
				f.Old, f.New = inZone(f.Old, zone), inZone(f.New, zone)
			}
			changed = append(changed, f)
		}
	}
	return changed
}

// displayZone is the zone a time field is shown in: its own timezone, else the start's, else the item's.
func displayZone(name string, i domain.Item) string {
	switch name {
	case "start_at":
		return firstOf(i.StartTimezone, i.Timezone)
	case "end_at":
		return firstOf(i.EndTimezone, i.StartTimezone, i.Timezone)
	case "due_at", "remind_at":
		return i.Timezone
	}
	return ""
}

func firstOf(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// inZone rewrites an RFC3339 instant with the zone's offset, so the app shows the wall clock there.
func inZone(value, zone string) string {
	t, e := time.Parse(time.RFC3339, value)
	loc, le := time.LoadLocation(zone)
	if e != nil || le != nil {
		return value
	}
	return t.In(loc).Format(time.RFC3339)
}

var notSlug = regexp.MustCompile(`[^a-z0-9]+`)

func profileFields(old, now domain.Profile) []domain.Field {
	out := []domain.Field{}
	for _, f := range []domain.Field{
		{Name: "name", Old: old.Name, New: now.Name},
		{Name: "remind_before_minutes", Old: fmt.Sprint(old.RemindBeforeMinutes), New: fmt.Sprint(now.RemindBeforeMinutes)},
		{Name: "nag_minutes", Old: fmt.Sprint(old.NagMinutes), New: fmt.Sprint(now.NagMinutes)},
	} {
		if f.Old != f.New || old.ID == "" {
			out = append(out, f)
		}
	}
	return out
}

func previewProfile(t port.Transaction, p domain.Proposal) (domain.Entry, error) {
	var before *domain.Profile
	var after domain.Profile
	if p.Action == "create_profile" {
		slug := strings.Trim(notSlug.ReplaceAllString(strings.ToLower(p.ProfileName), "-"), "-")
		if slug == "" {
			return domain.Entry{}, &domain.Invalid{Field: "name", Message: "Use a profile name with 1 to 60 characters."}
		}
		if _, e := t.Profile(slug); e == nil {
			return domain.Entry{}, &domain.Invalid{Field: "name", Message: "A profile with this name exists."}
		} else if e != domain.ErrNotFound {
			return domain.Entry{}, e
		}
		after = domain.Profile{ID: slug, Name: p.ProfileName, Version: 1}
	} else {
		current, e := t.Profile(p.ProfileID)
		if e != nil {
			return domain.Entry{}, e
		}
		if p.ExpectedVersion < 1 || p.ExpectedVersion != current.Version {
			return domain.Entry{}, domain.ErrConflict
		}
		before = &current
		after = current
		after.Version++
		if p.ProfileName != "" {
			after.Name = p.ProfileName
		}
	}
	if p.ProfileRemindBefore != nil {
		after.RemindBeforeMinutes = *p.ProfileRemindBefore
	}
	if p.ProfileNag != nil {
		after.NagMinutes = *p.ProfileNag
	}
	if e := domain.ValidateProfile(&after); e != nil {
		return domain.Entry{}, e
	}
	id := domain.NewID()
	old := domain.Profile{}
	if before != nil {
		old = *before
	}
	entry, e := t.PutEntry(domain.Entry{Kind: "preview", Text: "Review this change.", Change: &domain.Change{ID: id, Action: p.Action, ItemTitle: after.Name, Fields: profileFields(old, after), State: "pending"}})
	if e != nil {
		return entry, e
	}
	return entry, t.PutPending(domain.Pending{ID: id, Proposal: p, BeforeProfile: before, AfterProfile: &after, State: "pending", EntryID: entry.ID})
}

func confirmProfile(t port.Transaction, p *domain.Pending, out *domain.Entry) error {
	if p.BeforeProfile != nil {
		current, e := t.Profile(p.BeforeProfile.ID)
		if e != nil {
			if errors.Is(e, domain.ErrNotFound) {
				return domain.ErrConflict
			}
			return e
		}
		if current.Version != p.BeforeProfile.Version {
			return domain.ErrConflict
		}
	} else if _, e := t.Profile(p.AfterProfile.ID); e == nil {
		return domain.ErrConflict
	} else if !errors.Is(e, domain.ErrNotFound) {
		return e
	}
	if e := t.PutProfile(*p.AfterProfile); e != nil {
		return e
	}
	audit := domain.Audit{ID: domain.NewID(), BeforeProfile: p.BeforeProfile, AfterProfile: p.AfterProfile, Actor: "agent"}
	if e := t.PutAudit(audit); e != nil {
		return e
	}
	preview, e := t.Entry(p.EntryID)
	if e != nil {
		return e
	}
	preview.Change.State = "confirmed"
	if _, e = t.PutEntry(preview); e != nil {
		return e
	}
	*out, e = t.PutEntry(domain.Entry{Kind: "result", Text: "Saved: " + p.AfterProfile.Name + ".", Change: &domain.Change{ID: p.ID, Action: p.Proposal.Action, ItemTitle: p.AfterProfile.Name, Fields: preview.Change.Fields, State: "confirmed", UndoID: audit.ID}})
	if e != nil {
		return e
	}
	p.State = "confirmed"
	p.ResultID = out.ID
	return t.PutPending(*p)
}

func undoProfile(t port.Transaction, a *domain.Audit, out *domain.Entry) error {
	current, e := t.Profile(a.AfterProfile.ID)
	if e != nil {
		if errors.Is(e, domain.ErrNotFound) {
			return domain.ErrConflict
		}
		return e
	}
	if current.Version != a.AfterProfile.Version {
		return domain.ErrConflict
	}
	if a.BeforeProfile == nil {
		e = t.DeleteProfile(current.ID)
	} else {
		restored := *a.BeforeProfile
		restored.Version = current.Version + 1
		e = t.PutProfile(restored)
	}
	if e != nil {
		return e
	}
	a.Undone = true
	if e = t.PutAudit(*a); e != nil {
		return e
	}
	*out, e = t.PutEntry(domain.Entry{Kind: "result", Text: "Undone: " + a.AfterProfile.Name + "."})
	return e
}

// profileName shows the profile name in previews and falls back to the id.
func profileName(t port.Transaction, id string) string {
	if id == "" {
		return ""
	}
	if p, e := t.Profile(id); e == nil {
		return p.Name
	}
	return id
}

// zeroEmpty shows an unset number as an empty value in previews.
func zeroEmpty(n int) string {
	if n == 0 {
		return ""
	}
	return fmt.Sprint(n)
}

// offsetsText names reminder offsets longest first: "1 day, 1 hour".
func offsetsText(offsets []int) string {
	parts := make([]string, 0, len(offsets))
	for _, m := range offsets {
		unit, n := "minute", m
		switch {
		case m%10080 == 0:
			unit, n = "week", m/10080
		case m%1440 == 0:
			unit, n = "day", m/1440
		case m%60 == 0:
			unit, n = "hour", m/60
		}
		if n != 1 {
			unit += "s"
		}
		parts = append(parts, fmt.Sprintf("%d %s", n, unit))
	}
	return strings.Join(parts, ", ")
}
