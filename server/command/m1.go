package command

import (
	"fmt"
	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/store"
	"regexp"
	"strings"
	"time"
)

// setAttributes applies the list, profile and recurrence parts of a proposal to an item.
func setAttributes(t *store.Tx, p domain.Proposal, i *domain.Item) error {
	if p.ClearLocation {
		i.Location = ""
	}
	if p.Location != "" {
		i.Location = p.Location
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

// roll moves a recurring item to its next occurrence. A reminder set on the item keeps its distance to the due time.
func roll(i *domain.Item, now time.Time) error {
	next, e := domain.NextDue(*i, now)
	if e != nil {
		return e
	}
	if i.RemindAt != "" {
		old, _ := time.Parse(time.RFC3339, i.DueAt)
		remind, _ := time.Parse(time.RFC3339, i.RemindAt)
		nd, _ := time.Parse(time.RFC3339, next)
		i.RemindAt = nd.Add(-old.Sub(remind)).UTC().Format(time.RFC3339)
	}
	i.DueAt = next
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

func previewProfile(t *store.Tx, p domain.Proposal) (domain.Entry, error) {
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
	id := store.ID()
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

func confirmProfile(t *store.Tx, p *domain.Pending, out *domain.Entry) error {
	if p.BeforeProfile != nil {
		current, e := t.Profile(p.BeforeProfile.ID)
		if e != nil || current.Version != p.BeforeProfile.Version {
			return domain.ErrConflict
		}
	} else if _, e := t.Profile(p.AfterProfile.ID); e == nil {
		return domain.ErrConflict
	}
	if e := t.PutProfile(*p.AfterProfile); e != nil {
		return e
	}
	audit := domain.Audit{ID: store.ID(), BeforeProfile: p.BeforeProfile, AfterProfile: p.AfterProfile, Actor: "agent"}
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

func undoProfile(t *store.Tx, a *domain.Audit, out *domain.Entry) error {
	current, e := t.Profile(a.AfterProfile.ID)
	if e != nil || current.Version != a.AfterProfile.Version {
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
func profileName(t *store.Tx, id string) string {
	if id == "" {
		return ""
	}
	if p, e := t.Profile(id); e == nil {
		return p.Name
	}
	return id
}
