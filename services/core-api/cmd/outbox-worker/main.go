package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/outbox"
	postgresstore "github.com/opportunity-os/opportunity-os/services/core-api/internal/postgres"
	"github.com/redis/go-redis/v9"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	databaseURL := env("DATABASE_URL", "postgres://opportunity:opportunity@localhost:5432/opportunity?sslmode=disable")
	redisURL := env("REDIS_URL", "redis://localhost:6379/0")
	redisOptions, err := redis.ParseURL(redisURL)
	if err != nil {
		logger.Error("invalid Redis URL", "error", err)
		os.Exit(1)
	}
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		logger.Error("database configuration failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	client := redis.NewClient(redisOptions)
	defer func() { _ = client.Close() }()
	if err = pool.Ping(ctx); err != nil {
		logger.Error("database unavailable", "error", err)
		os.Exit(1)
	}
	if err = client.Ping(ctx).Err(); err != nil {
		logger.Error("Redis unavailable", "error", err)
		os.Exit(1)
	}
	publisher := outbox.NewRedisPublisher(client, env("OUTBOX_STREAM", "opportunity.events"))
	if err = publisher.Check(ctx); err != nil {
		logger.Error("Redis Streams unavailable", "error", err)
		os.Exit(1)
	}
	hostname, _ := os.Hostname()
	worker := outbox.Worker{
		Repository: postgresstore.NewStore(pool), Publisher: publisher,
		WorkerID: hostname + "-" + strconv.Itoa(os.Getpid()), BatchSize: envInt("OUTBOX_BATCH_SIZE", 50),
		Lease: 30 * time.Second, MaxAttempts: envInt("OUTBOX_MAX_ATTEMPTS", 8),
	}
	poll := time.Duration(envInt("OUTBOX_POLL_MS", 1000)) * time.Millisecond
	logger.Info("outbox worker started", "worker_id", worker.WorkerID, "stream", env("OUTBOX_STREAM", "opportunity.events"))
	for {
		count, runErr := worker.RunOnce(ctx)
		if runErr != nil && ctx.Err() == nil {
			logger.Warn("outbox batch completed with failures", "leased", count, "error", runErr)
		}
		select {
		case <-ctx.Done():
			logger.Info("outbox worker stopped")
			return
		case <-time.After(poll):
		}
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) int {
	value, err := strconv.Atoi(os.Getenv(key))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
