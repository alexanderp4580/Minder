package openrouter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	neturl "net/url"
	"strings"
	"time"

	"github.com/alexanderp4580/Minder/server/assistant"
	"github.com/alexanderp4580/Minder/server/domain"
)

// maxOutputTokens leaves room for the model's thinking and for up to five tool calls.
// A reply cut off by this limit is an error, never a half-read proposal.
const maxOutputTokens = 4000

type Client struct {
	Key, Model, URL string
	Client          *http.Client
}

func (m *Client) Ready() error {
	if m.Key == "" {
		return &domain.ModelError{Message: "The model key is not configured."}
	}
	return nil
}
func (m *Client) Complete(ctx context.Context, messages []assistant.Message) (assistant.Response, error) {
	if m.Key == "" {
		slog.Error("model request failed", "stage", "setup", "reason", "model key is not configured")
		return assistant.Response{}, &domain.ModelError{Message: "The model key is not configured."}
	}
	model := m.Model
	if model == "" {
		model = "anthropic/claude-haiku-5.5"
	}
	url := m.URL
	if url == "" {
		url = "https://openrouter.ai/api/v1/chat/completions"
	}
	selectedTools := tools()
	if len(messages) > 0 && messages[0].Purpose == "email" {
		selectedTools = captureTools()
	}
	payload := map[string]any{"model": model, "messages": messages, "tools": selectedTools, "max_tokens": maxOutputTokens, "usage": map[string]bool{"include": true}}
	b, _ := json.Marshal(payload)
	started := time.Now()
	slog.Info("model request", "model", model, "endpoint_host", hostOf(url), "messages", len(messages), "tools", len(tools()), "request_bytes", len(b), "max_tokens", maxOutputTokens, "trace_id", domain.TraceID(ctx))
	req, e := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(b))
	if e != nil {
		slog.Error("model request failed", "stage", "build", "error", e)
		return assistant.Response{}, e
	}
	req.Header.Set("Authorization", "Bearer "+m.Key)
	req.Header.Set("Content-Type", "application/json")
	client := m.Client
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	res, e := client.Do(req)
	if e != nil {
		slog.Error("model request failed", "stage", "connect", "error", e, "duration_ms", time.Since(started).Milliseconds(), "trace_id", domain.TraceID(ctx))
		return assistant.Response{}, &domain.ModelError{Message: "The model could not connect. Try again."}
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		snippet, _ := io.ReadAll(io.LimitReader(res.Body, 512))
		slog.Error("model request failed", "stage", "status", "status", res.StatusCode, "body", string(snippet), "duration_ms", time.Since(started).Milliseconds(), "trace_id", domain.TraceID(ctx))
		return assistant.Response{}, &domain.ModelError{Message: fmt.Sprintf("The model returned an error (%d). Try again.", res.StatusCode)}
	}
	var wire struct {
		Choices []struct {
			FinishReason string `json:"finish_reason"`
			Message      struct {
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
		slog.Error("model reply rejected", "reason", "unreadable", "error", e, "status", res.StatusCode, "duration_ms", time.Since(started).Milliseconds())
		return assistant.Response{}, &domain.ModelError{Message: "The model reply was unreadable. Try again."}
	}
	if len(wire.Choices) != 1 {
		slog.Error("model reply rejected", "reason", "no_choice", "choices", len(wire.Choices))
		return assistant.Response{}, &domain.ModelError{Message: "The model sent no reply. Try again."}
	}
	if wire.Choices[0].FinishReason == "length" {
		slog.Error("model reply rejected", "reason", "truncated", "finish_reason", "length", "max_tokens", maxOutputTokens, "usage", wire.Usage)
		return assistant.Response{}, &domain.ModelError{Message: "The model reply was cut short. Try again."}
	}
	cost, ok := wire.Usage["cost"].(float64)
	if !ok || cost < 0 {
		slog.Error("model reply rejected", "reason", "cost_missing", "usage", wire.Usage)
		return assistant.Response{}, &domain.ModelError{Message: "The model cost is missing. Try again later."}
	}
	out := assistant.Response{CostUSD: cost, RawUsage: wire.Usage}
	if wire.Choices[0].Message.Content != nil {
		out.Text = *wire.Choices[0].Message.Content
	}
	var names []string
	for _, c := range wire.Choices[0].Message.Calls {
		out.Calls = append(out.Calls, assistant.Call{ID: c.ID, Name: c.Function.Name, Arguments: c.Function.Arguments})
		names = append(names, c.Function.Name)
	}
	slog.Info("model response", "status", res.StatusCode, "finish_reason", wire.Choices[0].FinishReason, "tool_calls", strings.Join(names, ","), "content_len", len(out.Text), "cost_usd", cost, "prompt_tokens", wire.Usage["prompt_tokens"], "completion_tokens", wire.Usage["completion_tokens"], "total_tokens", wire.Usage["total_tokens"], "duration_ms", time.Since(started).Milliseconds(), "trace_id", domain.TraceID(ctx))
	return out, nil
}
func hostOf(raw string) string {
	if u, e := neturl.Parse(raw); e == nil {
		return u.Host
	}
	return ""
}
func tools() []any {
	str := map[string]string{"type": "string"}
	flag := map[string]string{"type": "boolean"}
	props := map[string]any{"title": str, "due_at": map[string]string{"type": "string", "description": "The deadline, RFC3339 with timezone; omit if unknown"}, "timezone": str, "notes": str, "item_id": str, "expected_version": map[string]string{"type": "integer"}, "clear_due": flag, "clear_notes": flag}
	props["remind_at"] = map[string]string{"type": "string", "description": "Explicit reminder time in RFC3339. Set only when the owner asks for a reminder."}
	props["nag_minutes"] = map[string]string{"type": "integer", "description": "Minutes between nags, 1 to 1440. Use 15 by default for reminders. Zero disables nagging."}
	props["clear_reminder"], props["clear_nag"] = flag, flag
	props["list"] = map[string]string{"type": "string", "description": "List name. Reuse an existing list name."}
	props["clear_list"] = flag
	props["location"] = map[string]string{"type": "string", "description": "Where the item happens, for example an address or a place name."}
	props["clear_location"] = flag
	for name, description := range map[string]string{
		"phone":             "Contact phone number copied from the source. Keep it out of notes.",
		"email":             "Contact email address copied from the source. Keep it out of notes.",
		"website":           "Contact or booking website URL copied from the source. Keep it out of notes.",
		"booking_reference": "Booking or confirmation identifier as text. Preserve leading zeros and exact spelling. Keep it out of notes.",
	} {
		props[name] = map[string]string{"type": "string", "description": description}
		props["clear_"+name] = flag
	}

	props["start_at"] = map[string]string{"type": "string", "description": "When the item begins, RFC3339 with timezone. Use for appointments and meetings. Use end_at for the event end, due_at only for deadlines. For date-only bookings use start_date/end_date (exclusive checkout) or due_date. Do not invent times."}
	props["clear_start"] = flag
	for _, name := range []string{"kind", "end_at", "start_timezone", "end_timezone", "start_date", "end_date", "due_date"} {
		props[name] = str
	}
	for _, name := range []string{"clear_end", "clear_start_date", "clear_end_date", "clear_due_date", "clear_start_timezone", "clear_end_timezone"} {
		props[name] = flag
	}
	props["duration_minutes"] = map[string]string{"type": "integer", "description": "How long it takes, 1 to 10080 minutes."}
	props["clear_duration"] = flag
	props["priority"] = map[string]string{"type": "integer", "description": "1 (highest) to 5. Set only when the owner states importance."}
	props["clear_priority"] = flag
	props["profile"] = map[string]string{"type": "string", "description": "Profile id from the profile list."}
	props["clear_profile"] = flag
	props["recurrence"] = map[string]any{"type": "object", "description": "Repeat rule. The item needs a due time.", "properties": map[string]any{"every": map[string]any{"type": "string", "enum": []string{"daily", "weekly", "monthly", "yearly"}}, "interval": map[string]string{"type": "integer"}, "after_completion": flag}, "required": []string{"every", "interval"}, "additionalProperties": false}
	props["clear_recurrence"] = flag
	out := []any{}
	for _, n := range []string{"propose_create", "propose_update", "propose_complete", "propose_cancel", "propose_delete"} {
		required := []string{"title"}
		if n != "propose_create" {
			required = []string{"item_id", "expected_version"}
		}
		out = append(out, tool(n, "Propose an item change for owner confirmation. Never invent item IDs.", props, required))
	}
	profile := map[string]any{"profile_id": map[string]string{"type": "string", "description": "Omit to add a new profile."}, "expected_version": map[string]string{"type": "integer"}, "profile_name": str, "profile_remind_before_minutes": map[string]string{"type": "integer", "description": "Minutes before the due time, 0 to 20160."}, "profile_nag_minutes": map[string]string{"type": "integer", "description": "Minutes between nags, 0 to 1440."}}
	out = append(out, tool("propose_profile", "Propose a new profile or a change to one for owner confirmation.", profile, nil))
	out = append(out, tool("find_finished_items", "Look up done and cancelled items. Call it alone when the owner asks about finished items, or wants to cancel or delete one that is not in the current list. You then get the matches and answer.", map[string]any{"query": str, "list": str}, nil))
	out = append(out, tool("list_items", "List open items, optionally for one list.", map[string]any{"list": str}, nil))
	return out
}
func tool(name, description string, props map[string]any, required []string) any {
	if required == nil {
		required = []string{}
	}
	return map[string]any{"type": "function", "function": map[string]any{"name": name, "description": description, "parameters": map[string]any{"type": "object", "properties": props, "required": required, "additionalProperties": false}}}
}

func captureTools() []any {
	base := tools()[0].(map[string]any)["function"].(map[string]any)["parameters"].(map[string]any)["properties"].(map[string]any)
	props := map[string]any{}
	for k, v := range base {
		props[k] = v
	}
	props["action"] = map[string]any{"type": "string", "enum": []string{"create_item", "update_item", "cancel_item", "complete_item"}}
	props["reminder_offsets_minutes"] = map[string]any{"type": "array", "description": "Reminders before the start or deadline, in minutes. Choose from 10, 30, 60, 120, 1440, 2880, 4320, 10080. Omit when the item has no start or deadline.", "items": map[string]any{"type": "integer", "enum": domain.ReminderOffsetChoices}}
	proposal := map[string]any{"type": "object", "properties": props, "required": []string{"action"}, "additionalProperties": false}
	evidence := map[string]any{"type": "object", "properties": map[string]any{"part": map[string]string{"type": "string"}, "excerpt": map[string]string{"type": "string"}}, "required": []string{"part", "excerpt"}, "additionalProperties": false}
	options := map[string]any{"type": "array", "maxItems": 3, "description": "Up to three short quick replies for the question.", "items": map[string]string{"type": "string"}}
	question := map[string]any{"type": "object", "properties": map[string]any{"text": map[string]string{"type": "string"}, "options": options}, "required": []string{"text"}, "additionalProperties": false}
	candidate := map[string]any{"type": "object", "properties": map[string]any{"id": map[string]string{"type": "string"}, "proposal": proposal, "question": map[string]string{"type": "string"}, "options": options, "evidence": map[string]any{"type": "array", "items": evidence}}, "required": []string{"proposal", "evidence"}, "additionalProperties": false}
	return []any{tool("extract_capture", "Extract source-backed candidates or ask focused questions. Never save changes.", map[string]any{"candidates": map[string]any{"type": "array", "items": candidate}, "questions": map[string]any{"type": "array", "items": question}}, []string{"candidates", "questions"})}
}
