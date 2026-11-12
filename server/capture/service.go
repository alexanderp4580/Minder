// Package capture owns durable email ingestion and review orchestration.
package capture

import (
	"context"
	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/port"
	"io"
)

type Repository interface {
	port.UnitOfWork
	Capture(string) (domain.Capture, error)
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
type Service struct {
	repository Repository
	files      Files
	extractor  Extractor
}

func New(repository Repository, files Files, extractor Extractor) *Service {
	return &Service{repository, files, extractor}
}
func (s *Service) Get(id string) (domain.Capture, error) { return s.repository.Capture(id) }
