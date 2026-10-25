# Auth Service (`:8081`)

Authentication microservice for TickId: email registration/login, JWT access
tokens, rotating opaque refresh tokens, Google Sign-In verification, and a
`GET /me` profile endpoint. Same conventions as the other Go services
(Gin + GORM + UUID PKs + `pkg/database` + CORS middleware).

## Endpoints

| Method | Path | Auth | Description |
|---|---|---|---|
| GET | `/health` | — | Liveness probe |
| POST | `/api/v1/auth/register` | — | `{name,email,password}` → 201 session |
| POST | `/api/v1/auth/login` | — | `{email,password}` → 200 session |
| POST | `/api/v1/auth/refresh` | — | `{refreshToken}` → 200 rotated session |
| POST | `/api/v1/auth/google` | — | `{idToken}` (GIS) → 200 session |
| GET | `/api/v1/auth/me` | Bearer JWT | Current user profile |

Session shape (matches the web-portal contract):

```json
{
  "accessToken": "<jwt>",
  "token": "<jwt>",
  "refreshToken": "<opaque>",
  "user": { "id": "<uuid>", "email": "a@b.com", "name": "Nama" }
}
```

Error shape: `{ "message": "..." }` with 400 / 401 / 403 / 404 / 409 / 503.

## Security notes

- Passwords hashed with bcrypt; access JWTs are HS256, 15 min default.
- Refresh tokens are random 256-bit values; only their SHA-256 hash is stored,
  and each use rotates (old token revoked).
- Google ID tokens are verified via Google's tokeninfo endpoint (signature +
  expiry by Google) plus our own `aud == GOOGLE_CLIENT_ID` and
  `email_verified == true` checks. The Google profile alone never becomes a
  session — the service issues our own JWT.
- `JWT_SECRET` must be a strong random value in production
  (`openssl rand -hex 32`). Google login returns 503 while `GOOGLE_CLIENT_ID`
  is empty.

## Tables (AutoMigrate)

- `users(id uuid PK, name, email unique, password_hash, avatar_url, provider, google_sub unique NULL)`
  — `google_sub` is `NULL` for local accounts (Postgres allows many `NULL`s
  under a unique index, so unlimited local signups; only Google accounts carry
  a real sub).
- `refresh_tokens(id uuid PK, user_id, token_hash unique, expires_at, revoked)`

## Run

```powershell
# via compose (env injected from backend/docker-compose.yml)
cd backend
docker compose up -d --build auth-service

# local (needs .env with DATABASE_URL + JWT_SECRET)
cd services/auth-service
go run ./cmd/server
```

## Verify

```powershell
$base = "http://localhost:8081/api/v1/auth"
Invoke-RestMethod "$base/register" -Method POST -ContentType "application/json" `
  -Body '{"name":"Demo","email":"demo@tickid.id","password":"secret123"}'
Invoke-RestMethod "$base/login" -Method POST -ContentType "application/json" `
  -Body '{"email":"demo@tickid.id","password":"secret123"}'
```
