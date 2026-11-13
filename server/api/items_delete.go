package api

import (
	"encoding/json"
	"net/http"

	"github.com/alexanderp4580/Minder/server/domain"
)

// deleteItem previews deleting an item for the calendar. Confirm and Discard use the change routes.
func (s *Server) deleteItem(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ExpectedVersion int `json:"expected_version"`
	}
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024))
	d.DisallowUnknownFields()
	if e := d.Decode(&body); e != nil || body.ExpectedVersion < 1 {
		fail(w, &domain.Invalid{Field: "item", Message: "Send the item version."})
		return
	}
	commands, ok := s.commands.(interface {
		Preview(domain.Proposal) (domain.Entry, error)
	})
	if !ok {
		http.NotFound(w, r)
		return
	}
	note(w, "item_id", r.PathValue("id"))
	entry, e := commands.Preview(domain.Proposal{Action: "delete_item", ItemID: r.PathValue("id"), ExpectedVersion: body.ExpectedVersion})
	if e != nil {
		fail(w, e)
		return
	}
	note(w, "change_id", entry.Change.ID, "outcome", "previewed")
	write(w, http.StatusOK, entry)
}
