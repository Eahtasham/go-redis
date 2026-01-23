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

// ==================== ATOMIC SET OPERATIONS ====================

// SAdd atomically adds members to a set, returns count of new members added
func (s *Store) SAdd(key string, members []string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	e, ok := s.data[key]
	var set map[string]struct{}

	if ok {
		if e.Type != SetType {
			return 0, ErrWrongType
		}
		set = e.Value.(map[string]struct{})
	} else {
		set = make(map[string]struct{})
		s.data[key] = &Entry{Type: SetType, Value: set}
	}

	added := int64(0)
	for _, member := range members {
		if _, exists := set[member]; !exists {
			set[member] = struct{}{}
			added++
		}
	}

	return added, nil
}

// SRem atomically removes members from a set, returns count removed
func (s *Store) SRem(key string, members []string) (int64, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	e, ok := s.data[key]
	if !ok {
		return 0, false
	}

	if e.Type != SetType {
		return 0, false
	}

	set := e.Value.(map[string]struct{})
	removed := int64(0)

	for _, member := range members {
		if _, exists := set[member]; exists {
			delete(set, member)
			removed++
		}
	}

	// Delete key if set is empty
	if len(set) == 0 {
		delete(s.data, key)
	}

	return removed, true
}

// SMembers returns all members of a set (returns a copy)
func (s *Store) SMembers(key string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	e, ok := s.data[key]
	if !ok {
		return []string{}, nil
	}

	if e.Type != SetType {
		return nil, ErrWrongType
	}

	set := e.Value.(map[string]struct{})
	result := make([]string, 0, len(set))
	for member := range set {
		result = append(result, member)
	}

	return result, nil
}

// SIsMember checks if member exists in set
func (s *Store) SIsMember(key, member string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	e, ok := s.data[key]
	if !ok {
		return false, nil
	}

	if e.Type != SetType {
		return false, ErrWrongType
	}

	set := e.Value.(map[string]struct{})
	_, exists := set[member]
	return exists, nil
}

// SCard returns the cardinality (size) of a set
func (s *Store) SCard(key string) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	e, ok := s.data[key]
	if !ok {
		return 0, nil
	}

	if e.Type != SetType {
		return 0, ErrWrongType
	}

	set := e.Value.(map[string]struct{})
	return int64(len(set)), nil
}

// ==================== ATOMIC LIST OPERATIONS ====================

// LPush atomically prepends values to a list, returns new length
func (s *Store) LPush(key string, values []string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	e, ok := s.data[key]
	var list []string

	if ok {
		if e.Type != ListType {
			return 0, ErrWrongType
		}
		list = e.Value.([]string)
	}

	// Prepend in reverse order so first value ends up at head
	for i := len(values) - 1; i >= 0; i-- {
		list = append([]string{values[i]}, list...)
	}

	s.data[key] = &Entry{Type: ListType, Value: list}
	return int64(len(list)), nil
}

// RPush atomically appends values to a list, returns new length
func (s *Store) RPush(key string, values []string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	e, ok := s.data[key]
	var list []string

	if ok {
		if e.Type != ListType {
			return 0, ErrWrongType
		}
		list = e.Value.([]string)
	}

	list = append(list, values...)
	s.data[key] = &Entry{Type: ListType, Value: list}
	return int64(len(list)), nil
}

// LPop atomically removes and returns elements from head
func (s *Store) LPop(key string, count int) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	e, ok := s.data[key]
	if !ok {
		return nil, nil
	}

	if e.Type != ListType {
		return nil, ErrWrongType
	}

	list := e.Value.([]string)
	if len(list) == 0 {
		return nil, nil
	}

	if count > len(list) {
		count = len(list)
	}

	popped := make([]string, count)
	copy(popped, list[:count])
	remaining := list[count:]

	if len(remaining) == 0 {
		delete(s.data, key)
	} else {
		s.data[key] = &Entry{Type: ListType, Value: remaining}
	}

	return popped, nil
}

// RPop atomically removes and returns elements from tail
func (s *Store) RPop(key string, count int) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	e, ok := s.data[key]
	if !ok {
		return nil, nil
	}

	if e.Type != ListType {
		return nil, ErrWrongType
	}

	list := e.Value.([]string)
	if len(list) == 0 {
		return nil, nil
	}

	if count > len(list) {
		count = len(list)
	}

	start := len(list) - count
	popped := make([]string, count)
	copy(popped, list[start:])
	remaining := list[:start]

	// Reverse to return rightmost first
	for i, j := 0, len(popped)-1; i < j; i, j = i+1, j-1 {
		popped[i], popped[j] = popped[j], popped[i]
	}

	if len(remaining) == 0 {
		delete(s.data, key)
	} else {
		s.data[key] = &Entry{Type: ListType, Value: remaining}
	}

	return popped, nil
}

// LRange returns a range of elements from a list (returns a copy)
func (s *Store) LRange(key string, start, stop int) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	e, ok := s.data[key]
	if !ok {
		return []string{}, nil
	}

	if e.Type != ListType {
		return nil, ErrWrongType
	}

	list := e.Value.([]string)
	length := len(list)

	// Handle negative indices
	if start < 0 {
		start = length + start
	}
	if stop < 0 {
		stop = length + stop
	}

	// Clamp
	if start < 0 {
		start = 0
	}
	if stop >= length {
		stop = length - 1
	}

	if start > stop || start >= length {
		return []string{}, nil
	}

	result := make([]string, stop-start+1)
	copy(result, list[start:stop+1])
	return result, nil
}

// LLen returns the length of a list
func (s *Store) LLen(key string) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	e, ok := s.data[key]
	if !ok {
		return 0, nil
	}

	if e.Type != ListType {
		return 0, ErrWrongType
	}

	list := e.Value.([]string)
	return int64(len(list)), nil
}

// LIndex returns the element at index
func (s *Store) LIndex(key string, index int) (string, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	e, ok := s.data[key]
	if !ok {
		return "", false, nil
	}

	if e.Type != ListType {
		return "", false, ErrWrongType
	}

	list := e.Value.([]string)

	// Handle negative index
	if index < 0 {
		index = len(list) + index
	}

	if index < 0 || index >= len(list) {
		return "", false, nil
	}

	return list[index], true, nil
}

// GetListCopy returns a copy of the list for AOF logging
func (s *Store) GetListCopy(key string) ([]string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	e, ok := s.data[key]
	if !ok || e.Type != ListType {
		return nil, false
	}

	list := e.Value.([]string)
	result := make([]string, len(list))
	copy(result, list)
	return result, true
}
