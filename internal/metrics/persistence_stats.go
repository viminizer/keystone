package metrics

import (
	"sync/atomic"
	"time"
)

// PersistenceStats keeps the numbers about the append-only file. It is safe
// to use from many goroutines.
type PersistenceStats struct {
	fsyncLatency *Histogram
	sizeBytes    atomic.Int64
	rewrites     atomic.Int64
	// lastRewriteDuration is in nanoseconds.
	lastRewriteDuration atomic.Int64
}

func NewPersistenceStats() *PersistenceStats {
	return &PersistenceStats{fsyncLatency: NewHistogram()}
}

// RecordFsync adds one fsync, which puts the file on the disk.
func (p *PersistenceStats) RecordFsync(duration time.Duration) {
	p.fsyncLatency.Observe(duration)
}

// SetSize saves the size of the file now, in bytes.
func (p *PersistenceStats) SetSize(sizeBytes int64) {
	p.sizeBytes.Store(sizeBytes)
}

// RecordRewrite adds one rewrite that worked, and how long it took.
func (p *PersistenceStats) RecordRewrite(duration time.Duration) {
	p.rewrites.Add(1)
	p.lastRewriteDuration.Store(int64(duration))
}
