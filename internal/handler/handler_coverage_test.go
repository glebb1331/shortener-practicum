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
	"github.com/glebb1331/shortener-practicum/internal/usecase"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupRouterFull() *chi.Mux {
	store := storage.NewMemoryStorage()
	h := NewHandler(usecase.NewURLService(store, "http://localhost:8080"), audit.NewAuditService())

	r := chi.NewRouter()
	r.Use(middleware.WithAuth)

	r.Get("/ping", h.PingHandler)
	r.Post("/", h.ShortenHandler)
	r.Get("/{id}", h.RedirectHandler)
	r.Post("/api/shorten", h.APIShortenHandler)
	r.Post("/api/shorten/batch", h.APIShortenBatchHandler)
	r.Get("/api/user/urls", h.GetUserURLs)
	r.Delete("/api/user/urls", h.DeleteUserURLs)

	return r
}

func TestPingHandler_OK(t *testing.T) {
	r := setupRouterFull()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestGetUserURLs_NoContent(t *testing.T) {
	r := setupRouterFull()
	req := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	// У нового пользователя нет ссылок.
	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestGetUserURLs_HasURLs(t *testing.T) {
	store := storage.NewMemoryStorage()
	svc := usecase.NewURLService(store, "http://localhost:8080")
	h := NewHandler(svc, audit.NewAuditService())

	r := chi.NewRouter()
	r.Use(middleware.WithAuth)
	r.Post("/", h.ShortenHandler)
	r.Get("/api/user/urls", h.GetUserURLs)

	// Сначала сокращаем URL, чтобы у пользователя была хотя бы одна запись.
	postReq := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("https://example.com"))
	postReq.Header.Set("Content-Type", "text/plain")
	postRec := httptest.NewRecorder()
	r.ServeHTTP(postRec, postReq)
	require.Equal(t, http.StatusCreated, postRec.Code)

	// Извлекаем cookie пользователя, чтобы переиспользовать ту же идентичность.
	var userCookie *http.Cookie
	for _, c := range postRec.Result().Cookies() {
		if c.Name == "token" {
			userCookie = c
		}
	}
	require.NotNil(t, userCookie)

	getReq := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
	getReq.AddCookie(userCookie)
	getRec := httptest.NewRecorder()
	r.ServeHTTP(getRec, getReq)

	assert.Equal(t, http.StatusOK, getRec.Code)
}

func TestDeleteUserURLs_Success(t *testing.T) {
	store := storage.NewMemoryStorage()
	svc := usecase.NewURLService(store, "http://localhost:8080")
	h := NewHandler(svc, audit.NewAuditService())

	r := chi.NewRouter()
	r.Use(middleware.WithAuth)
	r.Post("/", h.ShortenHandler)
	r.Delete("/api/user/urls", h.DeleteUserURLs)

	// Сокращаем URL, чтобы получить короткий ID и cookie с токеном.
	postReq := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("https://delete-me.com"))
	postReq.Header.Set("Content-Type", "text/plain")
	postRec := httptest.NewRecorder()
	r.ServeHTTP(postRec, postReq)
	require.Equal(t, http.StatusCreated, postRec.Code)

	shortURL := strings.TrimSpace(postRec.Body.String())
	parts := strings.Split(shortURL, "/")
	shortID := parts[len(parts)-1]

	var userCookie *http.Cookie
	for _, c := range postRec.Result().Cookies() {
		if c.Name == "token" {
			userCookie = c
		}
	}
	require.NotNil(t, userCookie)

	body, _ := json.Marshal([]string{shortID})
	delReq := httptest.NewRequest(http.MethodDelete, "/api/user/urls", bytes.NewReader(body))
	delReq.Header.Set("Content-Type", "application/json")
	delReq.AddCookie(userCookie)
	delRec := httptest.NewRecorder()
	r.ServeHTTP(delRec, delReq)

	assert.Equal(t, http.StatusAccepted, delRec.Code)
}

func TestDeleteUserURLs_BadContentType(t *testing.T) {
	r := setupRouterFull()
	req := httptest.NewRequest(http.MethodDelete, "/api/user/urls", strings.NewReader(`["id1"]`))
	req.Header.Set("Content-Type", "text/plain")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestDeleteUserURLs_EmptyIDs(t *testing.T) {
	r := setupRouterFull()
	body, _ := json.Marshal([]string{})
	req := httptest.NewRequest(http.MethodDelete, "/api/user/urls", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAPIShortenBatchHandler_BadContentTypeCov(t *testing.T) {
	r := setupRouterFull()
	req := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", strings.NewReader("[]"))
	req.Header.Set("Content-Type", "text/plain")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
