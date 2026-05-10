package handler

import (
	"context"
	pb "github.com/BohdanKuzmenko1/URLShortener/proto"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/metadata"
	"net/http"
	"strconv"
	"time"
)

// GetURLStats returns redirect statistics for a given URL on a specific date.
func (h *Handler) GetURLStats(c *gin.Context) {
	token := c.GetHeader("Authorization")

	date := c.Query("date")
	if date == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "date is required"})
		return
	}

	if _, err := time.Parse("2006-01-02", date); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "date must be YYYY-MM-DD"})
		return
	}

	urlIdStr := c.Query("id")
	if urlIdStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "url id is required"})
		return
	}

	urlId, err := strconv.Atoi(urlIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid url id"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("authorization", token))

	resp, err := h.statsServiceClient.GetURLStats(ctx, &pb.GetURLStatsRequest{
		UrlId: int32(urlId),
		Date:  date,
	})

	if err != nil {
		logrus.WithFields(logrus.Fields{
			"url_id": urlId,
			"date":   date,
			"error":  err.Error(),
		}).Error("failed to get url stats")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "something went wrong"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"redirects": resp.UrlStats})
}
