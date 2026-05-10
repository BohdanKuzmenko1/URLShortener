package handler_test

import (
	"bytes"
	"encoding/json"
	pb "github.com/BohdanKuzmenko1/URLShortener/proto"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestURLShortenerService_ShortenURL(t *testing.T) {
	tests := []struct {
		name               string
		requestBody        interface{}
		mockResponse       *pb.CreateShortUrlResponse
		mockError          error
		expectedStatusCode int
		expectedResponse   map[string]interface{}
		tokenExpired       bool
		tokenExists        bool
	}{
		{
			name: "Success",
			requestBody: map[string]string{
				"target_url": "http://google.com",
			},
			mockResponse: &pb.CreateShortUrlResponse{
				ShortUrl: "https://short.com/abc123",
			},
			expectedStatusCode: http.StatusOK,
			expectedResponse: map[string]interface{}{
				"short_url": "https://short.com/abc123",
			},
			tokenExists: true,
		},
		{
			name: "URL without protocol",
			requestBody: map[string]string{
				"target_url": "google.com",
			},
			expectedStatusCode: http.StatusBadRequest,
			expectedResponse: map[string]interface{}{
				"error": "URL must start with http:// or https://",
			},
			tokenExists: true,
		},
		{
			name: "Without target_url field",
			requestBody: map[string]string{
				"slug": "my-slug",
			},
			expectedStatusCode: http.StatusBadRequest,
			tokenExists:        true,
		},
		{
			name: "Error from gRPC service",
			requestBody: map[string]string{
				"target_url": "http://google.com",
			},
			mockError:          assert.AnError,
			expectedStatusCode: http.StatusInternalServerError,
			expectedResponse: map[string]interface{}{
				"error": "something went wrong",
			},
			tokenExists: true,
		},
		{
			name: "Success with custom slug",
			requestBody: map[string]string{
				"target_url": "https://example.com",
				"slug":       "my-custom-slug",
			},
			mockResponse: &pb.CreateShortUrlResponse{
				ShortUrl: "https://short.com/my-custom-slug",
			},
			expectedStatusCode: http.StatusOK,
			expectedResponse: map[string]interface{}{
				"short_url": "https://short.com/my-custom-slug",
			},
			tokenExists: true,
		},
		{
			name: "Failure with no auth token",
			requestBody: map[string]string{
				"target_url": "https://example.com",
				"slug":       "my-custom-slug",
			},
			expectedStatusCode: http.StatusUnauthorized,
			tokenExists:        false,
		},
		{
			name: "Failure with expired token",
			requestBody: map[string]string{
				"target_url": "https://example.com",
				"slug":       "my-custom-slug",
			},
			expectedStatusCode: http.StatusUnauthorized,
			tokenExists:        true,
			tokenExpired:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			h, mockURLClient, _, _ := setupTestHandler()
			router := h.InitRoutes()
			gin.SetMode(gin.TestMode)

			// Check if we need to call client method
			if tt.mockResponse != nil || tt.mockError != nil {
				mockURLClient.On("CreateShortURL", mock.Anything, mock.MatchedBy(func(req *pb.CreateShortUrlRequest) bool {
					return true
				})).Return(tt.mockResponse, tt.mockError).Once()
			}

			jsonBody, _ := json.Marshal(tt.requestBody)

			// Act
			req, _ := http.NewRequest(http.MethodPost, "/api/url/shorten", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			var token string
			if tt.tokenExists {
				token, _ = generateToken(1, tt.tokenExpired)
			}
			req.Header.Set("Authorization", token)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Assert
			assert.Equal(t, tt.expectedStatusCode, w.Code)

			if tt.expectedResponse != nil {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)

				for key, expectedValue := range tt.expectedResponse {
					actualValue, exists := response[key]
					assert.True(t, exists, "Response should contain key %s", key)
					assert.Equal(t, expectedValue, actualValue, "Value for key %s doesn't match", key)
				}
			}

			if tt.mockResponse != nil || tt.mockError != nil {
				mockURLClient.AssertExpectations(t)
			}
		})
	}
}

func TestURLShortenerService_ResolveSlug(t *testing.T) {
	tests := []struct {
		name               string
		slug               string
		mockResponse       *pb.ResolveSlugResponse
		mockError          error
		expectedStatusCode int
		expectedLocation   string
	}{
		{
			name: "Success",
			slug: "abc123",
			mockResponse: &pb.ResolveSlugResponse{
				TargetUrl: "http://google.com",
			},
			expectedStatusCode: http.StatusFound,
			expectedLocation:   "http://google.com",
		},
		{
			name:               "Slug not found",
			slug:               "not-found",
			mockError:          assert.AnError,
			expectedStatusCode: http.StatusMovedPermanently,
			expectedLocation:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			h, mockURLClient, _, _ := setupTestHandler()
			router := h.InitRoutes()
			gin.SetMode(gin.TestMode)

			mockURLClient.On("ResolveSlug", mock.Anything, &pb.ResolveSlugRequest{
				Redirect: &pb.URLServiceRedirect{
					Slug:      tt.slug,
					UserAgent: "",
					ClientIp:  "",
					Country:   "unknown",
					Referer:   "",
					Language:  "",
				},
			}).Return(tt.mockResponse, tt.mockError).Once()

			// Act
			req, _ := http.NewRequest(http.MethodGet, "/"+tt.slug, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Assert
			assert.Equal(t, tt.expectedStatusCode, w.Code)

			if tt.expectedLocation != "" {
				assert.Equal(t, tt.expectedLocation, w.Header().Get("Location"))
			}

			mockURLClient.AssertExpectations(t)
		})
	}
}
