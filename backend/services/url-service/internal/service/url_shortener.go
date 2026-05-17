package service

import (
	"context"
	"fmt"
	"github.com/BohdanKuzmenko1/URLShortener/services/url-service/internal"
	"github.com/BohdanKuzmenko1/URLShortener/services/url-service/internal/broker"
	"github.com/BohdanKuzmenko1/URLShortener/services/url-service/internal/repository"
	"github.com/google/uuid"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"strconv"
	"time"
)

// TODO: Implement wrapper storage for redis to simplify tests and their clarity

const (
	redisTTL        = 30 * time.Minute
	lruTTL          = 10 * time.Minute
	lruSize         = 1_000
	eventWorkers    = 20
	eventBufferSize = 10_000
)

type cachedURL struct {
	id        int
	target    string
	expiresAt time.Time
}

type eventJob struct {
	urlID    int
	redirect internal.Redirect
}

type urlShortenerService struct {
	repoPostgres repository.URLShortenerRepository
	redis        *redis.Client
	producer     broker.RedirectProducer
	lruCache     *lru.Cache[string, cachedURL]
	eventCh      chan eventJob
}

type URLShortenerService interface {
	GetURL(ctx context.Context, userId int, urlId int) (internal.ShortURL, error)
	GenerateShortURL(ctx context.Context, userId int, targetURL, slug string) (string, error)
	ResolveSlug(ctx context.Context, redirect internal.Redirect) (string, error)
	Close()
}

func generateShortSlug() string {
	return uuid.New().String()[:8]
}

func NewURLShortenerService(repoPostgres repository.URLShortenerRepository, producer broker.RedirectProducer, redisClient *redis.Client) URLShortenerService {
	lruCache, _ := lru.New[string, cachedURL](lruSize)

	s := &urlShortenerService{
		repoPostgres: repoPostgres,
		producer:     producer,
		redis:        redisClient,
		lruCache:     lruCache,
		eventCh:      make(chan eventJob, eventBufferSize),
	}

	for range eventWorkers {
		go s.eventWorker()
	}

	return s
}

func (u *urlShortenerService) eventWorker() {
	for job := range u.eventCh {
		u.sendRedirectEvent(job.urlID, job.redirect)
	}
}

func (u *urlShortenerService) Close() {
	close(u.eventCh)
}

func (u *urlShortenerService) dispatchEvent(urlID int, redirect internal.Redirect) {
	select {
	case u.eventCh <- eventJob{urlID: urlID, redirect: redirect}:
	default:
		logrus.Warn("event channel full, dropping kafka event for slug: ", redirect.Slug)
	}
}

func (u *urlShortenerService) GetURL(ctx context.Context, userId int, urlId int) (internal.ShortURL, error) {
	url, err := u.repoPostgres.GetURLByUserId(ctx, userId, urlId)
	if err != nil {
		return internal.ShortURL{}, err
	}
	return url, nil
}

func (u *urlShortenerService) ResolveSlug(ctx context.Context, redirect internal.Redirect) (string, error) {
	key := "u:" + redirect.Slug

	// Try to get target URL from LRU cache
	if cached, ok := u.lruCache.Get(redirect.Slug); ok && time.Now().Before(cached.expiresAt) {
		u.dispatchEvent(cached.id, redirect)
		return cached.target, nil
	}

	// Try to get target URL from Redis
	res, err := u.redis.HMGet(ctx, key, "id", "target").Result()
	if err != nil {
		return "", err
	}

	var (
		urlID     int
		targetURL string
	)

	if len(res) == 2 && res[0] != nil && res[1] != nil {
		idStr, ok := res[0].(string)
		if !ok {
			return "", fmt.Errorf("invalid id type in redis")
		}

		urlID, err = strconv.Atoi(idStr)
		if err != nil {
			return "", err
		}

		targetStr, ok := res[1].(string)
		if !ok {
			return "", fmt.Errorf("invalid target type in redis")
		}

		targetURL = targetStr
	} else {
		// Get target URL from database if LRU and Redis don't have such slug
		urlID, targetURL, err = u.repoPostgres.GetURLBySlug(ctx, redirect.Slug)
		if err != nil {
			return "", err
		}

		// Add slug to Redis
		pipe := u.redis.TxPipeline()
		pipe.HSet(ctx, key, map[string]interface{}{
			"id":     urlID,
			"target": targetURL,
		})
		pipe.Expire(ctx, key, redisTTL)
		if _, err = pipe.Exec(ctx); err != nil {
			logrus.Error("redis cache write failed: ", err)
		}
	}

	// Add slug to LRU cache
	u.lruCache.Add(redirect.Slug, cachedURL{
		id:        urlID,
		target:    targetURL,
		expiresAt: time.Now().Add(lruTTL),
	})

	u.dispatchEvent(urlID, redirect)
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
	if err := u.producer.SendRedirect(event, redirect.Slug); err != nil {
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
