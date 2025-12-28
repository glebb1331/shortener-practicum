package storage

import (
	"context"
	"errors"
)

type Storage interface {
	Save(ctx context.Context, id, originalURL string) (string, error)
	Get(ctx context.Context, id string) (string, error)
	GetByOriginalURL(ctx context.Context, originalURL string) (string, error)
	Close() error
	Ping(ctx context.Context) error
	BatchSave(ctx context.Context, records []struct {
		ID          string
		OriginalURL string
	}) error
}

var (
	ErrNotFound  = errors.New("url not found")
	ErrURLExists = errors.New("url already exists")
)
