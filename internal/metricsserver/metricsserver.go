// Package metricsserver serves keystone's metrics over HTTP, for
// Prometheus.
package metricsserver

import (
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"

	"github.com/viminizer/keystone/internal/metrics"
)

// Server answers GET /metrics with the metrics in the Prometheus text
// format.
type Server struct {
	enabled  bool
	port     int
	commands *metrics.Commands
}

// New makes a metrics server. If enabled is false, MustStart does nothing.
func New(enabled bool, port int, commandMetrics *metrics.Commands) *Server {
	return &Server{enabled: enabled, port: port, commands: commandMetrics}
}

// MustStart opens the port and serves the metrics in the background. It
// opens the port before it returns, so if the port is taken, keystone
// stops right at start. Then it logs the error and exits.
func (s *Server) MustStart() {
	if !s.enabled {
		slog.Info("metrics are turned off")
		return
	}

	listener, err := net.Listen("tcp", ":"+strconv.Itoa(s.port))
	if err != nil {
		slog.Error("could not start the metrics server", "error", err)
		os.Exit(1)
	}
	slog.Info("serving metrics", "port", s.port, "path", "/metrics")

	routes := http.NewServeMux()
	routes.HandleFunc("GET /metrics", s.handleMetrics)
	go func() {
		// Serve only returns if the server breaks.
		err := http.Serve(listener, routes)
		slog.Error("the metrics server stopped", "error", err)
	}()
}

// handleMetrics writes every metric as the reply to one request.
func (s *Server) handleMetrics(writer http.ResponseWriter, request *http.Request) {
	// This is the content type of the Prometheus text format.
	writer.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	err := s.commands.WritePrometheus(writer)
	if err != nil {
		slog.Error("could not write the metrics", "error", err)
	}
}
