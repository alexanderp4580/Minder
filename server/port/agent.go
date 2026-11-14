package port

import "github.com/alexanderp4580/Minder/server/domain"

// AgentTurns stores the durable queue of agent work.
type AgentTurns interface {
	AgentTurn(string) (domain.AgentTurn, error)
	PutAgentTurn(domain.AgentTurn) error
}
