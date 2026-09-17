# 📂 Indeks Dokumentasi Fitur User Service

Koleksi file dokumentasi modular untuk setiap fitur yang tersedia pada **`user-service`**. Setiap file merinci aturan bisnis, spesifikasi endpoint, payload request, skema response, dan contoh perintah eksekusinya.

---

## 📑 Daftar Berkas Dokumentasi Fitur

| No | Fitur | File Dokumentasi | Endpoint | Akses |
|:---:|---|---|---|:---:|
| 1 | **Healthcheck** | [01-healthcheck.md](./01-healthcheck.md) | `GET /health` | Publik |
| 2 | **Registrasi Pengguna** | [02-user-registration.md](./02-user-registration.md) | `POST /api/v1/auth/register` | Publik |
| 3 | **Login & Otentikasi** | [03-login-authentication.md](./03-login-authentication.md) | `POST /api/v1/auth/login` | Publik |
| 4 | **Pembaruan Token Sesi** | [04-token-refresh.md](./04-token-refresh.md) | `POST /api/v1/auth/refresh` | Publik |
| 5 | **Logout & Pencabutan Sesi** | [05-logout.md](./05-logout.md) | `POST /api/v1/auth/logout` | JWT Bearer |
| 6 | **Lupa & Reset Password** | [06-password-reset.md](./06-password-reset.md) | `POST /api/v1/auth/forgot-password`<br>`POST /api/v1/auth/reset-password` | Publik |
| 7 | **Ambil Profil Sendiri** | [07-get-profile.md](./07-get-profile.md) | `GET /api/v1/users/me` | JWT Bearer |
| 8 | **Perbarui Profil** | [08-update-profile.md](./08-update-profile.md) | `PATCH /api/v1/users/me` | JWT Bearer |
| 9 | **Deaktivasi Akun** | [09-account-deactivation.md](./09-account-deactivation.md) | `DELETE /api/v1/users/me` | JWT Bearer |
| 10 | **Inspeksi Pengguna (Admin)** | [10-admin-inspection.md](./10-admin-inspection.md) | `GET /api/v1/users/:id` | JWT + Role Admin |

---

## 🔗 Dokumentasi Terkait
* 🗄️ [Dokumentasi Database & DDL (DATABASE.md)](../DATABASE.md)
* 📘 [Panduan Arsitektur & Pengerjaan (README.md)](../README.md)
