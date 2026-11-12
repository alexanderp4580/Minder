package api

import (
	"context"
	"github.com/alexanderp4580/Minder/server/domain"
	"io"
	"net/http"
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
