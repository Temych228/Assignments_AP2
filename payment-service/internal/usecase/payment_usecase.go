package usecase

import (
	"context"
	"errors"
	"time"

	"payment-service/internal/domain"
	"payment-service/internal/events"
	"payment-service/internal/messaging"
	"payment-service/internal/repository"

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

	status := "Authorized"
	if amount > 100000 {
		status = "Declined"
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

func (u *PaymentUsecase) GetPayment(orderID string) (*domain.Payment, error) {
	return u.repo.GetByOrderID(orderID)
}

func (u *PaymentUsecase) GetStats() (*domain.PaymentStats, error) {
	return u.repo.Stats()
}
