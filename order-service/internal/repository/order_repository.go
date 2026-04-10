package repository

import "order-service/internal/domain"

type OrderRepository interface {
	Create(order *domain.Order) error
	GetByID(id string) (*domain.Order, error)
	GetByIdempotencyKey(key string) (*domain.Order, error)
	UpdateStatus(id string, status string) error
	Stats() (*domain.OrderStats, error)
}
