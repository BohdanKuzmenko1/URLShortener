package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/BohdanKuzmenko1/URLShortener/services/auth-service/internal/repository"
	"github.com/BohdanKuzmenko1/URLShortener/shared"
	"github.com/golang-jwt/jwt"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
	"os"
	"strconv"
	"time"
)

const accessTokenTTL = 8 * time.Hour
const refreshTokenTTL = 7 * 24 * time.Hour

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
	repo  repository.AuthRepository
	redis *redis.Client
}

// NewAuthService returns a new Auth service backed by the given repository and Redis client.
func NewAuthService(repo repository.AuthRepository, redisClient *redis.Client) Auth {
	return &authService{
		repo:  repo,
		redis: redisClient,
	}
}

// RefreshToken invalidates the provided refresh token and issues a new token pair.
// Returns an error if the token is invalid, expired, or the request failed.
func (a authService) RefreshToken(ctx context.Context, refreshToken string) (*shared.TokenPair, error) {
	userID, err := a.getUserIdByRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, err
	}

	err = a.deleteRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, err
	}

	return a.generateTokenPair(ctx, userID)
}

// Logout invalidates the provided refresh token, ending the user session.
// Returns an error if the token could not be deleted.
func (a authService) Logout(ctx context.Context, refreshToken string) error {
	return a.deleteRefreshToken(ctx, refreshToken)
}

// Login verifies user credentials and returns a token pair on success.
// Returns an error if the user is not found or the password is incorrect.
func (a authService) Login(ctx context.Context, email, password string) (*shared.TokenPair, error) {
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
func (a authService) Register(ctx context.Context, email, password string) (*shared.TokenPair, error) {
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
func (a authService) generateTokenPair(ctx context.Context, userId int) (*shared.TokenPair, error) {
	accessToken, err := a.generateAccessToken(userId)
	if err != nil {
		return nil, err
	}

	refreshToken := a.generateRefreshToken()

	err = a.saveRefreshToken(ctx, userId, refreshToken)
	if err != nil {
		return nil, err
	}

	return &shared.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(accessTokenTTL.Seconds()),
		TokenType:    "Bearer",
	}, nil
}

// generateRefreshToken returns a cryptographically random 32-byte hex-encoded token.
func (a authService) generateRefreshToken() string {
	bytes := make([]byte, 32)
	read, err := rand.Read(bytes)
	if err != nil {
		return ""
	}

	return hex.EncodeToString(bytes[:read])
}

// generateAccessToken creates a signed JWT access token for the given user ID.
// Returns an error if the token could not be signed.
func (a authService) generateAccessToken(userId int) (string, error) {
	now := time.Now()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, &tokenClaims{
		jwt.StandardClaims{
			ExpiresAt: now.Add(accessTokenTTL).Unix(),
			IssuedAt:  now.Unix(),
		},
		userId,
	})

	return token.SignedString([]byte(os.Getenv("JWT_SIGNING_KEY")))
}

// saveRefreshToken stores the refresh token in Redis associated with the given user ID.
// The token expires after refreshTokenTTL.
func (a authService) saveRefreshToken(ctx context.Context, userID int, refreshToken string) error {
	key := fmt.Sprintf("refresh:%s", refreshToken)

	return a.redis.Set(ctx, key, userID, refreshTokenTTL).Err()
}

// getUserIdByRefreshToken retrieves the user ID associated with the given refresh token from Redis.
// Returns an error if the token is not found or the request failed.
func (a authService) getUserIdByRefreshToken(ctx context.Context, refreshToken string) (int, error) {
	key := fmt.Sprintf("refresh:%s", refreshToken)

	userIdStr, err := a.redis.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return 0, errors.New("refresh token not found or expired")
	}
	if err != nil {
		return 0, err
	}

	userId, err := strconv.Atoi(userIdStr)
	if err != nil {
		return 0, fmt.Errorf("invalid user id stored in redis: %w", err)
	}

	return userId, nil
}

// deleteRefreshToken removes the refresh token from Redis, invalidating the session.
func (a authService) deleteRefreshToken(ctx context.Context, refreshToken string) error {
	key := fmt.Sprintf("refresh:%s", refreshToken)
	return a.redis.Del(ctx, key).Err()
}
