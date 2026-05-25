package queue

import (
	"context"
	"log"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"github.com/srujanpendurthi/transcriptflow/internal/services/transcript"
	"gorm.io/gorm"
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

// RegisterHandlers wires all task types to their processors on mux.
func RegisterHandlers(mux *asynq.ServeMux, db *gorm.DB, p *transcript.Pipeline) {
	mux.HandleFunc(TypeTranscriptionProcess, NewTranscriptionProcessor(db, p).ProcessTask)
}
