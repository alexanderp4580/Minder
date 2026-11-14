package serper

import (
	"testing"

	"github.com/alexanderp4580/Minder/server/assistant"
)

const mapsBody = `{"places":[
{"title":"SnowWorld Landgraaf","address":"Witte Wereld 1, 6372 VG Landgraaf, Netherlands","type":"Indoor snowcenter","website":"https://www.snowworld.com/landgraaf","phoneNumber":"+31 85 902 0606","rating":4.2,"placeId":"x"},
{"title":"B"},{"title":"C"},{"title":"D"}],"credits":3}`

func TestDecodeVenuesReadsAMapsPlace(t *testing.T) {
	hits, err := decodeVenues([]byte(mapsBody))
	if err != nil || len(hits) != 3 {
		t.Fatalf("%d %v", len(hits), err)
	}
	want := assistant.Hit{Title: "SnowWorld Landgraaf", Address: "Witte Wereld 1, 6372 VG Landgraaf, Netherlands", Category: "Indoor snowcenter", Website: "https://www.snowworld.com/landgraaf", Phone: "+31 85 902 0606", Rating: 4.2}
	if hits[0] != want {
		t.Fatalf("%+v", hits[0])
	}
}

func TestDecodeVenuesReadsAPlacesCategory(t *testing.T) {
	hits, err := decodeVenues([]byte(`{"places":[{"title":"Rijksmuseum","address":"Museumstraat 1","category":"Art museum"}]}`))
	if err != nil || len(hits) != 1 || hits[0].Category != "Art museum" || hits[0].Address != "Museumstraat 1" {
		t.Fatalf("%+v %v", hits, err)
	}
}

func TestDecodeWebMapsLinkAndSnippet(t *testing.T) {
	hits, err := decodeWeb([]byte(`{"organic":[{"title":"SnowWorld","link":"https://www.snowworld.com/en/landgraaf","snippet":"Opening hours 09:00 - 22:00"},{"title":"B"},{"title":"C"},{"title":"D"}]}`))
	if err != nil || len(hits) != 3 {
		t.Fatalf("%d %v", len(hits), err)
	}
	if hits[0] != (assistant.Hit{Title: "SnowWorld", Website: "https://www.snowworld.com/en/landgraaf", Snippet: "Opening hours 09:00 - 22:00"}) {
		t.Fatalf("%+v", hits[0])
	}
}

func TestDecodersReturnAnEmptySliceForNoResults(t *testing.T) {
	for name, decode := range map[string]func([]byte) ([]assistant.Hit, error){"venues": decodeVenues, "web": decodeWeb} {
		hits, err := decode([]byte(`{}`))
		if err != nil || hits == nil || len(hits) != 0 {
			t.Fatalf("%s: %v %v", name, hits, err)
		}
	}
}

func TestDecodersRejectInvalidJSON(t *testing.T) {
	for name, decode := range map[string]func([]byte) ([]assistant.Hit, error){"venues": decodeVenues, "web": decodeWeb} {
		if _, err := decode([]byte(`not json`)); err == nil {
			t.Fatalf("%s accepted invalid JSON", name)
		}
	}
}
