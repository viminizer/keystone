package main

import (
	"fmt"
	"time"
)

// Workers run in the background, each in its own goroutine. They decide
// when to do something, and the engine does it. Their Run methods never
// return, so start them with go.
type Workers struct {
	engine *Engine
}

func NewWorkers(engine *Engine) *Workers {
	return &Workers{engine: engine}
}

// RunSync puts the log on the disk once a second, so a machine crash loses
// at most about one second of changes.
func (w *Workers) RunSync() {
	ticker := time.NewTicker(time.Second)
	for range ticker.C {
		err := w.engine.SyncLog()
		if err != nil {
			fmt.Println("could not sync the append-only file:", err)
		}
	}
}

// RunExpiry deletes expired keys that nobody reads, 10 times a second, like
// Redis.
func (w *Workers) RunExpiry() {
	ticker := time.NewTicker(100 * time.Millisecond)
	for range ticker.C {
		w.engine.DeleteExpiredKeys()
	}
}

// RunRewrite checks the size of the log once a second, and rewrites it when
// it has grown too big.
func (w *Workers) RunRewrite() {
	ticker := time.NewTicker(time.Second)
	for range ticker.C {
		w.engine.RewriteIfTooBig()
	}
}
