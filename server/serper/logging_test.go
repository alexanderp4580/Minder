package serper

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/alexanderp4580/Minder/server/assistant"
)

// logsOf runs fn and returns its JSON log lines by message.
func logsOf(t *testing.T, fn func()) map[string]map[string]any {
	t.Helper()
	var buf bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(previous) })
	fn()
	for _, private := range []string{"SnowWorld", "serper-secret", "Witte"} {
		if strings.Contains(buf.String(), private) {
			t.Fatalf("private text %q in logs: %s", private, buf.String())
		}
	}
	lines := map[string]map[string]any{}
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		var r map[string]any
		if json.Unmarshal([]byte(line), &r) == nil {
			lines[r["msg"].(string)] = r
		}
	}
	return lines
}

func TestLookupLogsKindCountsAndCreditsNotQueryText(t *testing.T) {
	client, _ := serperAt(t, 200, mapsBody)
	lines := logsOf(t, func() { client.Lookup(context.Background(), assistant.LookupMaps, "SnowWorld Landgraaf") })
	line := lines["serper lookup"]
	if line == nil || line["kind"] != "maps" || line["query_len"] != float64(19) || line["results"] != float64(3) || line["status"] != float64(200) || line["credits"] != float64(3) {
		t.Fatalf("log line %v", line)
	}
	if _, ok := line["duration_ms"]; !ok {
		t.Fatal("no duration_ms")
	}
}

func TestFailedLookupLogsAWarningWithoutPrivateText(t *testing.T) {
	client, _ := serperAt(t, 500, `{"message":"serper-secret"}`)
	lines := logsOf(t, func() { client.Lookup(context.Background(), assistant.LookupSearch, "SnowWorld Landgraaf") })
	line := lines["serper lookup failed"]
	if line == nil || line["level"] != "WARN" || line["kind"] != "search" || line["status"] != float64(500) {
		t.Fatalf("log line %v", line)
	}
}
