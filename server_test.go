package main

import (
	"io"
	"net"
	"testing"
)

func TestHandleConnection(t *testing.T) {
	executor, _ := newTestExecutor(t)
	server := NewServer(0, executor)
	// net.Pipe gives two connected ends that work in memory, without a port.
	client, serverEnd := net.Pipe()
	defer client.Close()
	go server.handleConnection(serverEnd)

	tests := []struct {
		request string
		want    string
	}{
		{"*1\r\n$4\r\nPING\r\n", "+PONG\r\n"},
		{"*3\r\n$3\r\nSET\r\n$4\r\nname\r\n$5\r\nkevin\r\n", "+OK\r\n"},
		{"*2\r\n$3\r\nGET\r\n$4\r\nname\r\n", "$5\r\nkevin\r\n"},
	}
	for _, test := range tests {
		client.Write([]byte(test.request))
		reply := make([]byte, len(test.want))
		_, err := io.ReadFull(client, reply)
		if err != nil {
			t.Fatalf("reading the reply to %q: %v", test.request, err)
		}
		if string(reply) != test.want {
			t.Errorf("reply to %q = %q, want %q", test.request, reply, test.want)
		}
	}
}
