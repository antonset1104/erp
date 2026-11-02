# Panduan Pengguna (User Documentation) — Odoo-Go ERP v1.0

## 1. Memulai Penggunaan Aplikasi (Getting Started)

### 1.1 Mengakses Sistem ERP
1. Buka peramban web modern (Google Chrome, Microsoft Edge, atau Mozilla Firefox) dan arahkan ke alamat:
   **`http://localhost:8080`**
2. Sistem dilengkapi fitur **Auto-Login Demo Enterprise** sebagai **Administrator (`admin`)** pada perusahaan **PT Odoo Golang Nusantara** sehingga Anda dapat langsung menjelajahi seluruh 50 modul tanpa hambatan.
3. Jika Anda keluar (*Logout*) melalui tombol **🚪** di pojok kiri bawah, sistem akan menyegarkan sesi autentikasi JWT secara otomatis.

### 1.2 Tata Letak Antarmuka (UI Layout)
Antarmuka Odoo-Go ERP terbagi menjadi 3 area utama:
- **Bilah Navigasi Kiri (*Sidebar Navigation*)**: Berisi daftar lengkap **50 Modul Enterprise** yang dapat digulir (*scrollable*). Klik nama modul mana pun untuk berpindah tampilan secara instan tanpa memuat ulang halaman (*Single Page Application*).
- **Bilah Atas (*Top Header*)**:
  - Menampilkan judul dan deskripsi modul yang sedang aktif.
  - Tombol **📷 Scan Barcode**: Membuka pemindai Barcode/QR universal untuk mencari produk, nomor lot/seri, atau memproses pengiriman gudang.
  - Tombol **✨ Ask ERP AI**: Membuka asisten cerdas *Executive AI Copilot* untuk menanyakan ringkasan bisnis dalam bahasa sehari-hari (Bahasa Indonesia maupun Bahasa Inggris).
- **Area Konten Utama (*Main Content Area*)**: Menampilkan kartu metrik KPI, sub-tab navigasi, tabel data dengan fitur pencarian/filter, tombol ekspor CSV, serta formulir modal interaktif.

---

## 2. Panduan Operasional Modul Utama

### 2.1 Executive Dashboard (`Dashboard`)
- Menampilkan **6 Kartu Metrik Utama**: *Total Sales (Confirmed)*, *Sales Orders*, *Purchase Orders*, *Stock Pickings*, *Invoices & Bills*, dan *Active Products*.
- Gunakan bagian **Quick Actions** untuk langsung membuat *Sales Order*, *Purchase Order*, *Product*, atau *Partner* baru dengan 1 klik.

### 2.2 Penjualan & Penagihan (`Sales Orders` & `Invoicing`)
1. Buka menu **Sales Orders** di bilah kiri, lalu klik tombol **+ Create Sales Order**.
2. Pilih **Customer**, tambahkan satu atau beberapa baris produk beserta kuantitas dan harga satuan, lalu klik **Save Quotation**.
3. Pada baris pesanan berstatus `DRAFT`, klik tombol **Confirm** untuk mengonfirmasi pesanan menjadi `SALE`. Sistem otomatis membuat surat jalan pengiriman (`WH/OUT`) di modul **Inventory**.
4. Untuk mencetak dokumen resmi, klik ikon **🖨️ Print** pada baris Sales Order atau Invoice untuk membuka dokumen A4 bersegel SHA-256 yang siap dicetak ke printer atau disimpan sebagai PDF.

### 2.3 Pembelian & Inventaris Gudang (`Purchase Orders` & `Inventory`)
1. Buka menu **Purchase Orders**, klik **+ Create RFQ**, pilih **Vendor**, masukkan barang yang dibeli, lalu simpan.
2. Klik **Confirm PO** pada daftar pesanan pembelian. Sistem secara otomatis membuat dokumen penerimaan gudang (`WH/IN`).
3. Buka menu **Inventory**, temukan dokumen `WH/IN` atau `WH/OUT` yang berstatus `ASSIGNED` / `DRAFT`, lalu klik **Validate**. Saldo stok fisik pada tab **Stock On-Hand (Quants)** akan langsung bertambah atau berkurang secara akurat.

### 2.4 Manufaktur & MRP II (`Manufacturing` & `MRP II: Work Center & OEE`)
1. Pada menu **Manufacturing**, Anda dapat membuat resep **Bill of Materials (BoM)** baru atau membuat perintah produksi (**+ Manufacturing Order**).
2. Klik **Produce** pada Manufacturing Order untuk mengonsumsi bahan baku dan menghasilkan barang jadi ke gudang.
3. Pada menu **MRP II: Work Center, Routing & OEE**, Anda dapat mengelola stasiun mesin pabrik (*Work Center*), menyusun *Routing*, memulai/menyelesaikan *Work Order* lantai produksi, serta menjalankan **🤖 AI OEE & Bottleneck Analyzer**.

### 2.5 SDM, Presensi, Cuti & Payroll (`HR`, `Attendance`, `Leave` & `Enterprise Payroll`)
1. **HR & Employees**: Tambah departemen dan karyawan baru beserta informasi gaji pokok.
2. **Presensi (Attendance)**: Gunakan tombol **📍 Clock In (Masuk)** dan **🏁 Clock Out (Pulang)** untuk mencatat kehadiran harian dengan koordinat GPS.
3. **Enterprise Payroll**:
   - Buka menu **Enterprise Payroll**, klik **+ Generate Payslip** atau **⚡ Batch Payroll Run**.
   - Sistem otomatis menghitung komponen Gaji Pokok, Tunjangan, BPJS Kesehatan, BPJS Ketenagakerjaan, dan potongan pajak **PPh 21 TER (PP 58/2023)**.
   - Klik **Confirm & Post to GL** untuk memposting jurnal gaji ke Buku Besar Akuntansi.

### 2.6 Kepatuhan Pajak Indonesia (`Pajak & e-Faktur DJP` & `Pajak e-Bupot Unifikasi`)
1. **e-Faktur PPN 11%**:
   - Tambahkan jatah **NSFP (Nomor Seri Faktur Pajak)** dari DJP melalui tombol **+ Registrasi NSFP**.
   - Buat Faktur Pajak Keluaran/Masukan, setujui (*Approve*), dan unduh format **CSV e-Faktur DJP** atau jalankan **🤖 AI Tax Compliance Auditor**.
2. **e-Bupot Unifikasi**:
   - Buat bukti potong PPh Pasal 21, 23, 26, atau Pasal 4 Ayat (2), terbitkan nomor bukti potong resmi, dan kelola SPT Masa PPh Unifikasi.

### 2.7 Kawasan Berikat & Kepabeanan DJBC (`Kawasan Berikat & DJBC`)
1. Buka menu **Kawasan Berikat & DJBC (CEISA 4.0)**.
2. Klik **+ Buat Dokumen BC** untuk mendaftarkan dokumen pabean (**BC 2.3**, **BC 2.5**, **BC 2.7**, **BC 4.0**, **BC 4.1**, **BC 2.6.1**, **BC 2.6.2**, atau **BC 3.0**).
3. Klik tombol **🛃 Terbitkan SPPB** pada dokumen yang telah diajukan untuk mengesahkan pengeluaran/pemasukan barang dan mencatat mutasi otomatis ke dalam **4 Buku IT Inventory DJBC** (Bahan Baku, Barang Jadi, Mesin/Alat, dan Scrap/Limbah).
4. Gunakan tombol **🤖 AI DJBC Compliance Audit** untuk memverifikasi kesesuaian stok fisik dengan saldo kepabeanan.

---

## 3. Panduan Fitur Platform Lanjutan (Enterprise Platform Tools)

### 3.1 Universal Low-Code Studio (`Low-Code Studio`)
Memungkinkan Anda membuat modul/tabel baru tanpa menulis kode pemrograman:
1. Buka menu **Low-Code Studio** di bagian bawah bilah navigasi.
2. **Membuat Model Secara Manual**:
   - Klik **➕ Buat Model Kustom**, isi Nama Model (misal: *Inspeksi Keselamatan Tambang*), *Technical Name* (`x_mine_safety`), pilih ikon dan kategori menu, lalu simpan.
   - Pada tabel katalog model, klik tombol **🧩 Kolom** untuk menambahkan kolom kustom baru (Teks, Angka, Tanggal, Pilihan Dropdown, Boolean, atau JSON).
3. **Merancang Model Otomatis dengan AI**:
   - Klik **🤖 AI Schema Copilot**, ketik deskripsi kebutuhan bisnis Anda dalam Bahasa Indonesia, klik **🚀 Rancang Skema dengan AI**, lalu klik **✅ Terapkan Skema ke Database ERP**.
4. **Mengisi & Melihat Data**:
   - Klik tombol **📊 Buka Data** pada model mana pun (atau buka sub-tab **Dynamic Data Explorer**) untuk menambah, melihat, dan menghapus baris data secara dinamis.
5. **Ekstensi Entitas Standar (Core Entity Extensions)**:
   - Buka sub-tab **🧩 Ekstensi Entitas Standar (Core)** untuk menambahkan atribut JSON kustom (`x_...`) pada data standar seperti `res.partner`, `product.product`, `sale.order`, atau `purchase.order`.

### 3.2 BPMN 2.0 Dynamic Workflow Engine (`Workflow Engine`)
Mengatur alur persetujuan dokumen berjenjang secara visual:
1. Buka menu **Workflow Engine**.
2. Pada sub-tab **📋 Definisi Alur Kerja & Visual Canvas**, klik **👁️ Lihat Diagram** pada salah satu alur kerja untuk menampilkan diagram *State Machine* BPMN 2.0 beserta aturan kondisi *Exclusive Gateway* (`IF amount >= 50000000`).
3. Klik **🤖 AI Workflow Designer** untuk merancang alur persetujuan baru secara otomatis menggunakan AI, atau klik **+ Buat Alur Kerja** untuk mendefinisikan *nodes* dan *transitions* sendiri.
4. Klik **🚀 Jalankan Instansi** untuk memulai proses persetujuan atas suatu dokumen (misal: `purchase.order` #1 dengan nilai `amount: 125000000`).
5. Pantau dan eksekusi persetujuan (**📤 Submit**, **✅ Approve**, **❌ Reject**) melalui sub-tab **📥 Kotak Masuk Tugas Persetujuan** atau lihat kronologi audit di **⚡ Instansi Berjalan & Audit Trail**.

### 3.3 Universal Integration Hub & Open RPC Gateway (`Integration Hub & API`)
Mengelola integrasi pihak ketiga dan akses API eksternal:
1. **Katalog Konektor Pihak Ketiga**: Uji koneksi langsung (**🔌 Test Koneksi**), sinkronisasi stok ke Shopee/Tokopedia (**🔄 Sync Stok**), atau uji pengiriman notifikasi WhatsApp (**💬 Test WA**).
2. **Manajemen API Key M2M**: Klik **🔑 Buat API Key M2M** untuk menghasilkan kunci rahasia (`ak_live_...`) lengkap dengan pengaturan *Scopes* dan *Rate Limit*.
3. **Universal Open RPC Playground**: Buka sub-tab **🧪 Universal Open RPC Playground**, pilih operasi (`search_read`, `create`, `write`, `execute_action`), pilih target model ERP, sesuaikan payload JSON, dan klik **🚀 Kirim Request RPC** untuk melihat respons JSON langsung dari server.
4. **Batch Data ETL Jobs**: Ekspor atau sinkronisasikan ribuan baris data master dalam format JSON/CSV melalui sub-tab **📦 Batch Data ETL Jobs & Telemetry Logs**.

---

## 4. Tips & Pemecahan Masalah Umum (FAQ / Troubleshooting)

- **Bagaimana cara mengekspor data tabel ke Microsoft Excel?**
  Hampir seluruh modul dilengkapi tombol **📥 Export CSV** di bagian atas tabel. File CSV yang diunduh dapat langsung dibuka di Microsoft Excel atau Google Sheets.
- **Apakah fitur AI tetap berfungsi tanpa koneksi internet atau tanpa akun Cloudflare?**
  **Ya.** Seluruh 20+ fitur AI Copilot di dalam Odoo-Go ERP dilengkapi mesin *Smart Local Heuristic Fallback* yang memberikan analisis akuntansi, pajak, kepabeanan, OEE, serta perancangan skema Low-Code dan BPMN secara instan meskipun variabel lingkungan Cloudflare AI belum diisi.
- **Bagaimana cara mencetak Faktur, PO, atau Slip Gaji ke PDF?**
  Klik tombol **🖨️ Print** pada dokumen terkait, lalu pada jendela pratinjau dokumen cetak yang terbuka, klik tombol **Cetak Dokumen** dan pilih tujuan **Save as PDF** pada dialog cetak browser Anda.


---

## 7. Panduan Fitur Produktivitas Baru (Continuous Improvement)

### 7.1 Menggunakan Command Palette & Pencarian Global (`Ctrl+K`)
1. Tekan pintasan keyboard **`Ctrl+K`** (atau klik tombol **🔍 Cari Global Ctrl+K** di bagian atas layar).
2. Ketik nama modul (misal: `pajak`, `gudang`, `payroll`, `bpmn`), nomor dokumen (`SO00001`, `PO00001`, `INV/`), atau nama produk/pelanggan.
3. Klik salah satu hasil pencarian untuk langsung membuka modul terkait.

### 7.2 Menyaring 50 Modul di Sidebar (`/`)
1. Tekan tombol **`/`** pada keyboard atau klik kotak **🔍 Cari dari 50 modul...** di bagian atas menu kiri (*sidebar*).
2. Ketik kata kunci modul untuk menyaring menu secara instan. Klik tombol **`✕`** untuk menampilkan kembali seluruh 50 modul.

### 7.3 Menjalankan Simulasi Siklus Bisnis Otomatis (1-Klik)
1. Buka tab **📊 Dashboard**.
2. Pada kartu **Enterprise Command Center**, klik tombol **🚀 Simulasi Siklus Bisnis Otomatis (1-Klik)**.
3. Sistem akan secara otomatis membuat & mengonfirmasi Purchase Order, menerima barang ke gudang (`WH/IN`), membuat & mengonfirmasi Sales Order, mengirim barang (`WH/OUT`), serta mem-posting Faktur Penjualan (`INV`) ke Buku Besar, lalu menampilkan laporan ringkasan 5 langkah.

### 7.4 Pusat Notifikasi (`🔔`) & Pengaturan Tema (`☀️ / 🌙`)
- Klik tombol **🔔 Notifikasi** di pojok kanan atas untuk melihat daftar tugas persetujuan, dokumen *draft*, atau alur kerja BPMN yang menunggu tindak lanjut.
- Klik tombol **☀️ Light / 🌙 Dark** di pojok kanan atas untuk mengganti tema tampilan sesuai kenyamanan mata Anda.
