package repository_test

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/BohdanKuzmenko1/URLShortener/services/auth-service/internal/repository"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"golang.org/x/crypto/bcrypt"
)

var testDB *sqlx.DB

// ---------------------------------------------------------------------------
// TestMain — single Postgres container for all tests
// ---------------------------------------------------------------------------

func TestMain(m *testing.M) {
	ctx := context.Background()

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

	var id int
	err = testDB.QueryRowContext(
		context.Background(),
		"INSERT INTO users (email, password) VALUES ($1, $2) RETURNING id",
		email, string(hash),
	).Scan(&id)
	require.NoError(t, err, "seedUser: insert failed")

	return id
}

func cleanDB(t *testing.T) {
	t.Helper()

	_, err := testDB.Exec("TRUNCATE TABLE users RESTART IDENTITY CASCADE")
	require.NoError(t, err)
}

type User struct {
	ID           int       `db:"id"`
	Email        string    `db:"email"`
	PasswordHash string    `db:"password"`
	CreatedAt    time.Time `db:"created_at"`
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestAuthRepository_Register(t *testing.T) {
	t.Cleanup(func() { cleanDB(t) })

	email := "new@gmail.com"
	passwordHash := "secret123"

	authRepo := repository.NewAuthRepository(testDB)

	id, err := authRepo.Register(context.Background(), email, passwordHash)
	require.NoError(t, err, "register should not fail")
	require.Greater(t, id, 0, "registered user should have valid id")

	var user User
	err = testDB.Get(&user, "SELECT * FROM users WHERE id = $1", id)
	require.NoError(t, err, "registered user should exist in db")
	require.Equal(t, email, user.Email)
	require.Equal(t, passwordHash, user.PasswordHash)

	t.Logf("registered user: %+v", user)
}

func TestAuthRepository_Register_DuplicateEmail(t *testing.T) {
	t.Cleanup(func() { cleanDB(t) })

	email := "dup@gmail.com"
	password := "secret123"

	authRepo := repository.NewAuthRepository(testDB)

	_, err := authRepo.Register(context.Background(), email, password)
	require.NoError(t, err, "first registration should succeed")

	_, err = authRepo.Register(context.Background(), email, password)
	require.Error(t, err, "duplicate email should return error")
}

func TestAuthRepository_GetUserByEmail(t *testing.T) {
	t.Cleanup(func() { cleanDB(t) })

	email := "get@gmail.com"
	password := "secret123"

	expectedID := seedUser(t, email, password)

	authRepo := repository.NewAuthRepository(testDB)

	id, passwordHash, err := authRepo.GetUserByEmail(context.Background(), email)
	require.NoError(t, err)
	require.Equal(t, expectedID, id)
	require.NotEmpty(t, passwordHash)
}

func TestAuthRepository_GetUserByEmail_NotFound(t *testing.T) {
	t.Cleanup(func() { cleanDB(t) })

	authRepo := repository.NewAuthRepository(testDB)

	_, _, err := authRepo.GetUserByEmail(context.Background(), "nonexistent@gmail.com")
	require.Error(t, err)
	require.ErrorIs(t, err, sql.ErrNoRows)
}
