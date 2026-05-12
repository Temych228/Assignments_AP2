package app

import (
	"context"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"notification-service/internal/messaging"
	"notification-service/internal/provider"

	"github.com/redis/go-redis/v9"
)

type App struct {
	consumer *messaging.RabbitMQConsumer
	rdb      *redis.Client
}

func New() (*App, error) {
	rabbitURL := os.Getenv("RABBITMQ_URL")
	if rabbitURL == "" {
		rabbitURL = "amqp://guest:guest@localhost:5672/"
	}

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "localhost:6379"
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: redisURL,
		DB:   0,
	})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}

	sender := buildSenderFromEnv()

	retryMax := mustInt(os.Getenv("NOTIF_RETRY_MAX"), 4)
	backoff := mustDuration(os.Getenv("NOTIF_BACKOFF_BASE"), 2*time.Second)
	ttl := mustDuration(os.Getenv("NOTIF_IDEMPOTENCY_TTL"), 24*time.Hour)

	consumer, err := messaging.NewRabbitMQConsumer(messaging.ConsumerConfig{
		RabbitURL:    rabbitURL,
		Sender:       sender,
		Redis:        rdb,
		RetryMax:     retryMax,
		BaseBackoff:  backoff,
		ProcessedTTL: ttl,
	})
	if err != nil {
		_ = rdb.Close()
		return nil, err
	}

	return &App{consumer: consumer, rdb: rdb}, nil
}

func (a *App) Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	return a.consumer.Consume(ctx)
}

func (a *App) Close() error {
	if a.consumer != nil {
		_ = a.consumer.Close()
	}
	if a.rdb != nil {
		return a.rdb.Close()
	}
	return nil
}

func buildSenderFromEnv() provider.EmailSender {
	mode := strings.ToUpper(strings.TrimSpace(os.Getenv("PROVIDER_MODE")))
	switch mode {
	case "REAL":
		// Если не хочешь сейчас поднимать SMTP/Mailjet,
		// оставь SIMULATED — для защиты этого достаточно.
		fallthrough
	default:
		latency := mustDuration(os.Getenv("PROVIDER_LATENCY"), 700*time.Millisecond)
		failureRate := mustFloat(os.Getenv("PROVIDER_FAILURE_RATE"), 0.25)
		return provider.NewMockEmailSender(latency, failureRate)
	}
}

func mustDuration(v string, def time.Duration) time.Duration {
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}

func mustInt(v string, def int) int {
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func mustFloat(v string, def float64) float64 {
	if v == "" {
		return def
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return def
	}
	return f
}
