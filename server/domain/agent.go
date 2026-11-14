package domain

import "fmt"

// Agent turn kinds say what woke the agent.
const (
	TurnOwner         = "owner"          // a typed message
	TurnSourceArrival = "source_arrival" // a prepared email waits for review
	TurnReviewReply   = "review_reply"   // a quick reply to an open review question
)

// Agent turn states. A turn ends as done or failed and is never run again.
const (
	TurnQueued = "queued"
	TurnActive = "active"
	TurnDone   = "done"
	TurnFailed = "failed"
)

// AgentTurn is one accepted piece of work for the agent. Its text stays on the retained
// chat entry or source; the turn only points at it.
type AgentTurn struct {
	ID               string   `json:"id"`
	Kind             string   `json:"kind"`
	EntryID          string   `json:"entry_id,omitempty"`
	SourceID         string   `json:"source_id,omitempty"`
	CandidateID      string   `json:"candidate_id,omitempty"`
	ExpectedRevision int      `json:"expected_revision,omitempty"`
	State            string   `json:"state"`
	CreatedAt        string   `json:"created_at"`
	LeaseUntil       int64    `json:"lease_until,omitempty"`
	Error            string   `json:"error,omitempty"`
	// EntryIDs are the chat entries the finished turn wrote, so a repeated commit can return them.
	EntryIDs []string `json:"entry_ids,omitempty"`
}

// ArrivalTurn is the turn that asks the agent to review a prepared source revision.
// Its identity is fixed, so preparing the same revision twice queues one turn.
func ArrivalTurn(sourceID string, revision int) AgentTurn {
	return AgentTurn{ID: fmt.Sprintf("arrival-%s-%d", sourceID, revision), Kind: TurnSourceArrival, SourceID: sourceID, ExpectedRevision: revision, State: TurnQueued, CreatedAt: Now()}
}

// Terminal reports whether the turn will not run again.
func (t AgentTurn) Terminal() bool { return t.State == TurnDone || t.State == TurnFailed }

// AgentOutcome is what the agent decided for one turn. Nothing in it is stored until the
// server commits it together with the turn.
type AgentOutcome struct {
	TurnID    string
	Text      string
	Proposals []Proposal
	Review    *ReviewPatch
	Offer     *Offer
	CostEUR   float64
}
