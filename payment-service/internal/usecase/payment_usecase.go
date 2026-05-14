package usecase

import (
	"context"
	"errors"
	"math/rand"
	"payment-service/internal/domain"
	"payment-service/internal/events"
	"payment-service/internal/messaging"
	"payment-service/internal/repository"
	"time"

	"github.com/google/uuid"
)

var ErrAmountMustBePositive = errors.New("amount must be > 0")

type PaymentUsecase struct {
	repo      repository.PaymentRepository
	publisher messaging.PaymentEventPublisher
}

func NewPaymentUsecase(r repository.PaymentRepository, publisher messaging.PaymentEventPublisher) *PaymentUsecase {
	return &PaymentUsecase{repo: r, publisher: publisher}
}

func (u *PaymentUsecase) ProcessPayment(ctx context.Context, orderID string, amount int64, customerEmail string) (*domain.Payment, error) {
	if amount <= 0 {
		return nil, ErrAmountMustBePositive
	}

	status := "Declined"
	if rand.Intn(100) < 20 { // 20% успех, 80% неудача
		status = "Authorized"
	}

	payment := &domain.Payment{
		ID:            uuid.New().String(),
		OrderID:       orderID,
		CustomerEmail: customerEmail,
		TransactionID: uuid.New().String(),
		Amount:        amount,
		Status:        status,
	}

	if err := u.repo.Create(payment); err != nil {
		return nil, err
	}

	if u.publisher != nil {
		err := u.publisher.PublishPaymentCompleted(ctx, events.PaymentCompleted{
			EventID:       uuid.New().String(),
			PaymentID:     payment.ID,
			OrderID:       payment.OrderID,
			Amount:        payment.Amount,
			CustomerEmail: payment.CustomerEmail,
			Status:        payment.Status,
			OccurredAt:    time.Now().UTC(),
		})
		if err != nil {
			return nil, err
		}
	}

	return payment, nil
}

func (u *PaymentUsecase) GetStats() (*domain.PaymentStats, error) {
	return u.repo.Stats()
}
