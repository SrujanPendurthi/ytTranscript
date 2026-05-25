package main

import (
	"log"

	"github.com/hibiken/asynq"
	"github.com/srujanpendurthi/transcriptflow/internal/config"
	"github.com/srujanpendurthi/transcriptflow/internal/db"
	"github.com/srujanpendurthi/transcriptflow/internal/queue"
	"github.com/srujanpendurthi/transcriptflow/internal/services/transcript"
)

func main() {
	cfg := config.Load()

	database, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}

	redisOpts, err := queue.ParseRedisURL(cfg.RedisURL)
	if err != nil {
		log.Fatalf("redis url: %v", err)
	}

	redisClient, err := queue.NewRedisClient(cfg.RedisURL)
	if err != nil {
		log.Fatalf("redis: %v", err)
	}
	defer redisClient.Close()

	pipeline := transcript.New(redisClient, cfg.WhisperURL)

	srv := queue.NewServer(redisOpts, cfg.WorkerConcurrency)

	mux := asynq.NewServeMux()
	queue.RegisterHandlers(mux, database, pipeline)

	log.Printf("TranscriptFlow worker starting (concurrency=%d, whisper=%s)…",
		cfg.WorkerConcurrency, cfg.WhisperURL)
	if err := srv.Run(mux); err != nil {
		log.Fatalf("worker: %v", err)
	}
}
