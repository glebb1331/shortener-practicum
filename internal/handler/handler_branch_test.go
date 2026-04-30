package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/glebb1331/shortener-practicum/internal/audit"
	"github.com/glebb1331/shortener-practicum/internal/storage"
	"github.com/glebb1331/shortener-practicum/internal/usecase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedirectHandler_Deleted(t *testing.T) {
	store := storage.NewMemoryStorage()
	svc := usecase.NewURLService(store, "http://localhost:8080")

	ctx := context.Background()
	_, err := store.Save(ctx, "deletedid", "https://gone.example", "u1")
	require.NoError(t, err)
	require.NoError(t, store.DeleteURLs(ctx, "u1", []string{"deletedid"}))

	h := NewHandler(svc, audit.NewAuditService())
	req := httptest.NewRequest(http.MethodGet, "/deletedid", nil)
	rec := httptest.NewRecorder()
	h.RedirectHandler(rec, req)
	assert.Equal(t, http.StatusGone, rec.Code)
}

func TestAPIShortenHandler_Unauthorized(t *testing.T) {
	store := storage.NewMemoryStorage()
	h := NewHandler(usecase.NewURLService(store, "http://localhost:8080"), audit.NewAuditService())

	req := httptest.NewRequest(http.MethodPost, "/api/shorten",
		strings.NewReader(`{"url":"https://x.example"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.APIShortenHandler(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAPIShortenHandler_Conflict(t *testing.T) {
	store := storage.NewMemoryStorage()
	svc := usecase.NewURLService(store, "http://localhost:8080")
	h := NewHandler(svc, audit.NewAuditService())

	first := httptest.NewRequest(http.MethodPost, "/api/shorten",
		strings.NewReader(`{"url":"https://dup.example"}`))
	first.Header.Set("Content-Type", "application/json")
	first.Header.Set("X-User-ID", "u1")
	rec := httptest.NewRecorder()
	h.APIShortenHandler(rec, first)
	require.Equal(t, http.StatusCreated, rec.Code)

	second := httptest.NewRequest(http.MethodPost, "/api/shorten",
		strings.NewReader(`{"url":"https://dup.example"}`))
	second.Header.Set("Content-Type", "application/json")
	second.Header.Set("X-User-ID", "u1")
	rec = httptest.NewRecorder()
	h.APIShortenHandler(rec, second)
	assert.Equal(t, http.StatusConflict, rec.Code)

	var body ShortenResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.Contains(t, body.Result, "http://localhost:8080/")
}

func TestShortenHandler_Conflict(t *testing.T) {
	store := storage.NewMemoryStorage()
	svc := usecase.NewURLService(store, "http://localhost:8080")
	h := NewHandler(svc, audit.NewAuditService())

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("https://dup2.example"))
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("X-User-ID", "u1")
	rec := httptest.NewRecorder()
	h.ShortenHandler(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

	req = httptest.NewRequest(http.MethodPost, "/", strings.NewReader("https://dup2.example"))
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("X-User-ID", "u1")
	rec = httptest.NewRecorder()
	h.ShortenHandler(rec, req)
	assert.Equal(t, http.StatusConflict, rec.Code)
	assert.Contains(t, rec.Body.String(), "http://localhost:8080/")
}

func TestShortenHandler_Unauthorized(t *testing.T) {
	store := storage.NewMemoryStorage()
	h := NewHandler(usecase.NewURLService(store, "http://localhost:8080"), audit.NewAuditService())

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("https://x.example"))
	req.Header.Set("Content-Type", "text/plain")
	rec := httptest.NewRecorder()
	h.ShortenHandler(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestGetUserURLs_Unauthorized(t *testing.T) {
	store := storage.NewMemoryStorage()
	h := NewHandler(usecase.NewURLService(store, "http://localhost:8080"), audit.NewAuditService())

	req := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
	rec := httptest.NewRecorder()
	h.GetUserURLs(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestDeleteUserURLs_Unauthorized(t *testing.T) {
	store := storage.NewMemoryStorage()
	h := NewHandler(usecase.NewURLService(store, "http://localhost:8080"), audit.NewAuditService())

	req := httptest.NewRequest(http.MethodDelete, "/api/user/urls", strings.NewReader(`["a"]`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.DeleteUserURLs(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestDeleteUserURLs_EmptyList(t *testing.T) {
	store := storage.NewMemoryStorage()
	h := NewHandler(usecase.NewURLService(store, "http://localhost:8080"), audit.NewAuditService())

	req := httptest.NewRequest(http.MethodDelete, "/api/user/urls", strings.NewReader(`[]`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "u1")
	rec := httptest.NewRecorder()
	h.DeleteUserURLs(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAPIShortenBatchHandler_Unauthorized(t *testing.T) {
	store := storage.NewMemoryStorage()
	h := NewHandler(usecase.NewURLService(store, "http://localhost:8080"), audit.NewAuditService())

	req := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", strings.NewReader(`[]`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.APIShortenBatchHandler(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
