package storage_test

import (
	"context"
	"errors"
	"fmt"
	"github.com/BohdanKuzmenko1/URLShortener/services/url-service/internal/storage"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"os"
	"testing"
	"time"
)

var testRedis *redis.Client

// ---------------------------------------------------------------------------
// TestMain — single Redis container for all tests
// ---------------------------------------------------------------------------

func TestMain(m *testing.M) {
	ctx := context.Background()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "redis:8.4.0-alpine",
			ExposedPorts: []string{"6379/tcp"},
			WaitingFor:   wait.ForListeningPort("6379/tcp").WithStartupTimeout(30 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		logrus.Fatalf("failed to start redis container: %v", err)
	}
	defer func() {
		if err := container.Terminate(ctx); err != nil {
			logrus.Errorf("failed to terminate redis container: %v", err)
		}
	}()

	host, _ := container.Host(ctx)
	port, _ := container.MappedPort(ctx, "6379")

	testRedis = redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", host, port.Port()),
	})
	defer testRedis.Close()

	if err := testRedis.Ping(ctx).Err(); err != nil {
		logrus.Fatalf("failed to ping redis: %v", err)
	}

	os.Exit(m.Run())
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func cleanRedis(t *testing.T) {
	t.Helper()

	err := testRedis.FlushDB(context.Background()).Err()
	require.NoError(t, err)
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestURLShortenerStorage_Set(t *testing.T) {
	// Arrange
	t.Cleanup(func() { cleanRedis(t) })

	urlID := 1
	targetURL := "https://google.com"
	slug := "test-slug"

	s := storage.NewRedisStorage(testRedis)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Act
	err := s.Set(ctx, urlID, targetURL, slug)

	// Assert
	require.NoError(t, err)

	cached, err := s.Get(ctx, slug)
	require.NoError(t, err)

	require.Equal(t, urlID, cached.ID)
	require.Equal(t, targetURL, cached.Target)

	cachedTTL, err := testRedis.TTL(ctx, "u:"+slug).Result()
	require.NoError(t, err)

	require.Greater(t, cachedTTL, 0*time.Second)
	require.LessOrEqual(t, cachedTTL, storage.RedisTTL)
}

func TestURLShortenerStorage_Get(t *testing.T) {
	// Arrange
	t.Cleanup(func() { cleanRedis(t) })

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := testRedis.HSet(ctx, "u:test-slug", map[string]interface{}{
		"id":     "1",
		"target": "https://google.com",
	}).Err()
	require.NoError(t, err)

	s := storage.NewRedisStorage(testRedis)

	// Act
	cached, err := s.Get(ctx, "test-slug")

	// Assert
	require.NoError(t, err)
	require.Equal(t, 1, cached.ID)
	require.Equal(t, "https://google.com", cached.Target)
}

func TestURLShortenerStorage_Get_NotFound(t *testing.T) {
	// Arrange
	t.Cleanup(func() { cleanRedis(t) })

	slug := "test-slug"

	s := storage.NewRedisStorage(testRedis)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Act
	cached, err := s.Get(ctx, slug)

	// Assert
	require.Error(t, err)
	require.True(t, errors.Is(err, storage.ErrNotFound))
	require.Empty(t, cached.ID)
	require.Empty(t, cached.Target)
}
