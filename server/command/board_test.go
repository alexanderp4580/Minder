package command

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/port"
	"github.com/alexanderp4580/Minder/server/store"
)

func addTask(t *testing.T, c *Service, title string) domain.Item {
	t.Helper()
	item, err := c.AddTask(title)
	if err != nil {
		t.Fatal(err)
	}
	return item
}

func chatEntries(t *testing.T, s *store.Store) int {
	t.Helper()
	entries, err := s.History(0, 200)
	if err != nil {
		t.Fatal(err)
	}
	return len(entries)
}

func TestAddTaskCreatesAnOpenUndatedTaskWithoutChat(t *testing.T) {
	c, s := setup(t)
	item := addTask(t, c, "  Call plumber  ")
	if item.Title != "Call plumber" || item.Status != "open" || item.Kind != "task" || item.Version != 1 || item.Timezone == "" || !domain.OnBoard(item) {
		t.Fatalf("%+v", item)
	}
	stored, _ := s.Items()
	if len(stored) != 1 || stored[0].ID != item.ID {
		t.Fatal("task not stored")
	}
	if chatEntries(t, s) != 0 {
		t.Fatal("quick add wrote chat entries")
	}
	if n, _ := s.AuditCount(); n != 1 {
		t.Fatalf("audit rows = %d", n)
	}
	for _, title := range []string{"", "   ", strings.Repeat("x", 201)} {
		var invalid *domain.Invalid
		if _, err := c.AddTask(title); !errors.As(err, &invalid) {
			t.Fatalf("title %q accepted: %v", title, err)
		}
	}
}

func TestMoveTaskThroughEveryTransition(t *testing.T) {
	statuses := []string{"open", "in_progress", "done"}
	for _, from := range statuses {
		for _, to := range statuses {
			if from == to {
				continue
			}
			t.Run(from+" to "+to, func(t *testing.T) {
				c, s := setup(t)
				item := addTask(t, c, "Call plumber")
				if from != "open" {
					var err error
					if item, err = c.MoveTask(item.ID, item.Version, from); err != nil {
						t.Fatal(err)
					}
				}
				moved, err := c.MoveTask(item.ID, item.Version, to)
				if err != nil {
					t.Fatal(err)
				}
				if moved.Status != to || moved.Version != item.Version+1 {
					t.Fatalf("%+v", moved)
				}
				if (to == "done") != (moved.CompletedAt != "") {
					t.Fatalf("completed_at %q after moving to %s", moved.CompletedAt, to)
				}
				stored, _ := s.Items()
				if stored[0].Status != to || stored[0].Version != moved.Version {
					t.Fatal("move not stored")
				}
				if chatEntries(t, s) != 0 {
					t.Fatal("move wrote chat entries")
				}
			})
		}
	}
}

func TestMoveTaskRejectsStaleVersionUnknownStatusAndCalendarItems(t *testing.T) {
	c, s := setup(t)
	item := addTask(t, c, "Call plumber")
	if _, err := c.MoveTask(item.ID, item.Version+1, "done"); !errors.Is(err, domain.ErrConflict) {
		t.Fatal("stale version", err)
	}
	if _, err := c.MoveTask(item.ID, 0, "done"); !errors.Is(err, domain.ErrConflict) {
		t.Fatal("missing version", err)
	}
	var invalid *domain.Invalid
	for _, status := range []string{"", "cancelled", "blocked"} {
		if _, err := c.MoveTask(item.ID, item.Version, status); !errors.As(err, &invalid) {
			t.Fatalf("status %q accepted: %v", status, err)
		}
	}
	if _, err := c.MoveTask("missing", 1, "done"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatal("missing item", err)
	}
	p, _ := c.Preview(domain.Proposal{Action: "create_item", Title: "Dentist", DueAt: "2026-12-01T10:00:00+01:00"})
	c.Confirm(p.Change.ID)
	items, _ := s.Items()
	var dated domain.Item
	for _, i := range items {
		if i.Title == "Dentist" {
			dated = i
		}
	}
	if _, err := c.MoveTask(dated.ID, dated.Version, "done"); !errors.As(err, &invalid) || !strings.Contains(invalid.Message, "calendar") {
		t.Fatal("calendar item moved", err)
	}
}

func TestMoveTaskWritesAuditAndKeepsRepeatedMovesSafe(t *testing.T) {
	c, s := setup(t)
	item := addTask(t, c, "Call plumber")
	before, _ := s.AuditCount()
	moved, _ := c.MoveTask(item.ID, item.Version, "in_progress")
	if after, _ := s.AuditCount(); after != before+1 {
		t.Fatal("move wrote no audit row")
	}
	if _, err := c.MoveTask(item.ID, item.Version, "in_progress"); !errors.Is(err, domain.ErrConflict) {
		t.Fatal("repeat with the old version", err)
	}
	back, err := c.MoveTask(moved.ID, moved.Version, "open")
	if err != nil || back.Status != "open" || back.Version != 3 {
		t.Fatal(back, err)
	}
}

func TestMovingADoneTaskClosesItsReminderAndReopeningKeepsFutureOnly(t *testing.T) {
	c, s := setup(t)
	future := time.Now().Add(48 * time.Hour).UTC().Format(time.RFC3339)
	p, err := c.Preview(domain.Proposal{Action: "create_item", Title: "Call plumber", RemindAt: future, NagMinutes: 15})
	if err != nil {
		t.Fatal(err)
	}
	c.Confirm(p.Change.ID)
	items, _ := s.Items()
	item := items[0]
	active := func() int {
		st, err := s.ReminderStats(time.Now())
		if err != nil {
			t.Fatal(err)
		}
		return st.SchedulesActive
	}
	if active() != 1 {
		t.Fatal("reminder not scheduled")
	}
	done, err := c.MoveTask(item.ID, item.Version, "done")
	if err != nil {
		t.Fatal(err)
	}
	if active() != 0 {
		t.Fatal("done task still has an active reminder")
	}
	if _, err = c.MoveTask(done.ID, done.Version, "open"); err != nil {
		t.Fatal(err)
	}
	if active() != 1 {
		t.Fatal("reopened task lost its future reminder")
	}
}

func TestBoardMovesAreLoggedWithoutTitles(t *testing.T) {
	logs := capture(t)
	c, _ := setup(t)
	item := addTask(t, c, "PRIVATEPLUMBER")
	if _, err := c.MoveTask(item.ID, item.Version, "in_progress"); err != nil {
		t.Fatal(err)
	}
	records := logs()
	added, moved := first(records, "board item added"), first(records, "board item moved")
	if added == nil || moved == nil || moved["from"] != "open" || moved["to"] != "in_progress" || moved["item_id"] != item.ID {
		t.Fatal(added, moved)
	}
	for _, r := range records {
		if strings.Contains(strings.ToLower(toJSON(r)), "privateplumber") {
			t.Fatal("title logged", r)
		}
	}
}

func TestProposalStatusStartsAndReopensTasks(t *testing.T) {
	c, s := setup(t)
	p, err := c.Preview(domain.Proposal{Action: "create_item", Title: "Call plumber", Status: "in_progress"})
	if err != nil {
		t.Fatal(err)
	}
	c.Confirm(p.Change.ID)
	items, _ := s.Items()
	if items[0].Status != "in_progress" {
		t.Fatalf("created status %q", items[0].Status)
	}
	done, _ := c.Preview(domain.Proposal{Action: "complete_item", ItemID: items[0].ID, ExpectedVersion: items[0].Version})
	c.Confirm(done.Change.ID)
	items, _ = s.Items()
	re, err := c.Preview(domain.Proposal{Action: "update_item", ItemID: items[0].ID, ExpectedVersion: items[0].Version, Status: "open"})
	if err != nil {
		t.Fatal(err)
	}
	c.Confirm(re.Change.ID)
	items, _ = s.Items()
	if items[0].Status != "open" || items[0].CompletedAt != "" {
		t.Fatalf("%+v", items[0])
	}
	var invalid *domain.Invalid
	if _, err = c.Preview(domain.Proposal{Action: "update_item", ItemID: items[0].ID, ExpectedVersion: items[0].Version, Status: "done"}); !errors.As(err, &invalid) {
		t.Fatal("proposal status done accepted", err)
	}
	if _, err = c.Preview(domain.Proposal{Action: "create_item", Title: "x", Status: "cancelled"}); !errors.As(err, &invalid) {
		t.Fatal("proposal status cancelled accepted", err)
	}
}

func toJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func TestReopeningADoneTaskNeverRenotifiesForAPastReminder(t *testing.T) {
	c, s := setup(t)
	item := addTask(t, c, "Call plumber")
	past := time.Now().Add(-48 * time.Hour).UTC().Format(time.RFC3339)
	done, err := c.MoveTask(item.ID, item.Version, "done")
	if err != nil {
		t.Fatal(err)
	}
	// The stored task has a reminder that already passed while it was open.
	if err = s.Transact(func(tx port.Transaction) error {
		stored, e := tx.Item(done.ID)
		stored.RemindAt, stored.NagMinutes = past, 15
		stored.Version++
		done = stored
		return errors.Join(e, tx.PutItem(stored))
	}); err != nil {
		t.Fatal(err)
	}
	reopened, err := c.MoveTask(done.ID, done.Version, "open")
	if err != nil {
		t.Fatal(err)
	}
	if reopened.RemindAt != "" || reopened.NagMinutes != 0 {
		t.Fatalf("past reminder kept: %+v", reopened)
	}
	if st, _ := s.ReminderStats(time.Now()); st.SchedulesActive != 0 {
		t.Fatal("reopening scheduled a notification in the past")
	}
}

func TestAgentReopenClearsAPastReminderLikeTheBoard(t *testing.T) {
	c, s := setup(t)
	item := addTask(t, c, "Call plumber")
	past := time.Now().Add(-48 * time.Hour).UTC().Format(time.RFC3339)
	done, _ := c.MoveTask(item.ID, item.Version, "done")
	if err := s.Transact(func(tx port.Transaction) error {
		stored, e := tx.Item(done.ID)
		stored.RemindAt, stored.NagMinutes = past, 15
		stored.Version++
		done = stored
		return errors.Join(e, tx.PutItem(stored))
	}); err != nil {
		t.Fatal(err)
	}
	p, err := c.Preview(domain.Proposal{Action: "update_item", ItemID: done.ID, ExpectedVersion: done.Version, Status: "open"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = c.Confirm(p.Change.ID); err != nil {
		t.Fatal(err)
	}
	items, _ := s.Items()
	if items[0].RemindAt != "" || items[0].NagMinutes != 0 {
		t.Fatalf("past reminder kept: %+v", items[0])
	}
	if st, _ := s.ReminderStats(time.Now()); st.SchedulesActive != 0 {
		t.Fatal("agent reopen scheduled a notification in the past")
	}
}

func TestMoveTaskNamesTheRealReasonForARefusal(t *testing.T) {
	c, s := setup(t)
	cancelled := addTask(t, c, "Old job")
	p, _ := c.Preview(domain.Proposal{Action: "cancel_item", ItemID: cancelled.ID, ExpectedVersion: cancelled.Version})
	c.Confirm(p.Change.ID)
	items, _ := s.Items()
	var invalid *domain.Invalid
	if _, err := c.MoveTask(items[0].ID, items[0].Version, "done"); !errors.As(err, &invalid) || !strings.Contains(invalid.Message, "cancelled") || strings.Contains(invalid.Message, "calendar") {
		t.Fatal("cancelled task", err)
	}
}

func TestAPreviewSaysWhenAnEditMovesATaskBetweenBoardAndCalendar(t *testing.T) {
	c, s := setup(t)
	item := addTask(t, c, "Call plumber")
	shown := func(p domain.Proposal) *domain.Field {
		entry, err := c.Preview(p)
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range entry.Change.Fields {
			if f.Name == "shown_on" {
				return &f
			}
		}
		return nil
	}
	if f := shown(domain.Proposal{Action: "update_item", ItemID: item.ID, ExpectedVersion: item.Version, Title: "Call the plumber"}); f != nil {
		t.Fatalf("a title edit reported %+v", f)
	}
	f := shown(domain.Proposal{Action: "update_item", ItemID: item.ID, ExpectedVersion: item.Version, DueAt: "2026-12-01T10:00:00+01:00"})
	if f == nil || f.Old != "Board" || f.New != "Calendar" {
		t.Fatalf("dating a board task reported %+v", f)
	}
	p, _ := c.Preview(domain.Proposal{Action: "update_item", ItemID: item.ID, ExpectedVersion: item.Version, DueAt: "2026-12-01T10:00:00+01:00"})
	if _, err := c.Confirm(p.Change.ID); err != nil {
		t.Fatal(err)
	}
	items, _ := s.Items()
	back := shown(domain.Proposal{Action: "update_item", ItemID: item.ID, ExpectedVersion: items[0].Version, ClearDue: true})
	if back == nil || back.Old != "Calendar" || back.New != "Board" {
		t.Fatalf("clearing the date reported %+v", back)
	}
}
