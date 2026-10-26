// Package httpserver runs a plain HTTP server. Other packages give it the
// routes and the handlers, so it knows nothing about what it serves.
package httpserver

import (
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"

	"github.com/viminizer/keystone/internal/config"
)

// Server is an HTTP server with routes added by Get and Post.
type Server struct {
	enabled bool
	port    int
	routes  *http.ServeMux
}

// New makes an HTTP server from the settings. If HTTP is turned off in the
// settings, MustStart does nothing.
func New(settings *config.Config) *Server {
	return &Server{
		enabled: settings.HTTPEnabled,
		port:    settings.HTTPPort,
		routes:  http.NewServeMux(),
	}
}

// Get adds a route for GET requests. The path has no slash at the start,
// like "metrics". It returns the server, so calls can be chained.
func (s *Server) Get(path string, handler func(http.ResponseWriter, *http.Request)) *Server {
	s.routes.HandleFunc("GET /"+path, handler)
	return s
}

// Post adds a route for POST requests, the same way as Get.
func (s *Server) Post(path string, handler func(http.ResponseWriter, *http.Request)) *Server {
	s.routes.HandleFunc("POST /"+path, handler)
	return s
}

// MustStart opens the port and serves the routes in the background. It
// opens the port before it returns, so if the port is taken, keystone
// stops right at start. Then it logs the error and exits.
func (s *Server) MustStart() {
	if !s.enabled {
		slog.Info("the http server is turned off")
		return
	}

	listener, err := net.Listen("tcp", ":"+strconv.Itoa(s.port))
	if err != nil {
		slog.Error("could not start the http server", "error", err)
		os.Exit(1)
	}
	slog.Info("the http server is listening", "port", s.port)

	go func() {
		// Serve only returns if the server breaks.
		err := http.Serve(listener, s.routes)
		slog.Error("the http server stopped", "error", err)
	}()
}
