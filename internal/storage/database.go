package storage

import (
	"context"
	"database/sql"
	"errors"
	"strings"

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

func (s *DatabaseStorage) Save(ctx context.Context, id, originalURL string) (string, error) {
	existingID, err := s.GetByOriginalURL(ctx, originalURL)
	if err == nil {
		return existingID, ErrURLExists
	}

	if err != nil && !errors.Is(err, ErrNotFound) {
		return "", err
	}

	query := `INSERT INTO urls (id, original_url) VALUES ($1, $2)`
	_, err = s.db.ExecContext(ctx, query, id, originalURL)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") ||
			strings.Contains(err.Error(), "unique constraint") {
			existingID, err2 := s.GetByOriginalURL(ctx, originalURL)
			if err2 != nil {
				return "", err2
			}
			return existingID, ErrURLExists
		}
		return "", err
	}

	return id, nil
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

func (s *DatabaseStorage) BatchSave(ctx context.Context, records []struct {
	ID          string
	OriginalURL string
}) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, "INSERT INTO urls (id, original_url) VALUES ($1, $2)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, record := range records {
		_, err := s.Save(ctx, record.ID, record.OriginalURL)
		if err != nil && !errors.Is(err, ErrURLExists) {
			return err
		}
	}

	return tx.Commit()
}

func (s *DatabaseStorage) GetByOriginalURL(ctx context.Context, originalURL string) (string, error) {
	query := `SELECT id FROM urls WHERE original_url = $1`
	var id string
	err := s.db.QueryRowContext(ctx, query, originalURL).Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", err
	}
	return id, nil
}
