package handler

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/glebb1331/shortener-practicum/internal/middleware"
	"github.com/glebb1331/shortener-practicum/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupRouter() (*chi.Mux, *Handler) {
	store := storage.NewMemoryStorage()
	h, err := NewHandler("http://localhost:8080", store)
	if err != nil {
		panic(err)
	}
	r := chi.NewRouter()
	r.Post("/", h.ShortenHandler)
	r.Post("/api/shorten", h.APIShortenHandler)
	r.Get("/{id}", h.RedirectHandler)
	return r, h
}

func TestShortenHandler(t *testing.T) {
	store := storage.NewMemoryStorage()
	h, err := NewHandler("http://localhost:8080", store)
	require.NoError(t, err)

	r := chi.NewRouter()
	r.Post("/", h.ShortenHandler)

	type want struct {
		statusCode  int
		checkBody   bool
		contentType string
	}

	tests := []struct {
		name        string
		method      string
		contentType string
		body        string
		want        want
	}{
		{
			name:        "valid request",
			method:      http.MethodPost,
			contentType: "text/plain",
			body:        "https://practicum.yandex.ru/",
			want: want{
				statusCode:  http.StatusCreated,
				checkBody:   true,
				contentType: "text/plain",
			},
		},
		{
			name:        "invalid method",
			method:      http.MethodGet,
			contentType: "text/plain",
			body:        "https://practicum.yandex.ru/",
			want: want{
				statusCode: http.StatusMethodNotAllowed,
			},
		},
		{
			name:        "invalid content type",
			method:      http.MethodPost,
			contentType: "application/json",
			body:        "https://practicum.yandex.ru/",
			want: want{
				statusCode: http.StatusBadRequest,
			},
		},
		{
			name:        "empty body",
			method:      http.MethodPost,
			contentType: "text/plain",
			body:        "",
			want: want{
				statusCode: http.StatusBadRequest,
			},
		},
		{
			name:        "body with spaces only",
			method:      http.MethodPost,
			contentType: "text/plain",
			body:        "   ",
			want: want{
				statusCode: http.StatusBadRequest,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/", bytes.NewBufferString(tt.body))
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}

			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)
			res := rec.Result()
			defer res.Body.Close()

			assert.Equal(t, tt.want.statusCode, res.StatusCode)

			body, err := io.ReadAll(res.Body)
			require.NoError(t, err)

			if tt.want.checkBody {

				assert.True(t, strings.HasPrefix(string(body), "http://localhost:8080/"))
				assert.Equal(t, tt.want.contentType, res.Header.Get("Content-Type"))
			}
		})
	}
}

func TestAPIShortenHandler_WithGzip(t *testing.T) {
	store := storage.NewMemoryStorage()
	h, err := NewHandler("http://localhost:8080", store)
	require.NoError(t, err)

	r := chi.NewRouter()
	r.Use(middleware.WithGzip)
	r.Post("/api/shorten", h.APIShortenHandler)

	t.Run("gzip compressed request", func(t *testing.T) {

		jsonBody := `{"url":"https://practicum.yandex.ru"}`
		var buf bytes.Buffer
		gzWriter := gzip.NewWriter(&buf)
		gzWriter.Write([]byte(jsonBody))
		gzWriter.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/shorten", &buf)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Content-Encoding", "gzip")
		req.Header.Set("Accept-Encoding", "gzip")

		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		res := rec.Result()
		defer res.Body.Close()

		assert.Equal(t, http.StatusCreated, res.StatusCode)
		assert.Equal(t, "gzip", res.Header.Get("Content-Encoding"))
		assert.Equal(t, "application/json", res.Header.Get("Content-Type"))

		gz, err := gzip.NewReader(res.Body)
		require.NoError(t, err)
		defer gz.Close()

		var resp ShortenResponse
		err = json.NewDecoder(gz).Decode(&resp)
		require.NoError(t, err)

		assert.True(t, strings.HasPrefix(resp.Result, "http://localhost:8080/"))
	})

	t.Run("gzip compressed response", func(t *testing.T) {
		jsonBody := `{"url":"https://practicum.yandex.ru"}`

		req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewBufferString(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept-Encoding", "gzip")

		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		res := rec.Result()
		defer res.Body.Close()

		assert.Equal(t, http.StatusCreated, res.StatusCode)
		assert.Equal(t, "gzip", res.Header.Get("Content-Encoding"))

		gz, err := gzip.NewReader(res.Body)
		require.NoError(t, err)
		defer gz.Close()

		var resp ShortenResponse
		err = json.NewDecoder(gz).Decode(&resp)
		require.NoError(t, err)

		assert.True(t, strings.HasPrefix(resp.Result, "http://localhost:8080/"))
	})
}

func TestRedirectHandler(t *testing.T) {
	r, _ := setupRouter()

	originalURL := "https://example.com"
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(originalURL))
	req.Header.Set("Content-Type", "text/plain")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	res := rec.Result()
	defer res.Body.Close()
	require.Equal(t, http.StatusCreated, res.StatusCode)

	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	shortURL := strings.TrimSpace(string(body))
	id := strings.TrimPrefix(shortURL, "http://localhost:8080/")

	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantLoc    string
	}{
		{
			name:       "correctly id",
			path:       "/" + id,
			wantStatus: http.StatusTemporaryRedirect,
			wantLoc:    originalURL,
		},
		{
			name:       "non-existing",
			path:       "/notexistid",
			wantStatus: http.StatusNotFound,
			wantLoc:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)
			res := rec.Result()
			defer res.Body.Close()

			assert.Equal(t, tt.wantStatus, res.StatusCode)
			if tt.wantLoc != "" {
				assert.Equal(t, tt.wantLoc, res.Header.Get("Location"))
			}
		})
	}
}

func TestAPIShortenHandler(t *testing.T) {
	r, _ := setupRouter()

	type want struct {
		statusCode  int
		checkBody   bool
		contentType string
	}

	tests := []struct {
		name        string
		method      string
		contentType string
		body        string
		want        want
	}{
		{
			name:        "valid JSON request",
			method:      http.MethodPost,
			contentType: "application/json",
			body:        `{"url":"https://practicum.yandex.ru"}`,
			want: want{
				statusCode:  http.StatusCreated,
				checkBody:   true,
				contentType: "application/json",
			},
		},
		{
			name:        "invalid content type",
			method:      http.MethodPost,
			contentType: "text/plain",
			body:        `{"url":"https://practicum.yandex.ru"}`,
			want: want{
				statusCode: http.StatusBadRequest,
			},
		},
		{
			name:        "invalid JSON",
			method:      http.MethodPost,
			contentType: "application/json",
			body:        `{"url":"https://practicum.yandex.ru"`,
			want: want{
				statusCode: http.StatusBadRequest,
			},
		},
		{
			name:        "empty URL",
			method:      http.MethodPost,
			contentType: "application/json",
			body:        `{"url":""}`,
			want: want{
				statusCode: http.StatusBadRequest,
			},
		},
		{
			name:        "URL with spaces only",
			method:      http.MethodPost,
			contentType: "application/json",
			body:        `{"url":"   "}`,
			want: want{
				statusCode: http.StatusBadRequest,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/api/shorten", bytes.NewBufferString(tt.body))
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}

			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)
			res := rec.Result()
			defer res.Body.Close()

			assert.Equal(t, tt.want.statusCode, res.StatusCode)

			if tt.want.checkBody {
				assert.Equal(t, tt.want.contentType, res.Header.Get("Content-Type"))

				var resp ShortenResponse
				err := json.NewDecoder(res.Body).Decode(&resp)
				require.NoError(t, err)

				assert.True(t, strings.HasPrefix(resp.Result, "http://localhost:8080/"))
				assert.NotEqual(t, "http://localhost:8080/", resp.Result)
			}
		})
	}
}
