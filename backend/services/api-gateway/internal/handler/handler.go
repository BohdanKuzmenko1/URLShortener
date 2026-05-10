// internal/handler/handler.go
package handler

import (
	pb "github.com/BohdanKuzmenko1/URLShortener/proto"
	"github.com/BohdanKuzmenko1/URLShortener/services/api-gateway/internal/middleware"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"net/http"
)

// Handler holds gRPC clients for each downstream microservice
// and serves as the dependency container for all HTTP route handlers.
type Handler struct {
	urlServiceClient   pb.URLServiceClient
	authServiceClient  pb.AuthServiceClient
	statsServiceClient pb.StatsServiceClient
}

// NewHandler creates a new Handler with the provided gRPC service clients.
func NewHandler(
	urlServiceClient pb.URLServiceClient,
	authServiceClient pb.AuthServiceClient,
	statsServiceClient pb.StatsServiceClient,
) *Handler {
	return &Handler{
		urlServiceClient:   urlServiceClient,
		authServiceClient:  authServiceClient,
		statsServiceClient: statsServiceClient,
	}
}

// InitRoutes registers all application routes and returns a configured gin.Engine.
//
// Route overview:
//
//	GET  /:slug                          — resolves a short URL slug and redirects to the original URL
//	GET  /                               — redirects to the frontend homepage
//
//	POST /api/url/shorten                — shortens a URL (requires authentication)
//
//	GET  /api/url-stats                  — retrieves URL statistics (requires authentication)
//	                                       query params: url_id, date (e.g. 2006-01-02)
//
//	POST   /api/auth/login               — authenticates a user and returns a token pair
//	POST   /api/auth/register            — creates a new user account and returns a token pair
//	POST   /api/auth/refresh-token       — rotates the token pair using the refresh token cookie
//	DELETE /api/auth/logout              — invalidates the refresh token and clears the cookie
func (h *Handler) InitRoutes() *gin.Engine {
	router := gin.New()

	router.GET("/:slug", h.ResolveSlug)
	router.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, viper.GetString("frontend.homepage"))
	})

	api := router.Group("/api")
	{
		url := api.Group("/url")
		url.Use(middleware.AuthMiddleware())
		{
			url.POST("/shorten", h.ShortenURL)
		}

		stats := api.Group("/url-stats")
		stats.Use(middleware.AuthMiddleware())
		{
			stats.GET("", h.GetURLStats) // /api/url-stats?id=123&date=2026-03-10
		}

		auth := api.Group("/auth")
		{
			auth.POST("/login", h.Login)
			auth.POST("/register", h.Register)
			auth.POST("/refresh-token", h.RefreshToken)
			auth.DELETE("/logout", h.Logout)
		}
	}

	return router
}
