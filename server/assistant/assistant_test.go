package assistant

import (
	"context"
	"github.com/alexanderp4580/Minder/server/command"
	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/store"
	"path/filepath"
	"testing"
)

type fakeModel struct {
	r     Response
	calls int
}

func (f *fakeModel) Complete(context.Context, []Message) (Response, error) {
	f.calls++
	return f.r, nil
}
func TestAssistantProposesWithoutApplying(t *testing.T) {
	s, e := store.Open(filepath.Join(t.TempDir(), "db"))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	f := &fakeModel{r: Response{Calls: []Call{{ID: "c", Name: "propose_create", Arguments: `{"title":"Call plumber","due_at":"2026-11-12T09:00:00Z"}`}}, CostUSD: 0.001}}
	a := New(s, command.New(s), f, 5, 1)
	entries, e := a.Reply(context.Background(), "Call the plumber")
	if e != nil || len(entries) != 1 || entries[0].Kind != "preview" {
		t.Fatalf("%+v %v", entries, e)
	}
	items, _ := s.Items()
	if len(items) != 0 {
		t.Fatal("model changed data")
	}
	cost, _ := s.MonthCost()
	if cost != 0.001 {
		t.Fatalf("usage %v", cost)
	}
}
func TestBudgetBlocksBeforeModel(t *testing.T) {
	s, _ := store.Open(filepath.Join(t.TempDir(), "db"))
	defer s.Close()
	s.RecordUsage(5, map[string]any{})
	f := &fakeModel{}
	a := New(s, command.New(s), f, 5, 1)
	_, e := a.Reply(context.Background(), "Hi")
	if e != domain.ErrLimit || f.calls != 0 {
		t.Fatal("budget allowed request", e)
	}
}
func TestClarificationChangesNothing(t *testing.T) {
	s, _ := store.Open(filepath.Join(t.TempDir(), "db"))
	defer s.Close()
	f := &fakeModel{r: Response{Text: "Which item do you mean?", CostUSD: 0.001}}
	a := New(s, command.New(s), f, 5, 1)
	entries, e := a.Reply(context.Background(), "Move it")
	if e != nil || entries[0].Text != "Which item do you mean?" {
		t.Fatal(entries, e)
	}
	items, _ := s.Items()
	if len(items) != 0 {
		t.Fatal("clarification changed items")
	}
}
