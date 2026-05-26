package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	stripe "github.com/stripe/stripe-go/v76"
	"github.com/srujanpendurthi/transcriptflow/internal/auth"
	"github.com/srujanpendurthi/transcriptflow/internal/config"
	"github.com/srujanpendurthi/transcriptflow/internal/db"
	"github.com/srujanpendurthi/transcriptflow/internal/handlers"
	mw "github.com/srujanpendurthi/transcriptflow/internal/middleware"
	"github.com/srujanpendurthi/transcriptflow/internal/queue"
)

func main() {
	cfg := config.Load()

	// Stripe global key — must be set before any stripe.* call.
	stripe.Key = cfg.StripeSecretKey

	// Postgres
	database, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	if err := db.RunMigrations(database); err != nil {
		log.Fatalf("migrations: %v", err)
	}

	// Redis
	redisClient, err := queue.NewRedisClient(cfg.RedisURL)
	if err != nil {
		log.Fatalf("redis: %v", err)
	}
	defer redisClient.Close()

	queueClient, err := queue.NewClient(cfg.RedisURL)
	if err != nil {
		log.Fatalf("queue client: %v", err)
	}
	defer queueClient.Close()

	// Handlers
	transcriptH := handlers.NewTranscriptHandler(database, queueClient)
	billingH := handlers.NewBillingHandler(
		database,
		cfg.StripeProPriceID,
		cfg.StripeWebhookSecret,
		cfg.AppURL,
	)
	apiKeyH := handlers.NewAPIKeyHandler(database)

	r := chi.NewRouter()
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		if err := redisClient.Ping(r.Context()).Err(); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, `{"status":"degraded","redis":"%v"}`, err)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"status":"ok"}`)
	})

	// Public — Stripe webhook (no auth; signature-validated inside handler)
	r.Post("/api/webhook/stripe", billingH.Webhook)

	// Public auth — under /api/auth so nginx can proxy /api/* uniformly.
	authH := auth.NewHandler(database, cfg.JWTSecret)
	r.Route("/api/auth", func(r chi.Router) {
		r.Post("/signup", authH.Signup)
		r.Post("/login", authH.Login)
	})

	// JWT-only routes (web dashboard — no API-key equivalent)
	r.Group(func(r chi.Router) {
		r.Use(mw.JWT(cfg.JWTSecret))
		r.Get("/api/v1/jobs", transcriptH.ListJobs)
		r.Post("/api/billing/checkout", billingH.Checkout)
		r.Get("/api/v1/keys", apiKeyH.List)
		r.Post("/api/v1/keys", apiKeyH.Create)
		r.Delete("/api/v1/keys/{keyID}", apiKeyH.Delete)
	})

	// Routes that accept both JWT (dashboard) and API-key (developer) callers.
	// Bearer token → JWT middleware; anything else → API-key middleware.
	r.Group(func(r chi.Router) {
		r.Use(mw.JWTOrAPIKey(cfg.JWTSecret, database))
		r.Post("/api/v1/transcribe", transcriptH.Submit)
		r.Get("/api/v1/jobs/{id}", transcriptH.GetJob)
	})

	fmt.Printf("TranscriptFlow API listening on :%s\n", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Fatalf("server: %v", err)
	}
}
