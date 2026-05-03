package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"payment-service/internal/events"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	paymentsExchange     = "payments"
	paymentsDLX          = "payments.dlx"
	paymentCompletedKey  = "payment.completed"
	paymentCompletedDLQ  = "payment.completed.dlq"
	paymentCompletedName = "payment.completed"
)

type RabbitMQPublisher struct {
	conn     *amqp.Connection
	ch       *amqp.Channel
	confirms chan amqp.Confirmation
	mu       sync.Mutex
}

func NewRabbitMQPublisher(url string) (*RabbitMQPublisher, error) {
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

	if err := ch.Confirm(false); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, err
	}

	return &RabbitMQPublisher{
		conn:     conn,
		ch:       ch,
		confirms: ch.NotifyPublish(make(chan amqp.Confirmation, 1)),
	}, nil
}

func (p *RabbitMQPublisher) PublishPaymentCompleted(ctx context.Context, event events.PaymentCompleted) error {
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if err := p.ch.PublishWithContext(
		ctx,
		paymentsExchange,
		paymentCompletedKey,
		true,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			MessageId:    event.EventID,
			Timestamp:    event.OccurredAt,
			Body:         body,
		},
	); err != nil {
		return err
	}

	select {
	case confirm := <-p.confirms:
		if !confirm.Ack {
			return fmt.Errorf("rabbitmq did not confirm payment event %s", event.EventID)
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *RabbitMQPublisher) Close() error {
	if p.ch != nil {
		_ = p.ch.Close()
	}
	if p.conn != nil {
		return p.conn.Close()
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
