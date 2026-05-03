package repository

import (
	"context"

	"order-service/internal/domain"
)

type OrderRepository interface {
	Create(order *domain.Order) error
	GetByID(id string) (*domain.Order, error)
	GetByIdempotencyKey(key string) (*domain.Order, error)
	UpdateStatus(id string, status string) error
	ListenStatusUpdates(ctx context.Context, orderID string) (<-chan domain.OrderStatusUpdate, <-chan error)
	Stats() (*domain.OrderStats, error)
}
