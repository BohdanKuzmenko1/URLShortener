package storage

import (
	"context"
	"github.com/redis/go-redis/v9"
)

type URLShortenerStorage interface {
	GetCachedURL(ctx context.Context, urlID int) (string, error)
}

type urlShortenerStorage struct {
	redis *redis.Client
}

func newUrlShortenerStorage(redis *redis.Client) URLShortenerStorage {
	return &urlShortenerStorage{redis: redis}
}
