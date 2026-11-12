package command

import (
	"github.com/alexanderp4580/Minder/server/domain"
	"testing"
)

func TestEventPreviewConfirm(t *testing.T) {
	c, s := setup(t)
	pv, e := c.Preview(domain.Proposal{Action: "create_item", Title: "Hotel", Kind: "event", StartDate: "2026-11-01", EndDate: "2026-11-15"})
	if e != nil {
		t.Fatal(e)
	}
	before, _ := s.Items()
	if len(before) != 0 {
		t.Fatal("saved before confirm")
	}
	r, e := c.Confirm(pv.Change.ID)
	if e != nil {
		t.Fatal(e)
	}
	items, _ := s.Items()
	if len(items) != 1 || items[0].EndDate != "2026-11-15" {
		t.Fatalf("%+v", items)
	}
	if _, e = c.Undo(r.Change.UndoID); e != nil {
		t.Fatal(e)
	}
	items, _ = s.Items()
	if len(items) != 0 {
		t.Fatal("undo failed")
	}
}
