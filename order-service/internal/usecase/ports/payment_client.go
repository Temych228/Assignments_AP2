package ports

type PaymentResult struct {
	Status        string
	TransactionID string
}

type PaymentClient interface {
	Authorize(orderID string, amount int64, customerEmail string) (*PaymentResult, error)
}
