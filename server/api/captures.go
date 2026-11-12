package api

import (
	"context"
	"encoding/json"
	"github.com/alexanderp4580/Minder/server/domain"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"time"
)

type EmailVerifier interface {
	Authenticate(*http.Request, []byte, time.Time) (domain.CaptureEnvelope, error)
}
type CaptureIngestor interface {
	Ingest(context.Context, domain.CaptureEnvelope, []byte) (string, bool, error)
}

func (s *Server) ingestEmail(w http.ResponseWriter, r *http.Request) {
	if s.EmailVerifier == nil || s.EmailIngestor == nil {
		http.NotFound(w, r)
		return
	}
	raw, e := io.ReadAll(http.MaxBytesReader(w, r.Body, domain.MaxEmailBytes))
	if e != nil {
		write(w, 413, map[string]string{"message": "Email exceeds the size limit."})
		return
	}
	envelope, e := s.EmailVerifier.Authenticate(r, raw, time.Now())
	if e != nil {
		write(w, 401, map[string]string{"message": "Email authentication failed."})
		return
	}
	noteAuth(w, "email_worker")
	id, dup, e := s.EmailIngestor.Ingest(r.Context(), envelope, raw)
	if e != nil {
		fail(w, e)
		return
	}
	status := 201
	if dup {
		status = 200
	}
	write(w, status, map[string]string{"capture_id": id})
}

type CaptureReviewer interface {
	Get(string) (domain.Capture, error)
	Answer(context.Context, string, int, string) (domain.Entry, error)
	AnswerCandidate(context.Context, string, int, string, string) (domain.Entry, error)
	Acknowledge(context.Context, string, int) (domain.Entry, error)
	Retry(string) (domain.Entry, error)
	Attachment(string, string) (io.ReadCloser, domain.CaptureAttachment, error)
}

func (s *Server) getCapture(w http.ResponseWriter, r *http.Request) {
	if s.Captures == nil {
		http.NotFound(w, r)
		return
	}
	c, e := s.Captures.Get(r.PathValue("id"))
	if e != nil {
		fail(w, e)
		return
	}
	write(w, 200, c)
}
func (s *Server) answerCapture(w http.ResponseWriter, r *http.Request) {
	if s.Captures == nil {
		http.NotFound(w, r)
		return
	}
	var answer struct {
		Revision    int    `json:"revision"`
		CandidateID string `json:"candidate_id"`
		Text        string `json:"text"`
		Acknowledge bool   `json:"acknowledge_coverage"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 20000))
	decoder.DisallowUnknownFields()
	if e := decoder.Decode(&answer); e != nil {
		fail(w, &domain.Invalid{Field: "answer", Message: "The answer is unreadable."})
		return
	}
	var entry domain.Entry
	var e error
	if answer.Acknowledge {
		entry, e = s.Captures.Acknowledge(r.Context(), r.PathValue("id"), answer.Revision)
	} else if answer.CandidateID != "" {
		entry, e = s.Captures.AnswerCandidate(r.Context(), r.PathValue("id"), answer.Revision, answer.CandidateID, answer.Text)
	} else {
		entry, e = s.Captures.Answer(r.Context(), r.PathValue("id"), answer.Revision, answer.Text)
	}
	if e != nil {
		fail(w, e)
		return
	}
	write(w, 200, entry)
}
func (s *Server) retryCapture(w http.ResponseWriter, r *http.Request) {
	if s.Captures == nil {
		http.NotFound(w, r)
		return
	}
	entry, e := s.Captures.Retry(r.PathValue("id"))
	if e != nil {
		fail(w, e)
		return
	}
	write(w, 200, entry)
}
func (s *Server) deleteCapture(w http.ResponseWriter, r *http.Request) {
	if s.Captures == nil {
		http.NotFound(w, r)
		return
	}
	var body struct {
		Revision int `json:"revision"`
	}
	if e := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024)).Decode(&body); e != nil {
		fail(w, &domain.Invalid{Field: "capture", Message: "Choose a source revision."})
		return
	}
	commands, ok := s.commands.(interface {
		Preview(domain.Proposal) (domain.Entry, error)
	})
	if !ok {
		http.NotFound(w, r)
		return
	}
	entry, e := commands.Preview(domain.Proposal{Action: "delete_capture", CaptureID: r.PathValue("id"), ExpectedVersion: body.Revision})
	if e != nil {
		fail(w, e)
		return
	}
	write(w, 200, entry)
}
func (s *Server) captureAttachment(w http.ResponseWriter, r *http.Request) {
	if s.Captures == nil {
		http.NotFound(w, r)
		return
	}
	f, a, e := s.Captures.Attachment(r.PathValue("id"), r.PathValue("attachmentId"))
	if e != nil {
		fail(w, domain.ErrNotFound)
		return
	}
	defer f.Close()
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", a.MediaType)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filepath.Base(a.Name)}))
	io.Copy(w, f)
}
