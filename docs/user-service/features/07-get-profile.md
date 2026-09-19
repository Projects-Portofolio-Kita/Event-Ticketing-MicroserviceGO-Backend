# 👤 Fitur: Ambil Profil Pengguna Login (Get Current Profile)

Dokumentasi spesifikasi fitur untuk melihat data profil diri sendiri bagi pengguna yang sedang terautentikasi.

---

## 1. Deskripsi & Aturan Bisnis

1. **Autentikasi Otomatis via Token**: Endpoint ini tidak memerlukan parameter ID di URL. Sistem membaca identitas pengguna (`user_id`) langsung dari payload **JWT Bearer Token** yang disematkan pada header request.
2. **Kerahasiaan Data**: Hanya mengembalikan data publik profil pengguna, tidak menyertakan password hash atau token database.
3. **Akun Aktif**: Hanya akun berstatus `is_active = true` yang dapat mengakses profil.

---

## 2. Rincian Endpoint

* **URL Path**: `/api/v1/users/me`
* **HTTP Method**: `GET`
* **Tingkat Akses**: Terlindungi (Memerlukan JWT Bearer Token)
* **Header yang Dibutuhkan**: `Authorization: Bearer <access_token>`

---

## 3. Parameter Request
Tidak ada parameter URL query ataupun body request.

---

## 4. Format Response

### 4.1 Sukses: Profil Ditemukan (200 OK)
```json
{
  "success": true,
  "message": "Profile retrieved successfully",
  "data": {
    "id": "67358c5c-3905-43c2-9a14-ac2ee3277a4c",
    "email": "budi.santoso@example.com",
    "first_name": "Budi",
    "last_name": "Santoso",
    "phone": "081234567890",
    "avatar_url": null,
    "role": "CUSTOMER",
    "email_verified": false,
    "is_active": true,
    "created_at": "2026-09-17T16:12:47.732Z",
    "updated_at": "2026-09-17T16:12:47.732Z"
  }
}
```

### 4.2 Error: Tidak Ada Token (401 Unauthorized)
```json
{
  "success": false,
  "message": "Authorization header is required",
  "error": "ERR_UNAUTHORIZED"
}
```

### 4.3 Error: Token Kedaluwarsa / Signature Tidak Sah (401 Unauthorized)
```json
{
  "success": false,
  "message": "Invalid or expired token",
  "error": "ERR_INVALID_TOKEN"
}
```

---

## 5. Contoh Perintah Eksekusi

### cURL
```bash
curl -X GET http://localhost:8085/api/v1/users/me \
  -H "Authorization: Bearer <access_token>"
```

### PowerShell
```powershell
$headers = @{
    Authorization = "Bearer $accessToken"
}

Invoke-RestMethod -Uri "http://localhost:8085/api/v1/users/me" -Method Get -Headers $headers | ConvertTo-Json
```
