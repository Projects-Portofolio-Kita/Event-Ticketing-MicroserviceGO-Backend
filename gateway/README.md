# gateway

Single entry point in front of every service. Listens on **:8080 inside the
container**, published as **http://localhost:8086** (host port 8080 was already
taken by another local process — change the `ports:` mapping in
`backend/docker-compose.yml` if you want it back).

## Routes

| Public path | Upstream | Rewrite |
|---|---|---|
| `/api/auth/*` | auth-service `:8081` | `/api/auth` → `/api/v1/auth` |
| `/api/catalog/*` | catalog-service `:8082` | `/api/catalog` → `/api/v1` |
| `/api/booking/*` | booking-service `:8083` | `/api/booking` → `/api/v1` |
| `/api/payment/*` | payment-service `:8084` | `/api/payment` → `/api/v1` |
| `/api/notifications/*` | notification-service `:8085` | `/api/notifications` → `/api/v1/notifications` |
| `/health/<name>` | that service's `/health` | — |
| `/health` | aggregate of all of the above | — |

The rewrite rules are identical to the Vite dev proxy (`web-portal/vite.config.ts`)
and the portal's `nginx.conf`, so a client uses the same paths whether it talks
to the gateway or to the portal.

## Why a gateway when nginx already proxies?

- **One address for clients** (desktop app, mobile, third parties) instead of
  five ports.
- **Aggregated health**: `GET :8086/health` reports every service in one call.
- **Indirection**: upstreams are configured by env vars, so a service can move
  (another host, another port) without touching any client.

## Configuration

See `.env.example`. Each upstream URL has a default matching the compose
network, so it runs with no configuration at all inside compose.

## Run

```powershell
docker compose -f backend/docker-compose.yml up -d gateway
# or locally
$env:AUTH_SERVICE_URL = "http://localhost:8081"   # ...same for the others
go run ./cmd/server
```

```powershell
Invoke-RestMethod http://localhost:8086/health
Invoke-RestMethod http://localhost:8086/api/catalog/events
```

## Errors

- Upstream unreachable → `502` with `{"message": "<service> tidak dapat dihubungi"}`.
- Unknown route → `404` with `{"message": "route tidak ditemukan di gateway"}`.

The body key is `message`, the same envelope as every other service
(`libs/common/httperr`).
