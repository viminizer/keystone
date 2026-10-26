package main

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/viminizer/keystone/aof"
	"github.com/viminizer/keystone/store"
)

// Engine runs the database operations. It changes the store and writes each
// change to the log.
type Engine struct {
	db     *store.Store
	aofLog *aof.Log
	// writeMutex is held while a change goes to the store and to the log.
	// So the commands in the log are always in the same order as they
	// changed the store. Reads do not need it, because they write nothing
	// to the log.
	writeMutex sync.Mutex
	// rewriting is true while the log is being rewritten, so only one
	// rewrite runs at a time.
	rewriting atomic.Bool
}

func NewEngine(db *store.Store, aofLog *aof.Log) *Engine {
	return &Engine{db: db, aofLog: aofLog}
}

func (e *Engine) Set(key string, value string) {
	e.writeMutex.Lock()
	defer e.writeMutex.Unlock()

	e.db.Set(key, value)
	e.appendToLog([]string{"SET", key, value})
}

func (e *Engine) SetWithExpiry(key string, value string, at time.Time) {
	e.writeMutex.Lock()
	defer e.writeMutex.Unlock()

	e.db.SetWithExpiry(key, value, at)
	e.appendToLog([]string{"SET", key, value})
	e.appendToLog([]string{"PEXPIREAT", key, unixMilli(at)})
}

func (e *Engine) Get(key string) (string, bool) {
	return e.db.Get(key)
}

// Delete deletes the keys and returns how many of them existed.
func (e *Engine) Delete(keys []string) int {
	e.writeMutex.Lock()
	defer e.writeMutex.Unlock()

	deleted := 0
	for _, key := range keys {
		if e.db.Delete(key) {
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

	if !e.db.ExpireAt(key, at) {
		return false
	}
	e.appendToLog([]string{"PEXPIREAT", key, unixMilli(at)})
	return true
}

func (e *Engine) ExpiresAt(key string) (time.Time, bool) {
	return e.db.ExpiresAt(key)
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

	items := e.db.Snapshot()
	commands := [][]string{}
	for _, item := range items {
		commands = append(commands, []string{"SET", item.Key, item.Value})
		if !item.ExpiresAt.IsZero() {
			commands = append(commands, []string{"PEXPIREAT", item.Key, unixMilli(item.ExpiresAt)})
		}
	}

	err := e.aofLog.Rewrite(commands)
	if err != nil {
		fmt.Println("could not rewrite the append-only file:", err)
		return
	}
	fmt.Printf("rewrote the append-only file with %d keys\n", len(items))
}

// Shutdown waits for a running write or rewrite to finish, and puts the log
// on the disk. It never lets go of writeMutex, so no write can happen after
// it. Call it only right before the program exits.
func (e *Engine) Shutdown() error {
	e.writeMutex.Lock()
	return e.aofLog.Sync()
}

// appendToLog writes one command to the log. If that fails, the change is
// still in memory, so the error is only printed.
func (e *Engine) appendToLog(command []string) {
	err := e.aofLog.Append(command)
	if err != nil {
		fmt.Println("could not write to the append-only file:", err)
	}
}

// replayCommand applies one command from the log to the store. It runs at
// start, before any client connects, so it needs no lock and writes nothing
// to the log. The log only has SET, DEL and PEXPIREAT. Anything else, or a
// command with the wrong number of arguments, is skipped.
func replayCommand(db *store.Store, command []string) {
	if len(command) == 0 {
		return
	}

	switch strings.ToUpper(command[0]) {
	case "SET":
		if len(command) == 3 {
			db.Set(command[1], command[2])
		}

	case "DEL":
		for _, key := range command[1:] {
			db.Delete(key)
		}

	case "PEXPIREAT":
		if len(command) != 3 {
			return
		}
		milliseconds, err := strconv.ParseInt(command[2], 10, 64)
		if err == nil {
			db.ExpireAt(command[1], time.UnixMilli(milliseconds))
		}
	}
}

func unixMilli(at time.Time) string {
	return strconv.FormatInt(at.UnixMilli(), 10)
}
