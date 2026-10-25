// Package store keeps keys and their values in memory.
package store

import (
	"sync"
	"time"
)

// entry is what the store saves for one key.
type entry struct {
	value string
	// expiresAt is the zero time when the key never expires.
	expiresAt time.Time
}

// Store is safe to use from many goroutines at the same time.
// The mutex makes sure only one of them uses the map at a time.
type Store struct {
	mutex sync.Mutex
	data  map[string]entry
}

// New returns an empty store.
func New() *Store {
	return &Store{data: map[string]entry{}}
}

// Set saves the value under the key. An old value and its expiry are replaced.
func (s *Store) Set(key string, value string) {
	s.mutex.Lock()
	s.data[key] = entry{value: value}
	s.mutex.Unlock()
}

// SetWithExpiry saves the value under the key, and makes the key expire
// after the given time. Both happen under one lock, so no other client
// can see the key without its expiry.
func (s *Store) SetWithExpiry(key string, value string, after time.Duration) {
	s.mutex.Lock()
	s.data[key] = entry{value: value, expiresAt: time.Now().Add(after)}
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
	delete(s.data, key)
	s.mutex.Unlock()
	return found
}

// Expire makes the key expire after the given time.
// It returns false if the key does not exist.
func (s *Store) Expire(key string, after time.Duration) bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	item, found := s.lookup(key)
	if !found {
		return false
	}
	item.expiresAt = time.Now().Add(after)
	s.data[key] = item
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
		delete(s.data, key)
		return entry{}, false
	}
	return item, true
}
