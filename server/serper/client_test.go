package serper

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexanderp4580/Minder/server/assistant"
)

// serperAt starts a fake Serper that answers every request with the body and records the last one.
func serperAt(t *testing.T, status int, body string) (*Client, *recorded) {
	t.Helper()
	seen := &recorded{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen.count++
		seen.path = r.URL.Path
		json.NewDecoder(r.Body).Decode(&seen.payload)
		w.WriteHeader(status)
		w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return &Client{Key: "serper-secret", URL: srv.URL}, seen
}

type recorded struct {
	count   int
	path    string
	payload map[string]string
}

func TestEachKindPostsToItsPathAndDecodes(t *testing.T) {
	cases := []struct {
		kind assistant.LookupKind
		path string
		body string
		want assistant.Hit
	}{
		{assistant.LookupMaps, "/maps", mapsBody, assistant.Hit{Title: "SnowWorld Landgraaf", Address: "Witte Wereld 1, 6372 VG Landgraaf, Netherlands", Category: "Indoor snowcenter", Website: "https://www.snowworld.com/landgraaf", Phone: "+31 85 902 0606", Rating: 4.2}},
		{assistant.LookupPlaces, "/places", `{"places":[{"title":"Rijksmuseum","address":"Museumstraat 1","category":"Art museum"}]}`, assistant.Hit{Title: "Rijksmuseum", Address: "Museumstraat 1", Category: "Art museum"}},
		{assistant.LookupSearch, "/search", `{"organic":[{"title":"SnowWorld","link":"https://www.snowworld.com","snippet":"Open daily"}]}`, assistant.Hit{Title: "SnowWorld", Website: "https://www.snowworld.com", Snippet: "Open daily"}},
	}
	for _, c := range cases {
		client, seen := serperAt(t, 200, c.body)
		hits, err := client.Lookup(context.Background(), c.kind, "SnowWorld Landgraaf")
		if err != nil || len(hits) == 0 || hits[0] != c.want {
			t.Fatalf("%s: %+v %v", c.kind, hits, err)
		}
		if seen.path != c.path || seen.payload["q"] != "SnowWorld Landgraaf" || seen.payload["gl"] != "nl" || seen.payload["hl"] != "en" {
			t.Fatalf("%s: %s %v", c.kind, seen.path, seen.payload)
		}
	}
}

func TestUnknownKindMakesNoRequest(t *testing.T) {
	client, seen := serperAt(t, 200, `{}`)
	if _, err := client.Lookup(context.Background(), "images", "x"); err == nil || seen.count != 0 {
		t.Fatalf("error %v requests %d", err, seen.count)
	}
}

func TestProviderFailureIsAnError(t *testing.T) {
	client, _ := serperAt(t, 500, `{"message":"down"}`)
	if _, err := client.Lookup(context.Background(), assistant.LookupMaps, "x"); err == nil {
		t.Fatal("a 500 was not an error")
	}
}

func TestTheDefaultBaseIsSerper(t *testing.T) {
	if got := (&Client{}).base(); got != "https://google.serper.dev" {
		t.Fatalf("base %s", got)
	}
}
