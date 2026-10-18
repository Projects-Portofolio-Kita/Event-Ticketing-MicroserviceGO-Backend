# TickId Notification Service (`notification-service`)

[![Go Version](https://img.shields.io/badge/go-1.27-blue.svg)](https://golang.org)
[![Framework](https://img.shields.io/badge/framework-Gin-green.svg)](https://github.com/gin-gonic/gin)
[![Cost](https://img.shields.io/badge/cost-$0%20free-brightgreen.svg)](#free-cost-guarantee)
[![Architecture](https://img.shields.io/badge/architecture-Clean%20Architecture-orange.svg)](#architecture-overview)

The **TickId Notification Service** is a zero-cost, high-performance transactional email and notification microservice designed for the TickId event ticketing platform. It provides asynchronous email dispatch, responsive mobile-friendly HTML templates, high-resolution E-Ticket PDF generation with cryptographic verification QR codes, and dual-mode standard SMTP delivery compatible with free personal Google Accounts (Gmail App Passwords) and local Mailpit containers.

---

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Project Layout](#project-layout)
3. [Zero-Cost Free Delivery Options](#zero-cost-free-delivery-options)
   - [Option A: Local Development with Mailpit (Zero Setup, 100% Free)](#option-a-local-development-with-mailpit-zero-setup-100-free)
   - [Option B: Real Email Delivery via Free Gmail SMTP (Google App Password)](#option-b-real-email-delivery-via-free-gmail-smtp-google-app-password)
4. [API Endpoints Reference](#api-endpoints-reference)
   - [1. Ticket Confirmation & PDF E-Ticket](#1-ticket-confirmation--pdf-e-ticket)
   - [2. Payment Failed / Reservation Expired](#2-payment-failed--reservation-expired)
   - [3. Send Custom / Broadcast Email](#3-send-custom--broadcast-email)
   - [4. Operational Health Check](#4-operational-health-check)
5. [Environment Variables Reference](#environment-variables-reference)
6. [Testing & Verification](#testing--verification)
7. [Building and Running the Service](#building-and-running-the-service)

---

## Architecture Overview

The Notification Service is organized strictly according to **Clean Architecture** principles in Go 1.27:

```
                  ┌────────────────────────────────────────┐
                  │          HTTP Transport Layer          │
                  │        (internal/handler - Gin)        │
                  └───────────────────┬────────────────────┘
                                      │ calls
                                      ▼
                  ┌────────────────────────────────────────┐
                  │       Notification Domain Service      │
                  │           (internal/service)           │
                  └─────────┬────────────────────┬─────────┘
                            │                    │
              submits Job   │                    │ coordinates
                            ▼                    ▼
             ┌─────────────────────────┐  ┌─────────────────────────┐
             │   Worker Pool Queue     │  │   Template & PDF Engines│
             │      (pkg/queue)        │  │(internal/template & pdf)│
             └─────────────┬───────────┘  └─────────────────────────┘
                           │ executes async
                           ▼
             ┌─────────────────────────┐
             │    Dual-Mode SMTP       │
             │       (pkg/smtp)        │
             └─────────────┬───────────┘
                           │ delivers
                           ▼
             ┌─────────────────────────┐
             │ Gmail SMTP (Port 587)   │
             │ or Mailpit (Port 1025)  │
             └─────────────────────────┘
```

- **Non-blocking Dispatch**: Upstream HTTP requests receive an immediate `202 Accepted` response with a unique UUID `job_id`.
- **Concurrent Worker Pool**: Background worker goroutines process queued email tasks with graceful shutdown and draining guarantees on `SIGINT`/`SIGTERM`.
- **In-Memory E-Ticket Engine**: Generates A4 PDF documents with embedded TrueType fonts (`Go-Regular`, `Go-Bold`) via `//go:embed` and 256x256 PNG QR codes via `skip2/go-qrcode` with zero temporary disk files.
- **Responsive HTML Templates**: Mobile-first responsive HTML email layouts with embedded base64 QR gate passes and clean plain-text fallback alternatives for text-only clients.
- **Dual-Mode SMTP**: Seamlessly connects to TLS/STARTTLS authenticated SMTP servers (Gmail on port 587) or unencrypted local mail catchers (Mailpit on port 1025).

---

## Project Layout

```
d:\KODINGANNN\TickId\backend\services\notification-service\
├── cmd/
│   └── server/
│       └── main.go                    # Bootstrap, DI wiring, graceful shutdown (SIGINT/SIGTERM)
├── internal/
│   ├── config/
│   │   ├── config.go                  # 12-factor env loader (SERVER_PORT, SMTP_*, WORKER_*)
│   │   └── config_test.go             # Env loader unit tests
│   ├── dto/
│   │   ├── notification_dto.go        # TicketConfirmationRequest, PaymentFailedRequest, CustomEmailRequest
│   │   ├── common_dto.go              # BaseResponse[T], APIResponse, QueuedResponse
│   │   └── dto_test.go                # DTO serialization and validation tests
│   ├── pdf/
│   │   ├── generator.go               # In-memory branded E-Ticket PDF generator (gopdf)
│   │   ├── qr.go                      # QR code generator (skip2/go-qrcode) with verification hash
│   │   ├── font.go                    # Embedded TrueType fonts via //go:embed
│   │   └── pdf_test.go                # PDF & QR unit tests (PNG magic, %PDF- header check)
│   ├── template/
│   │   ├── engine.go                  # Template rendering engine with HTML/Text fallbacks
│   │   ├── order_confirmation.go      # Order confirmation HTML/text template
│   │   ├── payment_failed.go          # Payment failed / reservation expired template
│   │   ├── event_notice.go            # Event change / cancellation advisory template
│   │   └── template_test.go           # Template rendering & XSS escaping unit tests
│   ├── service/
│   │   ├── notification_service.go    # Business orchestrator coordinating PDF, QR, template, and queue
│   │   └── service_test.go            # Business logic and offline mock SMTP tests
│   └── handler/
│       ├── notification_handler.go    # Gin HTTP handlers returning immediate 202 Accepted
│       ├── health_handler.go          # GET /health endpoint
│       ├── routes.go                  # Gin route registrations and middleware
│       └── handler_test.go            # HTTP integration tests via httptest
├── pkg/
│   ├── smtp/
│   │   ├── client.go                  # Dual-mode SMTP client (TLS/STARTTLS & Plain)
│   │   ├── mock_smtp.go               # In-memory mock RFC-5321 SMTP server for offline test suites
│   │   └── smtp_test.go               # SMTP client & mock server tests
│   └── queue/
│       ├── worker_pool.go             # Buffered Go channel worker pool with graceful drain
│       └── queue_test.go              # Concurrency and graceful shutdown tests
├── .env.example                       # Detailed Gmail App Password guide & Mailpit config
├── README.md                          # Full service documentation & run instructions
├── go.mod                             # Module definition (Go 1.27)
└── go.sum                             # Verified dependency checksums
```

---

## Zero-Cost Free Delivery Options

The TickId Notification Service does **not** require paid third-party email APIs (SendGrid, Mailgun, Amazon SES). Instead, it operates at **$0 recurring cost** using two primary delivery options:

| Option | Ideal For | Key Benefit | Setup Time |
|---|---|---|---|
| **[Option A: Mailpit](#option-a-local-development-with-mailpit-zero-setup-100-free)** | Local development & offline testing | Zero credentials, web UI inspector at `localhost:8025` | Instant (< 1 min) |
| **[Option B: Gmail SMTP](#option-b-real-email-delivery-via-free-gmail-smtp-google-app-password)** | Staging, QA, and real customer emails | Sends real emails directly to inboxes (500 free/day) | 2 minutes |

---

### Option A: Local Development with Mailpit (Zero Setup, 100% Free)

Mailpit captures all outgoing emails locally inside Docker without sending anything over the internet. You can view rendered HTML emails and download generated PDF E-Tickets in real time via a web interface.

#### 1. Start Mailpit via Docker
Using Docker Compose from `backend/`:
```bash
docker compose up -d mailpit
```
Or standalone:
```bash
docker run -d --name tickid-mailpit -p 1025:1025 -p 8025:8025 axllent/mailpit:latest
```

#### 2. Configure `.env` for Mailpit
Create or update `.env` in `services/notification-service/`:
```env
SERVER_PORT=8085
WORKER_COUNT=5
QUEUE_CAPACITY=500

SMTP_HOST=localhost
SMTP_PORT=1025
SMTP_USER=
SMTP_PASS=
SMTP_FROM=noreply@tickid.com
SMTP_FROM_NAME=TickId Local Dev
SMTP_USE_TLS=false
```

#### 3. Inspect Outgoing Emails in Browser
Open **[http://localhost:8025](http://localhost:8025)** to view sent emails, inspect headers, check responsive layouts, and download the attached PDF E-Tickets.

---

### Option B: Real Email Delivery via Free Gmail SMTP (Google App Password)

Google provides **up to 500 free transactional emails per day** for personal Gmail accounts via standard SMTP using an App Password (no credit card or paid subscription needed).

#### 9-Step Tutorial: Generating Your Free Google App Password

1. **Log In to Google Account**: Navigate to [https://myaccount.google.com/](https://myaccount.google.com/) and sign in.
2. **Navigate to Security**: In the left menu, select **Security** (Keamanan).
3. **Verify 2-Step Verification**: Under **"How you sign in to Google"**, confirm **2-Step Verification** is turned **ON**.
   > *Note*: Google requires 2-Step Verification before allowing App Passwords. If it is OFF, click it and follow the prompts to enable SMS or Authenticator verification.
4. **Open App Passwords**: Click **2-Step Verification** and scroll down to the bottom to find **App passwords** (Sandi aplikasi), or visit direct link: [https://myaccount.google.com/apppasswords](https://myaccount.google.com/apppasswords).
5. **Verify Identity**: Enter your Google account password when prompted.
6. **Name the App**: In the **App name** field, type:
   ```
   TickId Notification Service
   ```
7. **Generate Password**: Click **Create** (Buat).
8. **Copy the 16-Character Password**: A dialog will display a 16-character yellow-highlighted password:
   ```
   abcd efgh ijkl mnop
   ```
   - Copy it immediately and remove the spaces: `abcdefghijklmnop`.
   - *Note*: Google will never display this password again after clicking "Done".
9. **Configure `.env` for Gmail**:
   Create or update `.env` in `services/notification-service/`:
   ```env
   SERVER_PORT=8085
   WORKER_COUNT=5
   QUEUE_CAPACITY=500

   SMTP_HOST=smtp.gmail.com
   SMTP_PORT=587
   SMTP_USER=your-email@gmail.com
   SMTP_PASS=abcdefghijklmnop
   SMTP_FROM=your-email@gmail.com
   SMTP_FROM_NAME=TickId Official
   SMTP_USE_TLS=true
   ```

---

## API Endpoints Reference

All endpoints are hosted on port `:8085` under prefix `/api/v1/notifications`.

### 1. Ticket Confirmation & PDF E-Ticket

Dispatches an order confirmation email with embedded QR verification gate pass and an attached print-ready A4 E-Ticket PDF.

- **Method**: `POST`
- **Path**: `/api/v1/notifications/ticket-confirmation`
- **Content-Type**: `application/json`

#### Request Body
```json
{
  "booking_ref": "TICK-202611-987654",
  "customer_email": "attendee@example.com",
  "customer_name": "Alexander Johnson",
  "event_id": "EVT-889911",
  "event_title": "Coldplay: Music of the Spheres World Tour",
  "event_date": "Saturday, Nov 21, 2026 at 19:30 WIB",
  "venue_name": "Gelora Bung Karno Main Stadium",
  "venue_address": "Jl. Pintu Satu Senayan, Gelora, Kecamatan Tanah Abang, Jakarta Pusat",
  "tier_name": "VIP Platinum",
  "total_amount": 5000000,
  "currency": "IDR",
  "seats": [
    {
      "seat_id": "SEAT-VIP-001",
      "section": "Platinum-A",
      "row": "Row-01",
      "number": "A-12",
      "price": 2500000
    },
    {
      "seat_id": "SEAT-VIP-002",
      "section": "Platinum-A",
      "row": "Row-01",
      "number": "A-14",
      "price": 2500000
    }
  ]
}
```

#### Response: `202 Accepted`
```json
{
  "success": true,
  "message": "Ticket confirmation notification queued for delivery",
  "data": {
    "job_id": "9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d"
  }
}
```

#### Response: `400 Bad Request` (Validation Error)
```json
{
  "success": false,
  "message": "Validation failed",
  "error": "Key: 'TicketConfirmationRequest.CustomerEmail' Error:Field validation for 'CustomerEmail' failed on the 'email' tag"
}
```

---

### 2. Payment Failed / Reservation Expired

Notifies customer that their reservation payment window has expired and seats have been released back into the public inventory.

- **Method**: `POST`
- **Path**: `/api/v1/notifications/payment-failed`
- **Content-Type**: `application/json`

#### Request Body
```json
{
  "booking_ref": "TICK-202611-987654",
  "customer_email": "unpaid@example.com",
  "customer_name": "Alexander Johnson",
  "event_title": "Coldplay: Music of the Spheres World Tour",
  "reason": "Payment window expired (15 minutes time limit exceeded)",
  "expired_at": "2026-10-18 22:45 WIB"
}
```

#### Response: `202 Accepted`
```json
{
  "success": true,
  "message": "Payment failed notification queued for delivery",
  "data": {
    "job_id": "c3e98d1a-4710-4f51-b8eb-9464d7c0411d"
  }
}
```

---

### 3. Send Custom / Broadcast Email

Delivers an ad-hoc or administrative broadcast message to one or more recipient email addresses.

- **Method**: `POST`
- **Path**: `/api/v1/notifications/send-custom`
- **Content-Type**: `application/json`

#### Request Body
```json
{
  "to": [
    "user1@example.com",
    "user2@example.com"
  ],
  "subject": "TickId Platform Scheduled Maintenance",
  "body": "<p>Dear valued user, our systems will undergo scheduled maintenance from 02:00 to 04:00 WIB.</p>",
  "is_html": true
}
```

#### Response: `202 Accepted`
```json
{
  "success": true,
  "message": "Custom email notification queued for delivery",
  "data": {
    "job_id": "2d8f6e10-1533-4f93-b68e-177cf093d562"
  }
}
```

---

### 4. Operational Health Check

Liveness and operational readiness check.

- **Method**: `GET`
- **Path**: `/health`

#### Response: `200 OK`
```json
{
  "status": "ok",
  "service": "notification-service"
}
```

---

## Environment Variables Reference

| Variable | Type | Default | Description |
|---|---|---|---|
| `SERVER_PORT` | String | `8085` | Port on which the HTTP server listens. |
| `SMTP_HOST` | String | `localhost` | SMTP server hostname (`smtp.gmail.com` or `localhost`). |
| `SMTP_PORT` | Integer | `1025` | SMTP port (`587` for Gmail STARTTLS, `1025` for Mailpit). |
| `SMTP_USER` | String | *empty* | SMTP username / Gmail address. |
| `SMTP_PASS` | String | *empty* | SMTP password / 16-character Google App Password. |
| `SMTP_FROM` | String | `noreply@tickid.com` | From email address header. |
| `SMTP_FROM_NAME` | String | `TickId Official` | Friendly display name for the sender. |
| `SMTP_USE_TLS` | Boolean | `false` | Enable TLS/STARTTLS (`true` for Gmail, `false` for Mailpit). |
| `WORKER_COUNT` | Integer | `5` | Number of concurrent background worker goroutines. |
| `QUEUE_CAPACITY` | Integer | `500` | Size of buffered channel holding in-flight email jobs. |

---

## Testing & Verification

The test suite runs **100% offline** with zero internet connection or third-party email service requirements.

### Run All Unit and Integration Tests
```powershell
cd d:\KODINGANNN\TickId\backend\services\notification-service
go test -v ./...
```

### Run Handler Package Tests
```powershell
go test -v -count=1 ./internal/handler
```

### Run Static Analysis (Go Vet)
```powershell
go vet ./...
```

---

## Building and Running the Service

### 1. Build Server Binary
```powershell
go build -o server ./cmd/server
```

### 2. Run Server
```powershell
./server
```

When started, the service prints startup diagnostics and begins accepting HTTP requests:
```
[INFO] Starting TickId Notification Service...
[INFO] Loaded configuration: port=8085, smtp_host=localhost:1025, tls=false, workers=5, queue_cap=500
[INFO] SMTP client configured for host localhost:1025 (from: TickId Official <noreply@tickid.com>, tls: false)
[INFO] Worker pool initialized with 5 workers and 500 queue capacity
[INFO] Notification Service HTTP server listening on :8085
```

### 3. Graceful Shutdown
Send `SIGINT` (Ctrl+C) or `SIGTERM`. The service stops accepting incoming HTTP requests and drains pending email jobs before exiting cleanly.
