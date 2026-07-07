package queue

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// NewRedisClientWithRetry calls NewRedisClient repeatedly until it succeeds or
// attempts run out, sleeping delay between tries — for container startup when
// Redis may not be reachable yet (e.g. after a Docker restart).
func NewRedisClientWithRetry(redisURL string, attempts int, delay time.Duration) (*redis.Client, error) {
	var lastErr error
	for i := 1; i <= attempts; i++ {
		client, err := NewRedisClient(redisURL)
		if err == nil {
			return client, nil
		}
		lastErr = err
		log.Printf("redis: connect attempt %d/%d failed: %v", i, attempts, err)
		time.Sleep(delay)
	}
	return nil, fmt.Errorf("redis: giving up after %d attempts: %w", attempts, lastErr)
}

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
