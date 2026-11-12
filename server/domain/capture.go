package domain

import "time"

const MaxEmailBytes = 20 << 20

// CaptureEnvelope contains only the outer sender, recipient and gateway-authenticated result.
type CaptureEnvelope struct {
	From      string `json:"from"`
	To        string `json:"to"`
	DMARC     string `json:"dmarc"`
	RequestID string `json:"-"`
}
type Capture struct {
	ID           string              `json:"id"`
	Status       string              `json:"status"`
	Revision     int                 `json:"revision"`
	Sender       string              `json:"sender,omitempty"`
	Subject      string              `json:"subject,omitempty"`
	ReceivedAt   string              `json:"received_at"`
	OriginalDate string              `json:"original_date,omitempty"`
	EntryID      string              `json:"entry_id"`
	Attachments  []CaptureAttachment `json:"attachments"`
	Coverage     []CaptureCoverage   `json:"coverage"`
	Candidates   []CaptureCandidate  `json:"candidates"`
	Answers      []CaptureAnswer     `json:"answers,omitempty"`
	Questions    []string            `json:"questions,omitempty"`
	Error        string              `json:"error,omitempty"`
	Attempts     int                 `json:"attempts"`
	Calls        int                 `json:"calls"`
	NextAt       int64               `json:"next_at,omitempty"`
	LeaseUntil   int64               `json:"lease_until,omitempty"`
	Deleted      bool                `json:"deleted,omitempty"`
	CostEUR      float64             `json:"cost_eur,omitempty"`
}
type CaptureAttachment struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	MediaType string `json:"media_type"`
	Size      int64  `json:"size"`
	SHA256    string `json:"sha256"`
}
type CaptureCoverage struct {
	Part   string `json:"part"`
	Name   string `json:"name,omitempty"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}
type CaptureEvidence struct {
	Part    string `json:"part"`
	Excerpt string `json:"excerpt"`
}
type CaptureCandidate struct {
	ID       string            `json:"id"`
	Revision int               `json:"revision"`
	State    string            `json:"state"`
	Proposal Proposal          `json:"proposal"`
	Question string            `json:"question,omitempty"`
	Evidence []CaptureEvidence `json:"evidence,omitempty"`
	ChangeID string            `json:"change_id,omitempty"`
	ItemID   string            `json:"item_id,omitempty"`
}
type CaptureAnswer struct {
	Revision    int    `json:"revision"`
	CandidateID string `json:"candidate_id,omitempty"`
	Text        string `json:"text"`
}
type CaptureReceipt struct{ ID, Digest, CaptureID string }
type CapturePart struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	MediaType string `json:"media_type"`
	Text      string `json:"text,omitempty"`
	Data      []byte `json:"-"`
}
type CaptureDocument struct {
	Subject      string              `json:"subject"`
	OriginalDate string              `json:"original_date,omitempty"`
	Parts        []CapturePart       `json:"parts"`
	Attachments  []CaptureAttachment `json:"attachments"`
	Coverage     []CaptureCoverage   `json:"coverage"`
	Questions    []string            `json:"questions,omitempty"`
}
type CaptureExtraction struct {
	Candidates []CaptureCandidate `json:"candidates"`
	Questions  []string           `json:"questions,omitempty"`
}

func (c Capture) Due(now time.Time) bool {
	return !c.Deleted && (c.Status == "received" || c.Status == "processing") && c.NextAt <= now.Unix() && c.LeaseUntil <= now.Unix()
}
