package server

import (
	pb "github.com/BohdanKuzmenko1/URLShortener/proto"
	"github.com/BohdanKuzmenko1/URLShortener/services/stats-service/internal/service"
)

type StatsServer struct {
	pb.UnimplementedStatsServiceServer
	statsService service.StatsService
}

func NewStatsServer(statsService service.StatsService) *StatsServer {
	return &StatsServer{statsService: statsService}
}
