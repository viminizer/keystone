package main

import (
	"time"

	"github.com/viminizer/keystone/aof"
)

// The log is rewritten by itself when it is at least this big. Without it,
// a very small file would be rewritten all the time.
const autoRewriteMinSize = 64 * 1024 * 1024

// runAutoRewrite checks the size of the log once a second, and starts a
// rewrite when it has grown too big. It never returns, so start it with go.
func runAutoRewrite(engine *Engine, aofLog *aof.Log) {
	ticker := time.NewTicker(time.Second)
	for range ticker.C {
		size, baseSize := aofLog.Sizes()
		if shouldRewrite(size, baseSize) {
			engine.StartRewrite()
		}
	}
}

// shouldRewrite says if the log should be rewritten: when it is twice as big
// as after the last rewrite, and at least autoRewriteMinSize. Redis uses the
// same rule.
func shouldRewrite(size int64, baseSize int64) bool {
	return size >= autoRewriteMinSize && size >= 2*baseSize
}
