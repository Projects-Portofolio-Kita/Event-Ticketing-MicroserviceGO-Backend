# 🩺 Fitur: Healthcheck & System Status

Dokumentasi spesifikasi fitur pemantauan kesehatan service dan status koneksi database.

---

## 1. Deskripsi Fitur
Endpoint ini digunakan untuk memantau apakah `user-service` aktif (liveness) dan apakah aplikasi berhasil terhubung ke database PostgreSQL (readiness). Sangat penting untuk orchestrator seperti Docker Compose, Kubernetes, load balancer, atau monitoring service (Prometheus/Grafana).

---

## 2. Rincian Endpoint

* **URL Path**: `/health`
* **HTTP Method**: `GET`
* **Tingkat Akses**: Publik (Tidak membutuhkan otentikasi/token)
* **Header yang Dibutuhkan**: Tidak ada

---

## 3. Parameter Request
Tidak ada parameter query ataupun body.

---

## 4. Format Response

### 4.1 Sukses: Service Aktif & Database Terhubung (200 OK)
```json
{
  "database": "connected",
  "service": "user-service",
  "status": "ok"
}
```

### 4.2 Sukses: Service Aktif Tetapi Database Terputus (200 OK)
Jika database PostgreSQL down, service tetap merespons status `ok` dengan indikator database terputus:
```json
{
  "database": "disconnected",
  "service": "user-service",
  "status": "ok"
}
```

---

## 5. Contoh Perintah Eksekusi

### cURL
```bash
curl -X GET http://localhost:8085/health
```

### PowerShell
```powershell
Invoke-RestMethod -Uri "http://localhost:8085/health" -Method Get | ConvertTo-Json
```
