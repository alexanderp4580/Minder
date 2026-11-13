package serper

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const fixture = `{"searchParameters":{"q":"x"},"places":[
{"title":"SnowWorld Landgraaf","address":"Witte Wereld 1","category":"Indoor snowcenter","website":"https://www.snowworld.com/landgraaf","rating":4.2,"latitude":50.8},
{"title":"B","address":"Street 2"},{"title":"C"},{"title":"D"}],"credits":1}`

func server(t *testing.T, status int, body string, seen *http.Request, payload *map[string]string) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if seen != nil {
			*seen = *r.Clone(context.Background())
		}
		if payload != nil {
			json.NewDecoder(r.Body).Decode(payload)
		}
		w.WriteHeader(status)
		w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return &Client{Key: "serper-secret", URL: srv.URL}
}

func TestFindPlacesSendsTheQueryAndKeepsThreePlaces(t *testing.T) {
	var seen http.Request
	var payload map[string]string
	c := server(t, 200, fixture, &seen, &payload)
	places, err := c.FindPlaces(context.Background(), "SnowWorld Landgraaf")
	if err != nil {
		t.Fatal(err)
	}
	if seen.Method != "POST" || seen.Header.Get("X-API-KEY") != "serper-secret" {
		t.Fatalf("request %s %v", seen.Method, seen.Header)
	}
	if payload["q"] != "SnowWorld Landgraaf" || payload["gl"] != "nl" || payload["hl"] != "en" {
		t.Fatalf("payload %v", payload)
	}
	if len(places) != 3 {
		t.Fatalf("places %d", len(places))
	}
	first := places[0]
	if first.Title != "SnowWorld Landgraaf" || first.Address != "Witte Wereld 1" || first.Category != "Indoor snowcenter" || first.Website != "https://www.snowworld.com/landgraaf" || first.Rating != 4.2 {
		t.Fatalf("first %+v", first)
	}
}

func TestFindPlacesWithNoPlacesIsEmpty(t *testing.T) {
	c := server(t, 200, `{"places":[]}`, nil, nil)
	places, err := c.FindPlaces(context.Background(), "nothing")
	if err != nil || len(places) != 0 {
		t.Fatalf("%v %v", places, err)
	}
}

func TestFindPlacesErrorsNeverCarryTheKeyOrBody(t *testing.T) {
	for _, status := range []int{401, 500} {
		c := server(t, status, `{"message":"bad key serper-secret"}`, nil, nil)
		_, err := c.FindPlaces(context.Background(), "x")
		if err == nil || strings.Contains(err.Error(), "serper-secret") || strings.Contains(err.Error(), "bad key") {
			t.Fatalf("status %d error %v", status, err)
		}
	}
}

func TestFindPlacesRejectsABodyOverTheLimit(t *testing.T) {
	c := server(t, 200, `{"places":[{"title":"`+strings.Repeat("a", 70<<10)+`"}]}`, nil, nil)
	if _, err := c.FindPlaces(context.Background(), "x"); err == nil {
		t.Fatal("accepted an oversized body")
	}
}

func TestFindPlacesWithoutAKeyMakesNoRequest(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	t.Cleanup(srv.Close)
	if _, err := (&Client{URL: srv.URL}).FindPlaces(context.Background(), "x"); err == nil || called {
		t.Fatalf("error %v called %v", err, called)
	}
}

func TestFindPlacesStopsWhenTheContextEnds(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)
		<-r.Context().Done()
	}))
	t.Cleanup(srv.Close)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	started := time.Now()
	if _, err := (&Client{Key: "k", URL: srv.URL}).FindPlaces(ctx, "x"); err == nil || time.Since(started) > 2*time.Second {
		t.Fatalf("error %v after %v", err, time.Since(started))
	}
}
