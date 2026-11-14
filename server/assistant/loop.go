package assistant

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"github.com/alexanderp4580/Minder/server/domain"
)

// Read tools return data to the same loop. Every other tool ends the turn.
var readTools = map[string]bool{"find_sources": true, "read_source": true, "inspect_review": true, "find_finished_items": true}

// loop is one turn's model conversation: ask, run the read tools, ask again, until the model
// gives an answer the server can commit.
type loop struct {
	service   *Service
	turn      domain.AgentTurn
	items     []domain.Item
	documents map[string]domain.SourceDocument // sources read in this turn, for evidence checks
	trace     *runTrace
	patch     *domain.ReviewPatch // the checked review of the final reply
	costEUR   float64
}

func (l *loop) run(ctx context.Context, messages []Message) (domain.AgentOutcome, error) {
	a := l.service
	for {
		if requestBytes(messages) > maxRequestBytes {
			return domain.AgentOutcome{}, &domain.Invalid{Field: "request", Message: "The conversation or source is too large for one request. Try a shorter request."}
		}
		l.trace.rounds++
		r, err := a.ask(ctx, l.turn, messages, l.trace)
		if err != nil {
			return domain.AgentOutcome{}, err
		}
		l.costEUR += r.CostUSD * a.rate
		l.trace.stage = "interpret model reply"
		if a.finder != nil {
			if offer, found, err := offerLookup(r); err != nil {
				return domain.AgentOutcome{}, err
			} else if found {
				// The offer stands alone; the search waits for the owner's tap.
				logOffer(ctx, offer)
				return domain.AgentOutcome{Offer: &offer, CostEUR: l.costEUR}, nil
			}
		}
		follow, err := l.followUp(ctx, r)
		if err != nil {
			return domain.AgentOutcome{}, err
		}
		if follow == nil {
			return l.finish(ctx, r)
		}
		messages = append(append(messages, Message{Role: "assistant", Content: r.Text, ToolCalls: r.Calls}), follow...)
	}
}

// followUp answers the reply's tool calls when the model must see data before it can finish:
// read tools, or a review the server cannot accept yet. Every call gets a result. It returns
// nil when the reply is final.
func (l *loop) followUp(ctx context.Context, r Response) ([]Message, error) {
	needs := false
	for _, call := range r.Calls {
		needs = needs || readTools[call.Name]
	}
	var reviews []Call
	for _, call := range r.Calls {
		if call.Name == "propose_review" {
			reviews = append(reviews, call)
		}
	}
	var problem string
	if len(reviews) > 0 {
		problem = l.checkReview(reviews)
		needs = needs || problem != ""
	}
	if !needs {
		return nil, nil
	}
	var results, images []Message
	for _, call := range r.Calls {
		content, found, err := l.toolResult(ctx, call, problem)
		if err != nil {
			return nil, err
		}
		results = append(results, Message{Role: "tool", ToolCallID: call.ID, Content: content})
		images = append(images, found...)
	}
	return append(results, images...), nil
}

// toolResult runs one read tool, or says why another call was not run.
func (l *loop) toolResult(ctx context.Context, call Call, reviewProblem string) (string, []Message, error) {
	switch {
	case call.Name == "propose_review" && reviewProblem != "":
		return reviewProblem, nil, nil
	case !readTools[call.Name]:
		return "Not run. Call it again once you have the data.", nil, nil
	}
	slog.Debug("assistant read tool", "tool", call.Name, "call_id", call.ID, "argument_bytes", len(call.Arguments), "trace_id", domain.TraceID(ctx))
	return l.readTool(ctx, call)
}

// finish turns a final reply into the outcome the server commits.
func (l *loop) finish(ctx context.Context, r Response) (domain.AgentOutcome, error) {
	if len(r.Calls) > maxChanges+1 {
		return domain.AgentOutcome{}, &domain.Invalid{Field: "request", Message: "Ask for at most five changes at once."}
	}
	out := domain.AgentOutcome{CostEUR: l.costEUR}
	var lists []string
	zone, _ := time.LoadLocation("Europe/Amsterdam")
	for _, call := range r.Calls {
		slog.Debug("assistant tool call", "tool", call.Name, "call_id", call.ID, "argument_bytes", len(call.Arguments), "trace_id", domain.TraceID(ctx))
		switch call.Name {
		case "list_items":
			var filter struct {
				List string `json:"list"`
			}
			if call.Arguments != "" {
				if e := json.Unmarshal([]byte(call.Arguments), &filter); e != nil {
					return out, &domain.Invalid{Field: "request", Message: "The request was unreadable. Try again."}
				}
			}
			lists = append(lists, listText(l.items, filter.List, zone))
			slog.Info("assistant listed items", "list_filter_set", filter.List != "", "trace_id", domain.TraceID(ctx))
		case "propose_review":
			out.Review = l.patch
		default:
			p, err := decodeProposal(call)
			if err != nil {
				return out, err
			}
			out.Proposals = append(out.Proposals, p)
		}
	}
	if len(out.Proposals) > maxChanges {
		return out, &domain.Invalid{Field: "request", Message: "Ask for at most five changes at once."}
	}
	out.Text = strings.Join(lists, "\n\n")
	if out.Text == "" && len(out.Proposals) == 0 && out.Review == nil {
		out.Text = strings.TrimSpace(r.Text)
		if out.Text == "" {
			slog.Error("model sent an empty reply", "trace_id", domain.TraceID(ctx))
			return out, &domain.ModelError{Message: "The model sent an empty reply."}
		}
	}
	return out, nil
}

// decodeProposal reads a propose tool call into a proposal.
func decodeProposal(call Call) (domain.Proposal, error) {
	var p domain.Proposal
	if e := json.Unmarshal([]byte(call.Arguments), &p); e != nil {
		return p, &domain.Invalid{Field: "request", Message: "The proposal was unreadable. Try again."}
	}
	switch call.Name {
	case "propose_create":
		p.Action = "create_item"
	case "propose_update":
		p.Action = "update_item"
	case "propose_complete":
		p.Action = "complete_item"
	case "propose_cancel":
		p.Action = "cancel_item"
	case "propose_delete":
		p.Action = "delete_item"
	case "propose_profile":
		p.Action = "update_profile"
		if p.ProfileID == "" {
			p.Action = "create_profile"
		}
	default:
		slog.Error("model used an unknown tool", "tool", call.Name)
		return p, &domain.ModelError{Message: "The model used an unknown tool."}
	}
	return p, nil
}
