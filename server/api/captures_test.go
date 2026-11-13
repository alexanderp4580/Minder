package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/alexanderp4580/Minder/server/domain"
	"io"
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

type dismissReviewer struct {
	id       string
	revision int
}

func (d *dismissReviewer) Get(string) (domain.Capture, error) { return domain.Capture{}, nil }
func (d *dismissReviewer) Answer(context.Context, string, int, string) (domain.Entry, error) {
	return domain.Entry{}, nil
}
func (d *dismissReviewer) AnswerCandidate(context.Context, string, int, string, string) (domain.Entry, error) {
	return domain.Entry{}, nil
}
func (d *dismissReviewer) Acknowledge(context.Context, string, int) (domain.Entry, error) {
	return domain.Entry{}, nil
}
func (d *dismissReviewer) Dismiss(_ context.Context, id string, revision int) (domain.Entry, error) {
	d.id, d.revision = id, revision
	if revision != 3 {
		return domain.Entry{}, domain.ErrConflict
	}
	return domain.Entry{Kind: "capture", Text: "Nothing to do. The email is kept."}, nil
}
func (d *dismissReviewer) Retry(string) (domain.Entry, error) { return domain.Entry{}, nil }
func (d *dismissReviewer) Attachment(string, string) (io.ReadCloser, domain.CaptureAttachment, error) {
	return nil, domain.CaptureAttachment{}, domain.ErrNotFound
}

func TestDismissRouteNeedsTheOwnerAndTheCurrentRevision(t *testing.T) {
	reviewer := &dismissReviewer{}
	s := New(nil, nil, nil, nil, rejectOwner{}, "")
	s.Captures = reviewer
	post := func(body string) int {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/v1/captures/abc/dismiss", bytes.NewBufferString(body)))
		return w.Code
	}
	if code := post(`{"revision":3}`); code != 401 || reviewer.id != "" {
		t.Fatal("machine or anonymous caller reached dismiss", code)
	}
	s = New(nil, nil, nil, nil, allowOwner{}, "")
	s.Captures = reviewer
	if code := post(`{"revision":3}`); code != 200 || reviewer.id != "abc" || reviewer.revision != 3 {
		t.Fatal(code, reviewer)
	}
	if code := post(`{"revision":2}`); code != 409 {
		t.Fatal("stale revision", code)
	}
	if code := post(`{"revision":3,"extra":1}`); code != 400 {
		t.Fatal("unknown field accepted", code)
	}
}

type allowOwner struct{}

func (allowOwner) Authenticate(*http.Request) error { return nil }
