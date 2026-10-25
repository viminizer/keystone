package main

import (
	"bufio"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/viminizer/keystone/aof"
	"github.com/viminizer/keystone/resp"
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
		got := runCommand(test.command, store.New(), nil)
		if got != test.want {
			t.Errorf("runCommand(%q) = %q, want %q", test.command, got, test.want)
		}
	}
}

func TestSetThenGet(t *testing.T) {
	db := store.New()

	got := runCommand([]string{"SET", "name", "kevin"}, db, nil)
	if got != "+OK\r\n" {
		t.Fatalf("SET = %q", got)
	}

	got = runCommand([]string{"get", "name"}, db, nil)
	if got != "$5\r\nkevin\r\n" {
		t.Fatalf("GET = %q", got)
	}
}

func TestDel(t *testing.T) {
	db := store.New()
	runCommand([]string{"SET", "a", "1"}, db, nil)
	runCommand([]string{"SET", "b", "2"}, db, nil)

	got := runCommand([]string{"DEL", "a", "b", "c"}, db, nil)
	if got != ":2\r\n" {
		t.Fatalf("DEL a b c = %q, want 2 deleted", got)
	}

	got = runCommand([]string{"DEL", "a"}, db, nil)
	if got != ":0\r\n" {
		t.Fatalf("DEL a again = %q, want 0 deleted", got)
	}

	got = runCommand([]string{"DEL"}, db, nil)
	if got != "-ERR wrong number of arguments for 'del' command\r\n" {
		t.Fatalf("DEL with no key = %q", got)
	}
}

func TestExpireAndTTL(t *testing.T) {
	db := store.New()
	runCommand([]string{"SET", "name", "kevin"}, db, nil)

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
		got := runCommand(test.command, db, nil)
		if got != test.want {
			t.Errorf("runCommand(%q) = %q, want %q", test.command, got, test.want)
		}
	}
}

func TestSetWithEX(t *testing.T) {
	db := store.New()

	tests := []struct {
		command []string
		want    string
	}{
		{[]string{"SET", "name", "kevin", "EX", "10"}, "+OK\r\n"},
		{[]string{"TTL", "name"}, ":10\r\n"},
		{[]string{"GET", "name"}, "$5\r\nkevin\r\n"},
		{[]string{"SET", "name", "alex", "ex", "20"}, "+OK\r\n"},
		{[]string{"TTL", "name"}, ":20\r\n"},
		{[]string{"SET", "name", "kevin"}, "+OK\r\n"},
		{[]string{"TTL", "name"}, ":-1\r\n"},
		{[]string{"SET", "name", "kevin", "EX"}, "-ERR syntax error\r\n"},
		{[]string{"SET", "name", "kevin", "FOO", "10"}, "-ERR syntax error\r\n"},
		{[]string{"SET", "name", "kevin", "EX", "ten"}, "-ERR value is not an integer or out of range\r\n"},
		{[]string{"SET", "name", "kevin", "EX", "0"}, "-ERR invalid expire time in 'set' command\r\n"},
		{[]string{"SET", "name", "kevin", "EX", "-5"}, "-ERR invalid expire time in 'set' command\r\n"},
	}

	// The cases run in order on the same store, so each one sees the one before.
	for _, test := range tests {
		got := runCommand(test.command, db, nil)
		if got != test.want {
			t.Errorf("runCommand(%q) = %q, want %q", test.command, got, test.want)
		}
	}
}

func TestPexpireat(t *testing.T) {
	db := store.New()
	runCommand([]string{"SET", "name", "kevin"}, db, nil)
	inTenSeconds := strconv.FormatInt(time.Now().Add(10*time.Second).UnixMilli(), 10)
	aSecondAgo := strconv.FormatInt(time.Now().Add(-time.Second).UnixMilli(), 10)

	tests := []struct {
		command []string
		want    string
	}{
		{[]string{"PEXPIREAT", "name", inTenSeconds}, ":1\r\n"},
		{[]string{"TTL", "name"}, ":10\r\n"},
		{[]string{"PEXPIREAT", "missing", inTenSeconds}, ":0\r\n"},
		{[]string{"PEXPIREAT", "name", "soon"}, "-ERR value is not an integer or out of range\r\n"},
		{[]string{"PEXPIREAT", "name"}, "-ERR wrong number of arguments for 'pexpireat' command\r\n"},
		{[]string{"PEXPIREAT", "name", aSecondAgo}, ":1\r\n"},
		{[]string{"GET", "name"}, "$-1\r\n"},
	}

	// The cases run in order on the same store, so each one sees the one before.
	for _, test := range tests {
		got := runCommand(test.command, db, nil)
		if got != test.want {
			t.Errorf("runCommand(%q) = %q, want %q", test.command, got, test.want)
		}
	}
}

func TestWriteCommandsAreLogged(t *testing.T) {
	path := filepath.Join(t.TempDir(), "appendonly.aof")
	aofLog, err := aof.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	db := store.New()

	commands := [][]string{
		{"SET", "name", "kevin"},
		{"GET", "name"},
		{"SET", "session", "abc", "EX", "10"},
		{"DEL", "missing"},
		{"DEL", "name", "missing"},
		{"EXPIRE", "missing", "10"},
		{"EXPIRE", "session", "20"},
		{"SET", "name"},
		{"TTL", "session"},
	}
	for _, command := range commands {
		runCommand(command, db, aofLog)
	}

	file, _ := os.Open(path)
	reader := bufio.NewReader(file)
	var logged [][]string
	for {
		command, err := resp.ReadCommand(reader)
		if err != nil {
			break
		}
		logged = append(logged, command)
	}

	// The times in PEXPIREAT change on every run, so only the names and keys are checked.
	want := [][]string{
		{"SET", "name", "kevin"},
		{"SET", "session", "abc"},
		{"PEXPIREAT", "session"},
		{"DEL", "name", "missing"},
		{"PEXPIREAT", "session"},
	}
	if len(logged) != len(want) {
		t.Fatalf("logged %q, want %d commands", logged, len(want))
	}
	for i := range want {
		if !slices.Equal(logged[i][:len(want[i])], want[i]) {
			t.Errorf("command %d = %q, want it to start with %q", i, logged[i], want[i])
		}
	}
}

func TestExpireIsLoggedAsAMoment(t *testing.T) {
	path := filepath.Join(t.TempDir(), "appendonly.aof")
	aofLog, _ := aof.Open(path)
	db := store.New()
	runCommand([]string{"SET", "name", "kevin"}, db, aofLog)

	before := time.Now().Add(10 * time.Second).UnixMilli()
	runCommand([]string{"EXPIRE", "name", "10"}, db, aofLog)
	after := time.Now().Add(10 * time.Second).UnixMilli()

	file, _ := os.Open(path)
	reader := bufio.NewReader(file)
	resp.ReadCommand(reader)
	command, _ := resp.ReadCommand(reader)
	milliseconds, _ := strconv.ParseInt(command[2], 10, 64)
	if milliseconds < before || milliseconds > after {
		t.Errorf("logged %q, want a time between %d and %d", command, before, after)
	}
}
