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

func (s *URLService) Resolve(id string) (string, error) {
	url, err := s.store.Get(context.Background(), id)
	if err != nil {
		return "", err
	}
	return url, nil
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

func (s *URLService) DeleteURLs(ctx context.Context, userID string, ids []string) error {
	if len(ids) == 0 {
		return errors.New("empty ids")
	}

	records, err := s.store.GetBatchByUserID(ctx, userID, ids)
	if err != nil {
		return err
	}

	if len(records) != len(ids) {
		return storage.ErrNotOwner
	}

	go s.asyncDeleteURLs(ctx, userID, ids)
	return nil
}

func (s *URLService) asyncDeleteURLs(ctx context.Context, userID string, ids []string) {
	batchSize := 100
	idChannels := make([]chan []string, 0)

	for i := 0; i < len(ids); i += batchSize {
		end := i + batchSize
		if end > len(ids) {
			end = len(ids)
		}
		batch := ids[i:end]

		ch := make(chan []string, 1)
		ch <- batch
		close(ch)
		idChannels = append(idChannels, ch)
	}

	//resultChan := s.fanInDelete(ctx, userID, idChannels)
}

func (s *URLService) fanInDelete(ctx context.Context, userID string, channels []chan []string) <-chan error {
	out := make(chan error)
	var wg sync.WaitGroup

	processChannel := func(ch <-chan []string) {
		defer wg.Done()
		for batch := range ch {
			if err := s.store.DeleteURLs(ctx, userID, batch); err != nil {
				out <- err
			}
		}
	}

	wg.Add(len(channels))
	for _, ch := range channels {
		go processChannel(ch)
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}
