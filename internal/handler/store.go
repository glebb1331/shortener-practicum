package handler

import (
	"math/rand"
	"sync"
	"time"
)

var rnd = rand.New(rand.NewSource(time.Now().UnixNano()))

type URLStore struct {
	mu   sync.RWMutex
	urls map[string]string
}

func NewURLStore() *URLStore {
	return &URLStore{
		urls: make(map[string]string),
	}
}

func generatedID() string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 8)
	for i := range b {
		b[i] = letters[rnd.Intn(len(letters))]
	}
	return string(b)
}

func (s *URLStore) Save(original string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := generatedID()
	s.urls[id] = original
	return id
}

func (s *URLStore) Get(id string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	url, ok := s.urls[id]
	return url, ok
}
