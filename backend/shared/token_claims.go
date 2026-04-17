package shared

import (
	"errors"
	"github.com/golang-jwt/jwt"
	"strings"
)

type TokenClaims struct {
	jwt.StandardClaims
	UserId int `json:"user_id"`
}

func ValidateToken(authHeader, signingKey string) (*TokenClaims, error) {
	if authHeader == "" {
		return nil, errors.New("auth header missing")
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return nil, errors.New("invalid auth header format")
	}

	tokenStr := parts[1]

	token, err := jwt.ParseWithClaims(tokenStr, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(signingKey), nil
	})
	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(*TokenClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	return claims, nil
}
