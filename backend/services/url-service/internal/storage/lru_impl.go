package storage

import (
	"context"
	lru "github.com/hashicorp/golang-lru/v2"
	"time"
)

type lruStorage struct {
	lru *lru.Cache[string, CachedURL]
}

func (l lruStorage) Get(ctx context.Context, slug string) (CachedURL, error) {
	if err := ctx.Err(); err != nil {
		return CachedURL{}, err
	}

	if cached, ok := l.lru.Get(slug); ok && time.Now().Before(cached.ExpiresAt) {
		return cached, nil
	}

	return CachedURL{}, ErrNotFound
}

func (l lruStorage) Set(ctx context.Context, urlID int, target, slug string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	cached := CachedURL{
		ID:        urlID,
		Target:    target,
		ExpiresAt: time.Now().Add(lruTTL),
	}

	l.lru.Add(slug, cached)

	return nil
}

func NewLRUStorage() URLShortenerStorage {
	lruCache, err := lru.New[string, CachedURL](lruSize)
	if err != nil {
		panic(err)
	}

	return &lruStorage{lru: lruCache}
}
