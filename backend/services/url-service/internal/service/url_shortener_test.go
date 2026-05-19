package service_test

import (
	"errors"
	"github.com/BohdanKuzmenko1/URLShortener/services/url-service/internal"
	"github.com/BohdanKuzmenko1/URLShortener/services/url-service/internal/broker"
	_ "github.com/BohdanKuzmenko1/URLShortener/services/url-service/internal/repository"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
	"time"
	"unicode/utf8"
)

type MockURLShortenerRepository struct {
	mock.Mock
}

type MockRedirectProducer struct {
	mock.Mock
}

func (m *MockRedirectProducer) SendRedirect(event broker.RedirectEvent, slug string) error {
	args := m.Called(event, slug)
	return args.Error(0)
}

func (repo *MockURLShortenerRepository) AddShortURL(userId int, targetURL, slug string) error {
	args := repo.Called(userId, targetURL, slug)
	return args.Error(0)
}

func (repo *MockURLShortenerRepository) GetURLBySlug(slug string) (int, string, error) {
	args := repo.Called(slug)
	if args.Get(2) != nil {
		return 0, "", args.Error(2)
	}
	return args.Get(0).(int), args.Get(1).(string), args.Error(2)
}

func (repo *MockURLShortenerRepository) GetURLByUserId(userId, urlId int) (internal.ShortURL, error) {
	args := repo.Called(userId, urlId)
	if args.Get(1) != nil {
		return internal.ShortURL{}, args.Error(1)
	}

	return args.Get(0).(internal.ShortURL), args.Error(1)
}

func TestURLShortenerService_GetURLByUserId(t *testing.T) {
	tests := []struct {
		name             string
		userId           int
		urlId            int
		expectedError    error
		expectedResponse internal.ShortURL
	}{
		{
			name:          "Success",
			userId:        1,
			urlId:         1,
			expectedError: nil,
			expectedResponse: internal.ShortURL{
				UrlId:     1,
				Slug:      "slug",
				TargetUrl: "http://google.com",
				UserId:    1,
				CreatedAt: time.Now().Add(-24 * time.Hour).String(),
			},
		},
		{
			name:             "Error from repository",
			userId:           1,
			urlId:            1,
			expectedError:    errors.New("repository error"),
			expectedResponse: internal.ShortURL{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Arrange
			mockRepo := new(MockURLShortenerRepository)
			mockProducer := new(MockRedirectProducer)

			service := NewURLShortenerService(mockRepo, mockProducer)

			mockRepo.On("GetURLByUserId", test.userId, test.urlId).Return(test.expectedResponse, test.expectedError)

			// Action
			result, err := service.GetURL(test.userId, test.urlId)

			// Assert
			assert.Equal(t, test.expectedResponse, result)
			assert.Equal(t, test.expectedError, err)
		})
	}
}

func TestURLShortenerService_GenerateShortURL(t *testing.T) {
	viper.Set("api-gateway.baseURL", "http://localhost:8080/")

	tests := []struct {
		name             string
		userId           int
		targetURL        string
		slug             string
		expectedError    error
		expectedShortURL string
	}{
		{
			name:             "Success",
			slug:             "test",
			userId:           1,
			expectedError:    nil,
			targetURL:        "http://google.com",
			expectedShortURL: viper.GetString("api-gateway.baseURL") + "test",
		},
		{
			name:          "Error from repo",
			slug:          "test",
			userId:        1,
			expectedError: errors.New("repo error"),
			targetURL:     "http://google.com",
		},
		{
			name:      "Success with auto-generated slug",
			userId:    1,
			targetURL: "http://google.com",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Arrange
			mockRepo := &MockURLShortenerRepository{}
			mockProducer := &MockRedirectProducer{}
			service := NewURLShortenerService(mockRepo, mockProducer)

			mockRepo.On("AddShortURL", test.userId, test.targetURL, mock.Anything).Return(test.expectedError)

			// Action
			shortURL, err := service.GenerateShortURL(test.userId, test.targetURL, test.slug)

			// Assert
			if test.slug == "" {
				baseURLRunes := utf8.RuneCountInString(viper.GetString("api-gateway.baseURL"))
				shortURLRunes := utf8.RuneCountInString(shortURL)

				assert.Equal(t, baseURLRunes, shortURLRunes-8) // Checks if 8 runes slug was generated
			}

			if test.slug != "" {
				assert.Equal(t, test.expectedError, err)
				assert.Equal(t, test.expectedShortURL, shortURL)
			}
		})
	}
}

func TestUrlShortenerService_ResolveSlug(t *testing.T) {
	tests := []struct {
		name                  string
		redirect              internal.Redirect
		expectedTargetURL     string
		expectedURLId         int
		expectedRepoError     error
		expectedProducerError error
	}{
		{
			name: "Success",
			redirect: internal.Redirect{
				Slug:      "search",
				ClientIP:  "0.0.0.0",
				Country:   "US",
				Language:  "UK",
				UserAgent: "Mozilla/5.0",
				Referer:   "referer.com",
			},
			expectedTargetURL: "http://google.com",
			expectedURLId:     1,
		},
		{
			name: "Repository returns error",
			redirect: internal.Redirect{
				Slug:      "search",
				ClientIP:  "0.0.0.0",
				Country:   "US",
				Language:  "UK",
				UserAgent: "Mozilla/5.0",
				Referer:   "referer.com",
			},
			expectedTargetURL: "",
			expectedURLId:     0,
			expectedRepoError: errors.New("db error"),
		},
		{
			name: "Producer returns error",
			redirect: internal.Redirect{
				Slug:      "search",
				ClientIP:  "0.0.0.0",
				Country:   "US",
				Language:  "UK",
				UserAgent: "Mozilla/5.0",
				Referer:   "referer.com",
			},
			expectedTargetURL:     "http://google.com",
			expectedURLId:         1,
			expectedProducerError: errors.New("producer error"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockRepo := &MockURLShortenerRepository{}
			mockProducer := &MockRedirectProducer{}
			service := NewURLShortenerService(mockRepo, mockProducer)

			mockRepo.
				On("GetURLBySlug", test.redirect.Slug).
				Return(test.expectedURLId, test.expectedTargetURL, test.expectedRepoError)

			if test.expectedRepoError == nil {
				mockProducer.
					On("SendRedirect", mock.Anything, test.redirect.Slug).
					Return(test.expectedProducerError)
			}

			result, err := service.ResolveSlug(test.redirect)

			if test.expectedRepoError != nil {
				assert.Equal(t, test.expectedRepoError, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, test.expectedTargetURL, result)

			mockRepo.AssertExpectations(t)
			mockProducer.AssertExpectations(t)
		})
	}
}
