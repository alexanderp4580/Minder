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
	str := map[string]string{"type": "string"}
	flag := map[string]string{"type": "boolean"}
	props := map[string]any{"title": str, "due_at": map[string]string{"type": "string", "description": "RFC3339 with timezone; omit if unknown"}, "timezone": str, "notes": str, "item_id": str, "expected_version": map[string]string{"type": "integer"}, "clear_due": flag, "clear_notes": flag}
	props["remind_at"] = map[string]string{"type": "string", "description": "Explicit reminder time in RFC3339. Set only when the owner asks for a reminder."}
	props["nag_minutes"] = map[string]string{"type": "integer", "description": "Minutes between nags, 1 to 1440. Use 15 by default for reminders. Zero disables nagging."}
	props["clear_reminder"], props["clear_nag"] = flag, flag
	props["list"] = map[string]string{"type": "string", "description": "List name. Reuse an existing list name."}
	props["clear_list"] = flag
	props["location"] = map[string]string{"type": "string", "description": "Where the item happens, for example an address or a place name."}
	props["clear_location"] = flag
	props["profile"] = map[string]string{"type": "string", "description": "Profile id from the profile list."}
	props["clear_profile"] = flag
	props["recurrence"] = map[string]any{"type": "object", "description": "Repeat rule. The item needs a due time.", "properties": map[string]any{"every": map[string]any{"type": "string", "enum": []string{"daily", "weekly", "monthly", "yearly"}}, "interval": map[string]string{"type": "integer"}, "after_completion": flag}, "required": []string{"every", "interval"}, "additionalProperties": false}
	props["clear_recurrence"] = flag
	out := []any{}
	for _, n := range []string{"propose_create", "propose_update", "propose_complete"} {
		required := []string{"title"}
		if n != "propose_create" {
			required = []string{"item_id", "expected_version"}
		}
		out = append(out, tool(n, "Propose an item change for owner confirmation. Never invent item IDs.", props, required))
	}
	profile := map[string]any{"profile_id": map[string]string{"type": "string", "description": "Omit to add a new profile."}, "expected_version": map[string]string{"type": "integer"}, "profile_name": str, "profile_remind_before_minutes": map[string]string{"type": "integer", "description": "Minutes before the due time, 0 to 20160."}, "profile_nag_minutes": map[string]string{"type": "integer", "description": "Minutes between nags, 0 to 1440."}}
	out = append(out, tool("propose_profile", "Propose a new profile or a change to one for owner confirmation.", profile, nil))
	out = append(out, tool("list_items", "List open items, optionally for one list.", map[string]any{"list": str}, nil))
	return out
}
func tool(name, description string, props map[string]any, required []string) any {
	if required == nil {
		required = []string{}
	}
	return map[string]any{"type": "function", "function": map[string]any{"name": name, "description": description, "parameters": map[string]any{"type": "object", "properties": props, "required": required, "additionalProperties": false}}}
}
