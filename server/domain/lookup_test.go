package domain

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestOfferRoundTripsForEveryKind(t *testing.T) {
	for kind, want := range map[LookupKind]string{
		LookupMaps:   `Search Google Maps for "SnowWorld Landgraaf"?`,
		LookupPlaces: `Search Google Places for "SnowWorld Landgraaf"?`,
		LookupSearch: `Search the web for "SnowWorld Landgraaf"?`,
	} {
		text, ok := OfferText(Offer{kind, "SnowWorld Landgraaf"})
		if !ok || text != want {
			t.Fatalf("%s: %q %v", kind, text, ok)
		}
		offer, ok := ParseOffer(text)
		if !ok || offer != (Offer{kind, "SnowWorld Landgraaf"}) {
			t.Fatalf("%s: parsed %+v %v", kind, offer, ok)
		}
	}
}

func TestOfferRefusesAnUnknownKindOrABlankQuery(t *testing.T) {
	for _, offer := range []Offer{{"images", "x"}, {"", "x"}, {LookupMaps, ""}, {LookupMaps, "   "}, {LookupMaps, "\n \t"}} {
		if text, ok := OfferText(offer); ok {
			t.Fatalf("offered %+v as %q", offer, text)
		}
	}
}

func TestOfferCleansTheQuery(t *testing.T) {
	text, ok := OfferText(Offer{LookupSearch, "  Snow\"World\n  Landgraaf "})
	if !ok {
		t.Fatal("refused a usable query")
	}
	offer, ok := ParseOffer(text)
	if !ok || offer.Query != "Snow'World Landgraaf" {
		t.Fatalf("parsed %+v %v", offer, ok)
	}
	long, _ := OfferText(Offer{LookupMaps, strings.Repeat("a", 101)})
	offer, _ = ParseOffer(long)
	if len([]rune(offer.Query)) != 100 {
		t.Fatalf("query kept %d runes", len([]rune(offer.Query)))
	}
}

func TestParseOfferRejectsOtherText(t *testing.T) {
	for _, text := range []string{
		`Look up "SnowWorld" on Google for its address and website?`, // the old format
		`Search Google Maps for ""?`,
		`search Google Maps for "x"?`,
		`Search Google Maps for "x"? Reply yes`,
		`Search Google Maps for "x?`,
		`Search Google Images for "x"?`,
		`Search the web "x"?`,
		"",
	} {
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
	offer, _ := OfferText(Offer{LookupMaps, "SnowWorld Landgraaf"})
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
