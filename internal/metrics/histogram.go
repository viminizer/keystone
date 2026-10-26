// Package metrics keeps numbers about what keystone does, like how many
// commands ran and how long they took.
package metrics

import (
	"sync/atomic"
	"time"
)

// bucketLimits are the upper limits of the histogram buckets. A measurement
// goes into the first bucket it fits in. Anything slower goes into one more
// bucket at the end, which has no limit. Most commands take microseconds,
// so the small buckets are close together.
var bucketLimits = []time.Duration{
	10 * time.Microsecond,
	50 * time.Microsecond,
	100 * time.Microsecond,
	500 * time.Microsecond,
	1 * time.Millisecond,
	5 * time.Millisecond,
	10 * time.Millisecond,
	50 * time.Millisecond,
	100 * time.Millisecond,
	500 * time.Millisecond,
	1 * time.Second,
}

// Histogram counts how many measurements fell into each bucket, and adds
// them all up. It keeps the same few numbers no matter how many
// measurements there are. It is safe to use from many goroutines.
type Histogram struct {
	// buckets has one counter for each limit, and one more at the end.
	buckets []atomic.Int64
	// sum is the total of all measurements, in nanoseconds.
	sum atomic.Int64
}

// HistogramSnapshot is a copy of a histogram's numbers at one moment.
type HistogramSnapshot struct {
	// Limits are the upper limits of the buckets.
	Limits []time.Duration
	// Buckets has the count for each limit, and one more at the end for
	// the measurements above the last limit.
	Buckets []int64
	// Count is the number of measurements.
	Count int64
	// Sum is the total of all measurements.
	Sum time.Duration
}

func NewHistogram() *Histogram {
	return &Histogram{buckets: make([]atomic.Int64, len(bucketLimits)+1)}
}

// Observe adds one measurement.
func (h *Histogram) Observe(duration time.Duration) {
	bucket := len(bucketLimits)
	for i, limit := range bucketLimits {
		if duration <= limit {
			bucket = i
			break
		}
	}
	h.buckets[bucket].Add(1)
	h.sum.Add(int64(duration))
}

// Snapshot returns a copy of the numbers. While other goroutines keep
// observing, Sum can be a little ahead of or behind the buckets.
func (h *Histogram) Snapshot() HistogramSnapshot {
	snapshot := HistogramSnapshot{
		Limits:  bucketLimits,
		Buckets: make([]int64, len(h.buckets)),
		Sum:     time.Duration(h.sum.Load()),
	}
	for i := range h.buckets {
		snapshot.Buckets[i] = h.buckets[i].Load()
		snapshot.Count += snapshot.Buckets[i]
	}
	return snapshot
}
