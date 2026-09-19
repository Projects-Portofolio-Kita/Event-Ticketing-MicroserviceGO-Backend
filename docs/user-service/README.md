# 📘 Panduan Pengerjaan User Service

Dokumen ini adalah panduan lengkap dan terstruktur bagi backend developer untuk membangun **`user-service`** pada project **Event-Ticketing-MicroserviceGO-Backend**, yang telah diselaraskan dengan arsitektur dan konfigurasi dari **`Event-Ticketing-MicroserviceGO-Infrastructure`**.

---

## 📌 Daftar Isi
1. [Ringkasan Service & Arsitektur](#1-ringkasan-service--arsitektur)
2. [Spesifikasi Teknis](#2-spesifikasi-teknis)
3. [Integrasi dengan Infrastructure (PostgreSQL Setup)](#3-integrasi-dengan-infrastructure-postgresql-setup)
4. [Skema Database & Migration](#4-skema-database--migration)
5. [Daftar Endpoint API](#5-daftar-endpoint-api)
6. [Struktur Direktori Standar](#6-struktur-direktori-standar)
7. [Tahapan Pengerjaan Runtut](#7-tahapan-pengerjaan-runtut)
8. [Standard Coding & Best Practices](#8-standard-coding--best-practices)
9. [Perintah Menjalankan & Testing](#9-perintah-menjalankan--testing)

---

## 1. Ringkasan Service & Arsitektur

* **Nama Service**: `user-service`
* **Port HTTP**: `8081`
* **Go Module**: `github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/user-service`
* **Database Instance**: PostgreSQL 16 Alpine (`event-ticketing-postgres`)
* **Nama Database**: `event_ticketing_db`
* **Docker Network**: `app-network` (bridge)
* **Tanggung Jawab Utama**:
  * Autentikasi (Register, Login, Refresh Token, Logout).
  * Otorisasi berbasis peran / RBAC (`CUSTOMER`, `ORGANIZER`, `ADMIN`).
  * Manajemen Profil Pengguna (Lihat profil, update profil, hapus/deaktivasi akun).
  * Reset Password & Verifikasi Email.
  * Mempublikasikan Domain Event (`user.registered`, `user.email_verified`).

---

## 2. Spesifikasi Teknis

| Komponen | Pilihan Teknologi / Standar |
|---|---|
| **Bahasa & Framework** | Go (Golang) 1.22+ & Gin Web Framework (`github.com/gin-gonic/gin`) |
| **Database Driver** | PostgreSQL driver (`gorm.io/driver/postgres` atau `pgx` / `lib/pq`) |
| **ORM / Query Builder** | GORM (`gorm.io/gorm`) atau Raw SQL Repository |
| **Token & Keamanan** | JWT (`golang-jwt/jwt/v5`), Hashing Password (`golang.org/x/crypto/bcrypt`) |
| **Konfigurasi** | Environment variables (`.env` via `godotenv`) |
| **Event Broker** | RabbitMQ / Kafka (`user.registered`, `user.email_verified`) |

---

## 3. Integrasi dengan Infrastructure (PostgreSQL Setup)

Database dijalankan dan dikelola melalui repository **`Event-Ticketing-MicroserviceGO-Infrastructure`**.

### 3.1 Parameter Database Infrastructure
Berdasarkan `docker-compose.yml` dan `.env.example` di repositori infrastruktur:

| Parameter | Nilai Default | Keterangan |
|---|---|---|
| **Container Name** | `event-ticketing-postgres` | Nama container database |
| **Image** | `postgres:16-alpine` | Versi PostgreSQL |
| **Host (Dev Lokal)** | `localhost` | Diakses saat service jalan via `go run` di host laptop |
| **Host (Docker)** | `event-ticketing-postgres` | Diakses saat service jalan di dalam container Docker |
| **Port** | `5432` | Port database |
| **Database** | `event_ticketing_db` | Dibuat otomatis oleh `database-init.sh` |
| **User** | `admin` (atau sesuai `.env` infra) | Default admin database |
| **Password** | (Sesuai `POSTGRES_PASSWORD` di `.env` infra) | Contoh: `your-secret-password` atau `tiketid` |
| **Network** | `app-network` | Docker bridge network bersama |

> ⚠️ **Catatan Penting dari Infrastructure**:  
> Script `database-init.sh` pada infrastruktur hanya membuat database kosong (`event_ticketing_db`).  
> **Migrations (pembuatan tabel & index) sepenuhnya dilakukan oleh backend service**, bukan di repository infrastruktur.

### 3.2 Menjalankan Database dari Repository Infrastructure
Sebelum menjalankan `user-service`, pastikan database sudah aktif:

```bash
# 1. Masuk ke direktori infrastructure
cd ../Event-Ticketing-MicroserviceGO-Infrastructure

# 2. Salin .env dan sesuaikan jika belum ada
cp .env.example .env

# 3. Jalankan container PostgreSQL
docker-compose up -d

# 4. Verifikasi status database
docker-compose ps
```

### 3.3 Memeriksa Database via PSQL Container
Untuk melihat atau memverifikasi tabel langsung di dalam container database:
```bash
docker exec -it event-ticketing-postgres psql -U admin -d event_ticketing_db
```

### 3.4 Format `DATABASE_URL` untuk `user-service`
Pada file `.env` di `services/user-service/`:

* **Saat running lokal di laptop (`go run ./cmd/server`)**:
  ```env
  DATABASE_URL=postgres://admin:your-secret-password@localhost:5432/event_ticketing_db?sslmode=disable
  ```
* **Saat running di dalam Docker Container** (terkoneksi ke `app-network`):
  ```env
  DATABASE_URL=postgres://admin:your-secret-password@event-ticketing-postgres:5432/event_ticketing_db?sslmode=disable
  ```

---

## 4. Skema Database & Migration

Tabel berikut harus dibuat di dalam database **`event_ticketing_db`**.

### 4.1 Entitas `users`
```sql
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100),
    phone VARCHAR(20),
    avatar_url TEXT,
    role VARCHAR(20) NOT NULL DEFAULT 'CUSTOMER' CHECK (role IN ('CUSTOMER', 'ORGANIZER', 'ADMIN')),
    email_verified BOOLEAN NOT NULL DEFAULT FALSE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_users_email ON users(email);
```

### 4.2 Entitas `refresh_tokens`
```sql
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    revoked BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
```

---

## 5. Daftar Endpoint API

Semua endpoint diawali prefix `/api/v1`.

| Method | Endpoint | Keterangan | Autentikasi |
|---|---|---|---|
| `POST` | `/api/v1/auth/register` | Mendaftar akun baru | Publik |
| `POST` | `/api/v1/auth/login` | Login, memperoleh Access Token & Refresh Token | Publik |
| `POST` | `/api/v1/auth/refresh` | Memperbarui Access Token menggunakan Refresh Token | Publik / Refresh Token |
| `POST` | `/api/v1/auth/logout` | Revoke/mencabut Refresh Token | JWT Bearer |
| `POST` | `/api/v1/auth/forgot-password` | Kirim tautan reset password ke email | Publik |
| `POST` | `/api/v1/auth/reset-password` | Reset password dengan token verifikasi | Publik |
| `GET` | `/api/v1/users/me` | Mengambil data profil user yang sedang login | JWT Bearer |
| `PATCH` | `/api/v1/users/me` | Update informasi profil user | JWT Bearer |
| `GET` | `/api/v1/users/:id` | Mengambil profil user berdasarkan ID | JWT Bearer (Admin Only) |
| `DELETE`| `/api/v1/users/me` | Deaktivasi akun | JWT Bearer |

---

## 6. Struktur Direktori Standar

Struktur folder mengadopsi clean/layered architecture sesuai root `README.md` dan `readmegorm.md`:

```text
services/user-service/
├── cmd/
│   └── server/
│       └── main.go               # Entry point, dependency injection, router setup
├── pkg/
│   └── database/
│       └── db.go                 # Inisialisasi koneksi PostgreSQL / GORM
├── internal/
│   ├── config/
│   │   └── config.go             # Load file .env dan konfigurasi sistem
│   ├── handler/
│   │   ├── auth_handler.go       # HTTP Controller authentication
│   │   └── user_handler.go       # HTTP Controller user profile
│   ├── service/
│   │   ├── auth_service.go       # Business logic registrasi, token, login
│   │   └── user_service.go       # Business logic profile & data user
│   ├── repository/
│   │   ├── user_repository.go    # Query SQL / GORM tabel users
│   │   └── token_repository.go   # Query SQL / GORM tabel refresh_tokens
│   ├── model/
│   │   ├── user.go               # Struct domain entity User
│   │   └── refresh_token.go      # Struct domain entity RefreshToken
│   ├── dto/
│   │   ├── auth_dto.go           # Request & Response payload (Register, Login, Token)
│   │   └── user_dto.go           # Request & Response payload profil
│   └── middleware/
│       ├── auth_middleware.go    # Validasi JWT Bearer
│       └── rbac_middleware.go    # Pengecekan role (CUSTOMER/ORGANIZER/ADMIN)
├── migrations/
│   ├── 000001_create_users.up.sql
│   ├── 000001_create_users.down.sql
│   ├── 000002_create_refresh_tokens.up.sql
│   └── 000002_create_refresh_tokens.down.sql
├── tests/
│   └── integration/
│       └── auth_test.go
├── .env.example
├── .gitignore
├── Dockerfile
├── go.mod
└── go.sum
```

---

## 7. Tahapan Pengerjaan Runtut

### Langkah 1: Pastikan Database Infrastructure Aktif
```bash
# Di direktori Event-Ticketing-MicroserviceGO-Infrastructure
docker-compose up -d
docker-compose ps
```

### Langkah 2: Inisialisasi Go Module & Dependensi
```powershell
cd services/user-service

# Inisialisasi Go module
go mod init github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/user-service

# Buat folder layer arsitektur
mkdir -p cmd/server pkg/database migrations tests/integration
mkdir -p internal/config internal/handler internal/service internal/repository internal/model internal/dto internal/middleware

# Install dependensi utama
go get github.com/gin-gonic/gin
go get gorm.io/gorm gorm.io/driver/postgres
go get github.com/golang-jwt/jwt/v5
go get golang.org/x/crypto/bcrypt
go get github.com/joho/godotenv
go mod tidy
```

### Langkah 3: Setup File Environment
Buat file `.env.example` dan `.env`:

```env
SERVER_PORT=8081
# Sesuaikan password dengan yang ada di .env Event-Ticketing-MicroserviceGO-Infrastructure
DATABASE_URL=postgres://admin:your-secret-password@localhost:5432/event_ticketing_db?sslmode=disable
JWT_SECRET=super_secret_jwt_key_minimal_32_characters_long
JWT_EXPIRATION_MINUTES=15
REFRESH_TOKEN_EXPIRATION_DAYS=7
```

### Langkah 4: Buat Koneksi Database (`pkg/database/db.go`)
```go
package database

import (
    "fmt"
    "os"

    "gorm.io/driver/postgres"
    "gorm.io/gorm"
)

func NewConnection() (*gorm.DB, error) {
    dsn := os.Getenv("DATABASE_URL")
    if dsn == "" {
        return nil, fmt.Errorf("DATABASE_URL is not set")
    }

    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
    if err != nil {
        return nil, fmt.Errorf("failed to connect to database: %w", err)
    }

    return db, nil
}
```

### Langkah 5: Eksekusi Migrasi Tabel
Jalankan migrasi DDL pada Section 4 ke database `event_ticketing_db` (bisa menggunakan `golang-migrate`, SQL client, atau GORM `AutoMigrate(&model.User{}, &model.RefreshToken{})`).

### Langkah 6: Implementasi Lapisan Kode (Bottom-Up)
1. **Model** (`internal/model/user.go`): Definisikan struct entity dengan tag GORM/JSON.
2. **DTO** (`internal/dto/auth_dto.go`): Struct payload request & response lengkap dengan validation tag (`binding:"required,email"`).
3. **Repository** (`internal/repository/user_repository.go`): Handle operasi database (CRUD).
4. **Service** (`internal/service/auth_service.go`): Logika hashing password (`bcrypt`), validasi kredensial, dan penerbitan JWT access & refresh token.
5. **Handler** (`internal/handler/auth_handler.go`): Bind HTTP payload, invoke service, dan kirim JSON response.
6. **Middleware** (`internal/middleware/auth_middleware.go`): Validasi JWT header `Authorization: Bearer <token>`.
7. **Entry Point** (`cmd/server/main.go`): Wiring database, repository, service, handler, middleware, dan daftarkan route Gin.

---

## 8. Standard Coding & Best Practices

1. **Kesesuaian Database**:
   * Gunakan selalu nama database `event_ticketing_db` sesuai yang diinisiasi oleh infrastructure.
   * Isolasi tabel: pastikan migration `user-service` hanya mengelola tabel `users` dan `refresh_tokens`.
2. **Format Response JSON Konsisten**:
   ```json
   {
     "success": true,
     "message": "Login successful",
     "data": {
       "access_token": "eyJhbG...",
       "refresh_token": "d7a8b...",
       "expires_in": 900
     }
   }
   ```
3. **Keamanan & Credentials**:
   * Jangan simpan password plain text di database; wajib gunakan `bcrypt` cost minimal 10.
   * Jangan sertakan file `.env` ke Git (`.gitignore`).
   * Jangan kembalikan field sensitif seperti `password_hash` pada response DTO.

---

## 9. Perintah Menjalankan & Testing

Jalankan perintah berikut di folder `services/user-service`:

```powershell
# 1. Format kode Go
go fmt ./...

# 2. Pastikan dependensi rapi
go mod tidy

# 3. Jalankan service lokal
go run ./cmd/server

# 4. Jalankan unit test
go test ./...

# 5. Jalankan test dengan race detector
go test -race ./...
```

Periksa kesehatan service melalui endpoint health check:
```powershell
curl http://localhost:8081/health
```
*(atau buka browser di `http://localhost:8081/health`)*
