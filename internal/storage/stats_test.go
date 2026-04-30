package storage

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryStorage_Stats(t *testing.T) {
	store := NewMemoryStorage()
	ctx := context.Background()

	urls, users, err := store.Stats(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, urls)
	assert.Equal(t, 0, users)

	_, err = store.Save(ctx, "id1", "https://a.example", "u1")
	require.NoError(t, err)
	_, err = store.Save(ctx, "id2", "https://b.example", "u1")
	require.NoError(t, err)
	_, err = store.Save(ctx, "id3", "https://c.example", "u2")
	require.NoError(t, err)

	urls, users, err = store.Stats(ctx)
	require.NoError(t, err)
	assert.Equal(t, 3, urls)
	assert.Equal(t, 2, users)
}

func TestFileStorage_Stats(t *testing.T) {
	dir := t.TempDir()
	s, err := NewFileStorage(filepath.Join(dir, "store.json"))
	require.NoError(t, err)
	ctx := context.Background()

	_, err = s.Save(ctx, "id1", "https://a.example", "u1")
	require.NoError(t, err)
	_, err = s.Save(ctx, "id2", "https://b.example", "u2")
	require.NoError(t, err)

	urls, users, err := s.Stats(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, urls)
	assert.Equal(t, 2, users)
}
