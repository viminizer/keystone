package main

import (
	"fmt"
	"time"
)

// The workers run in the background, each in its own goroutine. They decide
// when to do something, and the engine does it. They never return, so start
// them with go.

// runSyncWorker puts the log on the disk once a second, so a machine crash
// loses at most about one second of changes.
func runSyncWorker(engine *Engine) {
	ticker := time.NewTicker(time.Second)
	for range ticker.C {
		err := engine.SyncLog()
		if err != nil {
			fmt.Println("could not sync the append-only file:", err)
		}
	}
}

// runExpiryWorker deletes expired keys that nobody reads, 10 times a second,
// like Redis.
func runExpiryWorker(engine *Engine) {
	ticker := time.NewTicker(100 * time.Millisecond)
	for range ticker.C {
		engine.DeleteExpiredKeys()
	}
}

// runRewriteWorker checks the size of the log once a second, and rewrites
// it when it has grown too big.
func runRewriteWorker(engine *Engine) {
	ticker := time.NewTicker(time.Second)
	for range ticker.C {
		engine.RewriteIfTooBig()
	}
}
