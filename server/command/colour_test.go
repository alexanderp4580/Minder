package command

import (
	"errors"
	"github.com/alexanderp4580/Minder/server/domain"
	"testing"
)

func TestColourPreviewConfirmDiscardUndoAndConflict(t *testing.T) {
	c, s := setup(t)
	created, err := c.Preview(domain.Proposal{Action: "create_item", Title: "Party", Color: "lilac", Location: "Home", LocationKind: "label"})
	if err != nil {
		t.Fatal(err)
	}
	if created.Change.Color != "lilac" {
		t.Fatal(created.Change)
	}
	if _, err = c.SetColor(created.Change.ID, "red"); err == nil {
		t.Fatal("accepted non-palette colour")
	}
	edited, err := c.SetColor(created.Change.ID, "teal")
	if err != nil || edited.Change.Color != "teal" {
		t.Fatal(edited, err)
	}
	items, _ := s.Items()
	if len(items) != 0 {
		t.Fatal("picker saved before Confirm")
	}
	c.Confirm(created.Change.ID)
	items, _ = s.Items()
	item := items[0]
	if item.Color != "teal" || item.LocationKind != "label" {
		t.Fatal(item)
	}
	if _, err = c.SetColor(created.Change.ID, "blue"); !errors.Is(err, domain.ErrConflict) {
		t.Fatal("resolved picker", err)
	}
	p, err := c.Preview(domain.Proposal{Action: "update_item", ItemID: item.ID, ExpectedVersion: item.Version, Color: "amber"})
	if err != nil {
		t.Fatal(err)
	}
	c.Discard(p.Change.ID)
	items, _ = s.Items()
	if items[0].Color != "teal" {
		t.Fatal("discard changed item")
	}
	p, _ = c.Preview(domain.Proposal{Action: "update_item", ItemID: item.ID, ExpectedVersion: item.Version, Color: "green"})
	result, err := c.Confirm(p.Change.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = c.Preview(domain.Proposal{Action: "update_item", ItemID: item.ID, ExpectedVersion: item.Version, Color: "blue"}); !errors.Is(err, domain.ErrConflict) {
		t.Fatal("stale item", err)
	}
	if _, err = c.Undo(result.Change.UndoID); err != nil {
		t.Fatal(err)
	}
	items, _ = s.Items()
	if items[0].Color != "teal" {
		t.Fatal("Undo lost colour")
	}
	p, _ = c.Preview(domain.Proposal{Action: "update_item", ItemID: item.ID, ExpectedVersion: items[0].Version, Title: "Party updated"})
	c.Confirm(p.Change.ID)
	items, _ = s.Items()
	if items[0].Color != "teal" {
		t.Fatal("unrelated edit lost colour")
	}
}
func TestLocationClassificationDoesNotFollowAChangedLocation(t *testing.T) {
	c, s := setup(t)
	p, err := c.Preview(domain.Proposal{Action: "create_item", Title: "Workshop", Location: "Damrak 1, Amsterdam", LocationKind: "address"})
	if err != nil {
		t.Fatal(err)
	}
	c.Confirm(p.Change.ID)
	items, _ := s.Items()
	p, err = c.Preview(domain.Proposal{Action: "update_item", ItemID: items[0].ID, ExpectedVersion: items[0].Version, Location: "Home"})
	if err != nil {
		t.Fatal(err)
	}
	c.Confirm(p.Change.ID)
	items, _ = s.Items()
	if items[0].LocationKind != "" {
		t.Fatal("changed label inherited address kind", items[0])
	}
}
