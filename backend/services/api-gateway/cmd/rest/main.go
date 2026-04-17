package main

import (
	"github.com/BohdanKuzmenko1/URLShortener/services/api-gateway/internal/client"
	"github.com/BohdanKuzmenko1/URLShortener/services/api-gateway/internal/handler"
	"github.com/BohdanKuzmenko1/URLShortener/services/api-gateway/internal/server"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"log"
)

func main() {
	if err := initConfig(); err != nil {
		logrus.Fatalf("an error initializing configs: %s", err.Error())
	}
	if err := godotenv.Load(); err != nil {
		logrus.Fatalf("an error loading env variables: %s", err.Error())
	}

	urlServiceClient, err := client.NewURLServiceClient(viper.GetString("url-service.address"))
	if err != nil {
		log.Fatalf("failed to connect to URL service: %v", err)
	}
	authServiceClient, err := client.NewAuthServiceClient(viper.GetString("auth-service.address"))
	if err != nil {
		log.Fatalf("failed to connect to auth service: %v", err)
	}
	statsService, err := client.NewStatsServiceClient(viper.GetString("stats-service.address"))
	if err != nil {
		log.Fatalf("failed to connect to stats service: %v", err)
	}
	handlers := handler.NewHandler(urlServiceClient, authServiceClient, statsService)

	srv := new(server.Server)

	if err := srv.Run(viper.GetString("api-gateway.port"), handlers.InitRoutes()); err != nil {
		log.Fatal(err)
	}
}

// initConfig loads the application configuration from the configs/config file
// using Viper. Returns an error if the config file cannot be read.
func initConfig() error {
	viper.AddConfigPath("configs")
	viper.SetConfigName("config")
	return viper.ReadInConfig()
}
