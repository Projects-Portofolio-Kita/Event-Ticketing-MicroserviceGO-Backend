# 🛡️ Fitur: Inspeksi Pengguna oleh Admin (Admin User Inspection)

Dokumentasi spesifikasi fitur inspeksi profil pengguna tertentu berdasarkan UUID khusus untuk akun administrator.

---

## 1. Deskripsi & Aturan Otorisasi (RBAC)

1. **Role-Based Access Control (RBAC)**: Endpoint ini dilindungi oleh middleware `RequireRoles(model.RoleAdmin)`.
2. **Aturan Hak Akses**:
   * Hanya token dengan klaim role **`ADMIN`** yang diizinkan mengakses data.
   * Pengguna dengan role `CUSTOMER` atau `ORGANIZER` otomatis ditolak dengan pesan error **`403 Forbidden`**.
3. **Lookup Berdasarkan UUID**: Admin dapat memeriksa profil akun pengguna siapa saja dengan memasukkan User UUID pada parameter path URL.

---

## 2. Rincian Endpoint

* **URL Path**: `/api/v1/users/:id`
* **HTTP Method**: `GET`
* **Tingkat Akses**: Terlindungi Khusus Role **`ADMIN`**
* **Header yang Dibutuhkan**: `Authorization: Bearer <admin_access_token>`

---

## 3. Parameter URL Path

| Parameter | Tipe Data | Wajib? | Keterangan |
|---|---|---|---|
| `id` | UUID string | Ya | UUID pengguna yang ingin diinspeksi |

Contoh: `/api/v1/users/67358c5c-3905-43c2-9a14-ac2ee3277a4c`

---

## 4. Format Response

### 4.1 Sukses: Data Pengguna Ditemukan (200 OK)
```json
{
  "success": true,
  "message": "User found",
  "data": {
    "id": "67358c5c-3905-43c2-9a14-ac2ee3277a4c",
    "email": "budi.santoso@example.com",
    "first_name": "Budi Hendra",
    "last_name": "Santoso",
    "phone": "089876543210",
    "role": "CUSTOMER",
    "email_verified": false,
    "is_active": true,
    "created_at": "2026-09-17T16:12:47.732Z",
    "updated_at": "2026-09-17T16:12:47.809Z"
  }
}
```

### 4.2 Error: Bukan Role Admin (403 Forbidden)
```json
{
  "success": false,
  "message": "Access denied: insufficient permissions",
  "error": "ERR_INSUFFICIENT_PERMISSIONS"
}
```

### 4.3 Error: User Tidak Ditemukan / Non-Aktif (404 Not Found)
```json
{
  "success": false,
  "message": "user not found",
  "error": "ERR_USER_NOT_FOUND"
}
```

### 4.4 Error: Format ID Bukan UUID Valid (400 Bad Request)
```json
{
  "success": false,
  "message": "Invalid user ID format",
  "error": "ERR_INVALID_UUID"
}
```

---

## 5. Contoh Perintah Eksekusi

### cURL
```bash
curl -X GET http://localhost:8085/api/v1/users/67358c5c-3905-43c2-9a14-ac2ee3277a4c \
  -H "Authorization: Bearer <admin_access_token>"
```

### PowerShell
```powershell
$adminHeaders = @{
    Authorization = "Bearer $adminAccessToken"
}

$targetUserId = "67358c5c-3905-43c2-9a14-ac2ee3277a4c"
Invoke-RestMethod -Uri "http://localhost:8085/api/v1/users/$targetUserId" -Method Get -Headers $adminHeaders | ConvertTo-Json
```
