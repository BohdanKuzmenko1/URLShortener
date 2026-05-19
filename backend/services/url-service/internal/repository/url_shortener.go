package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/BohdanKuzmenko1/URLShortener/services/url-service/internal"
	"github.com/jmoiron/sqlx"
)

var ErrNotFound = errors.New("url not found")

type urlShortenerRepository struct {
	db *sqlx.DB
}

type URLShortenerRepository interface {
	AddShortURL(ctx context.Context, userId int, targetURL, slug string) error
	GetURLBySlug(ctx context.Context, slug string) (int, string, error)
	GetURLByUserId(ctx context.Context, userId, urlId int) (internal.ShortURL, error)
}

func NewURLShortenerRepository(db *sqlx.DB) URLShortenerRepository {
	return &urlShortenerRepository{
		db: db,
	}
}

func (U urlShortenerRepository) GetURLByUserId(ctx context.Context, userId, urlId int) (internal.ShortURL, error) {
	var url internal.ShortURL

	query := `
		SELECT id, user_id, slug, target_url, created_at
		FROM urls
		WHERE id = $1 AND user_id = $2
	`

	err := U.db.QueryRowContext(ctx, query, urlId, userId).Scan(
		&url.UrlId,
		&url.UserId,
		&url.Slug,
		&url.TargetUrl,
		&url.CreatedAt,
	)
	if err != nil {
		return internal.ShortURL{}, err
	}

	return url, nil
}

func (U urlShortenerRepository) GetURLBySlug(ctx context.Context, slug string) (int, string, error) {
	var id int
	var targetURL string

	query := `SELECT id, target_url FROM urls WHERE slug = $1`

	err := U.db.QueryRowContext(ctx, query, slug).Scan(&id, &targetURL)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, "", ErrNotFound
		}
		return 0, "", fmt.Errorf("failed to get URL: %w", err)
	}

	return id, targetURL, nil
}

func (U urlShortenerRepository) AddShortURL(ctx context.Context, userId int, targetURL, slug string) error {
	query := fmt.Sprintf("INSERT INTO %s (user_id, target_url, slug) VALUES ($1, $2, $3)", URLsTable)

	_, err := U.db.ExecContext(
		ctx, query, userId, targetURL, slug,
	)

	if err != nil {
		return fmt.Errorf("failed to insert URL: %w", err)
	}

	return nil
}
