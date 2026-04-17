package handler

import (
	"context"
	pb "github.com/BohdanKuzmenko1/URLShortener/proto"
	"github.com/BohdanKuzmenko1/URLShortener/shared"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net/http"
	"time"
)

// request holds the credentials required for authentication endpoints.
type request struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=20"`
}

// Login authenticates a user with the provided email and password.
// On success, it sets an HttpOnly refresh token cookie and returns an access token in the response body.
// Returns 400 if the request body is invalid, or 401 if the credentials are incorrect.
func (h *Handler) Login(c *gin.Context) {
	var req request

	err := c.ShouldBindJSON(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	resp, err := h.authServiceClient.Login(ctx, &pb.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	tokenPair := &shared.TokenPair{
		AccessToken:  resp.Token.AccessToken,
		RefreshToken: resp.Token.RefreshToken,
		ExpiresIn:    int(resp.Token.ExpiresIn),
		TokenType:    resp.Token.TokenType,
	}

	c.SetCookie(
		"refresh_token",
		tokenPair.RefreshToken,
		int(7*24*time.Hour.Seconds()),
		"/api/auth/refresh-token",
		"",
		true,
		true,
	)

	c.JSON(http.StatusOK, gin.H{
		"access_token": tokenPair.AccessToken,
		"expires_in":   tokenPair.ExpiresIn,
		"token_type":   tokenPair.TokenType,
	})
}

// Register creates a new user account with the provided email and password.
// On success, it sets an HttpOnly refresh token cookie and returns an access token in the response body.
// Returns 400 if the request body is invalid, or 500 if the registration fails.
func (h *Handler) Register(c *gin.Context) {
	var req request

	err := c.ShouldBindJSON(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	resp, err := h.authServiceClient.Register(ctx, &pb.RegisterRequest{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	tokenPair := &shared.TokenPair{
		AccessToken:  resp.Token.AccessToken,
		RefreshToken: resp.Token.RefreshToken,
		ExpiresIn:    int(resp.Token.ExpiresIn),
		TokenType:    resp.Token.TokenType,
	}

	c.SetCookie(
		"refresh_token",
		tokenPair.RefreshToken,
		int(7*24*time.Hour.Seconds()),
		"/api/auth/refresh-token",
		"",
		true,
		true,
	)

	c.JSON(http.StatusCreated, gin.H{
		"access_token": tokenPair.AccessToken,
		"expires_in":   tokenPair.ExpiresIn,
		"token_type":   tokenPair.TokenType,
	})
}

// RefreshToken issues a new token pair using the refresh token stored in the cookie.
// On success, it rotates the refresh token cookie and returns a new access token in the response body.
// Returns 401 if the cookie is missing or the refresh token is invalid.
func (h *Handler) RefreshToken(c *gin.Context) {
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "refresh token required"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	resp, err := h.authServiceClient.RefreshToken(ctx, &pb.RefreshRequest{
		RefreshToken: refreshToken,
	})
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
		return
	}

	tokenPair := &shared.TokenPair{
		AccessToken:  resp.Token.AccessToken,
		RefreshToken: resp.Token.RefreshToken,
		ExpiresIn:    int(resp.Token.ExpiresIn),
		TokenType:    resp.Token.TokenType,
	}

	c.SetCookie(
		"refresh_token",
		tokenPair.RefreshToken,
		int(7*24*time.Hour.Seconds()),
		"/api/auth/refresh-token",
		"",
		true,
		true,
	)

	c.JSON(http.StatusOK, gin.H{
		"access_token": tokenPair.AccessToken,
		"expires_in":   tokenPair.ExpiresIn,
		"token_type":   tokenPair.TokenType,
	})
}

// Logout invalidates the user's refresh token and clears the refresh token cookie.
// If no refresh token cookie is present, the handler returns 200 without contacting the Auth Service.
// Errors from the Auth Service are logged but do not affect the response — the cookie is always cleared.
func (h *Handler) Logout(c *gin.Context) {
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "logged out"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	_, err = h.authServiceClient.Logout(ctx, &pb.LogoutRequest{
		RefreshToken: refreshToken,
	})

	if err != nil {
		logrus.Error(err)
	}

	c.SetCookie(
		"refresh_token",
		"",
		-1,
		"/api/auth/refresh-token",
		"",
		false, // need to change to true for prod env
		true,
	)

	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}
