package ports

type PaymentResult struct {
	Status        string
	TransactionID string
}

type PaymentClient interface {
	Authorize(orderID string, amount int64) (*PaymentResult, error)
}
