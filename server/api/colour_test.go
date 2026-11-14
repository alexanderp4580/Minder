package api

import (
	"github.com/alexanderp4580/Minder/server/command"
	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/store"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestColourRoutesKeepChangesPendingAndRejectStaleVersions(t *testing.T) {
	s, e := store.Open(filepath.Join(t.TempDir(), "db"))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	c := command.New(s)
	h := testServer(t, s, c, replier{s}, nil, "")
	post := func(url, body string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, url, strings.NewReader(body)))
		return w
	}
	p, _ := c.Preview(domain.Proposal{Action: "create_item", Title: "Workshop", Color: "blue"})
	route := "/v1/changes/" + p.Change.ID + "/color"
	if w := post(route, `{"color":"amber"}`); w.Code != 200 || !strings.Contains(w.Body.String(), `"color":"amber"`) {
		t.Fatal(w.Code, w.Body.String())
	}
	for _, body := range []string{`{"color":"red"}`, `{"color":"teal","extra":1}`, `{}`, `{"color":"blue"} {}`} {
		if w := post(route, body); w.Code != 400 {
			t.Fatal(body, w.Code)
		}
	}
	c.Confirm(p.Change.ID)
	items, _ := s.Items()
	item := items[0]
	route = "/v1/items/" + item.ID + "/color"
	w := post(route, `{"expected_version":1,"color":"green"}`)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	items, _ = s.Items()
	if items[0].Color != "amber" {
		t.Fatal("item preview wrote colour")
	}
	if w := post(route, `{"expected_version":2,"color":"green"}`); w.Code != 409 {
		t.Fatal("stale", w.Code)
	}
	if w := post("/v1/changes/"+p.Change.ID+"/color", `{"color":"teal"}`); w.Code != 409 {
		t.Fatal("resolved", w.Code)
	}
}
