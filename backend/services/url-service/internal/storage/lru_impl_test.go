package storage_test

import (
	"context"
	"errors"
	"github.com/BohdanKuzmenko1/URLShortener/services/url-service/internal/storage"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestLRUStorage_Set_Get(t *testing.T) {
	// Arrange
	s := storage.NewLRUStorage()
	ctx := context.Background()

	// Act
	err := s.Set(ctx, 1, "https://google.com", "test-slug")

	// Assert
	require.NoError(t, err)

	cached, err := s.Get(ctx, "test-slug")
	require.NoError(t, err)
	require.Equal(t, 1, cached.ID)
	require.Equal(t, "https://google.com", cached.Target)
}

func TestLRUStorage_Get_NotFound(t *testing.T) {
	// Arrange
	s := storage.NewLRUStorage()
	ctx := context.Background()

	// Act
	cached, err := s.Get(ctx, "nonexistent")

	// Assert
	require.Error(t, err)
	require.True(t, errors.Is(err, storage.ErrNotFound))
	require.Empty(t, cached.ID)
	require.Empty(t, cached.Target)
}

func TestLRUStorage_Set_CancelledContext(t *testing.T) {
	// Arrange
	s := storage.NewLRUStorage()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Act
	err := s.Set(ctx, 1, "https://google.com", "slug")

	// Assert
	require.Error(t, err)
	require.ErrorIs(t, err, context.Canceled)
}

func TestLRUStorage_Get_CancelledContext(t *testing.T) {
	// Arrange
	s := storage.NewLRUStorage()
	ctx := context.Background()

	require.NoError(t, s.Set(ctx, 1, "https://google.com", "slug"))

	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	// Act
	_, err := s.Get(cancelledCtx, "slug")

	// Assert
	require.Error(t, err)
	require.ErrorIs(t, err, context.Canceled)
}
