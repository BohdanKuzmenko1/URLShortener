package middleware

import (
	"github.com/BohdanKuzmenko1/URLShortener/shared"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/sirupsen/logrus"
	"net/http"
	"os"
	"strings"
)

// AuthMiddleware validates the Bearer JWT token from the Authorization header.
// On success, it sets "userId" in the gin context for use by downstream handlers.
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header missing"})
			return
		}
		logrus.Println("Authorization header:", authHeader)

		parts := strings.SplitN(authHeader, " ", 2)
		logrus.Println("Authorization header parts:", parts)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid Authorization header format"})
			return
		}

		tokenStr := parts[1]

		token, err := jwt.ParseWithClaims(tokenStr, &shared.TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
			signingKey := os.Getenv("JWT_SIGNING_KEY")
			return []byte(signingKey), nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}

		claims, ok := token.Claims.(*shared.TokenClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			return
		}

		c.Set("userId", claims.UserId)

		c.Next()
	}
}
