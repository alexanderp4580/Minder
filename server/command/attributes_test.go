package command

import (
	"errors"
	"testing"
	"time"

	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/port"
	"github.com/alexanderp4580/Minder/server/reminder"
)

func create(t *testing.T, c *Service, p domain.Proposal) domain.Item {
	t.Helper()
	p.Action = "create_item"
	pv, e := c.Preview(p)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = c.Confirm(pv.Change.ID); e != nil {
		t.Fatal(e)
	}
	var item domain.Item
	if err := c.repository.Transact(func(tx port.Transaction) error {
		pending, err := tx.Pending(pv.Change.ID)
		item = pending.After
		return err
	}); err != nil {
		t.Fatal(err)
	}
	return item
}
func ptr(n int) *int { return &n }

func TestListNamesKeepOneSpelling(t *testing.T) {
	c, database := setup(t)
	create(t, c, domain.Proposal{Title: "Milk", List: "Groceries"})
	second := create(t, c, domain.Proposal{Title: "Eggs", List: "  groceries "})
	if second.List != "Groceries" {
		t.Fatalf("list %q", second.List)
	}
	third := create(t, c, domain.Proposal{Title: "Pasta"})
	pv, e := c.Preview(domain.Proposal{Action: "update_item", ItemID: third.ID, ExpectedVersion: third.Version, List: "GROCERIES"})
	if e != nil {
		t.Fatal(e)
	}
	c.Confirm(pv.Change.ID)
	lists, _ := database.Lists()
	if len(lists) != 1 || lists[0] != "Groceries" {
		t.Fatalf("lists %v", lists)
	}
	pv, _ = c.Preview(domain.Proposal{Action: "update_item", ItemID: third.ID, ExpectedVersion: third.Version + 1, ClearList: true})
	c.Confirm(pv.Change.ID)
	items, _ := c.repository.Items()
	for _, i := range items {
		if i.ID == third.ID && i.List != "" {
			t.Fatal("list not cleared")
		}
	}
}

func TestUnknownProfileIsRejected(t *testing.T) {
	c, _ := setup(t)
	_, e := c.Preview(domain.Proposal{Action: "create_item", Title: "x", DueAt: "2026-11-12T09:00:00Z", Profile: "nope"})
	var invalid *domain.Invalid
	if !errors.As(e, &invalid) || invalid.Field != "profile" {
		t.Fatalf("got %v", e)
	}
}

func TestProfileGivesReminderAfterConfirm(t *testing.T) {
	c, s := setup(t)
	due := time.Now().Add(3 * time.Hour).UTC()
	create(t, c, domain.Proposal{Title: "Bins", Profile: "day-event", DueAt: due.Format(time.RFC3339)})
	reminder.New(s).ScheduleDue(due.Add(-2 * time.Hour))
	if h, _ := s.History(0, 20); countReminders(h) != 0 {
		t.Fatal("fired before the profile time")
	}
	reminder.New(s).ScheduleDue(due.Add(-59 * time.Minute))
	if h, _ := s.History(0, 20); countReminders(h) != 1 {
		t.Fatal("profile reminder missing")
	}
}
func countReminders(h []domain.Entry) (n int) {
	for _, e := range h {
		if e.Reminder != nil {
			n++
		}
	}
	return
}

func TestCompletingRecurringItemMovesItForwardAndUndoRestores(t *testing.T) {
	c, s := setup(t)
	due := time.Now().Add(2 * time.Hour).UTC().Truncate(time.Second)
	remind := due.Add(-30 * time.Minute)
	i := create(t, c, domain.Proposal{Title: "Water plants", DueAt: due.Format(time.RFC3339), RemindAt: remind.Format(time.RFC3339), NagMinutes: 15, Recurrence: &domain.Recurrence{Every: "weekly", Interval: 1}})
	pv, e := c.Preview(domain.Proposal{Action: "complete_item", ItemID: i.ID, ExpectedVersion: i.Version})
	if e != nil {
		t.Fatal(e)
	}
	r, e := c.Confirm(pv.Change.ID)
	if e != nil {
		t.Fatal(e)
	}
	items, _ := s.Items()
	got := items[0]
	wantDue := due.AddDate(0, 0, 7)
	if got.Status != "open" || got.DueAt != wantDue.Format(time.RFC3339) || got.RemindAt != wantDue.Add(-30*time.Minute).Format(time.RFC3339) || got.CompletedAt == "" {
		t.Fatalf("after complete: %+v", got)
	}
	if _, e = c.Undo(r.Change.UndoID); e != nil {
		t.Fatal(e)
	}
	items, _ = s.Items()
	if items[0].DueAt != due.Format(time.RFC3339) || items[0].CompletedAt != "" {
		t.Fatalf("after undo: %+v", items[0])
	}
}

func TestProfileEditCreateConflictAndUndo(t *testing.T) {
	c, s := setup(t)
	pv, e := c.Preview(domain.Proposal{Action: "update_profile", ProfileID: "chore", ExpectedVersion: 1, ProfileNag: ptr(30)})
	if e != nil {
		t.Fatal(e)
	}
	if ps, _ := s.Profiles(); profile(ps, "chore").NagMinutes != 0 {
		t.Fatal("preview wrote the profile")
	}
	stale, _ := c.Preview(domain.Proposal{Action: "update_profile", ProfileID: "chore", ExpectedVersion: 1, ProfileRemindBefore: ptr(15)})
	r, e := c.Confirm(pv.Change.ID)
	if e != nil || r.Change.UndoID == "" {
		t.Fatal(e)
	}
	ps, _ := s.Profiles()
	if p := profile(ps, "chore"); p.NagMinutes != 30 || p.Version != 2 {
		t.Fatalf("%+v", p)
	}
	if _, e = c.Confirm(stale.Change.ID); !errors.Is(e, domain.ErrConflict) {
		t.Fatalf("stale profile preview: %v", e)
	}
	if _, e = c.Undo(r.Change.UndoID); e != nil {
		t.Fatal(e)
	}
	ps, _ = s.Profiles()
	if p := profile(ps, "chore"); p.NagMinutes != 0 {
		t.Fatalf("undo: %+v", p)
	}
	np, e := c.Preview(domain.Proposal{Action: "create_profile", ProfileName: "Painful", ProfileRemindBefore: ptr(120), ProfileNag: ptr(10)})
	if e != nil {
		t.Fatal(e)
	}
	r, _ = c.Confirm(np.Change.ID)
	ps, _ = s.Profiles()
	if p := profile(ps, "painful"); p.ID == "" || p.Builtin {
		t.Fatalf("new profile missing: %+v", ps)
	}
	if _, e = c.Undo(r.Change.UndoID); e != nil {
		t.Fatal(e)
	}
	ps, _ = s.Profiles()
	if len(ps) != 5 {
		t.Fatalf("undo of create left %d profiles", len(ps))
	}
}
func profile(ps []domain.Profile, id string) domain.Profile {
	for _, p := range ps {
		if p.ID == id || p.Name == id {
			return p
		}
	}
	return domain.Profile{}
}

func TestLocationIsStoredShownAndLimited(t *testing.T) {
	c, _ := setup(t)
	i := create(t, c, domain.Proposal{Title: "Dentist", DueAt: "2026-11-13T09:00:00Z", Location: "  Dr Smit, Damrak 1 "})
	if i.Location != "Dr Smit, Damrak 1" {
		t.Fatalf("location %q", i.Location)
	}
	pv, e := c.Preview(domain.Proposal{Action: "update_item", ItemID: i.ID, ExpectedVersion: i.Version, Location: "Clinic Noord"})
	if e != nil {
		t.Fatal(e)
	}
	found := false
	for _, f := range pv.Change.Fields {
		found = found || (f.Name == "location" && f.Old == "Dr Smit, Damrak 1" && f.New == "Clinic Noord")
	}
	if !found {
		t.Fatalf("fields %+v", pv.Change.Fields)
	}
	c.Confirm(pv.Change.ID)
	pv, _ = c.Preview(domain.Proposal{Action: "update_item", ItemID: i.ID, ExpectedVersion: i.Version + 1, ClearLocation: true})
	c.Confirm(pv.Change.ID)
	items, _ := c.repository.Items()
	if items[0].Location != "" {
		t.Fatal("location not cleared")
	}
	_, e = c.Preview(domain.Proposal{Action: "create_item", Title: "x", Location: string(make([]byte, 201))})
	var invalid *domain.Invalid
	if !errors.As(e, &invalid) || invalid.Field != "location" {
		t.Fatalf("long location: %v", e)
	}
}

func TestPreviewAllIsAllOrNothing(t *testing.T) {
	c, s := setup(t)
	_, e := c.PreviewAll([]domain.Proposal{{Action: "create_item", Title: "Fine"}, {Action: "create_item", Title: ""}})
	if e == nil {
		t.Fatal("invalid proposal accepted")
	}
	if h, _ := s.History(0, 20); len(h) != 0 {
		t.Fatalf("failed batch left %d entries", len(h))
	}
	got, e := c.PreviewAll([]domain.Proposal{{Action: "create_item", Title: "One"}, {Action: "create_item", Title: "Two"}})
	if e != nil || len(got) != 2 || got[0].Change.ID == got[1].Change.ID {
		t.Fatalf("%+v %v", got, e)
	}
}

func TestStartDurationAndPriorityRoundTrip(t *testing.T) {
	c, _ := setup(t)
	i := create(t, c, domain.Proposal{Title: "Dentist", StartAt: "2026-11-13T10:00:00+01:00", DurationMinutes: 30, Priority: 2})
	if i.StartAt != "2026-11-13T09:00:00Z" || i.DurationMinutes != 30 || i.Priority != 2 {
		t.Fatalf("%+v", i)
	}
	pv, e := c.Preview(domain.Proposal{Action: "update_item", ItemID: i.ID, ExpectedVersion: i.Version, DurationMinutes: 45, Priority: 1})
	if e != nil {
		t.Fatal(e)
	}
	got := map[string]string{}
	for _, f := range pv.Change.Fields {
		got[f.Name] = f.Old + ">" + f.New
	}
	if got["duration_minutes"] != "30>45" || got["priority"] != "2>1" {
		t.Fatalf("fields %v", got)
	}
	c.Confirm(pv.Change.ID)
	pv, _ = c.Preview(domain.Proposal{Action: "update_item", ItemID: i.ID, ExpectedVersion: i.Version + 1, ClearStart: true, ClearDuration: true, ClearPriority: true})
	c.Confirm(pv.Change.ID)
	items, _ := c.repository.Items()
	if x := items[0]; x.StartAt != "" || x.DurationMinutes != 0 || x.Priority != 0 {
		t.Fatalf("not cleared: %+v", x)
	}
	if _, e = c.Preview(domain.Proposal{Action: "create_item", Title: "x", StartAt: "2026-11-14T10:00:00Z", DueAt: "2026-11-13T10:00:00Z"}); e == nil {
		t.Fatal("start after deadline accepted")
	}
}

func TestEventProfileRemindsBeforeStart(t *testing.T) {
	c, s := setup(t)
	start := time.Now().Add(3 * time.Hour).UTC()
	create(t, c, domain.Proposal{Title: "Dentist", Profile: "day-event", StartAt: start.Format(time.RFC3339)})
	reminder.New(s).ScheduleDue(start.Add(-2 * time.Hour))
	if h, _ := s.History(0, 20); countReminders(h) != 0 {
		t.Fatal("fired early")
	}
	reminder.New(s).ScheduleDue(start.Add(-59 * time.Minute))
	if h, _ := s.History(0, 20); countReminders(h) != 1 {
		t.Fatal("start-based reminder missing")
	}
}
