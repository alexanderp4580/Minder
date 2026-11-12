// Package conversation owns accepted messages and asynchronous reply lifetimes.
package conversation

import (
	"context"
	"errors"
	"log"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/alexanderp4580/Minder/server/domain"
)

type Repository interface {
	AddEntry(domain.Entry) (domain.Entry, error)
}
type Replier interface {
	Reply(context.Context, string) ([]domain.Entry, error)
}
type Accepted struct {
	Entry   domain.Entry `json:"entry"`
	ReplyID string       `json:"reply_id"`
}
type Service struct {
	repository   Repository
	replier      Replier
	ctx          context.Context
	cancel       context.CancelFunc
	mu           sync.Mutex
	pending      sync.WaitGroup
	busy, closed bool
}

func New(ctx context.Context, repository Repository, replier Replier) *Service {
	ctx, cancel := context.WithCancel(ctx)
	return &Service{repository: repository, replier: replier, ctx: ctx, cancel: cancel}
}
func (s *Service) Accept(text string) (Accepted, error) {
	text = strings.TrimSpace(text)
	if n := utf8.RuneCountInString(text); n < 1 || n > 4000 {
		return Accepted{}, &domain.Invalid{Field: "text", Message: "Use 1 to 4,000 characters."}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.ctx.Err() != nil {
		return Accepted{}, errors.New("conversation is closed")
	}
	if ready, ok := s.replier.(interface{ Preflight() error }); ok {
		if err := ready.Preflight(); err != nil {
			return Accepted{}, err
		}
	}
	if s.busy {
		return Accepted{}, domain.ErrBusy
	}
	entry, err := s.repository.AddEntry(domain.Entry{Kind: "user", Text: text})
	if err != nil {
		return Accepted{}, err
	}
	accepted := Accepted{Entry: entry, ReplyID: domain.NewID()}
	s.busy = true
	s.pending.Add(1)
	go s.reply(accepted.ReplyID, text)
	return accepted, nil
}
func (s *Service) reply(id, text string) {
	defer s.pending.Done()
	defer func() { s.mu.Lock(); s.busy = false; s.mu.Unlock() }()
	defer func() {
		if recover() != nil {
			s.recordError(id, errors.New("assistant panic"))
		}
	}()
	ctx, cancel := context.WithTimeout(s.ctx, 90*time.Second)
	defer cancel()
	if _, err := s.replier.Reply(ctx, text); err != nil {
		s.recordError(id, err)
	}
}
func (s *Service) recordError(id string, err error) {
	if _, writeErr := s.repository.AddEntry(domain.Entry{ID: id, Kind: "error", Text: domain.Describe(err)}); writeErr != nil {
		log.Print("Could not record the assistant error")
	}
}

// Close stops new messages, cancels active replies and waits before storage closes.
func (s *Service) Close() {
	s.mu.Lock()
	s.closed = true
	s.cancel()
	s.mu.Unlock()
	s.pending.Wait()
}
