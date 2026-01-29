package main

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/glebb1331/shortener-practicum/internal/config"
	"github.com/glebb1331/shortener-practicum/internal/storage"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func initStorage(cfg *config.Config) (storage.Storage, error) {
	if cfg.DatabaseDSN != "" {
		db, err := sql.Open("pgx", cfg.DatabaseDSN)
		if err != nil {
			return nil, fmt.Errorf("database connection error: %w", err)
		}

		store, err := storage.NewDatabaseStorage(db)
		if err != nil {
			return nil, fmt.Errorf("database migration error: %w", err)
		}

		return store, nil
	}

	if cfg.FileStoragePath != "" {
		store, err := storage.NewFileStorage(cfg.FileStoragePath)
		if err != nil {
			log.Printf("Failed to init file storage: %v", err)
		}
		return store, nil
	}

	return storage.NewMemoryStorage(), nil
}
