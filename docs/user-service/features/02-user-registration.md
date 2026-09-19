# 📝 Fitur: Registrasi Pengguna (User Registration)

Dokumentasi spesifikasi fitur pendaftaran akun baru pada platform Event Ticketing.

---

## 1. Deskripsi & Aturan Bisnis

1. **Email Unik**: Setiap pengguna wajib mendaftar menggunakan alamat email yang belum pernah terdaftar sebelumnya di sistem.
2. **Validasi Password**: Password minimal memiliki panjang **8 karakter**. Password di-hash secara aman menggunakan algoritma **bcrypt** (cost default: 10) sebelum disimpan di database.
3. **Role Pengguna**:
   * Jika field `role` tidak diisi, sistem otomatis memberikan role default: **`CUSTOMER`**.
   * Pilihan role yang sah: `CUSTOMER`, `ORGANIZER`, `ADMIN`.
4. **Status Awal**: Akun yang baru didaftarkan berstatus aktif (`is_active = true`) dan status verifikasi email bernilai belum diverifikasi (`email_verified = false`).
5. **Keamanan Data**: Response JSON tidak akan pernah membocorkan field `password_hash`.

---

## 2. Rincian Endpoint

* **URL Path**: `/api/v1/auth/register`
* **HTTP Method**: `POST`
* **Tingkat Akses**: Publik
* **Header yang Dibutuhkan**: `Content-Type: application/json`

---

## 3. Spesifikasi Payload Request

### 3.1 Skema Field Request
| Field | Tipe Data | Wajib? | Aturan Validasi | Keterangan |
|---|---|---|---|---|
| `email` | string | Ya | Valid email format | Alamat email unik |
| `password` | string | Ya | Min: 8 karakter | Password akun |
| `first_name` | string | Ya | Min: 2, Max: 100 | Nama depan |
| `last_name` | string | Tidak | Max: 100 | Nama belakang |
| `phone` | string | Tidak | Max: 20 | Nomor HP / WhatsApp |
| `role` | string | Tidak | `CUSTOMER`, `ORGANIZER`, `ADMIN` | Default: `CUSTOMER` |

### 3.2 Contoh Request Body
```json
{
  "email": "budi.santoso@example.com",
  "password": "SuperP@ssword123!",
  "first_name": "Budi",
  "last_name": "Santoso",
  "phone": "081234567890",
  "role": "CUSTOMER"
}
```

---

## 4. Format Response

### 4.1 Sukses: Pengguna Berhasil Didaftarkan (201 Created)
```json
{
  "success": true,
  "message": "User registered successfully",
  "data": {
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
```

### 4.2 Error: Email Sudah Terdaftar (400 Bad Request)
```json
{
  "success": false,
  "message": "email already registered",
  "error": "ERR_REGISTER_FAILED"
}
```

### 4.3 Error: Validasi Input Gagal (400 Bad Request)
```json
{
  "success": false,
  "message": "Key: 'RegisterRequest.Password' Error:Field validation for 'Password' failed on the 'min' tag",
  "error": "ERR_BAD_REQUEST"
}
```

---

## 5. Contoh Perintah Eksekusi

### cURL
```bash
curl -X POST http://localhost:8085/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "budi.santoso@example.com",
    "password": "SuperP@ssword123!",
    "first_name": "Budi",
    "last_name": "Santoso",
    "phone": "081234567890",
    "role": "CUSTOMER"
  }'
```

### PowerShell
```powershell
$body = @{
    email      = "budi.santoso@example.com"
    password   = "SuperP@ssword123!"
    first_name = "Budi"
    last_name  = "Santoso"
    phone      = "081234567890"
    role       = "CUSTOMER"
} | ConvertTo-Json

Invoke-RestMethod -Uri "http://localhost:8085/api/v1/auth/register" -Method Post -Body $body -ContentType "application/json" | ConvertTo-Json
```
