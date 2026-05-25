package queue

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
)

// Client wraps asynq.Client for enqueuing transcription jobs.
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

// EnqueueTranscription pushes a transcription job onto the queue.
// The caller is responsible for having already written the job row to Postgres
// with status=pending before calling this.
func (c *Client) EnqueueTranscription(jobID uuid.UUID) error {
	task, err := NewTranscriptionTask(jobID)
	if err != nil {
		return err
	}
	_, err = c.inner.Enqueue(task)
	return err
}

func (c *Client) Close() error {
	return c.inner.Close()
}
