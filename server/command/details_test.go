package command

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/alexanderp4580/Minder/server/domain"
)

func proposalFrom(t *testing.T, raw string) domain.Proposal {
	t.Helper()
	var p domain.Proposal
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestDetailsPreviewConfirmReplaceClearAndUndo(t *testing.T) {
	commands, db := setup(t)
	preview, err := commands.Preview(proposalFrom(t, `{"action":"create_item","title":"Follow-up","details":[{"label":"Meeting link","value":"https://teams.example/join/abc"},{"label":"Dress code","value":"Casual"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	shown := ""
	for _, field := range preview.Change.Fields {
		if field.Name == "details" {
			shown = field.New
		}
	}
	if shown != "Meeting link: https://teams.example/join/abc\nDress code: Casual" {
		t.Fatalf("the preview shows %q", shown)
	}
	if _, err = commands.Confirm(preview.Change.ID); err != nil {
		t.Fatal(err)
	}
	items, _ := db.Items()
	if len(items[0].Details) != 2 || items[0].Details[0] != (domain.Detail{Label: "Meeting link", Value: "https://teams.example/join/abc"}) {
		t.Fatalf("%+v", items[0].Details)
	}
	replace := proposalFrom(t, `{"action":"update_item","details":[{"label":"Room","value":"B4"}]}`)
	replace.ItemID, replace.ExpectedVersion = items[0].ID, items[0].Version
	change, err := commands.Preview(replace)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = commands.Confirm(change.Change.ID); err != nil {
		t.Fatal(err)
	}
	items, _ = db.Items()
	if len(items[0].Details) != 1 || items[0].Details[0].Label != "Room" {
		t.Fatalf("a proposal's details must replace the list: %+v", items[0].Details)
	}
	clearing := proposalFrom(t, `{"action":"update_item","clear_details":true}`)
	clearing.ItemID, clearing.ExpectedVersion = items[0].ID, items[0].Version
	change, err = commands.Preview(clearing)
	if err != nil {
		t.Fatal(err)
	}
	result, err := commands.Confirm(change.Change.ID)
	if err != nil {
		t.Fatal(err)
	}
	if items, _ = db.Items(); len(items[0].Details) != 0 {
		t.Fatalf("clear_details left %+v", items[0].Details)
	}
	if _, err = commands.Undo(result.Change.UndoID); err != nil {
		t.Fatal(err)
	}
	if items, _ = db.Items(); len(items[0].Details) != 1 || items[0].Details[0].Label != "Room" {
		t.Fatalf("undo did not restore the details: %+v", items[0].Details)
	}
}

func TestDetailsAreTrimmedAndBlankPairsDropped(t *testing.T) {
	commands, db := setup(t)
	preview, err := commands.Preview(proposalFrom(t, `{"action":"create_item","title":"x","details":[{"label":"  Link ","value":" https://x.example "},{"label":"","value":"orphan"},{"label":"Empty","value":"  "}]}`))
	if err != nil {
		t.Fatal(err)
	}
	commands.Confirm(preview.Change.ID)
	items, _ := db.Items()
	if len(items[0].Details) != 1 || items[0].Details[0] != (domain.Detail{Label: "Link", Value: "https://x.example"}) {
		t.Fatalf("%+v", items[0].Details)
	}
}

func TestDetailsRejectTooManyOrTooLong(t *testing.T) {
	commands, _ := setup(t)
	many := strings.Repeat(`{"label":"a","value":"b"},`, 11)
	for name, raw := range map[string]string{
		"eleven pairs": `{"action":"create_item","title":"x","details":[` + strings.TrimSuffix(many, ",") + `]}`,
		"long label":   `{"action":"create_item","title":"x","details":[{"label":"` + strings.Repeat("l", 61) + `","value":"v"}]}`,
		"long value":   `{"action":"create_item","title":"x","details":[{"label":"l","value":"` + strings.Repeat("v", 2049) + `"}]}`,
	} {
		if _, err := commands.Preview(proposalFrom(t, raw)); err == nil {
			t.Fatalf("%s was accepted", name)
		}
	}
}
