package assistant

import (
	"context"
	"errors"
	"log/slog"
	"sort"
	"sync"
	"time"

	"github.com/alexanderp4580/Minder/server/domain"
)

const (
	maxChanges = 5
	// maxContextItems bounds the open items sent to the model, so the request and its cost stay small.
	maxContextItems = 200
	// maxFinished bounds one lookup of done and cancelled items.
	maxFinished = 50
	// maxRequestBytes bounds the text of one request: the 400,000-byte source budget plus
	// the instructions, the owner's data and recent history. Image data is not counted.
	maxRequestBytes = 450000
	// maxHistoryEntries is how many recent chat entries the agent sees.
	maxHistoryEntries = 20
	// maxHistoryText is the longest earlier owner message shown whole. A longer one is shown as
	// a source reference, so a pasted email does not repeat in every later request.
	maxHistoryText = 4000
)

type Repository interface {
	MonthCost() (float64, error)
	Items() ([]domain.Item, error)
	Lists() ([]string, error)
	Profiles() ([]domain.Profile, error)
	History(int64, int) ([]domain.Entry, error)
	Entry(string) (domain.Entry, error)
	Capture(string) (domain.Capture, error)
	OpenReviews() ([]domain.Capture, error)
	ReserveUsage(float64, float64) (string, error)
	SettleUsage(string, float64, any) error
}

// SourceReader gives the agent the retained sources. It reads only.
type SourceReader interface {
	FindSources(context.Context, string) ([]domain.SourceRef, error)
	ReadSource(context.Context, string) (domain.SourceDocument, error)
}

type Service struct {
	repository  Repository
	model       Model
	finder      Finder
	sources     SourceReader
	limit, rate float64
	mu          sync.Mutex
}

func New(s Repository, m Model, limit, rate float64) *Service {
	return &Service{repository: s, model: m, limit: limit, rate: rate}
}

// WithLookups turns on lookups: the offer tool, and the lookup tools once the owner approves.
func (a *Service) WithLookups(f Finder) *Service {
	a.finder = f
	return a
}

// WithSources lets the agent find and read retained sources.
func (a *Service) WithSources(r SourceReader) *Service {
	a.sources = r
	return a
}
func (a *Service) Preflight() error {
	cost, e := a.repository.MonthCost()
	if e != nil {
		return e
	}
	if cost+0.02*a.rate > a.limit {
		return domain.ErrLimit
	}
	if ready, ok := a.model.(interface{ Ready() error }); ok {
		return ready.Ready()
	}
	return nil
}

// runTrace collects what a turn did, for the log line written when it ends.
type runTrace struct {
	stage     string
	rounds    int
	monthCost float64
}

// Run is the one model execution entry. It builds the context for the turn, lets the model
// read sources and use tools, and returns what the model decided. It stores nothing.
func (a *Service) Run(ctx context.Context, turn domain.AgentTurn) (domain.AgentOutcome, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	started := time.Now()
	trace := &runTrace{stage: "start"}
	out, err := a.run(ctx, turn, trace)
	duration := time.Since(started).Milliseconds()
	switch {
	case errors.Is(err, domain.ErrLimit):
		slog.Warn("assistant reply blocked", "reason", "monthly_limit", "month_cost_eur", trace.monthCost, "limit_eur", a.limit, "reserve_eur", 0.02*a.rate, "trace_id", domain.TraceID(ctx))
	case err != nil:
		slog.Error("assistant reply failed", "stage", trace.stage, "error", err, "error_code", domain.ErrorCode(err), "duration_ms", duration, "trace_id", domain.TraceID(ctx))
	default:
		slog.Info("assistant reply finished", "kind", turn.Kind, "rounds", trace.rounds, "proposals", len(out.Proposals), "review", out.Review != nil, "offer", out.Offer != nil, "cost_eur", out.CostEUR, "duration_ms", duration, "trace_id", domain.TraceID(ctx))
	}
	return out, err
}

func (a *Service) run(ctx context.Context, turn domain.AgentTurn, trace *runTrace) (domain.AgentOutcome, error) {
	trace.stage = "usage check"
	cost, e := a.repository.MonthCost()
	if e != nil {
		return domain.AgentOutcome{}, e
	}
	trace.monthCost = cost
	if cost+0.02*a.rate > a.limit {
		return domain.AgentOutcome{}, domain.ErrLimit
	}
	trace.stage = "build context"
	messages, items, ownerText, e := a.buildContext(turn)
	if e != nil {
		return domain.AgentOutcome{}, e
	}
	slog.Info("assistant reply started", "kind", turn.Kind, "items_total", len(items), "messages", len(messages), "prompt_bytes", requestBytes(messages), "month_cost_eur", cost, "limit_eur", a.limit, "trace_id", domain.TraceID(ctx))
	trace.stage = "model readiness"
	if ready, ok := a.model.(interface{ Ready() error }); ok {
		if e := ready.Ready(); e != nil {
			return domain.AgentOutcome{}, e
		}
	}
	if turn.Kind != domain.TurnSourceArrival {
		if offer, ok := a.approvedOffer(a.history(), ownerText); ok {
			// The owner accepted an offer: run exactly that search, then let the model use the result.
			trace.stage = "lookup"
			messages = append(messages, a.lookupMessage(ctx, offer))
		}
	}
	loop := &loop{service: a, turn: turn, items: items, documents: map[string]domain.SourceDocument{}, trace: trace}
	return loop.run(ctx, messages)
}

// history is the recent chat, read again where a turn needs it unfiltered.
func (a *Service) history() []domain.Entry {
	entries, _ := a.repository.History(0, maxHistoryEntries)
	return entries
}

// contextItems keeps the open items the model can act on: at most maxContextItems, newest change first.
func contextItems(all []domain.Item) []domain.Item {
	open := make([]domain.Item, 0, len(all))
	for _, i := range all {
		if i.Status == "open" || i.Status == "in_progress" {
			open = append(open, i)
		}
	}
	sort.SliceStable(open, func(a, b int) bool { return open[a].UpdatedAt > open[b].UpdatedAt })
	if len(open) > maxContextItems {
		open = open[:maxContextItems]
	}
	return open
}

// requestBytes is the text the request carries. Image data is left out.
func requestBytes(messages []Message) int {
	total := 0
	for _, m := range messages {
		total += len(m.Content)
		for _, p := range m.Parts {
			total += len(p.Text)
		}
		for _, c := range m.ToolCalls {
			total += len(c.Name) + len(c.Arguments)
		}
	}
	return total
}

// ask reserves usage, calls the model once and settles the cost.
func (a *Service) ask(ctx context.Context, turn domain.AgentTurn, messages []Message, trace *runTrace) (Response, error) {
	trace.stage = "reserve usage"
	reservation, e := a.repository.ReserveUsage(a.limit, 0.02*a.rate)
	if e != nil {
		return Response{}, e
	}
	slog.Info("usage reserved", "reservation_id", reservation, "amount_eur", 0.02*a.rate, "trace_id", domain.TraceID(ctx))
	trace.stage = "model call"
	r, e := a.model.Complete(ctx, messages)
	if e != nil {
		slog.Warn("usage reservation kept after model failure", "reservation_id", reservation, "amount_eur", 0.02*a.rate, "trace_id", domain.TraceID(ctx))
		return Response{}, e
	}
	trace.stage = "settle usage"
	detail := map[string]any{"cost_usd": r.CostUSD, "usd_to_eur": a.rate, "usage": r.RawUsage, "turn_kind": turn.Kind}
	if turn.SourceID != "" {
		detail["source_id"] = turn.SourceID
	}
	if e = a.repository.SettleUsage(reservation, r.CostUSD*a.rate, detail); e != nil {
		return Response{}, e
	}
	slog.Info("usage settled", "reservation_id", reservation, "cost_usd", r.CostUSD, "usd_to_eur", a.rate, "cost_eur", r.CostUSD*a.rate, "trace_id", domain.TraceID(ctx))
	return r, nil
}
