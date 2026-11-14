package capture_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/openrouter"
)

// Exercises ingestion and the real private recorder, without any model call.
func TestPrivateTraceKeepsWholeEmailBeforeAgentReadsIt(t *testing.T) {
	f := newFixture(t)
	dir := t.TempDir()
	trace, err := openrouter.NewTrace(dir, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	f.svc.Diagnostics = (&openrouter.Client{Trace: trace}).RecordDiagnostic
	raw := []byte("From: owner@example.com\r\nSubject: PRIVATE_BOOKING\r\n\r\n" + strings.Repeat("long body\n", 50000) + "PRIVATE_QUESTIONNAIRE_AT_END\xff")
	id, _, err := f.svc.Ingest(context.Background(), domain.CaptureEnvelope{RequestID: "whole-mail"}, raw)
	if err != nil {
		t.Fatal(err)
	}
	files, err := os.ReadDir(dir)
	if err != nil || len(files) != 1 {
		t.Fatalf("original capture missing: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(dir, files[0].Name()))
	if err != nil {
		t.Fatal(err)
	}
	var record struct {
		Event string `json:"msg"`
		Data  struct {
			SourceID string `json:"source_id"`
			MIME     string `json:"mime_base64"`
		} `json:"data"`
	}
	if err = json.Unmarshal(b, &record); err != nil {
		t.Fatal(err)
	}
	decoded, err := base64.StdEncoding.DecodeString(record.Data.MIME)
	if err != nil || !bytes.Equal(decoded, raw) || record.Data.SourceID != id || record.Event != "email_received" {
		t.Fatal("whole original email was not preserved byte for byte")
	}
}

func TestPrivateTraceIncludesParsedEmailAndArrivalTurn(t *testing.T) {
	f := newFixture(t)
	dir := t.TempDir()
	trace, err := openrouter.NewTrace(dir, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	f.svc.Diagnostics = (&openrouter.Client{Trace: trace}).RecordDiagnostic
	id := f.ingest(t, "parsed-mail", "From: owner@example.com\r\nSubject: PRIVATE_BOOKING\r\n\r\nPRIVATE_QUESTIONNAIRE")
	f.prepare(t)
	files, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, file := range files {
		b, err := os.ReadFile(filepath.Join(dir, file.Name()))
		if err != nil {
			t.Fatal(err)
		}
		var record struct {
			Event   string `json:"msg"`
			TraceID string `json:"trace_id"`
			Data    struct {
				SourceID string                 `json:"source_id"`
				Document domain.CaptureDocument `json:"document"`
			} `json:"data"`
		}
		if err = json.Unmarshal(b, &record); err != nil {
			t.Fatal(err)
		}
		if record.Event != "email_prepared" {
			continue
		}
		found = true
		if record.Data.SourceID != id || record.TraceID != domain.ArrivalTurn(id, 1).ID || record.Data.Document.Subject != "PRIVATE_BOOKING" || len(record.Data.Document.Parts) != 1 || !strings.Contains(record.Data.Document.Parts[0].Text, "PRIVATE_QUESTIONNAIRE") {
			t.Fatal("parsed email or turn correlation missing")
		}
	}
	if !found {
		t.Fatal("parsed email missing")
	}
}
