package storage

import (
	"context"
	"errors"
	"time"
)

const (
	lruSize  = 1_000
	lruTTL   = 10 * time.Minute
	redisTTL = 30 * time.Minute
)

var ErrNotFound = errors.New("not found")

type CachedURL struct {
	ID        int
	Target    string
	ExpiresAt time.Time
}

type URLShortenerStorage interface {
	Get(ctx context.Context, slug string) (CachedURL, error)
	Set(ctx context.Context, urlID int, target, slug string)
}
