package store

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/port"
)

// agentLease is how long a claimed turn blocks the queue before another worker may take it.
const agentLease = 5 * time.Minute

func (t *Tx) AgentTurn(id string) (domain.AgentTurn, error) {
	return decode[domain.AgentTurn](t.tx.QueryRow("SELECT body FROM agent_turn WHERE id=?", id))
}

func (t *Tx) PutAgentTurn(turn domain.AgentTurn) error {
	raw, err := json.Marshal(turn)
	if err != nil {
		return err
	}
	_, err = t.tx.Exec("INSERT INTO agent_turn(id,body) VALUES(?,?) ON CONFLICT(id) DO UPDATE SET body=excluded.body", turn.ID, string(raw))
	return err
}

// EnqueueAgentTurn stores the turn as queued, with its owner entry when there is one, in one
// transaction. A turn ID that already exists returns the stored turn and writes nothing.
func (s *Store) EnqueueAgentTurn(turn domain.AgentTurn, entry *domain.Entry) (out domain.AgentTurn, stored domain.Entry, err error) {
	err = s.Transact(func(tx port.Transaction) error {
		existing, e := tx.AgentTurn(turn.ID)
		if e == nil {
			out = existing
			if existing.EntryID != "" {
				stored, e = tx.Entry(existing.EntryID)
			}
			return e
		}
		if !errors.Is(e, domain.ErrNotFound) {
			return e
		}
		if entry != nil {
			if stored, e = tx.PutEntry(*entry); e != nil {
				return e
			}
			turn.EntryID = stored.ID
		}
		turn.State = domain.TurnQueued
		if turn.CreatedAt == "" {
			turn.CreatedAt = domain.Now()
		}
		out = turn
		return tx.PutAgentTurn(turn)
	})
	return
}

// ClaimAgentTurn gives the oldest queued turn to one worker. While another turn holds an
// unexpired lease nothing is claimed, so turns run one at a time and in arrival order.
// A turn whose lease expired is claimed again.
func (s *Store) ClaimAgentTurn(now time.Time) (out domain.AgentTurn, err error) {
	err = s.Write(func(t *Tx) error {
		turns, e := queryAll[domain.AgentTurn](t.tx, "SELECT body FROM agent_turn WHERE json_extract(body,'$.state') IN ('queued','active') ORDER BY rowid")
		if e != nil {
			return e
		}
		for _, turn := range turns {
			if turn.State == domain.TurnActive && turn.LeaseUntil > now.Unix() {
				return domain.ErrNotFound
			}
		}
		if len(turns) == 0 {
			return domain.ErrNotFound
		}
		out = turns[0]
		out.State = domain.TurnActive
		out.LeaseUntil = now.Add(agentLease).Unix()
		return t.PutAgentTurn(out)
	})
	return
}

// OwnerTurnPending reports whether a typed message or quick reply still waits or runs.
func (s *Store) OwnerTurnPending() (bool, error) {
	turns, err := queryAll[domain.AgentTurn](s.db, "SELECT body FROM agent_turn WHERE json_extract(body,'$.state') IN ('queued','active') AND json_extract(body,'$.kind')<>?", domain.TurnSourceArrival)
	return len(turns) > 0, err
}

// ReleaseAgentTurn returns a claimed turn to the queue, so a restart resumes it.
func (s *Store) ReleaseAgentTurn(id string) error {
	return s.Transact(func(tx port.Transaction) error {
		turn, err := tx.AgentTurn(id)
		if err != nil || turn.State != domain.TurnActive {
			return err
		}
		turn.State, turn.LeaseUntil = domain.TurnQueued, 0
		return tx.PutAgentTurn(turn)
	})
}
