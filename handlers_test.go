package main

import (
	"bufio"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/viminizer/keystone/internal/aof"
	"github.com/viminizer/keystone/internal/resp"
	"github.com/viminizer/keystone/internal/store"
)

func TestExecute(t *testing.T) {
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
		executor, _ := newTestExecutor(t)
		got := executor.Execute(test.command)
		if got != test.want {
			t.Errorf("Execute(%q) = %q, want %q", test.command, got, test.want)
		}
	}
}

func TestSetThenGet(t *testing.T) {
	executor, _ := newTestExecutor(t)

	got := executor.Execute([]string{"SET", "name", "kevin"})
	if got != "+OK\r\n" {
		t.Fatalf("SET = %q", got)
	}

	got = executor.Execute([]string{"get", "name"})
	if got != "$5\r\nkevin\r\n" {
		t.Fatalf("GET = %q", got)
	}
}

func TestDel(t *testing.T) {
	executor, _ := newTestExecutor(t)
	executor.Execute([]string{"SET", "a", "1"})
	executor.Execute([]string{"SET", "b", "2"})

	got := executor.Execute([]string{"DEL", "a", "b", "c"})
	if got != ":2\r\n" {
		t.Fatalf("DEL a b c = %q, want 2 deleted", got)
	}

	got = executor.Execute([]string{"DEL", "a"})
	if got != ":0\r\n" {
		t.Fatalf("DEL a again = %q, want 0 deleted", got)
	}

	got = executor.Execute([]string{"DEL"})
	if got != "-ERR wrong number of arguments for 'del' command\r\n" {
		t.Fatalf("DEL with no key = %q", got)
	}
}

func TestExpireAndTTL(t *testing.T) {
	executor, _ := newTestExecutor(t)
	executor.Execute([]string{"SET", "name", "kevin"})

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
		got := executor.Execute(test.command)
		if got != test.want {
			t.Errorf("Execute(%q) = %q, want %q", test.command, got, test.want)
		}
	}
}

func TestSetWithEX(t *testing.T) {
	executor, _ := newTestExecutor(t)

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
		got := executor.Execute(test.command)
		if got != test.want {
			t.Errorf("Execute(%q) = %q, want %q", test.command, got, test.want)
		}
	}
}

func TestPexpireat(t *testing.T) {
	executor, _ := newTestExecutor(t)
	executor.Execute([]string{"SET", "name", "kevin"})
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
		got := executor.Execute(test.command)
		if got != test.want {
			t.Errorf("Execute(%q) = %q, want %q", test.command, got, test.want)
		}
	}
}

func TestWriteCommandsAreLogged(t *testing.T) {
	executor, path := newTestExecutor(t)

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
		executor.Execute(command)
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
	executor, path := newTestExecutor(t)
	executor.Execute([]string{"SET", "name", "kevin"})

	before := time.Now().Add(10 * time.Second).UnixMilli()
	executor.Execute([]string{"EXPIRE", "name", "10"})
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

func TestRestartKeepsData(t *testing.T) {
	before, path := newTestExecutor(t)
	before.Execute([]string{"SET", "name", "kevin"})
	before.Execute([]string{"SET", "session", "abc", "EX", "100"})
	before.Execute([]string{"SET", "old", "x"})
	before.Execute([]string{"DEL", "old"})

	after := restart(t, path)

	tests := []struct {
		command []string
		want    string
	}{
		{[]string{"GET", "name"}, "$5\r\nkevin\r\n"},
		{[]string{"GET", "session"}, "$3\r\nabc\r\n"},
		{[]string{"TTL", "session"}, ":100\r\n"},
		{[]string{"GET", "old"}, "$-1\r\n"},
	}
	for _, test := range tests {
		got := after.Execute(test.command)
		if got != test.want {
			t.Errorf("after restart, Execute(%q) = %q, want %q", test.command, got, test.want)
		}
	}
}

func TestRestartDropsExpiredKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "appendonly.aof")
	aofLog, _ := aof.Open(path)
	aofLog.Append([]string{"SET", "name", "kevin"})
	aSecondAgo := strconv.FormatInt(time.Now().Add(-time.Second).UnixMilli(), 10)
	aofLog.Append([]string{"PEXPIREAT", "name", aSecondAgo})

	after := restart(t, path)

	if got := after.Execute([]string{"GET", "name"}); got != "$-1\r\n" {
		t.Errorf("GET name after restart = %q, want the key to be expired", got)
	}
}

func TestReplayCommand(t *testing.T) {
	db := store.New()
	commands := [][]string{
		{"SET", "a", "1"},
		{"SET", "b", "2"},
		{"set", "c", "3"},
		// Older logs can have the name as the client sent it.
		{"del", "b"},
		// These are skipped, and must not crash.
		{},
		{"SET", "d"},
		{"PEXPIREAT", "a"},
		{"PEXPIREAT", "a", "soon"},
		{"PING"},
	}
	for _, command := range commands {
		replayCommand(db, command)
	}

	for key, want := range map[string]bool{"a": true, "b": false, "c": true, "d": false} {
		_, found := db.Get(key)
		if found != want {
			t.Errorf("key %q found = %v, want %v", key, found, want)
		}
	}
	if expiresAt, _ := db.ExpiresAt("a"); !expiresAt.IsZero() {
		t.Errorf("key a expires at %v, want no expiry", expiresAt)
	}
}

func TestRewrite(t *testing.T) {
	executor, path := newTestExecutor(t)
	executor.Execute([]string{"SET", "name", "kevin"})
	executor.Execute([]string{"SET", "name", "alex"})
	executor.Execute([]string{"SET", "name", "sam"})
	executor.Execute([]string{"SET", "temp", "x"})
	executor.Execute([]string{"DEL", "temp"})
	executor.Execute([]string{"SET", "session", "abc", "EX", "100"})

	executor.engine.rewrite()
	// A write after the rewrite must go into the new file.
	executor.Execute([]string{"SET", "after", "yes"})

	logged, _ := aof.Load(path, func(command []string) {})
	if logged != 4 {
		t.Fatalf("logged %d commands, want 4", logged)
	}

	after := restart(t, path)
	tests := []struct {
		command []string
		want    string
	}{
		{[]string{"GET", "name"}, "$3\r\nsam\r\n"},
		{[]string{"GET", "temp"}, "$-1\r\n"},
		{[]string{"GET", "session"}, "$3\r\nabc\r\n"},
		{[]string{"TTL", "session"}, ":100\r\n"},
		{[]string{"GET", "after"}, "$3\r\nyes\r\n"},
	}
	for _, test := range tests {
		got := after.Execute(test.command)
		if got != test.want {
			t.Errorf("after rewrite and load, Execute(%q) = %q, want %q", test.command, got, test.want)
		}
	}
}

func TestBgrewriteaofOneAtATime(t *testing.T) {
	executor, _ := newTestExecutor(t)
	// Act as if a rewrite is already running.
	executor.engine.rewriting.Store(true)

	got := executor.Execute([]string{"BGREWRITEAOF"})
	if got != "-ERR Background append only file rewriting already in progress\r\n" {
		t.Errorf("BGREWRITEAOF during a rewrite = %q", got)
	}
}

// newTestExecutor returns an executor with an empty store and a new log in
// a temporary folder, and the path of that log.
func newTestExecutor(t *testing.T) (*Executor, string) {
	path := filepath.Join(t.TempDir(), "appendonly.aof")
	aofLog, err := aof.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	return NewExecutor(NewEngine(store.New(), aofLog)), path
}

// restart loads the log at path into a new, empty store, like keystone does
// at start, and returns an executor for that store.
func restart(t *testing.T, path string) *Executor {
	db := store.New()
	aof.Load(path, func(command []string) {
		replayCommand(db, command)
	})
	aofLog, err := aof.Open(filepath.Join(t.TempDir(), "appendonly.aof"))
	if err != nil {
		t.Fatal(err)
	}
	return NewExecutor(NewEngine(db, aofLog))
}
