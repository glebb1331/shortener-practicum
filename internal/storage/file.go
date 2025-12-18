package storage

import (
	"context"
	"encoding/json"
	"os"
	"sync"
)

type URLRecord struct {
	UUID        string `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

type FileStorage struct {
	mu       sync.RWMutex
	urls     map[string]string
	filePath string
}

func NewFileStorage(filePath string) (*FileStorage, error) {
	s := &FileStorage{
		urls:     make(map[string]string),
		filePath: filePath,
	}

	if err := s.loadFromFile(); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *FileStorage) Save(ctx context.Context, id, originalURL string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.urls[id] = originalURL
	return s.saveToFile()
}

func (s *FileStorage) Get(ctx context.Context, id string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	url, ok := s.urls[id]
	if !ok {
		return "", ErrNotFound
	}
	return url, nil
}

func (s *FileStorage) Close() error {
	return nil
}

func (s *FileStorage) Ping(ctx context.Context) error {
	return nil
}

func (s *FileStorage) loadFromFile() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	if len(data) == 0 {
		return nil
	}

	if err := json.Unmarshal(data, &s.urls); err == nil {
		return nil
	}

	var records []URLRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return err
	}

	for _, record := range records {
		s.urls[record.ShortURL] = record.OriginalURL
	}

	return nil
}

func (s *FileStorage) saveToFile() error {
	data, err := json.MarshalIndent(s.urls, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.filePath, data, 0644)
}
