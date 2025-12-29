package main

import (
	"database/sql"
	"log"

	"github.com/glebb1331/shortener-practicum/internal/config"
	"github.com/glebb1331/shortener-practicum/internal/storage"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func initStorage(cfg *config.Config) storage.Storage {
	if cfg.DatabaseDSN != "" {
		db, err := sql.Open("pgx", cfg.DatabaseDSN)
		if err != nil {
			log.Printf("Failed to open database: %v", err)
		} else {
			store, err := storage.NewDatabaseStorage(db)
			if err != nil {
				log.Printf("Failed to init database storage: %v", err)
				db.Close()
			} else {
				log.Println("Using PostgreSQL storage")
				return store
			}
		}
	}

	if cfg.FileStoragePath != "" {
		store, err := storage.NewFileStorage(cfg.FileStoragePath)
		if err != nil {
			log.Printf("Failed to init file storage: %v", err)
		} else {
			log.Printf("Using file storage at: %s", cfg.FileStoragePath)
			return store
		}
	}

	log.Println("Using in-memory storage")
	return storage.NewMemoryStorage()
}
