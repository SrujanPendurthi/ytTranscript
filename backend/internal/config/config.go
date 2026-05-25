package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port                string
	DatabaseURL         string
	RedisURL            string
	JWTSecret           string
	StripeSecretKey     string
	StripeWebhookSecret string
	StripeProPriceID    string // price_… for the $10/month Pro plan
	AppURL              string // frontend base URL for Stripe redirect
	WorkerConcurrency   int
	WhisperURL          string
}

func Load() *Config {
	return &Config{
		Port:                getEnv("PORT", "8080"),
		DatabaseURL:         getEnv("DATABASE_URL", "postgres://transcriptflow:secret@localhost:5432/transcriptflow?sslmode=disable"),
		RedisURL:            getEnv("REDIS_URL", "redis://localhost:6379"),
		JWTSecret:           getEnv("JWT_SECRET", "changeme"),
		StripeSecretKey:     os.Getenv("STRIPE_SECRET_KEY"),
		StripeWebhookSecret: os.Getenv("STRIPE_WEBHOOK_SECRET"),
		StripeProPriceID:    os.Getenv("STRIPE_PRO_PRICE_ID"),
		AppURL:              getEnv("APP_URL", "http://localhost:5173"),
		WorkerConcurrency:   getEnvInt("WORKER_CONCURRENCY", 5),
		WhisperURL:          getEnv("WHISPER_URL", "http://localhost:9000"),
	}
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
