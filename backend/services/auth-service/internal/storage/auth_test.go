package storage_test

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/BohdanKuzmenko1/URLShortener/services/auth-service/internal/storage"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
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

func TestAuthStorage_SaveRefreshToken(t *testing.T) {
	t.Cleanup(func() { cleanRedis(t) })

	userID := 1
	refreshToken := "refresh-token"

	authStorage := storage.NewAuthStorage(testRedis)

	err := authStorage.SaveRefreshToken(context.Background(), userID, refreshToken)
	require.NoError(t, err)

	val, err := testRedis.Get(context.Background(), fmt.Sprintf("refresh:%s", refreshToken)).Result()
	require.NoError(t, err)
	require.Equal(t, strconv.Itoa(userID), val)
}

func TestAuthStorage_GetUserIdByRefreshToken(t *testing.T) {
	t.Cleanup(func() { cleanRedis(t) })

	userID := 42
	refreshToken := "refresh-token"

	authStorage := storage.NewAuthStorage(testRedis)

	err := authStorage.SaveRefreshToken(context.Background(), userID, refreshToken)
	require.NoError(t, err)

	res, err := authStorage.GetUserIdByRefreshToken(context.Background(), refreshToken)
	require.NoError(t, err)
	require.Equal(t, userID, res)
}

func TestAuthStorage_GetUserIdByRefreshToken_NotFound(t *testing.T) {
	t.Cleanup(func() { cleanRedis(t) })

	refreshToken := "refresh-token"

	authStorage := storage.NewAuthStorage(testRedis)

	_, err := authStorage.GetUserIdByRefreshToken(context.Background(), refreshToken)
	require.Error(t, err)
	require.EqualError(t, err, "refresh token not found or expired")
}

func TestAuthStorage_DeleteRefreshToken(t *testing.T) {
	t.Cleanup(func() { cleanRedis(t) })

	userID := 1
	refreshToken := "refresh-token"

	authStorage := storage.NewAuthStorage(testRedis)

	err := authStorage.SaveRefreshToken(context.Background(), userID, refreshToken)
	require.NoError(t, err)

	err = authStorage.DeleteRefreshToken(context.Background(), refreshToken)
	require.NoError(t, err)

	_, err = authStorage.GetUserIdByRefreshToken(context.Background(), refreshToken)
	require.Error(t, err)
	require.EqualError(t, err, "refresh token not found or expired")
}

func TestAuthStorage_SaveRefreshToken_TTL(t *testing.T) {
	t.Cleanup(func() { cleanRedis(t) })

	userID := 1
	refreshToken := "refresh-token"

	authStorage := storage.NewAuthStorage(testRedis)

	err := authStorage.SaveRefreshToken(context.Background(), userID, refreshToken)
	require.NoError(t, err)

	ttl, err := testRedis.TTL(context.Background(), fmt.Sprintf("refresh:%s", refreshToken)).Result()
	require.NoError(t, err)
	require.InDelta(t, (7 * 24 * time.Hour).Seconds(), ttl.Seconds(), 5)
}
