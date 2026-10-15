# Order Tracking API

A backend order tracking system built with Go, Fiber, and PostgreSQL. This project demonstrates order lifecycle management, payment confirmation, idempotent order creation, background processing, and order event history.

## Features

- **Order Management** — Create and retrieve orders.
- **Order State Machine** — Control valid order status transitions.
- **Idempotency** — Prevent duplicate order creation when a request is retried with the same idempotency key.
- **Payment Confirmation** — Confirm payment before placing an order in the processing queue.
- **Background Worker** — Process queued orders asynchronously.
- **Order Timeline** — Record order status transitions and related events.
- **Issue Reporting** — Record issues associated with orders.
- **Health Checks** — Provide health and readiness endpoints.
- **Automated Tests** — Include unit tests and integration tests.

## Tech Stack

| Component | Technology |
|---|---|
| Language | Go |
| HTTP Framework | Fiber v2 |
| Database | PostgreSQL 16 |
| Database Driver | pgx |
| Containerization | Docker and Docker Compose |
| Testing | Go testing package |

## Project Structure

```text
order-tracking-api/
├── cmd/
│   └── api/
│       └── main.go
├── internal/
│   ├── model/
│   ├── repository/
│   ├── handler/
│   └── ...
├── migrations/
│   ├── 001_create_orders.sql
│   ├── 002_create_order_timeline.sql
│   └── 003_create_order_issues.sql
├── tests/
├── docs/
├── .env.example
├── .gitignore
├── Dockerfile
├── docker-compose.yml
├── go.mod
├── go.sum
└── README.md
```

The internal package structure above is illustrative. Refer to the actual repository for the complete list of packages and files.

## Prerequisites

Install the following before running the project:

- Go version compatible with `go.mod`
- Docker Desktop with Docker Compose
- Git

## Getting Started

### 1. Clone the Repository

```bash
git clone https://github.com/Rafiana219/order-tracking-api.git
cd order-tracking-api
```

### 2. Configure Environment Variables

Create a local environment file from the example:

**PowerShell**

```powershell
Copy-Item .env.example .env
```

The example configuration for local development is:

```dotenv
DB_HOST=127.0.0.1
DB_PORT=5433
DB_USER=order_user
DB_PASSWORD=local_dev_password
DB_NAME=order_tracking
DB_SSLMODE=disable
```

Update `.env` if your local configuration differs.

> **Security:** `.env` is intended for local configuration and must not be committed to Git. The credentials shown here are for local development only and must not be used in a production environment.

### 3. Start PostgreSQL

Start the database container:

```powershell
docker compose up -d
```

The Compose configuration uses PostgreSQL 16 and maps host port `5433` to container port `5432`.

Check the container status:

```powershell
docker compose ps
```

Wait until the PostgreSQL health check reports that the database is ready before applying migrations or starting the API.

### 4. Apply Database Migrations

The SQL migration files must be applied in numerical order:

1. `migrations/001_create_orders.sql`
2. `migrations/002_create_order_timeline.sql`
3. `migrations/003_create_order_issues.sql`

For a new local database, apply each migration using PowerShell:

```powershell
docker cp .\migrations\001_create_orders.sql order-tracking-db:/tmp/001_create_orders.sql
docker exec order-tracking-db psql -U order_user -d order_tracking -f /tmp/001_create_orders.sql

docker cp .\migrations\002_create_order_timeline.sql order-tracking-db:/tmp/002_create_order_timeline.sql
docker exec order-tracking-db psql -U order_user -d order_tracking -f /tmp/002_create_order_timeline.sql

docker cp .\migrations\003_create_order_issues.sql order-tracking-db:/tmp/003_create_order_issues.sql
docker exec order-tracking-db psql -U order_user -d order_tracking -f /tmp/003_create_order_issues.sql
```

These commands assume the Compose container is named `order-tracking-db` and the database and user match the example configuration.

For an existing database, check which migrations have already been applied before executing them again.

### 5. Run the API

From the project root:

```powershell
go run ./cmd/api
```

Alternatively, build and run the executable:

```powershell
go build -o api.exe ./cmd/api
.\api.exe
```

The examples in this README assume the API listens on `http://localhost:8080`. Confirm the actual port in the application configuration if it differs.

### 6. Verify the Service

Check the health endpoint:

```http
GET http://localhost:8080/health
```

Check the readiness endpoint:

```http
GET http://localhost:8080/ready
```

A successful response indicates that the corresponding health or readiness check has passed.

## Order Lifecycle

The system uses order statuses to control the order lifecycle.

```text
PENDING_PAYMENT
       |
       | Payment confirmed
       v
     QUEUED
       |
       | Worker claims order
       v
  IN_PROGRESS
       |
       | Processing succeeds
       v
   COMPLETED
```

Additional statuses include:

- `EXPIRED`
- `FAILED`
- `REFUNDED`

Valid transitions depend on the state machine rules implemented in the application. An invalid transition should be rejected rather than silently changing the order status.

## API Reference

The following endpoints are available in the application.

| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/health` | Check API health |
| `GET` | `/ready` | Check service readiness |
| `POST` | `/api/orders/` | Create an order |
| `GET` | `/api/orders/` | Retrieve the order list |
| `GET` | `/api/orders/:id` | Retrieve an order by ID |
| `POST` | `/api/orders/:id/confirm-payment` | Confirm payment for an order |
| `GET` | `/api/orders/:id/timeline` | Retrieve an order's event history |
| `POST` | `/api/orders/:id/issues` | Report an issue associated with an order |

### Create an Order

**Request**

```http
POST /api/orders/
Content-Type: application/json
Idempotency-Key: example-order-001
```

```json
{
  "customer_id": "customer-001",
  "items": [
    {
      "product_id": "product-001",
      "quantity": 1
    }
  ]
}
```

**Example response**

```json
{
  "data": {
    "id": "generated-order-uuid",
    "customer_id": "customer-001",
    "items": [
      {
        "product_id": "product-001",
        "quantity": 1
      }
    ],
    "status": "PENDING_PAYMENT",
    "attempts": 0,
    "max_attempts": 3,
    "created_at": "2026-01-01T10:00:00Z",
    "updated_at": "2026-01-01T10:00:00Z"
  },
  "meta": {
    "created": true
  }
}
```

The IDs and timestamps above are illustrative. The API generates actual values when an order is created.

### Confirm Payment

```http
POST /api/orders/{id}/confirm-payment
```

Replace `{id}` with an actual order ID.

A successfully confirmed order transitions from `PENDING_PAYMENT` to `QUEUED`, allowing the background worker to process it.

### Retrieve an Order

```http
GET /api/orders/{id}
```

Returns the order identified by the supplied ID.

### Retrieve Order Timeline

```http
GET /api/orders/{id}/timeline
```

Returns recorded events for the specified order, including status changes and processing events.

### Report an Order Issue

```http
POST /api/orders/{id}/issues
Content-Type: application/json
```

The request body must follow the issue fields defined by the application. Refer to the handler implementation or API examples for the exact schema.

For complete request and response examples, see [`docs/API.md`](docs/API.md) if available.

## Idempotency

Order creation supports an `Idempotency-Key` request header.

- Repeating a request with the same key and the same payload should return the existing order instead of creating a duplicate.
- Reusing a key with a different payload should return `409 Conflict`.

This helps prevent duplicate order creation when clients retry requests after timeouts or uncertain responses.

Example:

```http
POST /api/orders/
Idempotency-Key: example-order-001
Content-Type: application/json
```

Use a unique key for each logical order creation request.

## Error Handling

The API uses HTTP status codes to indicate the result of a request. Expected error cases include:

| Status | Meaning |
|---|---|
| `400 Bad Request` | Invalid request or validation failure |
| `404 Not Found` | Requested order does not exist |
| `409 Conflict` | Idempotency conflict or invalid state transition |
| `500 Internal Server Error` | Unexpected server-side error |

The exact response body depends on the endpoint and error condition.

## Testing

Run all Go tests:

```powershell
go test ./...
```

Run the tests package with verbose output:

```powershell
go test ./tests -v
```

The integration tests require a running API and a configured PostgreSQL database. Some tests may be skipped when optional environment variables or test fixtures are not provided.

## API Testing with Postman

Use Postman to test the endpoints and inspect their responses.

Recommended test sequence:

1. Check `GET /health`.
2. Create an order using `POST /api/orders/`.
3. Save the returned order ID.
4. Confirm payment.
5. Retrieve the order and verify its status.
6. Retrieve the order timeline.
7. Test idempotency by repeating a request with the same key.
8. Test invalid requests and invalid status transitions.

If a Postman Collection is included in the repository, import it into Postman and configure its base URL before running the requests.

## Live Demo

Live demo URL: https://legendary-space-meme-5vpv49qpx9w245q7-8080.app.github.dev/

Health Check: https://legendary-space-meme-5vpv49qpx9w245q7-8080.app.github.dev/health

Readiness Check: https://legendary-space-meme-5vpv49qpx9w245q7-8080.app.github.dev/ready

**Note:** This demo is hosted on GitHub Codespaces. The API is accessible while the Codespace is running and the forwarded port is publicly available.

The demo URL should point to a publicly accessible API environment. Do not expose local database credentials, private configuration files, or administrative database access.

## Limitations

- Order processing is demonstrated through a background worker and the processing behavior implemented in this project.
- Payment confirmation represents an application workflow and does not, by itself, establish integration with a real payment provider.
- Deployment availability depends on the environment used to host the API and database.

## License

This project was created as a technical assessment submission.