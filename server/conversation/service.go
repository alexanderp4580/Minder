// Package conversation accepts owner messages into the durable agent queue and runs the queue
// one turn at a time.
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

// MaxMessageBytes is the textual source budget. One message may use all of it.
const MaxMessageBytes = 400000

const (
	turnTimeout  = 3 * time.Minute
	pollInterval = 500 * time.Millisecond
)

type Repository interface {
	EnqueueAgentTurn(domain.AgentTurn, *domain.Entry) (domain.AgentTurn, domain.Entry, error)
	ClaimAgentTurn(time.Time) (domain.AgentTurn, error)
	ReleaseAgentTurn(string) error
	OwnerTurnPending() (bool, error)
}

// Agent decides what a turn does. It stores nothing.
type Agent interface {
	Run(context.Context, domain.AgentTurn) (domain.AgentOutcome, error)
}

// Finisher stores the outcome of a turn, or its failure, together with the turn's end state.
type Finisher interface {
	CommitAgentTurn(domain.AgentTurn, domain.AgentOutcome) ([]domain.Entry, error)
	FailAgentTurn(domain.AgentTurn, error) error
}

type Accepted struct {
	Entry   domain.Entry `json:"entry"`
	ReplyID string       `json:"reply_id"`
}
type Service struct {
	repository Repository
	agent      Agent
	finisher   Finisher
	ctx        context.Context
	cancel     context.CancelFunc
	wake       chan struct{}
	mu         sync.Mutex
	worker     sync.WaitGroup
	closed     bool
}

// New starts the worker that runs queued turns. Close stops it.
func New(ctx context.Context, repository Repository, agent Agent, finisher Finisher) *Service {
	ctx, cancel := context.WithCancel(ctx)
	s := &Service{repository: repository, agent: agent, finisher: finisher, ctx: ctx, cancel: cancel, wake: make(chan struct{}, 1)}
	s.worker.Add(1)
	go s.work()
	return s
}

// Accept saves an owner message and queues its turn. It returns before the agent runs.
func (s *Service) Accept(text string) (Accepted, error) {
	text = strings.TrimSpace(text)
	n := len(text)
	if n < 1 || n > MaxMessageBytes || !utf8.ValidString(text) {
		slog.Warn("message rejected", "reason", "length", "text_len", utf8.RuneCountInString(text), "text_bytes", n)
		return Accepted{}, &domain.Invalid{Field: "text", Message: "Use 1 to 400,000 bytes of text."}
	}
	turn, entry, err := s.enqueue(domain.AgentTurn{ID: domain.NewID(), Kind: domain.TurnOwner}, domain.Entry{Kind: "user", Text: text}, true)
	if err != nil {
		return Accepted{}, err
	}
	slog.Info("message accepted", "entry_id", entry.ID, "cursor", entry.Cursor, "reply_id", turn.ID, "text_len", utf8.RuneCountInString(text), "text_bytes", n)
	return Accepted{Entry: entry, ReplyID: turn.ID}, nil
}

// enqueue saves an owner entry with its turn. Owner work waits for the previous owner turn to
// finish when exclusive is set; incoming sources queue independently.
func (s *Service) enqueue(turn domain.AgentTurn, entry domain.Entry, exclusive bool) (domain.AgentTurn, domain.Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.ctx.Err() != nil {
		slog.Warn("message rejected", "reason", "closed")
		return turn, entry, errors.New("conversation is closed")
	}
	if ready, ok := s.agent.(interface{ Preflight() error }); ok {
		if err := ready.Preflight(); err != nil {
			slog.Warn("message rejected", "reason", "preflight", "error_code", domain.ErrorCode(err), "error", err)
			return turn, entry, err
		}
	}
	if exclusive {
		busy, err := s.repository.OwnerTurnPending()
		if err != nil {
			return turn, entry, err
		}
		if busy {
			slog.Warn("message rejected", "reason", "busy")
			return turn, entry, domain.ErrBusy
		}
	}
	turn, entry, err := s.repository.EnqueueAgentTurn(turn, &entry)
	if err != nil {
		slog.Error("message rejected", "reason", "store", "error", err)
		return turn, entry, err
	}
	s.nudge()
	return turn, entry, nil
}

func (s *Service) nudge() {
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

func (s *Service) work() {
	defer s.worker.Done()
	for s.ctx.Err() == nil {
		turn, err := s.repository.ClaimAgentTurn(time.Now())
		switch {
		case err == nil:
			s.run(turn)
			continue
		case !errors.Is(err, domain.ErrNotFound):
			slog.Error("agent queue claim failed", "error", err)
		}
		select {
		case <-s.ctx.Done():
		case <-s.wake:
		case <-time.After(pollInterval):
		}
	}
}

// run executes one claimed turn and stores its outcome. A turn cut short by shutdown goes
// back to the queue and resumes after restart.
func (s *Service) run(turn domain.AgentTurn) {
	started := time.Now()
	defer func() {
		if p := recover(); p != nil {
			slog.Error("reply panicked", "reply_id", turn.ID, "panic", p, "stack", string(debug.Stack()), "duration_ms", time.Since(started).Milliseconds())
			s.fail(turn, errors.New("assistant panic"))
		}
	}()
	ctx, cancel := context.WithTimeout(domain.WithTrace(s.ctx, turn.ID), turnTimeout)
	defer cancel()
	slog.Info("reply started", "reply_id", turn.ID, "kind", turn.Kind)
	outcome, err := s.agent.Run(ctx, turn)
	if err == nil {
		outcome.TurnID = turn.ID
		var entries []domain.Entry
		if entries, err = s.finisher.CommitAgentTurn(turn, outcome); err == nil {
			kinds := make([]string, 0, len(entries))
			for _, e := range entries {
				kinds = append(kinds, e.Kind)
			}
			slog.Info("reply finished", "reply_id", turn.ID, "entries", len(entries), "kinds", strings.Join(kinds, ","), "duration_ms", time.Since(started).Milliseconds())
			return
		}
	}
	if s.ctx.Err() != nil {
		slog.Info("reply interrupted by shutdown; turn stays queued", "reply_id", turn.ID, "duration_ms", time.Since(started).Milliseconds())
		if releaseErr := s.repository.ReleaseAgentTurn(turn.ID); releaseErr != nil {
			slog.Error("turn could not be released", "reply_id", turn.ID, "error", releaseErr)
		}
		return
	}
	slog.Error("reply failed", "reply_id", turn.ID, "error", err, "error_code", domain.ErrorCode(err), "duration_ms", time.Since(started).Milliseconds(), "timed_out", errors.Is(ctx.Err(), context.DeadlineExceeded))
	s.fail(turn, err)
}

func (s *Service) fail(turn domain.AgentTurn, cause error) {
	if err := s.finisher.FailAgentTurn(turn, cause); err != nil {
		slog.Error("could not record the turn failure", "reply_id", turn.ID, "error", err, "original_error", cause)
	}
}

// Close stops new messages, cancels the active turn and waits before storage closes.
func (s *Service) Close() {
	s.mu.Lock()
	s.closed = true
	s.cancel()
	s.mu.Unlock()
	slog.Info("conversation closing; waiting for the active turn")
	s.worker.Wait()
	slog.Info("conversation closed")
}
