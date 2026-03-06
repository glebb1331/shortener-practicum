package handler

import (
	"context"
	"math/rand"
	"testing"

	"github.com/glebb1331/shortener-practicum/internal/storage"
)

func BenchmarkGenerateID(b *testing.B) {
	rnd := rand.New(rand.NewSource(42))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = generateID(rnd)
	}
}

func BenchmarkShorten(b *testing.B) {
	store := storage.NewMemoryStorage()
	svc := NewURLService(store, "http://localhost:8080")
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = svc.Shorten(ctx, "https://example.com/bench/"+generateID(svc.rnd), "user1")
	}
}

func BenchmarkShortenBatch(b *testing.B) {
	store := storage.NewMemoryStorage()
	svc := NewURLService(store, "http://localhost:8080")
	ctx := context.Background()

	urls := make([]BatchRequestItem, 10)
	for i := range urls {
		urls[i] = BatchRequestItem{CorrelationID: "id", OriginalURL: "https://example.com/"}
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		items := make([]BatchRequestItem, 10)
		for j := range items {
			items[j] = BatchRequestItem{
				CorrelationID: "cid",
				OriginalURL:   "https://example.com/bench/" + generateID(svc.rnd),
			}
		}
		_, _ = svc.ShortenBatch(ctx, items, "user1")
	}
}
