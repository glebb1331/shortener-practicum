package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/glebb1331/shortener-practicum/internal/logger"
	"github.com/golang-jwt/jwt/v4"
	"go.uber.org/zap"
)

const cookieName = "token"

var secretKey = []byte("super-secret-key")

// Claims — JWT-клеймы с идентификатором пользователя.
type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

func generateUserID() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "", fmt.Errorf("failed to generate secure user ID: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func createToken(userID string) (string, error) {
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secretKey)
}

func parseToken(tokenStr string) (*Claims, bool) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return secretKey, nil
	})
	if err != nil || !token.Valid {
		return nil, false
	}
	claims, ok := token.Claims.(*Claims)
	return claims, ok
}

// GenerateUserID создаёт случайный идентификатор пользователя.
func GenerateUserID() (string, error) {
	return generateUserID()
}

// CreateToken подписывает JWT-токен с заданным userID.
func CreateToken(userID string) (string, error) {
	return createToken(userID)
}

// ParseToken разбирает JWT-токен и возвращает userID.
// Второй результат — флаг валидности токена.
func ParseToken(tokenStr string) (string, bool) {
	claims, ok := parseToken(tokenStr)
	if !ok || claims == nil {
		return "", false
	}
	return claims.UserID, true
}

// WithAuth — middleware аутентификации на основе JWT-куки.
// Если кука отсутствует, создаёт нового пользователя и устанавливает куку.
// Передаёт userID в заголовке X-User-ID.
func WithAuth(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var tokenStr string

		c, err := r.Cookie(cookieName)
		if err != nil {
			userID, err := generateUserID()
			if err != nil {
				logger.Log.Error("Failed to generate user ID", zap.Error(err))
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
			tokenStr, err = createToken(userID)
			if err != nil {
				logger.Log.Error("Failed to create token", zap.Error(err))
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}

			http.SetCookie(w, &http.Cookie{
				Name:     cookieName,
				Value:    tokenStr,
				Path:     "/",
				Expires:  time.Now().Add(24 * time.Hour),
				HttpOnly: true,
			})
			r.Header.Set("X-User-ID", userID)
		} else {
			tokenStr = c.Value
			claims, ok := parseToken(tokenStr)
			if !ok || claims.UserID == "" {
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}
			r.Header.Set("X-User-ID", claims.UserID)
		}
		h.ServeHTTP(w, r)
	})
}
