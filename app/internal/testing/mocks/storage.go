// Package mocks provides in-memory fakes for the one real interface seam
// in this codebase's storage layer -- see internal/testing/doc.go's own
// "Scope note on mocks" for why there isn't more here.
package mocks

import (
	"fmt"
	"sync"

	"menata.id/app/internal/storage"
)

// Storage is an in-memory storage.Store for handler tests that touch file
// upload/download without real disk I/O. Keys are sequential, not
// content-derived like LocalDisk's real token -- tests should treat the
// key as opaque, not assert its shape.
type Storage struct {
	mu    sync.Mutex
	n     int
	files map[string][]byte
	types map[string]string
}

var _ storage.Store = (*Storage)(nil)

func NewStorage() *Storage {
	return &Storage{files: map[string][]byte{}, types: map[string]string{}}
}

func (s *Storage) Put(data []byte, contentType string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.n++
	key := fmt.Sprintf("mock-%d", s.n)
	s.files[key] = data
	s.types[key] = contentType
	return key, nil
}

func (s *Storage) Get(key string) ([]byte, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, ok := s.files[key]
	if !ok {
		return nil, "", fmt.Errorf("mocks.Storage: no such key %q", key)
	}
	return data, s.types[key], nil
}

func (s *Storage) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.files, key)
	delete(s.types, key)
	return nil
}
