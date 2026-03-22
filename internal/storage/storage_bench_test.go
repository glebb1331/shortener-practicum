package storage

import (
	"context"
	"fmt"
	"testing"
)

func BenchmarkMemoryStorageSave(b *testing.B) {
	store := NewMemoryStorage()
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id := fmt.Sprintf("id%d", i)
		url := fmt.Sprintf("https://example.com/%d", i)
		_, _ = store.Save(ctx, id, url, "user1")
	}
}

func BenchmarkGet(b *testing.B) {
	store := NewMemoryStorage()
	ctx := context.Background()
	for i := 0; i < 1000; i++ {
		id := fmt.Sprintf("id%d", i)
		url := fmt.Sprintf("https://example.com/%d", i)
		store.Save(ctx, id, url, "user1")
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id := fmt.Sprintf("id%d", i%1000)
		_, _ = store.Get(ctx, id)
	}
}

func BenchmarkGetByUserID(b *testing.B) {
	store := NewMemoryStorage()
	ctx := context.Background()
	for i := 0; i < 1000; i++ {
		id := fmt.Sprintf("id%d", i)
		url := fmt.Sprintf("https://example.com/%d", i)
		userID := "user1"
		if i%2 == 0 {
			userID = "user2"
		}
		store.Save(ctx, id, url, userID)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = store.GetByUserID(ctx, "user1")
	}
}
