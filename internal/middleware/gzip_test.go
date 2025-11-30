package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithGzip_Compression(t *testing.T) {
	tests := []struct {
		name           string
		acceptEncoding string
		contentType    string
		expectGzip     bool
	}{
		{
			name:           "compress JSON with gzip support",
			acceptEncoding: "gzip",
			contentType:    "application/json",
			expectGzip:     true,
		},
		{
			name:           "compress HTML with gzip support",
			acceptEncoding: "gzip",
			contentType:    "text/html",
			expectGzip:     true,
		},
		{
			name:           "no compression without Accept-Encoding",
			acceptEncoding: "",
			contentType:    "application/json",
			expectGzip:     false,
		},
		{
			name:           "no compression for text/plain",
			acceptEncoding: "gzip",
			contentType:    "text/plain",
			expectGzip:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем handler с нужным Content-Type
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", tt.contentType)
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("test response data"))
			})

			middlewareHandler := WithGzip(testHandler)

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.acceptEncoding != "" {
				req.Header.Set("Accept-Encoding", tt.acceptEncoding)
			}

			rec := httptest.NewRecorder()
			middlewareHandler.ServeHTTP(rec, req)

			res := rec.Result()
			defer res.Body.Close()

			if tt.expectGzip {
				assert.Equal(t, "gzip", res.Header.Get("Content-Encoding"))

				// Проверяем что данные действительно сжаты
				gz, err := gzip.NewReader(res.Body)
				require.NoError(t, err)
				defer gz.Close()

				decompressed, err := io.ReadAll(gz)
				require.NoError(t, err)
				assert.Equal(t, "test response data", string(decompressed))
			} else {
				assert.NotEqual(t, "gzip", res.Header.Get("Content-Encoding"))

				body, err := io.ReadAll(res.Body)
				require.NoError(t, err)
				assert.Equal(t, "test response data", string(body))
			}
		})
	}
}

func TestWithGzip_Decompression(t *testing.T) {
	receivedBody := ""

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		receivedBody = string(body)
		w.WriteHeader(http.StatusOK)
	})

	middlewareHandler := WithGzip(handler)

	// Создаем gzip-сжатое тело запроса
	originalBody := `{"url":"https://practicum.yandex.ru"}`
	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)
	gzWriter.Write([]byte(originalBody))
	gzWriter.Close()

	req := httptest.NewRequest(http.MethodPost, "/", &buf)
	req.Header.Set("Content-Encoding", "gzip")

	rec := httptest.NewRecorder()
	middlewareHandler.ServeHTTP(rec, req)

	assert.Equal(t, originalBody, receivedBody)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestWithGzip_InvalidGzipData(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middlewareHandler := WithGzip(handler)

	// Отправляем невалидные gzip данные
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("invalid gzip data"))
	req.Header.Set("Content-Encoding", "gzip")

	rec := httptest.NewRecorder()
	middlewareHandler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
