package adapter

import (
	"github.com/BohdanKuzmenko1/URLShortener/services/stats-service/internal"
	"github.com/BohdanKuzmenko1/URLShortener/services/stats-service/internal/broker"
	"github.com/BohdanKuzmenko1/URLShortener/services/stats-service/internal/detector"
	"time"
)

// EventToClick converts a RedirectEvent from Kafka into a Click domain model.
// It enriches the event with bot detection and device classification based on User-Agent.
// Country defaults to "XX" (unknown) until Cloudflare integration is implemented.
// CreatedAt unix timestamp is truncated to day for aggregated statistics storage.
func EventToClick(event broker.RedirectEvent) internal.Click {
	return internal.Click{
		URLId:   event.URLId,
		Country: "XX",
		IsBot:   detector.IsBot(event.UserAgent),
		Device:  detector.DetectDevice(event.UserAgent),
		Date:    time.Unix(event.CreatedAt, 0).UTC().Truncate(24 * time.Hour),
	}
}
