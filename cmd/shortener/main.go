package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"

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

	logger.Log.Info(
		"server started",
		zap.String("address", cfg.ServerAddress),
		zap.Bool("https", cfg.EnableHTTPS),
	)

	if cfg.EnableHTTPS {
		tlsConfig, err := tlscert.SelfSignedTLSConfig()
		if err != nil {
			log.Fatal("Failed to create TLS config:", err)
		}
		listener, err := tls.Listen("tcp", cfg.ServerAddress, tlsConfig)
		if err != nil {
			log.Fatal("Failed to start TLS listener:", err)
		}
		defer listener.Close()
		log.Fatal(http.Serve(listener, r))
	} else {
		listener, err := net.Listen("tcp", cfg.ServerAddress)
		if err != nil {
			log.Fatal("Failed to start listener:", err)
		}
		defer listener.Close()
		log.Fatal(http.Serve(listener, r))
	}
}
