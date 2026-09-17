# ✏️ Fitur: Perbarui Profil Pengguna (Update Profile)

Dokumentasi spesifikasi fitur pembaruan informasi profil pengguna.

---

## 1. Deskripsi & Aturan Bisnis

1. **Pembaruan Parsial (Partial Update)**: Menggunakan method HTTP `PATCH`. Pengguna hanya perlu mengirimkan field yang ingin diubah. Field yang tidak disertakan dalam JSON request tidak akan dimodifikasi di database.
2. **Field yang Dapat Diubah**:
   * `first_name` (nama depan)
   * `last_name` (nama belakang)
   * `phone` (nomor telepon)
   * `avatar_url` (tautan foto profil)
3. **Field yang Terproteksi**: Email dan Role tidak dapat diubah melalui endpoint ini untuk menjaga integritas akun.

---

## 2. Rincian Endpoint

* **URL Path**: `/api/v1/users/me`
* **HTTP Method**: `PATCH`
* **Tingkat Akses**: Terlindungi (Memerlukan JWT Bearer Token)
* **Header yang Dibutuhkan**:
  * `Authorization: Bearer <access_token>`
  * `Content-Type: application/json`

---

## 3. Spesifikasi Payload Request

### 3.1 Skema Field Request (Semua Opsional)
| Field | Tipe Data | Aturan Validasi | Keterangan |
|---|---|---|---|
| `first_name` | string | Min: 2, Max: 100 | Nama depan baru |
| `last_name` | string | Max: 100 | Nama belakang baru |
| `phone` | string | Max: 20 | Nomor telepon baru |
| `avatar_url` | string | Format URL valid | URL gambar avatar profil |

### 3.2 Contoh Request Body
```json
{
  "first_name": "Budi Hendra",
  "phone": "089876543210",
  "avatar_url": "https://cdn.example.com/avatars/budi-hendra.png"
}
```

---

## 4. Format Response

### 4.1 Sukses: Profil Berhasil Diperbarui (200 OK)
```json
{
  "success": true,
  "message": "Profile updated successfully",
  "data": {
    "id": "67358c5c-3905-43c2-9a14-ac2ee3277a4c",
    "email": "budi.santoso@example.com",
    "first_name": "Budi Hendra",
    "last_name": "Santoso",
    "phone": "089876543210",
    "avatar_url": "https://cdn.example.com/avatars/budi-hendra.png",
    "role": "CUSTOMER",
    "email_verified": false,
    "is_active": true,
    "created_at": "2026-09-17T16:12:47.732Z",
    "updated_at": "2026-09-17T16:12:47.809Z"
  }
}
```

### 4.2 Error: Format URL Avatar Salah (400 Bad Request)
```json
{
  "success": false,
  "message": "Key: 'UpdateProfileRequest.AvatarURL' Error:Field validation for 'AvatarURL' failed on the 'url' tag",
  "error": "ERR_BAD_REQUEST"
}
```

---

## 5. Contoh Perintah Eksekusi

### cURL
```bash
curl -X PATCH http://localhost:8085/api/v1/users/me \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "first_name": "Budi Hendra",
    "phone": "089876543210"
  }'
```

### PowerShell
```powershell
$headers = @{
    Authorization = "Bearer $accessToken"
}
$body = @{
    first_name = "Budi Hendra"
    phone      = "089876543210"
} | ConvertTo-Json

Invoke-RestMethod -Uri "http://localhost:8085/api/v1/users/me" -Method Patch -Headers $headers -Body $body -ContentType "application/json" | ConvertTo-Json
```
