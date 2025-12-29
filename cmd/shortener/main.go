package main

import (
	"log"
	"net/http"

	"github.com/glebb1331/shortener-practicum/internal/config"
	"github.com/glebb1331/shortener-practicum/internal/handler"
	"github.com/glebb1331/shortener-practicum/internal/logger"
	"github.com/glebb1331/shortener-practicum/internal/middleware"
	"github.com/go-chi/chi/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
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

	store := initStorage(cfg)
	defer store.Close()

	r := chi.NewRouter()

	r.Use(middleware.WithLogging)
	r.Use(middleware.WithGzip)

	h, err := handler.NewHandler(cfg.BaseURL, store)
	if err != nil {
		log.Fatal(err)
	}

	r.Get("/ping", h.PingHandler)
	r.Post("/api/shorten", h.APIShortenHandler)
	r.Post("/api/shorten/batch", h.APIShortenBatchHandler)

	// Эндпоинт для истории URL пользователя
	r.Get("/api/user/urls", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Если данных нет, по спецификации часто требуется 204 No Content
		w.WriteHeader(http.StatusNoContent)
	})

	// В самом конце — жадный поиск по ID
	r.Post("/", h.ShortenHandler)
	r.Get("/{id}", h.RedirectHandler)

	log.Println("Сервер запущен", cfg.ServerAddress)
	log.Fatal(http.ListenAndServe(cfg.ServerAddress, r))
}
