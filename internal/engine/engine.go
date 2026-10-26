// Package engine runs the database operations. It keeps the store and the
// append-only file in step.
package engine

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/viminizer/keystone/internal/aof"
	"github.com/viminizer/keystone/internal/store"
)

// Engine runs the database operations. It changes the store and writes each
// change to the log.
type Engine struct {
	storage *store.Store
	aofLog  *aof.Log
	// writeMutex is held while a change goes to the store and to the log.
	// So the commands in the log are always in the same order as they
	// changed the store. Reads do not need it, because they write nothing
	// to the log.
	writeMutex sync.Mutex
	// rewriting is true while the log is being rewritten, so only one
	// rewrite runs at a time.
	rewriting atomic.Bool
}

func New(storage *store.Store, aofLog *aof.Log) *Engine {
	return &Engine{storage: storage, aofLog: aofLog}
}

// Open loads the saved changes from the append-only file into a new
// store, opens the file for new changes, and returns an engine for both.
func Open(aofPath string) (*Engine, error) {
	storage := store.New()

	// The saved changes are loaded before any client can connect, so no
	// client sees the data half loaded.
	loaded, err := aof.Load(aofPath, func(command []string) {
		replayCommand(storage, command)
	})
	if err != nil {
		return nil, fmt.Errorf("could not load the append-only file: %w", err)
	}
	slog.Info("loaded the append-only file", "path", aofPath, "commands", loaded)

	// The file is opened after loading, because loading can cut a broken
	// end off the file, and Open reads the size of the file.
	aofLog, err := aof.Open(aofPath)
	if err != nil {
		return nil, fmt.Errorf("could not open the append-only file: %w", err)
	}
	return New(storage, aofLog), nil
}

// MustOpen works like Open, but if that fails, it logs the error and
// exits.
func MustOpen(aofPath string) *Engine {
	engine, err := Open(aofPath)
	if err != nil {
		slog.Error("could not open the engine", "error", err)
		os.Exit(1)
	}
	return engine
}

func (e *Engine) Set(key string, value string) {
	e.writeMutex.Lock()
	defer e.writeMutex.Unlock()

	e.storage.Set(key, value)
	e.appendToLog([]string{"SET", key, value})
}

func (e *Engine) SetWithExpiry(key string, value string, at time.Time) {
	e.writeMutex.Lock()
	defer e.writeMutex.Unlock()

	e.storage.SetWithExpiry(key, value, at)
	e.appendToLog([]string{"SET", key, value})
	e.appendToLog([]string{"PEXPIREAT", key, unixMilli(at)})
}

func (e *Engine) Get(key string) (string, bool) {
	return e.storage.Get(key)
}

// Delete deletes the keys and returns how many of them existed.
func (e *Engine) Delete(keys []string) int {
	e.writeMutex.Lock()
	defer e.writeMutex.Unlock()

	deleted := 0
	for _, key := range keys {
		if e.storage.Delete(key) {
			deleted++
		}
	}
	if deleted > 0 {
		command := []string{"DEL"}
		command = append(command, keys...)
		e.appendToLog(command)
	}
	return deleted
}

// ExpireAt makes the key expire at the given time. It returns false if the
// key does not exist. The expiry is written to the log as PEXPIREAT,
// because a moment in time stays correct after a restart, and "10 seconds
// from now" does not.
func (e *Engine) ExpireAt(key string, at time.Time) bool {
	e.writeMutex.Lock()
	defer e.writeMutex.Unlock()

	if !e.storage.ExpireAt(key, at) {
		return false
	}
	e.appendToLog([]string{"PEXPIREAT", key, unixMilli(at)})
	return true
}

func (e *Engine) ExpiresAt(key string) (time.Time, bool) {
	return e.storage.ExpiresAt(key)
}

// StartRewrite rewrites the log in the background. It returns false if a
// rewrite is already running.
func (e *Engine) StartRewrite() bool {
	// CompareAndSwap sets rewriting to true only if it was false.
	if !e.rewriting.CompareAndSwap(false, true) {
		return false
	}
	go func() {
		e.rewrite()
		e.rewriting.Store(false)
	}()
	return true
}

// rewrite replaces the log with one SET for every key, and a PEXPIREAT for
// every key with an expiry. It holds writeMutex the whole time, so writes
// wait until it is done and none of them go into the old file.
// With many keys that wait gets long. Redis avoids it by writing the new
// file in a child process and buffering the writes that come in meanwhile.
func (e *Engine) rewrite() {
	e.writeMutex.Lock()
	defer e.writeMutex.Unlock()

	items := e.storage.Snapshot()
	commands := [][]string{}
	for _, item := range items {
		commands = append(commands, []string{"SET", item.Key, item.Value})
		if !item.ExpiresAt.IsZero() {
			commands = append(commands, []string{"PEXPIREAT", item.Key, unixMilli(item.ExpiresAt)})
		}
	}

	err := e.aofLog.Rewrite(commands)
	if err != nil {
		slog.Error("could not rewrite the append-only file", "error", err)
		return
	}
	slog.Info("rewrote the append-only file", "keys", len(items))
}

// SyncLog puts the log on the disk.
func (e *Engine) SyncLog() error {
	return e.aofLog.Sync()
}

// DeleteExpiredKeys deletes some of the expired keys that nobody reads. It
// does not need writeMutex, because it writes nothing to the log. After a
// restart, the PEXPIREAT in the log makes these keys expire again.
func (e *Engine) DeleteExpiredKeys() {
	e.storage.DeleteExpiredKeys()
}

// RewriteIfTooBig starts a rewrite when the log has grown too big.
func (e *Engine) RewriteIfTooBig() {
	size, baseSize := e.aofLog.Sizes()
	if shouldRewrite(size, baseSize) {
		e.StartRewrite()
	}
}

// The log is rewritten by itself when it is at least this big. Without it,
// a very small file would be rewritten all the time.
const autoRewriteMinSize = 64 * 1024 * 1024

// shouldRewrite says if the log should be rewritten: when it is twice as big
// as after the last rewrite, and at least autoRewriteMinSize. Redis uses the
// same rule.
func shouldRewrite(size int64, baseSize int64) bool {
	return size >= autoRewriteMinSize && size >= 2*baseSize
}

// Shutdown waits for a running write or rewrite to finish, and puts the log
// on the disk. It never lets go of writeMutex, so no write can happen after
// it. Call it only right before the program exits.
func (e *Engine) Shutdown() error {
	e.writeMutex.Lock()
	return e.aofLog.Sync()
}

// appendToLog writes one command to the log. If that fails, the change is
// still in memory, so the error is only logged.
func (e *Engine) appendToLog(command []string) {
	err := e.aofLog.Append(command)
	if err != nil {
		slog.Error("could not write to the append-only file", "error", err)
	}
}

// replayCommand applies one command from the log to the store. It runs at
// start, before any client connects, so it needs no lock and writes nothing
// to the log. The log only has SET, DEL and PEXPIREAT. Anything else, or a
// command with the wrong number of arguments, is skipped.
func replayCommand(storage *store.Store, command []string) {
	if len(command) == 0 {
		return
	}

	switch strings.ToUpper(command[0]) {
	case "SET":
		if len(command) == 3 {
			storage.Set(command[1], command[2])
		}

	case "DEL":
		for _, key := range command[1:] {
			storage.Delete(key)
		}

	case "PEXPIREAT":
		if len(command) != 3 {
			return
		}
		milliseconds, err := strconv.ParseInt(command[2], 10, 64)
		if err == nil {
			storage.ExpireAt(command[1], time.UnixMilli(milliseconds))
		}
	}
}

func unixMilli(at time.Time) string {
	return strconv.FormatInt(at.UnixMilli(), 10)
}
