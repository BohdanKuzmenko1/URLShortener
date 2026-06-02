package storage

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"strconv"
)

type redisStorage struct {
	redis *redis.Client
}

func (r redisStorage) Get(ctx context.Context, slug string) (CachedURL, error) {
	key := "u:" + slug

	res, err := r.redis.HMGet(ctx, key, "id", "target").Result()
	if err != nil {
		logrus.Warn(err)
		return CachedURL{}, err
	}

	var (
		urlID     int
		targetURL string
	)

	if len(res) == 2 && res[0] != nil && res[1] != nil {
		idStr, ok := res[0].(string)
		if !ok {
			return CachedURL{}, fmt.Errorf("invalid id type in redis")
		}

		urlID, err = strconv.Atoi(idStr)
		if err != nil {
			return CachedURL{}, err
		}

		targetStr, ok := res[1].(string)
		if !ok {
			return CachedURL{}, fmt.Errorf("invalid target type in redis")
		}

		targetURL = targetStr

		return CachedURL{ID: urlID, Target: targetURL}, nil
	}

	return CachedURL{}, ErrNotFound
}

func (r redisStorage) Set(ctx context.Context, urlID int, target, slug string) error {
	key := "u:" + slug

	pipe := r.redis.TxPipeline()
	pipe.HSet(ctx, key, map[string]interface{}{
		"id":     urlID,
		"target": target,
	})
	pipe.Expire(ctx, key, RedisTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		logrus.Error("redis cache write failed: ", err)
		return err
	}

	return nil
}

func NewRedisStorage(redis *redis.Client) URLShortenerStorage {
	return &redisStorage{redis: redis}
}
