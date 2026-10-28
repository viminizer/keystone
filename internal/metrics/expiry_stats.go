package metrics

import "sync/atomic"

// ExpiryStats counts the keys deleted because their time ran out. It is
// safe to use from many goroutines.
type ExpiryStats struct {
	// active counts the keys deleted by the background cleanup.
	active atomic.Int64
	// lazy counts the keys deleted when a client touched them.
	lazy atomic.Int64
	// keysWithExpiry is how many keys have an expiry right now.
	keysWithExpiry atomic.Int64
}

func NewExpiryStats() *ExpiryStats {
	return &ExpiryStats{}
}

// RecordActiveExpiry adds keys deleted by the background cleanup.
func (e *ExpiryStats) RecordActiveExpiry(count int) {
	e.active.Add(int64(count))
}

// RecordLazyExpiry adds one key deleted when a client touched it.
func (e *ExpiryStats) RecordLazyExpiry() {
	e.lazy.Add(1)
}

// SetKeysWithExpiry saves how many keys have an expiry right now.
func (e *ExpiryStats) SetKeysWithExpiry(count int) {
	e.keysWithExpiry.Store(int64(count))
}
