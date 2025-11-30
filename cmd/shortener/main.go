package main

import (
	"log"
	"net/http"

	"github.com/glebb1331/shortener-practicum/internal/config"
	"github.com/glebb1331/shortener-practicum/internal/handler"
	"github.com/glebb1331/shortener-practicum/internal/logger"
	"github.com/glebb1331/shortener-practicum/internal/middleware"
	"github.com/go-chi/chi/v5"
)

func main() {
	if err := logger.Initialize("info"); err != nil {
		log.Fatal("Failed to initialize logger:", err)
	}
	defer logger.Log.Sync()

	cfg := config.NewConfig()

	r := chi.NewRouter()

	// Добавляем middleware для логирования и gzip
	r.Use(middleware.WithLogging)
	r.Use(middleware.WithGzip)

	h := handler.NewHandler(cfg.BaseURL)

	r.Post("/", h.ShortenHandler)
	r.Post("/api/shorten", h.APIShortenHandler)

	r.Get("/{id}", h.RedirectHandler)

	log.Println("Сервер запущен", cfg.ServerAddress)
	log.Fatal(http.ListenAndServe(cfg.ServerAddress, r))
}
