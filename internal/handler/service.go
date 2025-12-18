package handler

import (
	"context"
	"errors"
	"math/rand"
	"strings"
	"time"

	"github.com/glebb1331/shortener-practicum/internal/storage"
)

var ErrEmptyURL = errors.New("empty url")
var rnd = rand.New(rand.NewSource(time.Now().UnixNano()))

type URLService struct {
	store   storage.Storage
	baseURL string
}

func NewURLService(store storage.Storage, baseURL string) *URLService {
	return &URLService{
		store:   store,
		baseURL: baseURL,
	}
}

func (s *URLService) Shorten(url string) (string, error) {
	if strings.TrimSpace(url) == "" {
		return "", ErrEmptyURL
	}

	id := generateID()
	if err := s.store.Save(context.Background(), id, url); err != nil {
		return "", err
	}

	return s.baseURL + "/" + id, nil
}

func (s *URLService) Resolve(id string) (string, bool) {
	url, err := s.store.Get(context.Background(), id)
	if err != nil {
		return "", false
	}
	return url, true
}

func (s *URLService) Ping(ctx context.Context) error {
	return s.store.Ping(ctx)
}

func generateID() string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 8)
	for i := range b {
		b[i] = letters[rnd.Intn(len(letters))]
	}
	return string(b)
}
