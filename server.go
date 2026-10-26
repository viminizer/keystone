package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strconv"

	"github.com/viminizer/keystone/resp"
)

// Server accepts clients on a TCP port, reads their commands and sends the
// replies back.
type Server struct {
	port     int
	executor *Executor
}

func NewServer(port int, executor *Executor) *Server {
	return &Server{port: port, executor: executor}
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
			fmt.Println("could not accept a connection:", err)
			continue
		}

		go s.handleConnection(connection)
	}
}

// handleConnection runs the commands of one client until it disconnects.
func (s *Server) handleConnection(connection net.Conn) {
	address := connection.RemoteAddr().String()
	fmt.Println("client connected from", address)

	reader := bufio.NewReader(connection)
	for {
		command, err := resp.ReadCommand(reader)
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Printf("could not read from %s: %s\n", address, err)
			break
		}

		fmt.Printf("received %q from %s\n", command, address)

		reply := s.executor.Execute(command)
		if reply == "" {
			continue
		}
		_, err = connection.Write([]byte(reply))
		if err != nil {
			fmt.Printf("could not write to %s: %s\n", address, err)
			break
		}
	}

	connection.Close()
}
