package main

import (
	pb "github.com/BohdanKuzmenko1/URLShortener/proto"
	"github.com/BohdanKuzmenko1/URLShortener/services/auth-service/internal/repository"
	"github.com/BohdanKuzmenko1/URLShortener/services/auth-service/internal/server"
	"github.com/BohdanKuzmenko1/URLShortener/services/auth-service/internal/service"
	"github.com/BohdanKuzmenko1/URLShortener/services/auth-service/internal/storage"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
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
		logrus.Warnf(".env file not found, using environment variables: %s", err.Error())
	}

	redisClient, err := storage.NewRedisClient(storage.RedisConfig{
		Addr: viper.GetString("redis.address"),
	})
	if err != nil {
		logrus.Fatalf("failed to connect to redis: %v", err)
	}
	defer func(client *redis.Client) {
		if err := client.Close(); err != nil {
			logrus.Errorf("failed to close redis connection: %v", err)
		}
	}(redisClient)

	postgres, err := repository.NewPostgresDB(repository.Config{
		Host:     viper.GetString("postgres.host"),
		Port:     viper.GetString("postgres.port"),
		Username: viper.GetString("postgres.username"),
		Password: os.Getenv("DB_PASSWORD"),
		DBName:   viper.GetString("postgres.dbname"),
		SSLMode:  viper.GetString("postgres.sslmode"),
	})
	if err != nil {
		logrus.Fatalf("failed to connect to database: %v", err)
	}
	defer func(db *sqlx.DB) {
		if err := db.Close(); err != nil {
			logrus.Errorf("failed to close postgres connection: %v", err)
		}
	}(postgres)

	lis, err := net.Listen("tcp", viper.GetString("auth-service.port"))
	if err != nil {
		logrus.Fatalf("failed to listen: %v", err)
	}

	authRepo := repository.NewAuthRepository(postgres)
	authStorage := storage.NewAuthStorage(redisClient)
	authService := service.NewAuthService(authRepo, authStorage)

	s := grpc.NewServer()
	pb.RegisterAuthServiceServer(s, server.NewAuthServer(authService))

	logrus.Infof("Auth Service is running on port %s...", viper.GetString("auth-service.port"))

	go func() {
		if err := s.Serve(lis); err != nil {
			logrus.Fatalf("failed to serve: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	logrus.Info("Auth Service shutting down...")
	s.GracefulStop()
}

// initConfig loads the application configuration from the configs/config file
// using Viper. Returns an error if the config file cannot be read.
func initConfig() error {
	viper.AddConfigPath("configs")
	viper.SetConfigName("config")
	return viper.ReadInConfig()
}
