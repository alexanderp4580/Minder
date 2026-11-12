package assistant_test

import (
	"context"
	"github.com/alexanderp4580/Minder/server/assistant"
	"github.com/alexanderp4580/Minder/server/command"
	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/store"
	"path/filepath"
	"testing"
)

type captureModel struct{ messages []assistant.Message }

func (m *captureModel) Complete(_ context.Context, messages []assistant.Message) (assistant.Response, error) {
	m.messages = messages
	return assistant.Response{Calls: []assistant.Call{{Name: "extract_capture", Arguments: `{"candidates":[{"proposal":{"action":"create_item","title":"Appointment","kind":"event"},"question":"What date is the appointment?","evidence":[{"part":"1","excerpt":"appointment"}]}],"questions":[]}`}}, CostUSD: 0.001}, nil
}
func TestCaptureExtractionKeepsQuestionsAndDoesNotSave(t *testing.T) {
	db, e := store.Open(filepath.Join(t.TempDir(), "db"))
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	m := &captureModel{}
	svc := assistant.New(db, command.New(db), m, 5, 1)
	result, e := svc.Extract(context.Background(), domain.CaptureDocument{Parts: []domain.CapturePart{{ID: "1", Text: "appointment"}}}, nil)
	if e != nil || len(result.Candidates) != 1 || result.Candidates[0].Question == "" {
		t.Fatal(result, e)
	}
	items, _ := db.Items()
	if len(items) != 0 {
		t.Fatal("model saved items")
	}
	cost, e := db.MonthCost()
	if e != nil || cost != 0.001 {
		t.Fatal(cost, e)
	}
}
