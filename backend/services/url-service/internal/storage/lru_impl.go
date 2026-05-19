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
	if cached, ok := l.lru.Get(slug); ok && time.Now().Before(cached.ExpiresAt) {
		return cached, nil
	}

	return CachedURL{}, ErrNotFound
}

func (l lruStorage) Set(ctx context.Context, urlID int, target, slug string) {
	cached := CachedURL{
		ID:        urlID,
		Target:    target,
		ExpiresAt: time.Now().Add(lruTTL),
	}

	l.lru.Add(slug, cached)
}

func NewLRUStorage() URLShortenerStorage {
	lruCache, err := lru.New[string, CachedURL](lruSize)
	if err != nil {
		panic(err)
	}

	return &lruStorage{lru: lruCache}
}
