package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedirectHandler(t *testing.T) {

	urlStore = map[string]string{
		"id1": "https://practicum.yandex.ru/",
	}

	type want struct {
		statusCode int
		location   string
	}
	tests := []struct {
		name   string
		method string
		path   string
		want   want
	}{
		{
			name:   "valid request",
			method: http.MethodGet,
			path:   "/id1",
			want: want{
				statusCode: http.StatusTemporaryRedirect,
				location:   "https://practicum.yandex.ru/",
			},
		},
		{
			name:   "invalid method",
			method: http.MethodPost,
			path:   "/id1",
			want: want{
				statusCode: http.StatusBadRequest,
				location:   "https://practicum.yandex.ru/",
			},
		},
		{
			name:   "invalid path",
			method: http.MethodGet,
			path:   "/id-invalid",
			want: want{
				statusCode: http.StatusBadRequest,
				location:   "https://practicum.yandex.ru/",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			BaseURL = "http://localhost:8080"

			RedirectHandler(w, req)

			res := w.Result()
			defer res.Body.Close()

			assert.Equal(t, tt.want.statusCode, res.StatusCode)

			loc := res.Header.Get("Location")
			require.NotNil(t, tt.want.location, loc)
		})
	}
}
