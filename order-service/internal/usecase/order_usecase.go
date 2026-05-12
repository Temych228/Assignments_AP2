package usecase

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"order-service/internal/cache"
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
	repo     repository.OrderRepository
	payAPI   ports.PaymentClient
	cache    cache.OrderCache
	cacheTTL time.Duration
}

func NewOrderUsecase(
	r repository.OrderRepository,
	payAPI ports.PaymentClient,
	cache cache.OrderCache,
	cacheTTL time.Duration,
) *OrderUsecase {
	return &OrderUsecase{
		repo:     r,
		payAPI:   payAPI,
		cache:    cache,
		cacheTTL: cacheTTL,
	}
}

func (u *OrderUsecase) GetOrder(ctx context.Context, id string) (*domain.Order, error) {
	if u.cache != nil {
		if cached, hit, err := u.cache.Get(ctx, id); err == nil && hit {
			return cached, nil
		}
	}

	order, err := u.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	if u.cache != nil {
		_ = u.cache.Set(ctx, order, u.cacheTTL)
	}

	return order, nil
}

func (u *OrderUsecase) CreateOrder(customerID, customerEmail, itemName string, amount int64, idempotencyKey string) (*domain.Order, error) {
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
		CustomerEmail:  customerEmail,
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

	paymentResp, err := u.payAPI.Authorize(order.ID, order.Amount, order.CustomerEmail)
	if err != nil {
		_ = u.repo.UpdateStatus(order.ID, domain.OrderStatusFailed)
		order.Status = domain.OrderStatusFailed
		if u.cache != nil {
			_ = u.cache.Set(context.Background(), order, u.cacheTTL)
		}
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

	if u.cache != nil {
		_ = u.cache.Set(context.Background(), order, u.cacheTTL)
	}

	return order, nil
}

func (u *OrderUsecase) GetOrderStats() (*domain.OrderStats, error) {
	return u.repo.Stats()
}

func (u *OrderUsecase) SubscribeToOrderUpdates(ctx context.Context, orderID string) (<-chan domain.OrderStatusUpdate, <-chan error) {
	return u.repo.ListenStatusUpdates(ctx, orderID)
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

	if err := u.repo.UpdateStatus(id, domain.OrderStatusCancelled); err != nil {
		return err
	}

	order.Status = domain.OrderStatusCancelled
	if u.cache != nil {
		_ = u.cache.Set(context.Background(), order, u.cacheTTL)
	}

	return nil
}
