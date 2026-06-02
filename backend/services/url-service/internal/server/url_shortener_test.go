package server_test

import (
	"context"
	"errors"
	"fmt"
	pb "github.com/BohdanKuzmenko1/URLShortener/proto"
	"github.com/BohdanKuzmenko1/URLShortener/services/url-service/internal"
	"github.com/BohdanKuzmenko1/URLShortener/services/url-service/internal/server"
	"github.com/golang-jwt/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/metadata"
	"os"
	"testing"
	"time"
)

type testTokenClaims struct {
	jwt.StandardClaims
	UserId int `json:"user_id"`
}

func generateToken(t *testing.T, userId int, isExpired bool) (string, error) {
	t.Helper()

	if isExpired {
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, &testTokenClaims{
			jwt.StandardClaims{
				ExpiresAt: time.Now().Add(-24 * time.Hour).Unix(),
				IssuedAt:  time.Now().Add(-48 * time.Hour).Unix(),
			},
			userId,
		})

		signedToken, err := token.SignedString([]byte(os.Getenv("JWT_SIGNING_KEY")))
		if err != nil {
			return "", err
		}

		return fmt.Sprintf("Bearer %s", signedToken), nil
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, &testTokenClaims{
		jwt.StandardClaims{
			ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
			IssuedAt:  time.Now().Unix(),
		},
		userId,
	})

	signedToken, err := token.SignedString([]byte(os.Getenv("JWT_SIGNING_KEY")))
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Bearer %s", signedToken), nil
}

type MockURLShortenerService struct {
	mock.Mock
}

func (m *MockURLShortenerService) GetURL(ctx context.Context, userId int, urlId int) (internal.ShortURL, error) {
	args := m.Called(ctx, userId, urlId)
	return args.Get(0).(internal.ShortURL), args.Error(1)
}

func (m *MockURLShortenerService) GenerateShortURL(ctx context.Context, userId int, targetURL, slug string) (string, error) {
	args := m.Called(ctx, userId, targetURL, slug)
	return args.String(0), args.Error(1)
}

func (m *MockURLShortenerService) ResolveSlug(ctx context.Context, redirect internal.Redirect) (string, error) {
	args := m.Called(ctx, redirect)
	return args.String(0), args.Error(1)
}

func TestURLServer_CreateShortURL(t *testing.T) {
	tests := []struct {
		name           string
		userId         int
		slug           string
		targetURL      string
		mockResult     string
		mockError      error
		expectedError  bool
		expectedResult string
		tokenExpired   bool
		tokenExists    bool
		mockCall       bool
		metadataExists bool
	}{
		{
			name:           "Success",
			userId:         1,
			slug:           "regular-slug",
			targetURL:      "https://google.com",
			mockResult:     "https://localhost:8080/regular-slug",
			expectedResult: "https://localhost:8080/regular-slug",
			tokenExists:    true,
			mockCall:       true,
			metadataExists: true,
		},
		{
			name:           "Service returns error",
			userId:         1,
			slug:           "regular-slug",
			targetURL:      "https://google.com",
			mockError:      errors.New("service error"),
			expectedError:  true,
			tokenExists:    true,
			mockCall:       true,
			metadataExists: true,
		},
		{
			name:           "Empty target url",
			userId:         1,
			slug:           "regular-slug",
			expectedError:  true,
			tokenExists:    true,
			metadataExists: true,
		},
		{
			name:           "No token for authorization in metadata",
			userId:         1,
			slug:           "regular-slug",
			targetURL:      "https://google.com",
			mockResult:     "https://localhost:8080/regular-slug",
			expectedResult: "https://localhost:8080/regular-slug",
			expectedError:  true,
			metadataExists: true,
		},
		{
			name:           "No metadata",
			userId:         1,
			slug:           "regular-slug",
			targetURL:      "https://google.com",
			mockResult:     "https://localhost:8080/regular-slug",
			expectedResult: "https://localhost:8080/regular-slug",
			expectedError:  true,
		},
		{
			name:           "Expired token",
			userId:         1,
			slug:           "regular-slug",
			targetURL:      "https://google.com",
			tokenExists:    true,
			metadataExists: true,
			tokenExpired:   true,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockService := new(MockURLShortenerService)

			srv := server.NewURLServer(mockService)

			var (
				token string
				err   error
			)

			if tt.tokenExists {
				token, err = generateToken(t, tt.userId, tt.tokenExpired)
				if err != nil {
					t.Fatalf("Error generating token: %v", err)
				}
			} else {
				token = ""
			}

			ctx := context.Background()

			if tt.metadataExists {
				ctx = metadata.NewIncomingContext(ctx, metadata.Pairs("authorization", token))
			}

			if tt.mockCall {
				mockService.
					On("GenerateShortURL", mock.Anything, tt.userId, tt.targetURL, tt.slug).
					Return(tt.mockResult, tt.mockError)
			}

			req := &pb.CreateShortUrlRequest{
				TargetUrl: tt.targetURL,
				Slug:      tt.slug,
			}

			// Act
			shortURL, err := srv.CreateShortURL(ctx, req)

			// Assert
			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, shortURL)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, shortURL.ShortUrl)
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestURLServer_ResolveSlug(t *testing.T) {
	tests := []struct {
		name           string
		redirect       internal.Redirect
		mockResult     string
		mockError      error
		expectedError  bool
		expectedResult string
	}{
		{
			name: "Success",
			redirect: internal.Redirect{
				Slug:      "regular-slug",
				UserAgent: "agent",
				ClientIP:  "127.0.0.1",
				Country:   "XX",
				Language:  "en-US",
			},
			mockResult:     "https://localhost:8080/regular-slug",
			expectedResult: "https://localhost:8080/regular-slug",
		},
		{
			name: "Service returns error",
			redirect: internal.Redirect{
				Slug:      "not-found-slug",
				UserAgent: "agent",
				ClientIP:  "127.0.0.1",
				Country:   "XX",
				Language:  "en-US",
			},
			mockError:     errors.New("slug not found"),
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockService := new(MockURLShortenerService)
			srv := server.NewURLServer(mockService)

			mockService.
				On("ResolveSlug", mock.Anything, tt.redirect).
				Return(tt.mockResult, tt.mockError)

			req := &pb.ResolveSlugRequest{
				Redirect: &pb.URLServiceRedirect{
					Slug:      tt.redirect.Slug,
					UserAgent: tt.redirect.UserAgent,
					ClientIp:  tt.redirect.ClientIP,
					Country:   tt.redirect.Country,
					Language:  tt.redirect.Language,
					Referer:   tt.redirect.Referer,
				},
			}

			// Act
			resp, err := srv.ResolveSlug(context.Background(), req)

			// Assert
			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, resp.TargetUrl)
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestURLServer_GetURL(t *testing.T) {
	tests := []struct {
		name           string
		userId         int
		urlId          int32
		mockResult     internal.ShortURL
		mockError      error
		expectedError  bool
		tokenExpired   bool
		tokenExists    bool
		metadataExists bool
	}{
		{
			name:   "Success",
			userId: 1,
			urlId:  1,
			mockResult: internal.ShortURL{
				UrlId:     1,
				TargetUrl: "https://google.com",
				Slug:      "abc123",
				CreatedAt: "2026-01-01",
			},
			tokenExists:    true,
			metadataExists: true,
		},
		{
			name:           "Error from service",
			userId:         1,
			urlId:          1,
			mockError:      errors.New("url not found"),
			expectedError:  true,
			tokenExists:    true,
			metadataExists: true,
		},
		{
			name:           "No metadata",
			userId:         1,
			urlId:          1,
			expectedError:  true,
			metadataExists: false,
		},
		{
			name:           "No token in metadata",
			userId:         1,
			urlId:          1,
			expectedError:  true,
			tokenExists:    false,
			metadataExists: true,
		},
		{
			name:           "Expired token",
			userId:         1,
			urlId:          1,
			expectedError:  true,
			tokenExists:    true,
			tokenExpired:   true,
			metadataExists: true,
		},
		{
			name:           "Invalid userId — zero value",
			userId:         0,
			urlId:          1,
			expectedError:  true,
			tokenExists:    true,
			metadataExists: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockService := new(MockURLShortenerService)
			srv := server.NewURLServer(mockService)

			var token string
			var err error

			if tt.tokenExists {
				token, err = generateToken(t, tt.userId, tt.tokenExpired)
				if err != nil {
					t.Fatalf("failed to generate token: %v", err)
				}
			}

			ctx := context.Background()
			if tt.metadataExists {
				ctx = metadata.NewIncomingContext(ctx, metadata.Pairs("authorization", token))
			}

			shouldCallService := tt.metadataExists &&
				tt.tokenExists &&
				!tt.tokenExpired &&
				tt.userId != 0

			if shouldCallService {
				mockService.
					On("GetURL", mock.Anything, tt.userId, int(tt.urlId)).
					Return(tt.mockResult, tt.mockError)
			}

			req := &pb.GetURLRequest{
				UrlId: tt.urlId,
			}

			// Act
			resp, err := srv.GetURL(ctx, req)

			// Assert
			if tt.expectedError || tt.mockError != nil {
				assert.Error(t, err)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Equal(t, tt.mockResult.UrlId, resp.ShortUrl.UrlId)
				assert.Equal(t, tt.mockResult.TargetUrl, resp.ShortUrl.TargetUrl)
				assert.Equal(t, tt.mockResult.Slug, resp.ShortUrl.Slug)
				assert.Equal(t, tt.mockResult.CreatedAt, resp.ShortUrl.CreatedAt)
			}

			mockService.AssertExpectations(t)
		})
	}
}
