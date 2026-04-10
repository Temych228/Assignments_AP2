package usecase

import (
	"payment-service/internal/domain"
	"payment-service/internal/repository"

	"github.com/google/uuid"
)

type PaymentUsecase struct {
	repo repository.PaymentRepository
}

func NewPaymentUsecase(r repository.PaymentRepository) *PaymentUsecase {
	return &PaymentUsecase{repo: r}
}

func (u *PaymentUsecase) ProcessPayment(orderID string, amount int64) (*domain.Payment, error) {

	status := "Authorized"
	if amount > 100000 {
		status = "Declined"
	}

	payment := &domain.Payment{
		ID:            uuid.New().String(),
		OrderID:       orderID,
		TransactionID: uuid.New().String(),
		Amount:        amount,
		Status:        status,
	}

	err := u.repo.Create(payment)
	if err != nil {
		return nil, err
	}

	return payment, nil
}

func (u *PaymentUsecase) GetPayment(orderID string) (*domain.Payment, error) {
	return u.repo.GetByOrderID(orderID)
}
