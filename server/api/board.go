package api

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/alexanderp4580/Minder/server/domain"
)

// boardBody decodes one small JSON object and rejects unknown fields and trailing data.
func boardBody(w http.ResponseWriter, r *http.Request, field string, body any) error {
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
	d.DisallowUnknownFields()
	if err := d.Decode(body); err != nil {
		return &domain.Invalid{Field: field, Message: "Send a valid request."}
	}
	if err := d.Decode(&struct{}{}); err != io.EOF {
		return &domain.Invalid{Field: field, Message: "Send one request."}
	}
	return nil
}

func (s *Server) board(w http.ResponseWriter, r *http.Request) {
	b, err := s.queries.Board(time.Now())
	if err != nil {
		fail(w, err)
		return
	}
	note(w, "open", len(b.Open), "in_progress", len(b.InProgress), "done", len(b.Done))
	write(w, http.StatusOK, b)
}

func (s *Server) addTask(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Title string `json:"title"`
	}
	if err := boardBody(w, r, "title", &body); err != nil {
		fail(w, err)
		return
	}
	item, err := s.commands.AddTask(body.Title)
	if err != nil {
		fail(w, err)
		return
	}
	note(w, "item_id", item.ID)
	write(w, http.StatusCreated, item)
}

func (s *Server) moveTask(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ExpectedVersion int    `json:"expected_version"`
		Status          string `json:"status"`
	}
	if err := boardBody(w, r, "status", &body); err != nil {
		fail(w, err)
		return
	}
	item, err := s.commands.MoveTask(r.PathValue("id"), body.ExpectedVersion, body.Status)
	if err != nil {
		fail(w, err)
		return
	}
	note(w, "item_id", item.ID, "status", item.Status, "version", item.Version)
	write(w, http.StatusOK, item)
}
