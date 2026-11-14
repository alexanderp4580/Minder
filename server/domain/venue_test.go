package domain_test

import (
	"testing"

	"github.com/alexanderp4580/Minder/server/domain"
)

// VirtualVenueCases is mirrored by app/test/domain/virtual_venue_test.dart. Keep them identical.
var virtualVenueCases = map[string]bool{
	"Teams": true, "Microsoft Teams": true, "Microsoft Teams meeting": true, "Via Teams": true, "teams": true,
	"Zoom": true, "Zoom meeting link": true, "Google Meet": true, "Webex": true, "Skype": true,
	"Online": true, "Online meeting": true, "Video call": true, "Phone call": true, "Telephone": true, "Virtual": true,
	"Teamsstraat 4, Utrecht": false, "Zoomlaan 1": false, "Kerkstraat 1": false, "Meeting room Teams-hoek 2": false,
	"Teams or in person": false, "Witte Wereld 1, 6372 VG Landgraaf": false, "": false, "   ": false,
}

func TestIsVirtualVenue(t *testing.T) {
	for location, want := range virtualVenueCases {
		if got := domain.IsVirtualVenue(location); got != want {
			t.Fatalf("%q: %v, want %v", location, got, want)
		}
	}
}
