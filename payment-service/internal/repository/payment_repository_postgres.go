package repository

import (
	"database/sql"
	"payment-service/internal/domain"
)

type paymentRepository struct {
	db *sql.DB
}

func NewPaymentRepository(db *sql.DB) PaymentRepository {
	return &paymentRepository{db: db}
}

func (r *paymentRepository) Create(p *domain.Payment) error {
	_, err := r.db.Exec(
		"INSERT INTO payments (id, order_id, customer_email, transaction_id, amount, status) VALUES ($1,$2,$3,$4,$5,$6)",
		p.ID, p.OrderID, p.CustomerEmail, p.TransactionID, p.Amount, p.Status,
	)
	return err
}

func (r *paymentRepository) GetByOrderID(orderID string) (*domain.Payment, error) {
	row := r.db.QueryRow(
		"SELECT id, order_id, customer_email, transaction_id, amount, status FROM payments WHERE order_id=$1",
		orderID,
	)

	var p domain.Payment
	err := row.Scan(&p.ID, &p.OrderID, &p.CustomerEmail, &p.TransactionID, &p.Amount, &p.Status)
	if err != nil {
		return nil, err
	}

	return &p, nil
}

func (r *paymentRepository) Stats() (*domain.PaymentStats, error) {
	row := r.db.QueryRow(`
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE status = 'Authorized'),
			COUNT(*) FILTER (WHERE status = 'Declined'),
			COALESCE(SUM(amount) FILTER (WHERE status = 'Authorized'), 0)
		FROM payments
	`)
	var s domain.PaymentStats
	if err := row.Scan(&s.TotalCount, &s.AuthorizedCount, &s.DeclinedCount, &s.TotalAmount); err != nil {
		return nil, err
	}
	return &s, nil
}
