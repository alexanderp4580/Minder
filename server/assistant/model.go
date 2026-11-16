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

	// CachePrefix is the length of the stable start of a system message. The model adapter
	// marks it as a prompt-cache breakpoint, so the rules and tools are read from cache.
	CachePrefix int `json:"-"`
	// Cache marks a prompt-cache breakpoint after this message, so a tool loop that sends the
	// same conversation again reads it from cache.
	Cache bool `json:"-"`

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

// textBlock is a text content block. A block with cache_control ends a cached prefix.
type textBlock struct {
	Type         string        `json:"type"`
	Text         string        `json:"text"`
	CacheControl *cacheControl `json:"cache_control,omitempty"`
}
type cacheControl struct {
	Type string `json:"type"`
}

var ephemeral = &cacheControl{Type: "ephemeral"}

func (m Message) MarshalJSON() ([]byte, error) {
	content := any(m.Content)
	switch {
	case len(m.Parts) > 0:
		content = m.Parts
	case m.Role == "system" && m.CachePrefix > 0 && m.CachePrefix < len(m.Content):
		content = []textBlock{{Type: "text", Text: m.Content[:m.CachePrefix], CacheControl: ephemeral}, {Type: "text", Text: m.Content[m.CachePrefix:]}}
	case m.Cache && m.Content != "":
		content = []textBlock{{Type: "text", Text: m.Content, CacheControl: ephemeral}}
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
