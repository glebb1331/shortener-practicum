package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/glebb1331/shortener-practicum/internal/audit"
	"github.com/glebb1331/shortener-practicum/internal/config"
	"github.com/glebb1331/shortener-practicum/internal/grpcserver"
	"github.com/glebb1331/shortener-practicum/internal/handler"
	"github.com/glebb1331/shortener-practicum/internal/logger"
	"github.com/glebb1331/shortener-practicum/internal/middleware"
	"github.com/glebb1331/shortener-practicum/internal/tlscert"
	"github.com/glebb1331/shortener-practicum/internal/usecase"
	pb "github.com/glebb1331/shortener-practicum/pkg/shortenerpb"
	"github.com/go-chi/chi/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

var buildVersion string
var buildDate string
var buildCommit string

func valueOrNA(s string) string {
	if s == "" {
		return "N/A"
	}
	return s
}

func printBuildInfo(w io.Writer) {
	fmt.Fprintf(w, "Build version: %s\n", valueOrNA(buildVersion))
	fmt.Fprintf(w, "Build date: %s\n", valueOrNA(buildDate))
	fmt.Fprintf(w, "Build commit: %s\n", valueOrNA(buildCommit))
}

func newRouter(svc *usecase.URLService, auditSvc *audit.AuditService, trustedSubnet string) (http.Handler, error) {
	h := handler.NewHandler(svc, auditSvc)
	if err := h.SetTrustedSubnet(trustedSubnet); err != nil {
		return nil, fmt.Errorf("invalid trusted subnet %q: %w", trustedSubnet, err)
	}

	r := chi.NewRouter()
	r.Use(middleware.WithLogging)
	r.Use(middleware.WithGzip)
	r.Use(middleware.WithAuth)

	r.Get("/ping", h.PingHandler)
	r.Post("/api/shorten", h.APIShortenHandler)
	r.Post("/api/shorten/batch", h.APIShortenBatchHandler)
	r.Get("/api/user/urls", h.GetUserURLs)
	r.Delete("/api/user/urls", h.DeleteUserURLs)
	// Эндпоинт /api/internal/stats подключаем только при заданной доверенной подсети.
	if subnet := h.TrustedSubnet(); subnet != nil {
		r.With(middleware.WithTrustedSubnet(subnet)).Get("/api/internal/stats", h.InternalStatsHandler)
	}
	r.Post("/", h.ShortenHandler)
	r.Get("/{id}", h.RedirectHandler)

	return r, nil
}

func main() {
	printBuildInfo(os.Stdout)

	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatal(err)
	}

	if err = logger.Initialize("info"); err != nil {
		log.Fatal("Failed to initialize logger:", err)
	}
	defer logger.Log.Sync()

	store, err := initStorage(cfg)
	if err != nil {
		logger.Log.Fatal("Failed to initialize storage")
	}
	defer store.Close()

	auditSvc := audit.NewAuditService()
	if cfg.AuditFile != "" {
		auditSvc.Register(audit.NewFileObserver(cfg.AuditFile))
	}
	if cfg.AuditURL != "" {
		auditSvc.Register(audit.NewHTTPObserver(cfg.AuditURL))
	}

	svc := usecase.NewURLService(store, cfg.BaseURL)
	r, err := newRouter(svc, auditSvc, cfg.TrustedSubnet)
	if err != nil {
		logger.Log.Fatal("failed to build router", zap.Error(err))
	}

	srv := &http.Server{
		Handler:      r,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Канал для получения сигналов завершения.
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

	logger.Log.Info(
		"server started",
		zap.String("address", cfg.ServerAddress),
		zap.Bool("https", cfg.EnableHTTPS),
	)

	var listener net.Listener
	if cfg.EnableHTTPS {
		tlsConfig, tlsErr := tlscert.SelfSignedTLSConfig()
		if tlsErr != nil {
			logger.Log.Fatal("Failed to create TLS config:", zap.Error(tlsErr))
		}
		listener, err = tls.Listen("tcp", cfg.ServerAddress, tlsConfig)
	} else {
		listener, err = net.Listen("tcp", cfg.ServerAddress)
	}
	if err != nil {
		logger.Log.Fatal("Failed to start listener:", zap.Error(err))
	}

	// Группа горутин серверов: дожидаемся их завершения перед выходом из main,
	// чтобы не оставить активные операции (логирование, обработка запросов) после возврата.
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		if serveErr := srv.Serve(listener); serveErr != nil && serveErr != http.ErrServerClosed {
			logger.Log.Error("HTTP server error", zap.Error(serveErr))
		}
	}()

	// Поднимаем gRPC-сервер, если задан адрес.
	var grpcSrv *grpc.Server
	if cfg.GRPCAddress != "" {
		grpcListener, grpcErr := net.Listen("tcp", cfg.GRPCAddress)
		if grpcErr != nil {
			logger.Log.Fatal("Failed to start gRPC listener", zap.Error(grpcErr))
		}
		grpcSrv = grpc.NewServer(grpc.UnaryInterceptor(grpcserver.AuthInterceptor()))
		pb.RegisterShortenerServiceServer(grpcSrv, grpcserver.NewShortenerServer(svc, auditSvc))

		logger.Log.Info("grpc server started", zap.String("address", cfg.GRPCAddress))
		wg.Add(1)
		go func() {
			defer wg.Done()
			if serveErr := grpcSrv.Serve(grpcListener); serveErr != nil {
				logger.Log.Error("gRPC server error", zap.Error(serveErr))
			}
		}()
	}

	// Ожидаем сигнал завершения.
	sig := <-sigChan
	logger.Log.Info("shutting down server", zap.String("signal", sig.String()))

	// Даём серверу время на завершение текущих запросов.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Log.Error("server shutdown error", zap.Error(err))
	}

	if grpcSrv != nil {
		stopped := make(chan struct{})
		go func() {
			grpcSrv.GracefulStop()
			close(stopped)
		}()
		select {
		case <-stopped:
		case <-ctx.Done():
			grpcSrv.Stop()
		}
	}

	// Дожидаемся, пока все Serve-горутины полностью завершатся.
	wg.Wait()

	logger.Log.Info("server stopped gracefully")
}
