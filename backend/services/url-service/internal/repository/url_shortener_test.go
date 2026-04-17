package repository

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

var testDB *sqlx.DB

// =======================
// TestMain
// =======================

func TestMain(m *testing.M) {
	ctx := context.Background()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "postgres:18.1-alpine3.22",
			ExposedPorts: []string{"5432/tcp"},
			Env: map[string]string{
				"POSTGRES_USER":     "postgres",
				"POSTGRES_PASSWORD": "postgres",
				"POSTGRES_DB":       "postgres",
			},
			WaitingFor: wait.ForListeningPort("5432/tcp").
				WithStartupTimeout(30 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		log.Fatal(err)
	}

	defer container.Terminate(ctx)

	host, err := container.Host(ctx)
	if err != nil {
		log.Fatal(err)
	}

	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		log.Fatal(err)
	}

	dsn := fmt.Sprintf(
		"postgres://postgres:postgres@%s:%s/postgres?sslmode=disable",
		host, port.Port(),
	)

	testDB, err = sqlx.Connect("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}

	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		email VARCHAR(50) UNIQUE NOT NULL,
		password VARCHAR(250) NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS urls (
		id SERIAL PRIMARY KEY,
		user_id INT NOT NULL,
		slug VARCHAR(100) UNIQUE NOT NULL,
		target_url VARCHAR(2000) NOT NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

		CONSTRAINT fk_user
			FOREIGN KEY (user_id)
			REFERENCES users(id)
			ON DELETE CASCADE
	);
	`

	testDB.MustExec(schema)

	code := m.Run()
	os.Exit(code)
}

// =======================
// Helpers
// =======================

func cleanDB(t *testing.T) {
	t.Helper()

	_, err := testDB.Exec("TRUNCATE TABLE users RESTART IDENTITY CASCADE")
	require.NoError(t, err)
}

func seedUser(t *testing.T) int {
	t.Helper()

	var userID int

	err := testDB.QueryRow(
		`INSERT INTO users (email, password, created_at)
		 VALUES ($1, $2, $3)
		 RETURNING id`,
		"test@example.com",
		"password",
		time.Now(),
	).Scan(&userID)

	require.NoError(t, err)
	return userID
}

func seedURL(t *testing.T, userID int) string {
	t.Helper()

	var slug string

	err := testDB.QueryRow(
		`INSERT INTO urls (user_id, target_url, slug)
		VALUES ($1, $2, $3)
		RETURNING slug`,
		userID,
		"http://example.com",
		"test",
	).Scan(&slug)

	require.NoError(t, err)
	return slug
}

type URL struct {
	ID        int       `db:"id"`
	UserID    int       `db:"user_id"`
	Slug      string    `db:"slug"`
	TargetURL string    `db:"target_url"`
	CreatedAt time.Time `db:"created_at"`
}

// =======================
// Tests
// =======================

func TestAddShortURL_Success(t *testing.T) {
	// Arrange
	cleanDB(t)
	userID := seedUser(t)

	repo := NewURLShortenerRepository(testDB)

	// Act
	err := repo.AddShortURL(userID, "http://google.com", "google")
	require.NoError(t, err)

	// Assert
	var url URL
	err = testDB.Get(&url, "SELECT * FROM urls WHERE slug = $1", "google")
	require.NoError(t, err)

	assert.Equal(t, userID, url.UserID)
	assert.Equal(t, "google", url.Slug)
	assert.Equal(t, "http://google.com", url.TargetURL)
	assert.NotZero(t, url.ID)
	assert.False(t, url.CreatedAt.IsZero())
}

func TestAddShortURL_FailDuplicate(t *testing.T) {
	// Arrange
	cleanDB(t)
	userID := seedUser(t)

	repo := NewURLShortenerRepository(testDB)

	// Act
	err := repo.AddShortURL(userID, "http://google.com", "google")
	require.NoError(t, err)

	err = repo.AddShortURL(userID, "http://google.com", "google")
	require.Error(t, err)

	// Assert
	var count int
	err = testDB.QueryRow(
		"SELECT COUNT(*) FROM urls WHERE slug = $1",
		"google",
	).Scan(&count)

	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestAddShortURL_InvalidUserId(t *testing.T) {
	// Arrange
	cleanDB(t)
	repo := NewURLShortenerRepository(testDB)

	// Act
	err := repo.AddShortURL(9999, "http://google.com", "google")

	// Assert
	require.Error(t, err)
}

func TestAddShortURL_SlugMaxLength(t *testing.T) {
	// Arrange
	cleanDB(t)
	userID := seedUser(t)

	repo := NewURLShortenerRepository(testDB)

	slug := strings.Repeat("a", 100)

	// Act
	err := repo.AddShortURL(userID, "http://example.com", slug)
	require.NoError(t, err)

	// Assert
	var count int
	err = testDB.QueryRow(
		"SELECT COUNT(*) FROM urls WHERE slug = $1",
		slug,
	).Scan(&count)

	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestAddShortURL_SlugTooLong(t *testing.T) {
	// Arrange
	cleanDB(t)
	userID := seedUser(t)

	repo := NewURLShortenerRepository(testDB)

	slug := strings.Repeat("a", 101)

	// Act
	err := repo.AddShortURL(userID, "http://example.com", slug)

	// Assert
	require.Error(t, err)
}

func TestAddShortURL_TargetURLMaxLength(t *testing.T) {
	// Arrange
	cleanDB(t)
	userID := seedUser(t)

	repo := NewURLShortenerRepository(testDB)

	longURL := "http://example.com/" + strings.Repeat("a", 1980)

	// Act
	err := repo.AddShortURL(userID, longURL, "longurl")

	// Assert
	require.NoError(t, err)
}

func TestAddShortURL_TargetURLTooLong(t *testing.T) {
	// Arrange
	cleanDB(t)
	userID := seedUser(t)

	repo := NewURLShortenerRepository(testDB)

	longURL := "http://example.com/" + strings.Repeat("a", 1985)

	// Act
	err := repo.AddShortURL(userID, longURL, "toolong")

	// Assert
	require.Error(t, err)
}

func TestAddShortURL_ConcurrentDuplicate(t *testing.T) {
	// Arrange
	cleanDB(t)
	userID := seedUser(t)

	repo := NewURLShortenerRepository(testDB)

	var wg sync.WaitGroup
	errors := make(chan error, 2)

	// Act
	for i := 0; i < 2; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()
			err := repo.AddShortURL(userID, "http://google.com", "google")
			errors <- err
		}()
	}

	wg.Wait()
	close(errors)

	// Assert
	successCount := 0
	for err := range errors {
		if err == nil {
			successCount++
		}
	}

	assert.Equal(t, 1, successCount)

	var count int
	err := testDB.QueryRow(
		"SELECT COUNT(*) FROM urls WHERE slug = $1",
		"google",
	).Scan(&count)

	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestGetURLBySlug_Success(t *testing.T) {
	// Arrange
	cleanDB(t)
	userID := seedUser(t)
	slug := seedURL(t, userID)

	repo := NewURLShortenerRepository(testDB)

	// Act
	urlID, targetURL, err := repo.GetURLBySlug(slug)

	// Assert
	require.NoError(t, err)
	require.NotZero(t, urlID)
	require.Equal(t, "http://example.com", targetURL)

	var url URL
	err = testDB.Get(&url, "SELECT * FROM urls WHERE slug = $1", slug)
	require.NoError(t, err)

	assert.Equal(t, userID, url.UserID)
	assert.Equal(t, slug, url.Slug)
	assert.Equal(t, "http://example.com", url.TargetURL)
	assert.Equal(t, urlID, url.ID)
	assert.False(t, url.CreatedAt.IsZero())
}

func TestGetURLBySlug_NotFound(t *testing.T) {
	// Arrange
	cleanDB(t)
	repo := NewURLShortenerRepository(testDB)

	// Act
	urlID, targetURL, err := repo.GetURLBySlug("unknown")

	// Assert
	require.Error(t, err)
	assert.Equal(t, 0, urlID)
	assert.Equal(t, "", targetURL)
	assert.Contains(t, err.Error(), "not found")
}

func TestGetURLBySlug_MaxLengthSlug(t *testing.T) {
	cleanDB(t)
	userID := seedUser(t)

	slug := strings.Repeat("a", 100)
	_, err := testDB.Exec(
		`INSERT INTO urls (user_id, target_url, slug)
		 VALUES ($1, $2, $3)`,
		userID,
		"http://example.com",
		slug,
	)
	require.NoError(t, err)

	repo := NewURLShortenerRepository(testDB)

	urlID, targetURL, err := repo.GetURLBySlug(slug)

	require.NoError(t, err)
	assert.NotZero(t, urlID)
	assert.Equal(t, "http://example.com", targetURL)
}

func TestGetURLBySlug_SQLInjectionAttempt(t *testing.T) {
	// Arrange
	cleanDB(t)
	repo := NewURLShortenerRepository(testDB)

	slug := "'; DROP TABLE urls; --"

	// Act
	urlID, targetURL, err := repo.GetURLBySlug(slug)

	// Assert
	require.Error(t, err)
	assert.Equal(t, 0, urlID)
	assert.Equal(t, "", targetURL)
}

func TestGetURLBySlug_Concurrent(t *testing.T) {
	// Arrange
	cleanDB(t)
	userID := seedUser(t)
	slug := seedURL(t, userID)

	repo := NewURLShortenerRepository(testDB)

	var wg sync.WaitGroup

	for range 10 {
		wg.Add(1)

		go func() {
			defer wg.Done()
			// Act
			urlID, targetURL, err := repo.GetURLBySlug(slug)

			// Assert
			require.NoError(t, err)
			assert.NotZero(t, urlID)
			assert.Equal(t, "http://example.com", targetURL)
		}()
	}

	wg.Wait()
}

func TestGetURLByUserId_Success(t *testing.T) {
	// Arrange
	cleanDB(t)
	userID := seedUser(t)
	slug := seedURL(t, userID)

	var expected URL
	err := testDB.Get(&expected, "SELECT * FROM urls WHERE slug = $1", slug)
	require.NoError(t, err)

	repo := NewURLShortenerRepository(testDB)

	// Act
	url, err := repo.GetURLByUserId(userID, expected.ID)

	// Assert
	require.NoError(t, err)

	assert.Equal(t, expected.ID, int(url.UrlId))
	assert.Equal(t, expected.UserID, int(url.UserId))
	assert.Equal(t, expected.Slug, url.Slug)
	assert.Equal(t, expected.TargetURL, url.TargetUrl)
	assert.NotEmpty(t, url.CreatedAt)
}

func TestGetURLByUserId_InvalidUserID(t *testing.T) {
	// Arrange
	cleanDB(t)

	userID := seedUser(t)
	_ = seedURL(t, userID)

	invalidUserId := 0

	repo := NewURLShortenerRepository(testDB)

	// Act
	_, err := repo.GetURLByUserId(invalidUserId, 1)

	// Assert
	require.Error(t, err)
}

func TestGetURLByUserId_InvalidURLId(t *testing.T) {
	// Arrange
	cleanDB(t)

	userID := seedUser(t)
	seedURL(t, userID)

	invalidURLId := 0

	repo := NewURLShortenerRepository(testDB)

	// Act
	_, err := repo.GetURLByUserId(userID, invalidURLId)

	// Assert
	require.Error(t, err)
}

func TestGetURLByUserId_Concurrent(t *testing.T) {
	// Arrange
	cleanDB(t)
	userID := seedUser(t)
	seedURL(t, userID)

	repo := NewURLShortenerRepository(testDB)

	var wg sync.WaitGroup

	for range 10 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			url, err := repo.GetURLByUserId(userID, 1)

			require.NoError(t, err)
			assert.NotZero(t, url)
			assert.Equal(t, "http://example.com", url.TargetUrl)
		}()
	}
}
