package storage

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

type URLRecord struct {
	UUID        string `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
	UserID      string `json:"user_id"`
	CreatedAt   string `json:"created_at"`
}

type FileStorage struct {
	mu       sync.RWMutex
	records  []URLRecord
	index    map[string]string
	filePath string
}

func NewFileStorage(filePath string) (*FileStorage, error) {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	s := &FileStorage{
		records:  []URLRecord{},
		index:    make(map[string]string),
		filePath: filePath,
	}

	if err := s.load(); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *FileStorage) Save(ctx context.Context, id, originalURL, userID string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, r := range s.records {
		if r.OriginalURL == originalURL {
			return r.ShortURL, ErrURLExists
		}
	}

	record := URLRecord{
		UUID:        strconv.Itoa(len(s.records) + 1),
		ShortURL:    id,
		OriginalURL: originalURL,
		UserID:      userID,
		CreatedAt:   time.Now().Format(time.RFC3339),
	}

	s.records = append(s.records, record)
	s.index[id] = originalURL

	return id, s.save()
}

func (s *FileStorage) Get(ctx context.Context, id string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	url, ok := s.index[id]
	if !ok {
		return "", ErrNotFound
	}
	return url, nil
}

func (s *FileStorage) load() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	if err := json.Unmarshal(data, &s.records); err != nil {
		return err
	}

	for _, r := range s.records {
		s.index[r.ShortURL] = r.OriginalURL
	}

	return nil
}

func (s *FileStorage) save() error {
	data, err := json.MarshalIndent(s.records, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.filePath, data, 0644)
}

func (s *FileStorage) Close() error                   { return nil }
func (s *FileStorage) Ping(ctx context.Context) error { return nil }

func (s *FileStorage) BatchSave(ctx context.Context, records []Record) error {
	for _, r := range records {
		if _, err := s.Save(ctx, r.ID, r.OriginalURL, r.UserID); err != nil {
			return err
		}
	}
	return nil
}

func (s *FileStorage) GetByOriginalURL(ctx context.Context, originalURL string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, r := range s.records {
		if r.OriginalURL == originalURL {
			return r.ShortURL, nil
		}
	}
	return "", ErrNotFound
}

func (s *FileStorage) GetByUserID(ctx context.Context, userID string) ([]Record, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []Record
	for _, r := range s.records {
		if r.UserID == userID {
			result = append(result, Record{
				ID:          r.ShortURL,
				OriginalURL: r.OriginalURL,
				UserID:      r.UserID,
			})
		}
	}

	return result, nil
}
