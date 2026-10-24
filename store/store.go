// Package store keeps keys and their values in memory.
package store

import "sync"

// Store is safe to use from many goroutines at the same time.
// The mutex makes sure only one of them uses the map at a time.
type Store struct {
	mutex sync.Mutex
	data  map[string]string
}

// New returns an empty store.
func New() *Store {
	return &Store{data: map[string]string{}}
}

// Set saves the value under the key. An old value is replaced.
func (s *Store) Set(key string, value string) {
	s.mutex.Lock()
	s.data[key] = value
	s.mutex.Unlock()
}

// Get returns the value for the key, and false if the key does not exist.
func (s *Store) Get(key string) (string, bool) {
	s.mutex.Lock()
	value, found := s.data[key]
	s.mutex.Unlock()
	return value, found
}
