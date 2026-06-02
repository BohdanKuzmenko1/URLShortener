package service

import (
	"context"
	"errors"
	"github.com/BohdanKuzmenko1/URLShortener/services/url-service/internal"
	"github.com/BohdanKuzmenko1/URLShortener/services/url-service/internal/broker"
	"github.com/BohdanKuzmenko1/URLShortener/services/url-service/internal/repository"
	"github.com/BohdanKuzmenko1/URLShortener/services/url-service/internal/storage"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"time"
)

type urlShortenerService struct {
	repoPostgres repository.URLShortenerRepository
	producer     broker.RedirectProducer
	lruStorage   storage.URLShortenerStorage
	redisStorage storage.URLShortenerStorage
}

type URLShortenerService interface {
	GetURL(ctx context.Context, userId int, urlId int) (internal.ShortURL, error)
	GenerateShortURL(ctx context.Context, userId int, targetURL, slug string) (string, error)
	ResolveSlug(ctx context.Context, redirect internal.Redirect) (string, error)
}

func generateShortSlug() string {
	return uuid.New().String()[:8]
}

func NewURLShortenerService(
	repoPostgres repository.URLShortenerRepository,
	producer broker.RedirectProducer,
	redisStorage storage.URLShortenerStorage,
	lruStorage storage.URLShortenerStorage,
) URLShortenerService {
	s := &urlShortenerService{
		repoPostgres: repoPostgres,
		producer:     producer,
		redisStorage: redisStorage,
		lruStorage:   lruStorage,
	}

	return s
}

func (u *urlShortenerService) GetURL(ctx context.Context, userId int, urlId int) (internal.ShortURL, error) {
	url, err := u.repoPostgres.GetURLByUserId(ctx, userId, urlId)
	if err != nil {
		return internal.ShortURL{}, err
	}
	return url, nil
}

func (u *urlShortenerService) ResolveSlug(ctx context.Context, redirect internal.Redirect) (string, error) {
	// Try to get target URL from LRU cache
	cached, err := u.lruStorage.Get(ctx, redirect.Slug)
	if err == nil {
		u.sendRedirectEvent(cached.ID, redirect)
		return cached.Target, nil
	}

	// Try to get target URL from Redis
	cached, err = u.redisStorage.Get(ctx, redirect.Slug)
	if err == nil {
		u.sendRedirectEvent(cached.ID, redirect)
		return cached.Target, nil
	}

	// If error != not found print warn log
	if !errors.Is(err, storage.ErrNotFound) {
		logrus.Warn("error getting cached url from Redis: ", err.Error())
	}

	var (
		urlID     int
		targetURL string
	)

	// Get target URL from database if LRU and Redis don't have such slug
	urlID, targetURL, err = u.repoPostgres.GetURLBySlug(ctx, redirect.Slug)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		logrus.Warn("error getting cached url: ", err.Error())
		return "", err
	}

	if errors.Is(err, repository.ErrNotFound) {
		return "", err
	}

	// Add slug to LRU cache
	u.lruStorage.Set(ctx, urlID, targetURL, redirect.Slug)

	// Add slug to Redis
	u.redisStorage.Set(ctx, urlID, targetURL, redirect.Slug)

	u.sendRedirectEvent(urlID, redirect)
	return targetURL, nil
}

func (u *urlShortenerService) sendRedirectEvent(urlID int, redirect internal.Redirect) {
	event := broker.RedirectEvent{
		URLId:     urlID,
		ClientIP:  redirect.ClientIP,
		Referer:   redirect.Referer,
		Country:   redirect.Country,
		Language:  redirect.Language,
		UserAgent: redirect.UserAgent,
		CreatedAt: time.Now().UTC().Unix(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()

	if err := u.producer.SendRedirect(ctx, event, redirect.Slug); err != nil {
		logrus.Error("error sending redirect event: ", err)
	}
}

func (u *urlShortenerService) GenerateShortURL(ctx context.Context, userId int, targetURL, slug string) (string, error) {
	if slug == "" {
		slug = generateShortSlug()
	}

	shortURL := viper.GetString("api-gateway.baseURL") + slug

	err := u.repoPostgres.AddShortURL(ctx, userId, targetURL, slug)
	if err != nil {
		logrus.Errorln(err.Error())
		return "", err
	}

	return shortURL, nil
}
