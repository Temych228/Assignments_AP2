package app

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"notification-service/internal/clients"
	"notification-service/internal/messaging"
)

type App struct {
	consumer    *messaging.RabbitMQConsumer
	orderClient *clients.OrderClientGRPC
}

func New() (*App, error) {
	rabbitURL := os.Getenv("RABBITMQ_URL")
	if rabbitURL == "" {
		rabbitURL = "amqp://guest:guest@localhost:5672/"
	}

	consumer, err := messaging.NewRabbitMQConsumer(
		rabbitURL,
		os.Getenv("FAIL_ORDER_ID"),
		os.Getenv("FAIL_CUSTOMER_EMAIL"),
	)
	if err != nil {
		return nil, err
	}

	// Optional: connect to Order Service gRPC for streaming order updates.
	// If ORDER_GRPC_ADDR is not set we skip it gracefully.
	var orderClient *clients.OrderClientGRPC
	if addr := os.Getenv("ORDER_GRPC_ADDR"); addr != "" {
		orderClient, err = clients.NewOrderClientGRPC(addr)
		if err != nil {
			log.Printf("[Notification] WARNING: could not connect to order-service gRPC at %s: %v", addr, err)
			orderClient = nil
		} else {
			log.Printf("[Notification] Connected to order-service gRPC at %s", addr)
		}
	}

	return &App{consumer: consumer, orderClient: orderClient}, nil
}

func (a *App) Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// If gRPC client is available, subscribe to a wildcard order stream.
	// In a real system you'd subscribe per-order after receiving a payment event;
	// here we subscribe to a fixed test order ID from env to demonstrate the gRPC link.
	if a.orderClient != nil {
		if testOrderID := os.Getenv("SUBSCRIBE_ORDER_ID"); testOrderID != "" {
			go func() {
				log.Printf("[Notification][gRPC] Subscribing to updates for order %s", testOrderID)
				if err := a.orderClient.SubscribeToOrderUpdates(ctx, testOrderID); err != nil {
					log.Printf("[Notification][gRPC] stream ended: %v", err)
				}
			}()
		}
	}

	return a.consumer.Consume(ctx)
}

func (a *App) Close() error {
	if a.orderClient != nil {
		_ = a.orderClient.Close()
	}
	return a.consumer.Close()
}
