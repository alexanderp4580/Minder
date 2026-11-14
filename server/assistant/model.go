package assistant

import (
	"context"
	"encoding/json"
)

type Call struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}
type Response struct {
	Text     string
	Calls    []Call
	CostUSD  float64
	RawUsage map[string]any
}
type ContentPart struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
}
type ImageURL struct {
	URL string `json:"url"`
}

// Message is one record of the conversation with the model. An assistant message may carry
// the tool calls it made; a tool message answers one of them by ID.
type Message struct {
	Purpose string        `json:"-"`
	Parts   []ContentPart `json:"-"`
	// ToolCalls are the calls an assistant message made. ToolCallID marks a tool result.
	ToolCalls  []Call `json:"-"`
	ToolCallID string `json:"-"`

	Role    string `json:"role"`
	Content string `json:"content"`
}

// Purpose on a request's first message tells the model adapter which tools to offer.
const (
	PurposeChat  = ""
	PurposeOffer = "offer" // chat that may offer a lookup
)

type Model interface {
	Complete(context.Context, []Message) (Response, error)
}

// Diagnostics are optional; the adapter owns their private storage and expiry.
func (a *Service) recordDiagnostic(ctx context.Context, event string, data any) {
	if recorder, ok := a.model.(interface {
		RecordDiagnostic(context.Context, string, any)
	}); ok {
		recorder.RecordDiagnostic(ctx, event, data)
	}
}

type wireCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

func (m Message) MarshalJSON() ([]byte, error) {
	content := any(m.Content)
	if len(m.Parts) > 0 {
		content = m.Parts
	}
	wire := struct {
		Role       string     `json:"role"`
		Content    any        `json:"content"`
		ToolCalls  []wireCall `json:"tool_calls,omitempty"`
		ToolCallID string     `json:"tool_call_id,omitempty"`
	}{Role: m.Role, Content: content, ToolCallID: m.ToolCallID}
	for _, call := range m.ToolCalls {
		w := wireCall{ID: call.ID, Type: "function"}
		w.Function.Name, w.Function.Arguments = call.Name, call.Arguments
		wire.ToolCalls = append(wire.ToolCalls, w)
	}
	return json.Marshal(wire)
}
