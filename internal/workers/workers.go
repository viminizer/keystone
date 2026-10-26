// Package workers runs the background jobs: syncing the log, deleting
// expired keys, rewriting the log and stopping on a signal.
package workers

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/viminizer/keystone/internal/engine"
)

// Workers run in the background, each in its own goroutine. They decide
// when to do something, and the engine does it.
type Workers struct {
	db *engine.Engine
}

func New(db *engine.Engine) *Workers {
	return &Workers{db: db}
}

// Start starts every worker in its own goroutine, and returns right away.
func (w *Workers) Start() {
	go w.runSync()
	go w.runExpiry()
	go w.runRewrite()
	go w.runStopOnSignal()
}

// runStopOnSignal waits for Ctrl+C (SIGINT) or kill (SIGTERM), and then
// stops keystone cleanly.
func (w *Workers) runStopOnSignal() {
	stopSignals := make(chan os.Signal, 1)
	signal.Notify(stopSignals, syscall.SIGINT, syscall.SIGTERM)
	received := <-stopSignals
	fmt.Printf("received %s, stopping\n", received)

	err := w.db.Shutdown()
	if err != nil {
		fmt.Println("could not sync the append-only file:", err)
		os.Exit(1)
	}
	fmt.Println("keystone stopped")
	os.Exit(0)
}

// runSync puts the log on the disk once a second, so a machine crash loses
// at most about one second of changes.
func (w *Workers) runSync() {
	ticker := time.NewTicker(time.Second)
	for range ticker.C {
		err := w.db.SyncLog()
		if err != nil {
			fmt.Println("could not sync the append-only file:", err)
		}
	}
}

// runExpiry deletes expired keys that nobody reads, 10 times a second, like
// Redis.
func (w *Workers) runExpiry() {
	ticker := time.NewTicker(100 * time.Millisecond)
	for range ticker.C {
		w.db.DeleteExpiredKeys()
	}
}

// runRewrite checks the size of the log once a second, and rewrites it when
// it has grown too big.
func (w *Workers) runRewrite() {
	ticker := time.NewTicker(time.Second)
	for range ticker.C {
		w.db.RewriteIfTooBig()
	}
}
