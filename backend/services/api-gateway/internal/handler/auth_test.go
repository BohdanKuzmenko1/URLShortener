package handler

import (
	"bytes"
	"context"
	"encoding/json"
	pb "github.com/BohdanKuzmenko1/URLShortener/proto"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

type MockAuthServiceClient struct {
	mock.Mock
}

func (m *MockAuthServiceClient) Register(ctx context.Context, in *pb.RegisterRequest, opts ...grpc.CallOption) (*pb.RegisterResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb.RegisterResponse), args.Error(1)
}

func (m *MockAuthServiceClient) Login(ctx context.Context, in *pb.LoginRequest, opts ...grpc.CallOption) (*pb.LoginResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb.LoginResponse), args.Error(1)
}

func (m *MockAuthServiceClient) RefreshToken(ctx context.Context, in *pb.RefreshRequest, opts ...grpc.CallOption) (*pb.RefreshResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb.RefreshResponse), args.Error(1)
}

func (m *MockAuthServiceClient) Logout(ctx context.Context, in *pb.LogoutRequest, opts ...grpc.CallOption) (*pb.LogoutResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb.LogoutResponse), args.Error(1)
}

func setupAuthTestHandler() (*Handler, *MockAuthServiceClient) {
	mockURLClient := new(MockUrlServiceClient)
	mockAuthClient := new(MockAuthServiceClient)
	mockStatsClient := new(MockStatsServiceClient)
	handler := NewHandler(mockURLClient, mockAuthClient, mockStatsClient)
	return handler, mockAuthClient
}

func parseCookies(response *http.Response) map[string]*http.Cookie {
	cookies := make(map[string]*http.Cookie)
	for _, cookie := range response.Cookies() {
		cookies[cookie.Name] = cookie
	}
	return cookies
}

func TestAuthService_Register(t *testing.T) {
	tests := []struct {
		name                string
		requestBody         interface{}
		mockResponse        *pb.RegisterResponse
		mockError           error
		expectedStatusCode  int
		expectedResponse    map[string]interface{}
		expectedCookieName  string
		expectedCookieValue string
		shouldCheckCookies  bool
	}{
		{
			name: "Success",
			requestBody: map[string]string{
				"email":    "testuser@urlshortener.com",
				"password": "password123",
			},
			mockResponse: &pb.RegisterResponse{
				Token: &pb.TokenPair{
					AccessToken:  "access_token",
					RefreshToken: "refresh_token",
					ExpiresIn:    12345,
					TokenType:    "Bearer",
				},
			},
			expectedStatusCode: http.StatusCreated,
			expectedResponse: map[string]interface{}{
				"access_token": "access_token",
				"expires_in":   float64(12345),
				"token_type":   "Bearer",
			},
			shouldCheckCookies:  true,
			expectedCookieValue: "refresh_token",
			expectedCookieName:  "refresh_token",
		},
		{
			name: "Without email",
			requestBody: map[string]string{
				"password": "password123",
			},
			expectedStatusCode: http.StatusBadRequest,
			expectedResponse: map[string]interface{}{
				"error": "Key: 'request.Email' Error:Field validation for 'Email' failed on the 'required' tag",
			},
		},
		{
			name: "Without password",
			requestBody: map[string]string{
				"email": "testuser@urlshortener.com",
			},
			expectedStatusCode: http.StatusBadRequest,
			expectedResponse: map[string]interface{}{
				"error": "Key: 'request.Password' Error:Field validation for 'Password' failed on the 'required' tag",
			},
		},
		{
			name: "Incorrect email",
			requestBody: map[string]string{
				"email":    "123456",
				"password": "password123",
			},
			expectedStatusCode: http.StatusBadRequest,
			expectedResponse: map[string]interface{}{
				"error": "Key: 'request.Email' Error:Field validation for 'Email' failed on the 'email' tag",
			},
		},
		{
			name: "Too short password",
			requestBody: map[string]string{
				"email":    "testuser@urlshortener.com",
				"password": "1234567",
			},
			expectedStatusCode: http.StatusBadRequest,
			expectedResponse: map[string]interface{}{
				"error": "Key: 'request.Password' Error:Field validation for 'Password' failed on the 'min' tag",
			},
		},
		{
			name: "Too long password",
			requestBody: map[string]string{
				"email":    "testuser@urlshortener.com",
				"password": "1234567890qwertyuiopq",
			},
			expectedStatusCode: http.StatusBadRequest,
			expectedResponse: map[string]interface{}{
				"error": "Key: 'request.Password' Error:Field validation for 'Password' failed on the 'max' tag",
			},
		},
		{
			name: "Error from client",
			requestBody: map[string]string{
				"email":    "testuser@urlshortener.com",
				"password": "1234567890",
			},
			mockError:          assert.AnError,
			expectedStatusCode: http.StatusInternalServerError,
			expectedResponse: map[string]interface{}{
				"error": "assert.AnError general error for testing",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			h, mockAuthClient := setupAuthTestHandler()
			router := h.InitRoutes()
			gin.SetMode(gin.TestMode)

			if tt.mockResponse != nil || tt.mockError != nil {
				mockAuthClient.On("Register", mock.Anything, mock.MatchedBy(func(req *pb.RegisterRequest) bool {
					return true
				})).Return(tt.mockResponse, tt.mockError).Once()
			}

			jsonBody, _ := json.Marshal(tt.requestBody)

			// Act
			req, _ := http.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			resp := w.Result()
			defer func(Body io.ReadCloser) {
				err := Body.Close()
				if err != nil {
					t.Errorf("error closing response body")
				}
			}(resp.Body)

			// Assert
			assert.Equal(t, tt.expectedStatusCode, w.Code)

			if tt.shouldCheckCookies {
				cookies := parseCookies(resp)
				cookie, exists := cookies[tt.expectedCookieName]
				assert.True(t, exists, "Cookie %s must be set", tt.expectedCookieName)
				assert.Equal(t, tt.expectedCookieValue, cookie.Value, "Cookie value doesn't match")
				assert.Equal(t, "/api/auth/refresh-token", cookie.Path, "Cookie path doesn't match")
				assert.True(t, cookie.HttpOnly, "Cookie must be HttpOnly")
				assert.True(t, cookie.Secure, "Cookie must be Secure")
				assert.Greater(t, cookie.MaxAge, 0, "Cookie MaxAge must be positive integer")
			}

			if tt.expectedResponse != nil {
				body, err := io.ReadAll(resp.Body)
				assert.NoError(t, err)

				var response map[string]interface{}
				err = json.Unmarshal(body, &response)
				assert.NoError(t, err)

				for key, expectedValue := range tt.expectedResponse {
					actualValue, exists := response[key]
					assert.True(t, exists, "Response should contain key %s", key)
					assert.Equal(t, expectedValue, actualValue, "Value for key %s doesn't match", key)
				}
			}

			if tt.mockResponse != nil || tt.mockError != nil {
				mockAuthClient.AssertExpectations(t)
			}
		})
	}
}

func TestAuthService_Login(t *testing.T) {
	tests := []struct {
		name                string
		requestBody         interface{}
		expectedStatusCode  int
		expectedResponse    map[string]interface{}
		mockResponse        *pb.LoginResponse
		mockError           error
		expectedCookieName  string
		expectedCookieValue string
		shouldCheckCookies  bool
	}{
		{
			name: "Success",
			requestBody: map[string]string{
				"email":    "testuser@urlshortener.com",
				"password": "password123",
			},
			expectedStatusCode: http.StatusOK,
			expectedResponse: map[string]interface{}{
				"access_token": "access_token",
				"expires_in":   float64(12345),
				"token_type":   "Bearer",
			},
			mockResponse: &pb.LoginResponse{
				Token: &pb.TokenPair{
					AccessToken:  "access_token",
					RefreshToken: "refresh_token",
					ExpiresIn:    12345,
					TokenType:    "Bearer",
				},
			},
			shouldCheckCookies:  true,
			expectedCookieValue: "refresh_token",
			expectedCookieName:  "refresh_token",
		},
		{
			name: "Without email",
			requestBody: map[string]string{
				"password": "password123",
			},
			expectedStatusCode: http.StatusBadRequest,
			expectedResponse: map[string]interface{}{
				"error": "Key: 'request.Email' Error:Field validation for 'Email' failed on the 'required' tag",
			},
		},
		{
			name: "Without password",
			requestBody: map[string]string{
				"email": "testuser@urlshortener.com",
			},
			expectedStatusCode: http.StatusBadRequest,
			expectedResponse: map[string]interface{}{
				"error": "Key: 'request.Password' Error:Field validation for 'Password' failed on the 'required' tag",
			},
		},
		{
			name: "Incorrect email",
			requestBody: map[string]string{
				"email":    "123456",
				"password": "password123",
			},
			expectedStatusCode: http.StatusBadRequest,
			expectedResponse: map[string]interface{}{
				"error": "Key: 'request.Email' Error:Field validation for 'Email' failed on the 'email' tag",
			},
		},
		{
			name: "Too short password",
			requestBody: map[string]string{
				"email":    "testuser@urlshortener.com",
				"password": "1234567",
			},
			expectedStatusCode: http.StatusBadRequest,
			expectedResponse: map[string]interface{}{
				"error": "Key: 'request.Password' Error:Field validation for 'Password' failed on the 'min' tag",
			},
		},
		{
			name: "Too long password",
			requestBody: map[string]string{
				"email":    "testuser@urlshortener.com",
				"password": "1234567890qwertyuiopq",
			},
			expectedStatusCode: http.StatusBadRequest,
			expectedResponse: map[string]interface{}{
				"error": "Key: 'request.Password' Error:Field validation for 'Password' failed on the 'max' tag",
			},
		},
		{
			name: "Error from client",
			requestBody: map[string]string{
				"email":    "testuser@urlshortener.com",
				"password": "1234567890",
			},
			mockError:          assert.AnError,
			expectedStatusCode: http.StatusUnauthorized,
			expectedResponse: map[string]interface{}{
				"error": "invalid credentials",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mockAuthClient := setupAuthTestHandler()
			router := h.InitRoutes()
			gin.SetMode(gin.TestMode)

			if tt.mockResponse != nil || tt.mockError != nil {
				mockAuthClient.On("Login", mock.Anything, mock.MatchedBy(func(req *pb.LoginRequest) bool {
					return true
				})).Return(tt.mockResponse, tt.mockError).Once()
			}

			jsonBody, _ := json.Marshal(tt.requestBody)

			req, _ := http.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			resp := w.Result()
			defer func(Body io.ReadCloser) {
				err := Body.Close()
				if err != nil {
					t.Errorf("error closing response body")
				}
			}(resp.Body)

			assert.Equal(t, tt.expectedStatusCode, w.Code)

			if tt.shouldCheckCookies {
				actualCookies := parseCookies(resp)
				actualCookie, exists := actualCookies[tt.expectedCookieName]
				assert.True(t, exists, "Cookie %s must be set", tt.expectedCookieName)
				assert.Equal(t, tt.expectedCookieValue, actualCookie.Value, "Cookie value doesn't match")
				assert.Equal(t, "/api/auth/refresh-token", actualCookie.Path, "Cookie path doesn't match")
				assert.True(t, actualCookie.HttpOnly, "Cookie must be HttpOnly")
				assert.True(t, actualCookie.Secure, "Cookie must be Secure")
				assert.Greater(t, actualCookie.MaxAge, 0, "Cookie MaxAge must be positive integer")
			}

			if tt.expectedResponse != nil {
				body, err := io.ReadAll(resp.Body)
				assert.NoError(t, err)

				var response map[string]interface{}
				err = json.Unmarshal(body, &response)
				assert.NoError(t, err)

				for key, expectedValue := range tt.expectedResponse {
					actualValue, exists := response[key]
					assert.True(t, exists, "Response should contain key %s", key)
					assert.Equal(t, expectedValue, actualValue, "Value for key %s doesn't match", key)
				}
			}

			if tt.mockResponse != nil || tt.mockError != nil {
				mockAuthClient.AssertExpectations(t)
			}
		})
	}
}

func TestAuthService_RefreshToken(t *testing.T) {
	tests := []struct {
		name               string
		expectedResponse   map[string]interface{}
		mockResponse       *pb.RefreshResponse
		mockError          error
		expectedStatusCode int
		cookieBefore       *http.Cookie
		expectedCookie     *http.Cookie
		shouldCheckCookies bool
	}{
		{
			name: "Success",
			mockResponse: &pb.RefreshResponse{
				Token: &pb.TokenPair{
					AccessToken:  "access_token",
					RefreshToken: "refresh_token_after",
					ExpiresIn:    12345,
					TokenType:    "Bearer",
				},
			},
			expectedResponse: map[string]interface{}{
				"access_token": "access_token",
				"expires_in":   float64(12345),
				"token_type":   "Bearer",
			},
			expectedStatusCode: http.StatusOK,
			cookieBefore: &http.Cookie{
				Name:     "refresh_token",
				Value:    "refresh_token_before",
				Path:     "/api/auth/refresh-token",
				HttpOnly: true,
				Secure:   true,
				MaxAge:   12345,
			},
			expectedCookie: &http.Cookie{
				Name:     "refresh_token",
				Value:    "refresh_token_after",
				Path:     "/api/auth/refresh-token",
				HttpOnly: true,
				Secure:   true,
				MaxAge:   12345,
			},
			shouldCheckCookies: true,
		},
		{
			name: "No refresh_token cookie in request",
			expectedResponse: map[string]interface{}{
				"error": "refresh token required",
			},
			expectedStatusCode: http.StatusUnauthorized,
			cookieBefore: &http.Cookie{
				Name:     "no_required_cookie",
				Value:    "",
				Path:     "",
				HttpOnly: true,
				Secure:   true,
				MaxAge:   12345,
			},
			shouldCheckCookies: true,
		},
		{
			name: "Error from client",
			expectedResponse: map[string]interface{}{
				"error": "invalid refresh token",
			},
			mockError:          assert.AnError,
			expectedStatusCode: http.StatusUnauthorized,
			cookieBefore: &http.Cookie{
				Name:     "refresh_token",
				Value:    "refresh_token_before",
				Path:     "/api/auth/refresh-token",
				HttpOnly: true,
				Secure:   true,
				MaxAge:   12345,
			},
			shouldCheckCookies: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mockAuthClient := setupAuthTestHandler()
			router := h.InitRoutes()
			gin.SetMode(gin.TestMode)

			if tt.mockResponse != nil || tt.mockError != nil {
				mockAuthClient.On("RefreshToken", mock.Anything, mock.MatchedBy(func(req *pb.RefreshRequest) bool {
					return req.RefreshToken == tt.cookieBefore.Value
				})).Return(tt.mockResponse, tt.mockError).Once()
			}

			req, _ := http.NewRequest(http.MethodPost, "/api/auth/refresh-token", nil)
			req.Header.Set("Content-Type", "application/json")
			if tt.cookieBefore != nil {
				req.AddCookie(tt.cookieBefore)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			resp := w.Result()
			defer func(Body io.ReadCloser) {
				err := Body.Close()
				if err != nil {
					t.Errorf("error closing response body")
				}
			}(resp.Body)

			assert.Equal(t, tt.expectedStatusCode, w.Code)

			if tt.shouldCheckCookies && tt.expectedCookie != nil {
				actualCookies := parseCookies(resp)
				actualCookie, exists := actualCookies[tt.expectedCookie.Name]
				assert.True(t, exists, "Cookie %s must be set", tt.expectedCookie.Name)
				assert.Equal(t, tt.expectedCookie.Value, actualCookie.Value, "Cookie value doesn't match")
				assert.Equal(t, tt.expectedCookie.Path, actualCookie.Path, "Cookie path doesn't match")
				assert.True(t, actualCookie.HttpOnly, "Cookie must be HttpOnly")
				assert.True(t, actualCookie.Secure, "Cookie must be Secure")
				assert.Greater(t, actualCookie.MaxAge, 0, "Cookie MaxAge must be positive integer")
			}

			if tt.expectedCookie == nil {
				actualCookies := parseCookies(resp)
				_, exists := actualCookies["refresh_token"]
				assert.False(t, exists, "Cookie %s must not be set", "refresh_token")
			}

			if tt.expectedResponse != nil {
				body, err := io.ReadAll(resp.Body)
				assert.NoError(t, err)

				var response map[string]interface{}
				err = json.Unmarshal(body, &response)
				assert.NoError(t, err)

				for key, expectedValue := range tt.expectedResponse {
					actualValue, exists := response[key]
					assert.True(t, exists, "Response should contain key %s", key)
					assert.Equal(t, expectedValue, actualValue, "Value for key %s doesn't match", key)
				}
			}

			if tt.mockResponse != nil || tt.mockError != nil {
				mockAuthClient.AssertExpectations(t)
			}
		})
	}
}

func TestAuthService_Logout(t *testing.T) {
	tests := []struct {
		name               string
		expectedResponse   map[string]interface{}
		mockResponse       *pb.LogoutResponse
		mockError          error
		expectedStatusCode int
		cookieBefore       *http.Cookie
	}{
		{
			name: "Success with refresh token",
			expectedResponse: map[string]interface{}{
				"message": "logged out",
			},
			mockResponse:       &pb.LogoutResponse{Success: true},
			expectedStatusCode: http.StatusOK,
			cookieBefore: &http.Cookie{
				Name:     "refresh_token",
				Value:    "refresh_token_before",
				Path:     "/api/auth/refresh-token",
				HttpOnly: true,
				Secure:   true,
				MaxAge:   12345,
			},
		},
		{
			name: "Error from client",
			expectedResponse: map[string]interface{}{
				"message": "logged out",
			},
			mockError:          assert.AnError,
			expectedStatusCode: http.StatusOK,
			cookieBefore: &http.Cookie{
				Name:     "refresh_token",
				Value:    "refresh_token_before",
				Path:     "/api/auth/refresh-token",
				HttpOnly: true,
				Secure:   true,
				MaxAge:   12345,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mockAuthClient := setupAuthTestHandler()
			router := h.InitRoutes()

			if tt.cookieBefore != nil && tt.cookieBefore.Name == "refresh_token" {
				mockAuthClient.On("Logout", mock.Anything, mock.MatchedBy(func(req *pb.LogoutRequest) bool {
					return req.RefreshToken == tt.cookieBefore.Value
				})).Return(tt.mockResponse, tt.mockError).Once()
			}

			req, _ := http.NewRequest(http.MethodDelete, "/api/auth/logout", nil)
			req.Header.Set("Content-Type", "application/json")

			if tt.cookieBefore != nil && tt.cookieBefore.Name == "refresh_token" {
				req.AddCookie(tt.cookieBefore)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			resp := w.Result()
			defer func(Body io.ReadCloser) {
				err := Body.Close()
				if err != nil {
					t.Errorf("error closing response body")
				}
			}(resp.Body)

			assert.Equal(t, tt.expectedStatusCode, resp.StatusCode)

			var responseBody map[string]interface{}
			err := json.NewDecoder(resp.Body).Decode(&responseBody)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedResponse, responseBody)

			cookies := resp.Cookies()
			found := false
			for _, cookie := range cookies {
				if cookie.Name == "refresh_token" {
					found = true
					assert.Equal(t, -1, cookie.MaxAge, "Cookie should have MaxAge = -1 for deletion")
					assert.Equal(t, "", cookie.Value, "Cookie value should be empty")
					assert.Equal(t, "/api/auth/refresh-token", cookie.Path, "Cookie path should match")
					break
				}
			}

			assert.True(t, found, "Cookie refresh_token should be present in response with deletion instructions")
		})
	}
}
