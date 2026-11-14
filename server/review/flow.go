// Package review walks the owner through the items of one email, one at a time. It decides what
// is published next and what the chat says about it; it never applies a change.
package review

import (
	"fmt"
	"strings"

	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/port"
)

// PreviewFunc stores a preview for a proposal and returns its chat entry.
type PreviewFunc func(port.Transaction, domain.Proposal) (domain.Entry, error)

// noun says what a set of candidates is: appointments when every one is an event, otherwise items.
func noun(candidates []domain.CaptureCandidate) string {
	for _, candidate := range candidates {
		if candidate.Proposal.Kind != "event" {
			return "item"
		}
	}
	return "appointment"
}

// Opening is the first message about an email: what was found and how it will be reviewed.
func Opening(subject string, candidates []domain.CaptureCandidate) string {
	where := "your email"
	if subject != "" {
		where = `"` + subject + `"`
	}
	if len(candidates) == 1 {
		return fmt.Sprintf("I found 1 %s in %s.", noun(candidates), where)
	}
	return fmt.Sprintf("I found %d %ss in %s. We will go through them one at a time.", len(candidates), noun(candidates), where)
}

// Advance publishes the next queued candidate when none is open: its question, or its preview.
// With more than one live candidate it first says which one comes next. It changes only the
// capture it is given and the chat entries it writes; the caller stores the capture.
func Advance(tx port.Transaction, c *domain.Capture, preview PreviewFunc) error {
	live, next := 0, -1
	var liveCandidates []domain.CaptureCandidate
	for index, candidate := range c.Candidates {
		switch candidate.State {
		case "superseded":
			continue
		case "pending", "needs-answer":
			return nil
		case "queued":
			if next < 0 {
				next = index
			}
		}
		live++
		liveCandidates = append(liveCandidates, candidate)
	}
	if next < 0 {
		return nil
	}
	candidate := &c.Candidates[next]
	if live > 1 {
		position := 1
		for _, before := range c.Candidates[:next] {
			if before.State != "superseded" {
				position++
			}
		}
		label := noun(liveCandidates)
		text := fmt.Sprintf("%s %d of %d: %s", strings.ToUpper(label[:1])+label[1:], position, live, candidate.Proposal.Title)
		if _, err := tx.PutEntry(domain.Entry{Kind: "minder", Text: text}); err != nil {
			return err
		}
	}
	if candidate.Question != "" {
		candidate.State = "needs-answer"
		return PostQuestions(tx, *c)
	}
	entry, err := preview(tx, candidate.Proposal)
	if err != nil {
		return err
	}
	candidate.ChangeID, candidate.State = entry.Change.ID, "pending"
	return nil
}

// Summary says how the review ended, once every candidate of a multi-item email is resolved.
func Summary(c domain.Capture) (string, bool) {
	live, added, skipped := 0, 0, 0
	for _, candidate := range c.Candidates {
		switch candidate.State {
		case "superseded":
			continue
		case "confirmed":
			added++
		case "discarded":
			skipped++
		default:
			return "", false
		}
		live++
	}
	if live < 2 {
		return "", false
	}
	return fmt.Sprintf("Done: %d added, %d skipped.", added, skipped), true
}
