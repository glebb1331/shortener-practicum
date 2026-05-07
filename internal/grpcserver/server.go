// Package grpcserver реализует gRPC-фасад над бизнес-логикой сервиса сокращения ссылок.
package grpcserver

import (
	"context"
	"errors"
	"net/url"

	"github.com/glebb1331/shortener-practicum/internal/audit"
	"github.com/glebb1331/shortener-practicum/internal/logger"
	"github.com/glebb1331/shortener-practicum/internal/storage"
	"github.com/glebb1331/shortener-practicum/internal/usecase"
	pb "github.com/glebb1331/shortener-practicum/pkg/shortenerpb"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ShortenerServer реализует gRPC-сервис ShortenerService.
// Делегирует выполнение операций в общий слой бизнес-логики (usecase.URLService),
// чтобы HTTP- и gRPC-обработчики были фасадами к одному и тому же коду.
type ShortenerServer struct {
	pb.UnimplementedShortenerServiceServer

	service  *usecase.URLService
	auditSvc *audit.AuditService
}

// NewShortenerServer создаёт реализацию gRPC-сервиса.
func NewShortenerServer(svc *usecase.URLService, auditSvc *audit.AuditService) *ShortenerServer {
	return &ShortenerServer{service: svc, auditSvc: auditSvc}
}

// ShortenURL сокращает оригинальный URL и возвращает короткую ссылку.
func (s *ShortenerServer) ShortenURL(ctx context.Context, req *pb.URLShortenRequest) (*pb.URLShortenResponse, error) {
	userID, ok := UserIDFromContext(ctx)
	if !ok || userID == "" {
		return nil, status.Error(codes.Unauthenticated, "missing user id")
	}

	result, err := s.service.Shorten(ctx, req.GetUrl(), userID)
	if err != nil {
		if errors.Is(err, usecase.ErrEmptyURL) {
			return nil, status.Error(codes.InvalidArgument, "empty url")
		}
		if errors.Is(err, storage.ErrURLExists) {
			s.auditSvc.Notify(audit.AuditEvent{Action: "shorten", UserID: userID, URL: req.GetUrl()})
			return nil, status.Error(codes.AlreadyExists, result)
		}
		logger.Log.Error("grpc shorten failed", zap.Error(err), zap.String("userID", userID))
		return nil, status.Error(codes.Internal, "internal error")
	}

	s.auditSvc.Notify(audit.AuditEvent{Action: "shorten", UserID: userID, URL: req.GetUrl()})
	return pb.URLShortenResponse_builder{Result: result}.Build(), nil
}

// ExpandURL возвращает оригинальный URL по короткому идентификатору.
func (s *ShortenerServer) ExpandURL(ctx context.Context, req *pb.URLExpandRequest) (*pb.URLExpandResponse, error) {
	id := req.GetId()
	if id == "" {
		return nil, status.Error(codes.InvalidArgument, "empty id")
	}

	originalURL, err := s.service.Resolve(ctx, id)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "url not found")
		}
		if errors.Is(err, storage.ErrURLDeleted) {
			return nil, status.Error(codes.NotFound, "url deleted")
		}
		logger.Log.Error("grpc expand failed", zap.Error(err), zap.String("id", id))
		return nil, status.Error(codes.Internal, "internal error")
	}

	if userID, ok := UserIDFromContext(ctx); ok {
		s.auditSvc.Notify(audit.AuditEvent{Action: "follow", UserID: userID, URL: originalURL})
	}
	return pb.URLExpandResponse_builder{Result: originalURL}.Build(), nil
}

// ListUserURLs возвращает все ссылки текущего пользователя.
func (s *ShortenerServer) ListUserURLs(ctx context.Context, _ *pb.ListUserURLsRequest) (*pb.UserURLsResponse, error) {
	userID, ok := UserIDFromContext(ctx)
	if !ok || userID == "" {
		return nil, status.Error(codes.Unauthenticated, "missing user id")
	}

	records, err := s.service.GetByUserID(ctx, userID)
	if err != nil {
		logger.Log.Error("grpc list user urls failed", zap.Error(err), zap.String("userID", userID))
		return nil, status.Error(codes.Internal, "internal error")
	}

	urls := make([]*pb.URLData, 0, len(records))
	for _, rec := range records {
		shortURL, err := url.JoinPath(s.service.BaseURL(), rec.ID)
		if err != nil {
			logger.Log.Error("grpc list user urls join failed", zap.Error(err))
			continue
		}
		urls = append(urls, pb.URLData_builder{
			ShortUrl:    shortURL,
			OriginalUrl: rec.OriginalURL,
		}.Build())
	}
	return pb.UserURLsResponse_builder{Url: urls}.Build(), nil
}
