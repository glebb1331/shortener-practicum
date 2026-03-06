package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/glebb1331/shortener-practicum/internal/audit"
	"github.com/glebb1331/shortener-practicum/internal/middleware"
	"github.com/glebb1331/shortener-practicum/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupFullRouter() *chi.Mux {
	store := storage.NewMemoryStorage()
	h, err := NewHandler("http://localhost:8080", store, audit.NewAuditService())
	if err != nil {
		panic(err)
	}

	r := chi.NewRouter()
	r.Use(middleware.WithAuth)

	r.Post("/", h.ShortenHandler)
	r.Get("/{id}", h.RedirectHandler)
	r.Post("/api/shorten", h.APIShortenHandler)
	r.Post("/api/shorten/batch", h.APIShortenBatchHandler)
	r.Get("/ping", h.PingHandler)
	r.Get("/api/user/urls", h.GetUserURLs)
	r.Delete("/api/user/urls", h.DeleteUserURLs)

	return r
}

func TestPingHandler(t *testing.T) {
	r := setupFullRouter()

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAPIShortenBatchHandler(t *testing.T) {
	r := setupFullRouter()

	items := []BatchRequestItem{
		{CorrelationID: "1", OriginalURL: "https://a.com"},
		{CorrelationID: "2", OriginalURL: "https://b.com"},
	}
	body, _ := json.Marshal(items)

	req := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp []BatchResponseItem
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Len(t, resp, 2)
}

func TestAPIShortenBatchHandler_WrongContentType(t *testing.T) {
	r := setupFullRouter()

	req := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", strings.NewReader("[]"))
	req.Header.Set("Content-Type", "text/plain")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAPIShortenBatchHandler_EmptyBatch(t *testing.T) {
	r := setupFullRouter()

	req := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", strings.NewReader("[]"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAPIShortenBatchHandler_InvalidJSON(t *testing.T) {
	r := setupFullRouter()

	req := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", strings.NewReader("{invalid}"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestGetUserURLs_NoURLs(t *testing.T) {
	r := setupFullRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	// 204 because user exists (set by WithAuth) but has no URLs
	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestGetUserURLs_WithURLs(t *testing.T) {
	store := storage.NewMemoryStorage()
	h, err := NewHandler("http://localhost:8080", store, audit.NewAuditService())
	require.NoError(t, err)

	r := chi.NewRouter()
	r.Use(middleware.WithAuth)
	r.Post("/", h.ShortenHandler)
	r.Get("/api/user/urls", h.GetUserURLs)

	// First create a short URL
	req1 := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("https://example.com"))
	req1.Header.Set("Content-Type", "text/plain")
	rec1 := httptest.NewRecorder()
	r.ServeHTTP(rec1, req1)
	require.Equal(t, http.StatusCreated, rec1.Code)

	// Reuse cookie from first request
	res1 := rec1.Result()
	defer res1.Body.Close()
	cookie := res1.Cookies()

	req2 := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
	for _, c := range cookie {
		req2.AddCookie(c)
	}
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, req2)

	assert.Equal(t, http.StatusOK, rec2.Code)
}

func TestDeleteUserURLs_Valid(t *testing.T) {
	r := setupFullRouter()

	body, _ := json.Marshal([]string{"someid"})
	req := httptest.NewRequest(http.MethodDelete, "/api/user/urls", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusAccepted, rec.Code)
}

func TestDeleteUserURLs_WrongMethod(t *testing.T) {
	store := storage.NewMemoryStorage()
	h, err := NewHandler("http://localhost:8080", store, audit.NewAuditService())
	require.NoError(t, err)

	r := chi.NewRouter()
	r.Use(middleware.WithAuth)
	r.Post("/api/user/urls", h.DeleteUserURLs)

	req := httptest.NewRequest(http.MethodPost, "/api/user/urls", nil)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}

func TestDeleteUserURLs_InvalidJSON(t *testing.T) {
	r := setupFullRouter()

	req := httptest.NewRequest(http.MethodDelete, "/api/user/urls", strings.NewReader("{bad}"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
