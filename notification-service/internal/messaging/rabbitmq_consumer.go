package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	paymentsExchange     = "payments"
	paymentsDLX          = "payments.dlx"
	paymentCompletedKey  = "payment.completed"
	paymentCompletedDLQ  = "payment.completed.dlq"
	paymentCompletedName = "payment.completed"
)

type RabbitMQConsumer struct {
	conn        *amqp.Connection
	ch          *amqp.Channel
	seen        map[string]struct{}
	mu          sync.Mutex
	failOrderID string
	failEmail   string
}

func NewRabbitMQConsumer(url, failOrderID, failEmail string) (*RabbitMQConsumer, error) {
	conn, err := dialWithRetry(url, 20, 2*time.Second)
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

	return &RabbitMQConsumer{
		conn:        conn,
		ch:          ch,
		seen:        make(map[string]struct{}),
		failOrderID: failOrderID,
		failEmail:   failEmail,
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
			c.handleDelivery(delivery)
		}
	}
}

func (c *RabbitMQConsumer) handleDelivery(delivery amqp.Delivery) {
	var event PaymentCompleted
	if err := json.Unmarshal(delivery.Body, &event); err != nil {
		log.Printf("[Notification] invalid payment event: %v", err)
		_ = delivery.Nack(false, false)
		return
	}

	if (c.failOrderID != "" && event.OrderID == c.failOrderID) || (c.failEmail != "" && event.CustomerEmail == c.failEmail) {
		log.Printf("[Notification] simulated permanent failure for Order #%s", event.OrderID)
		_ = delivery.Nack(false, true)
		return
	}

	eventID := formatEventID(event)
	if c.isDuplicate(eventID) {
		_ = delivery.Ack(false)
		return
	}

	log.Printf(
		"[Notification] Sent email to %s for Order #%s. Amount: $%.2f",
		event.CustomerEmail,
		event.OrderID,
		float64(event.Amount)/100,
	)
	c.markSeen(eventID)
	_ = delivery.Ack(false)
}

func (c *RabbitMQConsumer) isDuplicate(eventID string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, ok := c.seen[eventID]
	return ok
}

func (c *RabbitMQConsumer) markSeen(eventID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.seen[eventID] = struct{}{}
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
	if err := ch.QueueBind(paymentCompletedName, paymentCompletedKey, paymentsExchange, false, nil); err != nil {
		return err
	}
	return nil
}

func formatEventID(event PaymentCompleted) string {
	if event.EventID != "" {
		return event.EventID
	}
	return fmt.Sprintf("%s:%d:%s", event.OrderID, event.Amount, event.Status)
}
