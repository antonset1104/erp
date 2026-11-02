# Kebijakan Keamanan (Security Policy)

Keamanan data finansial, operasional, dan privasi pengguna adalah prioritas tertinggi dalam pengembangan **Odoo-Go ERP**. Kami mengapresiasi upaya komunitas peneliti keamanan dan pengembang yang membantu menjaga integritas sistem kami melalui pengungkapan kerentanan yang bertanggung jawab (*responsible disclosure*).

---

## 🛡️ Versi yang Didukung (Supported Versions)

Pembaruan keamanan aktif dan patch kerentanan dirilis untuk versi rilis berikut:

| Versi | Didukung Pembaruan Keamanan | Status Pemeliharaan |
| :--- | :---: | :--- |
| `1.41.x` | ✅ Ya | Versi Produksi Aktif (Current Release) |
| `1.40.x` | ✅ Ya | Dukungan Patch Keamanan Kritis |
| `< 1.40.0`| ❌ Tidak | End-of-Life (Harap segera upgrade) |

---

## 🚨 Melaporkan Kerentanan (Reporting a Vulnerability)

**JANGAN** melaporkan kerentanan keamanan melalui *public GitHub Issue* atau media sosial terbuka.

Jika Anda menemukan potensi kerentanan keamanan (seperti kebocoran data, bypass autentikasi JWT, SQL Injection, Cross-Site Scripting / XSS, Remote Code Execution, atau eskalasi hak akses RBAC), ikuti langkah-langkah berikut:

### Opsi 1: GitHub Private Vulnerability Reporting (Disarankan)
1. Kunjungi repositori proyek: [https://github.com/antonset1104/erp](https://github.com/antonset1104/erp)
2. Klik tab **Security** $\rightarrow$ **Advisories** $\rightarrow$ **Report a vulnerability**.
3. Lampirkan rincian teknis, langkah reproduksi (*Proof-of-Concept* / PoC), dan dampak potensial secara privat.

### Opsi 2: Email Langsung ke Tim Keamanan
Kirimkan rincian laporan secara terenkripsi/rahasia ke:
📧 **antonset1104@users.noreply.github.com**

Sertakan informasi berikut dalam laporan Anda:
- Deskripsi kerentanan dan lokasi modul yang terpengaruh (misal `internal/core/database`, `internal/modules/account`).
- Langkah-langkah detail untuk mereproduksi masalah (curl command, script PoC, atau tangkapan layar HTTP payload).
- Analisis dampak dan tingkat keparahan menurut penilaian Anda (Low, Medium, High, Critical).
- Usulan perbaikan atau mitigasi (jika Anda memilikinya).

---

## ⏱️ Komitmen Respons & SLA Penanganan

1. **Konfirmasi Awal**: Tim maintainer akan mengonfirmasi penerimaan laporan Anda dalam kurun waktu **maksimal 48 jam**.
2. **Triase & Verifikasi**: Investigasi dan verifikasi keparahan kerentanan akan diselesaikan dalam kurun waktu **3-5 hari kerja**.
3. **Perbaikan & Pengujian**: Tim akan menyusun patch perbaikan di repositori privat terpisah dan melakukan regresi pengujian otomatis.
4. **Rilis Patch Keamanan**: Patch keamanan akan dirilis ke publik dalam versi patch terbaru (misal `v1.41.1`) dalam kurun waktu **maksimal 14 hari** sejak kerentanan diverifikasi.
5. **Kredit & Pengakuan**: Nama atau persona Anda akan dicantumkan secara terhormat dalam *Security Advisory Notes* dan *Release Notes* publik sebagai bentuk apresiasi kami.

---

## 🔒 Praktik Keamanan Bawaan Odoo-Go ERP

Platform ini telah menerapkan standar keamanan enterprise terkemuka:
- **Autentikasi Aman**: JWT (*JSON Web Tokens*) berdurasi terbatas dengan secret key minimum 32-karakter.
- **Hashing Password**: Bcrypt dengan work factor $\ge 10$.
- **Database Safety**: Prepared statements / parameter binding otomatis via GORM untuk mencegah SQL Injection.
- **Proteksi XSS**: Sanitasi HTML bawaan pada web client dan rendering DOM berbasis teks murni.
- **Isolasi Multi-Tenant**: Filter otomatis `company_id` pada seluruh kueri database.
- **Jejak Audit ISO 27001**: Modul `audit_trail` mencatat setiap mutasi data sensitif (CDC & user attribution).
