# 🔑 Fitur: Login & Otentikasi (Authentication)

Dokumentasi spesifikasi fitur login dan penerbitan token sesi JWT.

---

## 1. Deskripsi & Aturan Bisnis

1. **Verifikasi Kredensial**: Sistem memvalidasi kombinasi email dan password terhadap hash `bcrypt` di database.
2. **Pengecekan Akun**: Pengguna tidak dapat login jika akun telah dinonaktifkan (`is_active = false`).
3. **Penerbitan Token Ganda (Dual Token Architecture)**:
   * **Access Token (JWT)**: Ditandatangani menggunakan algoritma HS256 dengan masa berlaku pendek (**15 menit**). Digunakan pada header `Authorization: Bearer <token>` untuk mengakses endpoint yang dilindungi.
   * **Refresh Token (JWT)**: Memiliki masa berlaku panjang (**7 hari**). Hash SHA-256 dari token ini disimpan di tabel `refresh_tokens` database untuk validasi sesi dan pencabutan (*revocation*).

---

## 2. Rincian Endpoint

* **URL Path**: `/api/v1/auth/login`
* **HTTP Method**: `POST`
* **Tingkat Akses**: Publik
* **Header yang Dibutuhkan**: `Content-Type: application/json`

---

## 3. Spesifikasi Payload Request

### 3.1 Skema Field Request
| Field | Tipe Data | Wajib? | Keterangan |
|---|---|---|---|
| `email` | string | Ya | Email akun terdaftar |
| `password` | string | Ya | Kata sandi akun |

### 3.2 Contoh Request Body
```json
{
  "email": "budi.santoso@example.com",
  "password": "SuperP@ssword123!"
}
```

---

## 4. Format Response

### 4.1 Sukses: Login Berhasil (200 OK)
```json
{
  "success": true,
  "message": "Login successful",
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiNjczNThjNWMt...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiNjczNThjNWMt...",
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
      "is_active": true,
      "created_at": "2026-09-17T16:12:47.732Z",
      "updated_at": "2026-09-17T16:12:47.732Z"
    }
  }
}
```

### 4.2 Error: Email atau Password Salah (401 Unauthorized)
```json
{
  "success": false,
  "message": "invalid email or password",
  "error": "ERR_UNAUTHORIZED"
}
```

---

## 5. Contoh Perintah Eksekusi

### cURL
```bash
curl -X POST http://localhost:8085/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "budi.santoso@example.com",
    "password": "SuperP@ssword123!"
  }'
```

### PowerShell
```powershell
$body = @{
    email    = "budi.santoso@example.com"
    password = "SuperP@ssword123!"
} | ConvertTo-Json

$resp = Invoke-RestMethod -Uri "http://localhost:8085/api/v1/auth/login" -Method Post -Body $body -ContentType "application/json"
$resp | ConvertTo-Json

# Ambil token untuk pengetesan selanjutnya
$accessToken = $resp.data.access_token
$refreshToken = $resp.data.refresh_token
```
