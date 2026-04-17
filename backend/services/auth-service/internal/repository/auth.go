package repository

import (
	"context"
	"fmt"
	"github.com/jmoiron/sqlx"
)

// AuthRepository interface with functions to create and verify user
type AuthRepository interface {
	Login(ctx context.Context, email, passwordHash string) (int, error)
	Register(ctx context.Context, email, passwordHash string) (int, error)
}

// authRepository implements AuthRepository using a PostgreSQL database.
type authRepository struct {
	db *sqlx.DB
}

// Login verifies user credentials against the database records.
// Returns the user ID on success, or 0 and an error if the credentials
// are invalid or the database request failed.
func (a authRepository) Login(ctx context.Context, email, passwordHash string) (int, error) {
	var id int

	query := fmt.Sprintf("SELECT id FROM %s WHERE email=$1 AND password=$2", usersTable)

	err := a.db.GetContext(ctx, &id, query, email, passwordHash)
	if err != nil {
		return 0, err
	}

	return id, nil
}

// Register creates a new user with the provided email and password hash.
// Returns the new user ID on success, or 0 and an error if the request
// failed or a user with the given email already exists.
func (a authRepository) Register(ctx context.Context, email, passwordHash string) (int, error) {
	var id int

	query := fmt.Sprintf("INSERT INTO %s (email, password) VALUES ($1, $2) RETURNING id", usersTable)

	row := a.db.QueryRowContext(ctx, query, email, passwordHash)

	if err := row.Scan(&id); err != nil {
		return 0, err
	}

	return id, nil
}

// NewAuthRepository returns a new AuthRepository backed by the given sqlx.DB instance.
func NewAuthRepository(db *sqlx.DB) AuthRepository {
	return &authRepository{db: db}
}
