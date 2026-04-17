package main

import (
	"context"
	"fmt"
	pb "github.com/BohdanKuzmenko1/URLShortener/proto"
	"github.com/BohdanKuzmenko1/URLShortener/services/stats-service/internal/broker"
	"github.com/BohdanKuzmenko1/URLShortener/services/stats-service/internal/client"
	repository2 "github.com/BohdanKuzmenko1/URLShortener/services/stats-service/internal/repository"
	"github.com/BohdanKuzmenko1/URLShortener/services/stats-service/internal/server"
	"github.com/BohdanKuzmenko1/URLShortener/services/stats-service/internal/service"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"log"
	"net"
	"os"
	"time"
)

func main() {
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			fmt.Println("Tick:", time.Now())
		}
	}()
	if err := initConfig(); err != nil {
		logrus.Fatalf("an error initializing configs: %s", err.Error())
	}
	if err := godotenv.Load(); err != nil {
		logrus.Fatalf("an error loading env variables: %s", err.Error())
	}

	postgres, err := repository2.NewPostgresDB(repository2.Config{
		Host:     viper.GetString("postgres.host"),
		Port:     viper.GetString("postgres.port"),
		Username: viper.GetString("postgres.username"),
		Password: os.Getenv("DB_PASSWORD"),
		DBName:   viper.GetString("postgres.dbname"),
		SSLMode:  viper.GetString("postgres.sslmode"),
	})

	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	lis, err := net.Listen("tcp", viper.GetString("stats-service.port"))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	statsRepo := repository2.NewStatsRepository(postgres)

	urlServiceClient, err := client.NewURLServiceClient(viper.GetString("url-service.address"))
	if err != nil {
		log.Fatalf("failed to connect to URL service: %v", err)
	}
	statsService := service.NewStatsService(statsRepo, urlServiceClient)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	consumer, err := broker.NewRedirectConsumer()
	if err != nil {
		log.Fatalf("failed to start Kafka consumer: %v", err)
	}
	defer consumer.Close()

	go consumer.StartBatch(
		ctx,
		func(ctx context.Context, events []broker.RedirectEvent) error {
			return statsService.RecordClickBatch(ctx, events)
		},
		500,                  // batchSize
		100*time.Millisecond, // flushInterval
	)

	s := grpc.NewServer()
	pb.RegisterStatsServiceServer(s, server.NewStatsServer(statsService))

	log.Println(fmt.Sprintf("Stats Service NEW is running on port %s...", viper.GetString("stats-service-new.port")))
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func initConfig() error {
	viper.AddConfigPath("configs")
	viper.SetConfigName("config")
	return viper.ReadInConfig()
}
