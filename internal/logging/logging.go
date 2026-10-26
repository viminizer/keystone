// Package logging sets up the logger that every slog call in keystone uses.
package logging

import (
	"context"
	"log/slog"
	"os"

	"github.com/viminizer/keystone/internal/config"
)

// Setup makes every slog call in keystone write to stderr, in the format and
// from the level in the settings.
func Setup(settings *config.Config) {
	options := &slog.HandlerOptions{Level: settings.LogLevel}
	var output slog.Handler
	if settings.LogFormat == "json" {
		output = slog.NewJSONHandler(os.Stderr, options)
	} else {
		output = slog.NewTextHandler(os.Stderr, options)
	}
	slog.SetDefault(slog.New(&handler{next: output}))
}

// handler wraps the text or json handler. Every log line in keystone goes
// through it, so it is the one place to change how logs are handled. For
// now it only passes everything on to next.
type handler struct {
	next slog.Handler
}

// Enabled says if a line at this level is logged at all.
func (h *handler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

// Handle writes one log line.
func (h *handler) Handle(ctx context.Context, record slog.Record) error {
	return h.next.Handle(ctx, record)
}

// WithAttrs returns a handler that adds the given fields to every line.
func (h *handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &handler{next: h.next.WithAttrs(attrs)}
}

// WithGroup returns a handler that puts the fields that come after it under
// the given name.
func (h *handler) WithGroup(name string) slog.Handler {
	return &handler{next: h.next.WithGroup(name)}
}
