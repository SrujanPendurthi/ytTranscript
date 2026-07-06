package queue

import (
	"context"
	"log"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
)

// Queue names for asynq routing. Features enqueue onto these; the server below
// assigns their relative priority.
const (
	QueueTranscription = "transcription"
	QueueDefault       = "default"
)

// NewServer builds an asynq.Server from a parsed *redis.Options.
// The transcription queue runs at 2× the priority of the default queue.
func NewServer(opts *redis.Options, concurrency int) *asynq.Server {
	return asynq.NewServer(
		asynq.RedisClientOpt{
			Addr:     opts.Addr,
			Password: opts.Password,
			DB:       opts.DB,
		},
		asynq.Config{
			Concurrency: concurrency,
			Queues: map[string]int{
				QueueTranscription: 6,
				QueueDefault:       3,
			},
			ErrorHandler: asynq.ErrorHandlerFunc(
				func(ctx context.Context, task *asynq.Task, err error) {
					log.Printf("[asynq] task=%s err=%v", task.Type(), err)
				},
			),
		},
	)
}
