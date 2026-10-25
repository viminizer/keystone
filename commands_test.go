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

func TestDel(t *testing.T) {
	db := store.New()
	runCommand([]string{"SET", "a", "1"}, db)
	runCommand([]string{"SET", "b", "2"}, db)

	got := runCommand([]string{"DEL", "a", "b", "c"}, db)
	if got != ":2\r\n" {
		t.Fatalf("DEL a b c = %q, want 2 deleted", got)
	}

	got = runCommand([]string{"DEL", "a"}, db)
	if got != ":0\r\n" {
		t.Fatalf("DEL a again = %q, want 0 deleted", got)
	}

	got = runCommand([]string{"DEL"}, db)
	if got != "-ERR wrong number of arguments for 'del' command\r\n" {
		t.Fatalf("DEL with no key = %q", got)
	}
}

func TestExpireAndTTL(t *testing.T) {
	db := store.New()
	runCommand([]string{"SET", "name", "kevin"}, db)

	tests := []struct {
		command []string
		want    string
	}{
		{[]string{"TTL", "name"}, ":-1\r\n"},
		{[]string{"TTL", "missing"}, ":-2\r\n"},
		{[]string{"EXPIRE", "name", "10"}, ":1\r\n"},
		{[]string{"TTL", "name"}, ":10\r\n"},
		{[]string{"EXPIRE", "missing", "10"}, ":0\r\n"},
		{[]string{"EXPIRE", "name", "ten"}, "-ERR value is not an integer or out of range\r\n"},
		{[]string{"EXPIRE", "name"}, "-ERR wrong number of arguments for 'expire' command\r\n"},
		{[]string{"EXPIRE", "name", "-1"}, ":1\r\n"},
		{[]string{"GET", "name"}, "$-1\r\n"},
		{[]string{"TTL", "name"}, ":-2\r\n"},
	}

	// The cases run in order on the same store, so each one sees the one before.
	for _, test := range tests {
		got := runCommand(test.command, db)
		if got != test.want {
			t.Errorf("runCommand(%q) = %q, want %q", test.command, got, test.want)
		}
	}
}
