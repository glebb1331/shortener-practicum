package handler

import (
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

func newHandler(t *testing.T) *Handler {
	t.Helper()
	store := storage.NewMemoryStorage()
	return NewHandler(usecase.NewURLService(store, "http://localhost:8080"), audit.NewAuditService())
}

func TestSetTrustedSubnet(t *testing.T) {
	h := newHandler(t)

	require.NoError(t, h.SetTrustedSubnet(""))
	assert.Nil(t, h.trustedSubnet)

	require.NoError(t, h.SetTrustedSubnet("10.0.0.0/8"))
	assert.NotNil(t, h.trustedSubnet)

	require.Error(t, h.SetTrustedSubnet("not-a-cidr"))
}

func TestInternalStatsHandler_NoSubnet(t *testing.T) {
	h := newHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/api/internal/stats", nil)
	rec := httptest.NewRecorder()
	h.InternalStatsHandler(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestInternalStatsHandler_OutsideSubnet(t *testing.T) {
	h := newHandler(t)
	require.NoError(t, h.SetTrustedSubnet("10.0.0.0/8"))

	req := httptest.NewRequest(http.MethodGet, "/api/internal/stats", nil)
	req.Header.Set("X-Real-IP", "192.168.1.1")
	rec := httptest.NewRecorder()
	h.InternalStatsHandler(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestInternalStatsHandler_InvalidIP(t *testing.T) {
	h := newHandler(t)
	require.NoError(t, h.SetTrustedSubnet("10.0.0.0/8"))

	req := httptest.NewRequest(http.MethodGet, "/api/internal/stats", nil)
	req.Header.Set("X-Real-IP", "not-an-ip")
	rec := httptest.NewRecorder()
	h.InternalStatsHandler(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestInternalStatsHandler_OK(t *testing.T) {
	store := storage.NewMemoryStorage()
	svc := usecase.NewURLService(store, "http://localhost:8080")
	_, err := svc.Shorten(httptest.NewRequest(http.MethodGet, "/", nil).Context(), "https://a.example", "user-1")
	require.NoError(t, err)

	h := NewHandler(svc, audit.NewAuditService())
	require.NoError(t, h.SetTrustedSubnet("10.0.0.0/8"))

	req := httptest.NewRequest(http.MethodGet, "/api/internal/stats", nil)
	req.Header.Set("X-Real-IP", "10.1.2.3")
	rec := httptest.NewRecorder()
	h.InternalStatsHandler(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json"))

	var resp StatsResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, 1, resp.URLs)
	assert.Equal(t, 1, resp.Users)
}
