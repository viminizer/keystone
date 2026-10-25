package store

import "time"

// These numbers are the same as in Redis.
const (
	expiryInterval   = 100 * time.Millisecond
	expirySampleSize = 20
	expiryTimeLimit  = 25 * time.Millisecond
)

// RunActiveExpiry deletes expired keys that nobody reads, 10 times a second.
// It never returns, so start it with go.
func (s *Store) RunActiveExpiry() {
	ticker := time.NewTicker(expiryInterval)
	for range ticker.C {
		s.deleteExpiredKeys()
	}
}

// deleteExpiredKeys runs rounds of deleteExpiredSample. While more than
// 25% of a round was expired, there are probably many more, so it goes on.
// It stops after expiryTimeLimit, so clients are not kept waiting.
func (s *Store) deleteExpiredKeys() {
	start := time.Now()
	for {
		checked, deleted := s.deleteExpiredSample()
		if deleted*4 <= checked || time.Since(start) > expiryTimeLimit {
			return
		}
	}
}

// deleteExpiredSample checks up to expirySampleSize keys that have an
// expiry, and deletes the expired ones. Go visits map keys in a random
// order, so each call checks a different random sample.
// It returns how many keys it checked and how many it deleted.
func (s *Store) deleteExpiredSample() (int, int) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	now := time.Now()
	checked := 0
	deleted := 0
	for key := range s.expiring {
		if checked == expirySampleSize {
			break
		}
		checked++
		if now.After(s.data[key].expiresAt) {
			s.remove(key)
			deleted++
		}
	}
	return checked, deleted
}
