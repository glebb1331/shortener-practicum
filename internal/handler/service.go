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

// ErrEmptyURL возвращается, когда переданный URL пустой или состоит только из пробелов.
var ErrEmptyURL = errors.New("empty url")

var idBufPool = sync.Pool{New: func() any { b := make([]byte, 8); return &b }}

// URLService реализует бизнес-логику сокращения ссылок.
type URLService struct {
	store     storage.Storage
	baseURL   string
	urlPrefix string
	rnd       *rand.Rand
	mu        sync.Mutex
}

// NewURLService создаёт URLService с заданным хранилищем и базовым URL.
func NewURLService(store storage.Storage, baseURL string) *URLService {
	return &URLService{
		store:     store,
		baseURL:   baseURL,
		urlPrefix: strings.TrimRight(baseURL, "/") + "/",
		rnd:       rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Shorten сокращает URL и возвращает короткую ссылку.
// Если URL уже существует, возвращает существующую ссылку и storage.ErrURLExists.
func (s *URLService) Shorten(ctx context.Context, originalURL, userID string) (string, error) {
	if strings.TrimSpace(originalURL) == "" {
		return "", ErrEmptyURL
	}

	s.mu.Lock()
	id := generateID(s.rnd)
	s.mu.Unlock()

	storedID, err := s.store.Save(ctx, id, originalURL, userID)
	if err != nil {
		if errors.Is(err, storage.ErrURLExists) {
			return s.urlPrefix + storedID, storage.ErrURLExists
		}
		return "", err
	}

	return s.urlPrefix + storedID, nil
}

// Resolve возвращает оригинальный URL по короткому id.
func (s *URLService) Resolve(ctx context.Context, id string) (string, error) {
	return s.store.Get(ctx, id)
}

// Ping проверяет доступность хранилища.
func (s *URLService) Ping(ctx context.Context) error {
	return s.store.Ping(ctx)
}

func generateID(rnd *rand.Rand) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	bp := idBufPool.Get().(*[]byte)
	b := *bp
	for i := range b {
		b[i] = letters[rnd.Intn(len(letters))]
	}
	id := string(b)
	idBufPool.Put(bp)
	return id
}

// ShortenBatch сокращает несколько URL за одну операцию.
func (s *URLService) ShortenBatch(ctx context.Context, urls []BatchRequestItem, userID string) ([]BatchResponseItem, error) {
	logger.Log.Info("ShortenBatch called", zap.Int("count", len(urls)))

	if len(urls) == 0 {
		return nil, errors.New("empty urls")
	}

	records := make([]storage.Record, 0, len(urls))
	response := make([]BatchResponseItem, len(urls))

	for _, item := range urls {
		if strings.TrimSpace(item.OriginalURL) == "" {
			return nil, ErrEmptyURL
		}
	}

	ids := make([]string, len(urls))
	s.mu.Lock()
	for i := range urls {
		ids[i] = generateID(s.rnd)
	}
	s.mu.Unlock()

	for i, item := range urls {
		id := ids[i]
		records = append(records, storage.Record{
			ID:          id,
			OriginalURL: item.OriginalURL,
			UserID:      userID,
		})
		response[i] = BatchResponseItem{
			CorrelationID: item.CorrelationID,
			ShortURL:      s.urlPrefix + id,
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

// GetByUserID возвращает все ссылки, созданные указанным пользователем.
func (s *URLService) GetByUserID(ctx context.Context, userID string) ([]storage.Record, error) {
	return s.store.GetByUserID(ctx, userID)
}

// BaseURL возвращает базовый URL сервиса.
func (s *URLService) BaseURL() string {
	return s.baseURL
}

// DeleteURLs асинхронно помечает ссылки пользователя как удалённые.
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
		}
	}()

	return nil
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
