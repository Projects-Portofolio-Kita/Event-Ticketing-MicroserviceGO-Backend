# 🔄 Fitur: Pembaruan Token Sesi (Token Refresh & Rotation)

Dokumentasi spesifikasi fitur perpanjangan sesi login menggunakan Refresh Token dengan sistem keamanan Token Rotation.

---

## 1. Deskripsi & Aturan Bisnis

1. **Memperpanjang Sesi Tanpa Re-login**: Ketika Access Token kedaluwarsa setelah 15 menit, aplikasi client (frontend/mobile) dapat meminta Access Token baru menggunakan Refresh Token yang masih berlaku.
2. **Sistem Keamanan Token Rotation**:
   * Setiap kali endpoint ini dipanggil, Refresh Token yang dikirimkan **langsung dicabut / di-revoke (`revoked = true`)** di database.
   * Server kemudian menerbitkan **Access Token baru** DAN **Refresh Token baru**.
   * Jika ada pihak ketiga yang mencoba menggunakan token refresh lama yang sudah terpakai (*replay attack*), request akan langsung ditolak.
3. **Validasi Session**: Refresh token ditolak jika:
   * Format tanda tangan JWT tidak valid.
   * Masa berlaku token sudah habis (> 7 hari).
   * Token telah dicabut (misal sudah pernah dipakai atau user telah logout).

---

## 2. Rincian Endpoint

* **URL Path**: `/api/v1/auth/refresh`
* **HTTP Method**: `POST`
* **Tingkat Akses**: Publik (Menggunakan token refresh pada body)
* **Header yang Dibutuhkan**: `Content-Type: application/json`

---

## 3. Spesifikasi Payload Request

### 3.1 Skema Field Request
| Field | Tipe Data | Wajib? | Keterangan |
|---|---|---|---|
| `refresh_token` | string | Ya | String refresh token yang didapat saat login |

### 3.2 Contoh Request Body
```json
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiNjczNThjNWMt..."
}
```

---

## 4. Format Response

### 4.1 Sukses: Token Berhasil Diperbarui (200 OK)
```json
{
  "success": true,
  "message": "Token refreshed successfully",
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...new_access_token...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIs...new_refresh_token...",
    "token_type": "Bearer",
    "expires_in": 900,
    "user": {
      "id": "67358c5c-3905-43c2-9a14-ac2ee3277a4c",
      "email": "budi.santoso@example.com",
      "first_name": "Budi",
      "last_name": "Santoso",
      "phone": "081234567890",
      "role": "CUSTOMER",
      "email_verified": false,
      "is_active": true
    }
  }
}
```

### 4.2 Error: Refresh Token Tidak Valid / Dicabut (401 Unauthorized)
```json
{
  "success": false,
  "message": "refresh token is revoked or expired",
  "error": "ERR_REFRESH_FAILED"
}
```

---

## 5. Contoh Perintah Eksekusi

### cURL
```bash
curl -X POST http://localhost:8085/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
  }'
```

### PowerShell
```powershell
$body = @{
    refresh_token = $refreshToken
} | ConvertTo-Json

Invoke-RestMethod -Uri "http://localhost:8085/api/v1/auth/refresh" -Method Post -Body $body -ContentType "application/json" | ConvertTo-Json
```
