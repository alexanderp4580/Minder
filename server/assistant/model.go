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
type Message struct {
	Purpose string        `json:"-"`
	Parts   []ContentPart `json:"-"`

	Role    string `json:"role"`
	Content string `json:"content"`
}
type Model interface {
	Complete(context.Context, []Message) (Response, error)
}

func (m Message) MarshalJSON() ([]byte, error) {
	content := any(m.Content)
	if len(m.Parts) > 0 {
		content = m.Parts
	}
	return json.Marshal(struct {
		Role    string `json:"role"`
		Content any    `json:"content"`
	}{m.Role, content})
}
