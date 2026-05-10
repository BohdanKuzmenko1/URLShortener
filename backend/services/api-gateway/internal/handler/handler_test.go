package handler_test

import (
	"github.com/BohdanKuzmenko1/URLShortener/services/api-gateway/internal/handler"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestInitRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockURLClient := new(MockUrlServiceClient)
	mockAuthClient := new(MockAuthServiceClient)
	mockStatsClient := new(MockStatsServiceClient)

	h := handler.NewHandler(mockURLClient, mockAuthClient, mockStatsClient)
	router := h.InitRoutes()

	routes := router.Routes()

	registered := map[string]bool{}
	for _, r := range routes {
		registered[r.Method+":"+r.Path] = true
	}

	assert.True(t, registered["GET:/:slug"])
	assert.True(t, registered["GET:/"])
	assert.True(t, registered["POST:/api/url/shorten"])
	assert.True(t, registered["GET:/api/url-stats"])
	assert.True(t, registered["POST:/api/auth/login"])
	assert.True(t, registered["POST:/api/auth/register"])
	assert.True(t, registered["POST:/api/auth/refresh-token"])
	assert.True(t, registered["DELETE:/api/auth/logout"])
}
