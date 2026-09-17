# 🔐 Fitur: Lupa & Reset Password (Password Recovery)

Dokumentasi alur pemulihan kata sandi pengguna secara aman.

---

## 1. Deskripsi Alur Bisnis

Alur pemulihan kata sandi terdiri dari 2 tahapan:
1. **Permintaan Tautan Reset (`forgot-password`)**: Pengguna memasukkan email akun mereka.
2. **Eksekusi Penggantian Password (`reset-password`)**: Pengguna mengirimkan token verifikasi reset yang diterima bersama password baru.

### Kebijakan Keamanan:
* **Perlindungan Anti-Enumerasi Email**: Endpoint `forgot-password` selalu mengembalikan status `200 OK` dengan pesan sukses generik, baik email tersebut ada maupun tidak di database. Hal ini mencegah hacker mengetahui daftar email yang terdaftar pada sistem (*prevent user enumeration attack*).
* **Pencabutan Sesi Otomatis**: Ketika password berhasil diganti melalui `reset-password`, seluruh sesi aktif (refresh tokens) yang dimiliki akun tersebut otomatis dicabut demi keamanan akun.

---

## 2. Rincian Endpoint

### 2.1 Minta Tautan Reset Password
* **URL Path**: `/api/v1/auth/forgot-password`
* **HTTP Method**: `POST`
* **Tingkat Akses**: Publik
* **Header yang Dibutuhkan**: `Content-Type: application/json`

#### Request Body
```json
{
  "email": "budi.santoso@example.com"
}
```

#### Response: 200 OK
```json
{
  "success": true,
  "message": "If your email is registered, a password reset link has been sent."
}
```

---

### 2.2 Eksekusi Reset Password Baru
* **URL Path**: `/api/v1/auth/reset-password`
* **HTTP Method**: `POST`
* **Tingkat Akses**: Publik
* **Header yang Dibutuhkan**: `Content-Type: application/json`

#### Request Body
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "new_password": "NewSuperPassword2026!"
}
```

| Field | Tipe Data | Wajib? | Keterangan |
|---|---|---|---|
| `token` | string | Ya | Token JWT verifikasi reset password |
| `new_password` | string | Ya | Password baru (minimal 8 karakter) |

#### Response Sukses: 200 OK
```json
{
  "success": true,
  "message": "Password reset successful. Please login with your new password."
}
```

#### Response Error: 400 Bad Request (Token Tidak Valid/Kedaluwarsa)
```json
{
  "success": false,
  "message": "invalid or expired reset token",
  "error": "ERR_RESET_PASSWORD_FAILED"
}
```

---

## 3. Contoh Perintah Eksekusi

### cURL
```bash
# 1. Request Reset
curl -X POST http://localhost:8085/api/v1/auth/forgot-password \
  -H "Content-Type: application/json" \
  -d '{"email": "budi.santoso@example.com"}'

# 2. Submit Password Baru
curl -X POST http://localhost:8085/api/v1/auth/reset-password \
  -H "Content-Type: application/json" \
  -d '{
    "token": "<reset_token>",
    "new_password": "NewSuperPassword2026!"
  }'
```

### PowerShell
```powershell
# 1. Request Reset
$body1 = @{ email = "budi.santoso@example.com" } | ConvertTo-Json
Invoke-RestMethod -Uri "http://localhost:8085/api/v1/auth/forgot-password" -Method Post -Body $body1 -ContentType "application/json" | ConvertTo-Json

# 2. Submit Password Baru
$body2 = @{
    token        = "<reset_token>"
    new_password = "NewSuperPassword2026!"
} | ConvertTo-Json
Invoke-RestMethod -Uri "http://localhost:8085/api/v1/auth/reset-password" -Method Post -Body $body2 -ContentType "application/json" | ConvertTo-Json
```
