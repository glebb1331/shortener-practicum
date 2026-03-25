package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTempFileStorage(t *testing.T) (*FileStorage, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.json")
	s, err := NewFileStorage(path)
	require.NoError(t, err)
	return s, path
}

func TestFileStorage_SaveAndGet(t *testing.T) {
	s, _ := newTempFileStorage(t)
	ctx := context.Background()

	id, err := s.Save(ctx, "short1", "https://example.com", "user1")
	require.NoError(t, err)
	assert.Equal(t, "short1", id)

	url, err := s.Get(ctx, "short1")
	require.NoError(t, err)
	assert.Equal(t, "https://example.com", url)
}

func TestFileStorage_SaveDuplicate(t *testing.T) {
	s, _ := newTempFileStorage(t)
	ctx := context.Background()

	_, err := s.Save(ctx, "short1", "https://example.com", "user1")
	require.NoError(t, err)

	id, err := s.Save(ctx, "short2", "https://example.com", "user1")
	assert.ErrorIs(t, err, ErrURLExists)
	assert.Equal(t, "short1", id)
}

func TestFileStorage_GetNotFound(t *testing.T) {
	s, _ := newTempFileStorage(t)

	_, err := s.Get(context.Background(), "nonexistent")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestFileStorage_GetDeleted(t *testing.T) {
	s, _ := newTempFileStorage(t)
	ctx := context.Background()

	_, err := s.Save(ctx, "s1", "https://a.com", "u1")
	require.NoError(t, err)

	err = s.DeleteURLs(ctx, "u1", []string{"s1"})
	require.NoError(t, err)

	_, err = s.Get(ctx, "s1")
	assert.ErrorIs(t, err, ErrURLDeleted)
}

func TestFileStorage_PersistsAcrossReload(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "store.json")
	ctx := context.Background()

	// Записываем через первый экземпляр.
	s1, err := NewFileStorage(path)
	require.NoError(t, err)
	_, err = s1.Save(ctx, "s1", "https://persist.com", "u1")
	require.NoError(t, err)
	require.NoError(t, s1.Close())

	// Читаем через второй экземпляр.
	s2, err := NewFileStorage(path)
	require.NoError(t, err)
	url, err := s2.Get(ctx, "s1")
	require.NoError(t, err)
	assert.Equal(t, "https://persist.com", url)
}

func TestFileStorage_Ping(t *testing.T) {
	s, _ := newTempFileStorage(t)
	assert.NoError(t, s.Ping(context.Background()))
}

func TestFileStorage_Close(t *testing.T) {
	s, _ := newTempFileStorage(t)
	assert.NoError(t, s.Close())
}

func TestFileStorage_BatchSave(t *testing.T) {
	s, _ := newTempFileStorage(t)
	ctx := context.Background()

	records := []Record{
		{ID: "b1", OriginalURL: "https://batch1.com", UserID: "u1"},
		{ID: "b2", OriginalURL: "https://batch2.com", UserID: "u1"},
	}
	require.NoError(t, s.BatchSave(ctx, records))

	url, err := s.Get(ctx, "b1")
	require.NoError(t, err)
	assert.Equal(t, "https://batch1.com", url)
}

func TestFileStorage_BatchSave_SkipDuplicate(t *testing.T) {
	s, _ := newTempFileStorage(t)
	ctx := context.Background()

	_, err := s.Save(ctx, "s1", "https://a.com", "u1")
	require.NoError(t, err)

	// Батч с уже существующим коротким ID.
	records := []Record{
		{ID: "s1", OriginalURL: "https://should-be-skipped.com", UserID: "u1"},
		{ID: "s2", OriginalURL: "https://new.com", UserID: "u1"},
	}
	require.NoError(t, s.BatchSave(ctx, records))

	// Оригинальный URL должен остаться без изменений.
	url, err := s.Get(ctx, "s1")
	require.NoError(t, err)
	assert.Equal(t, "https://a.com", url)
}

func TestFileStorage_GetByOriginalURL(t *testing.T) {
	s, _ := newTempFileStorage(t)
	ctx := context.Background()

	_, err := s.Save(ctx, "s1", "https://find-me.com", "u1")
	require.NoError(t, err)

	id, err := s.GetByOriginalURL(ctx, "https://find-me.com")
	require.NoError(t, err)
	assert.Equal(t, "s1", id)
}

func TestFileStorage_GetByOriginalURL_NotFound(t *testing.T) {
	s, _ := newTempFileStorage(t)

	_, err := s.GetByOriginalURL(context.Background(), "https://not-there.com")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestFileStorage_GetByUserID(t *testing.T) {
	s, _ := newTempFileStorage(t)
	ctx := context.Background()

	s.Save(ctx, "s1", "https://a.com", "u1")
	s.Save(ctx, "s2", "https://b.com", "u1")
	s.Save(ctx, "s3", "https://c.com", "u2")

	records, err := s.GetByUserID(ctx, "u1")
	require.NoError(t, err)
	assert.Len(t, records, 2)
}

func TestFileStorage_DeleteURLs_WrongUser(t *testing.T) {
	s, _ := newTempFileStorage(t)
	ctx := context.Background()

	_, err := s.Save(ctx, "s1", "https://a.com", "u1")
	require.NoError(t, err)

	require.NoError(t, s.DeleteURLs(ctx, "u2", []string{"s1"}))

	url, err := s.Get(ctx, "s1")
	require.NoError(t, err)
	assert.Equal(t, "https://a.com", url)
}

func TestFileStorage_DeleteURLs_EmptyList(t *testing.T) {
	s, _ := newTempFileStorage(t)
	ctx := context.Background()

	require.NoError(t, s.DeleteURLs(ctx, "u1", nil))
}

func TestFileStorage_GetBatchByUserID(t *testing.T) {
	s, _ := newTempFileStorage(t)
	ctx := context.Background()

	s.Save(ctx, "s1", "https://a.com", "u1")
	s.Save(ctx, "s2", "https://b.com", "u1")
	s.Save(ctx, "s3", "https://c.com", "u2")

	records, err := s.GetBatchByUserID(ctx, "u1", []string{"s1", "s3"})
	require.NoError(t, err)
	assert.Len(t, records, 1)
	assert.Equal(t, "s1", records[0].ID)
}

func TestNewFileStorage_InvalidDir(t *testing.T) {
	// Передаём путь, где компонент директории является файлом.
	tmpFile, err := os.CreateTemp("", "notadir")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	_, err = NewFileStorage(filepath.Join(tmpFile.Name(), "sub", "store.json"))
	assert.Error(t, err)
}
