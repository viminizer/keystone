package main

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/viminizer/keystone/aof"
	"github.com/viminizer/keystone/store"
)

// The log is rewritten by itself when it is at least this big. Without it,
// a very small file would be rewritten all the time.
const autoRewriteMinSize = 64 * 1024 * 1024

// rewriting is true while the log is being rewritten, so only one rewrite
// runs at a time.
var rewriting atomic.Bool

// startRewrite rewrites the log in the background. It returns false if a
// rewrite is already running.
func startRewrite(db *store.Store, aofLog *aof.Log) bool {
	// CompareAndSwap sets rewriting to true only if it was false.
	if !rewriting.CompareAndSwap(false, true) {
		return false
	}
	go func() {
		rewriteLog(db, aofLog)
		rewriting.Store(false)
	}()
	return true
}

// runAutoRewrite checks the size of the log once a second, and starts a
// rewrite when it has grown too big. It never returns, so start it with go.
func runAutoRewrite(db *store.Store, aofLog *aof.Log) {
	ticker := time.NewTicker(time.Second)
	for range ticker.C {
		size, baseSize := aofLog.Sizes()
		if shouldRewrite(size, baseSize) {
			startRewrite(db, aofLog)
		}
	}
}

// shouldRewrite says if the log should be rewritten: when it is twice as big
// as after the last rewrite, and at least autoRewriteMinSize. Redis uses the
// same rule.
func shouldRewrite(size int64, baseSize int64) bool {
	return size >= autoRewriteMinSize && size >= 2*baseSize
}

// rewriteLog replaces the log with one SET for every key, and a PEXPIREAT
// for every key with an expiry. It holds writeMutex the whole time, so
// writes wait until it is done and none of them go into the old file.
// With many keys that wait gets long. Redis avoids it by writing the new
// file in a child process and buffering the writes that come in meanwhile.
func rewriteLog(db *store.Store, aofLog *aof.Log) {
	writeMutex.Lock()
	defer writeMutex.Unlock()

	items := db.Snapshot()
	commands := [][]string{}
	for _, item := range items {
		commands = append(commands, []string{"SET", item.Key, item.Value})
		if !item.ExpiresAt.IsZero() {
			commands = append(commands, []string{"PEXPIREAT", item.Key, unixMilli(item.ExpiresAt)})
		}
	}

	err := aofLog.Rewrite(commands)
	if err != nil {
		fmt.Println("could not rewrite the append-only file:", err)
		return
	}
	fmt.Printf("rewrote the append-only file with %d keys\n", len(items))
}
