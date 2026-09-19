# 🚪 Fitur: Logout & Pencabutan Sesi (Token Revocation)

Dokumentasi spesifikasi fitur logout untuk mengakhiri sesi aktif pengguna.

---

## 1. Deskripsi & Aturan Bisnis

1. **Pencabutan Token (Revocation)**:
   * Pengguna wajib menyertakan **Bearer Token** pada header Authorization untuk membuktikan kepemilikan akun.
   * Jika body menyertakan field `refresh_token`, maka sistem hanya mencabut sesi perangkat tersebut.
   * Jika body tidak menyertakan `refresh_token`, sistem mencabut **seluruh refresh token** milik pengguna tersebut (logout dari seluruh perangkat).
2. **Setelah Logout**: Refresh token yang telah dicabut tidak akan bisa lagi digunakan untuk memanggil endpoint `/api/v1/auth/refresh`.

---

## 2. Rincian Endpoint

* **URL Path**: `/api/v1/auth/logout`
* **HTTP Method**: `POST`
* **Tingkat Akses**: Terlindungi (Memerlukan JWT Bearer Token)
* **Header yang Dibutuhkan**:
  * `Authorization: Bearer <access_token>`
  * `Content-Type: application/json`

---

## 3. Spesifikasi Payload Request

### 3.1 Skema Field Request (Opsional)
| Field | Tipe Data | Wajib? | Keterangan |
|---|---|---|---|
| `refresh_token` | string | Tidak | Token refresh yang ingin dicabut secara spesifik |

### 3.2 Contoh Request Body
```json
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiNjczNThjNWMt..."
}
```

---

## 4. Format Response

### 4.1 Sukses: Logout Berhasil (200 OK)
```json
{
  "success": true,
  "message": "Logged out successfully"
}
```

### 4.2 Error: Token Authorization Tidak Ada / Tidak Valid (401 Unauthorized)
```json
{
  "success": false,
  "message": "Authorization header is required",
  "error": "ERR_UNAUTHORIZED"
}
```

---

## 5. Contoh Perintah Eksekusi

### cURL
```bash
curl -X POST http://localhost:8085/api/v1/auth/logout \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "<refresh_token>"
  }'
```

### PowerShell
```powershell
$headers = @{
    Authorization = "Bearer $accessToken"
}
$body = @{
    refresh_token = $refreshToken
} | ConvertTo-Json

Invoke-RestMethod -Uri "http://localhost:8085/api/v1/auth/logout" -Method Post -Headers $headers -Body $body -ContentType "application/json" | ConvertTo-Json
```
