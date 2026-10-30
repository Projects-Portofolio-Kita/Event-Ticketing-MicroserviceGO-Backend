# Contracts

Folder kontrak antar-service untuk proyek event ticketing microservice
(Go + Gin, 4 service). Tujuannya menjadikan **kode sebagai satu-satunya
sumber kebenaran** yang terdokumentasi, sehingga frontend, service lain, dan
integrasi berikutnya tidak menebak-nebak bentuk request/response.

Semua spec ditulis dari pembacaan langsung kode (route di
`services/*/cmd/server/main.go`, DTO di `internal/dto`, model di
`internal/model`, dan field yang benar-benar dikembalikan di
`internal/handler`), lalu dicek silang dengan kontrak yang dipakai
`../web-portal/src/services/*.ts`.

## Lokasi

Folder ini berada di **`backend/contracts/`** — di dalam repo Git `backend`
(root repo = folder `backend/`). Semua path, tree, dan command di bawah ini
dihitung dari **root repo backend**, sehingga folder ini terlihat sebagai
`contracts/`. Rujukan ke repo/folder lain di luar `backend/` diawali `../`
(mis. `../web-portal`, `../infrastucture`).

## Daftar file

```text
contracts/            # = backend/contracts/ (root workspace)
├── openapi/
│   ├── auth.yaml          # auth-service          :8081  prefix /api/v1/auth
│   ├── catalog.yaml       # catalog-service       :8082  prefix /api/v1
│   ├── booking.yaml       # booking-service       :8083  prefix /api/v1
│   ├── payment.yaml       # payment-service       :8084  prefix /api/v1
│   ├── notification.yaml  # notification-service  :8085  prefix /api/v1
│   ├── gateway.yaml       # gateway               :8086  tabel proxy + /health agregat
│   └── portal.yaml        # web-portal proxy      :3000  /api/* + /health/*
├── events/
│   ├── booking.confirmed.v1.json   # JSON Schema draft-07 (belum aktif)
│   ├── payment.succeeded.v1.json   # JSON Schema draft-07 (belum aktif)
│   └── README.md                   # daftar event + aturan versioning
└── README.md
```

| File | Isi | Path count |
|---|---|---|
| `openapi/auth.yaml` | register, login, refresh, google, `GET /me` (JWT), `/health` | 6 |
| `openapi/catalog.yaml` | `GET /events`, `GET /events/{id}/seatmap`, `/health` | 3 |
| `openapi/booking.yaml` | reservasi (create/get/confirm/cancel), seat-status, `/health` | 6 |
| `openapi/payment.yaml` | payments (create/get/confirm), `/health` | 4 |
| `openapi/notification.yaml` | create, list per user, mark-read, `/health` | 4 |
| `openapi/gateway.yaml` | `/health` agregat, `/health/{name}`, `x-proxy-routes` | 2 |
| `openapi/portal.yaml` | kontrak yang dipakai `../web-portal/src/services/*.ts` melalui proxy Vite/nginx | 19 |

Catatan spec tiap service:

- Semua file OpenAPI **3.0.3**, pakai `$ref` internal (`#/components/schemas/...`)
  saja — tidak ada `$ref` lintas file, jadi tiap file bisa dibuka sendiri-sendiri.
- `auth.yaml` adalah satu-satunya spec yang memberi `security` pada operation
  (`GET /api/v1/auth/me`) karena hanya endpoint itu yang dijaga middleware
  `RequireAuth`. booking/payment service belum memverifikasi JWT di server,
  tetapi web-portal tetap mengirim `Authorization: Bearer` (lihat
  `../web-portal/src/services/booking.ts`).
- Bentuk error: `{"message": string}` di **semua** service — termasuk payment,
  yang sebelumnya memakai `{"error": string}` dan kini diseragamkan lewat
  `libs/common/httperr`.
- `gateway.yaml` sengaja hanya mendokumentasikan endpoint milik gateway
  (`/health`, `/health/{name}`) plus tabel proxy di `x-proxy-routes`; skema
  request/response bisnis tetap hidup di spec service masing-masing, supaya
  tidak ada dua sumber kebenaran untuk bentuk yang sama.
- Field payment memakai snake_case (`reservation_id`, `qr_string`,
  `expired_at`, ...), sedangkan auth/booking memakai camelCase
  (`reservationId`, `expiresAt`, ...) — persis seperti di kode.

## Cara pakai

Buka di **Swagger Editor** (<https://editor.swagger.io>) dengan
*File → Import file*, atau drag & drop file `contracts/openapi/*.yaml`.

Redoc (CLI, sekali jalan):

```powershell
npx @redocly/cli preview-docs contracts/openapi/auth.yaml
```

Swagger UI statis:

```powershell
npx @redocly/cli preview-docs contracts/openapi/booking.yaml
```

Validasi YAML + JSON cepat (Node, tanpa dependensi repo):

```powershell
node -e "const fs=require('fs');const y=require('js-yaml');for(const f of fs.readdirSync('contracts/openapi')){y.load(fs.readFileSync('contracts/openapi/'+f,'utf8'));console.log('ok',f)}"
```

## Catatan: spec lama yang sudah SUPERSEDED

`../../infrastucture/spec/docs.yml` adalah **draft lama** spesifikasi payment
(mis. memakai `ErrorResponse { message }`, field camelCase, dan berkasnya
tidak lagi mencerminkan handler). Kontrak resmi pembayaran sekarang adalah
`contracts/openapi/payment.yaml`.

File lama tersebut **tidak dihapus dan tidak diubah** — dibiarkan sebagai
arsip; jangan dipakai sebagai referensi integrasi baru.

## Sumber kebenaran

| Spec | Route | DTO/Response |
|---|---|---|
| auth | `services/auth-service/cmd/server/main.go` | `internal/dto/register_request.go`, `internal/handler/auth_handler.go`, `model.PublicUser` |
| catalog | `services/catalog-service/cmd/server/main.go` | `internal/dto/catalog_response.go`, `internal/handler/catalog_handler.go` |
| booking | `services/booking-service/cmd/server/main.go` | `internal/dto/reservation_{request,response}.go`, `internal/handler/reservation_handler.go` |
| payment | `services/payment-service/cmd/server/main.go` | `internal/dto/payment_{request,response}.go`, `internal/handler/payment_handler.go` |
| portal | `../web-portal/vite.config.ts`, `../web-portal/nginx.conf` | `../web-portal/src/services/{auth,catalog,booking,health}.ts` |
