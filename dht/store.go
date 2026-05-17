package dht

import (
	"sync"
	"time"
)

type entry struct {
	value   []byte
	expires time.Time
}

type Store struct {
	mu   sync.RWMutex
	data map[string]*entry
}

func NewStore() *Store {
	return &Store{data: make(map[string]*entry)}
}

func (s *Store) Store(key string, value []byte, ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = &entry{
		value:   value,
		expires: time.Now().Add(ttl),
	}
}

func (s *Store) FindValue(key string) ([]byte, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.data[key]
	if !ok || time.Now().After(e.expires) {
		return nil, false
	}
	return e.value, true
}

func (s *Store) Cleanup() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	removed := 0
	for k, e := range s.data {
		if now.After(e.expires) {
			delete(s.data, k)
			removed++
		}
	}
	return removed
}

func (s *Store) StartCleanupLoop(interval time.Duration, quit <-chan struct{}) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.Cleanup()
		case <-quit:
			return
		}
	}
}

func (s *Store) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.data)
}
