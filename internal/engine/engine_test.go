package engine

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/viminizer/keystone/internal/aof"
	"github.com/viminizer/keystone/internal/store"
)

func TestShouldRewrite(t *testing.T) {
	const megabyte = 1024 * 1024
	tests := []struct {
		size     int64
		baseSize int64
		want     bool
	}{
		{size: 100 * megabyte, baseSize: 50 * megabyte, want: true},
		{size: 99 * megabyte, baseSize: 50 * megabyte, want: false},
		{size: 64 * megabyte, baseSize: 0, want: true},
		{size: 63 * megabyte, baseSize: 0, want: false},
		{size: 10 * megabyte, baseSize: 1 * megabyte, want: false},
	}

	for _, test := range tests {
		got := shouldRewrite(test.size, test.baseSize)
		if got != test.want {
			t.Errorf("shouldRewrite(%d, %d) = %v, want %v", test.size, test.baseSize, got, test.want)
		}
	}
}

func TestReplayCommand(t *testing.T) {
	storage := store.New()
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
		replayCommand(storage, command)
	}

	for key, want := range map[string]bool{"a": true, "b": false, "c": true, "d": false} {
		_, found := storage.Get(key)
		if found != want {
			t.Errorf("key %q found = %v, want %v", key, found, want)
		}
	}
	if expiresAt, _ := storage.ExpiresAt("a"); !expiresAt.IsZero() {
		t.Errorf("key a expires at %v, want no expiry", expiresAt)
	}
}

func TestRewrite(t *testing.T) {
	db, path := newTestEngine(t)
	db.Set("name", "kevin")
	db.Set("name", "alex")
	db.Set("name", "sam")
	db.Set("temp", "x")
	db.Delete([]string{"temp"})
	db.SetWithExpiry("session", "abc", time.Now().Add(100*time.Second))

	db.rewrite()
	// A write after the rewrite must go into the new file.
	db.Set("after", "yes")

	logged, _ := aof.Load(path, func(command []string) {})
	if logged != 4 {
		t.Fatalf("logged %d commands, want 4", logged)
	}

	after, err := Start(path)
	if err != nil {
		t.Fatal(err)
	}
	values := map[string]string{"name": "sam", "session": "abc", "after": "yes"}
	for key, want := range values {
		got, _ := after.Get(key)
		if got != want {
			t.Errorf("after rewrite and load, %s = %q, want %q", key, got, want)
		}
	}
	if _, found := after.Get("temp"); found {
		t.Error("after rewrite and load, temp still exists")
	}
	expiresAt, _ := after.ExpiresAt("session")
	if secondsLeft := time.Until(expiresAt).Round(time.Second); secondsLeft != 100*time.Second {
		t.Errorf("after rewrite and load, session expires in %v, want 100s", secondsLeft)
	}
}

func TestStartRewriteOneAtATime(t *testing.T) {
	db, _ := newTestEngine(t)
	// Act as if a rewrite is already running.
	db.rewriting.Store(true)

	if db.StartRewrite() {
		t.Error("StartRewrite started a second rewrite")
	}
}

// newTestEngine returns an engine with an empty store and a new log in a
// temporary folder, and the path of that log.
func newTestEngine(t *testing.T) (*Engine, string) {
	path := filepath.Join(t.TempDir(), "appendonly.aof")
	aofLog, err := aof.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	return New(store.New(), aofLog), path
}
