package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/BohdanKuzmenko1/URLShortener/services/auth-service/internal/repository"
	"github.com/BohdanKuzmenko1/URLShortener/services/auth-service/internal/storage"
	"github.com/BohdanKuzmenko1/URLShortener/shared"
	"github.com/golang-jwt/jwt"
	"golang.org/x/crypto/bcrypt"
	"os"
	"time"
)

const AccessTokenTTL = 8 * time.Hour

type tokenClaims struct {
	jwt.StandardClaims
	UserId int `json:"user_id"`
}

// Auth defines the authentication operations supported by the service.
type Auth interface {
	Login(ctx context.Context, email, password string) (*shared.TokenPair, error)
	Register(ctx context.Context, email, password string) (*shared.TokenPair, error)
	RefreshToken(ctx context.Context, refreshToken string) (*shared.TokenPair, error)
	Logout(ctx context.Context, refreshToken string) error
}

type authService struct {
	repo       repository.AuthRepository
	storage    storage.AuthStorage
	signingKey []byte
}

// NewAuthService returns a new Auth service backed by the given repository and Redis client.
func NewAuthService(repo repository.AuthRepository, storage storage.AuthStorage) Auth {
	return &authService{
		repo:       repo,
		storage:    storage,
		signingKey: []byte(os.Getenv("JWT_SIGNING_KEY")),
	}
}

// RefreshToken invalidates the provided refresh token and issues a new token pair.
// Returns an error if the token is invalid, expired, or the request failed.
func (a *authService) RefreshToken(ctx context.Context, refreshToken string) (*shared.TokenPair, error) {
	userID, err := a.storage.GetUserIdByRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, err
	}

	err = a.storage.DeleteRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, err
	}

	return a.generateTokenPair(ctx, userID)
}

// Logout invalidates the provided refresh token, ending the user session.
// Returns an error if the token could not be deleted.
func (a *authService) Logout(ctx context.Context, refreshToken string) error {
	return a.storage.DeleteRefreshToken(ctx, refreshToken)
}

// Login verifies user credentials and returns a token pair on success.
// Returns an error if the user is not found or the password is incorrect.
func (a *authService) Login(ctx context.Context, email, password string) (*shared.TokenPair, error) {
	userId, storedHash, err := a.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	return a.generateTokenPair(ctx, userId)
}

// Register creates a new user account with a bcrypt-hashed password and returns a token pair on success.
// Returns an error if the email already exists or the request failed.
func (a *authService) Register(ctx context.Context, email, password string) (*shared.TokenPair, error) {
	passwordHash, err := generatePasswordHash(password)
	if err != nil {
		return nil, err
	}

	userId, err := a.repo.Register(ctx, email, passwordHash)
	if err != nil {
		return nil, err
	}

	return a.generateTokenPair(ctx, userId)
}

// generatePasswordHash returns a bcrypt hash of the given password.
// Returns an error if hashing fails.
func generatePasswordHash(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// generateTokenPair creates a new access and refresh token pair for the given user ID,
// saves the refresh token to Redis, and returns the token pair.
func (a *authService) generateTokenPair(ctx context.Context, userId int) (*shared.TokenPair, error) {
	accessToken, err := a.generateAccessToken(userId)
	if err != nil {
		return nil, err
	}

	refreshToken, err := a.generateRefreshToken()
	if err != nil {
		return nil, err
	}

	err = a.storage.SaveRefreshToken(ctx, userId, refreshToken)
	if err != nil {
		return nil, err
	}

	return &shared.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(AccessTokenTTL.Seconds()),
		TokenType:    "Bearer",
	}, nil
}

// generateRefreshToken returns a cryptographically random 32-byte hex-encoded token.
func (a *authService) generateRefreshToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate refresh token: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// generateAccessToken creates a signed JWT access token for the given user ID.
// Returns an error if the token could not be signed.
func (a *authService) generateAccessToken(userId int) (string, error) {
	now := time.Now()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, &tokenClaims{
		jwt.StandardClaims{
			ExpiresAt: now.Add(AccessTokenTTL).Unix(),
			IssuedAt:  now.Unix(),
		},
		userId,
	})

	return token.SignedString(a.signingKey)
}
