# AP2 Assignment 2 — gRPC & Contract-First Microservices

This project demonstrates a microservices system built with Go using **gRPC for inter-service communication** and a **contract-first development approach**.

The system consists of two independent services:

* **Order Service** — coordinates order processing and communicates with Payment Service via gRPC
* **Payment Service** — processes payments and exposes a gRPC API

The architecture follows **Clean Architecture principles**, keeping business logic independent from transport and infrastructure layers.

---

## Repositories

The project is organized into three logical parts:

### 1. Main project (this repository)

Contains the implementation of both services:

* `order-service`
* `payment-service`

---

### 2. Proto repository

Stores only `.proto` files (service contracts).

You define all gRPC APIs here.

> `(https://github.com/Temych228/ap2-protos.git)`

---

### 3. Generated code repository

Stores generated Go code (`.pb.go`).

* Code is generated automatically via GitHub Actions
* Services import this module as a dependency
* Versioning is done using Git tags (e.g. `v1.0.0`)

> `(https://github.com/Temych228/ap2-protos-go.git)`

---

### Contract-first workflow

1. Define or update `.proto` files in the proto repository
2. Push changes
3. GitHub Actions generates Go code in the generated repo
4. Create a release tag (e.g. `v1.0.0`)
5. Update services:

```bash
go get <your-generated-module>@v1.0.0
```

---

## Architecture Overview

### Order Service

* Handles order lifecycle
* Calls Payment Service via gRPC
* Streams order updates to clients
* Owns its own database

### Payment Service

* Exposes gRPC API
* Processes payments
* Applies validation rules
* Uses gRPC interceptor for logging
* Owns its own database

---

## Project Structure

Both services follow the same structure:

```
cmd/                  # entry point
internal/app/         # dependency wiring
internal/domain/      # business entities
internal/usecase/     # core logic
internal/repository/  # database layer
internal/transport/
    grpc/             # gRPC handlers
    http/             # (only in order-service if needed)
internal/clients/     # gRPC clients
migrations/           # SQL files
```

---

## Business Rules

* Monetary values are stored as `int64` (cents)
* Order amount must be greater than zero
* Orders start in `Pending` state
* Payment above limit → `Declined`
* Services are isolated (no shared DB or models)

---

## gRPC API

### Payment Service

* `ProcessPayment(PaymentRequest) returns (PaymentResponse)`

### Order Service

* `SubscribeToOrderUpdates(OrderRequest) returns (stream OrderStatusUpdate)`

---

## PostgreSQL LISTEN / NOTIFY

Order updates are delivered in real time using PostgreSQL:

* Database emits `NOTIFY` on status changes
* Service subscribes using `LISTEN`
* Updates are pushed to clients via gRPC stream

This eliminates polling and ensures immediate delivery of updates.

---

## Environment Variables

Create `.env` files manually.

### order-service/.env

```
DB_URL=<your-order-db-url>
GRPC_ADDR=:9090
PAYMENT_GRPC_ADDR=localhost:50051
```

### payment-service/.env

```
DB_URL=<your-payment-db-url>
GRPC_ADDR=:50051
```

---

## Database Schema

### Orders

```sql
CREATE TABLE orders (
    id TEXT PRIMARY KEY,
    customer_id TEXT NOT NULL,
    item_name TEXT NOT NULL,
    amount BIGINT NOT NULL,
    status TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    idempotency_key TEXT UNIQUE
);
```

### Payments

```sql
CREATE TABLE payments (
    id TEXT PRIMARY KEY,
    order_id TEXT,
    transaction_id TEXT,
    amount BIGINT,
    status TEXT
);
```

---

## How to Run

### 1. Clone project

```bash
git clone <your-main-repo>
cd <project-folder>
```

---

### 2. Setup databases

Create two databases:

* `order_db`
* `payment_db`

Apply migrations manually.

---

### 3. Install dependencies

```bash
go mod tidy
```

---

### 4. Run Payment Service

```bash
cd payment-service
go run ./cmd/payment-service/main.go
```

---

### 5. Run Order Service

```bash
cd order-service
go run ./cmd/order-service/main.go
```

---

## Example Request

```bash
curl.exe -X POST http://localhost:8080/orders ^
  -H "Content-Type: application/json" ^
  -d "{\"customer_id\":\"c1\",\"item_name\":\"Laptop\",\"amount\":50000}"
```

---

## gRPC Flow

1. Order is created
2. Order Service sends gRPC request to Payment Service
3. Payment is processed
4. Order status is updated

---

## Streaming

Order Service provides real-time updates:

```
SubscribeToOrderUpdates → stream
```

* client subscribes once
* receives updates continuously
* backed by PostgreSQL events

---

## Interceptor

Payment Service includes a gRPC interceptor that logs:

* method name
* execution time
* result status
