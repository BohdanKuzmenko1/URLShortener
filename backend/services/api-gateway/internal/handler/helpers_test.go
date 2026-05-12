package handler_test

import (
	"fmt"
	"github.com/BohdanKuzmenko1/URLShortener/services/api-gateway/internal/handler"
	"github.com/BohdanKuzmenko1/URLShortener/shared"
	"github.com/golang-jwt/jwt"
	"net/http"
	"os"
	"testing"
	"time"
)

const testSigningKey = "test-secret-key"

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func TestMain(m *testing.M) {
	err := os.Setenv("JWT_SIGNING_KEY", testSigningKey)
	if err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

func parseCookies(response *http.Response) map[string]*http.Cookie {
	cookies := make(map[string]*http.Cookie)
	for _, cookie := range response.Cookies() {
		cookies[cookie.Name] = cookie
	}
	return cookies
}

func generateToken(userId int, isExpired bool) (string, error) {
	expiry := 24 * time.Hour
	if isExpired {
		expiry = -24 * time.Hour
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, &shared.TokenClaims{
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(expiry).Unix(),
			IssuedAt:  time.Now().Unix(),
		},
		UserId: userId,
	})

	signedToken, err := token.SignedString([]byte(testSigningKey))
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Bearer %s", signedToken), nil
}

func setupTestHandler() (*handler.Handler, *MockUrlServiceClient, *MockStatsServiceClient, *MockAuthServiceClient) {
	mockURLClient := new(MockUrlServiceClient)
	mockAuthClient := new(MockAuthServiceClient)
	mockStatsClient := new(MockStatsServiceClient)
	h := handler.NewHandler(mockURLClient, mockAuthClient, mockStatsClient)
	return h, mockURLClient, mockStatsClient, mockAuthClient
}
