package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/BohdanKuzmenko1/URLShortener/services/api-gateway/internal/client"
	"github.com/BohdanKuzmenko1/URLShortener/services/api-gateway/internal/handler"
	"github.com/BohdanKuzmenko1/URLShortener/services/api-gateway/internal/server"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func main() {
	if err := initConfig(); err != nil {
		logrus.Fatalf("error initializing config: %s", err)
	}

	if err := godotenv.Load(); err != nil {
		logrus.Warnf(".env file not found, using environment variables: %s", err)
	}

	clients, err := client.NewClients(
		viper.GetString("url-service.address"),
		viper.GetString("auth-service.address"),
		viper.GetString("stats-service.address"),
	)
	if err != nil {
		logrus.Fatalf("failed to initialize service clients: %s", err)
	}
	defer clients.Close()

	handlers := handler.NewHandler(clients.URL, clients.Auth, clients.Stats)

	srv := new(server.Server)

	go func() {
		if err := srv.Run(viper.GetString("api-gateway.port"), handlers.InitRoutes()); err != nil {
			logrus.Fatalf("server error: %s", err)
		}
	}()

	logrus.Info("API Gateway started")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	logrus.Info("API Gateway shutting down...")
}

func initConfig() error {
	viper.AddConfigPath("configs")
	viper.SetConfigName("config")
	return viper.ReadInConfig()
}
