# GoERP: Enterprise Resource Planning in Golang

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)
[![CI Status](https://img.shields.io/badge/CI-Passing-success?style=flat&logo=githubactions)](https://github.com/antonset1104/erp/actions)
[![Modules](https://img.shields.io/badge/Modules-50_Enterprise-indigo)](docs/FUNCTIONAL_DOCUMENTATION.md)
[![Release](https://img.shields.io/badge/Release-v1.41.0-emerald)](https://github.com/antonset1104/erp/releases)
[![Open Source](https://img.shields.io/badge/Open_Source-Community_Driven-purple)](docs/OPEN_SOURCE_GUIDE.md)

Proyek ini adalah platform Enterprise Resource Planning (ERP) modern kelas enterprise yang dibangun menggunakan bahasa pemrograman **Golang**. Dibangun dengan prinsip *modular monolith* (50 modul enterprise terintegrasi), performa tinggi, konkurensi native Go, memori rendah, serta menjamin integritas data transaksi bisnis.

---

## 📚 Dokumentasi Resmi Lengkap

Dokumentasi komprehensif tersedia di direktori `docs/`:
- 🌐 **[Panduan Proyek Open Source (Open Source Guide)](docs/OPEN_SOURCE_GUIDE.md)** — Cetak biru tata kelola open source, konfigurasi repositori GitHub, panduan rilis SemVer, dan kebijakan lisensi.
- 📘 **[Dokumentasi Teknis (Technical Documentation)](docs/TECHNICAL_DOCUMENTATION.md)** — Arsitektur sistem, struktur direktori 50 modul, skema database ERD, spesifikasi REST API & Open RPC, autentikasi JWT + API Key M2M, dan panduan deployment.
- 📗 **[Dokumentasi Fungsional (Functional Documentation)](docs/FUNCTIONAL_DOCUMENTATION.md)** — Spesifikasi alur bisnis *end-to-end* (O2C, P2P, MRP II, Akuntansi Buku Besar, HRIS & Payroll, Pajak e-Faktur/e-Bupot DJP, Kawasan Berikat DJBC, Low-Code Studio, BPMN 2.0 Workflow Engine, dan Integration Hub).
- 📙 **[Panduan Pengguna (User Documentation)](docs/USER_DOCUMENTATION.md)** — Panduan operasional langkah demi langkah (*step-by-step*) untuk Administrator, Sales, Purchasing, Gudang/WMS, Manufaktur, Finance/Accounting, dan HRD.
- 🤝 **[Panduan Kontribusi (Contributing Guidelines)](CONTRIBUTING.md)** — Alur pengajuan Pull Request, standar konvensi commit, dan pedoman coding Go.
- 📜 **[Pedoman Perilaku Komunitas (Code of Conduct)](CODE_OF_CONDUCT.md)** — Komitmen lingkungan komunitas inklusif dan profesional (Contributor Covenant v2.1).
- 🛡️ **[Kebijakan Keamanan (Security Policy)](SECURITY.md)** — Prosedur pengungkapan kerentanan privat (*responsible disclosure*) dan SLA rilis patch.

---

## ⚡ Executive Command Center & Fitur Produktivitas Baru (Continuous Improvement)

- **🔍 Universal Command Palette & Omni-Search (`Ctrl+K`)**: Pencarian terpadu lintas **50 modul ERP**, tindakan cepat (*Quick Actions*), dan rekaman database langsung (*Products*, *Partners*, *Sales Orders*, *Purchase Orders*, *Invoices*) via `GET /api/v1/search?q=...`.
- **🧭 Filter Instan Sidebar 50 Modul (`/` Shortcut)**: Saring daftar 50 modul di *sidebar* secara *real-time* beserta penghitung modul aktif.
- **🚀 1-Click End-to-End Business Cycle Simulator (`POST /api/v1/dashboard/simulate-cycle`)**: Eksekusi otomatis siklus *Procure-to-Pay* (`PO` $\rightarrow$ `WH/IN`), *Order-to-Cash* (`SO` $\rightarrow$ `WH/OUT`), hingga *General Ledger Posting* (`INV` berimbang dengan PPN 11%) dalam 1 klik beserta laporan jejak audit 5 langkah.
- **🔔 Actionable Notification & Approval Center (`GET /api/v1/dashboard/notifications`)**: Lonceng notifikasi pintar yang mengagregasi antrean *Approval Matrix*, *BPMN Workflow* aktif, dokumen *draft*, dan peringatan keamanan ISO 27001.
- **📊 Executive Financial Pulse Strip & 50-Module Ecosystem Health Matrix**: Pantau Kas & Likuiditas Treasury, Piutang Usaha (*AR*), Hutang Usaha (*AP*), *CRM Pipeline*, *Subscription MRR*, *Live Transaction Feed*, serta ekspor ringkasan KPI ke CSV (`GET /api/v1/dashboard/export-summary`).
- **🌓 Dark / Light Enterprise Theme Switcher**: Dukungan tema gelap dan terang (*Light Enterprise*) yang disimpan secara persisten di browser.

---

## 🚀 Fitur Utama & Modul ERP

### 1. **Modul Base (`internal/modules/base`)**
- **Multi-Company & Multi-Currency**: Mendukung hierarki perusahaan dan konversi mata uang (IDR, USD).
- **Contacts / Partners (`res.partner`)**: Manajemen pelanggan (*Customer*) dan pemasok (*Vendor*) lengkap dengan histori transaksi.
- **Otentikasi & Keamanan (`res.users`)**: Login berbasis JWT dan enkripsi password menggunakan Bcrypt.
- **Role-Based Access Control (RBAC)**: Pembatasan akses berbasis peran (`admin`, `manager`, `user`).

### 2. **Modul Produk (`internal/modules/product`)**
- **Katalog Produk & Kategori**: Manajemen produk barang berstok (*storable product*), barang konsumsi (*consumable*), dan jasa (*service*).
- **Satuan Ukuran / UoM**: Unit of Measure dengan konversi rasio faktor (Unit, Pcs, kg, Dozen, Jam).
- **Manajemen Harga**: Harga jual (*Sales Price*) dan harga modal (*Cost/Standard Price*).

### 3. **Modul Stok & Inventaris (`internal/modules/stock`) - Double-Entry & Multi-Warehouse**
- Mengadopsi prinsip inventaris pembukuan ganda (*double-entry*) seperti Enterprise ERP: barang tidak pernah bertambah/berkurang sembarangan, melainkan berpindah antar lokasi (*Source* $\rightarrow$ *Destination*).
- **Master Multi-Warehouse (`stock.warehouse`)**: Mendukung banyak cabang/gedung gudang (misal `WH-JKT` Jakarta, `WH-SBY` Surabaya, `WH-BDG` Bandung) dengan lokasi simpan utama masing-masing (`lot_stock_id`).
- **Transfer Antar-Gudang (*Inter-Warehouse Transfers* / `WH/INT`)**: Perpindahan stok antar-gudang secara atomik yang mengonservasi total stok produk di seluruh perusahaan.
- **Lokasi Fisik & Virtual**: `WH/Stock`, `WH-SBY/Stock`, `WH-BDG/Stock` (Internal), `Virtual Locations/Inter-Warehouse Transit`, `Virtual/Vendors`, `Virtual/Customers`, `Virtual/Inventory Adjustment`.
- **Stock Pickings & Moves**: Pengiriman (*Delivery Order* / `WH/OUT`), Penerimaan (*Receipt* / `WH/IN`), dan Transfer Internal Antar-Gudang (`WH/INT`).
- **Real-Time Stock Quants**: Pelacakan saldo stok per produk, gudang, dan lokasi secara atomik dalam transaksi database.

### 4. **Modul Penjualan (`internal/modules/sale`)**
- **Alur Dokumen**: Quotation (`draft`) $\rightarrow$ Sales Order (`sale`) $\rightarrow$ Selesai (`done`).
- **Otomasi Terintegrasi**: Konfirmasi Sales Order (`ActionConfirm`) otomatis memicu pembuatan **Delivery Order (`WH/OUT`)** untuk bagian gudang dan faktur tagihan pelanggan.

### 5. **Modul Pembelian (`internal/modules/purchase`)**
- **Alur Dokumen**: RFQ (`draft`) $\rightarrow$ Purchase Order (`purchase`) $\rightarrow$ Selesai (`done`).
- **Otomasi Terintegrasi**: Konfirmasi Purchase Order (`ActionConfirm`) otomatis memicu pembuatan **Penerimaan Barang Gudang (`WH/IN`)** dan tagihan vendor (*Vendor Bill*).

### 6. **Modul Akuntansi & Faktur (`internal/modules/account`)**
- **Entri Jurnal Berimbang**: Validasi mutlak $\sum \text{Debit} == \sum \text{Credit}$ sebelum entri dibukukan (*posted*) ke Buku Besar (*General Ledger*).
- **Bagan Akun (Chart of Accounts)**: Kas, Bank, Piutang Usaha, Persediaan, Hutang Usaha, PPN, Pendapatan Penjualan, HPP.
- **Jurnal Keuangan**: Customer Invoices (INV), Vendor Bills (BILL), Bank (BNK), Cash (CSH), General (MISC).
- **Pencatatan Pembayaran**: Rekonsiliasi pembayaran bertahap maupun lunas (*Paid*).

### 7. **Modul Manufaktur / MRP (`internal/modules/mrp`)**
- **Bill of Materials (BOM)**: Struktur bahan baku komponen (*BoM Lines*) untuk perakitan barang jadi.
- **Manufacturing Order (MO)**: Perintah kerja produksi dengan status siklus hidup (`draft` $\rightarrow$ `confirmed` $\rightarrow$ `progress` $\rightarrow$ `done`).
- **Otomasi Produksi (`ActionProduce`)**: Konsumsi komponen bahan baku secara atomik dari `WH/Stock` ke `Virtual/Production` dan penambahan stok produk barang jadi ke `WH/Stock`.

### 8. **Modul Point of Sale / POS (`internal/modules/pos`)**
- **Terminal & Sesi Kasir (`pos.config`, `pos.session`)**: Pembukaan sesi kasir dengan modal awal (*opening balance*) dan rekonsiliasi kas saat penutupan (*closing balance*).
- **Transaksi Ritel Cepat (`pos.order`)**: Transaksi kasir langsung lunas (Cash, Card, QRIS) dengan otomatisasi perhitungan uang kembalian.
- **Pengurangan Stok Otomatis**: Setiap transaksi kasir langsung memotong stok barang secara atomik di gudang toko ritel.

### 9. **Modul Human Resources & Payroll / HR (`internal/modules/hr`)**
- **Struktur Organisasi**: Manajemen Departemen (`hr.department`) dan Jabatan Kerja (`hr.job`).
- **Data Karyawan (`hr.employee`)**: Data personal, nomor kontak kerja, email, nomor rekening bank, dan gaji pokok.
- **Penggajian / Payslip (`hr.payslip`)**: Siklus penggajian bulanan (`draft` $\rightarrow$ `verify` $\rightarrow$ `done` $\rightarrow$ `paid`) dengan komponen gaji pokok, tunjangan, dan potongan.

### 10. **Modul CRM & Sales Pipeline (`internal/modules/crm`)**
- **Siklus Prospek & Peluang**: Pipeline tahapan (`new` $\rightarrow$ `qualified` $\rightarrow$ `proposition` $\rightarrow$ `won` / `lost`).
- **Kalkulasi Metrik Pipeline**: Estimasi nilai pendapatan (*Expected Revenue*), probabilitas keberhasilan (%), dan tingkat konversi (*Win Rate*).
- **Otomasi Konversi Peluang (`ConvertToQuotation`)**: Peluang bisnis yang berhasil dimenangkan dapat dikonversi langsung menjadi Sales Quotation / Sales Order (`SO`) secara satu klik.

### 11. **Modul Chatter & Jejak Audit (`internal/modules/chatter`) - `mail.message`**
- **Log Aktivitas & Riwayat Transaksi**: Mencatat jejak audit perubahan status transaksi bisnis secara otomatis.
- **Kolaborasi & Catatan Internal**: Tim dapat menuliskan komentar, notulen meeting klien, dan diskusi langsung pada setiap dokumen transaksi (Leads, Sales Orders, Invoices, Pickings).

### 12. **Modul Project Management & Task Timesheet (`internal/modules/project`)**
- **Proyek & Pelanggan (`project.project`)**: Manajemen proyek bisnis, warna aksen, integrasi kontak pelanggan (*Partner*), dan status aktif.
- **Tugas & Tahapan Kanban (`project.task`)**: Pelacakan tugas per proyek dengan tahapan (`todo`, `in_progress`, `review`, `done`), prioritas kerja, estimasi jam (*Planned Hours*), jam terealisasi (*Effective Hours*), dan persentase *progress* atomik.
- **Pencatatan Jam Kerja / Timesheets (`project.task.timesheet`)**: Analogi `account.analytic.line` pada Enterprise ERP untuk mencatat aktivitas harian karyawan dan jam kerja teknis.

### 13. **Modul Manajemen Aset Tetap & Depresiasi Otomatis (`internal/modules/asset`) - `account.asset`**
- **Registrasi Aset Tetap (`account.asset`)**: Nilai perolehan (*Original Value*), nilai sisa/residu (*Salvage Value*), durasi depresiasi (bulan/tahun), dan kategori aset (Hardware, Machinery, Vehicle, Furniture, Building).
- **Papan Jadwal Penyusutan Garis Lurus (*Linear Depreciation Board*)**: Otomatis menghitung angsuran penyusutan berkala (*Asset Depreciation Lines*) secara presisi dengan penyesuaian pembulatan pada periode akhir.
- **Integrasi Jurnal Akuntansi Buku Besar (*General Ledger Posting*)**: Setiap periode penyusutan dapat dibukukan langsung ke GL dengan entri berimbang ($\text{Debet Beban Penyusutan} == \text{Kredit Akumulasi Penyusutan}$), otomatis mengurangi nilai buku bersih (*Net Book Value*).
- **Otomasi Batch Depresiasi Jatuh Tempo (`AutoPostDueDepreciations`)**: Eksekusi batch satu klik untuk memposting seluruh jadwal penyusutan yang telah jatuh tempo ke Buku Besar.

### 14. **Modul Helpdesk & Customer Support Ticketing (`internal/modules/helpdesk`) - `helpdesk.ticket`**
- **Tim Dukungan Multi-Tier (`helpdesk.team`)**: Struktur tim penanganan tiket pelanggan (Customer Care, Technical Support Tier 2, Billing & Invoicing Help) dengan kode, warna aksen, dan team leader.
- **Tiket Layanan Pelanggan (`helpdesk.ticket`)**: Auto-sequence penomoran tiket (`TCK/2026/00001`), prioritas bertingkat (Urgent, High, Medium, Low), kategori kendala (Technical, Billing, Inquiry, Warranty), dan eskalasi tahapan kerja (`new` $\rightarrow$ `in_progress` $\rightarrow$ `waiting` $\rightarrow$ `solved` / `cancelled`).
- **Penugasan Agen & Resolusi Terdata**: Penugasan tiket ke teknisi spesialis secara otomatis memperbarui status menjadi *In Progress*. Penyelesaian tiket mencatat tanggal penutupan (*Close Date*) dan ringkasan tindakan solusi (*Resolution Summary*).
- **Integrasi Universal Chatter**: Seluruh interaksi, diskusi teknisi, dan tindak lanjut perbaikan terhubung langsung ke modul `chatter` (`mail.message`) untuk audit trail lengkap.

### 15. **Modul Perawatan Mesin & Manajemen Peralatan (`internal/modules/maintenance`) - `maintenance.equipment`**
- **Katalog Mesin & Peralatan Kerja (`maintenance.equipment`)**: Penomoran peralatan (`EQP/2026/00001`), serial number, model, lokasi kerja pabrik, biaya perolehan, teknisi penanggung jawab, dan integrasi langsung ke Aset Tetap Buku Besar (`account.asset`).
- **Jadwal Perawatan Preventif Otomatis (*Preventive Maintenance Scheduler*)**: Siklus interval perawatan berkala (misal tiap 30/45/60 hari), kalkulasi otomatis tanggal servis berikutnya (`next_action_date`), dan *engine* penjadwalan otomatis yang mendeteksi mesin yang jatuh tempo untuk digenerate menjadi *Work Order*.
- **Permohonan Perbaikan Kerusakan Darurat (*Corrective Breakdown Requests*)**: Pelaporan kendala mesin (`REQ/MAINT/2026/00001`) dengan tingkat urgensi (*Urgent Breakdown*), otomatis mengubah status mesin menjadi `in_maintenance` agar tidak digunakan pada lini perakitan manufaktur (*MRP*).
- **Penyelesaian Servis & Pemulihan Operasional**: Pencatatan durasi jam perbaikan teknisi, ringkasan tindakan solusi (*Resolution Summary*), pemulihan status mesin menjadi `operational`, dan pembaruan tanggal siklus preventif berikutnya.
- **Audit Jejak Aktivitas (*Chatter*)**: Terintegrasi penuh ke modul `chatter` (`mail.message`) untuk merekam riwayat servis dan penggantian sparepart.

### 16. **Modul Manajemen Armada Kendaraan & Pengiriman (`internal/modules/fleet`) - `fleet.vehicle`**
- **Model Kendaraan & Tipe Armada (`fleet.vehicle.model`)**: Pengelompokan merk dan spesifikasi armada (Truck Box, Blind Van, D-Cab, Motor Kurir) dengan klasifikasi tipe dan bahan bakar.
- **Registrasi Armada (`fleet.vehicle`)**: Plat nomor polisi unik, nomor rangka (*VIN/Chassis*), nomor mesin, kapasitas muatan, pangkalan lokasi, supir penanggung jawab (`hr.employee`), dan keterkaitan pembukuan aset tetap (`account.asset`).
- **Pencatatan Odometer Terkini (`fleet.vehicle.odometer`)**: Pencatatan log jarak tempuh armada (KM) yang otomatis memperbarui angka odometer terakhir kendaraan secara atomik.
- **Log Servis, Perbaikan & Uji KIR (`fleet.vehicle.log.services`)**: Penomoran resmi (`SRV/FLEET/2026/00001`), klasifikasi jenis servis (Ganti Oli, Ganti Ban, Servis Berkala, Uji KIR Dishub, Perbaikan Darurat), biaya bengkel rekanan (*Vendor Partner*), sinkronisasi odometer, dan catatan mekanik.
- **Integrasi Universal Chatter**: Rekam jejak perpanjangan pajak STNK, masa berlaku uji KIR Dishub, dan riwayat perbaikan tersimpan rapi pada modul `chatter` (`mail.message`).

### 17. **Modul Manajemen Cuti & Izin Karyawan (`internal/modules/leave`) - `hr.leave` & `hr.leave.allocation`**
- **Kategori & Tipe Cuti (`hr.leave.type`)**: Pengaturan jenis cuti (Cuti Tahunan berbayar, Cuti Sakit, Cuti Tanpa Gaji / *Unpaid Leave*, Cuti Melahirkan, dan Izin Khusus).
- **Alokasi Jatah Kuota Tahunan (`hr.leave.allocation`)**: Penetapan jatah hari kerja per karyawan per tahun kalender (misal 12 hari/tahun). Sistem memantau hari terpakai (*Used Days*) dan sisa saldo cuti (*Remaining Days*) secara presisi.
- **Pengajuan & Persetujuan Berjenjang (`hr.leave`)**: Alur permohonan cuti (`confirm` $\rightarrow$ `validate` / `refuse`). Persetujuan oleh HR/Manager secara otomatis memotong saldo kuota cuti tahunan karyawan.
- **Integrasi Otomatis ke Payroll (`hr.payslip`)**: Mendeteksi cuti tanpa gaji (*unpaid leave*) yang telah disetujui pada bulan berjalan untuk menghitung potongan prorata gaji pokok secara otomatis.
- **Integrasi Universal Chatter**: Rekam jejak audit dan catatan diskusi persetujuan cuti tercatat langsung pada modul `chatter` (`mail.message`).

### 18. **Modul Manajemen Biaya & Reimbursement Karyawan (`internal/modules/expense`) - `hr.expense` & `hr.expense.sheet`**
- **Master Kategori Biaya & Bagan Akun GL (`hr.expense.category`)**: Pengelompokan pengeluaran operasional (Perjalanan Dinas, Konsumsi & Jamuan Klien, BBM & Tol Armada, ATK Kantor, Komunikasi, Pelatihan) yang terhubung ke Bagan Akun Beban (*Expense Accounts* 620100 - 620600).
- **Pencatatan Item Pengeluaran (`hr.expense`)**: Input transaksi pengeluaran karyawan dengan referensi struk/kwitansi, metode pembayaran (talangan pribadi karyawan untuk direimburse vs kas perusahaan), serta tagging beban ke Proyek Bisnis (`project.project`) atau Armada Logistik (`fleet.vehicle`).
- **Rekapitulasi Laporan Pengeluaran (*Expense Reports / Sheets* / `hr.expense.sheet`)**: Pengelompokan banyak item kwitansi ke dalam satu berkas laporan pengajuan (`draft` $\rightarrow$ `submit` $\rightarrow$ `approve` $\rightarrow$ `post` $\rightarrow$ `done` / `refuse`).
- **Integrasi Buku Besar Akuntansi Otomatis (*Automatic GL Journal Entry*)**: Saat laporan disetujui dan dibukukan (`ActionPostSheet`), sistem otomatis membuat dan memposting **Entri Jurnal Berimbang (`account.move`)**:
  - $\text{Debet}$: Akun beban operasional masing-masing kategori pengeluaran (Travel, Konsumsi, BBM, dll).
  - $\text{Kredit}$: Hutang Usaha / Hutang Karyawan (`liability_payable` 210000) jika *own account*, atau Kas Operasional (`asset_cash` 110100) jika *company account*.
- **Pencatatan Pembayaran Reimbursement (`ActionPaySheet`)**: Memperbarui status laporan dan seluruh item pengeluaran menjadi selesai / lunas (*Done / Paid*).
- **Integrasi Universal Chatter**: Seluruh siklus pengajuan, persetujuan manajerial, penolakan dengan alasan tertulis, dan nomor entri jurnal tercatat pada modul `chatter` (`mail.message`).

### 19. **Modul Kontrol Kualitas & Inspeksi Mutu (`internal/modules/quality`) - `quality.point`, `quality.check`, `quality.alert`**
- **Titik Kontrol Mutu Standar (`quality.point`)**: Standar Operasional Prosedur (SOP) pengujian mutu otomatis saat penerimaan barang gudang (`incoming` / WH/IN), perakitan manufaktur (`mrp_operation` / MO), maupun pengiriman barang (`outgoing` / WH/OUT).
- **Metode Pengujian Multi-Tipe**:
  - *Pass / Fail*: Verifikasi keutuhan kemasan, segel, dan visual.
  - *Quantitative Measure*: Pengukuran teknis presisi (Voltase, Dimensi, Suhu, Berat) dengan toleransi batas aman $(\text{Target} \pm [\text{TolMin}, \text{TolMax}])$ yang divalidasi secara ketat oleh sistem.
  - *Instructions*: Langkah instruksi SOP khusus untuk operator pemeriksa.
- **Pemeriksaan Mutu Harian (`quality.check`)**: Eksekusi inspeksi mutu terintegrasi langsung dengan nomor penerimaan stok (`stock.picking`) dan surat perintah kerja manufaktur (`mrp.production`).
- **Otomasi Pembuatan Check**: Sistem otomatis menerbitkan seluruh daftar lembar inspeksi yang relevan saat barang tiba di gudang atau saat MO dibuat.
- **Laporan Ketidaksesuaian Mutu & CAPA (`quality.alert` / Non-Conformance Report - NCR)**:
  - Otomatis dibuat saat inspeksi dinyatakan gagal (*Fail*), mengkarantina barang dan menetapkan prioritas penanganan.
  - Siklus tahapan investigasi: `new` $\rightarrow$ `confirmed` $\rightarrow$ `in_progress` $\rightarrow$ `resolved` $\rightarrow$ `closed`.
  - Keputusan tindakan barang: Karantina (*Quarantine*), Pengerjaan Ulang (*Rework*), Retur ke Vendor (*Return*), atau Pemusnahan (*Scrap*).
  - Analisis Akar Masalah (*Root Cause Analysis* - RCA) dan Rencana Tindakan Pencegahan (*CAPA*).
- **Integrasi Universal Chatter**: Seluruh penerbitan inspeksi, penetapan pass/fail, dan update tahapan investigasi mutu tercatat otomatis pada audit trail `chatter` (`mail.message`).

### 20. **Modul Matriks Persetujuan & Otorisasi Korporat (`internal/modules/approval`) - `approval.category`, `approval.request`, `approval.approver`**
- **Kategori Template Persetujuan (`approval.category`)**: Konfigurasi aturan otorisasi korporat (Pengadaan Capex & Aset Tetap, Diskon Penjualan Khusus > 10%, Perjalanan Dinas VIP, Kontrak Legal).
- **Aturan Ambang Batas & Multi-Tier**: Mendukung jumlah persetujuan berjenjang minimum (`min_approvals_required`), kewajiban persetujuan manajer langsung (`is_manager_approver`), dan batas nominal anggaran (`min_amount`) yang memicu eskalasi ke Dewan Direksi/CFO.
- **Permohonan Persetujuan (`approval.request`)**: Penomoran resmi berurutan (`APPR/2026/00001` atau `CAPEX/2026/00001`), referensi dokumen transaksi terkait (`PO00002`, `SO00001`, `EXP/2026/00001`), serta tahapan alur kerja (`new` $\rightarrow$ `pending` $\rightarrow$ `approved` / `refused` / `cancelled`).
- **Matriks Approver Berjenjang (`approval.approver`)**: Otorisasi step-by-step per level (Level 1: Dept Head $\rightarrow$ Level 2: VP Finance/CFO $\rightarrow$ Level 3: Managing Director). Sistem otomatis mengaktifkan level berikutnya begitu level sebelumnya telah disetujui (*Approved*).
- **Pencatatan Alasan Penolakan (*Refusal Reason*)**: Pelacakan transparan terhadap alasan mengapa permohonan ditolak beserta catatan resmi dari reviewer.
- **Integrasi Universal Chatter**: Seluruh siklus pengajuan, persetujuan parsial per level, dan penolakan tercatat pada modul `chatter` (`mail.message`).

### 21. **Modul Dokumen & Cloudflare R2 Storage Hub (`internal/modules/attachment` & `internal/core/storage`)**
- **Universal Document Attachment (`ir.attachment`)**: Menghubungkan berkas digital ke record transaksi apapun di ERP (`hr.expense`, `account.move`, `quality.alert`, `approval.request`, `purchase.order`, `project.task`, dll) sesuai arsitektur `ir.attachment` Enterprise ERP.
- **Cloudflare R2 Zero-Egress Storage Adapter**: Mendukung penyimpanan cloud object storage Cloudflare R2 (S3-compatible) dengan penandatanganan AWS Signature Version 4 (SigV4). Menjamin nol biaya transfer keluar (*zero egress fees*) yang ideal untuk deployment Cloudflare.
- **Seamless Local Disk Fallback**: Jika kredensial Cloudflare R2 tidak disetel, sistem secara transparan dan tanpa kendala menyimpan file ke penyimpanan lokal (`./uploads/attachments`), sehingga pengujian dan deployment on-premise tetap berjalan 100%.
- **Validasi Integritas SHA256**: Setiap berkas dihitung nilai hash SHA256 untuk mendeteksi integritas data dan mencegah duplikasi file korporat.
- **Cloudflare Workers AI Vision OCR Engine**:
  - Memindai foto struk SPBU/BBM, kwitansi hotel, tiket perjalanan, dan faktur belanja menggunakan model vision AI edge `@cf/meta/llama-3.2-11b-vision-instruct` (dengan fallback ke Local Smart Heuristic OCR Engine saat offline).
  - Mengekstrak data terstruktur: `vendor_name`, `invoice_date`, `total_amount`, `tax_amount` (PPN), `currency`, `category_suggestion`, `line_items` (rincian item belanja), dan tingkat keyakinan (*confidence score*).
- **1-Click Expense Creation dari Struk**:
  - Mengonversi hasil pemindaian OCR struk langsung menjadi entri pengajuan klaim biaya (`hr.expense`) secara otomatis tanpa perlu input manual.
- **Audit Trail & Chatter Logging**: Mengirimkan notifikasi aktivitas ke modul `chatter` saat berkas diunggah dan saat hasil pemindaian OCR selesai diproses.

### 22. **Modul Presensi, Shift Kerja & Telemetri Edge Cloudflare (`internal/modules/attendance`) - `hr.attendance` & `hr.attendance.shift`**
- **Master Shift Kerja Fleksibel (`hr_attendance_shifts`)**: Konfigurasi jam masuk/pulang, toleransi keterlambatan (*grace period*), jam kerja harian standar (misal 8 jam), geofencing koordinat GPS kantor & radius maksimum (meter), serta flag shift WFH/remote.
- **Check-In & Check-Out Otomatis (`hr_attendances`)**:
  - Rekam presensi harian per karyawan dengan status `present`, `late`, `wfh`, atau `absent`.
  - Kalkulasi keterlambatan menit (`late_minutes`) otomatis terhadap toleransi shift.
  - Perhitungan jam kerja riil (`worked_hours`) dan jam lembur berlebih (`overtime_hours`) saat check-out.
  - Pencegahan sesi ganda (*duplicate active session prevention*).
- **Cloudflare Edge Telemetry & Anti-Fraud GPS**:
  - Deteksi lokasi IP dan geolokasi negara secara otomatis dari header Cloudflare Edge (`CF-Connecting-IP`, `CF-IPCountry`, `CF-Ray`).
  - Validasi jarak GPS Haversine terhadap koordinat kantor untuk mencegah manipulasi presensi di luar radius kantor non-WFH (`is_anomaly = true`).
- **1-Click Sinkronisasi ke Penggajian (`hr.payslip`)**:
  - Sinkronisasi akumulasi lembur dan keterlambatan bulan berjalan ke slip gaji karyawan.
  - Otomatis menambahkan baris tunjangan lembur (`overtimeBonus`) dan potongan denda keterlambatan (`latePenalty`).
- **Cloudflare Workers AI Anomaly & Burnout Risk Audit (`/api/v1/attendance/ai-anomaly-check`)**:
  - Memindai seluruh log presensi untuk mendeteksi pegawai dengan risiko keletihan berlebih (*burnout risk* > 8 jam lembur), keterlambatan kronis berulang, dan anomali geolokasi/IP luar negeri menggunakan model `@cf/meta/llama-3.1-8b-instruct`.
- **Integrasi Universal Chatter**: Setiap check-in, check-out berlebih/lembur, dan sinkronisasi gaji dicatat rapi pada jejak audit `chatter` (`mail.message`).

### 23. **Modul Rekrutmen & Talent Acquisition (`internal/modules/recruitment`) - `hr.recruitment` & Cloudflare Workers AI CV Screening**
- **Lowongan Kerja & Formasi Kebutuhan (`hr_job_openings`)**: Manajemen pembukaan loker per departemen, deskripsi tugas, syarat kualifikasi teknis, estimasi rentang gaji (*salary min/max*), target kuota penerimaan karyawan, serta status lowongan (`open` / `closed`).
- **Pipeline Tahapan Seleksi (`hr_recruitment_stages`)**: Alur seleksi kandidat bertahap (*Initial Qualification*, *Technical Assessment*, *First Interview*, *Executive Interview*, *Contract Proposal*, *Contract Signed*).
- **Database Kandidat & Pelamar (`hr_applicants`)**: Profil pelamar, nomor kontak, email, tautan portofolio, ekspektasi gaji, ketersediaan mulai kerja, lampiran resume/CV, serta status kandidat (`in_progress`, `hired`, `refused`).
- **Cloudflare Workers AI Resume & CV Screening (`/api/v1/recruitment/applicants/:id/ai-screening`)**:
  - Menganalisis kecocokan CV kandidat terhadap kualifikasi lowongan kerja menggunakan model LLM `@cf/meta/llama-3.1-8b-instruct` (dengan *Smart Local Heuristic Fallback*).
  - Menghasilkan evaluasi komprehensif: skor kecocokan (0-100%), rekomendasi keputusan (*Strong Hire*, *Hire*, *Consider*, *Reject*), ringkasan eksekutif, poin-poin keunggulan & kelemahan, serta 3-4 pertanyaan wawancara teknis mendalam yang disesuaikan secara dinamis.
- **1-Click Hire Otomatis ke Modul Karyawan Aktif (`ActionHire`)**:
  - Pelamar yang lolos seleksi dapat di-*hire* dengan satu klik; sistem otomatis membuat profil karyawan baru di direktori `hr.employee` lengkap dengan penempatan departemen dan jabatan kerja terkait.
  - Memperbarui status pelamar menjadi `hired` dan meningkatkan counter `current_hired` pada lowongan kerja.
- **Pencatatan Alasan Penolakan (*Refusal Tracking*)**: Transparansi alasan mengapa kandidat belum lolos seleksi beserta audit logging.
- **Integrasi Universal Chatter**: Seluruh pemindahan tahapan, hasil asesmen AI, konversi menjadi karyawan, dan alasan penolakan tercatat rapi pada jejak audit `chatter` (`mail.message`).

### 24. **Mesin Ekspor Data Excel & CSV (`internal/core/export`)**
- Menghasilkan file CSV terstandarisasi dengan **UTF-8 BOM (`\xEF\xBB\xBF`)** agar langsung terbaca rapi saat dibuka di Microsoft Excel tanpa masalah encoding simbol mata uang Rupiah maupun teks bahasa Indonesia.
- Tersedia pada Katalog Produk, Riwayat Sales Orders, dan Laporan Neraca Saldo (*Trial Balance*).

### 24. **Modul Stock Opname & Inventaris Fisik (`internal/modules/stock_inventory`) - `stock.inventory` & Cloudflare Workers AI Loss Prevention Auditor**
- **Dokumen Stock Opname (`stock_inventories`)**: Penomoran dokumen berkala (`OPNAME/2026/00001`), lokasi gudang penyimpanan (`stock.location`), tanggal inspeksi, status siklus (`draft` $\rightarrow$ `confirm` $\rightarrow$ `done` / `cancel`), total varians unit, dan agregasi nilai kerugian (*Loss*) serta surplus (*Gain*).
- **Snapshot Saldo Buku Otomatis (`ActionStart`)**: Saat opname dimulai, sistem secara otomatis mengambil foto saldo berjalan (*quant snapshot*) dari `stock_quants` untuk seluruh produk pada lokasi terkait ke dalam baris lembar hitung fisik (`stock_inventory_lines`).
- **Input Perhitungan Fisik & Baris Tambahan**: Tim gudang menginput hasil hitung aktual di lapangan (`counted_qty`) dan alasan selisih. Tersedia pula fitur untuk memasukkan temuan barang tak terduga (*unrecorded item*) yang ditemukan saat inspeksi rak.
- **Cloudflare Workers AI Warehouse Loss Prevention Auditor (`/api/v1/stock-inventory/inventories/:id/ai-audit`)**:
  - Menganalisis seluruh baris selisih fisik menggunakan model LLM `@cf/meta/llama-3.1-8b-instruct` (didukung *Smart Local Heuristic Fallback*).
  - Mengklasifikasikan tingkat risiko kehilangan (*Low*, *Medium*, *High*), menghasilkan ringkasan eksekutif pola selisih, mendeteksi item mencurigakan (*suspicious high-value shrinkage*), dan merumuskan rekomendasi perbaikan SOP gudang (misal rekonsiliasi CCTV rak, pemeriksaan nota penerimaan WH/IN, atau verifikasi ganda barcode scanner).
- **Penyesuaian Stok Atomik & Double-Entry (`ActionValidate`)**:
  - Untuk setiap selisih, sistem otomatis membuat **Stock Move Berpasangan**:
    - Selisih Kurang (*Loss/Shrinkage*): Berpindah dari `Lokasi Gudang` $\rightarrow$ `Virtual Locations/Inventory Adjustment`.
    - Selisih Lebih (*Gain/Surplus*): Berpindah dari `Virtual Locations/Inventory Adjustment` $\rightarrow$ `Lokasi Gudang`.
  - Mengupdate saldo riil `stock_quants` secara atomik sehingga akurat 100% dengan kondisi fisik rak.
- **Integrasi Otomatis Buku Besar Akuntansi (*Automatic GL Journal Entry*)**:
  - Saat opname divalidasi, sistem otomatis membuat dan memposting **Entri Jurnal Berimbang (`account.move`)** pada jurnal `MISC`:
    - $\text{Debet}$: Beban Selisih Kurang Stok (`620700` - Inventory Shrinkage/Loss) untuk nilai kerugian.
    - $\text{Kredit}$: Persediaan Barang Dagang (`130000` - Stock Valuation) untuk pengurangan fisik.
    - $\text{Debet}$: Persediaan Barang Dagang (`130000` - Stock Valuation) untuk penambahan fisik.
    - $\text{Kredit}$: Pendapatan Selisih Lebih Stok (`710200` - Inventory Gain) untuk nilai surplus.
    - Menjamin validasi mutlak $\sum \text{Debit} == \sum \text{Credit}$.
- **Integrasi Universal Chatter**: Seluruh siklus pembuatan, dimulainya penghitungan fisik, hasil audit AI, dan pembukuan jurnal GL tercatat rapi pada audit trail `chatter` (`mail.message`).

### 25. **Modul Enterprise Payroll & Salary Rules Engine (`internal/modules/payroll`) - `hr.payroll` & Cloudflare Workers AI Tax Auditor**
- **Multi-Tier Salary Structures (`payroll_structures` / `hr.payroll.structure`)**: Mendukung konfigurasi struktur penggajian bertingkat (misal `IDN_STD` - Karyawan Tetap, `IDN_PROB` - Karyawan Masa Percobaan, `IDN_EXEC` - Dewan Direksi) dengan aturan gaji fleksibel.
- **Dynamic Salary Rules Engine (`payroll_salary_rules` / `hr.salary.rule`)**:
  - Konfigurasi aturan komponen gaji berbasis urutan evaluasi (*sequence execution*): `BASIC`, `ALW_JOB` (Tunjangan Jabatan), `ALW_MEAL_TRANS` (Uang Makan & Transport), `ALW_OT` (Upah Lembur), `DED_UNPAID` (Potongan Cuti Unpaid), `DED_BPJS_TK` (BPJS TK JHT & JP), `DED_BPJS_KES` (BPJS Kesehatan), `DED_PPH21` (Pajak PPh 21 TER UU HPP), dan `NET` (*Take-Home Pay*).
  - Pemetaan akun debit dan kredit akuntansi otomatis untuk setiap aturan komponen gaji.
- **Batch Payroll Runs (`payroll_payslip_runs` / `hr.payslip.run`)**:
  - Manajemen siklus penggajian masal bulanan (`BATCH/PAY/2026/10`).
  - Fitur **1-Click Mass Generation**: Otomatis membuat dan menghitung (*compute*) slip gaji untuk seluruh karyawan aktif dalam satu langkah dengan sinkronisasi presensi dan cuti.
- **Sinkronisasi Presensi & Lembur Otomatis (`hr.attendance`)**:
  - Menghitung akumulasi jam lembur presensi karyawan aktual pada rentang periode penggajian secara otomatis.
  - Mengkalkulasi nilai rupiah upah lembur mengacu formula standar Kemenaker: $\text{Rate per Jam} = \frac{\text{Gaji Pokok}}{173}$.
- **Sinkronisasi Cuti Tanpa Gaji (*Unpaid Leave Deductions* / `hr.leave`)**:
  - Menghitung hari cuti *unpaid* yang telah disetujui (*validated*) dan memotong gaji secara proporsional: $\text{Daily Rate} = \frac{\text{Gaji Pokok}}{\text{Hari Kerja Sebulan}}$.
- **Cloudflare Workers AI Tax & Compliance Auditor (`/api/v1/payroll/payslips/:id/ai-audit`)**:
  - Mengevaluasi kepatuhan hukum ketenagakerjaan dan perpajakan menggunakan model LLM `@cf/meta/llama-3.1-8b-instruct` (didukung *Smart Local Heuristic Fallback*).
  - Memverifikasi batas maksimum jam lembur UU Cipta Kerja (maksimal 4 jam/hari atau 18 jam/minggu).
  - Mendeteksi lonjakan gaji abnormal (*salary spike anomaly* >60%), verifikasi kelengkapan iuran BPJS Ketenagakerjaan (3%) dan BPJS Kesehatan (1%), serta kepatuhan pemotongan PPh 21 Tarif Efektif Rata-Rata (TER) PP 58/2023.
  - Menghasilkan skor kepatuhan (*Compliance Score* 0–100%), tingkat risiko (*Low, Medium, High*), ringkasan eksekutif, dan rekomendasi audit kepatuhan.
- **Entri Jurnal Berimbang Otomatis ke Buku Besar GL (`ActionConfirm` $\rightarrow$ `account.move`)**:
  - Saat slip gaji disetujui, sistem otomatis memposting jurnal `MISC`:
    - $\text{Debet}$: Beban Gaji & Upah Pokok Karyawan (`610200`)
    - $\text{Debet}$: Beban Tunjangan & Lembur Karyawan (`610300`)
    - $\text{Kredit}$: Hutang Pajak PPh Pasal 21 (`210300`)
    - $\text{Kredit}$: Hutang Iuran BPJS Ketenagakerjaan & Kesehatan (`210400`)
    - $\text{Kredit}$: Hutang Gaji & Upah Karyawan / Net Take-Home Pay (`210200`)
    - Menjamin validasi mutlak $\sum \text{Debit} == \sum \text{Credit}$.
- **Pencatatan Pembayaran Transfer Bank (`ActionPay`)**:
  - Membukukan transfer pencairan gaji karyawan melalui Jurnal Bank `BNK` (Debet Hutang Gaji `210200` / Kredit Bank BCA `110200`).
- **Integrasi Universal Chatter**:
  - Log audit trail lengkap pada `hr.payslip` dan `hr.payslip.run` mencatat pembuatan draf, kalkulasi ulang komponen, audit kepatuhan AI, persetujuan jurnal GL, dan pencairan pembayaran.

### 26. **Modul Enterprise Budgeting & Akuntansi Analitik (`internal/modules/budget`)**
- **Hierarki Akun Analitik / Pusat Biaya (`account.analytic.account`)**:
  - Penomoran dan pengelompokan biaya ke departemen, divisi, dan proyek khusus (misal: `AN/ENG` Departemen IT, `AN/MKT` Divisi Marketing, `AN/OPS` Operasional Pabrik & Logistik).
  - Melacak saldo akumulasi biaya operasional (*Cost Center Balance*).
- **Pos Anggaran Belanja (`account.budget.post`)**:
  - Pemetaan kelompok akun Buku Besar (*General Ledger Accounts*) ke pos pagu belanja (misal `BP/PAYROLL` memetakan Beban Gaji `610200` dan Tunjangan `610300`).
- **Dokumen Rencana Kerja & Anggaran Biaya / RKAB (`account.budget`)**:
  - Siklus hidup dokumen anggaran korporat (`draft` $\rightarrow$ `confirm` $\rightarrow$ `validate` $\rightarrow$ `done` / `cancel`).
  - Penomoran otomatis berbasis kuartal/tahun (`BDG/2026/Q4/00001`).
- **Analisis Varians Dinamis & Target Teoretis Waktu (`account.budget.line`)**:
  - **Plafon Terencana (*Planned Amount*)**: Batas atas belanja pos biaya yang disetujui manajemen.
  - **Target Teoretis Waktu (*Theoretical Amount*)**: Nilai teoretis berdasarkan rasio hari berjalan terhadap total durasi periode:
    $$\text{Theoretical Amount} = \text{Planned Amount} \times \min\left(1, \max\left(0, \frac{T_{\text{now}} - T_{\text{start}}}{T_{\text{end}} - T_{\text{start}}}\right)\right)$$
  - **Sinkronisasi Realisasi GL Riil (*Practical Amount*)**: Menghitung secara atomik agregat $\sum(\text{Debit} - \text{Credit})$ dari seluruh mutasi `account_move_lines` yang berstatus `posted` pada rentang tanggal terkait.
  - **Varians Anggaran (*Variance Amount*)**: Selisih sisa anggaran $(\text{Planned} - \text{Practical})$. Bernilai positif jika *favorable* (anggaran tersisa) dan negatif jika defisit (*overbudget*).
  - **Persentase Konsumsi (*Percentage*)**: Rasio penyerapan $(\text{Practical} / \text{Planned} \times 100\%)$.
- **Cloudflare Workers AI Budget Variance & Overspend Early-Warning Auditor**:
  - Model `@cf/meta/llama-3.1-8b-instruct` mengevaluasi kecepatan belanja (*burn rate*), mendeteksi pos biaya defisit (>100%), lonjakan belanja abnormal (>1.3x target waktu), menetapkan skor risiko (0-100), dan merumuskan langkah mitigasi CFO.
- **Universal Chatter Audit Trail**:
  - Seluruh perubahan status draf, konfirmasi, validasi, kalkulasi GL, dan audit AI tercatat di model `account.budget`.

### 27. **Modul Barcode & QR Code Scanner Engine, Lot/Serial Tracking & Mobile Operations (`internal/modules/barcode`)**
- **Universal Context-Aware Barcode Scanner (`/api/v1/barcode/scan`)**:
  - Pemindai adaptif multi-tipe yang secara otomatis mengenali format barcode: Produk (`product`), Nomor Lot/Serial (`lot`), Rak/Lokasi Gudang (`location`), atau Nomor Picking Pengiriman (`picking`).
  - Pemrosesan kontekstual otomatis:
    - `context="pos"`: Menyiapkan item transaksi kasir (*Cart Item*) lengkap dengan kalkulasi subtotal instan.
    - `context="stock_inventory"`: Menemukan baris opname fisik, menambah kuantitas hitung (*counted_qty*) secara atomik atau mendaftarkan baris temuan baru, serta menghitung ulang saldo varians opname.
- **Pure Go Vector SVG Code 128 Generator (`GenerateCode128SVG`)**:
  - Algoritma encoding Code 128 (Subset B) murni dalam Golang tanpa library eksternal/CGO maupun ketergantungan font pihak ketiga.
  - Menghasilkan vektor `<svg>` tajam beresolusi tinggi dengan garis batang, teks human-readable, judul produk, SKU, dan harga Rupiah siap cetak ke thermal printer label 58mm/80mm.
- **Pelacakan Nomor Lot & Seri (`stock.production.lot`)**:
  - Pencatatan batch produksi perakitan MO atau penerimaan PO, tanggal kadaluwarsa (*Expiration Date*), tanggal peringatan mutu (*Alert Date*), saldo stok lot, dan lokasi rak gudang.
- **Web UI Glassmorphic Barcode Scanner & Laser Viewfinder**:
  - Viewfinder kamera interaktif dengan animasi sinar laser merah (`@keyframes scannerLaserScan`), crosshair targeting corners, emulasi input hardware scanner gun (penangkapan otomatis tombol Enter), sintesis audio feedback beep via Web Audio API, pratinjau cetak label SVG, dan playground uji coba cepat.
### 28. **Modul Purchase Requisition, Call for Bids (Tender) & Blanket Purchase Orders (`internal/modules/purchase_requisition`)**
- **Permintaan Pembelian Internal & Persetujuan Manajer (`purchase.requisition`)**:
  - Alur pengajuan pengadaan barang/jasa dari departemen operasional (status: `draft` -> `confirmed`).
  - Perhitungan otomatis estimasi pagu anggaran pengadaan (*Estimated Total Amount*) berdasarkan harga acuan master produk.
  - Integrasi audit chatter otomatis yang mencatat persetujuan, pembukaan lelang, dan penerbitan pesanan pembelian.
- **Lelang Multi-Vendor Kompetitif / Call for Bids (`type: purchase_tender`)**:
  - Pembukaan lelang tender pengadaan (`confirmed` -> `in_progress`).
  - Menerbitkan RFQ penawaran harga bersaing ke berbagai calon vendor rekanan (`POST /api/v1/purchase-requisitions/:id/generate-rfq`).
  - **Auto-Award & Competitor Auto-Cancellation Engine (`POST /api/v1/purchase-requisitions/:id/select-bid`)**:
    - Memilih vendor penawar terbaik (pemenang lelang), otomatis mengonfirmasi Purchase Order pemenang (`purchase`), membuat bukti penerimaan barang gudang (*stock picking WH/IN*), dan secara otomatis membatalkan seluruh RFQ penawaran pesaing lainnya (`state: cancel`).
    - Menutup status dokumen tender menjadi `done`.
- **Perjanjian Pasokan Berjangka / Blanket Purchase Orders (`type: blanket_order`)**:
  - Kontrak pembelian berulang jangka panjang dengan 1 vendor pilihan pada kuota kuantitas dan harga satuan yang telah disepakati di muka.
  - **Partial & Final PO Release (`POST /api/v1/purchase-requisitions/:id/release-blanket`)**:
    - Merilis Purchase Order bertahap (batch delivery) sesuai kebutuhan riil gudang/pabrik tanpa perlu negosiasi ulang.
    - Akumulasi kuantitas terpesan (`qty_ordered`) secara otomatis diperbarui pada setiap baris item perjanjian.
    - **Proteksi Anti Over-Quota**: Menolak permohonan rilis yang melebihi sisa kuota perjanjian yang sah.
    - Transisi otomatis ke status `done` saat seluruh kuota perjanjian telah terpenuhi 100%.
- **Web UI Tender & Blanket Order Hub**:
  - Dasbor metrik KPI pengadaan (Total Dokumen, Tender Aktif, Blanket Order, Selesai, Total Nilai Pagu Estimasi).
  - Tampilan tab rincian item, perbandingan penawaran rekanan (*Bid Comparison*), pemantauan sisa kuota, serta form interaktif pembuatan RFQ dan rilis parsial PO.

### 29. **Modul Multi-Currency Revaluation & Realized/Unrealized Forex Engine (`internal/modules/currency`)**
- **Standar Akuntansi PSAK 10 / IAS 21 Multi-Valas**:
  - Dukungan multi-mata uang internasional (`res.currency`) dengan Base Currency perusahaan (default: IDR) dan mata uang asing (USD, EUR, SGD, JPY, GBP, AUD, CNY).
  - Integrasi pencatatan multi-currency pada Buku Besar (*General Ledger*), Jurnal Entri, Faktur Penjualan (AR), dan Tagihan Pembelian (AP) melalui `CurrencyID`, `CurrencyRate`, dan `AmountCurrency`.
- **Manajemen Kurs Harian & Spot Rates JISDOR/BI (`res.currency.rates`)**:
  - Pencatatan riwayat kurs tanggal efektif (*effective date lookup*) dengan Direct Rate (e.g. 16.300 IDR/USD) dan Inverse Rate Enterprise ERP (Unit per 1 Base IDR).
  - Mesin konversi valas instan lintas mata uang (*cross-currency conversion engine*) dengan interpolasi kurs terdekat.
- **Revaluasi Kurs Bulanan / Month-End Forex Revaluation (`account.multicurrency.revaluation`)**:
  - Pemindaian otomatis faktur penjualan (AR) dan tagihan pembelian (AP) valas terbuka (*open balance*).
  - Perhitungan selisih kurs belum terealisasi (*Unrealized Foreign Exchange Gain/Loss*) terhadap kurs penutupan (*closing spot rate*).
  - **Posting Jurnal Penyesuaian Otomatis**: Membentuk entri jurnal seimbang (Debit == Credit) dengan akun Laba Selisih Kurs Belum Terealisasi (`710110`) dan Rugi Selisih Kurs Belum Terealisasi (`620110`).
- **Penyelesaian Transaksi & Selisih Kurs Terealisasi (*Realized Gain/Loss*)**:
  - Menghitung laba/rugi kurs riil saat pelunasan piutang/hutang valas (pembayaran invoice/bill) dibandingkan kurs historis saat booking.
  - Penyaluran otomatis ke akun Pendapatan Laba Kurs Terealisasi (`710100`) atau Beban Kerugian Kurs Terealisasi (`620100`).
- **Treasury AI Copilot & Forex Stress Test (`/api/v1/currencies/:id/ai-risk-audit`)**:
  - Didukung Cloudflare Workers AI Llama-3.1 & Heuristik Finansial.
  - Menghitung posisi terbuka Net Long / Net Short valas asing, matriks sensitivitas fluktuasi Rupiah (Shock ±2%, Shock ±5%, Shock ±10%), ringkasan risiko eksekutif, serta rekomendasi instrumen lindung nilai (*Hedging / Domestic Non-Deliverable Forward DNDF*).

### 30. **Modul Pajak Pertambahan Nilai & e-Faktur Elektronik DJP (`internal/modules/tax_efaktur`)**
- **Regulasi Perpajakan Indonesia (UU HPP & PMK 112/2022)**:
  - **Manajemen Jatah Nomor Seri Faktur Pajak (NSFP) DJP (`account_tax_nsfps`)**: Alokasi kuota seri faktur resmi dari DJP Online dengan validasi rentang awal-akhir 8-digit, pemantauan kuota sisa/terpakai, auto-increment atomik, dan pencegahan penomoran ganda.
  - **Penerbitan e-Faktur Pajak Keluaran (`account_tax_invoices`)**: Terintegrasi langsung dengan Customer Invoices (`account.move`) posted. Otomatis menghitung DPP, PPN 11%, pemotongan PPh (PPh 23, PPh 4 ayat 2, PPh 22), pembentukan nomor 16-digit standar DJP (`010.001-26.00000001`), dan payload QR Code verifikasi DJP (`efaktur.pajak.go.id`).
  - **Pencatatan e-Faktur Pajak Masukan**: Pencatatan faktur masukan dari tagihan vendor (`Vendor Bills`) untuk pengkreditan PPN pada Masa Pajak bersangkutan.
  - **Rekapitulasi SPT Masa PPN 1111 (Formulir Induk & AB)**: Perhitungan komprehensif DPP Penyerahan, PPN Keluaran Dipungut, DPP Perolehan, PPN Masukan Dikreditkan, serta status fiskal (*Kurang Bayar / Lebih Bayar / Nihil*).
  - **Mesin Ekspor CSV Resmi DJP (`ExportEFakturCSV`)**: Menghasilkan file CSV spesifikasi resmi aplikasi e-Faktur Desktop DJP lengkap dengan baris header `FK` (*Faktur Pajak*), baris `LT` (*Lawan Transaksi*), dan baris detail `OF` (*Objek Faktur Barang/Jasa Kena Pajak*) dengan sanitasi kutip ganda dan pembersihan format NPWP.
  - **Cloudflare Workers AI Tax Compliance Copilot (`/api/v1/tax-efaktur/ai-audit`)**: Validasi kepatuhan format NPWP 15/16-digit NIK baru PMK 112/2022, keselarasan tanggal faktur komersial vs masa pajak fiskal, skor kepatuhan fiskal (0-100%), level risiko denda, dan rekomendasi mitigasi perpajakan.
  - **Universal Chatter Audit Trail**: Pencatatan jejak audit penerbitan, pelaporan, dan pembatalan faktur pada model `account.tax.invoice`.

### 31. **Modul Rekonsiliasi Bank & Impor Rekening Koran (`internal/modules/bank_reconciliation`) - `account.bank.statement` & Auto-Match Engine**
- **Arsitektur Rekonsiliasi Modern Bergaya Modern Enterprise**:
  - **Rekening Koran Bank (`account_bank_statements` / `account.bank.statement`)**: Header rekonsiliasi yang mencatat Jurnal Bank (`BNK`), Tanggal, Saldo Awal (*Balance Start*), Saldo Akhir Buku (*Balance End*), Saldo Fisik Rekening Koran (*Balance End Real*), dan Selisih (*Difference*).
  - **Baris Transaksi Mutasi Bank (`account_bank_statement_lines` / `account.bank.statement.line`)**: Menyimpan riwayat pergerakan kas masuk (inflow/CR bernilai positif) dan kas keluar (outflow/DB bernilai negatif), memo transaksi dari bank, kode referensi, status rekonsiliasi, skor keyakinan pencocokan (*Auto-Match Confidence*), dan tautan faktur/jurnal.
  - **Aturan Rekonsiliasi Otomatis (`account_reconcile_models` / `account.reconcile.model`)**: Pengaturan aturan pencocokan otomatis (*contains*, *regex*, *exact*) untuk pembebanan langsung biaya admin bank, bunga giro, atau potongan transfer antar-bank ke akun GL tujuan.
- **Smart CSV Mutasi Parser Multi-Bank**:
  - Mendukung ekspor mutasi bank terpopuler di Indonesia:
    1. **Format BCA**: Format e-Banking KlikBCA / KlikBCA Bisnis (`Tanggal, Keterangan, CB, Mutasi, Saldo` dengan suffix `CR`/`DB`).
    2. **Format Bank Mandiri & BRI**: Format kolom berpasangan (`Tanggal, Deskripsi, Debit, Kredit, Saldo`).
    3. **Format Standar / Generic CSV**: Format umum dengan kolom `Date, Memo, Amount, Reference`.
  - Penanganan fleksibel pemisah ribuan titik/koma format Indonesia (`15.000.000,00`) maupun internasional (`15,000,000.00`), dan pemisah baris koma (`,`) maupun titik-koma (`;`).
- **Algorithmic Auto-Match Engine**:
  - **Aturan 1 (Regex Faktur Akurat - 98% Akurasi)**: Mendeteksi pola nomor faktur penjualan atau tagihan vendor (misal `INV/2026/00001` atau `BILL/2026/00001`) dalam memo bank dan mencocokkannya ke faktur terbuka dengan nilai sisa (*residual*) yang pas.
  - **Aturan 2 (Aturan Biaya Bank GL - 95% Akurasi)**: Mencocokkan memo seperti `BIAYA ADM`, `ADM BULANAN GIRO`, `BIAYA TRSF`, atau `BUNGA` ke akun beban operasional bank (`620100`) atau pendapatan bunga bank (`710100`).
  - **Aturan 3 (Identifikasi Rekanan & Nilai Pas - 90% Akurasi)**: Mencocokkan nama pelanggan/vendor dalam memo bank yang memiliki sisa tagihan persis sama dengan nilai mutasi.
  - **Aturan 4 (Nominal Pas Tunggal - 78% Akurasi)**: Menyarankan faktur terbuka tunggal yang memiliki sisa piutang/hutang persis sama dengan arah mutasi kas.
- **Eksekusi Rekonsiliasi Atomik (*Atomic Reconcile Execution*)**:
  - **Pencocokan Faktur Terbuka**: Mendaftarkan pembayaran resmi via `account.payment`, mengurangi saldo residual faktur, menandai faktur `paid`, dan menandai baris mutasi `is_reconciled = true`.
  - **Write-Off / Pembukuan Biaya Bank Langsung**: Otomatis membuat dan memposting entri jurnal akuntansi berimbang (`account.move` tipe `entry`) dengan mendebit akun beban bank dan mengkredit akun kas bank jurnal terkait.
  - **Fitur 1-Click Auto-Reconcile (>90%)**: Merekonsiliasi seluruh mutasi yang memiliki tingkat keyakinan tinggi secara serentak.
- **Penutupan & Penguncian Rekening Koran (`ActionConfirm`)**:
  - Memverifikasi secara ketat bahwa seluruh mutasi telah direkonsiliasi dan selisih saldo buku vs fisik adalah Rp 0.00 ($\text{Difference} == 0$). Mengunci rekening koran dari perubahan lebih lanjut.
- **Cloudflare Workers AI Bank Copilot (`/api/v1/bank-reconcile/statements/:id/ai-audit`)**:
  - Mengaudit rekening koran secara cerdas menggunakan LLM `@cf/meta/llama-3.1-8b-instruct` (didukung *Smart Local Heuristic Fallback*).
  - Mengevaluasi tingkat risiko audit (*Low, Medium, High Risk*), menganalisis penyebab diskrepansi saldo, mendeteksi pengeluaran besar tanpa faktur, dan menyajikan rekomendasi tindakan mitigasi CFO.
- **Ekspor CSV Mutasi & Universal Chatter Audit Trail**:
  - Fasilitas pengunduhan data mutasi ke file CSV UTF-8.
  - Pencatatan seluruh riwayat pembuatan, impor CSV, pencocokan otomatis, dan penutupan rekening koran pada audit trail `chatter` (`account.bank.statement`).

### 32. **Modul Landed Costs Engine & Alokasi Biaya Impor/Logistik (`internal/modules/landed_cost`) - `stock.landed.cost`**
- **Arsitektur Standard Enterprise ERP (`stock.landed.cost`, `stock.landed.cost.line`, `stock.valuation.adjustment.lines`)**:
  - Mengalokasikan biaya riil pengadaan internasional dan logistik (seperti *Ocean Freight*, Bea Masuk & Pajak Impor DJBC, *Terminal Handling Charges/THC Pelabuhan*, *Demurrage*, dan Asuransi Kargo) langsung ke nilai perolehan persediaan barang (*Stock Valuation Layer*).
  - Terhubung langsung dengan dokumen Penerimaan Barang Gudang (*Incoming Stock Receipt / Picking* `WH/IN`).
- **5 Metode Alokasi Biaya (*Split Allocation Methods*)**:
  1. **`by_value` (Berdasarkan Nilai Barang)**: Alokasi proporsional berdasarkan total harga beli barang ($Qty \times Cost$), ideal untuk asuransi dan premi kargo.
  2. **`by_quantity` (Berdasarkan Kuantitas Fisik)**: Alokasi proporsional berdasarkan jumlah unit fisik barang, ideal untuk tarif penanganan per item atau bea per kemasan.
  3. **`by_weight` (Berdasarkan Bobot Kotor / Gross Weight)**: Alokasi proporsional berdasarkan berat total barang (kg), standar industri untuk angkutan udara (*Air Freight*).
  4. **`by_volume` (Berdasarkan Kubikasi / Volume m3)**: Alokasi proporsional berdasarkan ruang kubikasi kontainer, standar industri untuk angkutan laut (*Ocean/Sea Freight*).
  5. **`equal` (Dibagi Rata Sama Banyak)**: Alokasi dibagi rata sama besar di antara seluruh produk yang diimpor.
- **Simulasi Penyesuaian Nilai Persediaan (*Stock Valuation Adjustment Lines*)**:
  - Menghitung HPP Awal (*Former Cost*), beban landed cost tambahan per unit (*Additional Landed Cost*), HPP Final (*Final Cost*), dan HPP per unit baru (*Final Cost per Unit*) dengan kompensasi pembulatan sen/rupiah tanpa deviasi saldo.
- **Pembukuan Jurnal Akuntansi Atomik (*Atomic General Ledger Posting*)**:
  - Saat divalidasi (*ActionValidate*), otomatis membuat dan memposting entri jurnal berimbang ($\sum \text{Debit} == \sum \text{Credit}$):
    - **Debit**: Persediaan Barang Dagang (`130000` Stock Valuation Account) untuk menaikkan nilai aset persediaan.
    - **Credit**: Akun Antara Biaya Impor & Logistik (`130090` Landed Cost Interim Clearing Account).
  - **Pembaruan Harga Pokok Standar Otomatis**: Memperbarui harga pokok (*Standard Price / Cost Price*) pada master produk (`product.product`) berdasarkan biaya landed baru.
- **Cloudflare Workers AI Landed Cost & Profitability Copilot (`/api/v1/landed-costs/:id/ai-audit`)**:
  - Didukung LLM Cloudflare Workers AI `@cf/meta/llama-3.3-70b-instruct` dan Smart Local Heuristic Engine.
  - Menghitung rasio beban pengadaan (*overhead percentage*), mensimulasikan dampak kenaikan HPP terhadap *Gross Margin* harga jual ritel (*List Price*), mendeteksi anomali produk berisiko margin tipis (*Tight Margin <20%*) atau negatif (*Negative Margin*), serta menyajikan rekomendasi konsolidasi kontainer FCL (*Full Container Load*).
- **Universal Chatter Audit Trail**:
  - Pencatatan jejak audit otomatis mulai dari pembuatan draf, simulasi alokasi, posting GL, hingga pembatalan dokumen pada `stock.landed.cost`.

### 33. **Mesin Ekspor Data Excel & CSV (`internal/core/export`)**
- Menghasilkan file CSV terstandarisasi dengan **UTF-8 BOM (`\xEF\xBB\xBF`)** agar langsung terbaca rapi saat dibuka di Microsoft Excel tanpa masalah encoding simbol mata uang Rupiah maupun teks bahasa Indonesia.
- Tersedia pada Katalog Produk, Riwayat Sales Orders, dan Laporan Neraca Saldo (*Trial Balance*).

### 34. **Cloudflare Workers AI & Edge Copilot (`internal/core/ai` & `cloudflare/`)**
- **Dual-Mode AI Engine**:
  - **Mode Cloudflare Edge GPU**: Terhubung langsung ke jaringan global Cloudflare Workers AI menggunakan model LLM unggulan `@cf/meta/llama-3.1-8b-instruct` dan vision model `@cf/meta/llama-3.2-11b-vision-instruct` via REST API atau edge binding.
  - **Mode Smart Local Heuristic**: Fallback offline otomatis tanpa dependensi eksternal, memastikan seluruh alur kerja ERP dan unit test tetap berjalan 100% cepat dan deterministik di lingkungan lokal/on-premise.
- **AI Budget Variance & Overspend Auditor (`/api/v1/budget/budgets/:id/ai-audit`)**:
  - Analisis penyerapan anggaran biaya, deteksi pos overbudget, kecepatan konsumsi pagu (burn rate), dan rekomendasi pengendalian belanja CFO.
- **AI Tax, Overtime & Labor Compliance Auditor (`/api/v1/payroll/payslips/:id/ai-audit`)**:
  - Menganalisis kepatuhan komponen gaji, batas lembur UU Kemenaker, potongan iuran BPJS TK/Kes, dan PPh 21 TER UU HPP.
- **AI Warehouse Shrinkage & Loss Prevention Auditor (`/api/v1/stock-inventory/inventories/:id/ai-audit`)**:
  - Menganalisis pola selisih fisik stok, mendeteksi kerugian persediaan abnormal, serta memberikan rekomendasi pengamanan dan pencegahan kebocoran gudang.
- **AI Talent Acquisition & CV Screening (`/api/v1/recruitment/applicants/:id/ai-screening`)**:
  - Evaluasi kecocokan resume kandidat pelamar, rekomendasi perekrutan (*Strong Hire/Hire/Consider/Reject*), identifikasi kelebihan/kekurangan, dan panduan pertanyaan wawancara spesifik.
- **AI Vision Receipt & Invoice OCR (`/api/v1/ai/receipt-ocr`)**:
  - Mengekstrak teks, nama toko/merchant, tanggal transaksi, pajak PPN, dan rincian belanja dari struk foto ke JSON terstruktur.
- **AI Helpdesk Ticket Triage & Auto-Reply (`/api/v1/ai/ticket-triage`)**:
  - Analisis sentimen pelanggan secara otomatis (*urgent*, *frustrated*, *neutral*).
  - Klasifikasi kategori kendala (*technical*, *billing*, *warranty*), rekomendasi tingkat prioritas, dan pembuatan draf balasan ramah & profesional berbasis SLA.
- **AI Quality RCA (5-Whys) & CAPA Advisor (`/api/v1/ai/quality-rca`)**:
  - Analisis akar masalah kegagalan mutu (*Root Cause Analysis* metodologi 5-Whys).
  - Rekomendasi tindakan korektif dan pencegahan (*Corrective & Preventive Action* - CAPA) sesuai standar ISO 9001:2015 serta penentuan rekomendasi tindakan fisik barang (*Quarantine*, *Rework*, *Return*, *Scrap*).
- **AI Auto-Categorization & GL Account Mapping (`/api/v1/ai/expense-categorize`)**:
  - Menguraikan teks kwitansi pengeluaran karyawan dan otomatis memetakan ke kategori biaya yang tepat serta nomor Bagan Akun Buku Besar GL (*General Ledger*).
  - Pemeriksaan kepatuhan perpajakan (*tax-deductible compliance*).
- **AI Corporate Approval Risk & Compliance Assessment (`/api/v1/ai/approval-risk`)**:
  - Mengevaluasi justifikasi permohonan, nominal rupiah, dan batas delegasi kewenangan (*Delegation of Authority*).
  - Menghasilkan tingkat risiko (*Low, Medium, High, Critical*), analisis kepatuhan anggaran, serta daftar pertanyaan audit kunci sebelum eksekutif menandatangani persetujuan.
- **AI Attendance Anomaly & Employee Burnout Risk Analyzer (`/api/v1/attendance/ai-anomaly-check`)**:
  - Menganalisis log presensi karyawan, anomali geofence/telemetri Edge, serta risiko beban kerja lembur berlebih (*work-life balance audit*).
- **Executive Natural Language Query ("Ask ERP AI" / `/api/v1/ai/ask-erp`)**:
  - Membantu eksekutif C-Level dan manajer bertanya dalam bahasa alami tentang status pendapatan, volume pesanan, tiket keluhan, dan mutu produksi dengan injeksi konteks data transaksi *real-time*.
- **Cloudflare Edge Deployment Artifacts**:
  - `cloudflare/worker.js`: Cloudflare Worker edge script yang memanggil `env.AI` binding secara native dengan fitur CORS dan caching.
  - `cloudflare/wrangler.toml`: Konfigurasi Wrangler untuk deployment instan ke Cloudflare edge network.
  - `cloudflare/cloudflared-config.yml`: Konfigurasi zero-trust Cloudflare Tunnel untuk menghubungkan backend Go lokal ke domain publik Cloudflare tanpa membuka port firewall masuk (*no inbound ports*).

### 30. **Web Dashboard Interaktif (`web/`)**
- Antarmuka web responsif dan modern (*glassmorphic design*) yang langsung disajikan oleh server Go di port `8080`.
- Dilengkapi tombol interaktif **📷 Scan Barcode**, **✨ Ask ERP AI Copilot**, asisten AI satu-klik pada formulir Helpdesk, Quality Control, Pengajuan Biaya (Expense), Matriks Persetujuan (Approvals), Presensi & Kiosk Mandiri (Attendance), Dokumen & R2 Storage Hub, Rekrutmen & AI Talent, Stock Opname Fisik, Enterprise Payroll, Enterprise Budgeting, serta **Barcode & QR Code Scanner Engine** dengan emulasi scanner gun dan cetak label vektor SVG Code 128.
- Memungkinkan simulasi dan pengujian transaksi bisnis lengkap langsung dari peramban untuk seluruh 27 modul bisnis.

---

## 📂 Struktur Direktori Proyek

```
erp-golang/
├── erp_reference/          # Source code referensi hasil pull dari erp/erp
├── cloudflare/              # Edge deployment configs (Workers AI & Cloudflare Tunnel)
│   ├── worker.js            # Cloudflare Worker script dengan binding env.AI
│   ├── wrangler.toml        # Wrangler CLI config untuk Cloudflare edge deploy
│   └── cloudflared-config.yml # Zero-Trust Cloudflare Tunnel template
├── cmd/
│   └── server/
│       └── main.go          # Entrypoint server, koneksi DB, dan seeding data awal
├── internal/
│   ├── config/              # Konfigurasi aplikasi, DB, Cloudflare R2 & Workers AI
│   ├── core/
│   │   ├── ai/              # Dual-mode AI Copilot & Vision OCR (Cloudflare + Local Heuristic)
│   │   ├── database/        # Koneksi GORM & auto-migration
│   │   ├── export/          # CSV & Excel Export Engine (UTF-8 BOM)
│   │   ├── middleware/      # JWT, RBAC, Multi-Company scoping, CORS
│   │   ├── models/          # ModelBase (ID, Timestamps, CompanyID, Auditing)
│   │   ├── registry/        # Module registry & lifecycle manager
│   │   └── storage/         # Universal Storage Provider (Cloudflare R2 + Local Disk fallback)
│   └── modules/
│       ├── base/            # User, Company, Partner, Currency, Auth
│       ├── product/         # Product, Category, UoM
│       ├── stock/           # Multi-Warehouse, Location, Picking, Quant, Inter-WH Transfer
│       ├── sale/            # Quotation, Sale Order, Order Lines
│       ├── purchase/        # RFQ, Purchase Order, PO Lines
│       ├── account/         # Journal, Account Move (Invoice), Move Lines, Payment
│       ├── mrp/             # BOM, Manufacturing Order, Production Moves
│       ├── pos/             # POS Session, Cash Register, POS Order
│       ├── hr/              # Department, Job, Employee, Payslip
│       ├── crm/             # CRM Leads, Opportunities, Stage Transitions
│       ├── chatter/         # Universal Audit Trail, Activity & Internal Notes
│       ├── project/         # Projects, Tasks, Kanban Stages, Timesheets
│       ├── asset/           # Fixed Assets, Linear Depreciation Board, GL Posting
│       ├── helpdesk/        # Helpdesk Teams, Customer Tickets, SLA Stages, Resolution
│       ├── maintenance/     # Equipment, Preventive Scheduler, Corrective Repair Orders
│       ├── fleet/           # Vehicles, Models, Odometer Logs, Service & KIR Records
│       ├── leave/           # Leave Types, Quota Allocations, Requests, Payroll Deductions
│       ├── expense/         # Expense Categories, Claims, Expense Sheets, GL Posting
│       ├── quality/         # Quality Points, Inspection Checks, Tolerance & Quality Alerts (NCR)
│       ├── approval/        # Multi-Tier Approval Matrix, Categories, Requests & AI Risk Assessment
│       ├── attachment/      # Dokumen & R2 Storage Hub, ir.attachment, AI Vision OCR Struk
│       ├── attendance/      # Presensi, Shift Kerja, Geofencing, Cloudflare Edge Telemetry, Payroll Sync
│       ├── recruitment/     # Lowongan Kerja, Pipeline Pelamar, Cloudflare AI CV Screening & 1-Click Hire
│       ├── stock_inventory/ # Stock Opname Fisik, Snapshot Quant, AI Shrinkage Audit & GL Posting
│       ├── payroll/         # Enterprise Payroll, Salary Rules Engine, AI Tax Auditor & Balanced GL Posting
│       ├── budget/          # Enterprise Budgeting, Analytic Accounts & Overspend Auditor
│       ├── barcode/         # Barcode & QR Code Engine, GS1-128, Tracking Lot/Serial
│       ├── purchase_requisition/ # Tender Pengadaan & Blanket Purchase Orders
│       ├── currency/        # Multi-Currency, Historical Spot Rates & Forex Revaluation (PSAK 10)
│       ├── tax_efaktur/     # Indonesian e-Faktur Pajak DJP, NSFP Quota & SPT Masa PPN 1111
│       ├── bank_reconciliation/ # Bank Statement Import (CSV), Auto-Matching Engine & Reconciliation
│       └── landed_cost/     # Landed Costs Engine & Alokasi Biaya Impor (5 Split Methods & GL Posting)
├── web/                     # Single Page Application Web Dashboard & AI Copilot UI
│   ├── index.html
│   ├── app.js
│   └── style.css
├── tests/
│   └── erp_test.go          # End-to-end integration test (32 modul, 36 sections, 100% pass)
├── Dockerfile               # Multi-stage production container
├── docker-compose.yml       # Stack: Go Backend + PostgreSQL 16 + Redis 7
├── .env.example             # Template konfigurasi environment
├── erp.db              # Database SQLite otomatis
└── erp-server               # Binary terkompilasi
```

---

## 🛠️ Cara Menjalankan

### 1. Menjalankan Binary Langsung
```bash
./erp-server
```

### 2. Menjalankan via Go Run
```bash
go run cmd/server/main.go
```

Server akan aktif di:
- **Web Dashboard**: [http://localhost:8080/](http://localhost:8080/)
- **API Health**: [http://localhost:8080/health](http://localhost:8080/health)
- **Default Admin Login**: `admin@erp.local` / `admin123`

### 3. Menjalankan via Docker Compose
```bash
docker compose up -d --build
```
Stack container akan otomatis menyiapkan:
- **ERP Server**: Port 8080
- **PostgreSQL 16**: Port 5432
- **Redis 7 Cache**: Port 6379

### 4. Menjalankan Automated Test
```bash
export PATH=$HOME/go-sdk/go/bin:$PATH && export GOTOOLCHAIN=local
go test -v ./tests/...
```

---

## 📡 Dokumentasi Endpoint REST API

| Method | Endpoint | Deskripsi |
|---|---|---|
| `POST` | `/api/v1/auth/login` | Login user & dapatkan JWT Token |
| `GET` | `/api/v1/auth/me` | Profil user aktif |
| `GET` | `/api/v1/dashboard/stats` | Statistik metrik eksekutif seluruh modul |
| `GET` | `/api/v1/partners` | Daftar Kontak / Partners (filter: `customer`, `supplier`) |
| `POST` | `/api/v1/partners` | Buat kontak baru |
| `GET` | `/api/v1/products` | Katalog produk & saldo stok per unit |
| `POST` | `/api/v1/products` | Tambah produk baru |
| `GET` | `/api/v1/sales/orders` | Daftar penawaran & sales orders |
| `POST` | `/api/v1/sales/orders` | Buat penawaran (*quotation*) |
| `POST` | `/api/v1/sales/orders/:id/confirm` | Konfirmasi SO $\rightarrow$ otomatis buat Delivery Order |
| `GET` | `/api/v1/purchases/orders` | Daftar RFQ & purchase orders |
| `POST` | `/api/v1/purchases/orders` | Buat RFQ pembelian |
| `POST` | `/api/v1/purchases/orders/:id/confirm` | Konfirmasi PO $\rightarrow$ otomatis buat Stock Receipt |
| `GET` | `/api/v1/stock/pickings` | Daftar pengiriman, penerimaan, transfer barang |
| `POST` | `/api/v1/stock/pickings/:id/validate` | Validasi perpindahan stok pembukuan ganda |
| `GET` | `/api/v1/stock/warehouses` | 🏬 Daftar gudang & cabang (*Multi-Warehouse*) |
| `POST` | `/api/v1/stock/warehouses` | Tambah gudang / pusat distribusi baru |
| `POST` | `/api/v1/stock/transfers` | 🔄 Buat & jalankan transfer stok antar-gudang (*Inter-Warehouse*) |
| `GET` | `/api/v1/stock/quants` | Saldo riil stok per produk dan lokasi gudang |
| `GET` | `/api/v1/stock/orderpoints` | Daftar aturan stok minimum (*Reordering Rules Min/Max*) |
| `POST` | `/api/v1/stock/orderpoints` | Buat aturan pemesanan ulang stok baru |
| `GET` | `/api/v1/stock/pickings/:id/print` | 🖨️ Cetak Surat Jalan / Dokumen Pengiriman (*Printable HTML*) |
| `POST` | `/api/v1/purchases/reordering/run` | ⚡ Eksekusi pengadaan otomatis (*Auto-generate RFQ*) |
| `GET` | `/api/v1/accounting/moves` | Daftar faktur tagihan pelanggan & vendor |
| `POST` | `/api/v1/accounting/moves/:id/post` | Validasi $\sum \text{Debit} == \sum \text{Credit}$ dan posting |
| `POST` | `/api/v1/accounting/moves/:id/pay` | Registrasi pembayaran tagihan |
| `GET` | `/api/v1/accounting/moves/:id/print` | 🖨️ Cetak Faktur Penjualan / Tagihan (*Printable HTML*) |
| `GET` | `/api/v1/accounting/reports/profit-loss` | Laporan Laba Rugi (*Profit and Loss / Income Statement*) |
| `GET` | `/api/v1/accounting/reports/balance-sheet` | Laporan Neraca Keuangan (*Balance Sheet*) |
| `GET` | `/api/v1/accounting/reports/trial-balance` | Laporan Neraca Saldo (*Trial Balance*) |
| `GET` | `/api/v1/mrp/boms` | Daftar Bill of Materials (BOM) perakitan |
| `POST` | `/api/v1/mrp/boms` | Buat struktur BOM baru |
| `GET` | `/api/v1/mrp/productions` | Daftar Perintah Manufaktur / MO |
| `POST` | `/api/v1/mrp/productions` | Buat Manufacturing Order baru |
| `POST` | `/api/v1/mrp/productions/:id/produce` | Jalankan produksi $\rightarrow$ kurangi bahan baku, tambah barang jadi |
| `GET` | `/api/v1/pos/configs` | Daftar Terminal / Register Kasir |
| `GET` | `/api/v1/pos/sessions` | Sesi kasir aktif & historis |
| `POST` | `/api/v1/pos/sessions/open` | Buka sesi kasir baru dengan modal awal |
| `POST` | `/api/v1/pos/sessions/:id/close` | Tutup sesi kasir & rekonsiliasi kas |
| `GET` | `/api/v1/pos/orders` | Histori transaksi pesanan POS |
| `POST` | `/api/v1/pos/orders` | Input transaksi POS ritel $\rightarrow$ potong stok instan |
| `GET` | `/api/v1/hr/departments` | Daftar Departemen Perusahaan |
| `GET` | `/api/v1/hr/jobs` | Daftar Jabatan Kerja (*Job Positions*) |
| `GET` | `/api/v1/hr/employees` | Direktori Karyawan & Gaji Pokok |
| `POST` | `/api/v1/hr/employees` | Registrasi karyawan baru |
| `GET` | `/api/v1/hr/payslips` | Daftar Slip Gaji / Payroll |
| `POST` | `/api/v1/hr/payslips/generate` | Generate slip gaji bulanan karyawan |
| `POST` | `/api/v1/hr/payslips/:id/pay` | Validasi dan bayar slip gaji |
| `GET` | `/api/v1/crm/leads` | Daftar prospek & peluang pipeline CRM |
| `POST` | `/api/v1/crm/leads` | Buat peluang / lead baru |
| `PUT` | `/api/v1/crm/leads/:id/stage` | Pindahkan tahapan pipeline (`new`, `qualified`, `proposition`, `won`, `lost`) |
| `POST` | `/api/v1/crm/leads/:id/convert` | 🛒 Konversi peluang menjadi penawaran harga (*Sales Quotation*) |
| `GET` | `/api/v1/chatter/:model/:res_id` | Histori jejak audit & diskusi untuk dokumen tertentu |
| `POST` | `/api/v1/chatter/:model/:res_id` | 💬 Kirim pesan / catatan internal pada chatter dokumen |
| `GET` | `/api/v1/products/export` | 📥 Ekspor katalog produk ke file CSV (*Excel compatible*) |
| `GET` | `/api/v1/sales/orders/export` | 📥 Ekspor daftar sales orders ke file CSV |
| `GET` | `/api/v1/accounting/reports/export/trial-balance` | 📥 Ekspor neraca saldo (*Trial Balance*) ke file CSV |
| `GET` | `/api/v1/projects` | Daftar proyek aktif & selesai |
| `POST` | `/api/v1/projects` | Buat proyek bisnis baru |
| `GET` | `/api/v1/projects/:id` | Detail proyek & daftar tugas terkait |
| `GET` | `/api/v1/tasks` | Daftar tugas (filter: `project_id`, `stage`) |
| `POST` | `/api/v1/tasks` | Buat tugas baru dalam proyek |
| `GET` | `/api/v1/tasks/:id` | Detail tugas, estimasi jam, & histori timesheet |
| `PUT` | `/api/v1/tasks/:id/stage` | Update tahapan tugas (`todo`, `in_progress`, `review`, `done`) |
| `POST` | `/api/v1/tasks/:id/timesheets` | ⏱️ Log jam kerja / timesheet pada tugas |
| `GET` | `/api/v1/tasks/:id/timesheets` | Riwayat log jam kerja pada tugas |
| `GET` | `/api/v1/assets` | Daftar aset tetap perusahaan (filter: `state`) |
| `POST` | `/api/v1/assets` | Registrasi aset tetap baru & kalkulasi jadwal depresiasi |
| `GET` | `/api/v1/assets/:id` | Detail aset & papan jadwal penyusutan (*Depreciation Board*) |
| `POST` | `/api/v1/assets/:id/compute` | Hitung ulang jadwal depresiasi aset |
| `POST` | `/api/v1/assets/:id/confirm` | Konfirmasi aset $\rightarrow$ status aktif berjalan (*Running*) |
| `POST` | `/api/v1/assets/lines/:line_id/post` | ⚡ Posting baris depresiasi ke Buku Besar (*General Ledger Entry*) |
| `POST` | `/api/v1/assets/auto-depreciate` | ⚡ Eksekusi otomatis posting depresiasi yang jatuh tempo |
| `GET` | `/api/v1/helpdesk/teams` | Daftar tim helpdesk customer care & technical support |
| `GET` | `/api/v1/helpdesk/tickets` | Daftar tiket keluhan pelanggan (filter: `team_id`, `stage`, `priority`) |
| `POST` | `/api/v1/helpdesk/tickets` | Buat tiket bantuan pelanggan baru |
| `PUT` | `/api/v1/helpdesk/tickets/:id/stage` | Update tahapan tiket (`new`, `in_progress`, `waiting`, `solved`, `cancelled`) |
| `PUT` | `/api/v1/helpdesk/tickets/:id/assign` | Tugaskan tiket ke teknisi/agen support |
| `POST` | `/api/v1/helpdesk/tickets/:id/resolve` | Selesaikan tiket dengan catatan solusi (*Resolution Summary*) |
| `GET` | `/api/v1/maintenance/equipments` | Katalog mesin pabrik & peralatan kerja |
| `POST` | `/api/v1/maintenance/equipments` | Daftarkan mesin/peralatan baru & interval servis preventif |
| `GET` | `/api/v1/maintenance/requests` | Daftar perintah kerja perawatan & perbaikan mesin |
| `POST` | `/api/v1/maintenance/requests` | Laporkan kerusakan mesin darurat (*Corrective Breakdown*) |
| `POST` | `/api/v1/maintenance/scheduler/run` | ⚡ Eksekusi otomatis jadwal perawatan berkala yang jatuh tempo |
| `POST` | `/api/v1/maintenance/requests/:id/complete` | Selesaikan servis mesin & pulihkan status menjadi *Operational* |
| `GET` | `/api/v1/fleet/models` | Daftar model & merk armada kendaraan logistik |
| `GET` | `/api/v1/fleet/vehicles` | Daftar armada kendaraan operasional (filter: `status`, `vehicle_type`) |
| `POST` | `/api/v1/fleet/vehicles` | Registrasi kendaraan baru, supir penanggung jawab & aset tetap |
| `POST` | `/api/v1/fleet/odometer` | Catat log jarak tempuh KM & sinkronisasi odometer armada |
| `POST` | `/api/v1/fleet/services` | Catat histori servis berkala, ganti ban/oli & uji berkala KIR Dishub |
| `GET` | `/api/v1/leave/types` | Daftar kategori & jenis cuti/izin karyawan |
| `POST` | `/api/v1/leave/types` | Tambah jenis cuti baru |
| `GET` | `/api/v1/leave/allocations` | Rekapitulasi jatah kuota tahunan cuti per karyawan |
| `POST` | `/api/v1/leave/allocations` | Tetapkan alokasi kuota cuti tahunan karyawan |
| `GET` | `/api/v1/leave/requests` | Daftar permohonan cuti (filter: `employee_id`, `state`, `year`) |
| `POST` | `/api/v1/leave/requests` | Ajukan permohonan cuti / izin kerja |
| `PUT` | `/api/v1/leave/requests/:id/approve` | Setujui cuti $\rightarrow$ potong kuota tahunan otomatis |
| `PUT` | `/api/v1/leave/requests/:id/refuse` | Tolak permohonan cuti dengan alasan tertulis |
| `GET` | `/api/v1/expenses/categories` | Daftar master kategori biaya & pemetaan akun GL |
| `POST` | `/api/v1/expenses/categories` | Tambah kategori pengeluaran baru |
| `GET` | `/api/v1/expenses/items` | Daftar rincian klaim item kwitansi pengeluaran |
| `POST` | `/api/v1/expenses/items` | Catat item pengeluaran dengan tag proyek / armada |
| `DELETE` | `/api/v1/expenses/items/:id` | Hapus item pengeluaran draft |
| `GET` | `/api/v1/expenses/sheets` | Rekapitulasi laporan pengajuan reimbursement (*Expense Reports*) |
| `POST` | `/api/v1/expenses/sheets` | Buat berkas laporan pengajuan baru bundling multi-item |
| `PUT` | `/api/v1/expenses/sheets/:id/submit` | Ajukan laporan ke manajer/finance untuk approval |
| `PUT` | `/api/v1/expenses/sheets/:id/approve` | Persetujuan manajerial (*Manager Approval*) |
| `POST` | `/api/v1/expenses/sheets/:id/post` | ⚡ Bukukan langsung ke Buku Besar Akuntansi (*General Ledger Entry*) |
| `PUT` | `/api/v1/expenses/sheets/:id/pay` | Registrasi pembayaran transfer kas/bank kepada karyawan (*Done/Paid*) |
| `PUT` | `/api/v1/expenses/sheets/:id/refuse` | Tolak laporan pengajuan dengan alasan penolakan tertulis |
| `GET` | `/api/v1/expenses/stats` | KPI metrik pengeluaran, pending approval, paid & kategori |
| `GET` | `/api/v1/quality/points` | Daftar SOP & aturan titik kontrol mutu (*Quality Control Points*) |
| `POST` | `/api/v1/quality/points` | Buat aturan titik kontrol mutu baru |
| `GET` | `/api/v1/quality/checks` | Daftar lembar pemeriksaan mutu harian (filter: `picking_id`, `state`) |
| `POST` | `/api/v1/quality/checks` | Buat lembar pemeriksaan mutu manual |
| `POST` | `/api/v1/quality/checks/:id/pass` | ✓ Nyatakan lolos uji (*Pass*) dengan validasi toleransi ketat |
| `POST` | `/api/v1/quality/checks/:id/fail` | ❌ Nyatakan gagal (*Fail*) & otomatis terbitkan Quality Alert NCR |
| `POST` | `/api/v1/quality/checks/generate/picking/:picking_id` | ⚡ Otomatis generate lembar inspeksi untuk penerimaan gudang WH/IN |
| `POST` | `/api/v1/quality/checks/generate/production/:production_id` | ⚡ Otomatis generate lembar inspeksi untuk perakitan manufaktur MO |
| `GET` | `/api/v1/quality/alerts` | Daftar laporan ketidaksesuaian mutu (filter: `stage`, `priority`) |
| `POST` | `/api/v1/quality/alerts` | Terbitkan Quality Alert & tindakan karantina barang baru |
| `PUT` | `/api/v1/quality/alerts/:id/stage` | Update tahapan penanganan alert (`new`, `confirmed`, `in_progress`, `resolved`, `closed`) |
| `GET` | `/api/v1/quality/stats` | KPI metrik total inspeksi, tingkat kelolosan (Pass Rate %), dan alert aktif |
| `GET` | `/api/v1/approvals/categories` | Daftar master template & aturan kategori persetujuan korporat |
| `POST` | `/api/v1/approvals/categories` | Buat kategori persetujuan baru |
| `GET` | `/api/v1/approvals/requests` | Daftar permohonan persetujuan (filter: `category_id`, `stage`, `user_id`) |
| `POST` | `/api/v1/approvals/requests` | Ajukan draf permohonan persetujuan baru |
| `GET` | `/api/v1/approvals/requests/:id` | Detail permohonan persetujuan & matriks approver berjenjang |
| `POST` | `/api/v1/approvals/requests/:id/submit` | 🚀 Ajukan permohonan untuk ditinjau approver Level 1 |
| `POST` | `/api/v1/approvals/requests/:id/approve` | ✅ Setujui permohonan pada level aktif (otomatis eskalasi ke level berikutnya) |
| `POST` | `/api/v1/approvals/requests/:id/refuse` | ❌ Tolak permohonan dengan alasan resmi penolakan |
| `POST` | `/api/v1/approvals/requests/:id/cancel` | Batalkan permohonan |
| `GET` | `/api/v1/approvals/stats` | KPI metrik permohonan: Total, Pending, Approved, Refused |
| `POST` | `/api/v1/attachments/upload` | 📎 Unggah file/foto struk ke Cloudflare R2 / Local Disk (`ir.attachment`) |
| `GET` | `/api/v1/attachments` | Daftar berkas lampiran (filter: `res_model`, `res_id`, `ocr_status`, `search`) |
| `GET` | `/api/v1/attachments/:id` | Metadata berkas lampiran |
| `GET` | `/api/v1/attachments/:id/download` | Streaming download / pratinjau file |
| `DELETE` | `/api/v1/attachments/:id` | Hapus file dari penyimpanan cloud & database |
| `POST` | `/api/v1/attachments/:id/ocr` | ✨ Jalankan Cloudflare AI Vision OCR pada berkas struk/faktur |
| `POST` | `/api/v1/attachments/:id/create-expense` | ⚡ 1-Click konversi struk terpindai menjadi pengajuan klaim biaya (`hr.expense`) |
| `GET` | `/api/v1/attachments/stats` | KPI metrik penyimpanan R2: Total Dokumen, Ukuran (MB), Struk Terpindai |
| `POST` | `/api/v1/attendance/check-in` | 📍 Check-in presensi (dukungan GPS geofencing & Cloudflare header IP/Ray) |
| `POST` | `/api/v1/attendance/check-out` | ⏱️ Check-out presensi (otomatis kalkulasi jam kerja & lembur) |
| `GET` | `/api/v1/attendance/active/:employee_id` | Cek status sesi presensi aktif karyawan |
| `GET` | `/api/v1/attendance` | Daftar log presensi (filter: tanggal, status, anomali, departemen) |
| `GET` | `/api/v1/attendance/:id` | Detail catatan presensi |
| `DELETE` | `/api/v1/attendance/:id` | Hapus catatan presensi (hanya untuk log draft/belum disinkron) |
| `GET` | `/api/v1/attendance/shifts` | Daftar master shift kerja (Reguler, Malam/Pabrik, WFH) |
| `POST` | `/api/v1/attendance/shifts` | Tambah master shift kerja baru |
| `GET` | `/api/v1/attendance/stats` | KPI kehadiran: Total hadir, aktif, terlambat, jam lembur, anomali |
| `POST` | `/api/v1/attendance/ai-anomaly-check` | 🤖 AI Audit Anomali Edge, Geofence fraud & Risiko Burnout Karyawan |
| `POST` | `/api/v1/attendance/sync-payroll` | ⚡ 1-Click sinkronisasi lembur & penalti telat ke slip gaji (`hr.payslip`) |
| `GET` | `/api/v1/recruitment/openings` | Daftar lowongan kerja / job openings (filter: `department_id`, `state`) |
| `POST` | `/api/v1/recruitment/openings` | Buat lowongan kerja baru |
| `GET` | `/api/v1/recruitment/stages` | Pipeline tahapan rekrutmen kandidat |
| `GET` | `/api/v1/recruitment/applicants` | Database pelamar (filter: `job_opening_id`, `stage_id`, `state`, `min_ai_score`, `search`) |
| `POST` | `/api/v1/recruitment/applicants` | Pendaftaran pelamar baru dengan resume CV |
| `GET` | `/api/v1/recruitment/applicants/:id` | Detail pelamar & hasil evaluasi AI |
| `PUT` | `/api/v1/recruitment/applicants/:id/stage` | Pindahkan tahapan seleksi pelamar |
| `POST` | `/api/v1/recruitment/applicants/:id/ai-screening` | ✨ Cloudflare Workers AI Resume/CV Screening, Skor Kecocokan & Panduan Interview |
| `POST` | `/api/v1/recruitment/applicants/:id/hire` | ⚡ 1-Click Hire: Buat karyawan aktif di `hr.employee` & tutup lowongan |
| `POST` | `/api/v1/recruitment/applicants/:id/refuse` | Tolak pelamar dengan alasan resmi & audit trail |
| `GET` | `/api/v1/recruitment/stats` | KPI rekrutmen: Total loker, pelamar, in-progress, hired, avg AI score |
| `GET` | `/api/v1/stock-inventory/inventories` | Daftar dokumen Stock Opname (filter: `state`, `location_id`, `search`) |
| `POST` | `/api/v1/stock-inventory/inventories` | Buat draf dokumen Stock Opname baru |
| `GET` | `/api/v1/stock-inventory/inventories/:id` | Detail Stock Opname & daftar baris produk |
| `POST` | `/api/v1/stock-inventory/inventories/:id/start` | ▶️ Mulai opname: Snapshot saldo quant buku ke lembar hitung fisik |
| `PUT` | `/api/v1/stock-inventory/inventories/:id/lines/:line_id` | Input/update hasil hitung fisik & alasan selisih per baris produk |
| `POST` | `/api/v1/stock-inventory/inventories/:id/lines` | Tambah item produk tak terduga yang ditemukan saat inspeksi fisik |
| `POST` | `/api/v1/stock-inventory/inventories/:id/ai-audit` | ✨ Cloudflare Workers AI Loss Prevention Auditor: deteksi anomali shrinkage |
| `POST` | `/api/v1/stock-inventory/inventories/:id/validate` | ✅ Validasi opname: Penyesuaian saldo quant & posting entri jurnal Buku Besar GL |
| `POST` | `/api/v1/stock-inventory/inventories/:id/cancel` | Batalkan dokumen stock opname |
| `GET` | `/api/v1/stock-inventory/stats` | KPI opname: Total, Draf, In Progress, Done, Kerugian (Rp), Surplus (Rp) |
| `GET` | `/api/v1/payroll/structures` | Daftar Struktur Penggajian & Aturan Gaji Aktif (`IDN_STD`) |
| `POST` | `/api/v1/payroll/structures` | Tambah struktur penggajian baru |
| `POST` | `/api/v1/payroll/structures/:id/rules` | Tambah aturan komponen gaji (sequence, kategori, akun GL) |
| `GET` | `/api/v1/payroll/runs` | Daftar Batch Penggajian Masal (filter: status, periode) |
| `POST` | `/api/v1/payroll/runs` | Buat Batch Penggajian Masal baru |
| `POST` | `/api/v1/payroll/runs/:id/generate` | ⚡ 1-Click Mass Generate: Buat & hitung slip seluruh karyawan aktif |
| `GET` | `/api/v1/payroll/payslips` | Database Slip Gaji Karyawan (filter: `state`, `period`, `employee_id`, `search`) |
| `POST` | `/api/v1/payroll/payslips` | Buat draf slip gaji individual karyawan |
| `GET` | `/api/v1/payroll/payslips/:id` | Detail slip gaji, rincian aturan kompensasi & integrasi jurnal GL |
| `POST` | `/api/v1/payroll/payslips/:id/compute` | 🔄 Hitung ulang komponen gaji, sinkronisasi jam lembur & cuti unpaid |
| `POST` | `/api/v1/payroll/payslips/:id/ai-audit` | ✨ Cloudflare Workers AI Tax & Labor Compliance Auditor (@cf/meta/llama-3.1-8b-instruct) |
| `POST` | `/api/v1/payroll/payslips/:id/confirm` | ⚖️ Setujui slip gaji & posting Entri Jurnal Berimbang ke GL (MISC) |
| `POST` | `/api/v1/payroll/payslips/:id/pay` | 💳 Catat pelunasan transfer bank & posting Jurnal Bank (BNK) |
| `POST` | `/api/v1/payroll/payslips/:id/cancel` | Batalkan slip gaji |
| `GET` | `/api/v1/payroll/stats` | KPI Eksekutif Payroll: Total Slips, Paid, Bruto, Potongan, Net, Jam Lembur |
| `GET` | `/api/v1/budget/analytic-accounts` | Daftar Akun Analitik / Cost Centers (`account_analytic_accounts`) |
| `POST` | `/api/v1/budget/analytic-accounts` | Tambah Akun Analitik / Cost Center baru |
| `GET` | `/api/v1/budget/budgetary-positions` | Daftar Pos Anggaran Belanja terpetakan akun GL (`account_budget_posts`) |
| `POST` | `/api/v1/budget/budgetary-positions` | Tambah Pos Anggaran Belanja baru beserta relasi akun GL |
| `GET` | `/api/v1/budget/budgets` | Daftar Dokumen Rencana Anggaran Biaya RKAB (filter: status, search) |
| `POST` | `/api/v1/budget/budgets` | Buat Dokumen Anggaran RKAB baru dengan rincian baris pos biaya |
| `GET` | `/api/v1/budget/budgets/:id` | Detail Dokumen Anggaran, rincian baris & analisis varians riil |
| `POST` | `/api/v1/budget/budgets/:id/compute` | 🔄 Hitung varians & sinkronisasi mutasi realisasi belanja dari entri Buku Besar GL |
| `POST` | `/api/v1/budget/budgets/:id/confirm` | Ajukan konfirmasi dokumen rencana anggaran |
| `POST` | `/api/v1/budget/budgets/:id/validate` | ✅ Validasi & aktifkan dokumen anggaran untuk memantau pengeluaran |
| `POST` | `/api/v1/budget/budgets/:id/done` | Tutup dokumen anggaran (Done) |
| `POST` | `/api/v1/budget/budgets/:id/cancel` | Batalkan dokumen anggaran |
| `POST` | `/api/v1/budget/budgets/:id/reset-draft` | Kembalikan status anggaran ke draf |
| `POST` | `/api/v1/budget/budgets/:id/ai-audit` | ✨ Cloudflare Workers AI Budget Variance & Overspend Early-Warning Auditor |
| `GET` | `/api/v1/budget/stats` | KPI Eksekutif Anggaran: Total, Validated, Plafon, Realisasi GL, Pos Overbudget, AI Risk |
| `POST` | `/api/v1/barcode/scan` | ⚡ Universal Barcode Scanner: deteksi produk, lot, lokasi rak, atau picking sesuai konteks POS/Opname |
| `GET` | `/api/v1/barcode/lookup/:code` | Quick barcode lookup GET |
| `GET` | `/api/v1/barcode/label/:type/:id` | 🖨️ Generator Label Vektor Pure Go SVG Code 128 (product, lot, location) siap cetak thermal 58/80mm |
| `GET` | `/api/v1/barcode/lots` | Daftar nomor lot & serial number (`stock_production_lots`) |
| `POST` | `/api/v1/barcode/lots` | Daftarkan nomor lot / serial baru |
| `GET` | `/api/v1/barcode/lots/:id` | Detail spesifik nomor lot & serial |
| `GET` | `/api/v1/barcode/stats` | KPI Eksekutif Barcode: Produk berbarcode, Total lot, Lot aktif berstok, Lokasi berbarcode |
| `GET` | `/api/v1/purchase-requisitions` | Daftar Purchase Requisition & Blanket Orders (filter: state, type, search) |
| `POST` | `/api/v1/purchase-requisitions` | Daftarkan Purchase Requisition atau Blanket Agreement baru dengan rincian item |
| `GET` | `/api/v1/purchase-requisitions/:id` | Detail lengkap Requisition, baris kuota, relasi PO, dan riwayat aktivitas |
| `PUT` | `/api/v1/purchase-requisitions/:id/confirm` | Otorisasi persetujuan pengadaan oleh manajer (draft -> confirmed) |
| `PUT` | `/api/v1/purchase-requisitions/:id/open-tender` | Buka lelang tender untuk penawaran rekanan (confirmed -> in_progress) |
| `POST` | `/api/v1/purchase-requisitions/:id/generate-rfq` | 📢 Terbitkan RFQ penawaran harga vendor untuk tender kompetitif |
| `POST` | `/api/v1/purchase-requisitions/:id/select-bid` | 🏆 Pilih pemenang lelang tender, auto-confirm PO pemenang & auto-cancel RFQ pesaing |
| `POST` | `/api/v1/purchase-requisitions/:id/release-blanket` | 📜 Rilis parsial/penuh Purchase Order dari kontrak Blanket Order |
| `PUT` | `/api/v1/purchase-requisitions/:id/cancel` | Batalkan dokumen pengadaan |
| `PUT` | `/api/v1/purchase-requisitions/:id/draft` | Kembalikan status pengadaan yang batal ke draft |
| `GET` | `/api/v1/purchase-requisitions/stats` | KPI Eksekutif Pengadaan: Total Requisitions, Tender, Blanket Order, Done, Total Nilai Pagu |
| `GET` | `/api/v1/currencies` | Daftar master mata uang internasional (res.currency) beserta kurs spot terakhir |
| `POST` | `/api/v1/currencies` | Daftarkan master mata uang baru (ISO 4217, symbol, desimal, pembulatan, is_base) |
| `GET` | `/api/v1/currencies/:id` | Detail master mata uang dan riwayat kurs |
| `PUT` | `/api/v1/currencies/:id` | Perbarui master mata uang |
| `POST` | `/api/v1/currencies/convert` | 🔄 Mesin konversi valas instan lintas mata uang pada tanggal efektif tertentu |
| `GET` | `/api/v1/currencies/stats` | KPI Eksekutif Valas: Total mata uang, total kurs tercatat, revaluasi selesai, YTD gain/loss, spot USD & EUR |
| `GET` | `/api/v1/currencies/:id/rates` | Riwayat kurs mata uang (res.currency.rates) terurut tanggal terbaru |
| `POST` | `/api/v1/currencies/:id/rates` | 📈 Catat kurs spot baru (Direct Rate IDR/Unit & auto-compute Inverse Rate Enterprise ERP) |
| `POST` | `/api/v1/currencies/realized-gain-loss` | Hitung selisih kurs terealisasi saat pelunasan faktur/tagihan valas |
| `GET` | `/api/v1/currencies/revaluations` | Riwayat revaluasi kurs bulanan PSAK 10 (account.multicurrency.revaluation) |
| `POST` | `/api/v1/currencies/revaluations` | ⚡ Hitung draf revaluasi kurs bulanan untuk open AR & AP |
| `GET` | `/api/v1/currencies/revaluations/:id` | Rincian dokumen revaluasi beserta rincian faktur per baris dan audit chatter |
| `PUT` | `/api/v1/currencies/revaluations/:id/post` | ⚡ Posting entri jurnal penyesuaian seimbang ke General Ledger |
| `PUT` | `/api/v1/currencies/revaluations/:id/cancel` | Batalkan dokumen revaluasi kurs |
| `GET` | `/api/v1/currencies/:id/ai-risk-audit` | ✨ Treasury AI Copilot: Net Exposure valas, simulasi stress test shock ±2%, ±5%, ±10% & rekomendasi hedging |
| `GET` | `/api/v1/tax-efaktur/stats` | KPI Eksekutif Pajak: PPN Keluaran, PPN Masukan, Net Kurang/Lebih Bayar, sisa jatah NSFP |
| `GET` | `/api/v1/tax-efaktur/nsfp` | Daftar rentang alokasi Nomor Seri Faktur Pajak DJP (`account_tax_nsfps`) |
| `POST` | `/api/v1/tax-efaktur/nsfp` | Daftarkan alokasi jatah NSFP baru (Prefix, Tahun, Range 8-digit nomor awal-akhir) |
| `GET` | `/api/v1/tax-efaktur/invoices` | Daftar seluruh Faktur Pajak Keluaran & Masukan terfilter masa/tahun/status |
| `GET` | `/api/v1/tax-efaktur/invoices/:id` | Detail lengkap faktur pajak, item BKP/JKP, verifikasi QR DJP & audit chatter |
| `POST` | `/api/v1/tax-efaktur/invoices/out` | 📝 Terbitkan e-Faktur Pajak Keluaran dari Customer Invoice posted (auto-decrement NSFP) |
| `POST` | `/api/v1/tax-efaktur/invoices/in` | 📝 Catat e-Faktur Pajak Masukan dari Vendor Bill posted untuk kredit PPN |
| `PUT` | `/api/v1/tax-efaktur/invoices/:id/state` | Perbarui status faktur (generated $\rightarrow$ reported $\rightarrow$ cancelled) |
| `GET` | `/api/v1/tax-efaktur/spt-summary` | Rekapitulasi SPT Masa PPN 1111 (Formulir Induk & AB) per masa pajak |
| `GET` | `/api/v1/tax-efaktur/export-csv` | 📥 Unduh file CSV e-Faktur format resmi DJP (Header FK, LT, dan detail OF) |
| `GET` | `/api/v1/tax-efaktur/ai-audit` | ✨ Cloudflare Workers AI Tax Compliance Copilot: Audit NPWP/NIK, tanggal & rekomendasi |
| `GET` | `/api/v1/ai/status` | Status engine AI (Cloudflare Workers AI vs Local Heuristic Engine) |
| `POST` | `/api/v1/ai/receipt-ocr` | 🤖 AI Vision Struk / Faktur OCR & ekstraksi JSON terstruktur |
| `POST` | `/api/v1/ai/ticket-triage` | 🤖 AI Customer Support Triage, sentimen pelanggan & draf balasan SLA |
| `POST` | `/api/v1/ai/quality-rca` | 🤖 AI Quality Control 5-Whys Root Cause Analysis & CAPA Advisor |
| `POST` | `/api/v1/ai/expense-categorize` | 🤖 AI Auto-kategori pengeluaran & pemetaan akun Buku Besar GL |
| `POST` | `/api/v1/ai/approval-risk` | 🤖 AI Evaluasi Risiko Otorisasi & Kepatuhan Kebijakan Pengeluaran |
| `POST` | `/api/v1/ai/ask-erp` | 🤖 AI Executive Natural Language Query dengan live database context |

---

## ☁️ Integrasi & Deployment ke Cloudflare

Sistem ERP ini siap di-deploy dan dihubungkan ke ekosistem **Cloudflare**:

### 1. Cloudflare Workers AI (LLM Edge Inference)
Backend Go terintegrasi langsung dengan API Cloudflare Workers AI (`@cf/meta/llama-3.1-8b-instruct`). Untuk mengaktifkannya, set environment variables berikut:
```bash
export CLOUDFLARE_ACCOUNT_ID="your-cf-account-id"
export CLOUDFLARE_API_TOKEN="your-cf-api-token"
export CLOUDFLARE_AI_MODEL="@cf/meta/llama-3.1-8b-instruct"
```
*Catatan: Jika variabel di atas tidak diisi, server otomatis beralih ke **Smart Local Heuristic Engine** tanpa memutus koneksi atau menyebabkan error.*

### 2. Cloudflare Worker Edge Proxy
Folder `cloudflare/` menyertakan template Worker yang dapat di-deploy dengan Wrangler:
```bash
cd cloudflare
npm install wrangler --save-dev
npx wrangler deploy
```

### 3. Cloudflare Tunnel (Zero-Trust Edge Access)
Untuk mengekspos backend Go yang berjalan di server internal/on-premise ke jaringan publik Cloudflare tanpa membuka port firewall masuk (*inbound ports*):
```bash
cloudflared tunnel --config cloudflare/cloudflared-config.yml run go-erp-tunnel
```


---

## 🌐 Komunitas & Proyek Open Source

GoERP didistribusikan secara bebas dan terbuka di bawah lisensi [Apache License 2.0](LICENSE). Kami percaya bahwa perangkat lunak ERP tingkat perusahaan harus transparan, bebas dari biaya lisensi per-pengguna yang memberatkan, dan dapat dikustomisasi secara leluasa oleh pengembang di seluruh dunia.

### Cara Berpartisipasi:
- 💬 **Diskusi & Q&A Komunitas**: Bergabunglah dalam [GitHub Discussions](https://github.com/antonset1104/erp/discussions) untuk berbagi pengalaman implementasi, mengajukan pertanyaan teknis, dan berdiskusi dengan sesama pengembang.
- 🐛 **Pelaporan Masalah**: Temukan bug atau anomali transaksi? Laporkan melalui [GitHub Issues](https://github.com/antonset1104/erp/issues/new/choose) menggunakan form laporan terstruktur kami.
- 🚀 **Pengajuan Kode**: Baca panduan [CONTRIBUTING.md](CONTRIBUTING.md) untuk mempelajari alur kerja branch, standar penulisan tes, dan pengajuan Pull Request.
- 🛡️ **Pelaporan Kerentanan**: Laporkan potensi isu keamanan secara privat sesuai prosedur di [SECURITY.md](SECURITY.md).

---

## ⚙️ Konfigurasi Repositori GitHub untuk Maintainer

Bagi maintainer dan pemilik repositori yang melakukan *mirroring* atau *forking*, disarankan memastikan konfigurasi GitHub berikut aktif:
1. **Branch Protection pada `main`**: Aktifkan *Require a pull request before merging*, *Require status checks to pass* (`Test (Go 1.22.x on ubuntu-latest)` dan `windows-latest`), dan *Require linear history*.
2. **Issue Forms & Templates**: Repositori telah dilengkapi template deklaratif di `.github/ISSUE_TEMPLATE/` (Bug Report, Feature Request, dan Config redirection ke Discussions).
3. **Automated CI/CD**: Workflow terkonfigurasi di `.github/workflows/ci.yml` untuk build, vet, dan test matrix otomatis pada setiap commit/PR.
4. **Dependabot Automated Updates**: File `.github/dependabot.yml` aktif memantau pembaruan modul Go dan GitHub Actions secara berkala.
5. **Multi-Platform Automated Releases**: Menerbitkan rilis biner multi-arsitektur (`linux/amd64`, `linux/arm64`, `windows/amd64`) dan checksum SHA256 otomatis saat membuat git tag `v*` (`.github/workflows/release.yml`).
