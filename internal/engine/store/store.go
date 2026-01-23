package store

import (
	"math/rand"
	"sync"
	"time"
)

const (
	// How often the expirer runs
	expirerInterval = 100 * time.Millisecond

	// How many keys to sample each cycle
	expirerSampleSize = 20

	// If more than this percentage of sampled keys are expired, run again immediately
	expirerThreshold = 0.25
)

type Store struct {
	mu     sync.RWMutex
	data   map[string]*Entry
	stopCh chan struct{}
	doneCh chan struct{}
}

func NewStore() *Store {
	return &Store{
		data:   make(map[string]*Entry),
		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
	}
}

func (s *Store) get(key string) (*Entry, bool) {
	e, ok := s.data[key]
	if !ok {
		return nil, false
	}
	//Lazy delete, if the entry is expired
	if e.IsExpired() {
		delete(s.data, key)
		return nil, false
	}

	return e, true
}

func (s *Store) Set(key string, t ValueType, val any) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data[key] = &Entry{
		Type:  t,
		Value: val,
	}

	return true
}

func (s *Store) Get(key string) (*Entry, bool) {
	s.mu.Lock() //read mutex as using lazy delete (it requires write lock)
	defer s.mu.Unlock()

	return s.get(key)
}

func (s *Store) Delete(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.data[key]; ok {
		delete(s.data, key)
		return true
	}

	return false
}

func (s *Store) SetExpiry(key string, ttl time.Duration) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if e, ok := s.data[key]; ok {
		e.Expiry = time.Now().Add(ttl)
		return true
	}

	return false
}

// StartExpirer starts the background expiration sweeper
func (s *Store) StartExpirer() {
	go func() {
		defer close(s.doneCh)
		ticker := time.NewTicker(expirerInterval)
		defer ticker.Stop()

		for {
			select {
			case <-s.stopCh:
				return
			case <-ticker.C:
				s.expireCycle()
			}
		}
	}()
}

// StopExpirer stops the background expiration sweeper
func (s *Store) StopExpirer() {
	close(s.stopCh)
	<-s.doneCh
}

// expireCycle performs one cycle of active expiration
// It samples random keys with expiry and deletes expired ones
// If many keys are expired, it loops again without waiting
func (s *Store) expireCycle() {
	for {
		expired := s.sampleAndExpire()
		// If less than 25% of sampled keys were expired, we're done
		if expired < int(float64(expirerSampleSize)*expirerThreshold) {
			return
		}
		// Otherwise, run again immediately (too many expired keys)
	}
}

// sampleAndExpire samples random keys and deletes expired ones
// Returns the number of expired keys found
func (s *Store) sampleAndExpire() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Collect keys that have expiry set
	var keysWithExpiry []string
	for k, e := range s.data {
		if !e.Expiry.IsZero() {
			keysWithExpiry = append(keysWithExpiry, k)
		}
	}

	if len(keysWithExpiry) == 0 {
		return 0
	}

	// Sample up to expirerSampleSize keys
	sampleSize := expirerSampleSize
	if len(keysWithExpiry) < sampleSize {
		sampleSize = len(keysWithExpiry)
	}

	// Fisher-Yates shuffle for random sampling
	for i := 0; i < sampleSize; i++ {
		j := i + rand.Intn(len(keysWithExpiry)-i)
		keysWithExpiry[i], keysWithExpiry[j] = keysWithExpiry[j], keysWithExpiry[i]
	}

	// Check sampled keys and delete expired ones
	expired := 0
	for i := 0; i < sampleSize; i++ {
		key := keysWithExpiry[i]
		if e, ok := s.data[key]; ok && e.IsExpired() {
			delete(s.data, key)
			expired++
		}
	}

	return expired
}

// KeyCount returns the number of keys in the store (for debugging/metrics)
func (s *Store) KeyCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.data)
}
