package grpc

import (
	"errors"

	orderv1 "github.com/Temych228/ap2-protos-go/order/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"order-service/internal/usecase"
)

type OrderServer struct {
	orderv1.UnimplementedOrderServiceServer
	usecase *usecase.OrderUsecase
}

func NewOrderServer(u *usecase.OrderUsecase) *OrderServer {
	return &OrderServer{usecase: u}
}

func (s *OrderServer) SubscribeToOrderUpdates(req *orderv1.OrderRequest, stream orderv1.OrderService_SubscribeToOrderUpdatesServer) error {
	orderID := req.GetOrderId()
	if orderID == "" {
		return status.Error(codes.InvalidArgument, "order_id is required")
	}

	order, err := s.usecase.GetOrder(orderID)
	if err != nil {
		return status.Error(codes.NotFound, "order not found")
	}

	if err := stream.Send(&orderv1.OrderStatusUpdate{
		OrderId:   order.ID,
		Status:    order.Status,
		UpdatedAt: timestamppb.New(order.CreatedAt),
	}); err != nil {
		return err
	}

	updates, errs := s.usecase.SubscribeToOrderUpdates(stream.Context(), orderID)
	for {
		select {
		case <-stream.Context().Done():
			return nil
		case err, ok := <-errs:
			if !ok {
				errs = nil
				continue
			}
			if err != nil && !errors.Is(err, stream.Context().Err()) {
				return status.Error(codes.Internal, err.Error())
			}
		case update, ok := <-updates:
			if !ok {
				return nil
			}
			if err := stream.Send(&orderv1.OrderStatusUpdate{
				OrderId:   update.OrderID,
				Status:    update.Status,
				UpdatedAt: timestamppb.New(update.UpdatedAt),
			}); err != nil {
				return err
			}
		}
	}
}
