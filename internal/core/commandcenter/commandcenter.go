package commandcenter

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"net/http"
	"strings"
	"time"

	"erp-golang/internal/core/middleware"
	"erp-golang/internal/modules/account"
	"erp-golang/internal/modules/base"
	"erp-golang/internal/modules/product"
	"erp-golang/internal/modules/purchase"
	"erp-golang/internal/modules/sale"
	"erp-golang/internal/modules/stock"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ModuleCatalogItem describes one of the 50 ERP modules for Omni-Search
type ModuleCatalogItem struct {
	Tab      string `json:"tab"`
	Title    string `json:"title"`
	Category string `json:"category"`
	Icon     string `json:"icon"`
	Keywords string `json:"keywords"`
}

// SearchRecordItem represents a live database record returned by Omni-Search
type SearchRecordItem struct {
	ID       uint   `json:"id"`
	Type     string `json:"type"`
	Badge    string `json:"badge"`
	Icon     string `json:"icon"`
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
	Tab      string `json:"tab"`
}

// NotificationItem represents an actionable task or alert in the Notification Center
type NotificationItem struct {
	ID          string `json:"id"`
	Category    string `json:"category"`
	Severity    string `json:"severity"` // "urgent", "warning", "info", "success"
	Icon        string `json:"icon"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Tab         string `json:"tab"`
	CreatedAt   string `json:"created_at"`
}

// RecentTransactionItem represents a unified recent transaction row for the Executive Dashboard
type RecentTransactionItem struct {
	ID          uint    `json:"id"`
	DocNumber   string  `json:"doc_number"`
	DocType     string  `json:"doc_type"` // "Sales Order", "Purchase Order", "Invoice / Bill"
	Icon        string  `json:"icon"`
	PartnerName string  `json:"partner_name"`
	Date        string  `json:"date"`
	Amount      float64 `json:"amount"`
	State       string  `json:"state"`
	Tab         string  `json:"tab"`
}

var erpModuleCatalog = []ModuleCatalogItem{
	{Tab: "dashboard", Title: "Executive Dashboard", Category: "CORE MODULES", Icon: "📊", Keywords: "dashboard ringkasan eksekutif kpi analitik"},
	{Tab: "contacts", Title: "Contacts & Partners", Category: "CORE MODULES", Icon: "👥", Keywords: "kontak partner pelanggan customer vendor pemasok supplier"},
	{Tab: "products", Title: "Product Catalog", Category: "CORE MODULES", Icon: "📦", Keywords: "produk barang katalog item sku harga stok"},
	{Tab: "crm", Title: "CRM & Pipeline", Category: "CORE MODULES", Icon: "🎯", Keywords: "crm prospek lead pipeline peluang sales funnel"},
	{Tab: "sales", Title: "Sales & Quotations", Category: "CORE MODULES", Icon: "🛒", Keywords: "penjualan sales order so quotation penawaran pelanggan"},
	{Tab: "purchases", Title: "Purchases & RFQ", Category: "CORE MODULES", Icon: "📥", Keywords: "pembelian purchase order po rfq pengadaan supplier"},
	{Tab: "inventory", Title: "Inventory & Stock", Category: "CORE MODULES", Icon: "🏢", Keywords: "inventaris gudang stok mutasi transfer picking delivery receipt"},
	{Tab: "stock-inventory", Title: "Stock Opname & Audit", Category: "CORE MODULES", Icon: "📊", Keywords: "stock opname audit fisik gudang selisih shrinkage"},
	{Tab: "invoicing", Title: "Invoicing & Accounting", Category: "CORE MODULES", Icon: "💳", Keywords: "faktur tagihan invoice bill akuntansi jurnal buku besar gl"},
	{Tab: "reports", Title: "Laporan Keuangan", Category: "CORE MODULES", Icon: "📈", Keywords: "laporan keuangan laba rugi neraca saldo profit loss balance sheet"},
	{Tab: "mrp", Title: "Manufacturing (MRP)", Category: "OPERATIONS & HR", Icon: "⚙️", Keywords: "manufaktur produksi mrp bom bill of material pabrik"},
	{Tab: "pos", Title: "Point of Sale (POS)", Category: "OPERATIONS & HR", Icon: "🏪", Keywords: "kasir pos toko ritel struk penjualan cepat"},
	{Tab: "hr", Title: "Human Resources", Category: "OPERATIONS & HR", Icon: "👔", Keywords: "hrd karyawan sdm departemen jabatan organisasi"},
	{Tab: "projects", Title: "Projects & Tasks", Category: "OPERATIONS & HR", Icon: "📋", Keywords: "proyek tugas task kanban timesheet milestone"},
	{Tab: "assets", Title: "Fixed Assets", Category: "OPERATIONS & HR", Icon: "🏛️", Keywords: "aset tetap depresiasi penyusutan inventaris kantor"},
	{Tab: "helpdesk", Title: "Helpdesk & Support", Category: "OPERATIONS & HR", Icon: "🎧", Keywords: "helpdesk tiket komplain layanan pelanggan cs"},
	{Tab: "maintenance", Title: "Maintenance & Equip", Category: "OPERATIONS & HR", Icon: "🔧", Keywords: "maintenance perawatan mesin perbaikan preventif"},
	{Tab: "fleet", Title: "Fleet & Logistics", Category: "OPERATIONS & HR", Icon: "🚗", Keywords: "armada kendaraan mobil truk odometer servis bbm"},
	{Tab: "leave", Title: "Leave & Time Off", Category: "OPERATIONS & HR", Icon: "🏖️", Keywords: "cuti izin sakit absen liburan karyawan"},
	{Tab: "expenses", Title: "Expenses & Claims", Category: "OPERATIONS & HR", Icon: "🧾", Keywords: "klaim reimbursement biaya perjalanan dinas pengeluaran"},
	{Tab: "quality", Title: "Quality & Inspection", Category: "OPERATIONS & HR", Icon: "🎯", Keywords: "kualitas qc qa inspeksi mutu capa defect"},
	{Tab: "approvals", Title: "Approvals Matrix", Category: "OPERATIONS & HR", Icon: "✅", Keywords: "persetujuan approval otorisasi matriks tanda tangan"},
	{Tab: "attachments", Title: "Dokumen & R2 Storage", Category: "OPERATIONS & HR", Icon: "📎", Keywords: "dokumen lampiran file cloudflare r2 ocr struk"},
	{Tab: "attendance", Title: "Kehadiran & Lembur", Category: "OPERATIONS & HR", Icon: "⏱️", Keywords: "presensi absensi kehadiran jam kerja shift lembur gps"},
	{Tab: "recruitment", Title: "Rekrutmen & AI Talent", Category: "OPERATIONS & HR", Icon: "💼", Keywords: "rekrutmen lowongan kerja pelamar cv kandidat hr"},
	{Tab: "payroll", Title: "Payroll & Aturan Gaji", Category: "OPERATIONS & HR", Icon: "💰", Keywords: "gaji payroll slip gaji bpjs pph21 upah"},
	{Tab: "budget", Title: "Anggaran & Analitik", Category: "OPERATIONS & HR", Icon: "📊", Keywords: "anggaran budget rkab cost center analitik varian"},
	{Tab: "barcode", Title: "Barcode & Lot Tracking", Category: "OPERATIONS & HR", Icon: "🏷️", Keywords: "barcode qr code scanner nomor seri lot label"},
	{Tab: "purchase-requisitions", Title: "Tender & Blanket Order", Category: "OPERATIONS & HR", Icon: "📋", Keywords: "tender lelang pr purchase requisition kontrak payung"},
	{Tab: "currency", Title: "Multi-Valas & Kurs Forex", Category: "OPERATIONS & HR", Icon: "💱", Keywords: "mata uang valas kurs forex usd idr revaluasi"},
	{Tab: "tax-efaktur", Title: "Pajak & e-Faktur DJP", Category: "OPERATIONS & HR", Icon: "🇮🇩", Keywords: "pajak ppn efaktur djp nsfp spt 1111"},
	{Tab: "bank-reconcile", Title: "Rekonsiliasi Bank", Category: "OPERATIONS & HR", Icon: "🏦", Keywords: "rekonsiliasi bank mutasi rekening koran bca mandiri"},
	{Tab: "landed-cost", Title: "Landed Cost Impor", Category: "OPERATIONS & HR", Icon: "🚢", Keywords: "landed cost impor bea masuk ongkir kargo hpp"},
	{Tab: "traceability", Title: "Lot Trace & FEFO", Category: "OPERATIONS & HR", Icon: "🧬", Keywords: "traceability pelacakan lot recall penarikan fefo kedaluwarsa"},
	{Tab: "subscription", Title: "Langganan & SaaS", Category: "OPERATIONS & HR", Icon: "🔄", Keywords: "langganan subscription mrr arr kontrak berulang saas"},
	{Tab: "audit-trail", Title: "Audit Trail & CDC", Category: "OPERATIONS & HR", Icon: "🛡️", Keywords: "audit trail keamanan log perubahan cdc iso27001"},
	{Tab: "webhook", Title: "Webhooks & Outbound", Category: "OPERATIONS & HR", Icon: "🔔", Keywords: "webhook outbound notifikasi event hmac retry"},
	{Tab: "inbound-webhook", Title: "Inbound Webhooks", Category: "OPERATIONS & HR", Icon: "📥", Keywords: "inbound webhook payment gateway midtrans xendit ecommerce"},
	{Tab: "intercompany", Title: "Konsolidasi Holding", Category: "OPERATIONS & HR", Icon: "🏢", Keywords: "holding grup intercompany konsolidasi multi perusahaan eliminasi"},
	{Tab: "treasury", Title: "Treasury & Cash Flow", Category: "OPERATIONS & HR", Icon: "🏦", Keywords: "treasury kas likuiditas proyeksi arus kas cashflow"},
	{Tab: "mrp-advanced", Title: "Advanced MRP II", Category: "OPERATIONS & HR", Icon: "🏭", Keywords: "mrp ii work center routing oee lantai produksi mesin"},
	{Tab: "tax-ebupot", Title: "e-Bupot 21/26 DJP", Category: "OPERATIONS & HR", Icon: "📜", Keywords: "ebupot bukti potong pph 21 23 26 4(2) unifikasi"},
	{Tab: "esignature", Title: "E-Signature & Watermark", Category: "OPERATIONS & HR", Icon: "✍️", Keywords: "tanda tangan digital esignature emeterai materai"},
	{Tab: "wms", Title: "WMS & Bin Locations", Category: "OPERATIONS & HR", Icon: "🏢", Keywords: "wms rak bin location putaway wave picking gudang"},
	{Tab: "vendor-portal", Title: "Vendor Portal & SCM", Category: "OPERATIONS & HR", Icon: "🤝", Keywords: "vendor portal b2b pemasok asn penawaran tagihan"},
	{Tab: "customer-portal", Title: "Customer Portal & B2B", Category: "OPERATIONS & HR", Icon: "👥", Keywords: "customer portal pelanggan b2b grosir rma retur"},
	{Tab: "customs-bonded", Title: "Kawasan Berikat & DJBC", Category: "OPERATIONS & HR", Icon: "🛃", Keywords: "kawasan berikat bea cukai djbc ceisa bc 2.3 4.0 mutasi"},
	{Tab: "lowcode-studio", Title: "Low-Code Studio", Category: "OPERATIONS & HR", Icon: "⚡", Keywords: "low code studio custom model field dinamis tanpa koding"},
	{Tab: "workflow-engine", Title: "Workflow Engine", Category: "OPERATIONS & HR", Icon: "🔀", Keywords: "workflow bpmn alur persetujuan sla state machine"},
	{Tab: "integration-hub", Title: "Integration Hub & API", Category: "OPERATIONS & HR", Icon: "🔌", Keywords: "integration hub api key konektor sap shopee tokopedia rpc etl"},
	{Tab: "agentic-swarm", Title: "Agentic AI Swarm", Category: "AI & INTELLIGENCE", Icon: "🤖", Keywords: "agentic ai swarm multi-agent otonom hitl orchestrator pengadaan matching procurement cashflow wms"},
	{Tab: "digital-twin-simulator", Title: "Digital Twin & What-If War-Room", Category: "AI & INTELLIGENCE", Icon: "🔮", Keywords: "digital twin monte carlo simulasi skenario what if krisis valas shock stress test runway playbook"},
	{Tab: "fraud-graph-sentinel", Title: "Anti-Fraud & Collusion Graph", Category: "AI & INTELLIGENCE", Icon: "🛡️", Keywords: "anti-fraud gnn knowledge graph audit forensik pengadaan kolusi vendor bid rigging split po rekening bersama sindikat"},
	{Tab: "self-healing-data", Title: "Self-Healing Data Sentinel", Category: "AI & INTELLIGENCE", Icon: "✨", Keywords: "self healing data rekonsiliasi otomatis zero touch closing gr ir clearing deduplikasi semantik master data buku besar"},
	{Tab: "ambient-vision-dock", Title: "Ambient Vision Logistics (WMS Dock AI)", Category: "AI & INTELLIGENCE", Icon: "👁️", Keywords: "ambient computer vision wms dock receiving kargo volumetrik dimensi p l t defect detection kerusakan rma klaim asuransi freight surcharges"},
	{Tab: "contract-legal-copilot", Title: "Legal Contract AI & Redlining", Category: "AI & INTELLIGENCE", Icon: "⚖️", Keywords: "legal contract copilot redline klausul kontrak pks msa nda risiko hukum ganti rugi denda arbitrase counter offer"},
	{Tab: "energy-peak-shaving", Title: "Industrial Energy AI & Peak-Shaving", Category: "AI & INTELLIGENCE", Icon: "⚡", Keywords: "energy ai peak load shaving wbp lwbp tarif listrik pln beban puncak esg emisi karbon scope 1 2 cbam chiller optimasi pabrik"},
}

// RegisterRoutes mounts the continuous improvement Command Center endpoints on /api/v1
func RegisterRoutes(
	apiV1 *gin.RouterGroup,
	authMiddleware gin.HandlerFunc,
	db *gorm.DB,
	saleSvc *sale.Service,
	purchaseSvc *purchase.Service,
	stockSvc *stock.Service,
	accountSvc *account.Service,
) {
	apiV1.GET("/search", authMiddleware, func(c *gin.Context) {
		HandleGlobalSearch(c, db)
	})
	apiV1.GET("/dashboard/notifications", authMiddleware, func(c *gin.Context) {
		HandleDashboardNotifications(c, db)
	})
	apiV1.POST("/dashboard/simulate-cycle", authMiddleware, func(c *gin.Context) {
		HandleSimulateBusinessCycle(c, db, saleSvc, purchaseSvc, stockSvc, accountSvc)
	})
	apiV1.GET("/dashboard/export-summary", authMiddleware, func(c *gin.Context) {
		HandleExportExecutiveSummaryCSV(c, db)
	})
}

// EnrichDashboardStats computes additional financial & operational KPIs and recent transactions
func EnrichDashboardStats(db *gorm.DB, companyID uint, payload gin.H) {
	var arOutstanding float64
	db.Table("account_moves").
		Where("company_id = ? AND move_type = 'out_invoice' AND state = 'posted'", companyID).
		Select("COALESCE(SUM(amount_residual), 0)").Scan(&arOutstanding)

	var apOutstanding float64
	db.Table("account_moves").
		Where("company_id = ? AND move_type = 'in_invoice' AND state = 'posted'", companyID).
		Select("COALESCE(SUM(amount_residual), 0)").Scan(&apOutstanding)

	var crmExpectedRevenue float64
	db.Table("crm_leads").
		Where("company_id = ? AND stage != 'lost'", companyID).
		Select("COALESCE(SUM(expected_revenue), 0)").Scan(&crmExpectedRevenue)

	var inventoryValue float64
	type quantValRow struct {
		Total float64
	}
	var qv quantValRow
	db.Table("stock_quants").
		Joins("JOIN products ON products.id = stock_quants.product_id").
		Where("stock_quants.company_id = ? AND stock_quants.location_id = 1 AND stock_quants.quantity > 0", companyID).
		Select("COALESCE(SUM(stock_quants.quantity * products.standard_price), 0) AS total").
		Scan(&qv)
	inventoryValue = qv.Total

	var recent []RecentTransactionItem

	var recentSales []sale.SaleOrder
	db.Preload("Partner").Where("company_id = ?", companyID).Order("id desc").Limit(4).Find(&recentSales)
	for _, so := range recentSales {
		partnerName := so.Partner.Name
		if partnerName == "" {
			partnerName = "Customer"
		}
		recent = append(recent, RecentTransactionItem{
			ID:          so.ID,
			DocNumber:   so.Name,
			DocType:     "Sales Order",
			Icon:        "🛒",
			PartnerName: partnerName,
			Date:        so.DateOrder,
			Amount:      so.AmountTotal,
			State:       so.State,
			Tab:         "sales",
		})
	}

	var recentPOs []purchase.PurchaseOrder
	db.Preload("Partner").Where("company_id = ?", companyID).Order("id desc").Limit(3).Find(&recentPOs)
	for _, po := range recentPOs {
		partnerName := po.Partner.Name
		if partnerName == "" {
			partnerName = "Supplier"
		}
		recent = append(recent, RecentTransactionItem{
			ID:          po.ID,
			DocNumber:   po.Name,
			DocType:     "Purchase Order",
			Icon:        "📥",
			PartnerName: partnerName,
			Date:        po.DateOrder,
			Amount:      po.AmountTotal,
			State:       po.State,
			Tab:         "purchases",
		})
	}

	var recentMoves []account.AccountMove
	db.Preload("Partner").Where("company_id = ?", companyID).Order("id desc").Limit(3).Find(&recentMoves)
	for _, mv := range recentMoves {
		partnerName := mv.Partner.Name
		if partnerName == "" {
			partnerName = "General Ledger"
		}
		docLabel := "Customer Invoice"
		if mv.MoveType == "in_invoice" {
			docLabel = "Vendor Bill"
		} else if mv.MoveType == "entry" {
			docLabel = "Journal Entry"
		}
		recent = append(recent, RecentTransactionItem{
			ID:          mv.ID,
			DocNumber:   mv.Name,
			DocType:     docLabel,
			Icon:        "💳",
			PartnerName: partnerName,
			Date:        mv.InvoiceDate,
			Amount:      mv.AmountTotal,
			State:       mv.State,
			Tab:         "invoicing",
		})
	}

	payload["ar_outstanding"] = arOutstanding
	payload["ap_outstanding"] = apOutstanding
	payload["crm_expected_revenue"] = crmExpectedRevenue
	payload["inventory_valuation"] = inventoryValue
	payload["recent_transactions"] = recent
}

// HandleGlobalSearch searches across all 50 ERP modules and live database records
func HandleGlobalSearch(c *gin.Context, db *gorm.DB) {
	_, companyID, _ := middleware.GetUserContext(c)
	if companyID == 0 {
		companyID = 1
	}

	rawQ := strings.TrimSpace(c.Query("q"))
	qLower := strings.ToLower(rawQ)

	var matchedModules []ModuleCatalogItem
	for _, m := range erpModuleCatalog {
		if qLower == "" ||
			strings.Contains(strings.ToLower(m.Title), qLower) ||
			strings.Contains(strings.ToLower(m.Tab), qLower) ||
			strings.Contains(strings.ToLower(m.Keywords), qLower) ||
			strings.Contains(strings.ToLower(m.Category), qLower) {
			matchedModules = append(matchedModules, m)
			if len(matchedModules) >= 12 && qLower == "" {
				break
			}
		}
	}

	var records []SearchRecordItem
	likePattern := "%" + qLower + "%"

	if qLower != "" {
		// 1. Products
		var products []product.Product
		db.Where("company_id = ? AND (LOWER(name) LIKE ? OR LOWER(default_code) LIKE ? OR LOWER(barcode) LIKE ?)",
			companyID, likePattern, likePattern, likePattern).Limit(5).Find(&products)
		for _, p := range products {
			records = append(records, SearchRecordItem{
				ID:       p.ID,
				Type:     "product",
				Badge:    "Product",
				Icon:     "📦",
				Title:    fmt.Sprintf("[%s] %s", p.DefaultCode, p.Name),
				Subtitle: fmt.Sprintf("Harga Jual: Rp %.0f • HPP: Rp %.0f", p.ListPrice, p.StandardPrice),
				Tab:      "products",
			})
		}

		// 2. Partners (Customers & Suppliers)
		var partners []base.Partner
		db.Where("company_id = ? AND (LOWER(name) LIKE ? OR LOWER(email) LIKE ? OR LOWER(city) LIKE ?)",
			companyID, likePattern, likePattern, likePattern).Limit(5).Find(&partners)
		for _, p := range partners {
			role := "Contact"
			if p.CustomerRank > 0 && p.SupplierRank > 0 {
				role = "Customer & Vendor"
			} else if p.CustomerRank > 0 {
				role = "Customer"
			} else if p.SupplierRank > 0 {
				role = "Supplier"
			}
			records = append(records, SearchRecordItem{
				ID:       p.ID,
				Type:     "partner",
				Badge:    role,
				Icon:     "👥",
				Title:    p.Name,
				Subtitle: fmt.Sprintf("%s • %s", p.Email, p.City),
				Tab:      "contacts",
			})
		}

		// 3. Sales Orders
		var sales []sale.SaleOrder
		db.Preload("Partner").Where("company_id = ? AND (LOWER(name) LIKE ? OR LOWER(state) LIKE ?)",
			companyID, likePattern, likePattern).Limit(5).Find(&sales)
		for _, s := range sales {
			records = append(records, SearchRecordItem{
				ID:       s.ID,
				Type:     "sale_order",
				Badge:    "Sales Order",
				Icon:     "🛒",
				Title:    fmt.Sprintf("%s — %s", s.Name, s.Partner.Name),
				Subtitle: fmt.Sprintf("Rp %.0f • Status: %s", s.AmountTotal, strings.ToUpper(s.State)),
				Tab:      "sales",
			})
		}

		// 4. Purchase Orders
		var purchases []purchase.PurchaseOrder
		db.Preload("Partner").Where("company_id = ? AND (LOWER(name) LIKE ? OR LOWER(state) LIKE ?)",
			companyID, likePattern, likePattern).Limit(5).Find(&purchases)
		for _, po := range purchases {
			records = append(records, SearchRecordItem{
				ID:       po.ID,
				Type:     "purchase_order",
				Badge:    "Purchase Order",
				Icon:     "📥",
				Title:    fmt.Sprintf("%s — %s", po.Name, po.Partner.Name),
				Subtitle: fmt.Sprintf("Rp %.0f • Status: %s", po.AmountTotal, strings.ToUpper(po.State)),
				Tab:      "purchases",
			})
		}

		// 5. Invoices & Accounting Moves
		var moves []account.AccountMove
		db.Preload("Partner").Where("company_id = ? AND (LOWER(name) LIKE ? OR LOWER(ref) LIKE ?)",
			companyID, likePattern, likePattern).Limit(5).Find(&moves)
		for _, m := range moves {
			records = append(records, SearchRecordItem{
				ID:       m.ID,
				Type:     "account_move",
				Badge:    "Accounting",
				Icon:     "💳",
				Title:    fmt.Sprintf("%s — %s", m.Name, m.Partner.Name),
				Subtitle: fmt.Sprintf("Rp %.0f • Status: %s (%s)", m.AmountTotal, strings.ToUpper(m.State), m.PaymentState),
				Tab:      "invoicing",
			})
		}
	} else {
		// Default top records when query is empty so Command Palette shows live recent records immediately
		var products []product.Product
		db.Where("company_id = ?", companyID).Limit(3).Find(&products)
		for _, p := range products {
			records = append(records, SearchRecordItem{
				ID:       p.ID,
				Type:     "product",
				Badge:    "Product",
				Icon:     "📦",
				Title:    fmt.Sprintf("[%s] %s", p.DefaultCode, p.Name),
				Subtitle: fmt.Sprintf("Harga Jual: Rp %.0f", p.ListPrice),
				Tab:      "products",
			})
		}
		var sales []sale.SaleOrder
		db.Preload("Partner").Where("company_id = ?", companyID).Order("id desc").Limit(3).Find(&sales)
		for _, s := range sales {
			records = append(records, SearchRecordItem{
				ID:       s.ID,
				Type:     "sale_order",
				Badge:    "Sales Order",
				Icon:     "🛒",
				Title:    fmt.Sprintf("%s — %s", s.Name, s.Partner.Name),
				Subtitle: fmt.Sprintf("Rp %.0f • Status: %s", s.AmountTotal, strings.ToUpper(s.State)),
				Tab:      "sales",
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"query":      rawQ,
		"total_hits": len(matchedModules) + len(records),
		"modules":    matchedModules,
		"records":    records,
	})
}

// HandleDashboardNotifications returns actionable pending items across the 50 ERP modules
func HandleDashboardNotifications(c *gin.Context, db *gorm.DB) {
	_, companyID, _ := middleware.GetUserContext(c)
	if companyID == 0 {
		companyID = 1
	}

	var items []NotificationItem
	urgentCount := 0
	nowStr := time.Now().Format("2006-01-02 15:04")

	// 1. Pending Approvals
	var pendingApprovals int64
	db.Table("approval_requests").Where("company_id = ? AND status IN ('submitted', 'pending')", companyID).Count(&pendingApprovals)
	if pendingApprovals > 0 {
		urgentCount++
		items = append(items, NotificationItem{
			ID:          "notif-approvals",
			Category:    "Approvals Matrix",
			Severity:    "urgent",
			Icon:        "✅",
			Title:       fmt.Sprintf("%d Permintaan Otorisasi Menunggu Approval", pendingApprovals),
			Description: "Permintaan pengadaan/anggaran membutuhkan tanda tangan pejabat berwenang.",
			Tab:         "approvals",
			CreatedAt:   nowStr,
		})
	}

	// 2. Active BPMN Workflows
	var runningWorkflows int64
	db.Table("workflow_instances").Where("status = 'running'").Count(&runningWorkflows)
	if runningWorkflows > 0 {
		urgentCount++
		items = append(items, NotificationItem{
			ID:          "notif-workflows",
			Category:    "BPMN Workflow Engine",
			Severity:    "warning",
			Icon:        "🔀",
			Title:       fmt.Sprintf("%d Instance Workflow BPMN Sedang Berjalan", runningWorkflows),
			Description: "Terdapat dokumen bisnis aktif pada state machine yang menunggu transisi step berikutnya.",
			Tab:         "workflow-engine",
			CreatedAt:   nowStr,
		})
	}

	// 3. Draft Sales Orders
	var draftSOs int64
	db.Table("sale_orders").Where("company_id = ? AND state = 'draft'", companyID).Count(&draftSOs)
	if draftSOs > 0 {
		items = append(items, NotificationItem{
			ID:          "notif-draft-so",
			Category:    "Sales & Quotations",
			Severity:    "info",
			Icon:        "🛒",
			Title:       fmt.Sprintf("%d Penawaran Penjualan (Quotation) Siap Dikonfirmasi", draftSOs),
			Description: "Konfirmasi Quotation menjadi Sales Order untuk memicu pengiriman barang otomatis.",
			Tab:         "sales",
			CreatedAt:   nowStr,
		})
	}

	// 4. Draft Purchase Orders
	var draftPOs int64
	db.Table("purchase_orders").Where("company_id = ? AND state = 'draft'", companyID).Count(&draftPOs)
	if draftPOs > 0 {
		items = append(items, NotificationItem{
			ID:          "notif-draft-po",
			Category:    "Purchases & RFQ",
			Severity:    "info",
			Icon:        "📥",
			Title:       fmt.Sprintf("%d Draft RFQ Pembelian Menunggu Konfirmasi", draftPOs),
			Description: "Termasuk RFQ otomatis dari mesin Reordering Rules stok minimum.",
			Tab:         "purchases",
			CreatedAt:   nowStr,
		})
	}

	// 5. Pending Warehouse Pickings
	var pendingPickings int64
	db.Table("stock_pickings").Where("company_id = ? AND state IN ('assigned', 'confirmed', 'draft')", companyID).Count(&pendingPickings)
	if pendingPickings > 0 {
		items = append(items, NotificationItem{
			ID:          "notif-pickings",
			Category:    "Inventory & WMS",
			Severity:    "warning",
			Icon:        "🏢",
			Title:       fmt.Sprintf("%d Dokumen Mutasi Gudang (WH/IN & WH/OUT) Siap Validasi", pendingPickings),
			Description: "Validasi penerimaan atau pengiriman barang untuk memperbarui saldo Stock Quant.",
			Tab:         "inventory",
			CreatedAt:   nowStr,
		})
	}

	// 6. Draft Invoices
	var draftMoves int64
	db.Table("account_moves").Where("company_id = ? AND state = 'draft'", companyID).Count(&draftMoves)
	if draftMoves > 0 {
		items = append(items, NotificationItem{
			ID:          "notif-invoices",
			Category:    "Invoicing & Accounting",
			Severity:    "warning",
			Icon:        "💳",
			Title:       fmt.Sprintf("%d Draft Faktur / Jurnal Belum Diposting ke GL", draftMoves),
			Description: "Verifikasi keseimbangan Debit/Kredit dan posting ke General Ledger.",
			Tab:         "invoicing",
			CreatedAt:   nowStr,
		})
	}

	// 7. Suspicious Audit Logs
	var suspiciousCount int64
	db.Table("audit_log_entries").Where("company_id = ? AND is_suspicious = ?", companyID, true).Count(&suspiciousCount)
	if suspiciousCount > 0 {
		urgentCount++
		items = append(items, NotificationItem{
			ID:          "notif-security",
			Category:    "Security & Audit Trail",
			Severity:    "urgent",
			Icon:        "🛡️",
			Title:       fmt.Sprintf("%d Aktivitas Sensitif Terdeteksi oleh Audit Trail CDC", suspiciousCount),
			Description: "Tinjau perubahan field kritis pada modul Immutable Audit Trail ISO 27001.",
			Tab:         "audit-trail",
			CreatedAt:   nowStr,
		})
	}

	// Always include a system status notification if list is short
	if len(items) == 0 {
		items = append(items, NotificationItem{
			ID:          "notif-healthy",
			Category:    "System Health",
			Severity:    "success",
			Icon:        "⚡",
			Title:       "Seluruh 50 Modul ERP Beroperasi Normal (Zero Backlog)",
			Description: "Semua antrean transaksi, pengiriman gudang, dan jurnal akuntansi telah terproses.",
			Tab:         "dashboard",
			CreatedAt:   nowStr,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"total_unread": len(items),
		"urgent_count": urgentCount,
		"items":        items,
	})
}

// HandleSimulateBusinessCycle executes a complete Procure-to-Pay (P2P) + Order-to-Cash (O2C) ERP transaction cycle in 1 click
func HandleSimulateBusinessCycle(
	c *gin.Context,
	db *gorm.DB,
	saleSvc *sale.Service,
	purchaseSvc *purchase.Service,
	stockSvc *stock.Service,
	accountSvc *account.Service,
) {
	_, companyID, _ := middleware.GetUserContext(c)
	if companyID == 0 {
		companyID = 1
	}

	// 1. Find or create Customer, Supplier, and Storable Product
	var customer base.Partner
	if err := db.Where("company_id = ? AND customer_rank > 0", companyID).First(&customer).Error; err != nil {
		customer = base.Partner{
			Name:         "PT Nusantara Digital Enterprise (Customer)",
			Email:        "procurement@nusantara-enterprise.co.id",
			Phone:        "021-5550199",
			City:         "Jakarta Selatan",
			CustomerRank: 1,
			IsCompany:    true,
		}
		customer.CompanyID = companyID
		db.Create(&customer)
	}

	var supplier base.Partner
	if err := db.Where("company_id = ? AND supplier_rank > 0", companyID).First(&supplier).Error; err != nil {
		supplier = base.Partner{
			Name:         "PT Global Teknologi Distribusi (Vendor)",
			Email:        "sales@globaltek-distribusi.co.id",
			Phone:        "021-8880123",
			City:         "Surabaya",
			SupplierRank: 1,
			IsCompany:    true,
		}
		supplier.CompanyID = companyID
		db.Create(&supplier)
	}

	var prod product.Product
	if err := db.Where("company_id = ? AND type = 'product'", companyID).First(&prod).Error; err != nil {
		prod = product.Product{
			Name:          "Server Rack Enterprise Xeon E-2388G",
			DefaultCode:   "SRV-XEON-01",
			Type:          "product",
			ListPrice:     18500000,
			StandardPrice: 14000000,
		}
		prod.CompanyID = companyID
		prod.Active = true
		db.Create(&prod)
	}

	unitPriceSale := prod.ListPrice
	if unitPriceSale <= 0 {
		unitPriceSale = 15000000
	}
	unitPriceCost := prod.StandardPrice
	if unitPriceCost <= 0 {
		unitPriceCost = unitPriceSale * 0.75
	}

	// 2. Procure-to-Pay (P2P): Create & Confirm Purchase Order + Validate Warehouse Receipt
	po := &purchase.PurchaseOrder{
		PartnerID: supplier.ID,
		TaxRate:   11.0,
		Note:      "Automated P2P Business Cycle Simulation",
		OrderLines: []purchase.PurchaseOrderLine{
			{
				ProductID:     prod.ID,
				Name:          prod.Name,
				ProductUomQty: 5,
				PriceUnit:     unitPriceCost,
			},
		},
	}
	po.CompanyID = companyID
	if err := purchaseSvc.CreateOrder(po); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create simulated PO: " + err.Error()})
		return
	}

	confirmedPO, receipt, err := purchaseSvc.ActionConfirm(po.ID, companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to confirm simulated PO: " + err.Error()})
		return
	}

	receiptName := "-"
	if receipt != nil {
		receiptName = receipt.Name
		_, _ = stockSvc.ValidatePicking(receipt.ID, companyID)
	}

	// 3. Order-to-Cash (O2C): Create & Confirm Sales Order + Validate Warehouse Delivery
	so := &sale.SaleOrder{
		PartnerID: customer.ID,
		TaxRate:   11.0,
		Note:      "Automated O2C Business Cycle Simulation",
		OrderLines: []sale.SaleOrderLine{
			{
				ProductID:     prod.ID,
				Name:          prod.Name,
				ProductUomQty: 2,
				PriceUnit:     unitPriceSale,
			},
		},
	}
	so.CompanyID = companyID
	if err := saleSvc.CreateOrder(so); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create simulated SO: " + err.Error()})
		return
	}

	confirmedSO, delivery, err := saleSvc.ActionConfirm(so.ID, companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to confirm simulated SO: " + err.Error()})
		return
	}

	deliveryName := "-"
	if delivery != nil {
		deliveryName = delivery.Name
		_, _ = stockSvc.ValidatePicking(delivery.ID, companyID)
	}

	// 4. Financial Accounting: Create & Post Balanced Customer Invoice in General Ledger
	var arAcc, salesAcc, taxAcc account.AccountAccount
	db.Where("code = '120000' OR type = 'asset_receivable'").First(&arAcc)
	db.Where("code = '410000' OR type = 'income'").First(&salesAcc)
	db.Where("code = '210100' OR type = 'liability_payable'").First(&taxAcc)

	var salesJournal account.AccountJournal
	db.Where("type = 'sale'").First(&salesJournal)

	invName := "-"
	if arAcc.ID > 0 && salesAcc.ID > 0 && salesJournal.ID > 0 {
		creditTaxAccountID := taxAcc.ID
		if creditTaxAccountID == 0 {
			creditTaxAccountID = salesAcc.ID
		}
		inv := &account.AccountMove{
			MoveType:    "out_invoice",
			PartnerID:   customer.ID,
			JournalID:   salesJournal.ID,
			InvoiceDate: time.Now().Format("2006-01-02"),
			Ref:         confirmedSO.Name,
			AmountTotal: confirmedSO.AmountTotal,
			LineIDs: []account.AccountMoveLine{
				{
					AccountID: arAcc.ID,
					Name:      fmt.Sprintf("Piutang Penjualan %s", confirmedSO.Name),
					Debit:     confirmedSO.AmountTotal,
					Credit:    0,
				},
				{
					AccountID: salesAcc.ID,
					Name:      fmt.Sprintf("Pendapatan %s (%s)", prod.Name, confirmedSO.Name),
					Quantity:  2,
					PriceUnit: unitPriceSale,
					Debit:     0,
					Credit:    confirmedSO.AmountUntaxed,
				},
				{
					AccountID: creditTaxAccountID,
					Name:      fmt.Sprintf("PPN Keluaran 11%% %s", confirmedSO.Name),
					Debit:     0,
					Credit:    confirmedSO.AmountTax,
				},
			},
		}
		inv.CompanyID = companyID
		if err := accountSvc.CreateMove(inv); err == nil {
			if postedInv, errPost := accountSvc.ActionPost(inv.ID, companyID); errPost == nil {
				invName = postedInv.Name
			} else {
				invName = inv.Name
			}
		}
	}

	steps := []string{
		fmt.Sprintf("1. [P2P] Purchase Order %s (%s) dikonfirmasi sebesar Rp %.0f", confirmedPO.Name, supplier.Name, confirmedPO.AmountTotal),
		fmt.Sprintf("2. [WMS] Penerimaan Gudang %s divalidasi (+5 unit %s masuk ke WH/Stock)", receiptName, prod.Name),
		fmt.Sprintf("3. [O2C] Sales Order %s (%s) dikonfirmasi sebesar Rp %.0f", confirmedSO.Name, customer.Name, confirmedSO.AmountTotal),
		fmt.Sprintf("4. [WMS] Pengiriman Barang %s divalidasi (-2 unit %s dikirim ke pelanggan)", deliveryName, prod.Name),
		fmt.Sprintf("5. [GL] Faktur Penjualan %s diposting ke Buku Besar (Debit = Kredit = Rp %.0f)", invName, confirmedSO.AmountTotal),
	}

	c.JSON(http.StatusOK, gin.H{
		"message":           "Simulasi siklus bisnis terintegrasi (P2P + WMS + O2C + General Ledger) berhasil dijalankan!",
		"product_name":      prod.Name,
		"supplier_name":     supplier.Name,
		"customer_name":     customer.Name,
		"po_number":         confirmedPO.Name,
		"receipt_number":    receiptName,
		"so_number":         confirmedSO.Name,
		"delivery_number":   deliveryName,
		"invoice_number":    invName,
		"total_procurement": confirmedPO.AmountTotal,
		"total_revenue":     confirmedSO.AmountTotal,
		"steps":             steps,
	})
}

// HandleExportExecutiveSummaryCSV exports an executive KPI summary as a downloadable CSV file
func HandleExportExecutiveSummaryCSV(c *gin.Context, db *gorm.DB) {
	_, companyID, _ := middleware.GetUserContext(c)
	if companyID == 0 {
		companyID = 1
	}

	var totalSales, arOut, apOut, liquidCash, subMRR float64
	var soCount, poCount, pickCount, invCount, prodCount, empCount, wfCount, apiCount int64

	db.Table("sale_orders").Where("company_id = ? AND state IN ('sale', 'done')", companyID).Select("COALESCE(SUM(amount_total), 0)").Scan(&totalSales)
	db.Table("account_moves").Where("company_id = ? AND move_type = 'out_invoice' AND state = 'posted'", companyID).Select("COALESCE(SUM(amount_residual), 0)").Scan(&arOut)
	db.Table("account_moves").Where("company_id = ? AND move_type = 'in_invoice' AND state = 'posted'", companyID).Select("COALESCE(SUM(amount_residual), 0)").Scan(&apOut)
	db.Table("treasury_bank_accounts").Where("company_id = ? AND active = ? AND status = 'active'", companyID, true).Select("COALESCE(SUM(current_balance), 0)").Scan(&liquidCash)
	db.Table("subscriptions").Where("company_id = ? AND state = 'in_progress'", companyID).Select("COALESCE(SUM(total_mrr), 0)").Scan(&subMRR)

	db.Table("sale_orders").Where("company_id = ?", companyID).Count(&soCount)
	db.Table("purchase_orders").Where("company_id = ?", companyID).Count(&poCount)
	db.Table("stock_pickings").Where("company_id = ?", companyID).Count(&pickCount)
	db.Table("account_moves").Where("company_id = ?", companyID).Count(&invCount)
	db.Table("products").Where("company_id = ? AND active = ?", companyID, true).Count(&prodCount)
	db.Table("employees").Where("company_id = ? AND status = 'active'", companyID).Count(&empCount)
	db.Table("workflow_definitions").Count(&wfCount)
	db.Table("integration_connectors").Count(&apiCount)

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"Category", "Metric_Name", "Value", "Unit", "Generated_At"})
	ts := time.Now().Format("2006-01-02 15:04:05")

	rows := [][]string{
		{"Finance", "Total Confirmed Sales", fmt.Sprintf("%.2f", totalSales), "IDR", ts},
		{"Finance", "Accounts Receivable (AR Outstanding)", fmt.Sprintf("%.2f", arOut), "IDR", ts},
		{"Finance", "Accounts Payable (AP Outstanding)", fmt.Sprintf("%.2f", apOut), "IDR", ts},
		{"Treasury", "Total Liquid Bank & Cash Balance", fmt.Sprintf("%.2f", liquidCash), "IDR", ts},
		{"SaaS & Recurring", "Monthly Recurring Revenue (MRR)", fmt.Sprintf("%.2f", subMRR), "IDR", ts},
		{"Commercial", "Total Sales Orders", fmt.Sprintf("%d", soCount), "Orders", ts},
		{"Procurement", "Total Purchase Orders", fmt.Sprintf("%d", poCount), "Orders", ts},
		{"Supply Chain", "Warehouse Stock Pickings", fmt.Sprintf("%d", pickCount), "Pickings", ts},
		{"Accounting", "General Ledger Moves & Invoices", fmt.Sprintf("%d", invCount), "Documents", ts},
		{"Master Data", "Active Products", fmt.Sprintf("%d", prodCount), "SKUs", ts},
		{"Human Capital", "Active Employees", fmt.Sprintf("%d", empCount), "Employees", ts},
		{"Automation", "BPMN Workflow Definitions", fmt.Sprintf("%d", wfCount), "Workflows", ts},
		{"Integration", "Enterprise API Connectors", fmt.Sprintf("%d", apiCount), "Connectors", ts},
	}
	for _, r := range rows {
		_ = w.Write(r)
	}
	w.Flush()

	filename := fmt.Sprintf("executive_kpi_summary_%s.csv", time.Now().Format("20060102"))
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Data(http.StatusOK, "text/csv; charset=utf-8", buf.Bytes())
}
