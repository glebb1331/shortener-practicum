package storage

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryStorage_SaveAndGet(t *testing.T) {
	store := NewMemoryStorage()
	ctx := context.Background()

	id, err := store.Save(ctx, "abc123", "https://example.com", "user1")
	require.NoError(t, err)
	assert.Equal(t, "abc123", id)

	url, err := store.Get(ctx, "abc123")
	require.NoError(t, err)
	assert.Equal(t, "https://example.com", url)
}

func TestMemoryStorage_SaveDuplicate(t *testing.T) {
	store := NewMemoryStorage()
	ctx := context.Background()

	_, err := store.Save(ctx, "abc123", "https://example.com", "user1")
	require.NoError(t, err)

	id, err := store.Save(ctx, "xyz789", "https://example.com", "user1")
	assert.ErrorIs(t, err, ErrURLExists)
	assert.Equal(t, "abc123", id)
}

func TestMemoryStorage_GetNotFound(t *testing.T) {
	store := NewMemoryStorage()
	ctx := context.Background()

	_, err := store.Get(ctx, "nonexistent")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestMemoryStorage_GetDeleted(t *testing.T) {
	store := NewMemoryStorage()
	ctx := context.Background()

	store.Save(ctx, "abc123", "https://example.com", "user1")
	store.DeleteURLs(ctx, "user1", []string{"abc123"})

	_, err := store.Get(ctx, "abc123")
	assert.ErrorIs(t, err, ErrURLDeleted)
}

func TestMemoryStorage_GetByUserID(t *testing.T) {
	store := NewMemoryStorage()
	ctx := context.Background()

	store.Save(ctx, "id1", "https://a.com", "user1")
	store.Save(ctx, "id2", "https://b.com", "user1")
	store.Save(ctx, "id3", "https://c.com", "user2")

	records, err := store.GetByUserID(ctx, "user1")
	require.NoError(t, err)
	assert.Len(t, records, 2)
}

func TestMemoryStorage_GetByUserID_Empty(t *testing.T) {
	store := NewMemoryStorage()
	ctx := context.Background()

	records, err := store.GetByUserID(ctx, "nobody")
	require.NoError(t, err)
	assert.Empty(t, records)
}

func TestMemoryStorage_BatchSave(t *testing.T) {
	store := NewMemoryStorage()
	ctx := context.Background()

	records := []Record{
		{ID: "id1", OriginalURL: "https://a.com", UserID: "user1"},
		{ID: "id2", OriginalURL: "https://b.com", UserID: "user1"},
	}
	err := store.BatchSave(ctx, records)
	require.NoError(t, err)

	url, err := store.Get(ctx, "id1")
	require.NoError(t, err)
	assert.Equal(t, "https://a.com", url)
}

func TestMemoryStorage_BatchSave_SkipDuplicate(t *testing.T) {
	store := NewMemoryStorage()
	ctx := context.Background()

	store.Save(ctx, "id1", "https://a.com", "user1")

	records := []Record{
		{ID: "id2", OriginalURL: "https://a.com", UserID: "user1"}, // duplicate URL
	}
	err := store.BatchSave(ctx, records)
	require.NoError(t, err)

	_, err = store.Get(ctx, "id2")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestMemoryStorage_DeleteURLs_WrongUser(t *testing.T) {
	store := NewMemoryStorage()
	ctx := context.Background()

	store.Save(ctx, "id1", "https://a.com", "user1")
	store.DeleteURLs(ctx, "user2", []string{"id1"})

	url, err := store.Get(ctx, "id1")
	require.NoError(t, err)
	assert.Equal(t, "https://a.com", url)
}

func TestMemoryStorage_GetByOriginalURL(t *testing.T) {
	store := NewMemoryStorage()
	ctx := context.Background()

	store.Save(ctx, "id1", "https://a.com", "user1")

	id, err := store.GetByOriginalURL(ctx, "https://a.com")
	require.NoError(t, err)
	assert.Equal(t, "id1", id)
}

func TestMemoryStorage_GetByOriginalURL_NotFound(t *testing.T) {
	store := NewMemoryStorage()
	ctx := context.Background()

	_, err := store.GetByOriginalURL(ctx, "https://notexists.com")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestMemoryStorage_Ping(t *testing.T) {
	store := NewMemoryStorage()
	assert.NoError(t, store.Ping(context.Background()))
}

func TestMemoryStorage_Close(t *testing.T) {
	store := NewMemoryStorage()
	assert.NoError(t, store.Close())
}

func TestMemoryStorage_GetBatchByUserID(t *testing.T) {
	store := NewMemoryStorage()
	ctx := context.Background()

	store.Save(ctx, "id1", "https://a.com", "user1")
	store.Save(ctx, "id2", "https://b.com", "user1")
	store.Save(ctx, "id3", "https://c.com", "user2")

	records, err := store.GetBatchByUserID(ctx, "user1", []string{"id1", "id3"})
	require.NoError(t, err)
	assert.Len(t, records, 1)
	assert.Equal(t, "id1", records[0].ID)
}
