// Package store keeps keys and their values in memory.
package store

import (
	"sync"
	"time"

	"github.com/viminizer/keystone/internal/metrics"
)

// entry is what the store saves for one key.
type entry struct {
	value string
	// expiresAt is the zero time when the key never expires.
	expiresAt time.Time
}

// Store is safe to use from many goroutines at the same time.
// The mutex makes sure only one of them uses the maps at a time.
type Store struct {
	mutex sync.Mutex
	data  map[string]entry
	// expiring holds only the keys that have an expiry, so active
	// expiry can find them without looking at every key.
	expiring    map[string]bool
	expiryStats *metrics.ExpiryStats
}

// New returns an empty store.
func New(expiryStats *metrics.ExpiryStats) *Store {
	return &Store{
		data:        map[string]entry{},
		expiring:    map[string]bool{},
		expiryStats: expiryStats,
	}
}

// Set saves the value under the key. An old value and its expiry are replaced.
func (s *Store) Set(key string, value string) {
	s.mutex.Lock()
	s.data[key] = entry{value: value}
	delete(s.expiring, key)
	s.mutex.Unlock()
}

// SetWithExpiry saves the value under the key, and makes the key expire
// at the given moment. Both happen under one lock, so no other client
// can see the key without its expiry.
func (s *Store) SetWithExpiry(key string, value string, at time.Time) {
	s.mutex.Lock()
	s.data[key] = entry{value: value, expiresAt: at}
	s.expiring[key] = true
	s.mutex.Unlock()
}

// Get returns the value for the key, and false if the key does not exist.
func (s *Store) Get(key string) (string, bool) {
	s.mutex.Lock()
	item, found := s.lookup(key)
	s.mutex.Unlock()
	return item.value, found
}

// Delete removes the key. It returns false if the key did not exist.
func (s *Store) Delete(key string) bool {
	s.mutex.Lock()
	_, found := s.lookup(key)
	s.remove(key)
	s.mutex.Unlock()
	return found
}

// ExpireAt makes the key expire at the given moment.
// It returns false if the key does not exist.
func (s *Store) ExpireAt(key string, at time.Time) bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	item, found := s.lookup(key)
	if !found {
		return false
	}
	item.expiresAt = at
	s.data[key] = item
	s.expiring[key] = true
	return true
}

// ExpiresAt returns when the key expires, and false if the key does not exist.
// The time is zero when the key never expires.
func (s *Store) ExpiresAt(key string) (time.Time, bool) {
	s.mutex.Lock()
	item, found := s.lookup(key)
	s.mutex.Unlock()
	return item.expiresAt, found
}

// lookup returns the entry for the key. An expired key is deleted
// and treated as missing. The mutex must be locked by the caller.
func (s *Store) lookup(key string) (entry, bool) {
	item, found := s.data[key]
	if !found {
		return entry{}, false
	}
	if !item.expiresAt.IsZero() && time.Now().After(item.expiresAt) {
		s.remove(key)
		s.expiryStats.RecordLookup()
		return entry{}, false
	}
	return item, true
}

// remove deletes the key from both maps. The mutex must be locked by the caller.
func (s *Store) remove(key string) {
	delete(s.data, key)
	delete(s.expiring, key)
}

// Item is one key with its value and expiry, as returned by Snapshot.
type Item struct {
	Key   string
	Value string
	// ExpiresAt is the zero time when the key never expires.
	ExpiresAt time.Time
}

// Snapshot returns a copy of every key that has not expired.
func (s *Store) Snapshot() []Item {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	now := time.Now()
	items := []Item{}
	for key, stored := range s.data {
		if !stored.expiresAt.IsZero() && now.After(stored.expiresAt) {
			continue
		}
		items = append(items, Item{Key: key, Value: stored.value, ExpiresAt: stored.expiresAt})
	}
	return items
}
