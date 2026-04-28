package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

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
)

func main() {
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
		log.Fatalf("failed to connect to database: %v", err)
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

	go func() {
		consumer.StartBatch(
			ctx,
			func(ctx context.Context, events []broker.RedirectEvent) error {
				dbCtx, dbCancel := context.WithTimeout(ctx, 5*time.Second)
				defer dbCancel()

				return statsService.RecordClickBatch(dbCtx, events)
			},
			500,
			100*time.Millisecond,
		)
		log.Println("Kafka consumer stopped")
	}()

	s := grpc.NewServer()
	pb.RegisterStatsServiceServer(s, server.NewStatsServer(statsService))

	// Graceful shutdown: on SIGTERM or SIGINT, cancel the context
	// to stop the Kafka consumer and wait for in-flight gRPC requests to complete.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		sig := <-quit
		log.Printf("received signal: %s, shutting down...", sig)
		cancel()
		s.GracefulStop()
	}()

	log.Println(fmt.Sprintf("Stats Service is running on port %s...", viper.GetString("stats-service.port")))
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func initConfig() error {
	viper.AddConfigPath("configs")
	viper.SetConfigName("config")
	return viper.ReadInConfig()
}
