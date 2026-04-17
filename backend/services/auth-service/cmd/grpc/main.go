package main

import (
	pb "github.com/BohdanKuzmenko1/URLShortener/proto"
	"github.com/BohdanKuzmenko1/URLShortener/services/auth-service/internal/client"
	"github.com/BohdanKuzmenko1/URLShortener/services/auth-service/internal/repository"
	"github.com/BohdanKuzmenko1/URLShortener/services/auth-service/internal/server"
	"github.com/BohdanKuzmenko1/URLShortener/services/auth-service/internal/service"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"net"
	"os"
)

func main() {
	if err := initConfig(); err != nil {
		logrus.Fatalf("an error initializing configs: %s", err.Error())
	}
	if err := godotenv.Load(); err != nil {
		logrus.Fatalf("an error loading env variables: %s", err.Error())
	}

	redisClient, err := client.NewClient(client.RedisConfig{
		Addr: viper.GetString("redis.address"),
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
		logrus.Fatalf("failed to connect to database: %v", err)
	}

	lis, err := net.Listen("tcp", viper.GetString("auth-service.port"))
	if err != nil {
		logrus.Fatalf("failed to listen: %v", err)
	}

	urlRepo := repository.NewAuthRepository(postgres)
	authService := service.NewAuthService(urlRepo, redisClient)

	s := grpc.NewServer()
	pb.RegisterAuthServiceServer(s, server.NewAuthServer(authService))

	logrus.Infof("Auth Service is running on port %s...", viper.GetString("auth-service.port"))
	if err := s.Serve(lis); err != nil {
		logrus.Fatalf("failed to serve: %v", err)
	}
}

// initConfig loads the application configuration from the configs/config file
// using Viper. Returns an error if the config file cannot be read.
func initConfig() error {
	viper.AddConfigPath("configs")
	viper.SetConfigName("config")
	return viper.ReadInConfig()
}
