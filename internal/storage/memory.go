package storage

import (
	"context"
	"sync"
)

type MemoryStorage struct {
	mu   sync.RWMutex
	urls map[string]Record
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		urls: make(map[string]Record),
	}
}

func (s *MemoryStorage) Save(ctx context.Context, id, originalURL, userID string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, rec := range s.urls {
		if rec.OriginalURL == originalURL {
			return rec.ID, ErrURLExists
		}
	}

	s.urls[id] = Record{ID: id, OriginalURL: originalURL, UserID: userID}
	return id, nil
}

func (s *MemoryStorage) Get(ctx context.Context, id string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rec, ok := s.urls[id]
	if !ok {
		return "", ErrNotFound
	}

	if rec.IsDeleted {
		return "", ErrURLDeleted
	}

	return rec.OriginalURL, nil
}

func (s *MemoryStorage) Close() error {
	return nil
}

func (s *MemoryStorage) Ping(ctx context.Context) error {
	return nil
}

func (s *MemoryStorage) BatchSave(ctx context.Context, records []Record) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, rec := range records {
		if _, exists := s.urls[rec.ID]; exists {
			continue
		}

		exists := false
		for _, existingRec := range s.urls {
			if existingRec.OriginalURL == rec.OriginalURL {
				exists = true
				break
			}
		}

		if !exists {
			s.urls[rec.ID] = rec
		}
	}

	return nil
}

func (s *MemoryStorage) GetByOriginalURL(ctx context.Context, originalURL string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, rec := range s.urls {
		if rec.OriginalURL == originalURL {
			return rec.ID, nil
		}
	}
	return "", ErrNotFound
}

func (s *MemoryStorage) GetByUserID(ctx context.Context, userID string) ([]Record, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]Record, 0, len(s.urls)/2)
	for _, rec := range s.urls {
		if rec.UserID == userID {
			result = append(result, rec)
		}
	}

	return result, nil
}

func (s *MemoryStorage) DeleteURLs(ctx context.Context, userID string, ids []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, id := range ids {
		if rec, ok := s.urls[id]; ok && rec.UserID == userID {
			rec.IsDeleted = true
			s.urls[id] = rec
		}
	}
	return nil
}

func (s *MemoryStorage) GetBatchByUserID(ctx context.Context, userID string, ids []string) ([]Record, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var records []Record
	for _, id := range ids {
		if rec, ok := s.urls[id]; ok && rec.UserID == userID {
			records = append(records, rec)
		}
	}
	return records, nil
}
