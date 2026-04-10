# Assignment 1 — Clean Architecture Based Microservices (Order & Payment)

This project is a **microservices-based platform** developed for the *Advanced Programming 2* course. It consists of two independent services — **Order Service** and **Payment Service** — built using **Go** and the **Gin HTTP framework**. The system is designed following **Clean Architecture** principles to ensure strict separation of concerns, clear service boundaries, and resilient communication.

---

## Architecture and Project Structure

Each service is self-contained and follows a standardized layered structure. This design ensures that business logic remains independent of external frameworks, databases, and transport layers.

* **`cmd/`**: Contains the composition root and application entry point where manual dependency injection occurs.
* **`internal/domain/`**: Contains pure domain models and entities, decoupled from JSON tags, database logic, and HTTP concerns.
* **`internal/usecase/`**: Implements core business logic and interacts with domain entities through interfaces (Ports).
* **`internal/repository/`**: Handles persistence logic and direct communication with PostgreSQL.
* **`internal/transport/http/`**: Manages Gin handlers, request parsing, and response formatting.
* **`internal/clients/`**: (Order Service only) Contains the custom HTTP client used for synchronous communication with the Payment Service.
* **`migrations/`**: Holds SQL scripts for database schema setup.

---

## Bounded Contexts and Data Ownership

The system strictly adheres to **microservice isolation rules**. There is no shared code, common packages, or shared entities between the services. Each service owns its **dedicated PostgreSQL database** (`order_db` and `payment_db`), preventing a distributed monolith scenario. Communication is strictly **REST-based**.

---

## Business Rules and Implementation

The implementation satisfies all functional requirements and constraints specified in the assignment:

* **Financial Accuracy**: All monetary values are handled using `int64` (representing cents) to avoid floating-point precision errors.
* **Order Invariants**: An order amount must be greater than zero. Orders can only be cancelled if their status is `"Pending"`; once an order is `"Paid"`, cancellation is prohibited.
* **Payment Limits**: The Payment Service enforces a transaction limit. Any payment request exceeding `100,000` units is automatically `"Declined"`.
* **Idempotency**: The `POST /orders` endpoint supports an `Idempotency-Key` header, ensuring retried requests do not result in duplicate orders or multiple payment authorizations.

---

## Failure Handling and Resilience

Resilience is a core component of inter-service communication. The Order Service uses a custom `http.Client` with a **2-second timeout** when calling the Payment Service.

If the Payment Service is unavailable or the request times out:

* The Order Service returns a **503 Service Unavailable** error to the client.
* The order is marked as **Failed** in the database to maintain state consistency and prevent hanging processes.

---

## API Endpoints

### Order Service (Port 8080)

* `POST /orders`: Creates a pending order and initiates payment.
* `GET /orders/{id}`: Retrieves order details and current status.
* `PATCH /orders/{id}/cancel`: Cancels an order (if allowed by business rules).

### Payment Service (Port 8081)

* `POST /payments`: Validates and processes payment requests.
* `GET /payments/{order_id}`: Retrieves payment status for a specific order.

---

## Testing

The system can be tested using `curl`. Example of creating an order:

```bash
curl -X POST http://localhost:8080/orders \
-H "Content-Type: application/json" \
-H "Idempotency-Key: unique-request-id-001" \
-d '{"customer_id": "user123", "item_name": "Laptop", "amount": 50000}'
```
