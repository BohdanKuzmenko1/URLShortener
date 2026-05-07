package handler

import (
	"context"
	"encoding/json"
	"fmt"
	pb "github.com/BohdanKuzmenko1/URLShortener/proto"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	"net/http"
	"net/http/httptest"
	"testing"
)

type MockStatsServiceClient struct {
	mock.Mock
}

func (m *MockStatsServiceClient) GetURLStats(ctx context.Context, in *pb.GetURLStatsRequest, opts ...grpc.CallOption) (*pb.GetURLStatsResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb.GetURLStatsResponse), args.Error(1)
}

func setupStatsTestHandler() (*Handler, *MockStatsServiceClient) {
	mockURLClient := new(MockUrlServiceClient)
	mockAuthClient := new(MockAuthServiceClient)
	mockStatsClient := new(MockStatsServiceClient)
	handler := NewHandler(mockURLClient, mockAuthClient, mockStatsClient)
	return handler, mockStatsClient
}

func TestStatsService_GetURLStats(t *testing.T) {
	tests := []struct {
		name               string
		queryParams        []string
		mockRequest        *pb.GetURLStatsRequest
		mockResponse       *pb.GetURLStatsResponse
		mockError          error
		expectedStatusCode int
		expectedResponse   map[string]interface{}
		tokenExpired       bool
		tokenExists        bool
	}{
		{
			name:        "Success",
			queryParams: []string{"?id=1&", "date=2026-04-28"},
			mockRequest: &pb.GetURLStatsRequest{
				UrlId: 1,
				Date:  "2026.04.28",
			},
			mockResponse: &pb.GetURLStatsResponse{
				UrlStats: []*pb.URLStats{
					&pb.URLStats{
						UrlId:   1,
						Date:    "2026-04-28",
						Device:  "desktop",
						Country: "XX",
						Clicks:  10,
					},
				},
			},
			expectedStatusCode: http.StatusOK,
			expectedResponse: map[string]interface{}{
				"redirects": []interface{}{
					map[string]interface{}{
						"url_id":  float64(1),
						"date":    "2026-04-28",
						"device":  "desktop",
						"country": "XX",
						"clicks":  float64(10),
					},
				},
			},
			tokenExists: true,
		},
		{
			name:        "Error from gRPC service",
			queryParams: []string{"?id=1&", "date=2026-04-28"},
			mockRequest: &pb.GetURLStatsRequest{
				UrlId: 1,
				Date:  "2026.04.28",
			},
			mockError:          assert.AnError,
			expectedStatusCode: http.StatusInternalServerError,
			expectedResponse: map[string]interface{}{
				"error": "Something went wrong",
			},
			tokenExists: true,
		},
		{
			name:               "Without url_id parameter",
			queryParams:        []string{"", "?date=2026-04-28"},
			expectedStatusCode: http.StatusBadRequest,
			expectedResponse: map[string]interface{}{
				"error": "url id is required",
			},
			tokenExists: true,
		},
		{
			name:               "Without date parameter",
			queryParams:        []string{"?id=1&", ""},
			expectedStatusCode: http.StatusBadRequest,
			expectedResponse: map[string]interface{}{
				"error": "date is required",
			},
			tokenExists: true,
		},
		{
			name:               "Token expired",
			queryParams:        []string{"?id=1&", "date=2026-04-28"},
			expectedStatusCode: http.StatusUnauthorized,
			tokenExists:        true,
			tokenExpired:       true,
		},
		{
			name:               "Token does not exist",
			queryParams:        []string{"?id=1&", "date=2026-04-28"},
			expectedStatusCode: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			h, mockStatsClient := setupStatsTestHandler()
			router := h.InitRoutes()
			gin.SetMode(gin.TestMode)

			// Check if we need to call client method
			if tt.mockResponse != nil || tt.mockError != nil {
				mockStatsClient.On("GetURLStats", mock.Anything, mock.MatchedBy(func(req *pb.GetURLStatsRequest) bool {
					return true
				})).Return(tt.mockResponse, tt.mockError).Once()
			}

			url := fmt.Sprintf("/api/url-stats%s%s", tt.queryParams[0], tt.queryParams[1])

			// Act
			req, _ := http.NewRequest(http.MethodGet, url, nil)

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
				mockStatsClient.AssertExpectations(t)
			}
		})
	}
}
