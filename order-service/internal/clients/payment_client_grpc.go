package clients

import (
	"context"
	"time"

	"order-service/internal/usecase/ports"

	paymentv1 "github.com/Temych228/ap2-protos-go/payment/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

type PaymentClientGRPC struct {
	conn    *grpc.ClientConn
	client  paymentv1.PaymentServiceClient
	timeout time.Duration
}

func NewPaymentClientGRPC(addr string) (*PaymentClientGRPC, error) {
	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	return &PaymentClientGRPC{
		conn:    conn,
		client:  paymentv1.NewPaymentServiceClient(conn),
		timeout: 2 * time.Second,
	}, nil
}

func (p *PaymentClientGRPC) Close() error {
	if p.conn != nil {
		return p.conn.Close()
	}
	return nil
}

func (p *PaymentClientGRPC) Authorize(orderID string, amount int64, customerEmail string) (*ports.PaymentResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), p.timeout)
	defer cancel()
	if customerEmail != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, "customer-email", customerEmail)
	}

	resp, err := p.client.ProcessPayment(ctx, &paymentv1.PaymentRequest{
		OrderId: orderID,
		Amount:  amount,
	})
	if err != nil {
		return nil, err
	}

	return &ports.PaymentResult{
		Status:        resp.GetStatus(),
		TransactionID: resp.GetTransactionId(),
	}, nil
}
