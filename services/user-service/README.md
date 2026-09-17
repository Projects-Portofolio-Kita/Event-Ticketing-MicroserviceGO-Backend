# User Service

Service untuk autentikasi, otorisasi (RBAC), dan manajemen profil pengguna pada platform Event Ticketing.

## Environment Variables
Salin `.env.example` ke `.env`:
```bash
cp .env.example .env
```

| Variable | Default | Keterangan |
|---|---|---|
| `SERVER_PORT` | `8081` | Port HTTP service |
| `DATABASE_URL` | `postgres://admin:your-secret-password@localhost:5432/event_ticketing_db?sslmode=disable` | Koneksi ke PostgreSQL |
| `JWT_SECRET` | `...` | Kunci enkripsi token JWT (min 32 karakter) |
| `JWT_EXPIRATION_MINUTES` | `15` | Masa berlaku Access Token |
| `REFRESH_TOKEN_EXPIRATION_DAYS` | `7` | Masa berlaku Refresh Token |

## Cara Menjalankan
```bash
# Running secara lokal
go run ./cmd/server

# Build executable
go build -o server ./cmd/server
```

## Health Check
```bash
curl http://localhost:8081/health
```
