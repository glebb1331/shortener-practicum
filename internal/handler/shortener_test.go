package handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShortenHandler(t *testing.T) {
	type want struct {
		statusCode  int
		body        string
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
				body:        "http://localhost:8080/id1",
				contentType: "text/plain",
			},
		},
		{
			name:        "invalid method",
			method:      http.MethodGet,
			contentType: "text/plain",
			body:        "https://practicum.yandex.ru/",
			want: want{
				statusCode: http.StatusBadRequest,
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
			req := httptest.NewRequest(tt.method, "/", strings.NewReader(tt.body))
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}

			w := httptest.NewRecorder()
			ShortenHandler(w, req)

			res := w.Result()
			defer res.Body.Close()

			assert.Equal(t, tt.want.statusCode, res.StatusCode)

			bodyBytes, err := io.ReadAll(res.Body)
			require.NoError(t, err)

			if tt.want.statusCode == http.StatusCreated {
				assert.Equal(t, tt.want.body, string(bodyBytes))
				assert.Equal(t, tt.want.contentType, res.Header.Get("Content-Type"))
			}
		})
	}
}
