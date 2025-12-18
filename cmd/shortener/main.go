package main

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/glebb1331/shortener-practicum/internal/config"
	"github.com/glebb1331/shortener-practicum/internal/handler"
	"github.com/glebb1331/shortener-practicum/internal/logger"
	"github.com/glebb1331/shortener-practicum/internal/middleware"
	"github.com/glebb1331/shortener-practicum/internal/storage"
	"github.com/go-chi/chi/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	if err := logger.Initialize("info"); err != nil {
		log.Fatal("Failed to initialize logger:", err)
	}
	defer logger.Log.Sync()

	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatal(err)
	}

	var store storage.Storage

	if cfg.DatabaseDSN != "" {
		db, err := sql.Open("pgx", cfg.DatabaseDSN)
		if err != nil {
			log.Printf("Failed to open database: %v, falling back to file storage", err)
		} else {
			store, err = storage.NewDatabaseStorage(db)
			if err != nil {
				log.Printf("Failed to initialize database storage: %v, falling back to file storage", err)
				db.Close()
			} else {
				log.Println("Using PostgreSQL storage")
				defer store.Close()
			}
		}
	}

	if store == nil && cfg.FileStoragePath != "" {
		fileStore, fileErr := storage.NewFileStorage(cfg.FileStoragePath)
		if fileErr != nil {
			log.Printf("Failed to initialize file storage: %v, falling back to memory storage", fileErr)
			store = nil
		} else {
			store = fileStore
			log.Println("Using file storage")
			defer store.Close()
		}
	}

	if store == nil {
		store = storage.NewMemoryStorage()
		log.Println("Using in-memory storage")
		defer store.Close()
	}

	r := chi.NewRouter()

	r.Use(middleware.WithLogging)
	r.Use(middleware.WithGzip)

	h, err := handler.NewHandler(cfg.BaseURL, store)
	if err != nil {
		log.Fatal(err)
	}

	r.Get("/ping", h.PingHandler)

	r.Post("/", h.ShortenHandler)
	r.Post("/api/shorten", h.APIShortenHandler)

	r.Get("/{id}", h.RedirectHandler)

	log.Println("Сервер запущен", cfg.ServerAddress)
	log.Fatal(http.ListenAndServe(cfg.ServerAddress, r))
}
