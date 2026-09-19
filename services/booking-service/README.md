# Booking & Inventory Service (`booking-service`)

Microservice for managing atomic seat locking, reservations, and inventory consistency for the TickId platform.

## Features

- **Zero Overselling**: 2-tier concurrency control (Redis `SET NX EX 300` + PostgreSQL transactional checks).
- **Multi-Seat Atomic Locking**: Batch lock acquisition with immediate automatic rollback on failure.
- **Auto-Release Sweeper**: Background goroutine purging expired `LOCKED` reservations every 30 seconds.
- **Graceful Shutdown**: Intercepts `SIGINT`/`SIGTERM` to safely terminate workers and in-flight requests.

## Endpoints

| Method | Path | Description |
|---|---|---|
| `GET` | `/health` | Service health status |
| `POST` | `/api/v1/reservations` | Reserve one or more seats (locks for 5 min) |
| `GET` | `/api/v1/reservations/:id` | Fetch reservation details |
| `POST` | `/api/v1/reservations/:id/confirm` | Confirm reservation after payment |
| `POST` | `/api/v1/reservations/:id/cancel` | Cancel reservation and release locks |

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `SERVER_PORT` | `8083` | HTTP server port |
| `DATABASE_URL` | `postgres://admin:tiketid@localhost:5432/event_ticketing_db?sslmode=disable` | PostgreSQL connection string |
| `REDIS_URL` | `localhost:6379` | Redis server address |
| `LOCK_TTL_MINUTES` | `5` | Reservation lock duration in minutes |
| `SWEEPER_INTERVAL_SECONDS` | `30` | Interval for background sweeper |
