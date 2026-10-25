# notification-service

User-facing notifications for the ticketing platform. Port **8085**.

## Why it exists

`booking-service` and `payment-service` emit user events (seat locked, booking
confirmed, booking cancelled, payment succeeded). They do not store or render
notifications themselves — they POST them here through the shared client in
`libs/common/notify`, which is fire-and-forget: a notification failure can
never fail a booking or a payment.

## API

| Method | Path | Description |
|---|---|---|
| `GET` | `/health` | Health check (`{"status":"ok","service":"notification-service"}`) |
| `POST` | `/api/v1/notifications` | Store a notification (called by other services) |
| `GET` | `/api/v1/notifications?userId=<uuid>&limit=20` | List a user's notifications, newest first (`limit` max 100) |
| `POST` | `/api/v1/notifications/:id/read` | Mark as read (idempotent) |

Request body for `POST`:

```json
{
  "userId": "9aa31210-fd5f-405a-a528-df10fb1441c2",
  "kind": "booking.confirmed",
  "title": "Tiket dikonfirmasi",
  "body": "Reservasi TICK-1792931922-8089 berstatus CONFIRMED. Total IDR 2150000.",
  "reference": "6b08a0e3-d49f-4327-85ab-5ecd60797d6f"
}
```

`kind` values are listed in `libs/common/notify` and in `contracts/events/`.

## Structure

```text
notification-service/
├── cmd/server/main.go              # wiring: db -> repo -> service -> handler -> routes
├── internal/
│   ├── handler/notification_handler.go
│   ├── service/notification_service.go
│   ├── repository/notification_repository.go
│   ├── model/notification.go
│   └── dto/
├── pkg/database/db.go
├── migrations/001_create_notifications.sql
├── DockerFile                      # built from the backend/ root
├── go.mod
└── README.md
```

Shared building blocks come from `libs/common` (`cors`, `health`, `httperr`).

## Run

```powershell
# part of the stack
docker compose -f backend/docker-compose.yml up -d

# standalone (needs postgres from infrastucture/)
$env:DATABASE_URL = "postgres://admin:tiketid@localhost:5432/event_ticketing_db?sslmode=disable"
go run ./cmd/server
```

Schema is created by GORM `AutoMigrate`; `migrations/` holds the equivalent SQL
for review.
