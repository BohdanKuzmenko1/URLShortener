package main

import (
	pb "github.com/BohdanKuzmenko1/URLShortener/proto"
	"github.com/BohdanKuzmenko1/URLShortener/services/url-service/internal/broker/kafka"
	"github.com/BohdanKuzmenko1/URLShortener/services/url-service/internal/repository"
	"github.com/BohdanKuzmenko1/URLShortener/services/url-service/internal/server"
	"github.com/BohdanKuzmenko1/URLShortener/services/url-service/internal/service"
	"github.com/BohdanKuzmenko1/URLShortener/services/url-service/internal/storage"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	if err := initConfig(); err != nil {
		logrus.Fatalf("an error initializing configs: %s", err.Error())
	}
	if err := godotenv.Load(); err != nil {
		logrus.Fatalf("an error loading env variables: %s", err.Error())
	}

	brokerAddr := os.Getenv("KAFKA_BROKER")
	if err := kafka.CreateTopic(brokerAddr, "redirects", 8); err != nil {
		logrus.Fatalf("failed to create kafka topic: %s", err.Error())
	}

	producer, err := kafka.NewProducer()
	if err != nil {
		logrus.Fatalf("an error initializing kafka client: %s", err.Error())
	}

	redisClient, err := storage.NewClient(storage.RedisConfig{
		Addr:     viper.GetString("redis.address"),
		Password: "",
		DB:       0,
	})
	if err != nil {
		logrus.Fatal(err)
	}

	postgres, err := repository.NewPostgresDB(repository.Config{
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

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	urlRepo := repository.NewURLShortenerRepository(postgres)
	urlService := service.NewURLShortenerService(urlRepo, producer, redisClient)
	defer urlService.Close()

	s := grpc.NewServer()
	pb.RegisterURLServiceServer(s, server.NewURLServer(urlService))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Println("URL Service is running on port 50051...")
		if err := s.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	<-quit
	log.Println("Shutting down gracefully...")
	s.GracefulStop()
}

func initConfig() error {
	viper.AddConfigPath("configs")
	viper.SetConfigName("config")
	return viper.ReadInConfig()
}
