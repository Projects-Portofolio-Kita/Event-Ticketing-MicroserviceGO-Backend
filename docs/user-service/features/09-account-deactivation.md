# 🛑 Fitur: Deaktivasi Akun Pengguna (Account Deactivation)

Dokumentasi spesifikasi fitur penonaktifan akun secara mandiri oleh pengguna.

---

## 1. Deskripsi & Aturan Bisnis

1. **Self-Deactivation**: Pengguna yang sedang login dapat mengajukan penonaktifan akun mereka sendiri.
2. **Soft Deactivation (Bukan Hapus Fisik)**:
   * Baris data di database tidak langsung di-`DELETE` secara permanen.
   * Nilai field diubah menjadi **`is_active = false`**.
   * Hal ini bertujuan untuk mempertahankan jejak audit (*audit trail*) dan riwayat pemesanan tiket pada service lain (Booking & Payment).
3. **Pencabutan Sesi Seketika**:
   * Seluruh refresh token milik pengguna ini otomatis dicabut (`revoked = true`).
   * Setelah akun dinonaktifkan, pengguna tidak akan dapat login kembali menggunakan email tersebut.

---

## 2. Rincian Endpoint

* **URL Path**: `/api/v1/users/me`
* **HTTP Method**: `DELETE`
* **Tingkat Akses**: Terlindungi (Memerlukan JWT Bearer Token)
* **Header yang Dibutuhkan**: `Authorization: Bearer <access_token>`

---

## 3. Parameter Request
Tidak membutuhkan query param ataupun request body. Identitas akun diambil langsung dari token JWT pengguna.

---

## 4. Format Response

### 4.1 Sukses: Akun Berhasil Dinonaktifkan (200 OK)
```json
{
  "success": true,
  "message": "Account deactivated successfully"
}
```

### 4.2 Error: Tidak Ada Token Autentikasi (401 Unauthorized)
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
curl -X DELETE http://localhost:8085/api/v1/users/me \
  -H "Authorization: Bearer <access_token>"
```

### PowerShell
```powershell
$headers = @{
    Authorization = "Bearer $accessToken"
}

Invoke-RestMethod -Uri "http://localhost:8085/api/v1/users/me" -Method Delete -Headers $headers | ConvertTo-Json
```
