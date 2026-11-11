package store

import (
	"github.com/alexanderp4580/Minder/server/domain"
	"path/filepath"
	"testing"
	"time"
)

func TestBuiltInProfilesSeedOnceAndKeepEdits(t *testing.T) {
	path := filepath.Join(t.TempDir(), "db")
	s, e := Open(path)
	if e != nil {
		t.Fatal(e)
	}
	ps, _ := s.Profiles()
	names := map[string]bool{}
	for _, p := range ps {
		names[p.Name] = p.Builtin && p.Version == 1
	}
	for _, n := range []string{"Daily task", "Chore", "Birthday", "Day event", "Trip"} {
		if !names[n] {
			t.Errorf("missing built-in %q in %v", n, names)
		}
	}
	e = s.Write(func(tx *Tx) error {
		p, e := tx.Profile("chore")
		if e != nil {
			return e
		}
		p.NagMinutes, p.Version = 45, p.Version+1
		return tx.PutProfile(p)
	})
	if e != nil {
		t.Fatal(e)
	}
	s.Close()
	s, e = Open(path)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	ps, _ = s.Profiles()
	if len(ps) != 5 {
		t.Fatalf("reopen changed profile count: %d", len(ps))
	}
	for _, p := range ps {
		if p.ID == "chore" && p.NagMinutes != 45 {
			t.Fatal("reopen reset an edited built-in")
		}
	}
}

func profileItem(offsetDue time.Time) domain.Item {
	return domain.Item{ID: "item", Title: "Bins", Status: "open", Version: 1, Timezone: "Europe/Amsterdam", ProfileID: "daily-task", DueAt: offsetDue.Format(time.RFC3339)}
}
func entries(s *Store) int { h, _ := s.History(0, 50); return len(h) }

func TestProfileDerivesReminderAndEditsReschedule(t *testing.T) {
	s, now, _ := reminderStore(t)
	due := now.Add(2 * time.Hour)
	e := s.Write(func(tx *Tx) error {
		p, e := tx.Profile("daily-task")
		if e != nil {
			return e
		}
		p.RemindBeforeMinutes, p.NagMinutes, p.Version = 30, 15, p.Version+1
		if e = tx.PutProfile(p); e != nil {
			return e
		}
		return tx.PutItem(profileItem(due))
	})
	if e != nil {
		t.Fatal(e)
	}
	s.DueReminders(due.Add(-31 * time.Minute))
	if entries(s) != 0 {
		t.Fatal("reminder before the profile time")
	}
	e = s.Write(func(tx *Tx) error {
		p, _ := tx.Profile("daily-task")
		p.RemindBeforeMinutes, p.Version = 10, p.Version+1
		return tx.PutProfile(p)
	})
	if e != nil {
		t.Fatal(e)
	}
	s.DueReminders(due.Add(-20 * time.Minute))
	if entries(s) != 0 {
		t.Fatal("old profile time still fired")
	}
	s.DueReminders(due.Add(-9 * time.Minute))
	if entries(s) != 1 {
		t.Fatal("edited profile time did not fire")
	}
}

func TestItemWithoutProfileOrReminderStaysSilent(t *testing.T) {
	s, now, _ := reminderStore(t)
	i := profileItem(now.Add(time.Hour))
	i.ProfileID = ""
	putReminder(t, s, i)
	s.DueReminders(now.Add(24 * time.Hour))
	if entries(s) != 0 {
		t.Fatal("old due date created a surprise reminder")
	}
}

func TestListsAreDistinctNames(t *testing.T) {
	s, _, i := reminderStore(t)
	i.RemindAt, i.NagMinutes, i.List = "", 0, "Groceries"
	putReminder(t, s, i)
	i.ID, i.List = "two", "Groceries"
	putReminder(t, s, i)
	i.ID, i.List = "three", "Admin"
	putReminder(t, s, i)
	i.ID, i.List = "four", ""
	putReminder(t, s, i)
	got, e := s.Lists()
	if e != nil || len(got) != 2 || got[0] != "Admin" || got[1] != "Groceries" {
		t.Fatalf("%v %v", got, e)
	}
}
