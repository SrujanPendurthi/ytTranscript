package main

import (
	"log"
	"time"

	"github.com/hibiken/asynq"
	"github.com/srujanpendurthi/transcriptflow/internal/features/transcript"
	"github.com/srujanpendurthi/transcriptflow/internal/platform/config"
	"github.com/srujanpendurthi/transcriptflow/internal/platform/db"
	"github.com/srujanpendurthi/transcriptflow/internal/platform/queue"
)

// startupRetries / startupDelay bound how long the worker waits for Postgres
// and Redis on boot before giving up (~60s) — covers Docker restart races.
const (
	startupRetries = 30
	startupDelay   = 2 * time.Second
)

func main() {
	cfg := config.Load()

	database, err := db.ConnectWithRetry(cfg.DatabaseURL, startupRetries, startupDelay)
	if err != nil {
		log.Fatalf("db: %v", err)
	}

	redisOpts, err := queue.ParseRedisURL(cfg.RedisURL)
	if err != nil {
		log.Fatalf("redis url: %v", err)
	}

	redisClient, err := queue.NewRedisClientWithRetry(cfg.RedisURL, startupRetries, startupDelay)
	if err != nil {
		log.Fatalf("redis: %v", err)
	}
	defer redisClient.Close()

	pipeline := transcript.New(redisClient, cfg.OpenAIAPIKey)

	srv := queue.NewServer(redisOpts, cfg.WorkerConcurrency)

	mux := asynq.NewServeMux()
	mux.HandleFunc(
		transcript.TypeTranscriptionProcess,
		transcript.NewTranscriptionProcessor(database, pipeline).ProcessTask,
	)

	log.Printf("TranscriptFlow worker starting (concurrency=%d, openai=%t)…",
		cfg.WorkerConcurrency, cfg.OpenAIAPIKey != "")
	if err := srv.Run(mux); err != nil {
		log.Fatalf("worker: %v", err)
	}
}
