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

func (s *DatabaseStorage) Save(ctx context.Context, id, originalURL, userID string) (string, error) {
	query := `
		INSERT INTO urls (id, original_url, user_id) 
		VALUES ($1, $2, $3) 
		ON CONFLICT (original_url) 
		DO UPDATE SET original_url = EXCLUDED.original_url 
		RETURNING id
	`

	var existingID string
	err := s.db.QueryRowContext(ctx, query, id, originalURL, userID).Scan(&existingID)
	if err != nil {
		return "", err
	}

	if existingID != id {
		return existingID, ErrURLExists
	}

	return id, nil
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
		return "", err
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
	return s.db.PingContext(ctx)
}

func (s *DatabaseStorage) BatchSave(ctx context.Context, records []Record) error {
	if len(records) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		CREATE TEMP TABLE temp_urls (
			id TEXT PRIMARY KEY,
			original_url TEXT UNIQUE,
			user_id TEXT
		) ON COMMIT DROP
	`)
	if err != nil {
		return err
	}

	stmt, err := tx.PrepareContext(ctx, `
		COPY temp_urls (id, original_url, user_id) FROM STDIN
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, rec := range records {
		_, err = stmt.ExecContext(ctx, rec.ID, rec.OriginalURL, rec.UserID)
		if err != nil {
			return err
		}
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO urls (id, original_url, user_id)
		SELECT t.id, t.original_url, t.user_id
		FROM temp_urls t
		WHERE NOT EXISTS (
			SELECT 1 FROM urls u WHERE u.original_url = t.original_url
		)
		ON CONFLICT (original_url) DO NOTHING
	`)
	if err != nil {
		return err
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

func (s *DatabaseStorage) GetByUserID(ctx context.Context, userID string) ([]Record, error) {
	query := `SELECT id, original_url, user_id, is_deleted FROM urls 
              WHERE user_id = $1 AND is_deleted = FALSE 
              ORDER BY created_at DESC`
	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []Record
	for rows.Next() {
		var rec Record
		if err := rows.Scan(&rec.ID, &rec.OriginalURL, &rec.UserID, &rec.IsDeleted); err != nil {
			return nil, err
		}
		records = append(records, rec)
	}

	if err = rows.Err(); err != nil {
		return nil, err
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
	return err
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
		return nil, err
	}
	defer rows.Close()

	var records []Record
	for rows.Next() {
		var rec Record
		if err := rows.Scan(&rec.ID, &rec.OriginalURL, &rec.UserID, &rec.IsDeleted); err != nil {
			return nil, err
		}
		records = append(records, rec)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return records, nil
}
