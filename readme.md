# Microservice Go - Service Initialization Guide

Panduan singkat untuk membuat dan menjalankan service baru pada project:

```text
backend/
├── services/
│   ├── auth-service/            # :8081  register/login/JWT/Google
│   ├── catalog-service/         # :8082  events + seatmap
│   ├── booking-service/         # :8083  reservasi & lock kursi
│   ├── payment-service/         # :8084  QRIS (mock)
│   └── notification-service/    # :8085  notifikasi pengguna
├── libs/
│   └── common/                  # Go module bersama: cors, health, httperr, notify
├── gateway/                     # :8086 (host) -> :8080 (container), proxy tunggal
├── docker-compose.yml           # include ../infrastucture/docker-compose.yml (postgres)
└── readme.md

../contracts/                    # OpenAPI + skema event (dipakai bersama frontend)
../infrastucture/                # postgres: satu-satunya definisi database
../web-portal/                   # Vite/React portal, dev :3000
```

Stack penuh (database + 5 service + gateway + portal):

```powershell
docker compose -f backend/docker-compose.yml up -d --build
# cek semuanya dalam satu request
Invoke-RestMethod http://localhost:8086/health
# atau per service: 8081 auth · 8082 catalog · 8083 booking · 8084 payment · 8085 notification
```

---

# 1. Prerequisite

Pastikan command berikut bisa dijalankan:

```powershell
go version
git --version
docker --version
docker compose version
```

Jika `go` tidak dikenali, pastikan Go sudah terinstall dan:

```text
C:\Program Files\Go\bin
```

sudah masuk ke Windows `PATH`.

---

# 2. Cara Init Service Baru

Contoh membuat `catalog-service`.

Dari root repository:

```powershell
cd services
mkdir catalog-service
cd catalog-service
```

Inisialisasi Go Module:

```powershell
go mod init github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service
```

Install Gin:

```powershell
go get github.com/gin-gonic/gin
```

Rapikan dependency:

```powershell
go mod tidy
```

Setelah itu struktur minimal:

```text
catalog-service/
├── go.mod
└── go.sum
```

> `go mod init` hanya membuat Go Module. Struktur folder aplikasi harus dibuat sendiri atau menggunakan generator/script.

---

# 3. Struktur Folder Recommended

Gunakan baseline berikut untuk service baru:

```text
service-name/
├── cmd/
│   └── server/
│       └── main.go
│
├── internal/
│   ├── config/
│   ├── handler/
│   ├── service/
│   ├── repository/
│   ├── model/
│   ├── dto/
│   └── middleware/
│
├── migrations/
├── tests/
│   └── integration/
│
├── .env.example
├── .gitignore
├── Dockerfile
├── README.md
├── go.mod
└── go.sum
```

Folder baseline:

| Folder/File | Fungsi |
|---|---|
| `cmd/server/main.go` | Entry point dan bootstrap aplikasi |
| `internal/config` | Load environment/configuration |
| `internal/handler` | HTTP handler / menerima request |
| `internal/service` | Business logic |
| `internal/repository` | Query dan akses database |
| `internal/model` | Entity/model domain |
| `internal/dto` | Request dan response DTO |
| `internal/middleware` | JWT, authorization, logging, dll |
| `migrations` | SQL migration |
| `tests` | Integration test |
| `.env.example` | Template environment variable |
| `Dockerfile` | Build container service |
| `go.mod` | Go module dan dependency |
| `go.sum` | Checksum dependency |

Tidak semua service harus mempunyai folder tambahan yang sama.

Contoh Booking Service:

```text
internal/
├── handler/
├── service/
├── repository/
├── model/
├── dto/
├── middleware/
├── lock/
└── event/
```

Contoh Payment Service:

```text
internal/
├── handler/
├── service/
├── repository/
├── model/
├── dto/
└── payment/
```

---

# 4. Cara Membuat Struktur Folder

PowerShell:

```powershell
mkdir cmd
mkdir cmd/server

mkdir internal
mkdir internal/config
mkdir internal/handler
mkdir internal/service
mkdir internal/repository
mkdir internal/model
mkdir internal/dto
mkdir internal/middleware

mkdir migrations

mkdir tests
mkdir tests/integration
```

Buat file:

```powershell
New-Item cmd/server/main.go
New-Item .env.example
New-Item .gitignore
New-Item Dockerfile
New-Item README.md
```

---

# 4b. Menggunakan `libs/common`

Semua service memakai satu modul Go bersama di `backend/libs/common`:

| Paket | Fungsi |
|---|---|
| `cors` | Preflight CORS seragam (dulu disalin di tiap service) |
| `health` | `GET /health` dengan payload `{"status","service"}` |
| `httperr` | Envelope error `{"message": "..."}` |
| `notify` | Client fire-and-forget ke notification-service |

Tarik lewat `replace` lokal (diinisialisasi otomatis saat `go mod tidy`):

```go.mod
require github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/libs/common v0.0.0

replace github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/libs/common => ../../libs/common
```

Pakai di `cmd/server/main.go`:

```go
router := gin.Default()
router.Use(cors.New())
health.Register(router, "my-service")
```

Dan di handler: `httperr.BadRequest(c, err.Error())`, di service:
`notify.New()` lalu `notifier.Send(notify.Message{...})`.

> **Penting untuk Docker:** `replace` lokal membuat path `../../libs/common`
> wajib ada saat build, jadi **build context harus folder `backend/`**, bukan
> folder service. Lihat `DockerFile` yang sudah ada — semua memakai
> `context: .` + `dockerfile: services/<nama>/DockerFile` di compose.

---

# 5. File Pertama: `main.go`

Lokasi:

```text
cmd/server/main.go
```

Isi minimal:

```go
package main

import (
    "net/http"

    "github.com/gin-gonic/gin"
)

func main() {
    router := gin.Default()

    router.GET("/health", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{
            "status":  "ok",
            "service": "catalog-service",
        })
    })

    router.Run(":8082")
}
```

Jalankan dari root service:

```powershell
go run ./cmd/server
```

Test:

```text
http://localhost:8082/health
```

Atau PowerShell:

```powershell
Invoke-RestMethod http://localhost:8082/health
```

---

# 6. Alur Antar Directory

Gunakan alur utama:

```text
HTTP Request
    ↓
Middleware
    ↓
Handler
    ↓
Service
    ↓
Repository
    ↓
Database
```

Penempatan code:

```text
internal/
│
├── handler/
│     ↓ menerima HTTP request
│
├── service/
│     ↓ business logic
│
├── repository/
│     ↓ database query
│
└── model/
      ↓ entity/data
```

Contoh login:

```text
POST /api/v1/auth/login
          ↓
auth_handler.go
          ↓
auth_service.go
          ↓
user_repository.go
          ↓
PostgreSQL
          ↓
Response kembali ke Handler
          ↓
HTTP Response
```

Dependency direction:

```text
Handler
   ↓
Service
   ↓
Repository
   ↓
Database
```

`main.go` melakukan wiring:

```text
Config
  ↓
Database
  ↓
Repository
  ↓
Service
  ↓
Handler
  ↓
Routes
  ↓
Server
```

---

# 7. Command Go yang Sering Digunakan

## Cek Go

```powershell
go version
```

## Membuat Go Module

```powershell
go mod init <module-path>
```

Contoh:

```powershell
go mod init github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/user-service
```

## Install dependency

```powershell
go get <package>
```

Contoh Gin:

```powershell
go get github.com/gin-gonic/gin
```

## Merapikan dependency

Jalankan setelah menambah atau menghapus package:

```powershell
go mod tidy
```

## Menjalankan service

```powershell
go run ./cmd/server
```

## Menjalankan test

```powershell
go test ./...
```

## Test dengan race detector

Terutama untuk concurrent logic:

```powershell
go test -race ./...
```

## Format semua code

```powershell
go fmt ./...
```

## Build service

```powershell
go build ./cmd/server
```

## Menampilkan dependency

```powershell
go list -m all
```

---

# 8. Workflow Saat Membuat Feature

Urutan yang digunakan:

```text
1. Model
   ↓
2. DTO
   ↓
3. Repository
   ↓
4. Service
   ↓
5. Handler
   ↓
6. Route
   ↓
7. Test
```

Contoh fitur:

```text
Create Event
```

File yang mungkin dibuat:

```text
internal/
├── model/
│   └── event.go
├── dto/
│   └── create_event_request.go
├── repository/
│   └── event_repository.go
├── service/
│   └── event_service.go
└── handler/
    └── event_handler.go
```

Alur:

```text
POST /api/v1/events
        ↓
EventHandler
        ↓
EventService
        ↓
EventRepository
        ↓
PostgreSQL
```

---

# 9. Dependency Injection di `main.go`

`main.go` membuat dependency secara berurutan:

```text
Database
   ↓
Repository
   ↓
Service
   ↓
Handler
```

Contoh:

```go
repository := NewEventRepository(db)
service := NewEventService(repository)
handler := NewEventHandler(service)
```

Kemudian handler didaftarkan ke route.

Jangan membuat repository langsung di dalam handler.

---

# 10. Menambahkan Database

Setiap service memiliki konfigurasi sendiri.

Contoh `.env.example`:

```env
SERVER_PORT=8082
DATABASE_URL=postgres://username:password@localhost:5432/catalog_db
```

Alur:

```text
.env
  ↓
config/
  ↓
main.go
  ↓
database connection
  ↓
repository
```

Jangan commit `.env` jika berisi credential asli.

Tambahkan ke `.gitignore`:

```gitignore
.env
*.exe
bin/
coverage.out
```

`go.mod` dan `go.sum` tetap di-commit.

---

# 11. Migration

Migration disimpan di:

```text
migrations/
```

Contoh:

```text
migrations/
├── 000001_create_users.up.sql
├── 000001_create_users.down.sql
├── 000002_create_refresh_tokens.up.sql
└── 000002_create_refresh_tokens.down.sql
```

Format:

```text
<number>_<description>.up.sql
<number>_<description>.down.sql
```

---

# 12. Cara Run Service

Dari root service:

```powershell
go run ./cmd/server
```

Contoh:

```powershell
cd services/user-service
go run ./cmd/server
```

Untuk menghentikan:

```text
Ctrl + C
```

Setiap service harus menggunakan port berbeda.

Contoh development:

```text
user-service          :8081
catalog-service       :8082
booking-service       :8083
payment-service       :8084
notification-service  :8085
```

---

# 13. Docker Command Dasar

Build image dari directory service:

```powershell
docker build -t user-service .
```

Run:

```powershell
docker run -p 8081:8081 user-service
```

Cek container:

```powershell
docker ps
```

Stop container:

```powershell
docker stop <container-id>
```

---

# 14. Docker Compose

Dari root project:

```powershell
docker compose up
```

Jalankan background:

```powershell
docker compose up -d
```

Stop semua:

```powershell
docker compose down
```

Cek:

```powershell
docker compose ps
```

Lihat semua log:

```powershell
docker compose logs
```

Lihat log service tertentu:

```powershell
docker compose logs user-service
```

---

# 15. Git Command untuk Project

Cek perubahan:

```powershell
git status
```

Tambah semua perubahan:

```powershell
git add .
```

Tambah satu service:

```powershell
git add services/user-service
```

Commit:

```powershell
git commit -m "feat(user): initialize user service"
```

Push:

```powershell
git push
```

Buat branch feature:

```powershell
git switch -c feat/catalog-service
```

Push branch pertama kali:

```powershell
git push -u origin feat/catalog-service
```

---

# 16. Jika `git push -u origin main` Error

Error:

```text
src refspec main does not match any
```

Biasanya karena repository belum memiliki commit.

Lakukan:

```powershell
git add .
git commit -m "chore: initialize project"
git branch -M main
git push -u origin main
```

Cek branch:

```powershell
git branch
```

Cek remote:

```powershell
git remote -v
```

---

# 17. Workflow Harian

Sebelum mulai:

```powershell
git pull
```

Setelah coding:

```powershell
go fmt ./...
go mod tidy
go test ./...
go test -race ./...
```

Kemudian:

```powershell
git status
git add .
git commit -m "feat(service): add feature"
git push
```

---

# 18. Komunikasi Antar Service

Jangan:

```text
booking-service
    ↓
akses langsung database user-service
```

Gunakan:

```text
booking-service
    ↓ HTTP/gRPC
user-service
```

atau event:

```text
booking-service
    ↓
Message Broker
    ↓
notification-service
```

Contract API:

```text
contracts/openapi/
```

Contract event:

```text
contracts/events/
```

---

# 19. Struktur Akhir Project

```text
Event-Ticketing-MicroserviceGO-Backend/
│
├── services/
│   ├── user-service/
│   │   ├── cmd/
│   │   ├── internal/
│   │   ├── migrations/
│   │   ├── tests/
│   │   ├── Dockerfile
│   │   ├── README.md
│   │   ├── go.mod
│   │   └── go.sum
│   │
│   ├── catalog-service/
│   ├── booking-service/
│   ├── payment-service/
│   └── notification-service/
│
├── libs/
│   └── common/
│
├── contracts/
│   ├── openapi/
│   └── events/
│
├── gateway/
│
├── docker-compose.yml
└── README.md
```

---

# 20. Checklist Init Service Baru

```text
[ ] Buat folder service
[ ] Masuk ke folder service
[ ] go mod init
[ ] go get dependency
[ ] Buat cmd/server/main.go
[ ] Buat internal/config
[ ] Buat internal/handler
[ ] Buat internal/service
[ ] Buat internal/repository
[ ] Buat internal/model
[ ] Buat internal/dto
[ ] Buat internal/middleware jika diperlukan
[ ] Buat migrations jika menggunakan database
[ ] Buat tests jika diperlukan
[ ] Buat .env.example
[ ] Buat .gitignore
[ ] Buat Dockerfile
[ ] Buat health endpoint
[ ] go fmt ./...
[ ] go mod tidy
[ ] go test ./...
[ ] go test -race ./...
[ ] git add
[ ] git commit
[ ] git push
```

---

# 21. Quick Start

Template command untuk service baru:

```powershell
cd services

mkdir <service-name>
cd <service-name>

go mod init github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/<service-name>

go get github.com/gin-gonic/gin

mkdir cmd
mkdir cmd/server

mkdir internal
mkdir internal/config
mkdir internal/handler
mkdir internal/service
mkdir internal/repository
mkdir internal/model
mkdir internal/dto
mkdir internal/middleware

mkdir migrations
mkdir tests
mkdir tests/integration

New-Item cmd/server/main.go
New-Item .env.example
New-Item .gitignore
New-Item Dockerfile
New-Item README.md

go mod tidy
```

Setelah `main.go` dibuat:

```powershell
go run ./cmd/server
```

Sebelum push:

```powershell
go fmt ./...
go mod tidy
go test ./...
go test -race ./...

git add .
git commit -m "feat(<service>): initialize service"
git push
```
