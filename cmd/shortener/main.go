package main

import (
	"log"
	"net/http"

	"github.com/glebb1331/shortener-practicum/internal/audit"
	"github.com/glebb1331/shortener-practicum/internal/config"
	"github.com/glebb1331/shortener-practicum/internal/handler"
	"github.com/glebb1331/shortener-practicum/internal/logger"
	"github.com/glebb1331/shortener-practicum/internal/middleware"
	"github.com/go-chi/chi/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"
)

func main() {

	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatal(err)
	}

	if err := logger.Initialize("info"); err != nil {
		log.Fatal("Failed to initialize logger:", err)
	}
	defer logger.Log.Sync()

	store, err := initStorage(cfg)
	if err != nil {
		logger.Log.Fatal("Failed to initialize logger")
	}
	defer store.Close()

	auditSvc := audit.NewAuditService()
	if cfg.AuditFile != "" {
		auditSvc.Register(audit.NewFileObserver(cfg.AuditFile))
	}
	if cfg.AuditURL != "" {
		auditSvc.Register(audit.NewHTTPObserver(cfg.AuditURL))
	}

	r := chi.NewRouter()

	r.Use(middleware.WithLogging)
	r.Use(middleware.WithGzip)
	r.Use(middleware.WithAuth)

	h, err := handler.NewHandler(cfg.BaseURL, store, auditSvc)
	if err != nil {
		log.Fatal(err)
	}

	r.Get("/ping", h.PingHandler)
	r.Post("/api/shorten", h.APIShortenHandler)
	r.Post("/api/shorten/batch", h.APIShortenBatchHandler)

	r.Get("/api/user/urls", h.GetUserURLs)
	r.Delete("/api/user/urls", h.DeleteUserURLs)

	r.Post("/", h.ShortenHandler)
	r.Get("/{id}", h.RedirectHandler)

	logger.Log.Info(
		"server started",
		zap.String("address", cfg.ServerAddress),
	)
	log.Fatal(http.ListenAndServe(cfg.ServerAddress, r))
}
