package usecase

import (
	"context"
	"testing"

	"github.com/glebb1331/shortener-practicum/internal/storage"
	"github.com/stretchr/testify/assert"
)

func TestFanInDelete(t *testing.T) {
	store := storage.NewMemoryStorage()
	ctx := context.Background()

	store.Save(ctx, "id1", "https://a.com", "u1")
	store.Save(ctx, "id2", "https://b.com", "u1")

	svc := NewURLService(store, "http://localhost:8080")

	ch1 := make(chan []string, 1)
	ch2 := make(chan []string, 1)

	ch1 <- []string{"id1"}
	ch2 <- []string{"id2"}
	close(ch1)
	close(ch2)

	errCh := svc.fanInDelete(ctx, "u1", []chan []string{ch1, ch2})

	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}

	assert.Empty(t, errs)

	_, err := store.Get(ctx, "id1")
	assert.ErrorIs(t, err, storage.ErrURLDeleted)

	_, err = store.Get(ctx, "id2")
	assert.ErrorIs(t, err, storage.ErrURLDeleted)
}

func TestFanInDelete_Empty(t *testing.T) {
	svc := NewURLService(storage.NewMemoryStorage(), "http://localhost:8080")

	errCh := svc.fanInDelete(context.Background(), "u1", nil)
	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}
	assert.Empty(t, errs)
}
