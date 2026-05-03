package repository

import (
	"payment-service/internal/domain"
)

type PaymentRepository interface {
	Create(payment *domain.Payment) error
	GetByOrderID(orderID string) (*domain.Payment, error)
	Stats() (*domain.PaymentStats, error)
}
