package capture

import (
	"errors"
	"fmt"

	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/port"
)

func questionEntryID(captureID string, revision int) string {
	return fmt.Sprintf("q-%s-%d", captureID, revision)
}

// hasOpenQuestions reports whether the assistant is waiting for the owner to answer something.
func hasOpenQuestions(c domain.Capture) bool {
	if len(c.Questions) > 0 {
		return true
	}
	for _, candidate := range c.Candidates {
		if candidate.State == "needs-answer" && candidate.Question != "" {
			return true
		}
	}
	return false
}

// postQuestions adds one chat message holding every open question of this revision, after the email card.
func postQuestions(tx port.Transaction, c domain.Capture) error {
	snapshot := c
	_, err := tx.PutEntry(domain.Entry{ID: questionEntryID(c.ID, c.Revision), CaptureID: c.ID, Kind: "question", Text: "Questions about the email.", Capture: &snapshot})
	return err
}

// closeQuestions turns every question message of the capture into a short closed note.
func closeQuestions(tx port.Transaction, c domain.Capture, text string) error {
	for revision := 1; revision <= c.Revision; revision++ {
		entry, err := tx.Entry(questionEntryID(c.ID, revision))
		if errors.Is(err, domain.ErrNotFound) {
			continue
		}
		if err != nil {
			return err
		}
		if entry.Kind != "question" {
			continue
		}
		entry.Kind, entry.Text, entry.Capture = "minder", text, nil
		if _, err = tx.PutEntry(entry); err != nil {
			return err
		}
	}
	return nil
}
