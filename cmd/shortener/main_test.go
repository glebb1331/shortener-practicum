package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/glebb1331/shortener-practicum/internal/audit"
	"github.com/glebb1331/shortener-practicum/internal/config"
	"github.com/glebb1331/shortener-practicum/internal/logger"
	"github.com/glebb1331/shortener-practicum/internal/storage"
	"github.com/glebb1331/shortener-practicum/internal/usecase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	_ = logger.Initialize("error")
}

func TestValueOrNA(t *testing.T) {
	assert.Equal(t, "N/A", valueOrNA(""))
	assert.Equal(t, "1.2.3", valueOrNA("1.2.3"))
}

func TestPrintBuildInfo(t *testing.T) {
	var buf bytes.Buffer
	printBuildInfo(&buf)
	out := buf.String()
	assert.Contains(t, out, "Build version:")
	assert.Contains(t, out, "Build date:")
	assert.Contains(t, out, "Build commit:")
}

func TestNewRouter_RoutesRespond(t *testing.T) {
	store := storage.NewMemoryStorage()
	svc := usecase.NewURLService(store, "http://localhost:8080")
	r := newRouter(svc, audit.NewAuditService(), "")

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	req = httptest.NewRequest(http.MethodPost, "/", strings.NewReader("https://example.com"))
	req.Header.Set("Content-Type", "text/plain")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusCreated, rec.Code)
}

func TestNewRouter_BadTrustedSubnet(t *testing.T) {
	store := storage.NewMemoryStorage()
	svc := usecase.NewURLService(store, "http://localhost:8080")

	// Невалидный CIDR не должен вызывать панику — только ошибка логируется.
	r := newRouter(svc, audit.NewAuditService(), "not-a-cidr")
	require.NotNil(t, r)
}

func TestInitStorage_Memory(t *testing.T) {
	cfg := &config.Config{}
	store, err := initStorage(cfg)
	require.NoError(t, err)
	require.NotNil(t, store)
	assert.NoError(t, store.Close())
}

func TestInitStorage_File(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{FileStoragePath: dir + "/store.json"}
	store, err := initStorage(cfg)
	require.NoError(t, err)
	require.NotNil(t, store)
	assert.NoError(t, store.Close())
}

func TestInitStorage_DatabaseInvalidDSN(t *testing.T) {
	cfg := &config.Config{DatabaseDSN: "this is not a valid dsn"}
	_, err := initStorage(cfg)
	assert.Error(t, err)
}
