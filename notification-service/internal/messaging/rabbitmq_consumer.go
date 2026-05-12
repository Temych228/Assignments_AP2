package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"notification-service/internal/provider"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
)

const (
	paymentsExchange     = "payments"
	paymentsDLX          = "payments.dlx"
	paymentCompletedKey  = "payment.completed"
	paymentCompletedDLQ  = "payment.completed.dlq"
	paymentCompletedName = "payment.completed"
)

type ConsumerConfig struct {
	RabbitURL    string
	Sender       provider.EmailSender
	Redis        *redis.Client
	RetryMax     int
	BaseBackoff  time.Duration
	ProcessedTTL time.Duration
}

type RabbitMQConsumer struct {
	conn         *amqp.Connection
	ch           *amqp.Channel
	sender       provider.EmailSender
	rdb          *redis.Client
	retryMax     int
	baseBackoff  time.Duration
	processedTTL time.Duration

	mu sync.Mutex
}

func NewRabbitMQConsumer(cfg ConsumerConfig) (*RabbitMQConsumer, error) {
	conn, err := dialWithRetry(cfg.RabbitURL, 20, 2*time.Second)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	if err := declareTopology(ch); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, err
	}

	if err := ch.Qos(1, 0, false); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, err
	}

	if cfg.RetryMax <= 0 {
		cfg.RetryMax = 4
	}
	if cfg.BaseBackoff <= 0 {
		cfg.BaseBackoff = 2 * time.Second
	}
	if cfg.ProcessedTTL <= 0 {
		cfg.ProcessedTTL = 24 * time.Hour
	}

	return &RabbitMQConsumer{
		conn:         conn,
		ch:           ch,
		sender:       cfg.Sender,
		rdb:          cfg.Redis,
		retryMax:     cfg.RetryMax,
		baseBackoff:  cfg.BaseBackoff,
		processedTTL: cfg.ProcessedTTL,
	}, nil
}

func (c *RabbitMQConsumer) Consume(ctx context.Context) error {
	deliveries, err := c.ch.Consume(paymentCompletedName, "", false, false, false, false, nil)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case delivery, ok := <-deliveries:
			if !ok {
				return nil
			}
			c.handleDelivery(ctx, delivery)
		}
	}
}

func (c *RabbitMQConsumer) handleDelivery(ctx context.Context, delivery amqp.Delivery) {
	var event PaymentCompleted
	if err := json.Unmarshal(delivery.Body, &event); err != nil {
		log.Printf("[Notification] invalid payment event: %v", err)
		_ = delivery.Nack(false, false)
		return
	}

	paymentID := event.PaymentID
	if paymentID == "" {
		paymentID = event.EventID
	}

	if processed, err := c.isProcessed(ctx, paymentID); err == nil && processed {
		_ = delivery.Ack(false)
		return
	}

	// помечаем как "processing"
	if err := c.markProcessing(ctx, paymentID); err != nil {
		log.Printf("[Notification] redis mark processing error: %v", err)
		_ = delivery.Nack(false, true)
		return
	}

	subject := fmt.Sprintf("Payment %s for order %s", event.Status, event.OrderID)
	body := fmt.Sprintf(
		"Order: %s\nPayment: %s\nAmount: %.2f\nStatus: %s\n",
		event.OrderID,
		event.PaymentID,
		float64(event.Amount)/100,
		event.Status,
	)

	var sendErr error
	for attempt := 1; attempt <= c.retryMax; attempt++ {
		sendErr = c.sender.Send(ctx, event.CustomerEmail, subject, body)
		if sendErr == nil {
			_ = c.markDone(ctx, paymentID)
			_ = delivery.Ack(false)
			return
		}

		if attempt < c.retryMax {
			backoff := c.baseBackoff * time.Duration(1<<(attempt-1))
			log.Printf("[Notification] send failed (attempt %d/%d): %v; retrying in %s", attempt, c.retryMax, sendErr, backoff)
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				_ = c.clearStatus(ctx, paymentID)
				return
			}
		}
	}

	log.Printf("[Notification] permanent failure for payment %s: %v", paymentID, sendErr)
	_ = c.clearStatus(ctx, paymentID)
	_ = delivery.Nack(false, false)
}

func (c *RabbitMQConsumer) statusKey(paymentID string) string {
	return "notif:payment:" + paymentID
}

func (c *RabbitMQConsumer) isProcessed(ctx context.Context, paymentID string) (bool, error) {
	val, err := c.rdb.Get(ctx, c.statusKey(paymentID)).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return val == "done", nil
}

func (c *RabbitMQConsumer) markProcessing(ctx context.Context, paymentID string) error {
	ok, err := c.rdb.SetNX(ctx, c.statusKey(paymentID), "processing", c.processedTTL).Result()
	if err != nil {
		return err
	}
	if !ok {
		val, err := c.rdb.Get(ctx, c.statusKey(paymentID)).Result()
		if err != nil {
			return err
		}
		if val == "done" {
			return nil
		}
	}
	return nil
}

func (c *RabbitMQConsumer) markDone(ctx context.Context, paymentID string) error {
	return c.rdb.Set(ctx, c.statusKey(paymentID), "done", c.processedTTL).Err()
}

func (c *RabbitMQConsumer) clearStatus(ctx context.Context, paymentID string) error {
	return c.rdb.Del(ctx, c.statusKey(paymentID)).Err()
}

func (c *RabbitMQConsumer) Close() error {
	if c.ch != nil {
		_ = c.ch.Close()
	}
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func dialWithRetry(url string, attempts int, delay time.Duration) (*amqp.Connection, error) {
	var lastErr error
	for i := 0; i < attempts; i++ {
		conn, err := amqp.Dial(url)
		if err == nil {
			return conn, nil
		}
		lastErr = err
		time.Sleep(delay)
	}
	return nil, lastErr
}

func declareTopology(ch *amqp.Channel) error {
	if err := ch.ExchangeDeclare(paymentsExchange, "direct", true, false, false, false, nil); err != nil {
		return err
	}
	if err := ch.ExchangeDeclare(paymentsDLX, "direct", true, false, false, false, nil); err != nil {
		return err
	}
	if _, err := ch.QueueDeclare(paymentCompletedDLQ, true, false, false, false, nil); err != nil {
		return err
	}
	if err := ch.QueueBind(paymentCompletedDLQ, paymentCompletedDLQ, paymentsDLX, false, nil); err != nil {
		return err
	}

	args := amqp.Table{
		"x-queue-type":              "quorum",
		"x-delivery-limit":          int32(3),
		"x-dead-letter-exchange":    paymentsDLX,
		"x-dead-letter-routing-key": paymentCompletedDLQ,
	}
	if _, err := ch.QueueDeclare(paymentCompletedName, true, false, false, false, args); err != nil {
		return err
	}
	return ch.QueueBind(paymentCompletedName, paymentCompletedKey, paymentsExchange, false, nil)
}
