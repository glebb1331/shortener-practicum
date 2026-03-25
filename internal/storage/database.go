package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/glebb1331/shortener-practicum/migrations"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

type DatabaseStorage struct {
	db *sql.DB
}

func NewDatabaseStorage(db *sql.DB) (*DatabaseStorage, error) {
	s := &DatabaseStorage{db: db}
	if err := migrations.RunMigrations(db); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *DatabaseStorage) Save(ctx context.Context, id, originalURL, userID string) (string, error) {
	query := `
		INSERT INTO urls (id, original_url, user_id) 
		VALUES ($1, $2, $3)
		ON CONFLICT (original_url) DO UPDATE SET original_url = EXCLUDED.original_url
		RETURNING id`

	var actualID string
	err := s.db.QueryRowContext(ctx, query, id, originalURL, userID).Scan(&actualID)
	if err != nil {
		return "", fmt.Errorf("failed to save URL: %w", err)
	}

	if actualID != id {
		return actualID, ErrURLExists
	}

	return actualID, nil
}

func (s *DatabaseStorage) Get(ctx context.Context, id string) (string, error) {
	query := `SELECT original_url, is_deleted FROM urls WHERE id = $1`
	var originalURL string
	var isDeleted bool
	err := s.db.QueryRowContext(ctx, query, id).Scan(&originalURL, &isDeleted)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("failed to get URL: %w", err)
	}
	if isDeleted {
		return "", ErrURLDeleted
	}
	return originalURL, nil
}

func (s *DatabaseStorage) Close() error {
	return s.db.Close()
}

func (s *DatabaseStorage) Ping(ctx context.Context) error {
	if err := s.db.PingContext(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}
	return nil
}

func (s *DatabaseStorage) BatchSave(ctx context.Context, records []Record) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO urls (id, original_url, user_id) 
		VALUES ($1, $2, $3) 
		ON CONFLICT (original_url) DO NOTHING`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, record := range records {
		if _, err := stmt.ExecContext(ctx, record.ID, record.OriginalURL, record.UserID); err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
				continue
			}
			return fmt.Errorf("failed to execute statement: %w", err)
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
		return "", fmt.Errorf("failed to get ID by original URL: %w", err)
	}
	return id, nil
}

func (s *DatabaseStorage) GetByUserID(ctx context.Context, userID string) ([]Record, error) {
	query := `SELECT id, original_url, user_id, is_deleted FROM urls 
              WHERE user_id = $1 AND is_deleted = FALSE 
              ORDER BY created_at DESC`
	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user URLs: %w", err)
	}
	defer rows.Close()

	var records []Record
	for rows.Next() {
		var rec Record
		err = rows.Scan(&rec.ID, &rec.OriginalURL, &rec.UserID, &rec.IsDeleted)
		if err != nil {
			return nil, fmt.Errorf("failed to scan record: %w", err)
		}
		records = append(records, rec)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return records, nil
}

func (s *DatabaseStorage) DeleteURLs(ctx context.Context, userID string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	query := `UPDATE urls SET is_deleted = TRUE 
              WHERE user_id = $1 AND id = ANY($2)`

	_, err := s.db.ExecContext(ctx, query, userID, ids)
	if err != nil {
		return fmt.Errorf("failed to delete URLs: %w", err)
	}
	return nil
}

func (s *DatabaseStorage) GetBatchByUserID(ctx context.Context, userID string, ids []string) ([]Record, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	query := `SELECT id, original_url, user_id, is_deleted 
              FROM urls 
              WHERE user_id = $1 AND id = ANY($2)`

	rows, err := s.db.QueryContext(ctx, query, userID, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to query batch by user ID: %w", err)
	}
	defer rows.Close()

	var records []Record
	for rows.Next() {
		var rec Record
		err = rows.Scan(&rec.ID, &rec.OriginalURL, &rec.UserID, &rec.IsDeleted)
		if err != nil {
			return nil, fmt.Errorf("failed to scan batch record: %w", err)
		}
		records = append(records, rec)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating batch rows: %w", err)
	}

	return records, nil
}
