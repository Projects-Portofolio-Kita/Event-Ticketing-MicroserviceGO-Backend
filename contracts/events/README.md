# Kontrak Event Async

Kontrak integrasi **antar-service berbasis event** untuk langkah berikutnya.
Saat ini **belum aktif** — belum ada message broker di repo ini, dan
booking <-> payment masih dipanggil langsung via HTTP:

- `payment-service/internal/service/booking_client.go` →
  `GET /api/v1/reservations/{id}` dan `POST /api/v1/reservations/{id}/confirm`
  (dipakai `payment_service.go`).
- `catalog-service/internal/client/booking_client.go` →
  `GET /api/v1/events/{eventId}/seat-status` (dipakai `catalog_service.go`).

## Daftar event

| File (JSON Schema draft-07) | Producer | Consumer (rencana) | Status |
|---|---|---|---|
| `booking.confirmed.v1.json` | `booking-service` (setelah reservasi `LOCKED` → `CONFIRMED`) | `payment-service` (penandaan audit/pemetaan pembayaran), **`notification-service` — sudah ada dan sudah menerima fakta ini** | **Belum aktif** |
| `payment.succeeded.v1.json` | `payment-service` (setelah pembayaran `SUCCEEDED`) | `booking-service` (menggantikan pemanggilan HTTP `POST /api/v1/reservations/{id}/confirm`), audit/ledger (belum ada) | **Belum aktif** |

> Status: **belum aktif — saat ini booking<->payment dipanggil langsung via HTTP
> (lihat `internal/service` dan `internal/client` pada masing-masing service).**

### Sudah berjalan tanpa broker

Fakta-fakta ini **sudah dikirim dan dikonsumsi hari ini**, hanya lewat HTTP
bukan broker:

- Producer → `libs/common/notify` (`POST {notification-service}/api/v1/notifications`,
  fire-and-forget, timeout 3 detik, error dibuang).
- Consumer → `notification-service` (`:8085`), `kind` yang dikirim:
  `reservation.locked`, `booking.confirmed`, `booking.cancelled`,
  `payment.succeeded`.
- Kontrak payload yang dipakai di jalan inilah yang dibungkus envelope di
  `booking.confirmed.v1.json` / `payment.succeeded.v1.json` — begitu broker
  (Kafka/RabbitMQ) masuk, payload-nya tinggal dipindahkan apa adanya.

## Bentuk envelope

Semua event memakai envelope yang sama:

```json
{
  "eventId": "booking.confirmed.v1",
  "occurredAt": "2026-10-25T08:00:00Z",
  "producer": "booking-service",
  "payload": { }
}
```

- `eventId` — **nama event pada broker**, bukan domain event / ID UUID.
- `occurredAt` — waktu fakta bisnis terjadi (RFC 3339, UTC).
- `producer` — service pembuat event.
- `payload` — data domain (dibedakan per event, didefinisikan di file masing-masing).

## Aturan versioning

1. **Menambah field baru** ke `payload` (atau envelope) **BOLEH** pada versi yang
   sama — konsumen wajib mengabaikan field yang tidak dikenal
   (*forward compatible*).
2. **Menghapus field** atau **mengubah nama field / tipe / makna** =
   **versi baru** (mis. `booking.confirmed.v2.json`), file versi lama tidak
   dihapus/diedit agar konsumen lama tetap jalan.
3. Nama file = `<domain>.<verb>.v<n>.json` dan harus sama dengan isi `eventId`.
4. Perubahan yang bersifat *optional* (default baru, enum baru untuk field yang
   bertipe string longgar) tidak dianggap breaking.
5. Setiap file event harus lolos validasi JSON Schema draft-07 sebelum dipakai.

## Validasi

```powershell
# contoh cepat dengan Node
node -e "const fs=require('fs');JSON.parse(fs.readFileSync('contracts/events/booking.confirmed.v1.json','utf8'));JSON.parse(fs.readFileSync('contracts/events/payment.succeeded.v1.json','utf8'));console.log('ok')"
```
