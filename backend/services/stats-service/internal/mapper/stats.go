package mapper

import (
	pb "github.com/BohdanKuzmenko1/URLShortener/proto"
	"github.com/BohdanKuzmenko1/URLShortener/services/stats-service/internal"
)

func ToProtoURLStats(stats []internal.URLStats) []*pb.URLStats {
	result := make([]*pb.URLStats, 0, len(stats))
	for _, v := range stats {
		result = append(result, &pb.URLStats{
			UrlId:     v.URLId,
			Country:   v.Country,
			Device:    v.Device,
			Date:      v.Date,
			Clicks:    v.Clicks,
			BotClicks: v.BotClicks,
		})
	}
	return result
}
