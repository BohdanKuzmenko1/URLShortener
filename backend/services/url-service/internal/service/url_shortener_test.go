package service_test

import (
	"context"
	"errors"
	"github.com/BohdanKuzmenko1/URLShortener/services/url-service/internal"
	"github.com/BohdanKuzmenko1/URLShortener/services/url-service/internal/broker"
	"github.com/BohdanKuzmenko1/URLShortener/services/url-service/internal/repository"
	"github.com/BohdanKuzmenko1/URLShortener/services/url-service/internal/service"
	"github.com/BohdanKuzmenko1/URLShortener/services/url-service/internal/storage"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"strings"
	"testing"
)

func TestUrlShortenerService_GetURL(t *testing.T) {
	tests := []struct {
		name         string
		userID       int
		urlID        int
		mockResponse internal.ShortURL
		mockError    error
	}{
		{
			name:   "Success",
			userID: 1,
			urlID:  1,
			mockResponse: internal.ShortURL{
				UrlId:     1,
				Slug:      "slug",
				TargetUrl: "https://google.com",
				UserId:    1,
				CreatedAt: "today",
			},
		},
		{
			name:         "Error from repository",
			userID:       1,
			urlID:        1,
			mockResponse: internal.ShortURL{},
			mockError:    errors.New("some error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			repoMock := new(URLShortenerRepositoryMock)
			lruStorageMock := new(LRUStorageMock)
			redisStorageMock := new(RedisStorageMock)
			brokerMock := new(RedirectProducerMock)
			testService := service.NewURLShortenerService(
				repoMock,
				brokerMock,
				redisStorageMock,
				lruStorageMock,
			)

			repoMock.On("GetURLByUserId", mock.Anything, tt.userID, tt.urlID).Return(tt.mockResponse, tt.mockError)

			// Action
			res, err := testService.GetURL(context.Background(), tt.userID, tt.urlID)

			// Assert
			if err != nil {
				assert.Error(t, err, tt.mockError)
				assert.Equal(t, internal.ShortURL{}, res)
			} else {
				assert.Equal(t, tt.mockResponse, res)
				assert.Nil(t, err)
			}

			repoMock.AssertExpectations(t)
		})
	}
}

func TestUrlShortenerService_GenerateShortURL(t *testing.T) {
	baseURL := "http://localhost:8090/"
	viper.Set("api-gateway.baseURL", baseURL)

	tests := []struct {
		name         string
		mockError    error
		mockResponse internal.ShortURL
		userID       int
		targetURL    string
		slug         string
		expectedURL  string
	}{
		{
			name:      "Success",
			userID:    1,
			targetURL: "https://google.com",
			slug:      "slug",
			mockResponse: internal.ShortURL{
				UrlId:     1,
				Slug:      "slug",
				TargetUrl: "https://google.com",
				UserId:    1,
				CreatedAt: "today",
			},
			expectedURL: baseURL + "slug",
		},
		{
			name:      "Success without slug specified",
			userID:    1,
			targetURL: "https://google.com",
			mockResponse: internal.ShortURL{
				UrlId:     1,
				Slug:      "slug",
				TargetUrl: "https://google.com",
				UserId:    1,
				CreatedAt: "today",
			},
		},
		{
			name:         "Error from repository",
			userID:       1,
			targetURL:    "https://google.com",
			slug:         "slug",
			mockResponse: internal.ShortURL{},
			mockError:    errors.New("some error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			repoMock := new(URLShortenerRepositoryMock)
			lruStorageMock := new(LRUStorageMock)
			redisStorageMock := new(RedisStorageMock)
			brokerMock := new(RedirectProducerMock)
			testService := service.NewURLShortenerService(
				repoMock,
				brokerMock,
				redisStorageMock,
				lruStorageMock,
			)

			if tt.slug == "" {
				repoMock.
					On("AddShortURL", mock.Anything, tt.userID, tt.targetURL, mock.AnythingOfType("string")).
					Return(tt.mockError)
			} else {
				repoMock.
					On("AddShortURL", mock.Anything, tt.userID, tt.targetURL, tt.slug).
					Return(tt.mockError)
			}

			// Action
			res, err := testService.GenerateShortURL(context.Background(), tt.userID, tt.targetURL, tt.slug)

			// Assert
			if err != nil {
				assert.Error(t, err, tt.mockError)
				assert.Equal(t, tt.expectedURL, res)
			} else {
				if tt.slug == "" {
					after, ok := strings.CutPrefix(res, viper.GetString("api-gateway.baseURL"))
					assert.True(t, ok)
					assert.Equal(t, 8, len(after))
					assert.NoError(t, err)
				} else {
					assert.Equal(t, tt.expectedURL, res)
					assert.NoError(t, err)
				}
			}

			repoMock.AssertExpectations(t)
		})
	}
}

func TestUrlShortenerService_ResolveSlug(t *testing.T) {
	tests := []struct {
		name                 string
		expectedErr          error
		repoMockErr          error
		lruMockErr           error
		redisMockErr         error
		urlID                int
		redirect             internal.Redirect
		expectedTargetURL    string
		shouldCallLRUSet     bool
		shouldCallRedisGet   bool
		shouldCallRedisSet   bool
		shouldCallRepoMock   bool
		shouldCallBrokerMock bool
	}{
		{
			name:                 "lru hit - returns from lru cache",
			urlID:                1,
			redirect:             internal.Redirect{Slug: "abc123", ClientIP: "127.0.0.1"},
			expectedTargetURL:    "https://example.com",
			lruMockErr:           nil,
			shouldCallBrokerMock: true,
		},
		{
			name:                 "redis hit - lru miss",
			urlID:                1,
			redirect:             internal.Redirect{Slug: "abc123"},
			expectedTargetURL:    "https://example.com",
			lruMockErr:           storage.ErrNotFound,
			redisMockErr:         nil,
			shouldCallRedisGet:   true,
			shouldCallBrokerMock: true,
		},
		{
			name:                 "db hit - lru and redis miss",
			urlID:                1,
			redirect:             internal.Redirect{Slug: "abc123"},
			expectedTargetURL:    "https://example.com",
			lruMockErr:           storage.ErrNotFound,
			redisMockErr:         storage.ErrNotFound,
			shouldCallRedisGet:   true,
			shouldCallRepoMock:   true,
			shouldCallLRUSet:     true,
			shouldCallRedisSet:   true,
			shouldCallBrokerMock: true,
		},
		{
			name:               "db not found - returns error",
			redirect:           internal.Redirect{Slug: "notexist"},
			lruMockErr:         storage.ErrNotFound,
			redisMockErr:       storage.ErrNotFound,
			repoMockErr:        repository.ErrNotFound,
			shouldCallRedisGet: true,
			shouldCallRepoMock: true,
			expectedErr:        repository.ErrNotFound,
		},
		{
			name:               "db error - returns error",
			redirect:           internal.Redirect{Slug: "abc123"},
			lruMockErr:         storage.ErrNotFound,
			redisMockErr:       storage.ErrNotFound,
			repoMockErr:        errors.New("db connection error"),
			shouldCallRedisGet: true,
			shouldCallRepoMock: true,
			expectedErr:        errors.New("db connection error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			repoMock := new(URLShortenerRepositoryMock)
			lruStorageMock := new(LRUStorageMock)
			redisStorageMock := new(RedisStorageMock)
			brokerMock := new(RedirectProducerMock)

			repoMock.Test(t)
			lruStorageMock.Test(t)
			redisStorageMock.Test(t)
			brokerMock.Test(t)

			testService := service.NewURLShortenerService(
				repoMock,
				brokerMock,
				redisStorageMock,
				lruStorageMock,
			)

			lruStorageMock.On("Get", mock.Anything, tt.redirect.Slug).
				Return(storage.CachedURL{ID: tt.urlID, Target: tt.expectedTargetURL}, tt.lruMockErr)

			if tt.shouldCallRedisGet {
				redisStorageMock.On("Get", mock.Anything, tt.redirect.Slug).
					Return(storage.CachedURL{ID: tt.urlID, Target: tt.expectedTargetURL}, tt.redisMockErr)
			}

			if tt.shouldCallRepoMock {
				repoMock.On("GetURLBySlug", mock.Anything, tt.redirect.Slug).
					Return(tt.urlID, tt.expectedTargetURL, tt.repoMockErr)
			}

			if tt.shouldCallLRUSet {
				lruStorageMock.On("Set", mock.Anything, tt.urlID, tt.expectedTargetURL, tt.redirect.Slug).
					Return()
			}

			if tt.shouldCallRedisSet {
				redisStorageMock.On("Set", mock.Anything, tt.urlID, tt.expectedTargetURL, tt.redirect.Slug).
					Return()
			}

			if tt.shouldCallBrokerMock {
				brokerMock.On("SendRedirect", mock.Anything, mock.MatchedBy(func(e broker.RedirectEvent) bool {
					return e.URLId == tt.urlID &&
						e.ClientIP == tt.redirect.ClientIP &&
						e.Referer == tt.redirect.Referer &&
						e.UserAgent == tt.redirect.UserAgent
				}), tt.redirect.Slug).Return(nil)
			}

			// Action
			res, err := testService.ResolveSlug(context.Background(), tt.redirect)

			// Assert
			if tt.expectedErr != nil {
				// Use ErrorIs only for sentinel errors (e.g. repository.ErrNotFound)
				// Use EqualError for dynamically created errors
				if errors.Is(tt.expectedErr, repository.ErrNotFound) || errors.Is(tt.expectedErr, storage.ErrNotFound) {
					assert.ErrorIs(t, err, tt.expectedErr)
				} else {
					assert.EqualError(t, err, tt.expectedErr.Error())
				}
				assert.Empty(t, res)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedTargetURL, res)
			}

			repoMock.AssertExpectations(t)
			lruStorageMock.AssertExpectations(t)
			redisStorageMock.AssertExpectations(t)
			brokerMock.AssertExpectations(t)
		})
	}
}
