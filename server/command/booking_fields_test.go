package command

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/port"
)

func TestBookingFieldsPreviewConfirmClearAndUndo(t *testing.T) {
	commands, db := setup(t)
	var proposal domain.Proposal
	if err := json.Unmarshal([]byte(`{"action":"create_item","title":"Dentist","phone":"020-1234567","email":"desk@example.com","website":"https://example.com","booking_reference":"00123456789","notes":"- Bring ID"}`), &proposal); err != nil {
		t.Fatal(err)
	}
	preview, err := commands.Preview(proposal)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"phone": "020-1234567", "email": "desk@example.com", "website": "https://example.com", "booking_reference": "00123456789"}
	for name, value := range want {
		found := false
		for _, field := range preview.Change.Fields {
			if field.Name == name && field.New == value {
				found = true
			}
		}
		if !found {
			t.Errorf("preview omitted %s", name)
		}
	}
	if t.Failed() {
		return
	}
	if _, err = commands.Confirm(preview.Change.ID); err != nil {
		t.Fatal(err)
	}
	items, err := db.Items()
	if err != nil {
		t.Fatal(err)
	}
	assertFields := func(item domain.Item, values map[string]string) {
		t.Helper()
		raw, _ := json.Marshal(item)
		var got map[string]any
		json.Unmarshal(raw, &got)
		for name, value := range values {
			actual, _ := got[name].(string)
			if actual != value {
				t.Errorf("%s: got %q want %q", name, actual, value)
			}
		}
	}
	assertFields(items[0], want)
	if items[0].Notes != "- Bring ID" {
		t.Fatal("booking fields entered notes")
	}
	var clear domain.Proposal
	json.Unmarshal([]byte(`{"action":"update_item","clear_phone":true,"clear_email":true,"clear_website":true,"clear_booking_reference":true}`), &clear)
	clear.ItemID, clear.ExpectedVersion = items[0].ID, items[0].Version
	change, err := commands.Preview(clear)
	if err != nil {
		t.Fatal(err)
	}
	result, err := commands.Confirm(change.Change.ID)
	if err != nil {
		t.Fatal(err)
	}
	items, _ = db.Items()
	assertFields(items[0], map[string]string{"phone": "", "email": "", "website": "", "booking_reference": ""})
	if _, err = commands.Undo(result.Change.UndoID); err != nil {
		t.Fatal(err)
	}
	items, _ = db.Items()
	assertFields(items[0], want)
}

func TestBookingFieldsRejectOversizedValues(t *testing.T) {
	for _, field := range []string{"phone", "email", "website", "booking_reference"} {
		t.Run(field, func(t *testing.T) {
			commands, _ := setup(t)
			limit := 200
			if field == "website" {
				limit = 2048
			}
			raw, _ := json.Marshal(map[string]any{"action": "create_item", "title": "Booking", field: strings.Repeat("x", limit+1)})
			var proposal domain.Proposal
			json.Unmarshal(raw, &proposal)
			if _, err := commands.Preview(proposal); err == nil {
				t.Fatal("oversized field accepted")
			}
		})
	}
}

func TestClearLegacyBookingDetailsDoesNotResurrectNotes(t *testing.T) {
	commands, db := setup(t)
	legacy := domain.Item{ID: "legacy", Title: "Dentist", Status: "open", Version: 1, Timezone: "Europe/Amsterdam", CreatedAt: domain.Now(), UpdatedAt: domain.Now(), Notes: "- Phone: 020-1234567\n- Booking reference: 00123456789\n- Bring ID"}
	if err := db.Transact(func(tx port.Transaction) error { return tx.PutItem(legacy) }); err != nil {
		t.Fatal(err)
	}
	preview, err := commands.Preview(domain.Proposal{Action: "update_item", ItemID: legacy.ID, ExpectedVersion: 1, ClearPhone: true, ClearBookingReference: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = commands.Confirm(preview.Change.ID); err != nil {
		t.Fatal(err)
	}
	items, _ := db.Items()
	if strings.Contains(items[0].Notes, "020-1234567") || strings.Contains(items[0].Notes, "00123456789") {
		t.Fatal("cleared details remain in legacy notes")
	}
	if items[0].Notes != "- Bring ID" {
		t.Fatal("unrelated instructions changed")
	}
}
