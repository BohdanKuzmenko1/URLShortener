package adapter

import (
	"github.com/BohdanKuzmenko1/URLShortener/services/stats-service/internal"
	"github.com/BohdanKuzmenko1/URLShortener/services/stats-service/internal/broker"
	detector2 "github.com/BohdanKuzmenko1/URLShortener/services/stats-service/internal/detector"
	"time"
)

func EventToClick(event broker.RedirectEvent) internal.Click {
	return internal.Click{
		URLId:   event.URLId,
		Country: "XX",
		IsBot:   detector2.IsBot(event.UserAgent),
		Device:  detector2.DetectDevice(event.UserAgent),
		Date:    time.Unix(event.CreatedAt, 0).UTC().Truncate(24 * time.Hour),
	}
}
