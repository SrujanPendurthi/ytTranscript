package queue

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// NewRedisClient parses a Redis URL and returns a pooled client.
// Pool is tuned for mixed API + worker use: 10 max connections, 2 always warm.
func NewRedisClient(redisURL string) (*redis.Client, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}
	opts.PoolSize = 10
	opts.MinIdleConns = 2

	client := redis.NewClient(opts)
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}
	return client, nil
}

// ParseRedisURL parses a Redis URL into a *redis.Options struct shared by
// both NewRedisClient and the asynq server/client constructors.
func ParseRedisURL(redisURL string) (*redis.Options, error) {
	return redis.ParseURL(redisURL)
}
