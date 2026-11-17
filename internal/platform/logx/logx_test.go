package logx

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

func TestNewJSON(t *testing.T) {
	var buf bytes.Buffer
	New(&buf, slog.LevelInfo, "json").Info("hello", "user_id", 7)

	var line map[string]any
	if err := json.Unmarshal(buf.Bytes(), &line); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, buf.String())
	}
	if line["msg"] != "hello" || line["user_id"] != float64(7) {
		t.Errorf("unexpected log line: %v", line)
	}
}

func TestNewText(t *testing.T) {
	var buf bytes.Buffer
	New(&buf, slog.LevelInfo, "text").Info("hello", "user_id", 7)

	if out := buf.String(); !strings.Contains(out, "msg=hello") || !strings.Contains(out, "user_id=7") {
		t.Errorf("text output = %q, want key=value pairs", out)
	}
}

func TestNewDropsLowerLevels(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf, slog.LevelWarn, "json")
	logger.Info("hidden")
	logger.Warn("shown")

	if out := buf.String(); strings.Contains(out, "hidden") || !strings.Contains(out, "shown") {
		t.Errorf("level filter not applied, output: %s", out)
	}
}

func TestContextRoundTrip(t *testing.T) {
	l := New(&bytes.Buffer{}, slog.LevelInfo, "json")
	ctx := WithLogger(t.Context(), l)

	if got := FromContext(ctx); got != l {
		t.Error("FromContext() did not return the logger stored by WithLogger")
	}
	if got := FromContext(t.Context()); got != slog.Default() {
		t.Error("FromContext() on an empty context should return slog.Default()")
	}
}
