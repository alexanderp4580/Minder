package domain

import (
	"encoding/json"
	"strings"
)

// The place-lookup offer is a minder entry in a fixed format. The server recognises it by that
// format, so a lookup only ever runs for a query the owner saw.
const (
	OfferSkip      = "Skip"
	OfferLookup    = "Look it up"
	offerPrefix    = `Look up "`
	offerSuffix    = `" on Google for its address and website?`
	maxOfferQuery  = 100
	lookupReplyKey = "look it up"
)

// OfferText returns the offer for a query: whitespace collapsed, quotes turned into
// apostrophes, at most 100 runes. It reports false for a blank query.
func OfferText(query string) (string, bool) {
	query = strings.Join(strings.Fields(strings.ReplaceAll(query, `"`, "'")), " ")
	if runes := []rune(query); len(runes) > maxOfferQuery {
		query = strings.TrimSpace(string(runes[:maxOfferQuery]))
	}
	if query == "" {
		return "", false
	}
	return offerPrefix + query + offerSuffix, true
}

// ParseOffer returns the query of an offer text, or false for any other text.
func ParseOffer(text string) (string, bool) {
	if !strings.HasPrefix(text, offerPrefix) || !strings.HasSuffix(text, offerSuffix) || len(text) <= len(offerPrefix)+len(offerSuffix) {
		return "", false
	}
	query := text[len(offerPrefix) : len(text)-len(offerSuffix)]
	if rebuilt, ok := OfferText(query); !ok || rebuilt != text {
		return "", false
	}
	return query, true
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
