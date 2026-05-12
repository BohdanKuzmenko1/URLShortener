package storage

import (
	"context"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"strconv"
	"time"
)

const refreshTokenTTL = 7 * 24 * time.Hour

type AuthStorage interface {
	SaveRefreshToken(ctx context.Context, userID int, refreshToken string) error
	GetUserIdByRefreshToken(ctx context.Context, refreshToken string) (int, error)
	DeleteRefreshToken(ctx context.Context, refreshToken string) error
}

type authStorage struct {
	redis *redis.Client
}

// SaveRefreshToken stores the refresh token in Redis associated with the given user ID.
// The token expires after refreshTokenTTL.
func (a authStorage) SaveRefreshToken(ctx context.Context, userID int, refreshToken string) error {
	key := fmt.Sprintf("refresh:%s", refreshToken)

	return a.redis.Set(ctx, key, userID, refreshTokenTTL).Err()
}

// GetUserIdByRefreshToken retrieves the user ID associated with the given refresh token from Redis.
// Returns an error if the token is not found or the request failed.
func (a authStorage) GetUserIdByRefreshToken(ctx context.Context, refreshToken string) (int, error) {
	key := fmt.Sprintf("refresh:%s", refreshToken)

	userIdStr, err := a.redis.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return 0, errors.New("refresh token not found or expired")
	}
	if err != nil {
		return 0, err
	}

	userId, err := strconv.Atoi(userIdStr)
	if err != nil {
		return 0, fmt.Errorf("invalid user id stored in redis: %w", err)
	}

	return userId, nil
}

// DeleteRefreshToken removes the refresh token from Redis, invalidating the session.
func (a authStorage) DeleteRefreshToken(ctx context.Context, refreshToken string) error {
	key := fmt.Sprintf("refresh:%s", refreshToken)
	return a.redis.Del(ctx, key).Err()
}

func NewAuthStorage(redis *redis.Client) AuthStorage {
	return authStorage{
		redis: redis,
	}
}
