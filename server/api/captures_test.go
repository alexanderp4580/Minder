package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/alexanderp4580/Minder/server/domain"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type rejectOwner struct{}

func (rejectOwner) Authenticate(*http.Request) error { return errors.New("owner required") }

type captureVerifier struct{ allowed bool }

func (v captureVerifier) Authenticate(*http.Request, []byte, time.Time) (domain.CaptureEnvelope, error) {
	if !v.allowed {
		return domain.CaptureEnvelope{}, errors.New("denied")
	}
	return domain.CaptureEnvelope{From: "owner@example.com"}, nil
}

type captureSink struct{ calls int }

func (s *captureSink) Ingest(context.Context, domain.CaptureEnvelope, []byte) (string, bool, error) {
	s.calls++
	return "source", s.calls > 1, nil
}
func TestEmailMachineScope(t *testing.T) {
	s := New(nil, nil, nil, nil, rejectOwner{}, "")
	sink := &captureSink{}
	s.EmailVerifier = captureVerifier{}
	s.EmailIngestor = sink
	request := func(path string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString("email"))
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		return w
	}
	if w := request("/v1/email/ingest"); w.Code != 401 || sink.calls != 0 {
		t.Fatal(w.Code, sink.calls)
	}
	s.EmailVerifier = captureVerifier{true}
	if w := request("/v1/email/ingest"); w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	if w := request("/v1/email/ingest"); w.Code != 200 {
		t.Fatal(w.Code)
	}
	if w := request("/v1/chat/messages"); w.Code != 401 {
		t.Fatal("machine reached owner route", w.Code)
	}
}

type reasonedDenial struct{}

func (reasonedDenial) Error() string  { return "denied" }
func (reasonedDenial) Reason() string { return "bad_signature" }

type reasonedVerifier struct{}

func (reasonedVerifier) Authenticate(*http.Request, []byte, time.Time) (domain.CaptureEnvelope, error) {
	return domain.CaptureEnvelope{}, reasonedDenial{}
}

func TestEmailIngestLogsEachOutcomeWithoutContent(t *testing.T) {
	logs := captureLogs(t)
	s := New(nil, nil, nil, nil, rejectOwner{}, "")
	s.EmailVerifier = reasonedVerifier{}
	s.EmailIngestor = &captureSink{}
	post := func() {
		s.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/v1/email/ingest", bytes.NewBufferString("secret body")))
	}
	post()
	denied := find(logs(), "email ingest rejected")
	if denied == nil || denied["level"] != "WARN" || denied["reason"] != "bad_signature" || denied["bytes"] != float64(11) {
		t.Fatal("rejection not logged with a reason", denied)
	}
	s.EmailVerifier = captureVerifier{true}
	post()
	post()
	var accepted []map[string]any
	for _, r := range logs() {
		if r["msg"] == "email ingest accepted" {
			accepted = append(accepted, r)
		}
	}
	if len(accepted) != 2 || accepted[0]["duplicate"] != false || accepted[1]["duplicate"] != true || accepted[0]["capture_id"] != "source" {
		t.Fatal("acceptance not logged", accepted)
	}
	for _, r := range logs() {
		line, _ := json.Marshal(r)
		if bytes.Contains(line, []byte("secret body")) || bytes.Contains(line, []byte("owner@example.com")) {
			t.Fatal("content leaked into the log", string(line))
		}
	}
}
