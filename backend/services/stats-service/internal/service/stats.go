package service

import (
	"context"
	pb "github.com/BohdanKuzmenko1/URLShortener/proto"
	internal2 "github.com/BohdanKuzmenko1/URLShortener/services/stats-service/internal"
	"github.com/BohdanKuzmenko1/URLShortener/services/stats-service/internal/adapter"
	"github.com/BohdanKuzmenko1/URLShortener/services/stats-service/internal/broker"
	"github.com/BohdanKuzmenko1/URLShortener/services/stats-service/internal/repository"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/metadata"
)

type StatsService interface {
	RecordClick(ctx context.Context, event broker.RedirectEvent) error
	RecordClickBatch(ctx context.Context, events []broker.RedirectEvent) error
	GetURLStats(ctx context.Context, token string, urlID int32, date string) ([]internal2.URLStats, error)
	CheckURLOwnership(ctx context.Context, token string, urlID int32) error
}

type statsService struct {
	repo             repository.StatsRepository
	urlServiceClient pb.URLServiceClient
}

func (s statsService) CheckURLOwnership(ctx context.Context, token string, urlID int32) error {
	ctx = metadata.NewOutgoingContext(
		ctx,
		metadata.Pairs("authorization", token),
	)

	_, err := s.urlServiceClient.GetURL(ctx, &pb.GetURLRequest{UrlId: urlID})
	if err != nil {
		return err
	}

	return nil
}

func (s statsService) GetURLStats(ctx context.Context, token string, urlID int32, date string) ([]internal2.URLStats, error) {
	err := s.CheckURLOwnership(ctx, token, urlID)
	if err != nil {
		logrus.Error("error checking url ownership: ", err)
		return nil, err
	}

	stats, err := s.repo.GetURLStats(ctx, urlID, date)
	if err != nil {
		return nil, err
	}

	return stats, nil
}

func (s statsService) RecordClickBatch(ctx context.Context, events []broker.RedirectEvent) error {
	clicks := make([]internal2.Click, 0, len(events))
	for _, event := range events {
		clicks = append(clicks, adapter.EventToClick(event))
	}

	logrus.Info("Received ", len(clicks), " clicks")

	return s.repo.RecordClickBatch(ctx, clicks)
}

func (s statsService) RecordClick(ctx context.Context, event broker.RedirectEvent) error {
	click := adapter.EventToClick(event)

	logrus.Info("Click received: ", click)

	return s.repo.RecordClick(ctx, click)
}

func NewStatsService(repo repository.StatsRepository, urlServiceClient pb.URLServiceClient) StatsService {
	return &statsService{
		repo:             repo,
		urlServiceClient: urlServiceClient,
	}
}
