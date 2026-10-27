package metrics

import (
	"io"
	"log/slog"
	"net/http"
)

// Registry holds every group of metrics in keystone, so they can all be
// written out together.
type Registry struct {
	CommandStats     *CommandStats
	PersistenceStats *PersistenceStats
}

func NewRegistry() *Registry {
	return &Registry{
		CommandStats:     NewCommandStats(),
		PersistenceStats: NewPersistenceStats(),
	}
}

// WritePrometheus writes every group in the Prometheus text format, one
// after another.
func (r *Registry) WritePrometheus(writer io.Writer) error {
	err := r.CommandStats.WritePrometheus(writer)
	if err != nil {
		return err
	}
	return r.PersistenceStats.WritePrometheus(writer)
}

// HandleMetrics is the HTTP handler that answers with every metric, for
// Prometheus.
func (r *Registry) HandleMetrics(writer http.ResponseWriter, request *http.Request) {
	// This is the content type of the Prometheus text format.
	writer.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	err := r.WritePrometheus(writer)
	if err != nil {
		slog.Error("could not write the metrics", "error", err)
	}
}
