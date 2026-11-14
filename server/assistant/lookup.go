package assistant

import (
	"encoding/json"

	"github.com/alexanderp4580/Minder/server/domain"
)

// lookupQuery returns the query of the offer the owner just accepted. It needs a finder, the
// text Look it up, and an offer as the newest entry before that text. Failed attempts of the
// same answer (error entries and earlier Look it up messages) do not hide the offer, so Retry
// works. Anything the server wrote after the offer, such as a saved result, does. The model
// never decides to search before the owner taps.
func (a *Service) lookupQuery(history []domain.Entry, text string) (string, bool) {
	if a.finder == nil || !domain.IsLookupReply(text) {
		return "", false
	}
	for n := len(history); n > 0; n-- {
		entry := history[n-1]
		switch {
		case entry.Kind == "error", entry.Kind == "user" && domain.IsLookupReply(entry.Text):
			continue
		case entry.Kind == "minder":
			return domain.ParseOffer(entry.Text)
		default:
			return "", false
		}
	}
	return "", false
}

// offerLookup reports whether the model called offer_lookup, and its query.
func offerLookup(r Response) (string, bool, error) {
	for _, call := range r.Calls {
		if call.Name != "offer_lookup" {
			continue
		}
		var args struct {
			Query string `json:"query"`
		}
		if e := json.Unmarshal([]byte(call.Arguments), &args); e != nil {
			return "", false, &domain.Invalid{Field: "request", Message: "The request was unreadable. Try again."}
		}
		if _, ok := domain.OfferText(args.Query); !ok {
			return "", false, &domain.Invalid{Field: "request", Message: "The lookup had no place to search for. Try again."}
		}
		return args.Query, true, nil
	}
	return "", false, nil
}
