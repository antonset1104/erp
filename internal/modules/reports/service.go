package reports

import (
	"time"

	"gorm.io/gorm"
)

type Service interface {
	GetCatalog() *ReportCatalogResponse
	RenderDocument(module string, id uint) (string, error)
}

type service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) Service {
	return &service{db: db}
}

func (s *service) GetCatalog() *ReportCatalogResponse {
	catalog := []ReportTemplateItem{
		// 1. Sales & Commercial
		{
			ModuleCode:      "sale",
			ModuleName:      "Sales & Penjualan",
			DocumentType:    "Sales Order & Penawaran",
			TitleIndonesian: "Surat Pesanan Penjualan (SO)",
			Icon:            "fas fa-file-invoice-dollar",
			PaperFormat:     "A4 Portrait",
			Endpoint:        "/api/v1/reports/print/sale/:id",
			Description:     "Surat pesanan penjualan resmi, penawaran harga, dan konfirmasi pemesanan pelanggan B2B.",
		},
		{
			ModuleCode:      "crm",
			ModuleName:      "CRM & Pipeline",
			DocumentType:    "Lead Opportunity Sheet",
			TitleIndonesian: "Lembar Peluang Penjualan & Prospek",
			Icon:            "fas fa-handshake",
			PaperFormat:     "A4 Portrait",
			Endpoint:        "/api/v1/reports/print/crm/:id",
			Description:     "Ringkasan profil prospek, riwayat interaksi, nilai potensial, dan tahapan pipeline.",
		},
		{
			ModuleCode:      "subscription",
			ModuleName:      "Subscription & SaaS Recurring",
			DocumentType:    "Billing Agreement",
			TitleIndonesian: "Kontrak Layanan Berlangganan",
			Icon:            "fas fa-sync-alt",
			PaperFormat:     "A4 Portrait",
			Endpoint:        "/api/v1/reports/print/subscription/:id",
			Description:     "Lembar kontrak langganan SaaS, jadwal penagihan berkala, dan metrik recurring.",
		},
		{
			ModuleCode:      "pos",
			ModuleName:      "Point of Sale (POS)",
			DocumentType:    "Cashier Thermal Receipt",
			TitleIndonesian: "Struk Kasir Penjualan Retail",
			Icon:            "fas fa-cash-register",
			PaperFormat:     "Thermal 80mm",
			Endpoint:        "/api/v1/reports/print/pos/:id",
			Description:     "Struk kasir belanja format thermal 80mm/58mm dengan rincian pembayaran tunai/QRIS.",
		},

		// 2. Purchasing & Procurement
		{
			ModuleCode:      "purchase",
			ModuleName:      "Purchasing & Pengadaan",
			DocumentType:    "Purchase Order (PO)",
			TitleIndonesian: "Surat Pesanan Pembelian Resmi",
			Icon:            "fas fa-shopping-cart",
			PaperFormat:     "A4 Portrait",
			Endpoint:        "/api/v1/reports/print/purchase/:id",
			Description:     "Surat pesanan pengadaan resmi ke pemasok lengkap dengan termin dan spesifikasi material.",
		},
		{
			ModuleCode:      "purchase_requisition",
			ModuleName:      "Purchase Requisition (PR)",
			DocumentType:    "Requisition Slip",
			TitleIndonesian: "Formulir Permintaan Pembelian",
			Icon:            "fas fa-clipboard-list",
			PaperFormat:     "A4 Portrait",
			Endpoint:        "/api/v1/reports/print/purchase_requisition/:id",
			Description:     "Lembar permohonan pengadaan barang dari departemen operasional sebelum diterbitkan PO.",
		},

		// 3. Accounting, Tax & Finance
		{
			ModuleCode:      "account",
			ModuleName:      "Akuntansi & Keuangan",
			DocumentType:    "Commercial Invoice / Bill",
			TitleIndonesian: "Faktur Penjualan / Tagihan Vendor",
			Icon:            "fas fa-receipt",
			PaperFormat:     "A4 Portrait",
			Endpoint:        "/api/v1/reports/print/account/:id",
			Description:     "Faktur komersial resmi pelanggan atau tagihan pemasok dengan akun general ledger.",
		},
		{
			ModuleCode:      "tax_efaktur",
			ModuleName:      "Pajak e-Faktur PPN 11%",
			DocumentType:    "Faktur Pajak Standar",
			TitleIndonesian: "Faktur Pajak Elektronik DJP",
			Icon:            "fas fa-file-invoice",
			PaperFormat:     "A4 Portrait",
			Endpoint:        "/api/v1/reports/print/tax_efaktur/:id",
			Description:     "Formulir Faktur Pajak Standar DJP sesuai format Lampiran PER-03/PJ/2022.",
		},
		{
			ModuleCode:      "tax_ebupot",
			ModuleName:      "Pajak e-Bupot 21/26",
			DocumentType:    "Bukti Potong PPh 21 (BP-21)",
			TitleIndonesian: "Bukti Pemotongan PPh Pasal 21",
			Icon:            "fas fa-stamp",
			PaperFormat:     "A4 Portrait",
			Endpoint:        "/api/v1/reports/print/tax_ebupot/:id",
			Description:     "Formulir resmi BP-21 pemotongan PPh 21 skema TER (PP 58/2023 & PMK 168/2023).",
		},
		{
			ModuleCode:      "bank_reconciliation",
			ModuleName:      "Rekonsiliasi Bank",
			DocumentType:    "Bank Reconciliation Statement",
			TitleIndonesian: "Laporan Rekonsiliasi Bank Bulanan",
			Icon:            "fas fa-university",
			PaperFormat:     "A4 Portrait",
			Endpoint:        "/api/v1/reports/print/bank_reconciliation/:id",
			Description:     "Lembar pencocokan saldo buku kas GL dengan mutasi rekening koran bank.",
		},
		{
			ModuleCode:      "budget",
			ModuleName:      "Anggaran Perusahaan",
			DocumentType:    "Budget Variance Report",
			TitleIndonesian: "Laporan Realisasi & Varian Anggaran",
			Icon:            "fas fa-chart-pie",
			PaperFormat:     "A4 Portrait",
			Endpoint:        "/api/v1/reports/print/budget/:id",
			Description:     "Laporan perbandingan plafon anggaran rencana vs realisasi aktual per pos biaya.",
		},
		{
			ModuleCode:      "treasury",
			ModuleName:      "Treasury & Cash Flow",
			DocumentType:    "Cash Flow Forecast Sheet",
			TitleIndonesian: "Laporan Proyeksi Arus Kas Harian",
			Icon:            "fas fa-coins",
			PaperFormat:     "A4 Portrait",
			Endpoint:        "/api/v1/reports/print/treasury/:id",
			Description:     "Proyeksi arus kas masuk, keluar, dan posisi likuiditas bank multi-horizon.",
		},
		{
			ModuleCode:      "intercompany",
			ModuleName:      "Konsolidasi Multi-Company",
			DocumentType:    "IFRS 10 Financials",
			TitleIndonesian: "Laporan Keuangan Konsolidasian",
			Icon:            "fas fa-sitemap",
			PaperFormat:     "A4 Portrait",
			Endpoint:        "/api/v1/reports/print/intercompany/:id",
			Description:     "Laporan laba rugi dan neraca konsolidasi grup dengan jurnal eliminasi timbal balik.",
		},

		// 4. Warehouse, Inventory & Logistics
		{
			ModuleCode:      "stock",
			ModuleName:      "Inventory & Pergudangan",
			DocumentType:    "Delivery Order / Receipt",
			TitleIndonesian: "Surat Jalan / Bukti Penerimaan",
			Icon:            "fas fa-truck-loading",
			PaperFormat:     "A4 Portrait",
			Endpoint:        "/api/v1/reports/print/stock/:id",
			Description:     "Surat jalan pengiriman fisik barang atau bukti penerimaan gudang penerima.",
		},
		{
			ModuleCode:      "wms",
			ModuleName:      "Enterprise WMS & Bins",
			DocumentType:    "Wave Picking Task List",
			TitleIndonesian: "Lembar Kerja Pengambilan Batch WMS",
			Icon:            "fas fa-boxes",
			PaperFormat:     "A4 Portrait",
			Endpoint:        "/api/v1/reports/print/wms/:id",
			Description:     "Daftar petik barang batch wave picking berdasarkan koordinat rak 3D terpendek.",
		},
		{
			ModuleCode:      "stock_inventory",
			ModuleName:      "Stock Opname",
			DocumentType:    "Physical Count Sheet",
			TitleIndonesian: "Berita Acara Stock Opname",
			Icon:            "fas fa-tasks",
			PaperFormat:     "A4 Portrait",
			Endpoint:        "/api/v1/reports/print/stock_inventory/:id",
			Description:     "Lembar penghitungan fisik persediaan dan rekonsiliasi selisih barang gudang.",
		},
		{
			ModuleCode:      "landed_cost",
			ModuleName:      "Landed Costs & Biaya Impor",
			DocumentType:    "Cost Allocation Sheet",
			TitleIndonesian: "Lembar Alokasi Biaya Masuk & Freight",
			Icon:            "fas fa-ship",
			PaperFormat:     "A4 Portrait",
			Endpoint:        "/api/v1/reports/print/landed_cost/:id",
			Description:     "Rincian alokasi biaya pengapalan, asuransi, dan bea masuk ke HPP persediaan.",
		},
		{
			ModuleCode:      "traceability",
			ModuleName:      "Lot Traceability & FEFO",
			DocumentType:    "Product Lot Pedigree",
			TitleIndonesian: "Sertifikat Silsilah & Penelusuran Lot",
			Icon:            "fas fa-barcode",
			PaperFormat:     "A4 Portrait",
			Endpoint:        "/api/v1/reports/print/traceability/:id",
			Description:     "Riwayat pelacakan hulu-ke-hilir batch produksi dari supplier hingga konsumen.",
		},

		// 5. Manufacturing & Production
		{
			ModuleCode:      "mrp",
			ModuleName:      "Manufacturing (MRP I)",
			DocumentType:    "Production Order (SPK)",
			TitleIndonesian: "Surat Perintah Kerja Manufaktur",
			Icon:            "fas fa-industry",
			PaperFormat:     "A4 Portrait",
			Endpoint:        "/api/v1/reports/print/mrp/:id",
			Description:     "Surat perintah perakitan produk jadi lengkap dengan daftar konsumsi material BOM.",
		},
		{
			ModuleCode:      "mrp_advanced",
			ModuleName:      "Advanced MRP II & Mesin",
			DocumentType:    "Work Center Job Card",
			TitleIndonesian: "Lembar Dispatch Stasiun Kerja Mesin",
			Icon:            "fas fa-cogs",
			PaperFormat:     "A4 Portrait",
			Endpoint:        "/api/v1/reports/print/mrp_advanced/:id",
			Description:     "Perintah kerja mesin terperinci dengan tarif jam mesin, OEE, dan tahapan routing.",
		},
		{
			ModuleCode:      "quality",
			ModuleName:      "Quality Control (QC)",
			DocumentType:    "Certificate of Analysis (CoA)",
			TitleIndonesian: "Sertifikat Hasil Uji Mutu Produk",
			Icon:            "fas fa-microscope",
			PaperFormat:     "A4 Portrait",
			Endpoint:        "/api/v1/reports/print/quality/:id",
			Description:     "Laporan inspeksi parameter mutu, toleransi presisi, dan lembar kelulusan QC.",
		},

		// 6. HR & Human Capital
		{
			ModuleCode:      "payroll",
			ModuleName:      "Payroll & Penggajian",
			DocumentType:    "Employee Payslip Voucher",
			TitleIndonesian: "Slip Gaji Karyawan Resmi",
			Icon:            "fas fa-money-check-alt",
			PaperFormat:     "A4 Portrait",
			Endpoint:        "/api/v1/reports/print/payroll/:id",
			Description:     "Slip gaji resmi dengan rincian take home pay, tunjangan, BPJS, dan PPh 21 TER.",
		},
		{
			ModuleCode:      "leave",
			ModuleName:      "Cuti & Izin Kerja",
			DocumentType:    "Leave Approval Form",
			TitleIndonesian: "Formulir Persetujuan Cuti Karyawan",
			Icon:            "fas fa-calendar-check",
			PaperFormat:     "A4 Portrait",
			Endpoint:        "/api/v1/reports/print/leave/:id",
			Description:     "Surat permohonan cuti resmi pegawai dengan pengesahan atasan dan HRD.",
		},
		{
			ModuleCode:      "expense",
			ModuleName:      "Expense & Reimbursement",
			DocumentType:    "Expense Claim Voucher",
			TitleIndonesian: "Bukti Pengeluaran Kas & Reimburse",
			Icon:            "fas fa-money-bill-wave",
			PaperFormat:     "A4 Portrait",
			Endpoint:        "/api/v1/reports/print/expense/:id",
			Description:     "Bukti pertanggungjawaban biaya perjalanan dinas dan klaim pengeluaran kerja.",
		},

		// 7. Asset & Operations
		{
			ModuleCode:      "asset",
			ModuleName:      "Aktiva Tetap (Fixed Assets)",
			DocumentType:    "Fixed Asset Register Card",
			TitleIndonesian: "Kartu Aktiva Tetap & Penyusutan",
			Icon:            "fas fa-building",
			PaperFormat:     "A4 Portrait",
			Endpoint:        "/api/v1/reports/print/asset/:id",
			Description:     "Kartu inventaris aset tetap beserta jadwal penyusutan komersial dan fiskal.",
		},
		{
			ModuleCode:      "maintenance",
			ModuleName:      "Pemeliharaan Mesin",
			DocumentType:    "Maintenance Work Order",
			TitleIndonesian: "Perintah Kerja Servis Mesin",
			Icon:            "fas fa-wrench",
			PaperFormat:     "A4 Portrait",
			Endpoint:        "/api/v1/reports/print/maintenance/:id",
			Description:     "Lembar tugas teknisi servis dan kalibrasi pencegahan kerusakan mesin pabrik.",
		},
		{
			ModuleCode:      "fleet",
			ModuleName:      "Armada Kendaraan (Fleet)",
			DocumentType:    "Vehicle Dispatch Order (SPJ)",
			TitleIndonesian: "Surat Perintah Jalan Kendaraan",
			Icon:            "fas fa-car",
			PaperFormat:     "A4 Portrait",
			Endpoint:        "/api/v1/reports/print/fleet/:id",
			Description:     "Surat perintah operasional pengemudi, riwayat odometer, dan pengiriman barang.",
		},
		{
			ModuleCode:      "helpdesk",
			ModuleName:      "Helpdesk & Layanan Servis",
			DocumentType:    "Service Ticket Job Sheet",
			TitleIndonesian: "Lembar Kerja Tiket Pelayanan",
			Icon:            "fas fa-headset",
			PaperFormat:     "A4 Portrait",
			Endpoint:        "/api/v1/reports/print/helpdesk/:id",
			Description:     "Lembar penanganan keluhan pelanggan dengan prioritas, solusi teknis, dan SLA.",
		},

		// 8. Legal, Customs & Portals
		{
			ModuleCode:      "esignature",
			ModuleName:      "Digital Signature (UU ITE)",
			DocumentType:    "Legal Completion Certificate",
			TitleIndonesian: "Sertifikat Keabsahan Tanda Tangan",
			Icon:            "fas fa-signature",
			PaperFormat:     "A4 Portrait",
			Endpoint:        "/api/v1/reports/print/esignature/:id",
			Description:     "Sertifikat integritas hukum dokumen bertanda tangan elektronik UU ITE No. 1/2024.",
		},
		{
			ModuleCode:      "customs_bonded",
			ModuleName:      "Kawasan Berikat CEISA 4.0",
			DocumentType:    "Pemberitahuan Pabean BC 2.3/3.0",
			TitleIndonesian: "Dokumen Kepabeanan Bea Cukai",
			Icon:            "fas fa-passport",
			PaperFormat:     "A4 Portrait",
			Endpoint:        "/api/v1/reports/print/customs_bonded/:id",
			Description:     "Lembar dokumen pabean resmi fasilitas Kawasan Berikat terhubung IT Inventory DJBC.",
		},
		{
			ModuleCode:      "vendor_portal",
			ModuleName:      "Portal Rekanan Vendor",
			DocumentType:    "Advance Shipping Notice (ASN)",
			TitleIndonesian: "Surat Jalan Pemasok Mandiri (ASN)",
			Icon:            "fas fa-user-tie",
			PaperFormat:     "A4 Portrait",
			Endpoint:        "/api/v1/reports/print/vendor_portal/:id",
			Description:     "Surat jalan pengiriman yang dibuat langsung oleh vendor via Portal Mandiri.",
		},
		{
			ModuleCode:      "customer_portal",
			ModuleName:      "Customer B2B Portal",
			DocumentType:    "RMA Claim Receipt",
			TitleIndonesian: "Surat Tanda Terima Klaim Garansi",
			Icon:            "fas fa-undo-alt",
			PaperFormat:     "A4 Portrait",
			Endpoint:        "/api/v1/reports/print/customer_portal/:id",
			Description:     "Bukti persetujuan klaim garansi dan penggantian unit produk bagi pelanggan B2B.",
		},

		// 9. Integration & Low-Code Studio
		{
			ModuleCode:      "audit_trail",
			ModuleName:      "Audit Trail & CDC",
			DocumentType:    "ISO 27001 Audit Log",
			TitleIndonesian: "Berita Acara Jejak Audit Digital",
			Icon:            "fas fa-shield-alt",
			PaperFormat:     "A4 Portrait",
			Endpoint:        "/api/v1/reports/print/audit_trail/:id",
			Description:     "Sertifikat jejak perubahan data yang tidak dapat diubah (immutable chronological audit).",
		},
		{
			ModuleCode:      "webhook",
			ModuleName:      "Outbound Webhooks",
			DocumentType:    "Webhook Delivery Receipt",
			TitleIndonesian: "Bukti Pengiriman Event Webhook",
			Icon:            "fas fa-satellite-dish",
			PaperFormat:     "A4 Portrait",
			Endpoint:        "/api/v1/reports/print/webhook/:id",
			Description:     "Laporan pengiriman payload event sistem ke server mitra dengan signature HMAC.",
		},
		{
			ModuleCode:      "inbound_webhook",
			ModuleName:      "Inbound Webhooks",
			DocumentType:    "Ingestion Gateway Report",
			TitleIndonesian: "Laporan Ingesti Webhook Masuk",
			Icon:            "fas fa-network-wired",
			PaperFormat:     "A4 Portrait",
			Endpoint:        "/api/v1/reports/print/inbound_webhook/:id",
			Description:     "Log verifikasi tanda tangan kriptografis callback payment gateway & kurir.",
		},
		{
			ModuleCode:      "lowcode_studio",
			ModuleName:      "Universal Low-Code Studio",
			DocumentType:    "Dynamic Entity Record Sheet",
			TitleIndonesian: "Lembar Formulir Entitas Dinamis",
			Icon:            "fas fa-puzzle-piece",
			PaperFormat:     "A4 Portrait",
			Endpoint:        "/api/v1/reports/print/lowcode_studio/:id",
			Description:     "Lembar cetak dokumen untuk model kustom yang dibuat melalui Low-Code Studio.",
		},
	}

	return &ReportCatalogResponse{
		TotalReports: len(catalog),
		GeneratedAt:  time.Now(),
		Catalog:      catalog,
	}
}

func (s *service) RenderDocument(module string, id uint) (string, error) {
	return RenderDocument(s.db, module, id)
}
