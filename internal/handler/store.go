package handler

import (
	"encoding/json"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"time"
)

var rnd = rand.New(rand.NewSource(time.Now().UnixNano()))

type URLRecord struct {
	UUID        string `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

type URLStore struct {
	mu              sync.RWMutex
	urls            map[string]string
	fileStoragePath string
}

func NewURLStore(fileStoragePath string) (*URLStore, error) {
	s := &URLStore{
		urls: make(map[string]string),
	}

	if err := s.LoadFromFile(); err != nil {
		return nil, err
	}
	return s, nil
}

func generatedID() string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 8)
	for i := range b {
		b[i] = letters[rnd.Intn(len(letters))]
	}
	return string(b)
}

func (s *URLStore) Save(original string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := generatedID()
	s.urls[id] = original

	if err := s.saveToFile(); err != nil {
		return "", err
	}
	return id, nil
}

func (s *URLStore) Get(id string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	url, ok := s.urls[id]
	return url, ok
}

func (s *URLStore) LoadFromFile() error {

	if s.fileStoragePath == "" {
		return nil
	}

	data, err := os.ReadFile(s.fileStoragePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	return json.Unmarshal(data, &s.urls)
}

func (s *URLStore) saveToFile() error {
	if s.fileStoragePath == "" {
		return nil
	}

	records := make([]URLRecord, 0, len(s.urls))
	uuidIndex := 1
	for shortURL, originalURL := range s.urls {
		records = append(records, URLRecord{
			UUID:        strconv.Itoa(uuidIndex),
			ShortURL:    shortURL,
			OriginalURL: originalURL,
		})
		uuidIndex++
	}

	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.fileStoragePath, data, 0644)
}
