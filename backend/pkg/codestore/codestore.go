// Package codestore is an in-memory TTL store for one-time codes (not shared
// between instances).
package codestore

import (
	"sync"
	"time"
)

type entry struct {
	value     string
	expiresAt time.Time
}

type Store struct {
	mu   sync.Mutex
	data map[string]entry
}

func New() *Store {
	return &Store{data: make(map[string]entry)}
}

func (s *Store) Save(key, value string, ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = entry{value: value, expiresAt: time.Now().Add(ttl)}
}

func (s *Store) Get(key string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.data[key]
	if !ok || time.Now().After(e.expiresAt) {
		if ok {
			delete(s.data, key)
		}
		return "", false
	}
	return e.value, true
}

func (s *Store) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
}

// Consume atomically returns the value for key and removes it; ok is false when the
// key is missing or expired. Unlike Get+Delete it cannot return the same value twice.
func (s *Store) Consume(key string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.data[key]
	if !ok {
		return "", false
	}
	delete(s.data, key)
	if time.Now().After(e.expiresAt) {
		return "", false
	}
	return e.value, true
}

// Sweep removes entries expired at now and returns how many.
func (s *Store) Sweep(now time.Time) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	removed := 0
	for key, e := range s.data {
		if now.After(e.expiresAt) {
			delete(s.data, key)
			removed++
		}
	}
	return removed
}

// Len reports the number of stored entries, including unswept expired ones.
func (s *Store) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.data)
}
