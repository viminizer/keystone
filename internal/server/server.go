// Package server accepts clients over TCP and answers their commands.
package server

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"strconv"

	"github.com/viminizer/keystone/internal/executor"
	"github.com/viminizer/keystone/internal/resp"
)

// Server accepts clients on a TCP port, reads their commands and sends the
// replies back.
type Server struct {
	port     int
	executor *executor.Executor
}

func New(port int, commandExecutor *executor.Executor) *Server {
	return &Server{port: port, executor: commandExecutor}
}

// MustListenAndServe works like ListenAndServe, but if the port could not be
// opened, it logs the error and exits.
func (s *Server) MustListenAndServe() {
	err := s.ListenAndServe()
	slog.Error("could not start the server", "error", err)
	os.Exit(1)
}

// ListenAndServe opens the port and serves every client in its own
// goroutine. It only returns if the port could not be opened.
func (s *Server) ListenAndServe() error {
	listener, err := net.Listen("tcp", ":"+strconv.Itoa(s.port))
	if err != nil {
		return fmt.Errorf("could not listen on port %d: %w", s.port, err)
	}

	for {
		connection, err := listener.Accept()
		if err != nil {
			slog.Error("could not accept a connection", "error", err)
			continue
		}

		go s.handleConnection(connection)
	}
}

// handleConnection runs the commands of one client until it disconnects.
func (s *Server) handleConnection(connection net.Conn) {
	address := connection.RemoteAddr().String()
	slog.Info("client connected", "client", address)

	reader := bufio.NewReader(connection)
	for {
		command, err := resp.ReadCommand(reader)
		if err == io.EOF {
			break
		}
		if err != nil {
			slog.Error("could not read from the client", "client", address, "error", err)
			break
		}

		slog.Debug("received a command", "client", address, "command", command)

		reply := s.executor.Execute(command)
		if reply == "" {
			continue
		}
		_, err = connection.Write([]byte(reply))
		if err != nil {
			slog.Error("could not write to the client", "client", address, "error", err)
			break
		}
	}

	connection.Close()
}
