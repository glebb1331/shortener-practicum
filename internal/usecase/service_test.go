package usecase

import (
	"context"
	"testing"

	"github.com/glebb1331/shortener-practicum/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newService() *URLService {
	return NewURLService(storage.NewMemoryStorage(), "http://localhost:8080")
}

func TestURLService_Shorten(t *testing.T) {
	svc := newService()
	ctx := context.Background()

	url, err := svc.Shorten(ctx, "https://example.com", "user1")
	require.NoError(t, err)
	assert.Contains(t, url, "http://localhost:8080/")
}

func TestURLService_Shorten_EmptyURL(t *testing.T) {
	svc := newService()
	_, err := svc.Shorten(context.Background(), "   ", "user1")
	assert.ErrorIs(t, err, ErrEmptyURL)
}

func TestURLService_Shorten_Duplicate(t *testing.T) {
	svc := newService()
	ctx := context.Background()

	url1, _ := svc.Shorten(ctx, "https://example.com", "user1")
	url2, err := svc.Shorten(ctx, "https://example.com", "user1")
	assert.ErrorIs(t, err, storage.ErrURLExists)
	assert.Equal(t, url1, url2)
}

func TestURLService_Resolve(t *testing.T) {
	svc := newService()
	ctx := context.Background()

	shortURL, err := svc.Shorten(ctx, "https://example.com", "user1")
	require.NoError(t, err)

	id := shortURL[len("http://localhost:8080/"):]
	original, err := svc.Resolve(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, "https://example.com", original)
}

func TestURLService_Ping(t *testing.T) {
	svc := newService()
	assert.NoError(t, svc.Ping(context.Background()))
}

func TestURLService_BaseURL(t *testing.T) {
	svc := newService()
	assert.Equal(t, "http://localhost:8080", svc.BaseURL())
}

func TestURLService_GetByUserID(t *testing.T) {
	svc := newService()
	ctx := context.Background()

	svc.Shorten(ctx, "https://a.com", "user1")
	svc.Shorten(ctx, "https://b.com", "user1")
	svc.Shorten(ctx, "https://c.com", "user2")

	records, err := svc.GetByUserID(ctx, "user1")
	require.NoError(t, err)
	assert.Len(t, records, 2)
}

func TestURLService_ShortenBatch(t *testing.T) {
	svc := newService()
	ctx := context.Background()

	items := []BatchRequestItem{
		{CorrelationID: "1", OriginalURL: "https://a.com"},
		{CorrelationID: "2", OriginalURL: "https://b.com"},
	}
	resp, err := svc.ShortenBatch(ctx, items, "user1")
	require.NoError(t, err)
	assert.Len(t, resp, 2)
	assert.Contains(t, resp[0].ShortURL, "http://localhost:8080/")
	assert.Contains(t, resp[1].ShortURL, "http://localhost:8080/")
}

func TestURLService_ShortenBatch_Empty(t *testing.T) {
	svc := newService()
	_, err := svc.ShortenBatch(context.Background(), nil, "user1")
	assert.Error(t, err)
}

func TestURLService_ShortenBatch_EmptyURL(t *testing.T) {
	svc := newService()
	items := []BatchRequestItem{{CorrelationID: "1", OriginalURL: "   "}}
	_, err := svc.ShortenBatch(context.Background(), items, "user1")
	assert.ErrorIs(t, err, ErrEmptyURL)
}

func TestURLService_DeleteURLs(t *testing.T) {
	svc := newService()
	ctx := context.Background()

	shortURL, _ := svc.Shorten(ctx, "https://a.com", "user1")
	id := shortURL[len("http://localhost:8080/"):]

	err := svc.DeleteURLs(ctx, "user1", []string{id})
	assert.NoError(t, err)
}

func TestURLService_DeleteURLs_Empty(t *testing.T) {
	svc := newService()
	err := svc.DeleteURLs(context.Background(), "user1", nil)
	assert.Error(t, err)
}

func TestURLService_UrlPrefix_TrailingSlash(t *testing.T) {
	svc := NewURLService(storage.NewMemoryStorage(), "http://localhost:8080/")
	ctx := context.Background()

	url, err := svc.Shorten(ctx, "https://example.com", "user1")
	require.NoError(t, err)
	path := url[len("http://"):]
	assert.NotContains(t, path, "//")
}
