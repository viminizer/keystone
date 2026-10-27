package server

import (
	"io"
	"net"
	"path/filepath"
	"testing"

	"github.com/viminizer/keystone/internal/aof"
	"github.com/viminizer/keystone/internal/engine"
	"github.com/viminizer/keystone/internal/executor"
	"github.com/viminizer/keystone/internal/metrics"
	"github.com/viminizer/keystone/internal/store"
)

func TestHandleConnection(t *testing.T) {
	aofLog, err := aof.Open(filepath.Join(t.TempDir(), "appendonly.aof"))
	if err != nil {
		t.Fatal(err)
	}
	server := New(0, executor.New(engine.New(store.New(), aofLog, metrics.NewPersistenceStats()), metrics.NewCommandStats()))
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
