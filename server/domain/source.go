package domain

import (
	"errors"
	"strings"
	"unicode/utf8"
)

// Source kinds.
const (
	SourceEmail = "email" // a forwarded message kept with its attachments
	SourceChat  = "chat"  // text the owner pasted into chat
)

// Tool-facing source failures. They say why a read found nothing, so the agent can tell the owner.
var (
	ErrSourceUnavailable = errors.New("The source was deleted or is no longer available.")
	ErrSourceUnreadable  = errors.New("The source is kept, but it could not be read.")
)

// SourceRef names retained material the agent can find and read. Payloads load by reference.
type SourceRef struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	EntryID   string `json:"entry_id,omitempty"`
	CaptureID string `json:"capture_id,omitempty"`
	Label     string `json:"label,omitempty"`
	Sender    string `json:"sender,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
	Deleted   bool   `json:"deleted,omitempty"`
}

// SourceDocument is a source's content: its parts, attachments and what could not be read.
type SourceDocument struct {
	Source   SourceRef
	Document CaptureDocument
}

// ChatLabel is the first line of a chat message, shortened to name the source in a list.
func ChatLabel(text string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(text), "\n")
	if utf8.RuneCountInString(line) > 80 {
		line = string([]rune(line)[:80])
	}
	return line
}
