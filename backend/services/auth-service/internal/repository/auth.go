package repository

import (
	"context"
	"fmt"
	"github.com/jmoiron/sqlx"
)

// AuthRepository interface with functions to create and verify user
type AuthRepository interface {
	GetUserByEmail(ctx context.Context, email string) (int, string, error)
	Register(ctx context.Context, email, passwordHash string) (int, error)
}

// authRepository implements AuthRepository using a PostgreSQL database.
type authRepository struct {
	db *sqlx.DB
}

func (a authRepository) GetUserByEmail(ctx context.Context, email string) (int, string, error) {
	var user struct {
		ID           int    `db:"id"`
		PasswordHash string `db:"password"`
	}

	query := fmt.Sprintf("SELECT id, password FROM %s WHERE email=$1", usersTable)

	err := a.db.GetContext(ctx, &user, query, email)
	if err != nil {
		return 0, "", err
	}

	return user.ID, user.PasswordHash, nil
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
