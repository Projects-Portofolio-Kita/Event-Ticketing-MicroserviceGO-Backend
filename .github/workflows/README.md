# GitHub Actions CI/CD

## Overview

Project ini menggunakan GitHub Actions untuk otomasi **lint**, **build**, **test**, dan **Docker image build** untuk setiap microservice.

Pipeline dijalankan secara paralel untuk 3 service:

```text
auth-service      (port 8082)
catalog-service   (port 8803)
payment-service   (port 8804)
```

---

## Pipeline Flow

```text
Push/PR ke main atau develop
         │
         ▼
┌─────────────────┐     ┌──────────────────┐
│   Lint           │     │   Vet & Staticcheck │
│ golangci-lint    │     │   go vet + staticcheck │
└────────┬────────┘     └────────┬───────────┘
         │                       │
         ▼                       ▼
┌─────────────────────────────────────────┐
│              Build                       │
│     go build -v -o /dev/null ./...      │
└────────────────┬────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────┐
│              Test                        │
│     go test -v -race -coverprofile      │
└────────────────┬────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────┐
│           Go Mod Tidy                    │
│  Pastikan go.mod & go.sum bersih        │
└────────────────┬────────────────────────┘
                 │
                 ▼ (hanya push ke main)
┌─────────────────────────────────────────┐
│          Docker Build                    │
│     Build image untuk setiap service     │
└─────────────────────────────────────────┘
```

---

## Jobs Breakdown

### 1. Lint

Menggunakan [`golangci-lint`](https://golangci-lint.run/) untuk mengecek:

- Syntax errors
- Unused code
- Code style violations
- Potential bugs
- Security issues

### 2. Vet & Staticcheck

- **`go vet`** - Mengecek suspicious constructs yang bisa menyebabkan bug
- **`staticcheck`** - Advanced static analysis untuk Go

### 3. Build

- Download dependencies
- Verify `go.sum` integrity
- Build binary (output ke `/dev/null` untuk CI)

### 4. Test

- Jalankan semua test dengan **race detector** (`-race`)
- Generate **coverage report** (`coverage.out`)
- Coverage report di-upload sebagai artifact (hanya auth-service)

### 5. Go Mod Tidy

- Jalankan `go mod tidy`
- Cek apakah ada perubahan pada `go.mod` atau `go.sum`
- Jika ada perubahan, CI **gagal** → developer harus menjalankan `go mod tidy` lokal dulu

### 6. Docker Build

- Build Docker image menggunakan multi-stage build
- **Hanya dijalankan** saat push ke branch `main`
- Tidak dijalankan saat pull request

---

## Matrix Strategy

Setiap job menggunakan **matrix strategy** untuk menjalankan job secara paralel untuk setiap service:

```yaml
strategy:
  matrix:
    service:
      - auth-service
      - catalog-service
      - payment-service
```

Ini berarti setiap job akan dijalankan **3 kali** (satu untuk setiap service).

---

## Trigger

Pipeline dijalankan saat:

| Event | Branch | Keterangan |
|---|---|---|
| `push` | `main`, `develop` | Semua job + Docker build |
| `pull_request` | `main`, `develop` | Semua job kecuali Docker build |

---

## Local Commands

Sebelum push, jalankan command berikut secara lokal:

### Lint

```bash
# Install golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Jalankan lint
cd auth-service
golangci-lint run ./...
```

### Vet & Staticcheck

```bash
cd auth-service

# Go vet
go vet ./...

# Install staticcheck
go install honnef.co/go/tools/cmd/staticcheck@latest

# Jalankan staticcheck
staticcheck ./...
```

### Build

```bash
cd auth-service
go build -v ./cmd/server
```

### Test

```bash
cd auth-service

# Test biasa
go test ./...

# Test dengan race detector
go test -race ./...

# Test dengan coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Tidy

```bash
cd auth-service
go mod tidy
```

### Format

```bash
cd auth-service
go fmt ./...
```

---

## Adding New Service

Untuk menambahkan service baru ke CI/CD:

### 1. Buat Service

Ikuti panduan di `readme.md` untuk membuat service baru.

### 2. Tambahkan ke Matrix

Edit `.github/workflows/go.yml`, tambahkan service baru ke setiap job:

```yaml
strategy:
  matrix:
    service:
      - auth-service
      - catalog-service
      - payment-service
      - new-service        # ← tambahkan di sini
```

### 3. Buat DockerFile

Buat `DockerFile` di root service dengan port yang sesuai.

---

## Required Secrets

Tidak ada secrets yang diperlukan untuk pipeline dasar ini.

Jika ingin menambahkan Docker Hub push atau deployment, tambahkan secrets di GitHub:

```text
Settings → Secrets and variables → Actions → New repository secret
```

Contoh secrets yang mungkin diperlukan nanti:

| Secret | Keterangan |
|---|---|
| `DOCKER_USERNAME` | Username Docker Hub |
| `DOCKER_PASSWORD` | Password Docker Hub |
| `DEPLOY_HOST` | IP/hostname server deployment |
| `DEPLOY_KEY` | SSH key untuk deployment |

---

## Troubleshooting

### Error: `go.mod` version mismatch

```text
go: go.mod requires go >= 1.22 (running go 1.20)
```

**Fix:** Pastikan `go-version` di workflow sama dengan `go.mod`:

```yaml
- name: Set up Go
  uses: actions/setup-go@v5
  with:
    go-version: '1.22'
```

### Error: `go.sum` mismatch

```text
go: go.sum mismatch
```

**Fix:** Jalankan lokal:

```bash
go mod tidy
git add go.mod go.sum
git commit -m "chore: update go.sum"
```

### Error: Lint errors

```text
level=error msg="Timeout exceeded"
```

**Fix:** Tambahkan timeout di args:

```yaml
args: --timeout=10m
```

### Error: Build fails with missing import

```text
main.go:8:2: no required module provides package github.com/gin-gonic/gin
```

**Fix:** Jalankan lokal:

```bash
go get github.com/gin-gonic/gin
go mod tidy
```

### Error: Docker build fails

```text
COPY go.mod go.sum ./ => ERROR [stage-0 3/6] COPY go.mod go.sum ./
```

**Fix:** Pastikan `go.sum` ada di service. Jalankan:

```bash
cd auth-service
go mod tidy
```

---

## Coverage Report

Coverage report di-upload sebagai artifact. Untuk melihat:

1. Buka GitHub Actions
2. Pilih run yang berhasil
3. Scroll ke bagian **Artifacts**
4. Download `coverage-auth-service`
5. Buka secara lokal:

```bash
go tool cover -html=coverage.out
```

---

## Reference

- [GitHub Actions Docs](https://docs.github.com/en/actions)
- [golangci-lint](https://golangci-lint.run/)
- [staticcheck](https://staticcheck.dev/)
- [Go Testing](https://pkg.go.dev/testing)
