package domain

import (
	"strings"
	"unicode"
)

// Words that name an online or phone venue, and filler words that may accompany them.
var (
	virtualVenueWords = map[string]bool{"teams": true, "zoom": true, "meet": true, "webex": true, "skype": true, "online": true, "video": true, "phone": true, "telephone": true, "virtual": true}
	virtualFillerWords = map[string]bool{"microsoft": true, "google": true, "via": true, "meeting": true, "call": true, "link": true, "conference": true, "remote": true}
)

// IsVirtualVenue reports whether a location names only an online or phone meeting, such as
// "Microsoft Teams meeting". A street address, or text that mixes a venue with an address, is not.
func IsVirtualVenue(location string) bool {
	words := strings.FieldsFunc(strings.ToLower(location), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
	venue := false
	for _, word := range words {
		switch {
		case virtualVenueWords[word]:
			venue = true
		case virtualFillerWords[word]:
		default:
			return false
		}
	}
	return venue
}
