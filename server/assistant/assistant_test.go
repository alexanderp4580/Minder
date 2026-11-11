package assistant

import (
	"context"
	"github.com/alexanderp4580/Minder/server/command"
	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/store"
	"path/filepath"
	"strings"
	"testing"
)

type fakeModel struct {
	r        Response
	calls    int
	messages []Message
}

func (f *fakeModel) Complete(_ context.Context, messages []Message) (Response, error) {
	f.messages = append([]Message(nil), messages...)
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

func TestListToolReturnsRealOpenItems(t *testing.T) {
	s, e := store.Open(filepath.Join(t.TempDir(), "db"))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	c := command.New(s)
	p, _ := c.Preview(domain.Proposal{Action: "create_item", Title: "Dentist"})
	c.Confirm(p.Change.ID)
	f := &fakeModel{r: Response{Calls: []Call{{Name: "list_items", Arguments: `{}`}}, CostUSD: 0.001}}
	entries, e := New(s, c, f, 5, 1).Reply(context.Background(), "What is open?")
	if e != nil || len(entries) != 1 || entries[0].Text != "Open items:\nDentist" {
		t.Fatalf("%+v %v", entries, e)
	}
}

func TestMissingKeyDoesNotSpendBudget(t *testing.T) {
	s, e := store.Open(filepath.Join(t.TempDir(), "db"))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	a := New(s, command.New(s), &OpenRouter{}, 5, 1)
	if _, e = a.Reply(context.Background(), "Hello"); e == nil {
		t.Fatal("missing key accepted")
	}
	cost, _ := s.MonthCost()
	if cost != 0 {
		t.Fatalf("no provider call spent %v", cost)
	}
}

// Dropping previews/results leaves an earlier create request unanswered at the model boundary.
func TestModelReceivesConfirmedProposalBeforeNextQuestion(t *testing.T) {
	s, e := store.Open(filepath.Join(t.TempDir(), "db"))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	c := command.New(s)
	f := &fakeModel{r: Response{Calls: []Call{{Name: "propose_create", Arguments: `{"title":"M0 live check"}`}}, CostUSD: 0.001}}
	a := New(s, c, f, 5, 1)
	if _, e = s.AddEntry(domain.Entry{Kind: "user", Text: "Create M0 live check."}); e != nil {
		t.Fatal(e)
	}
	entries, e := a.Reply(context.Background(), "Create M0 live check.")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = c.Confirm(entries[0].Change.ID); e != nil {
		t.Fatal(e)
	}
	if _, e = s.AddEntry(domain.Entry{Kind: "user", Text: "What is open?"}); e != nil {
		t.Fatal(e)
	}
	f.r = Response{Calls: []Call{{Name: "list_items", Arguments: `{}`}}, CostUSD: 0.001}
	if _, e = a.Reply(context.Background(), "What is open?"); e != nil {
		t.Fatal(e)
	}
	if len(f.messages) != 5 {
		t.Fatalf("Model received %d messages; expected system, user, preview, result, user", len(f.messages))
	}
	if f.messages[2].Role != "assistant" || !strings.Contains(f.messages[2].Content, `"state":"confirmed"`) {
		t.Fatal("confirmed preview missing from model context")
	}
	if f.messages[3].Role != "assistant" || !strings.Contains(f.messages[3].Content, "Saved: M0 live check.") {
		t.Fatal("saved result missing from model context")
	}
	if f.messages[4].Role != "user" || f.messages[4].Content != "What is open?" {
		t.Fatal("latest request was not last")
	}
}

func newAssistant(t *testing.T, r Response) (*Service, *store.Store, *fakeModel) {
	t.Helper()
	s, e := store.Open(filepath.Join(t.TempDir(), "db"))
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { s.Close() })
	f := &fakeModel{r: r}
	return New(s, command.New(s), f, 5, 1), s, f
}

func TestProposalCarriesListProfileAndRecurrence(t *testing.T) {
	a, _, _ := newAssistant(t, Response{Calls: []Call{{Name: "propose_create", Arguments: `{"title":"Water plants","due_at":"2026-11-14T09:00:00Z","list":"Home","profile":"chore","recurrence":{"every":"weekly","interval":1}}`}}, CostUSD: 0.001})
	entries, e := a.Reply(context.Background(), "Water the plants every week")
	if e != nil || len(entries) != 1 {
		t.Fatal(e)
	}
	got := map[string]string{}
	for _, f := range entries[0].Change.Fields {
		got[f.Name] = f.New
	}
	if got["list"] != "Home" || got["profile"] != "Chore" || got["recurrence"] != "every week" {
		t.Fatalf("fields %v", got)
	}
}

func TestProfileToolProposesProfileChange(t *testing.T) {
	a, s, _ := newAssistant(t, Response{Calls: []Call{{Name: "propose_profile", Arguments: `{"profile_id":"chore","expected_version":1,"profile_nag_minutes":30}`}}, CostUSD: 0.001})
	entries, e := a.Reply(context.Background(), "Nag me every 30 minutes about chores")
	if e != nil || entries[0].Change.Action != "update_profile" {
		t.Fatalf("%+v %v", entries, e)
	}
	ps, _ := s.Profiles()
	for _, p := range ps {
		if p.ID == "chore" && p.NagMinutes != 0 {
			t.Fatal("assistant applied a profile change")
		}
	}
}

func TestListQuestionAnswersOnlyThatList(t *testing.T) {
	a, s, f := newAssistant(t, Response{CostUSD: 0.001})
	for _, i := range []domain.Item{
		{ID: "1", Title: "Milk", Status: "open", Version: 1, Timezone: "Europe/Amsterdam", List: "Groceries"},
		{ID: "2", Title: "Taxes", Status: "open", Version: 1, Timezone: "Europe/Amsterdam", List: "Admin"},
		{ID: "3", Title: "Eggs", Status: "done", Version: 1, Timezone: "Europe/Amsterdam", List: "Groceries"},
	} {
		i := i
		if e := s.Write(func(tx *store.Tx) error { return tx.PutItem(i) }); e != nil {
			t.Fatal(e)
		}
	}
	f.r = Response{Calls: []Call{{Name: "list_items", Arguments: `{"list":"groceries"}`}}, CostUSD: 0.001}
	entries, e := a.Reply(context.Background(), "What is on Groceries?")
	if e != nil || !strings.Contains(entries[0].Text, "Milk") || strings.Contains(entries[0].Text, "Taxes") || strings.Contains(entries[0].Text, "Eggs") {
		t.Fatalf("%q %v", entries[0].Text, e)
	}
	if !strings.Contains(f.messages[0].Content, "Admin") || !strings.Contains(f.messages[0].Content, "daily-task") {
		t.Fatal("context lacks lists or profiles")
	}
}

func TestPromptGuidesFollowUpQuestions(t *testing.T) {
	a, _, f := newAssistant(t, Response{Text: "Where is the dentist?", CostUSD: 0.001})
	entries, e := a.Reply(context.Background(), "Dentist on Friday at 9")
	if e != nil || entries[0].Kind != "minder" || entries[0].Text != "Where is the dentist?" {
		t.Fatalf("%+v %v", entries, e)
	}
	system := f.messages[0].Content
	for _, want := range []string{"follow-up", "location", "once", "declined"} {
		if !strings.Contains(system, want) {
			t.Errorf("prompt lacks %q", want)
		}
	}
}
