package service_test

import (
	"context"
	"github.com/BohdanKuzmenko1/URLShortener/services/url-service/internal"
	"github.com/BohdanKuzmenko1/URLShortener/services/url-service/internal/broker"
	"github.com/BohdanKuzmenko1/URLShortener/services/url-service/internal/storage"
	"github.com/stretchr/testify/mock"
)

type LRUStorageMock struct {
	mock.Mock
}

func (L *LRUStorageMock) Get(ctx context.Context, slug string) (storage.CachedURL, error) {
	args := L.Called(ctx, slug)
	if args.Get(1) != nil {
		return storage.CachedURL{}, args.Error(1)
	}
	return args.Get(0).(storage.CachedURL), args.Error(1)
}

func (L *LRUStorageMock) Set(ctx context.Context, urlID int, target, slug string) {
	L.Called(ctx, urlID, target, slug)
}

type RedisStorageMock struct {
	mock.Mock
}

func (r *RedisStorageMock) Get(ctx context.Context, slug string) (storage.CachedURL, error) {
	args := r.Called(ctx, slug)
	if args.Get(1) != nil {
		return storage.CachedURL{}, args.Error(1)
	}
	return args.Get(0).(storage.CachedURL), args.Error(1)
}

func (r *RedisStorageMock) Set(ctx context.Context, urlID int, target, slug string) {
	r.Called(ctx, urlID, target, slug)
}

type URLShortenerRepositoryMock struct {
	mock.Mock
}

func (U *URLShortenerRepositoryMock) AddShortURL(ctx context.Context, userId int, targetURL, slug string) error {
	args := U.Called(ctx, userId, targetURL, slug)
	return args.Error(0)
}

func (U *URLShortenerRepositoryMock) GetURLBySlug(ctx context.Context, slug string) (int, string, error) {
	args := U.Called(ctx, slug)
	if args.Get(2) != nil {
		return 0, "", args.Error(2)
	}
	return args.Get(0).(int), args.Get(1).(string), args.Error(2)
}

func (U *URLShortenerRepositoryMock) GetURLByUserId(ctx context.Context, userId, urlId int) (internal.ShortURL, error) {
	args := U.Called(ctx, userId, urlId)
	return args.Get(0).(internal.ShortURL), args.Error(1)
}

type RedirectProducerMock struct {
	mock.Mock
}

func (r *RedirectProducerMock) SendRedirect(ctx context.Context, event broker.RedirectEvent, slug string) error {
	args := r.Called(ctx, event, slug)
	return args.Error(0)
}
