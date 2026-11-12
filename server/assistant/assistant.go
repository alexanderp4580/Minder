package assistant

import (
	"context"
	"encoding/json"
	"github.com/alexanderp4580/Minder/server/command"
	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/store"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	maxChanges = 5
	// maxContextItems bounds the open items sent to the model, so the request and its cost stay small.
	maxContextItems = 200
)

type Service struct {
	store       *store.Store
	commands    *command.Service
	model       Model
	limit, rate float64
	mu          sync.Mutex
}

func New(s *store.Store, c *command.Service, m Model, limit, rate float64) *Service {
	return &Service{store: s, commands: c, model: m, limit: limit, rate: rate}
}
func (a *Service) Preflight() error {
	cost, e := a.store.MonthCost()
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

func (a *Service) Reply(ctx context.Context, text string) ([]domain.Entry, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	cost, e := a.store.MonthCost()
	if e != nil {
		return nil, e
	}
	if cost+0.02*a.rate > a.limit {
		return nil, domain.ErrLimit
	}
	items, e := a.store.Items()
	if e != nil {
		return nil, e
	}
	lists, e := a.store.Lists()
	if e != nil {
		return nil, e
	}
	profiles, e := a.store.Profiles()
	if e != nil {
		return nil, e
	}
	zone, _ := time.LoadLocation("Europe/Amsterdam")
	messages := []Message{{Role: "system", Content: systemPrompt(time.Now().In(zone), lists, profiles, contextItems(items))}}
	history, e := a.store.History(0, 20)
	if e != nil {
		return nil, e
	}
	for _, h := range history {
		if h.Kind == "user" {
			messages = append(messages, Message{Role: "user", Content: h.Text})
		} else if h.Kind == "minder" {
			messages = append(messages, Message{Role: "assistant", Content: h.Text})
		} else if h.Kind == "preview" || h.Kind == "result" || h.Kind == "error" {
			encoded, err := json.Marshal(h)
			if err != nil {
				return nil, err
			}
			messages = append(messages, Message{Role: "assistant", Content: string(encoded)})
		}
	}
	if len(messages) == 1 || messages[len(messages)-1].Role != "user" || messages[len(messages)-1].Content != text {
		messages = append(messages, Message{Role: "user", Content: text})
	}
	encoded, _ := json.Marshal(messages)
	if len(encoded) > 60000 {
		return nil, &domain.Invalid{Field: "request", Message: "The conversation is too large. Try a shorter request."}
	}
	if ready, ok := a.model.(interface{ Ready() error }); ok {
		if e := ready.Ready(); e != nil {
			return nil, e
		}
	}
	reservation, e := a.store.ReserveUsage(a.limit, 0.02*a.rate)
	if e != nil {
		return nil, e
	}
	r, e := a.model.Complete(ctx, messages)
	if e != nil {
		return nil, e
	}
	if e = a.store.SettleUsage(reservation, r.CostUSD*a.rate, map[string]any{"cost_usd": r.CostUSD, "usd_to_eur": a.rate, "usage": r.RawUsage}); e != nil {
		return nil, e
	}
	if len(r.Calls) > maxChanges {
		return nil, &domain.Invalid{Field: "request", Message: "Ask for at most five changes at once."}
	}
	out := []domain.Entry{}
	proposals := []domain.Proposal{}
	unbilled := r.CostUSD * a.rate
	charge := func() float64 { // the request cost sits on its first entry only
		c := unbilled
		unbilled = 0
		return c
	}
	for _, call := range r.Calls {
		if call.Name == "list_items" {
			var filter struct {
				List string `json:"list"`
			}
			if call.Arguments != "" {
				if e = json.Unmarshal([]byte(call.Arguments), &filter); e != nil {
					return nil, &domain.Invalid{Field: "request", Message: "The request was unreadable. Try again."}
				}
			}
			var lines []string
			for _, i := range items {
				if filter.List != "" && !strings.EqualFold(i.List, filter.List) {
					continue
				}
				if i.Status != "done" && i.Status != "cancelled" {
					line := i.Title
					if at := domain.Anchor(i); at != "" {
						due, _ := time.Parse(time.RFC3339, at)
						line += " · " + due.In(zone).Format("Mon 2 Jan 15:04")
					}
					lines = append(lines, line)
				}
			}
			text, heading := "No open items.", "Open items:"
			if filter.List != "" {
				text, heading = "No open items on "+filter.List+".", "Open items on "+filter.List+":"
			}
			if len(lines) > 0 {
				text = heading + "\n" + strings.Join(lines, "\n")
			}
			entry, e := a.store.AddEntry(domain.Entry{Kind: "minder", Text: text, CostEUR: charge()})
			if e != nil {
				return nil, e
			}
			out = append(out, entry)
			continue
		}
		var p domain.Proposal
		if e = json.Unmarshal([]byte(call.Arguments), &p); e != nil {
			return nil, &domain.Invalid{Field: "request", Message: "The proposal was unreadable. Try again."}
		}
		switch call.Name {
		case "propose_create":
			p.Action = "create_item"
		case "propose_update":
			p.Action = "update_item"
		case "propose_complete":
			p.Action = "complete_item"
		case "propose_profile":
			p.Action = "update_profile"
			if p.ProfileID == "" {
				p.Action = "create_profile"
			}
		default:
			return nil, &domain.ModelError{Message: "The model used an unknown tool."}
		}
		proposals = append(proposals, p)
	}
	if len(proposals) > 0 {
		previews, e := a.commands.PreviewAll(proposals)
		if e != nil {
			return nil, e
		}
		if c := charge(); c > 0 && len(previews) > 0 {
			if previews[0], e = a.store.AttachCost(previews[0].ID, c); e != nil {
				return nil, e
			}
		}
		out = append(out, previews...)
	}
	if len(out) == 0 {
		if strings.TrimSpace(r.Text) == "" {
			return nil, &domain.ModelError{Message: "The model sent an empty reply."}
		}
		entry, e := a.store.AddEntry(domain.Entry{Kind: "minder", Text: r.Text, CostEUR: charge()})
		if e != nil {
			return nil, e
		}
		out = append(out, entry)
	}
	return out, nil
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
