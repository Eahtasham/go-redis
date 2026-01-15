package store

import "sync"

type Store struct {
	mu   sync.RWMutex
	data map[string]*Entry
}

func NewStore() *Store {
	return &Store{data: make(map[string]*Entry)}
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
