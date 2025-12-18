package storage

import (
	"context"
)

type Storage interface {
	Save(ctx context.Context, id, originalURL string) error
	Get(ctx context.Context, id string) (string, error)
	Close() error
	Ping(ctx context.Context) error
}
