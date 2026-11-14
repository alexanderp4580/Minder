package domain

import (
	"encoding/json"
	"strings"
)

// LookupKind names one way of searching the web for the owner.
type LookupKind string

const (
	LookupMaps   LookupKind = "maps"
	LookupPlaces LookupKind = "places"
	LookupSearch LookupKind = "search"
)

// Offer is a search the owner is asked to approve.
type Offer struct {
	Kind  LookupKind
	Query string
}

// The offer is a minder entry in a fixed format. The server recognises it by that format, so a
// search only ever runs for a kind and query the owner saw.
const (
	OfferSkip      = "Skip"
	OfferLookup    = "Look it up"
	offerQuote     = ` "`
	offerEnd       = `"?`
	maxOfferQuery  = 100
	lookupReplyKey = "look it up"
)

var offerLabels = map[LookupKind]string{
	LookupMaps:   "Search Google Maps for",
	LookupPlaces: "Search Google Places for",
	LookupSearch: "Search the web for",
}

// OfferText returns the offer text for a kind and query: whitespace collapsed, quotes turned
// into apostrophes, at most 100 runes. It reports false for an unknown kind or a blank query.
func OfferText(offer Offer) (string, bool) {
	label, known := offerLabels[offer.Kind]
	query := strings.Join(strings.Fields(strings.ReplaceAll(offer.Query, `"`, "'")), " ")
	if runes := []rune(query); len(runes) > maxOfferQuery {
		query = strings.TrimSpace(string(runes[:maxOfferQuery]))
	}
	if !known || query == "" {
		return "", false
	}
	return label + offerQuote + query + offerEnd, true
}

// ParseOffer returns the offer an offer text holds, or false for any other text.
func ParseOffer(text string) (Offer, bool) {
	for kind, label := range offerLabels {
		prefix := label + offerQuote
		if !strings.HasPrefix(text, prefix) || !strings.HasSuffix(text, offerEnd) || len(text) <= len(prefix)+len(offerEnd) {
			continue
		}
		offer := Offer{kind, text[len(prefix) : len(text)-len(offerEnd)]}
		if rebuilt, ok := OfferText(offer); ok && rebuilt == text {
			return offer, true
		}
	}
	return Offer{}, false
}

// IsLookupReply reports whether the owner's text accepts an offer: Look it up, ignoring case,
// surrounding space and one trailing full stop.
func IsLookupReply(text string) bool {
	text = strings.TrimSpace(text)
	text = strings.TrimSuffix(text, ".")
	return strings.EqualFold(strings.TrimSpace(text), lookupReplyKey)
}

// MarshalJSON adds the computed quick replies of an offer. Nothing stores them.
func (e Entry) MarshalJSON() ([]byte, error) {
	type plain Entry
	out := struct {
		plain
		Options []string `json:"options,omitempty"`
	}{plain: plain(e)}
	if _, ok := ParseOffer(e.Text); ok && e.Kind == "minder" {
		out.Options = []string{OfferSkip, OfferLookup}
	}
	return json.Marshal(out)
}
