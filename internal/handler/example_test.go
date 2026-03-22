package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/glebb1331/shortener-practicum/internal/audit"
	"github.com/glebb1331/shortener-practicum/internal/middleware"
	"github.com/glebb1331/shortener-practicum/internal/storage"
	"github.com/glebb1331/shortener-practicum/internal/usecase"
	"github.com/go-chi/chi/v5"
)

func newTestRouter() *chi.Mux {
	store := storage.NewMemoryStorage()
	h := NewHandler(usecase.NewURLService(store, "http://localhost:8080"), audit.NewAuditService())
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

// ExampleHandler_ShortenHandler демонстрирует сокращение URL через POST /.
func ExampleHandler_ShortenHandler() {
	r := newTestRouter()

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("https://example.com"))
	req.Header.Set("Content-Type", "text/plain")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)

	fmt.Println(res.StatusCode)
	fmt.Println(strings.HasPrefix(string(body), "http://localhost:8080/"))
	// Output:
	// 201
	// true
}

// ExampleHandler_APIShortenHandler демонстрирует сокращение URL через POST /api/shorten.
func ExampleHandler_APIShortenHandler() {
	r := newTestRouter()

	body := `{"url":"https://example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	var resp ShortenResponse
	json.NewDecoder(res.Body).Decode(&resp)

	fmt.Println(res.StatusCode)
	fmt.Println(strings.HasPrefix(resp.Result, "http://localhost:8080/"))
	// Output:
	// 201
	// true
}

// ExampleHandler_APIShortenBatchHandler демонстрирует пакетное сокращение через POST /api/shorten/batch.
func ExampleHandler_APIShortenBatchHandler() {
	r := newTestRouter()

	items := []usecase.BatchRequestItem{
		{CorrelationID: "1", OriginalURL: "https://a.com"},
		{CorrelationID: "2", OriginalURL: "https://b.com"},
	}
	data, _ := json.Marshal(items)
	req := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	var resp []usecase.BatchResponseItem
	json.NewDecoder(res.Body).Decode(&resp)

	fmt.Println(res.StatusCode)
	fmt.Println(len(resp))
	// Output:
	// 201
	// 2
}

// ExampleHandler_PingHandler демонстрирует проверку доступности хранилища через GET /ping.
func ExampleHandler_PingHandler() {
	r := newTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	fmt.Println(rec.Code)
	// Output:
	// 200
}

// ExampleHandler_GetUserURLs демонстрирует получение ссылок пользователя через GET /api/user/urls.
func ExampleHandler_GetUserURLs() {
	r := newTestRouter()

	// Создаём ссылку
	req1 := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("https://example.com"))
	req1.Header.Set("Content-Type", "text/plain")
	rec1 := httptest.NewRecorder()
	r.ServeHTTP(rec1, req1)

	// Берём куку для авторизации
	res1 := rec1.Result()
	defer res1.Body.Close()
	cookie := res1.Cookies()

	req2 := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
	for _, c := range cookie {
		req2.AddCookie(c)
	}
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, req2)

	fmt.Println(rec2.Code)
	// Output:
	// 200
}

// ExampleHandler_DeleteUserURLs демонстрирует удаление ссылок через DELETE /api/user/urls.
func ExampleHandler_DeleteUserURLs() {
	r := newTestRouter()

	ids, _ := json.Marshal([]string{"someid"})
	req := httptest.NewRequest(http.MethodDelete, "/api/user/urls", bytes.NewReader(ids))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	fmt.Println(rec.Code)
	// Output:
	// 202
}
