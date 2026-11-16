package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alexanderp4580/Minder/server/command"
	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/store"
)

func TestBoardRoutesAddMoveAndList(t *testing.T) {
	s, e := store.Open(filepath.Join(t.TempDir(), "db"))
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { s.Close() })
	c := command.New(s)
	h := testServer(t, s, c, replier{s}, nil, "")
	do := func(method, url, body string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(method, url, strings.NewReader(body)))
		return w
	}
	board := func() domain.Board {
		w := do(http.MethodGet, "/v1/board", "")
		if w.Code != 200 {
			t.Fatal(w.Code, w.Body.String())
		}
		var b domain.Board
		if e := json.Unmarshal(w.Body.Bytes(), &b); e != nil {
			t.Fatal(e)
		}
		return b
	}
	empty := do(http.MethodGet, "/v1/board", "")
	if empty.Body.String() != `{"open":[],"in_progress":[],"done":[]}`+"\n" && empty.Body.String() != `{"open":[],"in_progress":[],"done":[]}` {
		t.Fatal(empty.Body.String())
	}
	created := do(http.MethodPost, "/v1/items", `{"title":"Call plumber"}`)
	if created.Code != 201 {
		t.Fatal(created.Code, created.Body.String())
	}
	var item domain.Item
	json.Unmarshal(created.Body.Bytes(), &item)
	if item.Title != "Call plumber" || item.Status != "open" || len(board().Open) != 1 {
		t.Fatal(item, board())
	}
	for _, body := range []string{`{"title":""}`, `{"title":"x","extra":1}`, `{}`, `nope`, `{"title":"a"} {}`} {
		if w := do(http.MethodPost, "/v1/items", body); w.Code != 400 {
			t.Fatal(body, w.Code, w.Body.String())
		}
	}
	route := "/v1/items/" + item.ID + "/status"
	w := do(http.MethodPost, route, `{"expected_version":1,"status":"in_progress"}`)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"status":"in_progress"`) || !strings.Contains(w.Body.String(), `"version":2`) {
		t.Fatal(w.Code, w.Body.String())
	}
	if b := board(); len(b.Open) != 0 || len(b.InProgress) != 1 {
		t.Fatal(b)
	}
	if w = do(http.MethodPost, route, `{"expected_version":1,"status":"done"}`); w.Code != 409 {
		t.Fatal("stale", w.Code)
	}
	for _, body := range []string{`{"expected_version":2,"status":"cancelled"}`, `{"expected_version":2}`, `{"status":"done"}`, `{"expected_version":2,"status":"done","x":1}`} {
		if w = do(http.MethodPost, route, body); w.Code != 400 && w.Code != 409 {
			t.Fatal(body, w.Code)
		}
	}
	if w = do(http.MethodPost, "/v1/items/missing/status", `{"expected_version":1,"status":"done"}`); w.Code != 404 {
		t.Fatal("missing", w.Code)
	}
	if w = do(http.MethodPost, route, `{"expected_version":2,"status":"done"}`); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	if b := board(); len(b.Done) != 1 {
		t.Fatal(b)
	}
	entries, _ := s.History(0, 50)
	if len(entries) != 0 {
		t.Fatal("board routes wrote chat entries")
	}
	p, _ := c.Preview(domain.Proposal{Action: "create_item", Title: "Dentist", DueAt: "2026-12-01T10:00:00+01:00"})
	c.Confirm(p.Change.ID)
	items, _ := s.Items()
	for _, i := range items {
		if i.Title == "Dentist" {
			if w = do(http.MethodPost, "/v1/items/"+i.ID+"/status", `{"expected_version":1,"status":"done"}`); w.Code != 400 || !strings.Contains(w.Body.String(), "calendar") {
				t.Fatal("dated item moved", w.Code, w.Body.String())
			}
		}
	}
}

func TestBoardWritesRequireSameOrigin(t *testing.T) {
	s, e := store.Open(filepath.Join(t.TempDir(), "db"))
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { s.Close() })
	h := testServer(t, s, command.New(s), replier{s}, nil, "")
	r := httptest.NewRequest(http.MethodPost, "/v1/items", strings.NewReader(`{"title":"x"}`))
	r.Header.Set("Origin", "https://evil.example")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code == 201 || w.Code == 200 {
		t.Fatal("cross-origin write accepted", w.Code)
	}
}
