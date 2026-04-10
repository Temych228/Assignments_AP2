package clients

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"order-service/internal/usecase/ports"
)

type PaymentClientHTTP struct {
	client  *http.Client
	baseURL string
}

func NewPaymentClientHTTP(baseURL string) *PaymentClientHTTP {
	return &PaymentClientHTTP{
		client: &http.Client{
			Timeout: 2 * time.Second,
		},
		baseURL: baseURL,
	}
}

type paymentRequest struct {
	OrderID string `json:"order_id"`
	Amount  int64  `json:"amount"`
}

type paymentResponse struct {
	Status        string `json:"status"`
	TransactionID string `json:"transaction_id"`
}

func (p *PaymentClientHTTP) Authorize(orderID string, amount int64) (*ports.PaymentResult, error) {
	reqBody, err := json.Marshal(paymentRequest{
		OrderID: orderID,
		Amount:  amount,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, p.baseURL+"/payments", bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("payment service returned status %d", resp.StatusCode)
	}

	var decoded paymentResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, err
	}

	return &ports.PaymentResult{
		Status:        decoded.Status,
		TransactionID: decoded.TransactionID,
	}, nil
}
