Isi README (GORM)

# Database Setup dengan GORM

Panduan setup database PostgreSQL menggunakan GORM untuk semua service.

---

## Apa itu GORM?

GORM adalah ORM (Object Relational Mapping) untuk Go yang memudahkan
interaksi dengan database. GORM mengubah Go struct menjadi tabel database
dan menyediakan method untuk CRUD tanpa tulis SQL manual.

| Komponen       | Fungsi                                |
| -------------- | ------------------------------------- |
| **Gin**        | HTTP framework (handle API request)   |
| **GORM**       | Database driver (query ke PostgreSQL) |
| **PostgreSQL** | Database server                       |

---

## Arsitektur

Client (Browser/App)
↓ HTTP Request
Gin (API Layer)
↓ Business Logic
GORM (Database Layer)
↓ SQL Query
PostgreSQL

---

## Struktur Folder

services/
├── auth-service/
│ ├── cmd/server/main.go ← entry point
│ ├── pkg/database/db.go ← koneksi ke PostgreSQL
│ ├── internal/model/ ← definisi tabel (struct)
│ ├── internal/handler/ ← HTTP handler
│ ├── internal/service/ ← business logic
│ ├── internal/repository/ ← database query
│ └── .env ← DATABASE_URL
├── catalog-service/
│ └── (struktur sama)
└── payment-service/
└── (struktur sama)

---

## Port Service

| Service         | Port |
| --------------- | ---- |
| auth-service    | 8081 |
| catalog-service | 8082 |
| payment-service | 8083 |
| PostgreSQL      | 5432 |

---

## Database

Semua service menggunakan **1 PostgreSQL instance** dan **1 database** yang sama.

| Field    | Nilai                                |
| -------- | ------------------------------------ |
| Host     | localhost (dev) / postgres (Docker)  |
| Port     | 5432                                 |
| Database | event_ticketing_db                   |
| User     | admin                                |
| Password | (lihat di .env / docker-compose.yml) |

---

## Setup

### 1. Install Dependencies

Jalankan di folder masing-masing service:

```bash
go get gorm.io/gorm gorm.io/driver/postgres
2. Buat pkg/database/db.go
Buat folder pkg/database/ lalu buat file db.go:
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
3. Buat Model (Struct)
Contoh model di internal/model/user.go:
package model

import "gorm.io/gorm"

type User struct {
    gorm.Model
    Email    string `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
    Password string `gorm:"type:varchar(255);not null" json:"-"`
    Name     string `gorm:"type:varchar(255);not null" json:"name"`
    Role     string `gorm:"type:varchar(20);default:'user'" json:"role"`
}
4. Update main.go
package main

import (
    "fmt"
    "log"
    "net/http"
    "os"

    "github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/<service-name>/internal/model"
    "github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/<service-name>/pkg/database"
    "github.com/gin-gonic/gin"
)

func main() {
    // Koneksi ke database
    db, err := database.NewConnection()
    if err != nil {
        log.Fatal("Failed to connect to database:", err)
    }

    // AutoMigrate: buat/update tabel otomatis dari struct
    sqlDB, _ := db.DB()
    defer sqlDB.Close()

    db.AutoMigrate(&model.User{})

    // Setup router
    port := os.Getenv("SERVER_PORT")
    if port == "" {
        port = "8081"
    }

    router := gin.Default()
    router.GET("/health", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{
            "status":  "ok",
            "service": "<service-name>",
        })
    })

    log.Printf("Service running on :%s", port)
    router.Run(fmt.Sprintf(":%s", port))
}
5. Isi .env
SERVER_PORT=8081
DATABASE_URL=postgres://admin:password@localhost:5432/event_ticketing_db?sslmode=disable
GORM Commands (Query)
Create
user := User{Name: "John", Email: "john@example.com"}
result := db.Create(&user)
Read
var user User
db.First(&user, 1)                    // ambil by ID
db.Where("email = ?", "john@example.com").First(&user)  // by email
Read All
var users []User
db.Find(&users)
Update
db.Model(&user).Update("name", "Jane")
db.Model(&user).Updates(User{Name: "Jane", Email: "jane@example.com"})
Delete
db.Delete(&user, 1)  // soft delete (ada kolom deleted_at)
With Relation
// Event punya Category (belongs to)
var event Event
db.Preload("Category").First(&event, 1)

// Category punya banyak Events (has many)
var category Category
db.Preload("Events").First(&category, 1)
AutoMigrate
db.AutoMigrate() otomatis membuat atau update tabel dari struct:
db.AutoMigrate(&model.User{})
db.AutoMigrate(&model.RefreshToken{})
db.AutoMigrate(&model.Category{}, &model.Event{})
Catatan: AutoMigrate hanya menambah kolom baru, tidak menghapus kolom.
Untuk hapus kolom atau ubah type, pakai SQL migration manual.
Models
Auth Service
User (internal/model/user.go):
type User struct {
    gorm.Model
    Email    string `gorm:"type:varchar(255);uniqueIndex;not null"`
    Password string `gorm:"type:varchar(255);not null"`
    Name     string `gorm:"type:varchar(255);not null"`
    Role     string `gorm:"type:varchar(20);default:'user'"`
}
RefreshToken (internal/model/refresh_token.go):
type RefreshToken struct {
    gorm.Model
    UserID    uint      `gorm:"not null"`
    Token     string    `gorm:"type:varchar(512);uniqueIndex;not null"`
    ExpiresAt time.Time `gorm:"not null"`
    User      User      `gorm:"foreignKey:UserID"`
}
Catalog Service
Category (internal/model/category.go):
type Category struct {
    gorm.Model
    Name        string  `gorm:"type:varchar(100);uniqueIndex;not null"`
    Description string  `gorm:"type:text"`
    Events      []Event `gorm:"foreignKey:CategoryID"`
}
Event (internal/model/event.go):
type Event struct {
    gorm.Model
    Name        string    `gorm:"type:varchar(255);not null"`
    Description string    `gorm:"type:text"`
    Venue       string    `gorm:"type:varchar(255);not null"`
    Date        time.Time `gorm:"not null"`
    CategoryID  uint      `gorm:"not null"`
    Price       float64   `gorm:"type:decimal(10,2);not null"`
    Quota       int       `gorm:"not null"`
    Category    Category  `gorm:"foreignKey:CategoryID"`
}
Payment Service
Payment (internal/model/payment.go):
type Payment struct {
    gorm.Model
    ReservationID      string         `gorm:"type:varchar(255);not null"`
    UserID             uint           `gorm:"not null"`
    ProviderPaymentID  string         `gorm:"type:varchar(255)"`
    Gateway            PaymentGateway `gorm:"type:varchar(20);not null"`
    Amount             float64        `gorm:"type:decimal(12,2);not null"`
    Status             PaymentStatus  `gorm:"type:varchar(20);default:'PENDING'"`
    ExpiredAt          time.Time
}
Docker Compose
PostgreSQL Saja
services:
  postgres:
    image: postgres:16-alpine
    container_name: event-ticketing-postgres
    environment:
      POSTGRES_USER: admin
      POSTGRES_PASSWORD: your_password
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data

volumes:
  postgres_data:
Dengan Services
services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: admin
      POSTGRES_PASSWORD: your_password
    ports:
      - "5432:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U admin"]
      interval: 5s
      timeout: 5s
      retries: 5

  auth-service:
    build: ./services/auth-service
    ports:
      - "8081:8081"
    environment:
      DATABASE_URL: postgres://admin:your_password@postgres:5432/event_ticketing_db?sslmode=disable

  catalog-service:
    build: ./services/catalog-service
    ports:
      - "8082:8082"
    environment:
      DATABASE_URL: postgres://admin:your_password@postgres:5432/event_ticketing_db?sslmode=disable

  payment-service:
    build: ./services/payment-service
    ports:
      - "8083:8083"
    environment:
      DATABASE_URL: postgres://admin:your_password@postgres:5432/event_ticketing_db?sslmode=disable
Cara Jalankan
1. Start PostgreSQL
docker-compose up -d postgres
2. Jalankan Service (lokal)
cd services/auth-service
go run ./cmd/server
3. Test Health
curl http://localhost:8081/health
4. Jalankan Semua via Docker
docker-compose up -d --build
Troubleshooting
Error	Solusi
connection refused	PostgreSQL belum jalan, cek docker ps
password authentication failed	Cek password di .env sama dengan docker-compose.yml
database event_ticketing_db does not exist	Jalankan init-databases.sh atau buat manual
relation does not exist	Jalankan db.AutoMigrate() atau cek model
```
