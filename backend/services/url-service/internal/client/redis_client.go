package client

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

func NewClient(cfg RedisConfig) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := client.Ping(ctx).Result()
	if err != nil {
		return nil, err
	}

	log.Println("Connected to Redis successfully")
	return client, nil
}
