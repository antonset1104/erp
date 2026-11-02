# Panduan Kontribusi (Contributing Guide)

Terima kasih atas minat Anda untuk berkontribusi pada **GoERP**! Kami menyambut segala bentuk kontribusi dari komunitas pengembang open source di seluruh dunia, mulai dari pelaporan bug (*bug reports*), usulan fitur (*feature proposals*), perbaikan dokumentasi (*documentation improvements*), hingga pengiriman kode (*Pull Requests*).

---

## 📜 Kode Etik (Code of Conduct)

Sebagai kontributor dan pengelola proyek ini, kita berkomitmen untuk menciptakan komunitas yang ramah, inklusif, dan bebas pelecehan bagi semua orang. Harap membaca dan mematuhi [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) kami sebelum berpartisipasi.

---

## 🛠️ Persiapan Lingkungan Pengembangan (Development Setup)

### Prasyarat
1. **Golang 1.22 atau lebih baru** (disarankan Go 1.22+ untuk dukungan routing native `net/http` dan fitur konkurensi terbaru).
2. **Git**.
3. **C Compiler** (GCC / MinGW pada Windows, GCC/Clang pada Linux/macOS) untuk kompilasi driver CGO `go-sqlite3`.
4. *(Opsional)* **Docker & Docker Compose** untuk pengujian dengan PostgreSQL.

### Langkah Instalasi Lokal
1. **Fork** repositori ini ke akun GitHub Anda: [https://github.com/antonset1104/erp](https://github.com/antonset1104/erp)
2. **Clone** hasil fork Anda ke komputer lokal:
   ```bash
   git clone https://github.com/<username-anda>/erp.git
   cd erp
   ```
3. **Tambahkan upstream remote**:
   ```bash
   git remote add upstream https://github.com/antonset1104/erp.git
   ```
4. **Unduh dependencies Go**:
   ```bash
   go mod download
   go mod verify
   ```
5. **Salin konfigurasi environment**:
   ```bash
   cp .env.example .env
   ```
6. **Jalankan server aplikasi**:
   ```bash
   go run ./cmd/server
   ```
   Aplikasi akan aktif di `http://localhost:8080`.

---

## 🧪 Menjalankan Pengujian (Running Tests)

Sebelum mengajukan perubahan, pastikan seluruh pengujian integrasi lulus 100%:

```bash
# Menjalankan seluruh test suite
go test -v ./tests/...

# Menjalankan dengan deteksi race conditions
go test -v -race ./tests/...

# Menjalankan verifikasi kompilasi dan linting
go vet ./...
go fmt ./...
```

---

## 🏗️ Struktur Arsitektur & Panduan Menambah Modul

GoERP dibangun dengan arsitektur **Modular Monolith** berkinerja tinggi:

```
erp/
├── cmd/server/               # Entrypoint HTTP Server & Registrasi Rute Utama
├── internal/
│   ├── config/               # Konfigurasi Environment & Flags
│   ├── core/
│   │   ├── database/         # Abstraksi Database (SQLite/PostgreSQL) & Migrasi GORM
│   │   ├── commandcenter/    # Command Palette (Ctrl+K), Omni-Search, & Simulator
│   │   └── models/           # ModelBase (ID, CreateDate, WriteDate, Active Promoted)
│   └── modules/              # 50 Modul ERP Terisolasi (base, stock, sale, account, dll.)
│       └── <nama_modul>/
│           ├── model.go      # Struktur Data & Hook GORM
│           ├── repository.go # Query Data & Akses DB
│           ├── service.go    # Logika Bisnis & Invarian Transaksional
│           └── handler.go    # REST API Handlers
├── web/                      # Frontend SPA Modern (HTML5, Vanilla ES6, CSS Variables)
└── tests/                    # End-to-End & Integration Test Suites
```

### Standar Penulisan Modul Baru:
1. **Warisi `models.ModelBase`**: Setiap entitas master dan dokumen harus menyematkan `models.ModelBase` (`ID`, `CreatedAt`, `UpdatedAt`, `Active`).
2. **Integritas Transaksi Finansial**:
   - Jika modul berinteraksi dengan stok barang, gunakan transaksi atomik inventaris pembukuan ganda (*double-entry stock*).
   - Jika modul berinteraksi dengan jurnal akuntansi, pastikan validasi mutlak: $\sum \text{Debit} == \sum \text{Credit}$.
3. **Penomoran Dokumen Bebas Tabrakan (*Sequence Collision Resistance*)**:
   - Gunakan perulangan pengecekan keunikan nomor dokumen (misal `PO%05d`, `SO%05d`, `INV%05d`) agar aman terhadap *race conditions*.
4. **Kepatuhan Aturan Global Website**:
   - Setiap halaman/antarmuka HTML **wajib** menyertakan script AdSense (`ca-pub-6180580801533680`) dan Google Analytics (`G-79X7K7H4MJ`).

---

## 🌿 Konvensi Branching & Alur Git

1. Selalu buat branch baru dari branch `main` terbaru:
   ```bash
   git checkout main
   git pull upstream main
   git checkout -b feature/nama-fitur-baru
   # atau untuk perbaikan bug:
   git checkout -b fix/deskripsi-perbaikan
   ```
2. Gunakan konvensi **Conventional Commits**:
   - `feat:` Penambahan fitur atau modul baru
   - `fix:` Perbaikan bug atau penanganan error
   - `docs:` Perubahan atau penambahan dokumentasi
   - `perf:` Optimasi performa dan konkurensi
   - `refactor:` Restrukturisasi kode tanpa mengubah fungsionalitas
   - `test:` Penambahan atau perbaikan unit/integration test
   - `chore:` Pembaruan dependensi, build scripts, atau konfigurasi CI

Contoh pesan commit:
```bash
git commit -m "feat(stock): add multi-bin picking wave validation algorithm"
git commit -m "fix(account): prevent negative debit entries on manual reconciliation"
```

---

## 📤 Mengajukan Pull Request (PR)

1. Pastikan kode Anda telah diformat menggunakan `go fmt ./...`.
2. Pastikan `go test -v ./tests/...` lulus tanpa kegagalan.
3. Push branch Anda ke repositori fork:
   ```bash
   git push origin feature/nama-fitur-baru
   ```
4. Buka repositori utama [https://github.com/antonset1104/erp](https://github.com/antonset1104/erp) dan klik tombol **Compare & pull request**.
5. Isi template Pull Request secara lengkap:
   - Jelaskan latar belakang dan masalah yang diselesaikan.
   - Cantumkan modul yang terdampak.
   - Cantumkan bukti pengujian (output test atau tangkapan layar jika menyangkut UI).
6. Tim maintainer akan meninjau kode Anda dan memberikan umpan balik melalui review GitHub.

---

## 🐛 Melaporkan Bug & Mengusulkan Fitur

- **Bug Reports**: Sebelum membuat issue, lakukan pencarian pada tab *Issues* apakah masalah serupa sudah pernah dilaporkan. Jika belum, gunakan form issue **Bug Report** dan sertakan langkah reproduksi, versi Go, serta pesan error/log lengkap.
- **Feature Requests**: Diskusikan ide modul atau peningkatan alur bisnis Anda melalui form issue **Feature Request** atau tab **Discussions**.

Terima kasih atas kontribusi Anda dalam membangun masa depan open-source ERP berbasis Golang!
