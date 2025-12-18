package storage

import (
	"context"
	"database/sql"
	"errors"

	"github.com/glebb1331/shortener-practicum/migrations"
)

type DatabaseStorage struct {
	db *sql.DB
}

func NewDatabaseStorage(db *sql.DB) (*DatabaseStorage, error) {
	s := &DatabaseStorage{db: db}

	if err := migrations.RunMigrations(context.Background(), db); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *DatabaseStorage) Save(ctx context.Context, id, originalURL string) error {
	query := `INSERT INTO urls (id, original_url) VALUES ($1, $2)`
	_, err := s.db.ExecContext(ctx, query, id, originalURL)
	return err
}

func (s *DatabaseStorage) Get(ctx context.Context, id string) (string, error) {
	query := `SELECT original_url FROM urls WHERE id = $1`

	var originalURL string
	err := s.db.QueryRowContext(ctx, query, id).Scan(&originalURL)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", err
	}

	return originalURL, nil
}

func (s *DatabaseStorage) Close() error {
	return s.db.Close()
}

func (s *DatabaseStorage) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}
