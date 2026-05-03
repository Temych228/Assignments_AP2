package grpc

import (
	"context"
	"errors"

	paymentv1 "github.com/Temych228/ap2-protos-go/payment/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"payment-service/internal/domain"
	"payment-service/internal/usecase"
)

type PaymentServer struct {
	paymentv1.UnimplementedPaymentServiceServer
	usecase *usecase.PaymentUsecase
}

func NewPaymentServer(u *usecase.PaymentUsecase) *PaymentServer {
	return &PaymentServer{usecase: u}
}

func (s *PaymentServer) ProcessPayment(ctx context.Context, req *paymentv1.PaymentRequest) (*paymentv1.PaymentResponse, error) {
	if req.GetAmount() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "amount must be > 0")
	}

	customerEmail := ""
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		values := md.Get("customer-email")
		if len(values) > 0 {
			customerEmail = values[0]
		}
	}

	payment, err := s.usecase.ProcessPayment(ctx, req.GetOrderId(), req.GetAmount(), customerEmail)
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrAmountMustBePositive):
			return nil, status.Error(codes.InvalidArgument, err.Error())
		default:
			return nil, status.Error(codes.Internal, "failed to process payment")
		}
	}

	return toProtoPaymentResponse(payment), nil
}

func toProtoPaymentResponse(p *domain.Payment) *paymentv1.PaymentResponse {
	return &paymentv1.PaymentResponse{
		Status:        p.Status,
		TransactionId: p.TransactionID,
	}
}

func (s *PaymentServer) GetPaymentStats(ctx context.Context, _ *paymentv1.GetPaymentStatsRequest) (*paymentv1.PaymentStats, error) {
	stats, err := s.usecase.GetStats()
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get payment stats")
	}
	return &paymentv1.PaymentStats{
		TotalCount:      stats.TotalCount,
		AuthorizedCount: stats.AuthorizedCount,
		DeclinedCount:   stats.DeclinedCount,
		TotalAmount:     stats.TotalAmount,
	}, nil
}
