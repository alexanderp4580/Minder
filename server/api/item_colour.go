package api

import (
	"encoding/json"
	"github.com/alexanderp4580/Minder/server/domain"
	"io"
	"net/http"
)

func colourBody(w http.ResponseWriter, r *http.Request, body any) error {
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024))
	d.DisallowUnknownFields()
	if err := d.Decode(body); err != nil {
		return &domain.Invalid{Field: "color", Message: "Send a valid colour request."}
	}
	if err := d.Decode(&struct{}{}); err != io.EOF {
		return &domain.Invalid{Field: "color", Message: "Send one colour request."}
	}
	return nil
}
func (s *Server) itemColor(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ExpectedVersion int    `json:"expected_version"`
		Color           string `json:"color"`
	}
	if err := colourBody(w, r, &body); err != nil {
		fail(w, err)
		return
	}
	if body.ExpectedVersion < 1 || !domain.ValidColor(body.Color) {
		fail(w, &domain.Invalid{Field: "color", Message: "Send the item version and a palette colour."})
		return
	}
	entry, err := s.commands.Preview(domain.Proposal{Action: "update_item", ItemID: r.PathValue("id"), ExpectedVersion: body.ExpectedVersion, Color: body.Color})
	if err != nil {
		fail(w, err)
		return
	}
	note(w, "item_id", r.PathValue("id"), "change_id", entry.Change.ID, "outcome", "previewed")
	write(w, http.StatusOK, entry)
}
func (s *Server) changeColor(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Color string `json:"color"`
	}
	if err := colourBody(w, r, &body); err != nil {
		fail(w, err)
		return
	}
	entry, err := s.commands.SetColor(r.PathValue("id"), body.Color)
	if err != nil {
		fail(w, err)
		return
	}
	note(w, "change_id", r.PathValue("id"), "outcome", "previewed")
	write(w, http.StatusOK, entry)
}
