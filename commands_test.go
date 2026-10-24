package main

import (
	"testing"

	"github.com/viminizer/keystone/store"
)

func TestRunCommand(t *testing.T) {
	tests := []struct {
		command []string
		want    string
	}{
		{[]string{"PING"}, "+PONG\r\n"},
		{[]string{"ping"}, "+PONG\r\n"},
		{[]string{"foo"}, "-ERR unknown command 'foo'\r\n"},
		{[]string{}, ""},
		{[]string{"GET", "name"}, "$-1\r\n"},
		{[]string{"GET"}, "-ERR wrong number of arguments for 'get' command\r\n"},
		{[]string{"SET", "name"}, "-ERR wrong number of arguments for 'set' command\r\n"},
	}

	for _, test := range tests {
		got := runCommand(test.command, store.New())
		if got != test.want {
			t.Errorf("runCommand(%q) = %q, want %q", test.command, got, test.want)
		}
	}
}

func TestSetThenGet(t *testing.T) {
	db := store.New()

	got := runCommand([]string{"SET", "name", "kevin"}, db)
	if got != "+OK\r\n" {
		t.Fatalf("SET = %q", got)
	}

	got = runCommand([]string{"get", "name"}, db)
	if got != "$5\r\nkevin\r\n" {
		t.Fatalf("GET = %q", got)
	}
}
