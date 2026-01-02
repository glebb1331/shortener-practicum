package storage

import (
	"context"
	"errors"
)

type Record struct {
	ID          string
	OriginalURL string
	UserID      string
	IsDeleted   bool
}

type Storage interface {
	Save(ctx context.Context, id, originalURL, userID string) (string, error)
	Get(ctx context.Context, id string) (string, error)
	GetByOriginalURL(ctx context.Context, originalURL string) (string, error)
	GetByUserID(ctx context.Context, userID string) ([]Record, error)
	Close() error
	Ping(ctx context.Context) error
	BatchSave(ctx context.Context, records []Record) error
	DeleteURLs(ctx context.Context, userID string, ids []string) error
	GetBatchByUserID(ctx context.Context, userID string, ids []string) ([]Record, error)
}

var (
	ErrNotFound   = errors.New("url not found")
	ErrURLExists  = errors.New("url already exists")
	ErrURLDeleted = errors.New("url deleted")
	ErrNotOwner   = errors.New("user is not owner of the url")
)
