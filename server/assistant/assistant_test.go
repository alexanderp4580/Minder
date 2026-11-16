package assistant

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alexanderp4580/Minder/server/command"
	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/store"
)

type fakeModel struct {
	r        Response
	seq      []Response // replies for successive calls; r answers once seq is empty
	calls    int
	messages []Message
	history  [][]Message
}

func (f *fakeModel) Complete(_ context.Context, messages []Message) (Response, error) {
	f.messages = append([]Message(nil), messages...)
	f.history = append(f.history, f.messages)
	f.calls++
	if len(f.seq) > 0 {
		r := f.seq[0]
		f.seq = f.seq[1:]
		return r, nil
	}
	return f.r, nil
}

// reply runs one owner message the way the worker does: save it, run the agent, commit.
func reply(t *testing.T, a *Service, s *store.Store, text string) ([]domain.Entry, error) {
	t.Helper()
	return replyIn(t, context.Background(), a, s, text)
}

// replyIn is reply under a given context, for tests that follow the trace ID.
func replyIn(t *testing.T, ctx context.Context, a *Service, s *store.Store, text string) ([]domain.Entry, error) {
	t.Helper()
	if _, _, err := s.EnqueueAgentTurn(domain.AgentTurn{ID: domain.NewID(), Kind: domain.TurnOwner}, &domain.Entry{Kind: "user", Text: text}); err != nil {
		t.Fatal(err)
	}
	turn, err := s.ClaimAgentTurn(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	outcome, err := a.Run(ctx, turn)
	if err != nil {
		s.ReleaseAgentTurn(turn.ID)
		return nil, err
	}
	entries, err := command.New(s).CommitAgentTurn(turn, outcome)
	if err != nil {
		s.ReleaseAgentTurn(turn.ID)
	}
	return entries, err
}

func TestAssistantProposesWithoutApplying(t *testing.T) {
	s, e := store.Open(filepath.Join(t.TempDir(), "db"))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	f := &fakeModel{r: Response{Calls: []Call{{ID: "c", Name: "propose_create", Arguments: `{"title":"Call plumber","due_at":"2026-11-12T09:00:00Z"}`}}, CostUSD: 0.001}}
	a := New(s, f, 5, 1)
	entries, e := reply(t, a, s, "Call the plumber")
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
	a := New(s, f, 5, 1)
	_, e := reply(t, a, s, "Hi")
	if e != domain.ErrLimit || f.calls != 0 {
		t.Fatal("budget allowed request", e)
	}
}
func TestClarificationChangesNothing(t *testing.T) {
	s, _ := store.Open(filepath.Join(t.TempDir(), "db"))
	defer s.Close()
	f := &fakeModel{r: Response{Text: "Which item do you mean?", CostUSD: 0.001}}
	a := New(s, f, 5, 1)
	entries, e := reply(t, a, s, "Move it")
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
	entries, e := reply(t, New(s, f, 5, 1), s, "What is open?")
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
	a := New(s, &unavailableModel{}, 5, 1)
	if _, e = reply(t, a, s, "Hello"); e == nil {
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
	a := New(s, f, 5, 1)
	entries, e := reply(t, a, s, "Create M0 live check.")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = c.Confirm(entries[0].Change.ID); e != nil {
		t.Fatal(e)
	}
	f.r = Response{Calls: []Call{{Name: "list_items", Arguments: `{}`}}, CostUSD: 0.001}
	if _, e = reply(t, a, s, "What is open?"); e != nil {
		t.Fatal(e)
	}
	if len(f.messages) != 6 {
		t.Fatalf("Model received %d messages; expected system, user, preview, result, owner context, user", len(f.messages))
	}
	if f.messages[2].Role != "assistant" || !strings.Contains(f.messages[2].Content, `"state":"confirmed"`) {
		t.Fatal("confirmed preview missing from model context")
	}
	if f.messages[3].Role != "assistant" || !strings.Contains(f.messages[3].Content, "Saved: M0 live check.") {
		t.Fatal("saved result missing from model context")
	}
	if !strings.Contains(f.messages[4].Content, "owner_message_source_id") {
		t.Fatal("the owner's message source ID is missing from the context")
	}
	if f.messages[5].Role != "user" || f.messages[5].Content != "What is open?" {
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
	return New(s, f, 5, 1), s, f
}

func TestProposalCarriesListProfileAndRecurrence(t *testing.T) {
	a, s, _ := newAssistant(t, Response{Calls: []Call{{Name: "propose_create", Arguments: `{"title":"Water plants","due_at":"2026-11-14T09:00:00Z","list":"Home","profile":"chore","recurrence":{"every":"weekly","interval":1}}`}}, CostUSD: 0.001})
	entries, e := reply(t, a, s, "Water the plants every week")
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
	entries, e := reply(t, a, s, "Nag me every 30 minutes about chores")
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
	entries, e := reply(t, a, s, "What is on Groceries?")
	if e != nil || !strings.Contains(entries[0].Text, "Milk") || strings.Contains(entries[0].Text, "Taxes") || strings.Contains(entries[0].Text, "Eggs") {
		t.Fatalf("%q %v", entries[0].Text, e)
	}
	if !strings.Contains(f.messages[0].Content, "Admin") || !strings.Contains(f.messages[0].Content, "daily-task") {
		t.Fatal("context lacks lists or profiles")
	}
}

func TestPromptGuidesFollowUpQuestions(t *testing.T) {
	a, s, f := newAssistant(t, Response{Text: "Where is the dentist?", CostUSD: 0.001})
	entries, e := reply(t, a, s, "Dentist on Friday at 9")
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

func TestSeveralChangesInOneMessageGiveSeveralPreviews(t *testing.T) {
	a, s, f := newAssistant(t, Response{CostUSD: 0.001})
	for _, id := range []string{"a", "b"} {
		i := domain.Item{ID: id, Title: "Item " + id, Status: "open", Version: 1, Timezone: "Europe/Amsterdam", DueAt: "2026-11-12T09:00:00Z"}
		if e := s.Write(func(tx *store.Tx) error { return tx.PutItem(i) }); e != nil {
			t.Fatal(e)
		}
	}
	f.r = Response{Calls: []Call{
		{Name: "propose_update", Arguments: `{"item_id":"a","expected_version":1,"due_at":"2026-11-14T09:00:00Z"}`},
		{Name: "propose_update", Arguments: `{"item_id":"b","expected_version":1,"due_at":"2026-11-15T09:00:00Z"}`},
	}, CostUSD: 0.001}
	entries, e := reply(t, a, s, "Move a to Saturday and b to Sunday")
	if e != nil || len(entries) != 2 || entries[0].Kind != "preview" || entries[1].Kind != "preview" {
		t.Fatalf("%+v %v", entries, e)
	}
	items, _ := s.Items()
	for _, i := range items {
		if i.DueAt != "2026-11-12T09:00:00Z" {
			t.Fatal("a preview changed data")
		}
	}
	calls := make([]Call, 6)
	for n := range calls {
		calls[n] = Call{Name: "propose_create", Arguments: `{"title":"x"}`}
	}
	f.r = Response{Calls: calls, CostUSD: 0.001}
	if _, e = reply(t, a, s, "Add six things"); e == nil {
		t.Fatal("six changes accepted")
	}
	h, _ := s.History(0, 50)
	previews := 0
	for _, en := range h {
		if en.Kind == "preview" {
			previews++
		}
	}
	if previews != 2 {
		t.Fatalf("a rejected batch left %d previews", previews)
	}
}

func TestReplyEntriesCarryTheRequestCost(t *testing.T) {
	s, _ := store.Open(filepath.Join(t.TempDir(), "db"))
	defer s.Close()
	f := &fakeModel{r: Response{Text: "Which item do you mean?", CostUSD: 0.002}}
	a := New(s, f, 5, 0.9)
	entries, e := reply(t, a, s, "Move it")
	if e != nil || len(entries) != 1 || entries[0].CostEUR < 0.0017999 || entries[0].CostEUR > 0.0018001 {
		t.Fatalf("%+v %v", entries, e)
	}
	f.r = Response{Calls: []Call{{Name: "propose_create", Arguments: `{"title":"A"}`}, {Name: "propose_create", Arguments: `{"title":"B"}`}}, CostUSD: 0.002}
	entries, e = reply(t, a, s, "Add A and B")
	if e != nil || len(entries) != 2 || entries[0].CostEUR == 0 || entries[1].CostEUR != 0 {
		t.Fatalf("cost must sit on the first entry only: %+v %v", entries, e)
	}
	h, _ := s.History(0, 50)
	saved := 0
	for _, en := range h {
		if en.Kind == "preview" && en.CostEUR > 0 {
			saved++
		}
	}
	if saved != 1 {
		t.Fatalf("stored previews with cost: %d", saved)
	}
}

func TestEventsUseStartAndTasksUseDeadline(t *testing.T) {
	a, s, f := newAssistant(t, Response{Calls: []Call{{Name: "propose_create", Arguments: `{"title":"Dentist","start_at":"2026-11-13T09:00:00Z","duration_minutes":30,"priority":2}`}}, CostUSD: 0.001})
	entries, e := reply(t, a, s, "Dentist on Friday at 10 for half an hour, important")
	if e != nil || len(entries) != 1 {
		t.Fatal(e)
	}
	got := map[string]string{}
	for _, fl := range entries[0].Change.Fields {
		got[fl.Name] = fl.New
	}
	shown, parseErr := time.Parse(time.RFC3339, got["start_at"])
	if parseErr != nil || !shown.Equal(time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)) || got["duration_minutes"] != "30" || got["priority"] != "2" {
		t.Fatalf("%v", got)
	}
	system := f.messages[0].Content
	for _, want := range []string{"start_at", "deadline", "duration_minutes", "priority"} {
		if !strings.Contains(system, want) {
			t.Errorf("prompt lacks %q", want)
		}
	}
}

func TestContextHoldsOnlyOpenItemsAndStaysBounded(t *testing.T) {
	a, s, f := newAssistant(t, Response{Text: "Which one?", CostUSD: 0.001})
	put := func(i domain.Item) {
		i.Version, i.Timezone = 1, "Europe/Amsterdam"
		if e := s.Write(func(tx *store.Tx) error { return tx.PutItem(i) }); e != nil {
			t.Fatal(e)
		}
	}
	put(domain.Item{ID: "done", Title: "Finished chore", Status: "done"})
	put(domain.Item{ID: "cancelled", Title: "Dropped idea", Status: "cancelled"})
	for n := 0; n < 250; n++ {
		put(domain.Item{ID: fmt.Sprintf("open%03d", n), Title: fmt.Sprintf("Open task %03d", n), Status: "open", UpdatedAt: fmt.Sprintf("2026-11-12T10:%02d:%02dZ", n/60, n%60)})
	}
	if _, e := reply(t, a, s, "Move it"); e != nil {
		t.Fatal(e)
	}
	system := f.messages[0].Content
	if strings.Contains(system, "Finished chore") || strings.Contains(system, "Dropped idea") {
		t.Fatal("finished items reached the model")
	}
	if got := strings.Count(system, `"status":"open"`); got != maxContextItems {
		t.Fatalf("context holds %d open items, want %d", got, maxContextItems)
	}
	if !strings.Contains(system, "Open task 249") || strings.Contains(system, "Open task 000") {
		t.Fatal("the context must keep the most recently updated items")
	}
}

type unavailableModel struct{ fakeModel }

func (*unavailableModel) Ready() error { return &domain.ModelError{Message: "No model key"} }

func TestCancelRequestProposesACancellationNotACompletion(t *testing.T) {
	a, s, f := newAssistant(t, Response{CostUSD: 0.001})
	i := domain.Item{ID: "a", Title: "Dentist", Status: "open", Version: 1, Timezone: "Europe/Amsterdam", DueAt: "2026-11-12T09:00:00Z"}
	if e := s.Write(func(tx *store.Tx) error { return tx.PutItem(i) }); e != nil {
		t.Fatal(e)
	}
	f.r = Response{Calls: []Call{{Name: "propose_cancel", Arguments: `{"item_id":"a","expected_version":1}`}}, CostUSD: 0.001}
	entries, e := reply(t, a, s, "Cancel the dentist")
	if e != nil || len(entries) != 1 || entries[0].Change == nil || entries[0].Change.Action != "cancel_item" {
		t.Fatalf("%+v %v", entries, e)
	}
}

func TestPromptSeparatesDoneCancelledAndDeleted(t *testing.T) {
	system := systemPrompt(time.Now(), nil, nil, nil, nil)
	for _, want := range []string{"propose_complete", "propose_cancel", "propose_delete", "find_finished_items"} {
		if !strings.Contains(system, want) {
			t.Fatalf("prompt does not mention %s", want)
		}
	}
}

func TestDeleteRequestProposesADeletion(t *testing.T) {
	a, s, f := newAssistant(t, Response{CostUSD: 0.001})
	i := domain.Item{ID: "a", Title: "Dentist", Status: "open", Version: 1, Timezone: "Europe/Amsterdam", DueAt: "2026-11-12T09:00:00Z"}
	if e := s.Write(func(tx *store.Tx) error { return tx.PutItem(i) }); e != nil {
		t.Fatal(e)
	}
	f.r = Response{Calls: []Call{{Name: "propose_delete", Arguments: `{"item_id":"a","expected_version":1}`}}, CostUSD: 0.001}
	entries, e := reply(t, a, s, "Delete the dentist, I added it by mistake")
	if e != nil || len(entries) != 1 || entries[0].Change == nil || entries[0].Change.Action != "delete_item" {
		t.Fatalf("%+v %v", entries, e)
	}
}

func TestDeletedItemLeavesTheAgentsContext(t *testing.T) {
	a, s, f := newAssistant(t, Response{CostUSD: 0.001})
	c := command.New(s)
	p, e := c.Preview(domain.Proposal{Action: "create_item", Title: "Dentist", DueAt: "2099-01-01T10:00:00Z", Timezone: "Europe/Amsterdam"})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = c.Confirm(p.Change.ID); e != nil {
		t.Fatal(e)
	}
	items, _ := s.Items()
	d, e := c.Preview(domain.Proposal{Action: "delete_item", ItemID: items[0].ID, ExpectedVersion: items[0].Version})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = c.Confirm(d.Change.ID); e != nil {
		t.Fatal(e)
	}
	f.r = Response{Text: "Done.", CostUSD: 0.001}
	if _, e = reply(t, a, s, "Add a dentist visit tomorrow"); e != nil {
		t.Fatal(e)
	}
	seen, _ := json.Marshal(f.messages)
	for _, old := range []string{"Saved: Dentist.", items[0].ID} {
		if strings.Contains(string(seen), old) {
			t.Fatalf("the model still sees %q of a deleted item", old)
		}
	}
}

func TestPromptExplainsTheBoardAndStatus(t *testing.T) {
	system := systemPrompt(time.Now(), nil, nil, nil, nil)
	for _, want := range []string{"Board", "do not invent a due date", "in_progress", "propose_update", "never from earlier messages"} {
		if !strings.Contains(system, want) {
			t.Fatalf("prompt lacks %q", want)
		}
	}
}

func TestHistoryShownToTheModelCarriesNoIdentifiers(t *testing.T) {
	entries := []domain.Entry{
		{ID: "ENTRYID", Cursor: 1, Kind: "preview", Text: "Review this change.", CostEUR: 0.002, Change: &domain.Change{ID: "CHANGEID", Action: "create_item", ItemTitle: "Call plumber", State: "pending", Fields: []domain.Field{{Name: "title", New: "Call plumber"}}}},
		{ID: "RESULTID", Cursor: 2, Kind: "result", Text: "Saved: Call plumber.", Change: &domain.Change{ID: "CHANGEID", Action: "create_item", ItemTitle: "Call plumber", State: "confirmed", UndoID: "UNDOID"}},
		{ID: "ERRID", Cursor: 3, Kind: "error", Text: "The item changed."},
	}
	joined := ""
	for _, m := range historyMessages(entries, domain.Entry{}) {
		joined += m.Content + "\n"
	}
	for _, leaked := range []string{"ENTRYID", "RESULTID", "ERRID", "CHANGEID", "UNDOID", "cursor", "cost_eur"} {
		if strings.Contains(joined, leaked) {
			t.Fatalf("history leaks %q:\n%s", leaked, joined)
		}
	}
	for _, kept := range []string{"Call plumber", "Saved: Call plumber.", "confirmed", "create_item"} {
		if !strings.Contains(joined, kept) {
			t.Fatalf("history lost %q:\n%s", kept, joined)
		}
	}
}

func TestTheStablePartOfTheSystemPromptHoldsNothingThatChangesPerCall(t *testing.T) {
	item := func(title string) []domain.Item {
		return []domain.Item{{ID: "i", Title: title, Status: "open", Version: 1}}
	}
	staticA, dynamicA := systemPromptParts(time.Date(2026, 11, 16, 9, 0, 0, 0, time.UTC), []string{"ZZLIST"}, nil, item("Call plumber"), nil)
	staticB, dynamicB := systemPromptParts(time.Date(2026, 11, 17, 18, 30, 0, 0, time.UTC), []string{"YYLIST"}, nil, item("Renew passport"), nil)
	if staticA != staticB {
		t.Fatal("the stable prefix changes with the time, lists or items, so a cache could never hit")
	}
	for _, changing := range []string{"Current local time", "Call plumber", "ZZLIST", "2026-11-16"} {
		if strings.Contains(staticA, changing) {
			t.Fatalf("stable prefix holds %q", changing)
		}
		if !strings.Contains(dynamicA, changing) {
			t.Fatalf("dynamic part lacks %q", changing)
		}
	}
	if dynamicA == dynamicB || staticA+dynamicA != systemPrompt(time.Date(2026, 11, 16, 9, 0, 0, 0, time.UTC), []string{"ZZLIST"}, nil, item("Call plumber"), nil) {
		t.Fatal("systemPrompt must equal the stable part followed by the dynamic part")
	}
}

func TestEveryRequestMarksItsCacheBreakpoints(t *testing.T) {
	a, s, f := newAssistant(t, Response{Text: "Hi.", CostUSD: 0.001})
	if _, e := reply(t, a, s, "Hello"); e != nil {
		t.Fatal(e)
	}
	system, last := f.messages[0], f.messages[len(f.messages)-1]
	if system.Role != "system" || system.CachePrefix <= 0 || system.CachePrefix >= len(system.Content) {
		t.Fatalf("system prefix %d of %d", system.CachePrefix, len(system.Content))
	}
	if strings.Contains(system.Content[:system.CachePrefix], "Current local time") {
		t.Fatal("the cached prefix includes the clock")
	}
	if !last.Cache || last.Role != "user" || last.Content != "Hello" {
		t.Fatalf("the owner's message is not a breakpoint: %+v", last)
	}
	for _, m := range f.messages[1 : len(f.messages)-1] {
		if m.Cache {
			t.Fatalf("an earlier message is marked: %+v", m)
		}
	}
}
