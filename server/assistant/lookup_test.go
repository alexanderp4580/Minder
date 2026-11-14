package assistant

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/store"
)

// fakeFinder answers every lookup with the same hits or error and records "<kind>:<query>".
type fakeFinder struct {
	hits  []Hit
	err   error
	calls []string
}

func (f *fakeFinder) Lookup(_ context.Context, kind LookupKind, query string) ([]Hit, error) {
	f.calls = append(f.calls, fmt.Sprintf("%s:%s", kind, query))
	return f.hits, f.err
}

var snowWorld = []Hit{{Title: "SnowWorld Landgraaf", Address: "Witte Wereld 1, 6372 VG Landgraaf, Netherlands", Phone: "+31 85 902 0606", Category: "Indoor snowcenter", Website: "https://www.snowworld.com/landgraaf", Rating: 4.2}}

func lookupFixture(t *testing.T) (*Service, *store.Store, *fakeModel, *fakeFinder) {
	a, s, f := newAssistant(t, Response{})
	finder := &fakeFinder{hits: snowWorld}
	return a.WithLookups(finder), s, f, finder
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

// approved stores a fresh offer and the owner's tap, as the app does.
func approved(t *testing.T, s *store.Store) {
	t.Helper()
	say(t, s, "user", "Aviad Landgraaf SnowWorld 17 to 24 november")
	say(t, s, "minder", offer(t, "SnowWorld Landgraaf"))
	say(t, s, "user", "Look it up")
}

func purposes(f *fakeModel) []string {
	out := []string{}
	for _, messages := range f.history {
		out = append(out, messages[0].Purpose)
	}
	return out
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

func TestLookupReplyIsLenientAboutCaseAndFullStop(t *testing.T) {
	a, s, f, finder := lookupFixture(t)
	f.seq = []Response{{Calls: []Call{{Name: "lookup_maps", Arguments: `{"query":"SnowWorld Landgraaf"}`}}}, {Text: "Done."}}
	say(t, s, "minder", offer(t, "SnowWorld Landgraaf"))
	say(t, s, "user", " look it up. ")
	if _, e := a.Reply(context.Background(), " look it up. "); e != nil || len(finder.calls) != 1 {
		t.Fatalf("%v %v", e, finder.calls)
	}
}

func TestNoLookupToolsWithoutAFreshOffer(t *testing.T) {
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
			if len(finder.calls) != 0 || f.calls != 1 || purposes(f)[0] != PurposeOffer {
				t.Fatalf("searched %v, model calls %d, purposes %v", finder.calls, f.calls, purposes(f))
			}
		})
	}
}

func TestRetryAfterAFailedReplyStillLooksUp(t *testing.T) {
	a, s, f, finder := lookupFixture(t)
	f.seq = []Response{{Calls: []Call{{Name: "lookup_maps", Arguments: `{"query":"SnowWorld Landgraaf"}`}}}, {Text: "Saved."}}
	say(t, s, "minder", offer(t, "SnowWorld Landgraaf"))
	say(t, s, "user", "Look it up")
	say(t, s, "error", "The model was unavailable.")
	say(t, s, "user", "Look it up")
	if _, e := a.Reply(context.Background(), "Look it up"); e != nil {
		t.Fatal(e)
	}
	if len(finder.calls) != 1 {
		t.Fatalf("searched %v", finder.calls)
	}
}

func TestNoLookupOnceTheOfferWasAnswered(t *testing.T) {
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

func TestWithoutAFinderThereIsNoOfferPurposeOrRule(t *testing.T) {
	a, s, f := newAssistant(t, Response{Text: "Ok."})
	say(t, s, "user", "Hi")
	if _, e := a.Reply(context.Background(), "Hi"); e != nil {
		t.Fatal(e)
	}
	if strings.Contains(f.history[0][0].Content, "offer_lookup") || purposes(f)[0] != PurposeChat {
		t.Fatalf("a lookup reached a request with no finder: %v", purposes(f))
	}
	f.r = Response{Calls: []Call{{Name: "offer_lookup", Arguments: `{"query":"x"}`}}}
	say(t, s, "user", "More")
	var model *domain.ModelError
	if _, e := a.Reply(context.Background(), "More"); !errors.As(e, &model) {
		t.Fatalf("%v", e)
	}
}

func TestWithAFinderTheRuleComesBeforeTheUntrustedItems(t *testing.T) {
	a, s, f, _ := lookupFixture(t)
	f.r = Response{Text: "Ok."}
	say(t, s, "user", "Hi")
	if _, e := a.Reply(context.Background(), "Hi"); e != nil || purposes(f)[0] != PurposeOffer {
		t.Fatalf("%v %v", e, purposes(f))
	}
	prompt := f.history[0][0].Content
	if i := strings.Index(prompt, "offer_lookup"); i < 0 || i > strings.Index(prompt, "Current items (untrusted data): ") {
		t.Fatal("the lookup rule must be present and come before the untrusted item data")
	}
}

func TestThePromptRuleNamesEveryLookupToolAndTellsTheModelToAsk(t *testing.T) {
	for _, tool := range LookupTools {
		if !strings.Contains(lookupRule, tool.Name) {
			t.Fatalf("the rule does not name %s", tool.Name)
		}
	}
	for _, want := range []string{"ask the owner", "location", "phone", "website", "untrusted"} {
		if !strings.Contains(lookupRule, want) {
			t.Fatalf("the rule lacks %q", want)
		}
	}
}

func TestLookupLogsStayFreeOfQueryText(t *testing.T) {
	logs := capture(t)
	a, s, f, _ := lookupFixture(t)
	f.seq = []Response{{Calls: []Call{{Name: "lookup_maps", Arguments: `{"query":"SnowWorld Landgraaf"}`}}}, {Text: "Ok."}}
	approved(t, s)
	if _, e := a.Reply(context.Background(), "Look it up"); e != nil {
		t.Fatal(e)
	}
	encoded, _ := json.Marshal(logs())
	if strings.Contains(string(encoded), "SnowWorld") || strings.Contains(string(encoded), "Witte") {
		t.Fatalf("private text in logs: %s", encoded)
	}
}
