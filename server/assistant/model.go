package assistant

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
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
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
type Model interface {
	Complete(context.Context, []Message) (Response, error)
}
type OpenRouter struct {
	Key, Model, URL string
	Client          *http.Client
}

func (m *OpenRouter) Ready() error {
	if m.Key == "" {
		return errors.New("The model key is not configured.")
	}
	return nil
}
func (m *OpenRouter) Complete(ctx context.Context, messages []Message) (Response, error) {
	if m.Key == "" {
		return Response{}, errors.New("The model key is not configured.")
	}
	model := m.Model
	if model == "" {
		model = "anthropic/claude-haiku-5.5"
	}
	url := m.URL
	if url == "" {
		url = "https://openrouter.ai/api/v1/chat/completions"
	}
	payload := map[string]any{"model": model, "messages": messages, "tools": tools(), "max_tokens": 1000, "usage": map[string]bool{"include": true}}
	b, _ := json.Marshal(payload)
	req, e := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(b))
	if e != nil {
		return Response{}, e
	}
	req.Header.Set("Authorization", "Bearer "+m.Key)
	req.Header.Set("Content-Type", "application/json")
	client := m.Client
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	res, e := client.Do(req)
	if e != nil {
		return Response{}, errors.New("The model could not connect. Try again.")
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		io.Copy(io.Discard, io.LimitReader(res.Body, 4096))
		return Response{}, fmt.Errorf("The model returned an error (%d). Try again.", res.StatusCode)
	}
	var wire struct {
		Choices []struct {
			Message struct {
				Content *string `json:"content"`
				Calls   []struct {
					ID       string `json:"id"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
		Usage map[string]any `json:"usage"`
	}
	if e = json.NewDecoder(io.LimitReader(res.Body, 2<<20)).Decode(&wire); e != nil {
		return Response{}, errors.New("The model reply was unreadable. Try again.")
	}
	if len(wire.Choices) != 1 {
		return Response{}, errors.New("The model sent no reply. Try again.")
	}
	cost, ok := wire.Usage["cost"].(float64)
	if !ok || cost < 0 {
		return Response{}, errors.New("The model cost is missing. Try again later.")
	}
	out := Response{CostUSD: cost, RawUsage: wire.Usage}
	if wire.Choices[0].Message.Content != nil {
		out.Text = *wire.Choices[0].Message.Content
	}
	for _, c := range wire.Choices[0].Message.Calls {
		out.Calls = append(out.Calls, Call{ID: c.ID, Name: c.Function.Name, Arguments: c.Function.Arguments})
	}
	return out, nil
}
func tools() []any {
	props := map[string]any{"title": map[string]string{"type": "string"}, "due_at": map[string]string{"type": "string", "description": "RFC3339 with timezone; omit if unknown"}, "timezone": map[string]string{"type": "string"}, "notes": map[string]string{"type": "string"}, "item_id": map[string]string{"type": "string"}, "expected_version": map[string]string{"type": "integer"}, "clear_due": map[string]string{"type": "boolean"}, "clear_notes": map[string]string{"type": "boolean"}}
	out := []any{}
	for _, n := range []string{"propose_create", "propose_update", "propose_complete", "list_items"} {
		required := []string{}
		switch n {
		case "propose_create":
			required = []string{"title"}
		case "propose_update", "propose_complete":
			required = []string{"item_id", "expected_version"}
		}
		out = append(out, map[string]any{"type": "function", "function": map[string]any{"name": n, "description": "Propose a change for owner confirmation, or list current items. Never invent item IDs.", "parameters": map[string]any{"type": "object", "properties": props, "required": required, "additionalProperties": false}}})
	}
	return out
}
