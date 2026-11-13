package serper

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

func TestFindPlacesLogsCountsNotQueryText(t *testing.T) {
	var buf bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(previous) })
	c := server(t, 200, fixture, nil, nil)
	if _, err := c.FindPlaces(context.Background(), "SnowWorld Landgraaf"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if strings.Contains(out, "SnowWorld") || strings.Contains(out, "serper-secret") || strings.Contains(out, "Witte") {
		t.Fatalf("private text in logs: %s", out)
	}
	var found map[string]any
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		var r map[string]any
		if json.Unmarshal([]byte(line), &r) == nil && r["msg"] == "place lookup" {
			found = r
		}
	}
	if found == nil || found["query_len"] != float64(19) || found["places"] != float64(3) || found["status"] != float64(200) {
		t.Fatalf("log line %v", found)
	}
	if _, ok := found["duration_ms"]; !ok {
		t.Fatal("no duration_ms")
	}
}
