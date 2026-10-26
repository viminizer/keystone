package logging

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestHandler(t *testing.T) {
	var output bytes.Buffer
	options := &slog.HandlerOptions{Level: slog.LevelInfo}
	logger := slog.New(&handler{next: slog.NewJSONHandler(&output, options)})

	logger.Debug("hidden")
	logger.With("service", "keystone").Info("client connected", "client", "127.0.0.1:5000")

	got := output.String()
	if strings.Contains(got, "hidden") {
		t.Errorf("a debug line was logged at info level: %s", got)
	}
	for _, want := range []string{`"msg":"client connected"`, `"client":"127.0.0.1:5000"`, `"service":"keystone"`} {
		if !strings.Contains(got, want) {
			t.Errorf("log %s does not have %s", got, want)
		}
	}
}
