package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"order-service/internal/domain"

	"github.com/lib/pq"
)

var ErrDuplicateIdempotencyKey = errors.New("duplicate idempotency key")

type orderRepository struct {
	db    *sql.DB
	dbURL string
}

func NewOrderRepository(db *sql.DB, dbURL string) OrderRepository {
	return &orderRepository{db: db, dbURL: dbURL}
}

func (r *orderRepository) Create(o *domain.Order) error {
	var idem interface{}
	if o.IdempotencyKey != "" {
		idem = o.IdempotencyKey
	} else {
		idem = nil
	}

	_, err := r.db.Exec(
		`INSERT INTO orders (id, customer_id, customer_email, item_name, amount, status, created_at, idempotency_key)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		o.ID, o.CustomerID, o.CustomerEmail, o.ItemName, o.Amount, o.Status, o.CreatedAt, idem,
	)
	if err == nil {
		return nil
	}

	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == "23505" {
		return ErrDuplicateIdempotencyKey
	}

	return err
}

func (r *orderRepository) GetByID(id string) (*domain.Order, error) {
	row := r.db.QueryRow(
		`SELECT id, customer_id, customer_email, item_name, amount, status, created_at, COALESCE(idempotency_key, '')
		 FROM orders WHERE id=$1`,
		id,
	)

	var o domain.Order
	err := row.Scan(&o.ID, &o.CustomerID, &o.CustomerEmail, &o.ItemName, &o.Amount, &o.Status, &o.CreatedAt, &o.IdempotencyKey)
	if err != nil {
		return nil, err
	}

	return &o, nil
}

func (r *orderRepository) GetByIdempotencyKey(key string) (*domain.Order, error) {
	row := r.db.QueryRow(
		`SELECT id, customer_id, customer_email, item_name, amount, status, created_at, COALESCE(idempotency_key, '')
		 FROM orders WHERE idempotency_key=$1`,
		key,
	)

	var o domain.Order
	err := row.Scan(&o.ID, &o.CustomerID, &o.CustomerEmail, &o.ItemName, &o.Amount, &o.Status, &o.CreatedAt, &o.IdempotencyKey)
	if err != nil {
		return nil, err
	}

	return &o, nil
}

func (r *orderRepository) UpdateStatus(id, status string) error {
	_, err := r.db.Exec("UPDATE orders SET status=$1 WHERE id=$2", status, id)
	return err
}

func (r *orderRepository) ListenStatusUpdates(ctx context.Context, orderID string) (<-chan domain.OrderStatusUpdate, <-chan error) {
	updates := make(chan domain.OrderStatusUpdate)
	errs := make(chan error, 1)

	go func() {
		defer close(updates)
		defer close(errs)

		listener := pq.NewListener(r.dbURL, 10*time.Second, time.Minute, func(event pq.ListenerEventType, err error) {
			if err != nil {
				select {
				case errs <- err:
				default:
				}
			}
		})
		defer listener.Close()

		if err := listener.Listen("order_status_updates"); err != nil {
			errs <- err
			return
		}

		for {
			select {
			case <-ctx.Done():
				return
			case notification := <-listener.Notify:
				if notification == nil {
					continue
				}

				var update domain.OrderStatusUpdate
				if err := json.Unmarshal([]byte(notification.Extra), &update); err != nil {
					errs <- err
					continue
				}
				if update.OrderID != orderID {
					continue
				}

				select {
				case updates <- update:
				case <-ctx.Done():
					return
				}
			case <-time.After(30 * time.Second):
				if err := listener.Ping(); err != nil {
					errs <- err
				}
			}
		}
	}()

	return updates, errs
}

func (r *orderRepository) Stats() (*domain.OrderStats, error) {
	row := r.db.QueryRow(`
		SELECT
			COUNT(*) AS total,
			COALESCE(SUM(CASE WHEN status = 'Pending' THEN 1 ELSE 0 END), 0) AS pending,
			COALESCE(SUM(CASE WHEN status = 'Paid' THEN 1 ELSE 0 END), 0) AS paid,
			COALESCE(SUM(CASE WHEN status = 'Failed' THEN 1 ELSE 0 END), 0) AS failed,
			COALESCE(SUM(CASE WHEN status = 'Cancelled' THEN 1 ELSE 0 END), 0) AS cancelled
		FROM orders
	`)

	var s domain.OrderStats
	if err := row.Scan(&s.Total, &s.Pending, &s.Paid, &s.Failed, &s.Cancelled); err != nil {
		return nil, err
	}

	return &s, nil
}
