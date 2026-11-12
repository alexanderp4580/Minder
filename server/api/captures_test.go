package api

import (
	"bytes"
	"context"
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
