package queue

import (
	"fmt"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
)

// Client wraps asynq.Client for enqueuing background jobs.
type Client struct {
	inner *asynq.Client
}

// NewClient creates an asynq enqueue client from a Redis URL.
func NewClient(redisURL string) (*Client, error) {
	opts, err := ParseRedisURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("queue client: %w", err)
	}
	inner := asynq.NewClient(asynq.RedisClientOpt{
		Addr:     opts.Addr,
		Password: opts.Password,
		DB:       opts.DB,
		PoolSize: opts.PoolSize,
	})
	return &Client{inner: inner}, nil
}

// NewClientFromRedis creates an asynq client reusing an existing go-redis options set.
func NewClientFromRedis(opts *redis.Options) *Client {
	return &Client{
		inner: asynq.NewClient(asynq.RedisClientOpt{
			Addr:     opts.Addr,
			Password: opts.Password,
			DB:       opts.DB,
		}),
	}
}

// Enqueue pushes a prepared asynq task onto the queue. Callers own building the
// task (see each feature's task constructor) and any DB bookkeeping beforehand.
func (c *Client) Enqueue(task *asynq.Task) error {
	_, err := c.inner.Enqueue(task)
	return err
}

func (c *Client) Close() error {
	return c.inner.Close()
}
