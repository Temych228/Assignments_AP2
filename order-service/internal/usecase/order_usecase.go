package usecase

import (
	"database/sql"
	"errors"
	"time"

	"order-service/internal/domain"
	"order-service/internal/repository"
	"order-service/internal/usecase/ports"

	"github.com/google/uuid"
)

var (
	ErrAmountMustBePositive        = errors.New("amount must be > 0")
	ErrPaymentServiceDown          = errors.New("payment service unavailable")
	ErrCannotCancelPaidOrder       = errors.New("cannot cancel paid order")
	ErrCannotCancelNonPendingOrder = errors.New("only pending orders can be cancelled")
)

type OrderUsecase struct {
	repo   repository.OrderRepository
	payAPI ports.PaymentClient
}

func NewOrderUsecase(r repository.OrderRepository, payAPI ports.PaymentClient) *OrderUsecase {
	return &OrderUsecase{
		repo:   r,
		payAPI: payAPI,
	}
}

func (u *OrderUsecase) CreateOrder(customerID, itemName string, amount int64, idempotencyKey string) (*domain.Order, error) {
	if amount <= 0 {
		return nil, ErrAmountMustBePositive
	}

	if idempotencyKey != "" {
		existing, err := u.repo.GetByIdempotencyKey(idempotencyKey)
		if err == nil {
			return existing, nil
		}
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
	}

	order := &domain.Order{
		ID:             uuid.New().String(),
		CustomerID:     customerID,
		ItemName:       itemName,
		Amount:         amount,
		Status:         domain.OrderStatusPending,
		CreatedAt:      time.Now(),
		IdempotencyKey: idempotencyKey,
	}

	if err := u.repo.Create(order); err != nil {
		if errors.Is(err, repository.ErrDuplicateIdempotencyKey) && idempotencyKey != "" {
			existing, findErr := u.repo.GetByIdempotencyKey(idempotencyKey)
			if findErr == nil {
				return existing, nil
			}
			return nil, findErr
		}
		return nil, err
	}

	paymentResp, err := u.payAPI.Authorize(order.ID, order.Amount)
	if err != nil {
		_ = u.repo.UpdateStatus(order.ID, domain.OrderStatusFailed)
		order.Status = domain.OrderStatusFailed
		return nil, ErrPaymentServiceDown
	}

	switch paymentResp.Status {
	case "Authorized":
		order.Status = domain.OrderStatusPaid
	case "Declined":
		order.Status = domain.OrderStatusFailed
	default:
		order.Status = domain.OrderStatusFailed
	}

	if err := u.repo.UpdateStatus(order.ID, order.Status); err != nil {
		return nil, err
	}

	return order, nil
}

func (u *OrderUsecase) GetOrder(id string) (*domain.Order, error) {
	return u.repo.GetByID(id)
}

func (u *OrderUsecase) GetOrderStats() (*domain.OrderStats, error) {
	return u.repo.Stats()
}

func (u *OrderUsecase) CancelOrder(id string) error {
	order, err := u.repo.GetByID(id)
	if err != nil {
		return err
	}

	if order.Status == domain.OrderStatusPaid {
		return ErrCannotCancelPaidOrder
	}

	if order.Status != domain.OrderStatusPending {
		return ErrCannotCancelNonPendingOrder
	}

	return u.repo.UpdateStatus(id, domain.OrderStatusCancelled)
}
