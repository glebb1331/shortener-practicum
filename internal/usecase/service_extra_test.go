package usecase

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/glebb1331/shortener-practicum/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestURLService_Stats(t *testing.T) {
	svc := newService()
	ctx := context.Background()

	urls, users, err := svc.Stats(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, urls)
	assert.Equal(t, 0, users)

	_, err = svc.Shorten(ctx, "https://a.example", "u1")
	require.NoError(t, err)
	_, err = svc.Shorten(ctx, "https://b.example", "u1")
	require.NoError(t, err)
	_, err = svc.Shorten(ctx, "https://c.example", "u2")
	require.NoError(t, err)

	urls, users, err = svc.Stats(ctx)
	require.NoError(t, err)
	assert.Equal(t, 3, urls)
	assert.Equal(t, 2, users)
}

func TestURLService_DeleteURLs_Async(t *testing.T) {
	svc := newService()
	ctx := context.Background()

	id1, err := svc.Shorten(ctx, "https://a.example", "u1")
	require.NoError(t, err)
	id2, err := svc.Shorten(ctx, "https://b.example", "u1")
	require.NoError(t, err)
	id3, err := svc.Shorten(ctx, "https://c.example", "u1")
	require.NoError(t, err)

	require.NoError(t, svc.DeleteURLs(ctx, "u1", []string{
		shortID(id1), shortID(id2), shortID(id3),
	}))

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		_, err := svc.Resolve(ctx, shortID(id1))
		if err == storage.ErrURLDeleted {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	_, err = svc.Resolve(ctx, shortID(id1))
	assert.ErrorIs(t, err, storage.ErrURLDeleted)
}

func TestFanInDelete_DrainsAllChannels(t *testing.T) {
	store := storage.NewMemoryStorage()
	svc := NewURLService(store, "http://localhost:8080")
	ctx := context.Background()

	id1, _ := svc.Shorten(ctx, "https://a.example", "u1")
	id2, _ := svc.Shorten(ctx, "https://b.example", "u1")

	ch1 := make(chan []string, 1)
	ch2 := make(chan []string, 1)
	ch1 <- []string{shortID(id1)}
	ch2 <- []string{shortID(id2)}
	close(ch1)
	close(ch2)

	out := svc.fanInDelete(ctx, "u1", []chan []string{ch1, ch2})

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for range out {
		}
	}()
	wg.Wait()

	_, err := svc.Resolve(ctx, shortID(id1))
	assert.ErrorIs(t, err, storage.ErrURLDeleted)
	_, err = svc.Resolve(ctx, shortID(id2))
	assert.ErrorIs(t, err, storage.ErrURLDeleted)
}

func shortID(shortURL string) string {
	const prefix = "http://localhost:8080/"
	return shortURL[len(prefix):]
}
