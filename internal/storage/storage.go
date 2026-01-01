package storage

import (
	"context"
	"errors"
)

type Record struct {
	ID          string
	OriginalURL string
	UserID      string
}

type Storage interface {
	Save(ctx context.Context, id, originalURL, userID string) (string, error)
	Get(ctx context.Context, id string) (string, error)
	GetByOriginalURL(ctx context.Context, originalURL string) (string, error)
	GetByUserID(ctx context.Context, userID string) ([]Record, error)
	Close() error
	Ping(ctx context.Context) error
	BatchSave(ctx context.Context, records []Record) error
}

var (
	ErrNotFound  = errors.New("url not found")
	ErrURLExists = errors.New("url already exists")
)
