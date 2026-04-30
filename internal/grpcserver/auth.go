package grpcserver

import (
	"context"
	"strings"

	"github.com/glebb1331/shortener-practicum/internal/logger"
	"github.com/glebb1331/shortener-practicum/internal/middleware"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// authMetadataKey — имя метаданных, в которых клиент передаёт JWT-токен.
const authMetadataKey = "authorization"

type ctxKey int

const userIDKey ctxKey = iota

// UserIDFromContext извлекает userID, установленный AuthInterceptor'ом.
func UserIDFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(userIDKey).(string)
	return v, ok && v != ""
}

// withUserID кладёт userID в контекст.
func withUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// AuthInterceptor возвращает gRPC unary interceptor, который проверяет JWT-токен
// в метаданных authorization. Если токен отсутствует или невалиден, генерирует
// нового пользователя и возвращает выпущенный токен в заголовке authorization.
func AuthInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		userID, tokenStr, issued, err := resolveUser(ctx)
		if err != nil {
			return nil, err
		}

		if issued {
			if mdErr := grpc.SendHeader(ctx, metadata.Pairs(authMetadataKey, tokenStr)); mdErr != nil {
				logger.Log.Error("failed to send auth header", zap.Error(mdErr))
			}
		}

		return handler(withUserID(ctx, userID), req)
	}
}

// resolveUser извлекает userID из метаданных или создаёт нового пользователя.
// issued=true означает, что был выпущен новый токен и его необходимо вернуть клиенту.
func resolveUser(ctx context.Context) (userID string, tokenStr string, issued bool, err error) {
	md, _ := metadata.FromIncomingContext(ctx)
	if values := md.Get(authMetadataKey); len(values) > 0 {
		token := strings.TrimSpace(values[0])
		token = strings.TrimPrefix(token, "Bearer ")
		token = strings.TrimPrefix(token, "bearer ")
		if uid, ok := middleware.ParseToken(token); ok {
			return uid, token, false, nil
		}
		return "", "", false, status.Error(codes.Unauthenticated, "invalid token")
	}

	uid, genErr := middleware.GenerateUserID()
	if genErr != nil {
		return "", "", false, status.Error(codes.Internal, "failed to generate user id")
	}
	token, tokErr := middleware.CreateToken(uid)
	if tokErr != nil {
		return "", "", false, status.Error(codes.Internal, "failed to create token")
	}
	return uid, token, true, nil
}
