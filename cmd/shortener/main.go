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
	"syscall"
	"time"

	"github.com/glebb1331/shortener-practicum/internal/audit"
	"github.com/glebb1331/shortener-practicum/internal/config"
	"github.com/glebb1331/shortener-practicum/internal/handler"
	"github.com/glebb1331/shortener-practicum/internal/logger"
	"github.com/glebb1331/shortener-practicum/internal/middleware"
	"github.com/glebb1331/shortener-practicum/internal/tlscert"
	"github.com/glebb1331/shortener-practicum/internal/usecase"
	"github.com/go-chi/chi/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"
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

func newRouter(svc *usecase.URLService, auditSvc *audit.AuditService) http.Handler {
	h := handler.NewHandler(svc, auditSvc)

	r := chi.NewRouter()
	r.Use(middleware.WithLogging)
	r.Use(middleware.WithGzip)
	r.Use(middleware.WithAuth)

	r.Get("/ping", h.PingHandler)
	r.Post("/api/shorten", h.APIShortenHandler)
	r.Post("/api/shorten/batch", h.APIShortenBatchHandler)
	r.Get("/api/user/urls", h.GetUserURLs)
	r.Delete("/api/user/urls", h.DeleteUserURLs)
	r.Post("/", h.ShortenHandler)
	r.Get("/{id}", h.RedirectHandler)

	return r
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
	r := newRouter(svc, auditSvc)

	srv := &http.Server{
		Handler: r,
	}

	// Канал для получения сигналов завершения.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

	logger.Log.Info(
		"server started",
		zap.String("address", cfg.ServerAddress),
		zap.Bool("https", cfg.EnableHTTPS),
	)

	var listener net.Listener
	if cfg.EnableHTTPS {
		tlsConfig, tlsErr := tlscert.SelfSignedTLSConfig()
		if tlsErr != nil {
			log.Fatal("Failed to create TLS config:", tlsErr)
		}
		listener, err = tls.Listen("tcp", cfg.ServerAddress, tlsConfig)
	} else {
		listener, err = net.Listen("tcp", cfg.ServerAddress)
	}
	if err != nil {
		log.Fatal("Failed to start listener:", err)
	}

	// Запускаем сервер в отдельной горутине.
	go func() {
		if serveErr := srv.Serve(listener); serveErr != nil && serveErr != http.ErrServerClosed {
			log.Fatal("Server error:", serveErr)
		}
	}()

	// Ожидаем сигнал завершения.
	sig := <-quit
	logger.Log.Info("shutting down server", zap.String("signal", sig.String()))

	// Даём серверу время на завершение текущих запросов.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Log.Error("server shutdown error", zap.Error(err))
	}

	logger.Log.Info("server stopped gracefully")
}
