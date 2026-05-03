package clients

import (
	"context"
	"io"
	"log"
	"time"

	orderv1 "github.com/Temych228/ap2-protos-go/order/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// OrderClientGRPC subscribes to order status updates from the Order Service
// via the streaming gRPC API. It is used by Notification Service to log
// order state transitions independently of the RabbitMQ consumer path.
type OrderClientGRPC struct {
	conn   *grpc.ClientConn
	client orderv1.OrderServiceClient
}

func NewOrderClientGRPC(addr string) (*OrderClientGRPC, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &OrderClientGRPC{
		conn:   conn,
		client: orderv1.NewOrderServiceClient(conn),
	}, nil
}

// SubscribeToOrderUpdates opens a server-side stream and logs every status
// change for the given orderID until ctx is cancelled or the stream ends.
func (c *OrderClientGRPC) SubscribeToOrderUpdates(ctx context.Context, orderID string) error {
	stream, err := c.client.SubscribeToOrderUpdates(ctx, &orderv1.OrderRequest{OrderId: orderID})
	if err != nil {
		return err
	}
	for {
		update, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		log.Printf(
			"[Notification][gRPC] Order #%s status changed → %s at %s",
			update.GetOrderId(),
			update.GetStatus(),
			update.GetUpdatedAt().AsTime().Format(time.RFC3339),
		)
	}
}

func (c *OrderClientGRPC) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
