package handler

import (
	"context"
	"errors"
	"log"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/glebb1331/shortener-practicum/internal/logger"
	"github.com/glebb1331/shortener-practicum/internal/storage"
	"go.uber.org/zap"
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

func (s *URLService) Resolve(ctx context.Context, id string) (string, error) {
	return s.store.Get(ctx, id)
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
	logger.Log.Info("ShortenBatch called", zap.Int("count", len(urls)))

	if len(urls) == 0 {
		return nil, errors.New("empty urls")
	}

	records := make([]storage.Record, 0, len(urls))
	response := make([]BatchResponseItem, len(urls))

	for i, item := range urls {
		if strings.TrimSpace(item.OriginalURL) == "" {
			return nil, ErrEmptyURL
		}

		s.mu.Lock()
		id := generateID(s.rnd)
		s.mu.Unlock()

		records = append(records, storage.Record{
			ID:          id,
			OriginalURL: item.OriginalURL,
			UserID:      userID,
		})

		response[i] = BatchResponseItem{
			CorrelationID: item.CorrelationID,
			ShortURL:      s.baseURL + "/" + id,
		}
	}

	logger.Log.Info("Calling BatchSave", zap.Int("record_count", len(records)))
	if err := s.store.BatchSave(ctx, records); err != nil {
		logger.Log.Error("BatchSave failed", zap.Error(err))
		return nil, err
	}

	logger.Log.Info("BatchSave succeeded")
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

	go func() {
		bgCtx := context.Background()

		batchSize := 100
		for i := 0; i < len(ids); i += batchSize {
			end := i + batchSize
			if end > len(ids) {
				end = len(ids)
			}
			batch := ids[i:end]

			if err := s.store.DeleteURLs(bgCtx, userID, batch); err != nil {
				log.Printf("Error deleting URLs batch: %v", err)
			}

			time.Sleep(10 * time.Millisecond)
		}
	}()

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

	resultChan := s.fanInDelete(ctx, userID, idChannels)

	go func() {
		for err := range resultChan {
			_ = err
		}
	}()
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
