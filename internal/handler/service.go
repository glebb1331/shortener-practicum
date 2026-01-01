package handler

import (
	"context"
	"errors"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/glebb1331/shortener-practicum/internal/storage"
)

var ErrEmptyURL = errors.New("empty url")

type URLService struct {
	store   storage.Storage
	baseURL string
	rnd     *rand.Rand
	mu      sync.Mutex
}

func NewURLService(store storage.Storage, baseURL string) *URLService {
	return &URLService{
		store:   store,
		baseURL: baseURL,
		rnd:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (s *URLService) Shorten(ctx context.Context, url, userID string) (string, error) {
	if strings.TrimSpace(url) == "" {
		return "", ErrEmptyURL
	}

	s.mu.Lock()
	id := generateID(s.rnd)
	s.mu.Unlock()

	storedID, err := s.store.Save(ctx, id, url, userID)
	if err != nil {
		if errors.Is(err, storage.ErrURLExists) {
			return s.baseURL + "/" + storedID, storage.ErrURLExists
		}
		return "", err
	}

	return s.baseURL + "/" + storedID, nil
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

func generateID(rnd *rand.Rand) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 8)
	for i := range b {
		b[i] = letters[rnd.Intn(len(letters))]
	}
	return string(b)
}

func (s *URLService) ShortenBatch(ctx context.Context, urls []BatchRequestItem, userID string) ([]BatchResponseItem, error) {
	if len(urls) == 0 {
		return nil, errors.New("empty urls")
	}

	records := make([]storage.Record, len(urls))
	response := make([]BatchResponseItem, len(urls))

	for i, item := range urls {
		if strings.TrimSpace(item.OriginalURL) == "" {
			return nil, ErrEmptyURL
		}

		s.mu.Lock()
		id := generateID(s.rnd)
		s.mu.Unlock()

		records[i] = storage.Record{
			ID:          id,
			OriginalURL: item.OriginalURL,
			UserID:      userID,
		}

		response[i] = BatchResponseItem{
			CorrelationID: item.CorrelationID,
			ShortURL:      s.baseURL + "/" + id,
		}
	}

	if err := s.store.BatchSave(ctx, records); err != nil {
		return nil, err
	}
	return response, nil
}

func (s *URLService) GetByUserID(ctx context.Context, userID string) ([]storage.Record, error) {
	return s.store.GetByUserID(ctx, userID)
}

func (s *URLService) BaseURL() string {
	return s.baseURL
}
