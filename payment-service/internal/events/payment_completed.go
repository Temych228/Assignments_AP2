package events

import "time"

type PaymentCompleted struct {
	EventID       string    `json:"event_id"`
	PaymentID     string    `json:"payment_id"`
	OrderID       string    `json:"order_id"`
	Amount        int64     `json:"amount"`
	CustomerEmail string    `json:"customer_email"`
	Status        string    `json:"status"`
	OccurredAt    time.Time `json:"occurred_at"`
}
