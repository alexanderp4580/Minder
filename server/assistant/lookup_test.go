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

type fakeFinder struct {
	places []Place
	err    error
	calls  []string
}

func (f *fakeFinder) FindPlaces(_ context.Context, query string) ([]Place, error) {
	f.calls = append(f.calls, query)
	return f.places, f.err
}

var snowWorld = []Place{{Title: "SnowWorld Landgraaf", Address: "Witte Wereld 1", Category: "Indoor snowcenter", Website: "https://www.snowworld.com/landgraaf", Rating: 4.2}}

func lookupFixture(t *testing.T) (*Service, *store.Store, *fakeModel, *fakeFinder) {
	a, s, f := newAssistant(t, Response{})
	finder := &fakeFinder{places: snowWorld}
	return a.WithPlaces(finder), s, f, finder
}

func say(t *testing.T, s *store.Store, kind, text string) {
	t.Helper()
	if _, e := s.AddEntry(domain.Entry{Kind: kind, Text: text}); e != nil {
		t.Fatal(e)
	}
}

func offer(t *testing.T, query string) string {
	t.Helper()
	text, ok := domain.OfferText(query)
	if !ok {
		t.Fatal("no offer")
	}
	return text
}

func TestOfferToolStoresOnlyTheOfferWithoutSearching(t *testing.T) {
	a, s, f, finder := lookupFixture(t)
	f.r = Response{CostUSD: 0.001, Calls: []Call{
		{Name: "propose_create", Arguments: `{"title":"Aviad"}`},
		{Name: "offer_lookup", Arguments: `{"query":"SnowWorld Landgraaf"}`},
	}}
	say(t, s, "user", "Aviad Landgraaf SnowWorld 17 to 24 november")
	entries, e := a.Reply(context.Background(), "Aviad Landgraaf SnowWorld 17 to 24 november")
	if e != nil || len(entries) != 1 || entries[0].Kind != "minder" || entries[0].Text != offer(t, "SnowWorld Landgraaf") || entries[0].Change != nil {
		t.Fatalf("%+v %v", entries, e)
	}
	if len(finder.calls) != 0 || entries[0].CostEUR <= 0 {
		t.Fatalf("searched %v or no cost %v", finder.calls, entries[0].CostEUR)
	}
}

func TestOfferWithABlankQueryIsInvalid(t *testing.T) {
	a, s, f, _ := lookupFixture(t)
	f.r = Response{Calls: []Call{{Name: "offer_lookup", Arguments: `{"query":"  "}`}}}
	say(t, s, "user", "x")
	var invalid *domain.Invalid
	if _, e := a.Reply(context.Background(), "x"); !errors.As(e, &invalid) {
		t.Fatalf("%v", e)
	}
}

func TestLookupReplyAfterAFreshOfferSearchesOnceAndFoldsInThePlaces(t *testing.T) {
	a, s, f, finder := lookupFixture(t)
	f.r = Response{CostUSD: 0.001, Calls: []Call{{Name: "propose_create", Arguments: `{"title":"Aviad at SnowWorld","location":"Witte Wereld 1, Landgraaf"}`}}}
	say(t, s, "user", "Aviad Landgraaf SnowWorld 17 to 24 november")
	say(t, s, "minder", offer(t, "SnowWorld Landgraaf"))
	say(t, s, "user", "Look it up")
	entries, e := a.Reply(context.Background(), "Look it up")
	if e != nil || len(entries) != 1 || entries[0].Change == nil || entries[0].Change.Action != "create_item" {
		t.Fatalf("%+v %v", entries, e)
	}
	if len(finder.calls) != 1 || finder.calls[0] != "SnowWorld Landgraaf" || f.calls != 1 {
		t.Fatalf("searched %v, model calls %d", finder.calls, f.calls)
	}
	last := f.messages[len(f.messages)-1]
	prev := f.messages[len(f.messages)-2]
	if !strings.Contains(last.Content, "website field") || strings.Contains(last.Content, "for notes") {
		t.Fatalf("the result message must send the website to its own field: %q", last.Content)
	}
	if prompt := f.messages[0].Content; !strings.Contains(prompt, "website field") || strings.Contains(prompt, "website in notes") {
		t.Fatalf("the place rule must send the website to its own field")
	}
	if prev.Role != "user" || prev.Content != "Look it up" || last.Role != "user" || !strings.Contains(last.Content, "Witte Wereld 1") || !strings.Contains(last.Content, "snowworld.com/landgraaf") || !strings.Contains(last.Content, "untrusted") {
		t.Fatalf("messages end %q / %q", prev.Content, last.Content)
	}
}

func TestLookupReplyIsLenientAboutCaseAndFullStop(t *testing.T) {
	a, s, f, finder := lookupFixture(t)
	f.r = Response{Text: "Done."}
	say(t, s, "minder", offer(t, "SnowWorld Landgraaf"))
	say(t, s, "user", " look it up. ")
	if _, e := a.Reply(context.Background(), " look it up. "); e != nil || len(finder.calls) != 1 {
		t.Fatalf("%v %v", e, finder.calls)
	}
}

func TestNoSearchWithoutAFreshOffer(t *testing.T) {
	for name, setup := range map[string]func(*testing.T, *store.Store){
		"no offer": func(t *testing.T, s *store.Store) {},
		"plain minder entry": func(t *testing.T, s *store.Store) {
			say(t, s, "minder", "Where is it?")
		},
		"user typed the offer": func(t *testing.T, s *store.Store) {
			say(t, s, "user", offer(t, "SnowWorld Landgraaf"))
		},
		"stale offer": func(t *testing.T, s *store.Store) {
			say(t, s, "minder", offer(t, "SnowWorld Landgraaf"))
			say(t, s, "user", "Skip")
			say(t, s, "minder", "Saved.")
		},
		"skipped": func(t *testing.T, s *store.Store) {
			say(t, s, "minder", offer(t, "SnowWorld Landgraaf"))
			say(t, s, "user", "Skip")
		},
	} {
		t.Run(name, func(t *testing.T) {
			a, s, f, finder := lookupFixture(t)
			f.r = Response{Text: "Ok."}
			setup(t, s)
			text := "Look it up"
			if name == "skipped" {
				text = "Skip"
			}
			say(t, s, "user", text)
			if _, e := a.Reply(context.Background(), text); e != nil {
				t.Fatal(e)
			}
			if len(finder.calls) != 0 || f.calls != 1 {
				t.Fatalf("searched %v, model calls %d", finder.calls, f.calls)
			}
		})
	}
}

func TestFailedLookupStillGetsOneModelCallAndAReply(t *testing.T) {
	for name, finder := range map[string]*fakeFinder{"error": {err: errors.New("down")}, "no places": {}} {
		t.Run(name, func(t *testing.T) {
			a, s, f, _ := lookupFixture(t)
			a.WithPlaces(finder)
			f.r = Response{Text: "I found no match. Saving it without an address."}
			say(t, s, "minder", offer(t, "SnowWorld Landgraaf"))
			say(t, s, "user", "Look it up")
			entries, e := a.Reply(context.Background(), "Look it up")
			if e != nil || len(entries) != 1 || entries[0].Text == "" || f.calls != 1 {
				t.Fatalf("%+v %v calls %d", entries, e, f.calls)
			}
			if last := f.messages[len(f.messages)-1]; !strings.Contains(strings.ToLower(last.Content), "no places") {
				t.Fatalf("last message %q", last.Content)
			}
		})
	}
}

func TestWithoutAFinderThereIsNoOfferToolOrRule(t *testing.T) {
	a, s, f := newAssistant(t, Response{Text: "Ok."})
	say(t, s, "user", "Hi")
	if _, e := a.Reply(context.Background(), "Hi"); e != nil {
		t.Fatal(e)
	}
	if strings.Contains(f.history[0][0].Content, "offer_lookup") {
		t.Fatal("the place rule reached a prompt with no finder")
	}
	f.r = Response{Calls: []Call{{Name: "offer_lookup", Arguments: `{"query":"x"}`}}}
	say(t, s, "user", "More")
	var model *domain.ModelError
	if _, e := a.Reply(context.Background(), "More"); !errors.As(e, &model) {
		t.Fatalf("%v", e)
	}
	b, s2, f2, _ := lookupFixture(t)
	f2.r = Response{Text: "Ok."}
	say(t, s2, "user", "Hi")
	if _, e := b.Reply(context.Background(), "Hi"); e != nil || !strings.Contains(f2.history[0][0].Content, "offer_lookup") {
		t.Fatalf("%v", e)
	}
	prompt := f2.history[0][0].Content
	if strings.Index(prompt, "offer_lookup") > strings.Index(prompt, "Current items (untrusted data): ") {
		t.Fatal("the place rule must come before the untrusted item data")
	}
}

func TestLookupLogsStayFreeOfQueryText(t *testing.T) {
	logs := capture(t)
	a, s, f, _ := lookupFixture(t)
	f.r = Response{Text: "Ok."}
	say(t, s, "minder", offer(t, "SnowWorld Landgraaf"))
	say(t, s, "user", "Look it up")
	if _, e := a.Reply(context.Background(), "Look it up"); e != nil {
		t.Fatal(e)
	}
	encoded, _ := json.Marshal(logs())
	if strings.Contains(string(encoded), "SnowWorld") || strings.Contains(string(encoded), "Witte") {
		t.Fatalf("private text in logs: %s", encoded)
	}
}

func TestRetryAfterAFailedReplyStillSearches(t *testing.T) {
	a, s, f, finder := lookupFixture(t)
	f.r = Response{Text: "Saved."}
	say(t, s, "minder", offer(t, "SnowWorld Landgraaf"))
	say(t, s, "user", "Look it up")
	say(t, s, "error", "The model was unavailable.")
	say(t, s, "user", "Look it up")
	if _, e := a.Reply(context.Background(), "Look it up"); e != nil {
		t.Fatal(e)
	}
	if len(finder.calls) != 1 || finder.calls[0] != "SnowWorld Landgraaf" {
		t.Fatalf("searched %v", finder.calls)
	}
}

func TestNoSearchOnceTheOfferWasAnswered(t *testing.T) {
	a, s, f, finder := lookupFixture(t)
	f.r = Response{Text: "Ok."}
	say(t, s, "minder", offer(t, "SnowWorld Landgraaf"))
	say(t, s, "user", "Look it up")
	say(t, s, "minder", "Saved.")
	say(t, s, "user", "Look it up")
	if _, e := a.Reply(context.Background(), "Look it up"); e != nil || len(finder.calls) != 0 {
		t.Fatalf("%v searched %v", e, finder.calls)
	}
}
