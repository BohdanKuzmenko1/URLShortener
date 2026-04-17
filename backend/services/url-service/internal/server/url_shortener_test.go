package server

import (
	"context"
	"errors"
	"fmt"
	pb "github.com/BohdanKuzmenko1/URLShortener/proto"
	"github.com/BohdanKuzmenko1/URLShortener/services/url-service/internal"
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

type MockURLShortenerService struct {
	mock.Mock
}

func (m *MockURLShortenerService) GetURL(userId int, urlId int) (internal.ShortURL, error) {
	args := m.Called(userId, urlId)
	return args.Get(0).(internal.ShortURL), args.Error(1)
}

func (m *MockURLShortenerService) GenerateShortURL(userId int, targetURL, slug string) (string, error) {
	args := m.Called(userId, targetURL, slug)
	return args.String(0), args.Error(1)
}

func (m *MockURLShortenerService) ResolveSlug(redirect internal.Redirect) (string, error) {
	args := m.Called(redirect)
	return args.String(0), args.Error(1)
}

func generateToken(userId int, isExpired bool) (string, error) {
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
			slug:           "test-slug",
			targetURL:      "https://example.com",
			mockResult:     "https://short.ly/test-slug",
			mockError:      nil,
			expectedError:  false,
			expectedResult: "https://short.ly/test-slug",
			tokenExists:    true,
			mockCall:       true,
			metadataExists: true,
		},
		{
			name:           "Service returns error",
			userId:         1,
			slug:           "test-slug",
			targetURL:      "https://example.com",
			mockError:      errors.New("service error"),
			expectedError:  true,
			tokenExists:    true,
			mockCall:       true,
			metadataExists: true,
		},
		{
			name:           "Empty target url",
			userId:         1,
			slug:           "test-slug",
			targetURL:      "",
			mockCall:       false,
			expectedError:  true,
			tokenExists:    true,
			metadataExists: true,
		},
		{
			name:           "No token for authorization in metadata",
			userId:         1,
			slug:           "test-slug",
			targetURL:      "https://example.com",
			expectedError:  true,
			mockCall:       false,
			metadataExists: true,
		},
		{
			name:          "No metadata",
			userId:        1,
			slug:          "test-slug",
			targetURL:     "https://example.com",
			expectedError: true,
			mockCall:      false,
		},
		{
			name:           "Expired token",
			userId:         1,
			slug:           "test-slug",
			targetURL:      "https://example.com",
			expectedError:  true,
			mockCall:       false,
			metadataExists: true,
			tokenExists:    true,
			tokenExpired:   true,
		},
		{
			name:           "Expired token",
			userId:         0,
			slug:           "test-slug",
			targetURL:      "https://example.com",
			expectedError:  true,
			metadataExists: true,
			tokenExists:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockService := new(MockURLShortenerService)
			server := NewURLServer(mockService)

			req := &pb.CreateShortUrlRequest{
				Slug:      tt.slug,
				TargetUrl: tt.targetURL,
			}

			if tt.mockCall {
				mockService.On("GenerateShortURL", tt.userId, tt.targetURL, tt.slug).
					Return(tt.mockResult, tt.mockError).
					Once()
			}

			var token string
			if tt.tokenExists {
				token, _ = generateToken(tt.userId, tt.tokenExpired)
			}

			var ctx context.Context

			if tt.metadataExists {
				ctx = metadata.NewIncomingContext(
					context.Background(),
					metadata.Pairs("authorization", token),
				)
			} else {
				ctx = context.Background()
			}

			// Act
			resp, err := server.CreateShortURL(ctx, req)

			// Assert
			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Equal(t, tt.expectedResult, resp.ShortUrl)
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
				Slug:      "test-slug",
				ClientIP:  "0.0.0.0",
				Country:   "unknown",
				Language:  "UK",
				UserAgent: "Mozilla/5",
				Referer:   "referer.com",
			},
			mockResult:     "https://example.com",
			mockError:      nil,
			expectedError:  false,
			expectedResult: "https://example.com",
		},
		{
			name: "Service returns error",
			redirect: internal.Redirect{
				Slug:      "test-slug",
				ClientIP:  "0.0.0.0",
				Country:   "unknown",
				Language:  "UK",
				UserAgent: "Mozilla/5",
				Referer:   "referer.com",
			},
			mockResult:     "",
			mockError:      errors.New("error from service"),
			expectedError:  true,
			expectedResult: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockService := new(MockURLShortenerService)
			server := NewURLServer(mockService)

			req := &pb.ResolveSlugRequest{
				Redirect: &pb.URLServiceRedirect{
					Slug:      tt.redirect.Slug,
					ClientIp:  tt.redirect.ClientIP,
					Country:   tt.redirect.Country,
					Language:  tt.redirect.Language,
					UserAgent: tt.redirect.UserAgent,
					Referer:   tt.redirect.Referer,
				},
			}

			mockService.On("ResolveSlug", tt.redirect).
				Return(tt.mockResult, tt.mockError).
				Once()

			// Act
			resp, err := server.ResolveSlug(context.Background(), req)

			// Assert
			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Equal(t, tt.expectedResult, resp.TargetUrl)
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestURLServer_GetURL(t *testing.T) {
	tests := []struct {
		name           string
		urlId          int32
		userId         int
		mockResult     internal.ShortURL
		mockError      error
		expectedResult *pb.ShortURL
		expectedError  bool
		tokenExpired   bool
		tokenExists    bool
		mockCall       bool
		metadataExists bool
	}{
		{
			name:   "Success",
			urlId:  1,
			userId: 1,
			mockResult: internal.ShortURL{
				UrlId:     1,
				TargetUrl: "https://example.com",
				Slug:      "test-slug",
				CreatedAt: "2026-02-08 14:25:26",
				UserId:    1,
			},
			expectedResult: &pb.ShortURL{
				UrlId:     1,
				TargetUrl: "https://example.com",
				Slug:      "test-slug",
				CreatedAt: "2026-02-08 14:25:26",
			},
			metadataExists: true,
			tokenExists:    true,
			mockCall:       true,
		},
		{
			name:           "Error from service",
			urlId:          1,
			userId:         1,
			mockError:      errors.New("service error"),
			expectedResult: &pb.ShortURL{},
			expectedError:  true,
			metadataExists: true,
			tokenExists:    true,
			mockCall:       true,
		},
		{
			name:          "No metadata",
			urlId:         1,
			expectedError: true,
		},
		{
			name:           "No token in metadata",
			urlId:          1,
			expectedError:  true,
			metadataExists: true,
		},
		{
			name:           "Expired token",
			urlId:          1,
			userId:         1,
			expectedError:  true,
			metadataExists: true,
			tokenExists:    true,
			tokenExpired:   true,
		},
		{
			name:           "Invalid userId",
			urlId:          1,
			expectedError:  true,
			metadataExists: true,
			tokenExists:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockService := new(MockURLShortenerService)
			server := NewURLServer(mockService)

			req := &pb.GetURLRequest{
				UrlId: tt.urlId,
			}

			if tt.mockCall {
				mockService.On("GetURL", tt.userId, int(tt.urlId)).
					Return(tt.mockResult, tt.mockError).
					Once()
			}

			var token string
			if tt.tokenExists {
				token, _ = generateToken(tt.userId, tt.tokenExpired)
			}

			var ctx context.Context
			if tt.metadataExists {
				ctx = metadata.NewIncomingContext(
					context.Background(),
					metadata.Pairs("authorization", token))
			} else {
				ctx = context.Background()
			}

			// Action
			resp, err := server.GetURL(ctx, req)

			// Assert
			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Equal(t, tt.expectedResult, resp.ShortUrl)
			}

			mockService.AssertExpectations(t)
		})
	}
}
