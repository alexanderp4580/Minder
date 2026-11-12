// Package conversation owns accepted messages and asynchronous reply lifetimes.
package conversation

import (
	"context"
	"errors"
	"log/slog"
	"runtime/debug"
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
	n := utf8.RuneCountInString(text)
	if n < 1 || n > 4000 {
		slog.Warn("message rejected", "reason", "length", "text_len", n)
		return Accepted{}, &domain.Invalid{Field: "text", Message: "Use 1 to 4,000 characters."}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.ctx.Err() != nil {
		slog.Warn("message rejected", "reason", "closed", "text_len", n)
		return Accepted{}, errors.New("conversation is closed")
	}
	if ready, ok := s.replier.(interface{ Preflight() error }); ok {
		if err := ready.Preflight(); err != nil {
			slog.Warn("message rejected", "reason", "preflight", "error_code", domain.ErrorCode(err), "error", err, "text_len", n)
			return Accepted{}, err
		}
	}
	if s.busy {
		slog.Warn("message rejected", "reason", "busy", "text_len", n)
		return Accepted{}, domain.ErrBusy
	}
	entry, err := s.repository.AddEntry(domain.Entry{Kind: "user", Text: text})
	if err != nil {
		slog.Error("message rejected", "reason", "store", "error", err, "text_len", n)
		return Accepted{}, err
	}
	accepted := Accepted{Entry: entry, ReplyID: domain.NewID()}
	slog.Info("message accepted", "entry_id", entry.ID, "cursor", entry.Cursor, "reply_id", accepted.ReplyID, "text_len", n)
	s.busy = true
	s.pending.Add(1)
	go s.reply(accepted.ReplyID, text)
	return accepted, nil
}
func (s *Service) reply(id, text string) {
	defer s.pending.Done()
	defer func() { s.mu.Lock(); s.busy = false; s.mu.Unlock() }()
	started := time.Now()
	defer func() {
		if p := recover(); p != nil {
			slog.Error("reply panicked", "reply_id", id, "panic", p, "stack", string(debug.Stack()), "duration_ms", time.Since(started).Milliseconds())
			s.recordError(id, errors.New("assistant panic"))
		}
	}()
	ctx, cancel := context.WithTimeout(domain.WithTrace(s.ctx, id), 90*time.Second)
	defer cancel()
	slog.Info("reply started", "reply_id", id, "text_len", utf8.RuneCountInString(text))
	entries, err := s.replier.Reply(ctx, text)
	if err != nil {
		slog.Error("reply failed", "reply_id", id, "error", err, "error_code", domain.ErrorCode(err), "duration_ms", time.Since(started).Milliseconds(), "timed_out", errors.Is(ctx.Err(), context.DeadlineExceeded), "cancelled", errors.Is(ctx.Err(), context.Canceled))
		s.recordError(id, err)
		return
	}
	kinds := make([]string, 0, len(entries))
	for _, e := range entries {
		kinds = append(kinds, e.Kind)
	}
	slog.Info("reply finished", "reply_id", id, "entries", len(entries), "kinds", strings.Join(kinds, ","), "duration_ms", time.Since(started).Milliseconds())
}
func (s *Service) recordError(id string, err error) {
	if _, writeErr := s.repository.AddEntry(domain.Entry{ID: id, Kind: "error", Text: domain.Describe(err)}); writeErr != nil {
		slog.Error("could not record the assistant error", "reply_id", id, "error", writeErr, "original_error", err)
	}
}

// Close stops new messages, cancels active replies and waits before storage closes.
func (s *Service) Close() {
	s.mu.Lock()
	s.closed = true
	s.cancel()
	s.mu.Unlock()
	slog.Info("conversation closing; waiting for replies")
	s.pending.Wait()
	slog.Info("conversation closed")
}
