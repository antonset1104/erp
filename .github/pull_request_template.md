## 📌 Deskripsi Perubahan (Pull Request Overview)

Jelaskan secara ringkas perubahan yang dilakukan, latar belakang masalah yang diselesaikan, dan modul yang terdampak.

- **Nomor Issue Terkait (jika ada):** Closes #...
- **Tipe Perubahan:**
  - [ ] 🐛 Perbaikan Bug (*Bug fix*)
  - [ ] ✨ Fitur atau Modul Baru (*New feature*)
  - [ ] ⚡ Optimasi Performa (*Performance improvement*)
  - [ ] 📝 Pembaruan Dokumentasi (*Documentation update*)
  - [ ] 🎨 Restrukturisasi Kode (*Code refactoring*)
  - [ ] 🧪 Penambahan Test Case (*Testing*)

---

## 🏢 Modul yang Terdampak (Affected Modules)

- [ ] Core Engine / Database / Routing
- [ ] Sales & Quotations / CRM
- [ ] Purchases & Supplier RFQ
- [ ] Inventory / Stock / WMS
- [ ] Invoicing / Accounting / General Ledger
- [ ] MRP II / Manufacturing / BOM
- [ ] HRIS / Payroll / Kehadiran
- [ ] Perpajakan DJP (e-Faktur / e-Bupot)
- [ ] Kawasan Berikat & DJBC
- [ ] Low-Code Studio / Workflow Engine / Integration Hub
- [ ] Web UI (Frontend HTML/CSS/JS)

---

## 🧪 Bukti Pengujian & Verifikasi (Verification)

Jelaskan pengujian yang telah Anda jalankan sebelum mengajukan PR:

```bash
# Hasil verifikasi test suite lokal
go test -v ./tests/...
go test -v -race ./tests/...
```

- [ ] Seluruh unit & integration test **lulus 100%** (`PASS`).
- [ ] Tidak ada regresi pada modul yang beririsan.
- [ ] *(Jika menyangkut UI)* Telah diuji di browser tanpa adanya JS runtime exception.

---

## ✅ Daftar Periksa Kontributor (Contributor Checklist)

- [ ] Kode saya telah diformat menggunakan `go fmt ./...`.
- [ ] Kode saya mematuhi panduan [CONTRIBUTING.md](CONTRIBUTING.md).
- [ ] Saya telah menambahkan unit test atau integration test yang memverifikasi perubahan ini.
- [ ] Komit saya menggunakan pesan komit konvensional (*Conventional Commits*).
- [ ] Setiap halaman web HTML tetap mempertahankan skrip AdSense dan Analytics yang disyaratkan.
- [ ] Tidak ada kredensial, token privat, atau secret key yang ter-commit ke dalam repositori.
