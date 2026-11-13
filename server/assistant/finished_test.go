package assistant

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/store"
)

func put(t *testing.T, s *store.Store, items ...domain.Item) {
	t.Helper()
	for _, i := range items {
		if e := s.Write(func(tx *store.Tx) error { return tx.PutItem(i) }); e != nil {
			t.Fatal(e)
		}
	}
}

func finishedFixture(t *testing.T) (*Service, *store.Store, *fakeModel) {
	a, s, f := newAssistant(t, Response{})
	put(t, s,
		domain.Item{ID: "open1", Title: "Call plumber", Status: "open", Version: 1, Timezone: "Europe/Amsterdam", DueAt: "2099-01-01T09:00:00Z"},
		domain.Item{ID: "done1", Title: "Dentist", Status: "done", Version: 3, Timezone: "Europe/Amsterdam", DueAt: "2026-11-01T09:00:00Z", Location: "Damrak 1", Notes: "- bring card"},
		domain.Item{ID: "gone1", Title: "Tax form", Status: "cancelled", Version: 2, Timezone: "Europe/Amsterdam"},
	)
	return a, s, f
}

func TestFinishedItemsStayOutOfTheFirstRequest(t *testing.T) {
	a, _, f := finishedFixture(t)
	f.r = Response{Text: "Hello.", CostUSD: 0.001}
	if _, e := a.Reply(context.Background(), "Hi"); e != nil {
		t.Fatal(e)
	}
	seen, _ := json.Marshal(f.history[0])
	if strings.Contains(string(seen), "Dentist") || f.calls != 1 {
		t.Fatalf("finished items reached an ordinary request (calls %d)", f.calls)
	}
}

func TestLookupOfFinishedItemsAnswersFromThem(t *testing.T) {
	a, _, f := finishedFixture(t)
	f.seq = []Response{
		{Calls: []Call{{Name: "find_finished_items", Arguments: `{"query":"dent"}`}}, CostUSD: 0.001},
		{Text: "The dentist was at Damrak 1.", CostUSD: 0.002},
	}
	entries, e := a.Reply(context.Background(), "Where was the dentist I finished?")
	if e != nil || len(entries) != 1 || entries[0].Text != "The dentist was at Damrak 1." {
		t.Fatalf("%+v %v", entries, e)
	}
	second, _ := json.Marshal(f.history[1])
	if f.calls != 2 || !strings.Contains(string(second), "Damrak 1") || strings.Contains(string(second), "Tax form") {
		t.Fatalf("second request must hold the matching finished item only (calls %d): %s", f.calls, second)
	}
	if cost, _ := a.repository.MonthCost(); cost < 0.0029 || cost > 0.0031 {
		t.Fatalf("both calls must be billed, got %v", cost)
	}
	if entries[0].CostEUR < 0.0029 || entries[0].CostEUR > 0.0031 {
		t.Fatalf("the reply carries the cost of both calls, got %v", entries[0].CostEUR)
	}
}

func TestAFinishedItemCanBeDeletedFromChat(t *testing.T) {
	a, _, f := finishedFixture(t)
	f.seq = []Response{
		{Calls: []Call{{Name: "find_finished_items", Arguments: `{"query":"dentist"}`}}, CostUSD: 0.001},
		{Calls: []Call{{Name: "propose_delete", Arguments: `{"item_id":"done1","expected_version":3}`}}, CostUSD: 0.001},
	}
	entries, e := a.Reply(context.Background(), "Delete the dentist I finished")
	if e != nil || len(entries) != 1 || entries[0].Change == nil || entries[0].Change.Action != "delete_item" || entries[0].Change.ItemTitle != "Dentist" {
		t.Fatalf("%+v %v", entries, e)
	}
}

func TestLookupFiltersByListAndCapsTheAnswer(t *testing.T) {
	a, s, f := finishedFixture(t)
	for i := 0; i < 60; i++ {
		put(t, s, domain.Item{ID: "x" + string(rune('A'+i%26)) + string(rune('a'+i/26)), Title: "Old " + string(rune('A'+i%26)), Status: "done", Version: 1, List: "Home"})
	}
	f.seq = []Response{
		{Calls: []Call{{Name: "find_finished_items", Arguments: `{"list":"home"}`}}, CostUSD: 0.001},
		{Text: "Ok.", CostUSD: 0.001},
	}
	if _, e := a.Reply(context.Background(), "What did I finish on Home?"); e != nil {
		t.Fatal(e)
	}
	second, _ := json.Marshal(f.history[1])
	if n := strings.Count(string(second), `status\":\"done\"`); n != maxFinished {
		t.Fatalf("lookup returned %d finished items, want the cap %d", n, maxFinished)
	}
	if strings.Contains(string(second), "Dentist") {
		t.Fatal("the list filter let another list through")
	}
}

func TestLookupOfNothingTellsTheModel(t *testing.T) {
	a, _, f := finishedFixture(t)
	f.seq = []Response{
		{Calls: []Call{{Name: "find_finished_items", Arguments: `{"query":"zzz"}`}}, CostUSD: 0.001},
		{Text: "I found no finished item like that.", CostUSD: 0.001},
	}
	entries, e := a.Reply(context.Background(), "Show my finished zzz")
	second, _ := json.Marshal(f.history[1])
	if e != nil || len(entries) != 1 || !strings.Contains(string(second), "No finished items match") {
		t.Fatalf("%+v %v %s", entries, e, second)
	}
}

func TestASecondLookupIsRefusedSoRepliesCannotLoop(t *testing.T) {
	a, _, f := finishedFixture(t)
	f.r = Response{Calls: []Call{{Name: "find_finished_items", Arguments: `{"query":"dent"}`}}, CostUSD: 0.001}
	if _, e := a.Reply(context.Background(), "Where was the dentist?"); e == nil {
		t.Fatal("a second lookup was accepted")
	}
	if f.calls != 2 {
		t.Fatalf("model called %d times, want exactly 2", f.calls)
	}
}

func TestAnOversizedLookupIsRefusedBeforeTheSecondCall(t *testing.T) {
	a, s, f := finishedFixture(t)
	big := strings.Repeat("n", 4000)
	for i := 0; i < 50; i++ {
		put(t, s, domain.Item{ID: "big" + string(rune('A'+i%26)) + string(rune('a'+i/26)), Title: "Big", Status: "done", Version: 1, Notes: big})
	}
	f.seq = []Response{{Calls: []Call{{Name: "find_finished_items", Arguments: `{"query":"big"}`}}, CostUSD: 0.001}}
	_, e := a.Reply(context.Background(), "Show my finished big ones")
	var invalid *domain.Invalid
	if !errors.As(e, &invalid) || f.calls != 1 {
		t.Fatalf("want an Invalid error before a second call, got %v (calls %d)", e, f.calls)
	}
}
