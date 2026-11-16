package command

import (
	"errors"
	"log/slog"

	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/port"
	"github.com/alexanderp4580/Minder/server/review"
)

// turnWriter collects the chat entries one committed turn writes, so the turn can record them.
type turnWriter struct {
	tx      port.Transaction
	entries []domain.Entry
}

func (w *turnWriter) put(e domain.Entry) (domain.Entry, error) {
	stored, err := w.tx.PutEntry(e)
	if err == nil {
		w.entries = append(w.entries, stored)
	}
	return stored, err
}

// charge records the request cost on the first entry the turn wrote.
func (w *turnWriter) charge(costEUR float64) error {
	if costEUR <= 0 || len(w.entries) == 0 {
		return nil
	}
	first := w.entries[0]
	first.CostEUR = costEUR
	stored, err := w.tx.PutEntry(first)
	w.entries[0] = stored
	return err
}

// CommitAgentTurn stores what the agent decided and ends the turn, in one transaction. It
// posts the model's text, previews its proposals and applies its review patch. It never
// confirms an item. Committing a finished turn again returns its entries and writes nothing.
func (c *Service) CommitAgentTurn(turn domain.AgentTurn, outcome domain.AgentOutcome) (entries []domain.Entry, err error) {
	err = c.repository.Transact(func(tx port.Transaction) error {
		stored, e := tx.AgentTurn(turn.ID)
		if e != nil {
			return e
		}
		switch stored.State {
		case domain.TurnDone:
			entries, e = storedEntries(tx, stored)
			return e
		case domain.TurnFailed:
			return domain.ErrConflict
		}
		w := &turnWriter{tx: tx}
		if e = w.apply(stored, outcome); e != nil {
			return e
		}
		if e = w.charge(outcome.CostEUR); e != nil {
			return e
		}
		stored.State, stored.LeaseUntil = domain.TurnDone, 0
		for _, entry := range w.entries {
			stored.EntryIDs = append(stored.EntryIDs, entry.ID)
		}
		entries = w.entries
		return tx.PutAgentTurn(stored)
	})
	if err != nil {
		slog.Warn("agent turn commit rejected", "turn_id", turn.ID, "kind", turn.Kind, "error", err, "error_code", domain.ErrorCode(err))
		return nil, err
	}
	slog.Info("agent turn committed", "turn_id", turn.ID, "kind", turn.Kind, "entries", len(entries), "proposals", len(outcome.Proposals), "review", outcome.Review != nil)
	return entries, nil
}

func storedEntries(tx port.Transaction, turn domain.AgentTurn) ([]domain.Entry, error) {
	out := make([]domain.Entry, 0, len(turn.EntryIDs))
	for _, id := range turn.EntryIDs {
		entry, err := tx.Entry(id)
		if err != nil {
			return nil, err
		}
		out = append(out, entry)
	}
	return out, nil
}

// apply writes the outcome's entries: an offer or text, the review, then the previews.
func (w *turnWriter) apply(turn domain.AgentTurn, outcome domain.AgentOutcome) error {
	if outcome.Offer != nil {
		text, ok := domain.OfferText(*outcome.Offer)
		if !ok {
			return &domain.Invalid{Field: "request", Message: "The lookup had no kind or nothing to search for. Try again."}
		}
		_, err := w.put(domain.Entry{Kind: "minder", Text: text})
		return err
	}
	if outcome.Review == nil && turn.Kind == domain.TurnSourceArrival {
		return w.closeEmptyArrival(turn, outcome.Text)
	}
	if outcome.Text != "" {
		if _, err := w.put(domain.Entry{Kind: "minder", Text: outcome.Text}); err != nil {
			return err
		}
	}
	if outcome.Review != nil {
		if err := w.applyReview(*outcome.Review); err != nil {
			return err
		}
	}
	for _, p := range outcome.Proposals {
		var preview domain.Entry
		if err := useCurrentVersion(w.tx, &p); err != nil {
			return err
		}
		if err := previewIn(w.tx, p, &preview); err != nil {
			return err
		}
		w.entries = append(w.entries, preview)
	}
	return nil
}

// closeEmptyArrival ends the review of an email the agent found nothing to review in.
func (w *turnWriter) closeEmptyArrival(turn domain.AgentTurn, text string) error {
	if text == "" {
		text = "I found nothing to review in this email."
	}
	c, err := w.tx.Capture(turn.SourceID)
	if err != nil {
		return err
	}
	if c.Deleted || c.Revision != turn.ExpectedRevision {
		return domain.ErrConflict
	}
	if _, err = w.put(domain.Entry{Kind: "minder", Text: text}); err != nil {
		return err
	}
	c.Status = "reviewed"
	if err = w.tx.PutCapture(c); err != nil {
		return err
	}
	_, err = w.tx.PutEntry(domain.Entry{ID: c.EntryID, Kind: "capture", Text: "Nothing to do. The email is kept.", Capture: &c})
	return err
}

// FailAgentTurn ends a turn as failed, in one transaction, with one visible report. A source
// arrival fails its source, which keeps its Retry action and pushes once. Any other turn
// posts an error entry under the turn's ID. Failing a finished turn does nothing.
func (c *Service) FailAgentTurn(turn domain.AgentTurn, cause error) error {
	return c.repository.Transact(func(tx port.Transaction) error {
		stored, err := tx.AgentTurn(turn.ID)
		if err != nil || stored.Terminal() {
			return err
		}
		stored.State, stored.LeaseUntil, stored.Error = domain.TurnFailed, 0, domain.Describe(cause)
		if stored.Kind == domain.TurnSourceArrival {
			err = failSource(tx, stored, cause)
		} else {
			_, err = tx.PutEntry(domain.Entry{ID: stored.ID, Kind: "error", Text: stored.Error})
		}
		if err != nil {
			return err
		}
		return tx.PutAgentTurn(stored)
	})
}

// failSource fails the source of an arrival turn, unless it was deleted or moved on meanwhile.
func failSource(tx port.Transaction, turn domain.AgentTurn, cause error) error {
	c, err := tx.Capture(turn.SourceID)
	if errors.Is(err, domain.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if c.Deleted || c.Revision != turn.ExpectedRevision {
		return nil
	}
	return review.Fail(tx, c, cause)
}
