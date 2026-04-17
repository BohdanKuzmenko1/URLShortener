package client

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisConfig holds the configuration parameters required to establish
// a Redis client connection.
type RedisConfig struct {
	Addr     string // Redis server address in "host:port" format
	Password string // password for authentication, empty if not required
	DB       int    // database number to select after connecting
}

// NewClient creates and returns a new Redis client using the provided configuration.
// It verifies the connection with a 5-second timeout ping.
// Returns an error if the connection cannot be established.
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
