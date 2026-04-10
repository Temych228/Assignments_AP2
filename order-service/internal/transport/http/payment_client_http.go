package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"order-service/internal/usecase/ports"
)

type PaymentClientHTTP struct {
	client *http.Client
	url    string
}

func NewPaymentClientHTTP(url string) *PaymentClientHTTP {
	return &PaymentClientHTTP{
		client: &http.Client{Timeout: 2 * time.Second},
		url:    url,
	}
}

func (p *PaymentClientHTTP) Authorize(orderID string, amount int64) (*ports.PaymentResult, error) {
	body, _ := json.Marshal(map[string]interface{}{
		"order_id": orderID,
		"amount":   amount,
	})

	resp, err := p.client.Post(p.url+"/payments", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result ports.PaymentResult
	var respMap map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&respMap)

	status, ok := respMap["status"].(string)
	if !ok {
		return nil, err
	}

	transactionID, _ := respMap["transaction_id"].(string)

	result.Status = status
	result.TransactionID = transactionID

	return &result, nil
}
