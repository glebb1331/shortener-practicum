package storage

import (
	"context"
	"sync"
)

type MemoryStorage struct {
	mu   sync.RWMutex
	urls map[string]string
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		urls: make(map[string]string),
	}
}

func (s *MemoryStorage) Save(ctx context.Context, id, originalURL string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.urls[id] = originalURL
	return nil
}

func (s *MemoryStorage) Get(ctx context.Context, id string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	url, ok := s.urls[id]
	if !ok {
		return "", ErrNotFound
	}
	return url, nil
}

func (s *MemoryStorage) Close() error {
	return nil
}

func (s *MemoryStorage) Ping(ctx context.Context) error {
	return nil
}

func (s *MemoryStorage) BatchSave(ctx context.Context, records []struct {
	ID          string
	OriginalURL string
}) error {

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, record := range records {
		s.urls[record.ID] = record.OriginalURL
	}

	return nil
}
