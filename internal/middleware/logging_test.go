package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithLogging(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("hello"))
	})

	handler := WithLogging(inner)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Equal(t, "hello", rec.Body.String())
}

func TestLoggingResponseWriter_WriteHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	rd := &responseData{}
	lw := &loggingResponseWriter{ResponseWriter: rec, responseData: rd}

	lw.WriteHeader(http.StatusNotFound)

	assert.Equal(t, http.StatusNotFound, rd.status)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestLoggingResponseWriter_Write(t *testing.T) {
	rec := httptest.NewRecorder()
	rd := &responseData{}
	lw := &loggingResponseWriter{ResponseWriter: rec, responseData: rd}

	n, err := lw.Write([]byte("test data"))
	assert.NoError(t, err)
	assert.Equal(t, 9, n)
	assert.Equal(t, 9, rd.size)
	assert.Equal(t, "test data", rec.Body.String())
}
