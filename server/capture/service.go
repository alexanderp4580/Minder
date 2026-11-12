// Package capture owns durable email ingestion and review orchestration.
package capture

import (
	"context"
	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/port"
	"io"
	"time"
)

type Repository interface {
	port.UnitOfWork
	Capture(string) (domain.Capture, error)
	ClaimCapture(time.Time) (domain.Capture, error)
}
type Files interface {
	SaveOriginal(context.Context, string, io.Reader) (string, int64, error)
	Save(context.Context, string, string, io.Reader, int64) (string, int64, error)
	Open(string, string) (io.ReadCloser, error)
	DeleteSource(string) error
}
type Extractor interface {
	Extract(context.Context, domain.CaptureDocument, []domain.CaptureAnswer) (domain.CaptureExtraction, error)
}
type Parser interface {
	Parse(context.Context, io.Reader) (domain.CaptureDocument, error)
}
type Sender interface {
	Send(context.Context, string, map[string]string) (bool, error)
}
type Service struct {
	Sender     Sender
	Parser     Parser
	Preview    func(port.Transaction, domain.Proposal) (domain.Entry, error)
	repository Repository
	files      Files
	extractor  Extractor
}

func New(repository Repository, files Files, extractor Extractor) *Service {
	return &Service{repository: repository, files: files, extractor: extractor}
}
func (s *Service) Get(id string) (domain.Capture, error) { return s.repository.Capture(id) }
