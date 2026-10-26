package metrics

import (
	"sync"
	"testing"
	"time"
)

func TestHistogramObserve(t *testing.T) {
	histogram := NewHistogram()
	histogram.Observe(3 * time.Microsecond)  // first bucket
	histogram.Observe(10 * time.Microsecond) // exactly on a limit: still the first bucket
	histogram.Observe(11 * time.Microsecond) // second bucket
	histogram.Observe(2 * time.Millisecond)  // the 5ms bucket
	histogram.Observe(time.Minute)           // above every limit: the last bucket

	snapshot := histogram.Snapshot()
	want := map[int]int64{0: 2, 1: 1, 5: 1, len(bucketLimits): 1}
	for i, count := range snapshot.Buckets {
		if count != want[i] {
			t.Errorf("bucket %d has %d, want %d", i, count, want[i])
		}
	}
	if snapshot.Count != 5 {
		t.Errorf("Count = %d, want 5", snapshot.Count)
	}
	wantSum := 24*time.Microsecond + 2*time.Millisecond + time.Minute
	if snapshot.Sum != wantSum {
		t.Errorf("Sum = %v, want %v", snapshot.Sum, wantSum)
	}
}

func TestHistogramManyGoroutines(t *testing.T) {
	histogram := NewHistogram()
	var group sync.WaitGroup
	for range 10 {
		group.Add(1)
		go func() {
			defer group.Done()
			for range 1000 {
				histogram.Observe(time.Microsecond)
			}
		}()
	}
	group.Wait()

	if count := histogram.Snapshot().Count; count != 10000 {
		t.Errorf("Count = %d, want 10000", count)
	}
}
