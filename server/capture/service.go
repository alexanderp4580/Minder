// Package capture owns durable email ingestion, source preparation and source reading.
package capture

import (
	"context"
	"io"
	"time"

	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/port"
)

type Repository interface {
	port.UnitOfWork
	Capture(string) (domain.Capture, error)
	ClaimCapture(time.Time) (domain.Capture, error)
	FindSources(string) ([]domain.SourceRef, error)
}
type Files interface {
	SaveOriginal(context.Context, string, io.Reader) (string, int64, error)
	Save(context.Context, string, string, io.Reader, int64) (string, int64, error)
	Open(string, string) (io.ReadCloser, error)
	DeleteSource(string) error
}
type Parser interface {
	Parse(context.Context, io.Reader) (domain.CaptureDocument, error)
}
type Sender interface {
	Send(context.Context, string, map[string]string) (bool, error)
}

// Service stores forwarded emails, prepares them for the agent and reads retained sources.
// Files and Parser may be nil when email is off; chat sources still read.
type Service struct {
	Sender     Sender
	Parser     Parser
	repository Repository
	files      Files
}

func New(repository Repository, files Files) *Service {
	return &Service{repository: repository, files: files}
}
func (s *Service) Get(id string) (domain.Capture, error) { return s.repository.Capture(id) }
