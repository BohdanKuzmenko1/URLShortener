package repository_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/BohdanKuzmenko1/URLShortener/services/url-service/internal/repository"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"golang.org/x/crypto/bcrypt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

var testDB *sqlx.DB

// ---------------------------------------------------------------------------
// TestMain — single Postgres container for all tests
// ---------------------------------------------------------------------------

func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "postgres:18.1-alpine3.22",
			ExposedPorts: []string{"5432/tcp"},
			Env: map[string]string{
				"POSTGRES_USER":     "test",
				"POSTGRES_PASSWORD": "test",
				"POSTGRES_DB":       "testdb",
			},
			WaitingFor: wait.ForListeningPort("5432/tcp").WithStartupTimeout(30 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		logrus.Fatalf("failed to start container: %v", err)
	}
	defer func() {
		if err := container.Terminate(ctx); err != nil {
			logrus.Errorf("failed to terminate container: %v", err)
		}
	}()

	host, _ := container.Host(ctx)
	port, _ := container.MappedPort(ctx, "5432")
	dsn := fmt.Sprintf("postgres://test:test@%s:%s/testdb?sslmode=disable", host, port.Port())

	if err := runMigrations(dsn); err != nil {
		logrus.Fatalf("migrations failed: %v", err)
	}

	testDB, err = sqlx.Connect("postgres", dsn)
	if err != nil {
		logrus.Fatalf("failed to connect to postgres: %v", err)
	}
	defer testDB.Close()

	os.Exit(m.Run())
}

// ---------------------------------------------------------------------------
// Migrations
// ---------------------------------------------------------------------------

func runMigrations(dsn string) error {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("migration driver: %w", err)
	}

	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getwd: %w", err)
	}

	migrationsPath := filepath.Join(wd, "..", "..", "..", "..", "deploy", "migrations")

	fsys := os.DirFS(migrationsPath)
	sourceDriver, err := iofs.New(fsys, ".")
	if err != nil {
		return fmt.Errorf("iofs source: %w", err)
	}

	mg, err := migrate.NewWithInstance("iofs", sourceDriver, "postgres", driver)
	if err != nil {
		return fmt.Errorf("migrate instance: %w", err)
	}

	if err := mg.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migrate up: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func seedUser(t *testing.T, email, password string) int {
	t.Helper()

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	require.NoError(t, err, "bcrypt failed")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var id int
	err = testDB.QueryRowContext(
		ctx,
		"INSERT INTO users (email, password) VALUES ($1, $2) RETURNING id",
		email, string(hash),
	).Scan(&id)
	require.NoError(t, err, "seedUser: insert failed")

	return id
}

func seedURL(t *testing.T, userID int, targetURL, slug string) int {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var id int
	err := testDB.QueryRowContext(
		ctx,
		"INSERT INTO urls (user_id, target_url, slug) VALUES ($1, $2, $3) RETURNING id",
		userID,
		targetURL,
		slug,
	).Scan(&id)

	require.NoError(t, err, "seedURL: insert failed")

	return id
}

func cleanDB(t *testing.T) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := testDB.ExecContext(ctx, "TRUNCATE TABLE users, urls RESTART IDENTITY CASCADE")
	require.NoError(t, err)
}

type URL struct {
	ID        int       `db:"id"`
	UserID    int       `db:"user_id"`
	Slug      string    `db:"slug"`
	TargetURL string    `db:"target_url"`
	CreatedAt time.Time `db:"created_at"`
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestUrlShortenerRepository_AddShortURL(t *testing.T) {
	// Arrange
	t.Cleanup(func() { cleanDB(t) })

	targetURL := "http://example.com"
	slug := "test"

	userID := seedUser(t, "testuser", "testpass")

	urlRepo := repository.NewURLShortenerRepository(testDB)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Action
	err := urlRepo.AddShortURL(ctx, userID, targetURL, slug)

	// Assert
	require.NoError(t, err)

	var url URL

	err = testDB.GetContext(ctx, &url, "SELECT * FROM urls WHERE slug = $1", slug)
	require.NoError(t, err)
	require.Equal(t, userID, url.UserID)
	require.Equal(t, targetURL, url.TargetURL)
	require.Equal(t, slug, url.Slug)
}

func TestUrlShortenerRepository_AddShortURL_Duplicate(t *testing.T) {
	// Arrange
	t.Cleanup(func() { cleanDB(t) })

	targetURL := "http://example.com"
	slug := "test"

	userID := seedUser(t, "testuser", "testpass")

	urlRepo := repository.NewURLShortenerRepository(testDB)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Action
	errFirstCall := urlRepo.AddShortURL(ctx, userID, targetURL, slug)
	errSecondCall := urlRepo.AddShortURL(ctx, userID, targetURL, slug)

	// Assert
	require.NoError(t, errFirstCall)
	require.Error(t, errSecondCall)
}

func TestUrlShortenerRepository_AddShortURL_ContextCanceled(t *testing.T) {
	// Arrange
	t.Cleanup(func() { cleanDB(t) })

	urlRepo := repository.NewURLShortenerRepository(testDB)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Action
	err := urlRepo.AddShortURL(ctx, 999, "", "")

	// Assert
	require.Error(t, err)
	require.True(t, errors.Is(err, context.Canceled))
}

func TestUrlShortenerRepository_GetURLByUserId(t *testing.T) {
	// Arrange
	t.Cleanup(func() { cleanDB(t) })

	targetURL := "http://example.com"
	slug := "test"

	userID := seedUser(t, "testuser", "testpass")
	urlID := seedURL(t, userID, targetURL, slug)

	urlRepo := repository.NewURLShortenerRepository(testDB)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Action
	url, err := urlRepo.GetURLByUserId(ctx, userID, urlID)

	// Assert
	require.NoError(t, err)
	require.Equal(t, urlID, int(url.UrlId))
	require.Equal(t, targetURL, url.TargetUrl)
	require.Equal(t, slug, url.Slug)
	require.Equal(t, userID, int(url.UserId))
}

func TestUrlShortenerRepository_GetURLByUserId_NotFound(t *testing.T) {
	// Arrange
	t.Cleanup(func() { cleanDB(t) })

	urlRepo := repository.NewURLShortenerRepository(testDB)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Action
	url, err := urlRepo.GetURLByUserId(ctx, 999, 999)

	// Assert
	require.Error(t, err)
	require.Empty(t, url)
	require.True(t, errors.Is(err, sql.ErrNoRows))
}

func TestUrlShortenerRepository_GetURLByUserId_OtherUser(t *testing.T) {
	// Arrange
	t.Cleanup(func() { cleanDB(t) })

	targetURL := "http://example.com"
	slug := "test"

	ownerID := seedUser(t, "owner@test.com", "pass")
	otherID := seedUser(t, "other@test.com", "pass")

	urlID := seedURL(t, ownerID, targetURL, slug)

	urlRepo := repository.NewURLShortenerRepository(testDB)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Action
	_, err := urlRepo.GetURLByUserId(ctx, otherID, urlID)

	// Assert
	require.Error(t, err)
	require.True(t, errors.Is(err, sql.ErrNoRows))
}

func TestUrlShortenerRepository_GetURLByUserId_ContextCanceled(t *testing.T) {
	// Arrange
	t.Cleanup(func() { cleanDB(t) })

	urlRepo := repository.NewURLShortenerRepository(testDB)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Action
	_, err := urlRepo.GetURLByUserId(ctx, 999, 999)

	// Assert
	require.Error(t, err)
	require.True(t, errors.Is(err, context.Canceled))
}

func TestUrlShortenerRepository_GetURLBySlug(t *testing.T) {
	// Arrange
	t.Cleanup(func() { cleanDB(t) })

	targetURL := "http://example.com"
	slug := "test"

	userID := seedUser(t, "testuser", "testpass")
	urlID := seedURL(t, userID, targetURL, slug)

	urlRepo := repository.NewURLShortenerRepository(testDB)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Action
	receivedID, receivedTargetURL, err := urlRepo.GetURLBySlug(ctx, slug)

	// Assert
	require.NoError(t, err)
	require.Equal(t, urlID, receivedID)
	require.Equal(t, targetURL, receivedTargetURL)
}

func TestUrlShortenerRepository_GetURLBySlug_NotFound(t *testing.T) {
	// Arrange
	t.Cleanup(func() { cleanDB(t) })

	urlRepo := repository.NewURLShortenerRepository(testDB)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Action
	receivedID, receivedTargetURL, err := urlRepo.GetURLBySlug(ctx, "non_existing_slug")

	// Assert
	require.Error(t, err)
	require.Empty(t, receivedID)
	require.Empty(t, receivedTargetURL)
	require.True(t, errors.Is(err, repository.ErrNotFound))
}

func TestUrlShortenerRepository_GetURLBySlug_ContextCanceled(t *testing.T) {
	// Arrange
	t.Cleanup(func() { cleanDB(t) })

	urlRepo := repository.NewURLShortenerRepository(testDB)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Act
	_, _, err := urlRepo.GetURLBySlug(ctx, "test")

	// Assert
	require.Error(t, err)
	require.True(t, errors.Is(err, context.Canceled))
}
