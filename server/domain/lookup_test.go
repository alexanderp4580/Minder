package domain

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestOfferRoundTrips(t *testing.T) {
	text, ok := OfferText("SnowWorld Landgraaf")
	if !ok || text != `Look up "SnowWorld Landgraaf" on Google for its address and website?` {
		t.Fatalf("offer text %q %v", text, ok)
	}
	query, ok := ParseOffer(text)
	if !ok || query != "SnowWorld Landgraaf" {
		t.Fatalf("parsed %q %v", query, ok)
	}
}

func TestOfferCleansTheQuery(t *testing.T) {
	text, ok := OfferText("  Snow\"World\n  Landgraaf ")
	if !ok {
		t.Fatal("refused a usable query")
	}
	query, ok := ParseOffer(text)
	if !ok || query != "Snow'World Landgraaf" {
		t.Fatalf("parsed %q %v", query, ok)
	}
	long, _ := OfferText(strings.Repeat("a", 101))
	query, _ = ParseOffer(long)
	if len([]rune(query)) != 100 {
		t.Fatalf("query kept %d runes", len([]rune(query)))
	}
	for _, blank := range []string{"", "   ", "\n \t"} {
		if _, ok := OfferText(blank); ok {
			t.Fatalf("offered %q", blank)
		}
	}
}

func TestParseOfferRejectsOtherText(t *testing.T) {
	for _, text := range []string{"Look up it", `Look up "" on Google for its address and website?`, `look up "x" on Google for its address and website?`, `Look up "x" on Google for its address and website? Reply yes`, ""} {
		if _, ok := ParseOffer(text); ok {
			t.Fatalf("accepted %q", text)
		}
	}
}

func TestIsLookupReply(t *testing.T) {
	for _, text := range []string{"Look it up", " look it up. ", "LOOK IT UP"} {
		if !IsLookupReply(text) {
			t.Fatalf("rejected %q", text)
		}
	}
	for _, text := range []string{"Skip", "look it up please", "", "look it up.."} {
		if IsLookupReply(text) {
			t.Fatalf("accepted %q", text)
		}
	}
}

func TestEntryServesOptionsOnlyForAnOffer(t *testing.T) {
	offer, _ := OfferText("SnowWorld Landgraaf")
	encoded := func(kind, text string) string {
		b, err := json.Marshal(Entry{ID: "e", Kind: kind, Text: text})
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	if got := encoded("minder", offer); !strings.Contains(got, `"options":["Skip","Look it up"]`) {
		t.Fatalf("offer entry %s", got)
	}
	for _, got := range []string{encoded("minder", "Hello"), encoded("user", offer)} {
		if strings.Contains(got, "options") {
			t.Fatalf("plain entry %s", got)
		}
	}
	var back Entry
	if err := json.Unmarshal([]byte(encoded("minder", offer)), &back); err != nil || back.Text != offer || back.Kind != "minder" {
		t.Fatalf("decode %v %+v", err, back)
	}
}
