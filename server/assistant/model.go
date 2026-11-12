package assistant

import "context"

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
