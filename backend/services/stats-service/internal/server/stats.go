package server

import (
	"context"
	pb "github.com/BohdanKuzmenko1/URLShortener/proto"
	"github.com/BohdanKuzmenko1/URLShortener/services/stats-service/internal/mapper"
	"github.com/BohdanKuzmenko1/URLShortener/shared"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"os"
)

func (s *StatsServer) GetURLStats(ctx context.Context, r *pb.GetURLStatsRequest) (*pb.GetURLStatsResponse, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "no metadata")
	}

	authHeaders := md.Get("authorization")
	if len(authHeaders) == 0 {
		return nil, status.Error(codes.Unauthenticated, "no authorization header")
	}

	_, err := shared.ValidateToken(authHeaders[0], os.Getenv("JWT_SIGNING_KEY"))
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}

	urlStats, err := s.statsService.GetURLStats(ctx, authHeaders[0], r.UrlId, r.Date)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	protoURLStats := mapper.ToProtoURLStats(urlStats)

	return &pb.GetURLStatsResponse{
		UrlStats: protoURLStats,
	}, nil
}
