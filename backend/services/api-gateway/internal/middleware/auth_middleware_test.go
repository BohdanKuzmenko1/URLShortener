package middleware_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/BohdanKuzmenko1/URLShortener/services/api-gateway/internal/middleware"
	"github.com/BohdanKuzmenko1/URLShortener/shared"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/stretchr/testify/assert"
)

const testSigningKey = "test-secret-key"

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.AuthMiddleware([]byte(testSigningKey)))
	router.GET("/protected", func(c *gin.Context) {
		userId, _ := c.Get("userId")
		c.JSON(http.StatusOK, gin.H{"userId": userId})
	})
	return router
}

func generateToken(userId int, signingKey string, expiry time.Duration) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, &shared.TokenClaims{
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(expiry).Unix(),
			IssuedAt:  time.Now().Unix(),
		},
		UserId: userId,
	})
	signed, _ := token.SignedString([]byte(signingKey))
	return signed
}

func TestAuthMiddleware(t *testing.T) {
	t.Setenv("JWT_SIGNING_KEY", testSigningKey)

	router := setupRouter()

	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
		expectedUserId int
		expectedError  string
	}{
		{
			name:           "valid token",
			authHeader:     "Bearer " + generateToken(42, testSigningKey, time.Hour),
			expectedStatus: http.StatusOK,
			expectedUserId: 42,
		},
		{
			name:           "missing authorization header",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "authorization header missing",
		},
		{
			name:           "invalid format — no Bearer prefix",
			authHeader:     generateToken(1, testSigningKey, time.Hour),
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "invalid authorization header format",
		},
		{
			name:           "invalid format — wrong scheme",
			authHeader:     "Basic sometoken",
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "invalid authorization header format",
		},
		{
			name:           "expired token",
			authHeader:     "Bearer " + generateToken(1, testSigningKey, -time.Hour),
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "invalid or expired token",
		},
		{
			name:           "wrong signing key",
			authHeader:     "Bearer " + generateToken(1, "wrong-key", time.Hour),
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "invalid or expired token",
		},
		{
			name:           "invalid token string",
			authHeader:     "Bearer not.a.valid.token",
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "invalid or expired token",
		},
		{
			name:           "empty token after Bearer",
			authHeader:     "Bearer ",
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "invalid or expired token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				assert.Contains(t, w.Body.String(), tt.expectedError)
			}

			if tt.expectedUserId != 0 {
				assert.Contains(t, w.Body.String(), fmt.Sprintf("%d", tt.expectedUserId))
			}
		})
	}
}

func TestAuthMiddleware_SetsUserIdInContext(t *testing.T) {
	t.Setenv("JWT_SIGNING_KEY", testSigningKey)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.AuthMiddleware([]byte(testSigningKey)))

	var capturedUserId interface{}

	router.GET("/protected", func(c *gin.Context) {
		capturedUserId, _ = c.Get("userId")
		c.Status(http.StatusOK)
	})

	token := generateToken(99, testSigningKey, time.Hour)
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 99, capturedUserId)
}

func TestAuthMiddleware_BlocksWithoutToken(t *testing.T) {
	t.Setenv("JWT_SIGNING_KEY", testSigningKey)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.AuthMiddleware([]byte(testSigningKey)))

	handlerCalled := false
	router.GET("/protected", func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.False(t, handlerCalled, "handler should not be called without token")
}
