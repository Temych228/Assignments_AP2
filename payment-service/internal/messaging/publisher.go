package messaging

import (
	"context"

	"payment-service/internal/events"
)

type PaymentEventPublisher interface {
	PublishPaymentCompleted(ctx context.Context, event events.PaymentCompleted) error
	Close() error
}
