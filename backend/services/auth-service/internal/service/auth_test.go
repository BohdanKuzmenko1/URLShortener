package service_test

import (
	"context"
	"errors"
	"github.com/BohdanKuzmenko1/URLShortener/services/auth-service/internal/service"
	"github.com/stretchr/testify/assert"
	"testing"

	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
)

// ---------------------------------------------------------------------------
// Mock: AuthRepository
// ---------------------------------------------------------------------------

type MockAuthRepository struct {
	mock.Mock
}

func (m *MockAuthRepository) GetUserByEmail(ctx context.Context, email string) (int, string, error) {
	args := m.Called(ctx, email)
	return args.Int(0), args.String(1), args.Error(2)
}

func (m *MockAuthRepository) Register(ctx context.Context, email, passwordHash string) (int, error) {
	args := m.Called(ctx, email, passwordHash)
	return args.Int(0), args.Error(1)
}

type MockAuthStorage struct {
	mock.Mock
}

func (m *MockAuthStorage) SaveRefreshToken(ctx context.Context, userID int, refreshToken string) error {
	args := m.Called(ctx, userID, refreshToken)
	return args.Error(0)
}

func (m *MockAuthStorage) GetUserIdByRefreshToken(ctx context.Context, refreshToken string) (int, error) {
	args := m.Called(ctx, refreshToken)
	return args.Int(0), args.Error(1)
}

func (m *MockAuthStorage) DeleteRefreshToken(ctx context.Context, refreshToken string) error {
	args := m.Called(ctx, refreshToken)
	return args.Error(0)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// mustHash returns bcrypt-hash password or drops test with error.
func mustHash(t *testing.T, password string) string {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("bcrypt failed: %v", err)
	}
	return string(hash)
}

// ---------------------------------------------------------------------------
// Tests: Logout
// ---------------------------------------------------------------------------

func TestAuthService_Logout(t *testing.T) {
	tests := []struct {
		name           string
		refreshToken   string
		deleteTokenErr error
		expectedErr    string
	}{
		{
			name:         "success",
			refreshToken: "refresh-token",
		},
		{
			name:           "redis unavailable error",
			refreshToken:   "refresh-token",
			deleteTokenErr: errors.New("redis unavailable"),
			expectedErr:    "redis unavailable",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			mockRepo := new(MockAuthRepository)
			mockStorage := new(MockAuthStorage)

			t.Setenv("JWT_SIGNING_KEY", "test-secret-key")

			svc := service.NewAuthService(mockRepo, mockStorage)

			mockStorage.On("DeleteRefreshToken", mock.Anything, tc.refreshToken).
				Return(tc.deleteTokenErr)

			// Action
			err := svc.Logout(context.Background(), tc.refreshToken)

			// Assert
			if tc.expectedErr != "" {
				assert.ErrorContains(t, err, tc.expectedErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Tests: RefreshToken
// ---------------------------------------------------------------------------

func TestAuthService_RefreshToken(t *testing.T) {
	tests := []struct {
		name            string
		refreshToken    string
		storageUserId   int
		getUserErr      error
		deleteTokenErr  error
		saveTokenErr    error
		expectedErr     string
		expectTokenPair bool
	}{
		{
			name:            "success",
			refreshToken:    "refresh-token",
			storageUserId:   1,
			expectTokenPair: true,
		},
		{
			name:         "invalid refresh token",
			refreshToken: "nonexistent-token",
			getUserErr:   errors.New("token not found"),
			expectedErr:  "token not found",
		},
		{
			name:           "delete token fails",
			refreshToken:   "refresh-token",
			storageUserId:  1,
			deleteTokenErr: errors.New("redis unavailable"),
			expectedErr:    "redis unavailable",
		},
		{
			name:          "save new token fails",
			refreshToken:  "refresh-token",
			storageUserId: 1,
			saveTokenErr:  errors.New("redis unavailable"),
			expectedErr:   "redis unavailable",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			mockRepo := new(MockAuthRepository)
			mockStorage := new(MockAuthStorage)

			t.Setenv("JWT_SIGNING_KEY", "test-secret-key")

			svc := service.NewAuthService(mockRepo, mockStorage)

			mockStorage.On("GetUserIdByRefreshToken", mock.Anything, tc.refreshToken).
				Return(tc.storageUserId, tc.getUserErr)

			if tc.getUserErr == nil {
				mockStorage.On("DeleteRefreshToken", mock.Anything, tc.refreshToken).
					Return(tc.deleteTokenErr)
			}

			if tc.getUserErr == nil && tc.deleteTokenErr == nil {
				mockStorage.On("SaveRefreshToken", mock.Anything, tc.storageUserId, mock.AnythingOfType("string")).
					Return(tc.saveTokenErr)
			}

			// Action
			result, err := svc.RefreshToken(context.Background(), tc.refreshToken)

			// Assert
			if tc.expectedErr != "" {
				assert.Nil(t, result)
				assert.ErrorContains(t, err, tc.expectedErr)
			} else {
				assert.NoError(t, err)
			}

			if tc.expectTokenPair {
				assert.NotNil(t, result)
				assert.NotEmpty(t, result.AccessToken)
				assert.NotEmpty(t, result.RefreshToken)
				assert.Equal(t, "Bearer", result.TokenType)
				assert.Equal(t, int(service.AccessTokenTTL.Seconds()), result.ExpiresIn)
			}

			mockStorage.AssertExpectations(t)
		})
	}
}

// ---------------------------------------------------------------------------
// Tests: Register
// ---------------------------------------------------------------------------

func TestAuthService_Register(t *testing.T) {
	tests := []struct {
		name            string
		email           string
		password        string
		repoUserId      int
		repoErr         error
		saveTokenErr    error
		expectedErr     string
		expectTokenPair bool
	}{
		{
			name:            "success",
			email:           "user@example.com",
			password:        "password123",
			repoUserId:      1,
			expectTokenPair: true,
		},
		{
			name:        "email already exists",
			email:       "existing@example.com",
			password:    "password123",
			repoErr:     errors.New("email already exists"),
			expectedErr: "email already exists",
		},
		{
			name:         "failed to save refresh token",
			email:        "user@example.com",
			password:     "password123",
			repoUserId:   1,
			saveTokenErr: errors.New("redis unavailable"),
			expectedErr:  "redis unavailable",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			mockRepo := new(MockAuthRepository)
			mockStorage := new(MockAuthStorage)

			t.Setenv("JWT_SIGNING_KEY", "test-secret-key")

			svc := service.NewAuthService(mockRepo, mockStorage)

			mockRepo.On("Register", mock.Anything, tc.email, mock.AnythingOfType("string")).
				Return(tc.repoUserId, tc.repoErr)

			if tc.repoErr == nil {
				mockStorage.On("SaveRefreshToken", mock.Anything, tc.repoUserId, mock.AnythingOfType("string")).
					Return(tc.saveTokenErr)
			}

			// Action
			result, err := svc.Register(context.Background(), tc.email, tc.password)

			// Assert
			if tc.expectedErr != "" {
				assert.Nil(t, result)
				assert.ErrorContains(t, err, tc.expectedErr)
			} else {
				assert.NoError(t, err)
			}

			if tc.expectTokenPair {
				assert.NotNil(t, result)
				assert.NotEmpty(t, result.AccessToken)
				assert.NotEmpty(t, result.RefreshToken)
				assert.Equal(t, "Bearer", result.TokenType)
				assert.Equal(t, int(service.AccessTokenTTL.Seconds()), result.ExpiresIn)
			}

			mockRepo.AssertExpectations(t)
			mockStorage.AssertExpectations(t)
		})
	}
}

// ---------------------------------------------------------------------------
// Tests: Login
// ---------------------------------------------------------------------------

func TestAuthService_Login(t *testing.T) {
	tests := []struct {
		name            string
		email           string
		password        string
		repoUserId      int
		plainPassword   string
		repoErr         error
		saveTokenErr    error
		expectedErr     string
		expectTokenPair bool
	}{
		{
			name:            "success",
			email:           "user@example.com",
			password:        "password123",
			repoUserId:      1,
			plainPassword:   "password123",
			expectTokenPair: true,
		},
		{
			name:        "user not found",
			email:       "notfound@example.com",
			password:    "password123",
			repoErr:     errors.New("user not found"),
			expectedErr: "user not found",
		},
		{
			name:          "wrong password",
			email:         "user@example.com",
			password:      "wrongpassword",
			repoUserId:    1,
			plainPassword: "password123",
			expectedErr:   "invalid credentials",
		},
		{
			name:          "failed to save refresh token",
			email:         "user@example.com",
			password:      "password123",
			repoUserId:    1,
			plainPassword: "password123",
			saveTokenErr:  errors.New("redis unavailable"),
			expectedErr:   "redis unavailable",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			var repoHash string
			if tc.plainPassword != "" {
				repoHash = mustHash(t, tc.plainPassword)
			}

			mockRepo := new(MockAuthRepository)
			mockStorage := new(MockAuthStorage)

			t.Setenv("JWT_SIGNING_KEY", "test-secret-key")

			svc := service.NewAuthService(mockRepo, mockStorage)

			mockRepo.On("GetUserByEmail", mock.Anything, tc.email).
				Return(tc.repoUserId, repoHash, tc.repoErr)

			if tc.repoErr == nil && tc.expectedErr != "invalid credentials" {
				mockStorage.On("SaveRefreshToken", mock.Anything, tc.repoUserId, mock.AnythingOfType("string")).
					Return(tc.saveTokenErr)
			}

			// Act
			result, err := svc.Login(context.Background(), tc.email, tc.password)

			// Assert
			if tc.expectedErr != "" {
				assert.Nil(t, result)
				assert.ErrorContains(t, err, tc.expectedErr)
			} else {
				assert.NoError(t, err)
			}

			if tc.expectTokenPair {
				assert.NotNil(t, result)
				assert.NotEmpty(t, result.AccessToken)
				assert.NotEmpty(t, result.RefreshToken)
				assert.Equal(t, "Bearer", result.TokenType)
				assert.Equal(t, int(service.AccessTokenTTL.Seconds()), result.ExpiresIn)
			}

			mockRepo.AssertExpectations(t)
			mockStorage.AssertExpectations(t)
		})
	}
}
