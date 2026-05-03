# AP2 Assignment 3 — EDA with Message Queues

Microservices system in Go using **gRPC** for inter-service communication and **RabbitMQ** for async event-driven notifications.

---

## Repositories

| Repo | Purpose |
|------|---------|
| **this repo** | Order, Payment, Notification services |
| [ap2-protos](https://github.com/Temych228/ap2-protos) | `.proto` contract definitions |
| [ap2-protos-go](https://github.com/Temych228/ap2-protos-go) | Auto-generated Go code (imported as dependency) |

> **No proto files live in this repo.** Services import generated code via `go.mod`.

---

## Architecture

```
HTTP Client
    │
    ▼
Order Service ──gRPC──► Payment Service
    │  (9090)               │  (50051)
    │                       │
    │                       ▼
    │                  RabbitMQ
    │               (payment.completed)
    │                       │
    ▼                       ▼
Order DB            Notification Service
                    (consumer + gRPC client → Order Service)
```

**Flow:**
1. `POST /orders` → Order Service creates order → calls Payment Service via gRPC
2. Payment Service processes payment → publishes `payment.completed` event to RabbitMQ
3. Notification Service consumes event → logs email simulation
4. Notification Service also connects to Order Service gRPC stream for real-time status updates

---

## How to run

```bash
docker compose up --build
```

All services, databases, and RabbitMQ start automatically.

---

## How to update proto contracts

When you need to add/change a gRPC method:

**1. Edit `.proto` in the proto repo and push:**
```bash
# in ap2-protos repo
git add payment/v1/payment.proto
git commit -m "feat: add CancelPayment rpc"
git push
# GitHub Actions auto-generates Go code and creates a new tag (e.g. v1.0.3)
```

**2. Update the dependency in the affected service(s):**
```bash
# in this repo, inside the service directory
cd payment-service
go get github.com/Temych228/ap2-protos-go@latest
go mod tidy
cd ..

# same for order-service or notification-service if they use the changed proto
```

**3. Implement the new RPC handler and push.**

---

## Environment variables

### order-service
| Variable | Default | Description |
|----------|---------|-------------|
| `DB_URL` | — | Postgres connection string |
| `GRPC_ADDR` | `:9090` | gRPC listen address |
| `PAYMENT_GRPC_ADDR` | `localhost:50051` | Payment service address |

### payment-service
| Variable | Default | Description |
|----------|---------|-------------|
| `DB_URL` | — | Postgres connection string |
| `GRPC_ADDR` | `:50051` | gRPC listen address |
| `RABBITMQ_URL` | `amqp://guest:guest@localhost:5672/` | RabbitMQ connection |

### notification-service
| Variable | Default | Description |
|----------|---------|-------------|
| `RABBITMQ_URL` | `amqp://guest:guest@localhost:5672/` | RabbitMQ connection |
| `ORDER_GRPC_ADDR` | — | Order service gRPC address (optional) |
| `SUBSCRIBE_ORDER_ID` | — | Order UUID to subscribe to via gRPC stream (for demo) |
| `FAIL_CUSTOMER_EMAIL` | — | Email that simulates processing failure → triggers DLQ |

---

## Reliability & EDA design

| Feature | Implementation |
|---------|---------------|
| **Manual ACKs** | `ch.Qos(1,0,false)` + explicit `delivery.Ack/Nack` after log |
| **Durable queues** | `x-queue-type: quorum` — survives broker restart |
| **Idempotency** | In-memory `seen map[string]struct{}` keyed on `event_id` |
| **Publisher confirms** | `ch.Confirm` + wait on `NotifyPublish` channel before returning |
| **Dead Letter Queue** | `x-delivery-limit: 3` → failed messages move to `payment.completed.dlq` |
| **Graceful shutdown** | `os/signal.NotifyContext` + `grpcServer.GracefulStop()` |

### Simulate DLQ
Set `FAIL_CUSTOMER_EMAIL=dlq@example.com` in docker-compose (already set).  
Any order with that email will be Nack'd 3 times then appear in `payment.completed.dlq`.  
Check it in RabbitMQ Management UI → http://localhost:15672 (guest/guest).

---

## gRPC APIs

### PaymentService (payment-service:50051)
```protobuf
rpc ProcessPayment(PaymentRequest) returns (PaymentResponse)
rpc GetPaymentStats(GetPaymentStatsRequest) returns (PaymentStats)
```

### OrderService (order-service:9090)
```protobuf
rpc SubscribeToOrderUpdates(OrderRequest) returns (stream OrderStatusUpdate)
```

---

## HTTP API (order-service:8080)

```bash
# Create order
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{"customer_id":"c1","customer_email":"user@example.com","item_name":"Laptop","amount":50000}'

# Get order
curl http://localhost:8080/orders/{id}

# Get stats
curl http://localhost:8080/orders/stats

# Cancel order
curl -X PATCH http://localhost:8080/orders/{id}/cancel
```
