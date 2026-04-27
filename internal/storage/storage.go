package storage

import (
	"context"
	"errors"
)

// Record хранит данные об одной сокращённой ссылке.
type Record struct {
	ID          string
	OriginalURL string
	UserID      string
	IsDeleted   bool
}

// Storage описывает интерфейс хранилища сокращённых ссылок.
type Storage interface {
	// Save сохраняет пару id, originalURL для пользователя userID.
	// Если URL уже существует, возвращает существующий id и ErrURLExists.
	Save(ctx context.Context, id, originalURL, userID string) (string, error)

	// Get возвращает оригинальный URL по-короткому id.
	Get(ctx context.Context, id string) (string, error)

	// GetByOriginalURL возвращает короткий id по оригинальному URL.
	GetByOriginalURL(ctx context.Context, originalURL string) (string, error)

	// GetByUserID возвращает все записи указанного пользователя.
	GetByUserID(ctx context.Context, userID string) ([]Record, error)

	// Close освобождает ресурсы хранилища.
	Close() error

	// Ping проверяет доступность хранилища.
	Ping(ctx context.Context) error

	// BatchSave сохраняет несколько записей за одну операцию.
	BatchSave(ctx context.Context, records []Record) error

	// DeleteURLs помечает ссылки как удалённые (только для владельца).
	DeleteURLs(ctx context.Context, userID string, ids []string) error

	// GetBatchByUserID возвращает записи пользователя по списку id.
	GetBatchByUserID(ctx context.Context, userID string, ids []string) ([]Record, error)

	// Stats возвращает количество сокращённых URL и количество уникальных пользователей в хранилище.
	Stats(ctx context.Context) (urls int, users int, err error)
}

// Sentinel-ошибки хранилища.
var (
	ErrNotFound   = errors.New("url not found")
	ErrURLExists  = errors.New("url already exists")
	ErrURLDeleted = errors.New("url deleted")
	ErrNotOwner   = errors.New("user is not owner of the url")
)
