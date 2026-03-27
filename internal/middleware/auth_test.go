package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/glebb1331/shortener-practicum/internal/logger"
	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	// Инициализируем логгер один раз, чтобы WithAuth не падал с паникой.
	_ = logger.Initialize("error")
}

func TestGenerateUserID(t *testing.T) {
	id1, err := generateUserID()
	require.NoError(t, err)
	assert.Len(t, id1, 32)

	id2, err := generateUserID()
	require.NoError(t, err)
	assert.NotEqual(t, id1, id2, "two generated IDs should differ")
}

func TestCreateAndParseToken(t *testing.T) {
	userID := "test-user-42"

	tokenStr, err := createToken(userID)
	require.NoError(t, err)
	assert.NotEmpty(t, tokenStr)

	claims, ok := parseToken(tokenStr)
	require.True(t, ok)
	assert.Equal(t, userID, claims.UserID)
}

func TestParseToken_Invalid(t *testing.T) {
	_, ok := parseToken("not.a.valid.token")
	assert.False(t, ok)
}

func TestParseToken_Expired(t *testing.T) {
	claims := &Claims{
		UserID: "u1",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := tok.SignedString(secretKey)
	require.NoError(t, err)

	_, ok := parseToken(tokenStr)
	assert.False(t, ok)
}

func TestWithAuth_NoCookie(t *testing.T) {
	var capturedUserID string
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUserID = r.Header.Get("X-User-ID")
		w.WriteHeader(http.StatusOK)
	})

	handler := WithAuth(inner)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.NotEmpty(t, capturedUserID, "X-User-ID should be set for new user")

	// В ответе должна быть установлена cookie с токеном.
	cookies := rec.Result().Cookies()
	var found bool
	for _, c := range cookies {
		if c.Name == cookieName {
			found = true
		}
	}
	assert.True(t, found, "token cookie should be set")
}

func TestWithAuth_ValidCookie(t *testing.T) {
	userID := "existing-user"
	tokenStr, err := createToken(userID)
	require.NoError(t, err)

	var capturedUserID string
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUserID = r.Header.Get("X-User-ID")
		w.WriteHeader(http.StatusOK)
	})

	handler := WithAuth(inner)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: cookieName, Value: tokenStr})

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, userID, capturedUserID)
}

func TestWithAuth_InvalidCookie(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := WithAuth(inner)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: cookieName, Value: "bad-token"})

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
