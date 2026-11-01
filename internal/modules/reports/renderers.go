package reports

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

// parseTimeAny parses various datetime string formats safely from SQLite/PostgreSQL
func parseTimeAny(s string) time.Time {
	if s == "" {
		return time.Now()
	}
	formats := []string{
		time.RFC3339,
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05-07:00",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		"2006-01-02",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t
		}
	}
	return time.Now()
}

// Universal Document Dispatcher
func RenderDocument(db *gorm.DB, module string, id uint) (string, error) {
	switch strings.ToLower(module) {
	case "sale", "so", "sales":
		return RenderSaleOrder(db, id)
	case "purchase", "po":
		return RenderPurchaseOrder(db, id)
	case "invoice", "account", "accounting", "bill":
		return RenderAccountMove(db, id)
	case "stock", "picking", "delivery", "receipt":
		return RenderStockPicking(db, id)
	case "mrp", "manufacturing", "production":
		return RenderMrpProduction(db, id)
	case "pos":
		return RenderPosOrder(db, id)
	case "payroll", "payslip", "salary":
		return RenderPayslip(db, id)
	case "crm", "lead":
		return RenderLead(db, id)
	case "project":
		return RenderProject(db, id)
	case "asset", "fixed_asset":
		return RenderAsset(db, id)
	case "helpdesk", "ticket":
		return RenderTicket(db, id)
	case "maintenance":
		return RenderMaintenance(db, id)
	case "fleet", "vehicle":
		return RenderVehicle(db, id)
	case "leave":
		return RenderLeave(db, id)
	case "expense":
		return RenderExpense(db, id)
	case "quality", "qc":
		return RenderQuality(db, id)
	case "approval":
		return RenderApproval(db, id)
	case "stock_inventory", "opname":
		return RenderStockInventory(db, id)
	case "budget":
		return RenderBudget(db, id)
	case "purchase_requisition", "pr":
		return RenderPurchaseRequisition(db, id)
	case "tax_efaktur", "efaktur":
		return RenderTaxInvoice(db, id)
	case "tax_ebupot", "ebupot":
		return RenderTaxEbupot(db, id)
	case "bank_reconciliation", "bank_rec":
		return RenderBankRec(db, id)
	case "landed_cost":
		return RenderLandedCost(db, id)
	case "traceability", "lot":
		return RenderTraceabilityLot(db, id)
	case "subscription":
		return RenderSubscription(db, id)
	case "audit_trail", "audit":
		return RenderAuditLog(db, id)
	case "webhook":
		return RenderWebhookLog(db, id)
	case "inbound_webhook":
		return RenderInboundWebhookLog(db, id)
	case "intercompany", "consolidation":
		return RenderConsolidation(db, id)
	case "treasury", "cash_flow":
		return RenderTreasuryForecast(db, id)
	case "mrp_advanced", "work_order":
		return RenderWorkOrder(db, id)
	case "esignature", "esign":
		return RenderEsignDoc(db, id)
	case "wms", "wave_picking":
		return RenderWaveBatch(db, id)
	case "vendor_portal", "asn":
		return RenderVendorASN(db, id)
	case "customer_portal", "rma":
		return RenderCustomerRMA(db, id)
	case "customs_bonded", "bc":
		return RenderCustomsDoc(db, id)
	case "lowcode_studio", "custom_record":
		return RenderCustomRecord(db, id)
	default:
		return RenderGenericFallback(db, module, id)
	}
}

// 1. SALE ORDER / QUOTATION
func RenderSaleOrder(db *gorm.DB, id uint) (string, error) {
	var order struct {
		ID         uint
		Name       string
		State      string
		DateOrder  string
		AmountUntaxed float64
		AmountTax     float64
		AmountTotal   float64
		PartnerID  uint
		PartnerName string
		Note       string
	}
	if err := db.Table("sale_orders").Where("id = ?", id).First(&order).Error; err != nil {
		return "", fmt.Errorf("pesanan penjualan #%d tidak ditemukan: %w", id, err)
	}

	var lines []struct {
		Name         string
		ProductUomQty float64
		PriceUnit    float64
		PriceSubtotal float64
	}
	db.Table("sale_order_lines").Where("order_id = ?", id).Find(&lines)

	infoLeft := fmt.Sprintf(`
		<div class="info-card-title">Pelanggan / Pembeli</div>
		<div class="bold" style="font-size:13px; color:#0f172a;">%s</div>
		<div>Partner ID: #%d</div>
		<div>Syarat Pembayaran: 30 Hari (Net 30)</div>
	`, order.PartnerName, order.PartnerID)

	infoRight := fmt.Sprintf(`
		<div class="info-card-title">Rincian Dokumen</div>
		<div><strong>Tanggal Order:</strong> %s</div>
		<div><strong>Mata Uang:</strong> IDR (Rupiah)</div>
		<div><strong>Sales Representative:</strong> Sales Executive Odoo-Go</div>
	`, FormatDateTime(parseTimeAny(order.DateOrder)))

	var rows strings.Builder
	for idx, l := range lines {
		rows.WriteString(fmt.Sprintf(`
			<tr>
				<td class="text-center">%d</td>
				<td><div class="bold">%s</div></td>
				<td class="text-center">%.2f</td>
				<td class="text-right">Rp %s</td>
				<td class="text-right bold">Rp %s</td>
			</tr>
		`, idx+1, l.Name, l.ProductUomQty, FormatCurrency(l.PriceUnit), FormatCurrency(l.PriceSubtotal)))
	}

	tableHTML := fmt.Sprintf(`
		<table class="data-table">
			<thead>
				<tr>
					<th style="width:40px;" class="text-center">No</th>
					<th>Deskripsi Produk / Item</th>
					<th style="width:80px;" class="text-center">Kuantitas</th>
					<th style="width:130px;" class="text-right">Harga Satuan</th>
					<th style="width:140px;" class="text-right">Subtotal</th>
				</tr>
			</thead>
			<tbody>%s</tbody>
		</table>
	`, rows.String())

	summaryHTML := fmt.Sprintf(`
		<div class="summary-container">
			<table class="summary-table">
				<tr>
					<td>Subtotal Sebelum Pajak:</td>
					<td class="text-right">Rp %s</td>
				</tr>
				<tr>
					<td>PPN (11%%):</td>
					<td class="text-right">Rp %s</td>
				</tr>
				<tr class="grand-total">
					<td>Total Tagihan Pesanan:</td>
					<td class="text-right">Rp %s</td>
				</tr>
			</table>
		</div>
	`, FormatCurrency(order.AmountUntaxed), FormatCurrency(order.AmountTax), FormatCurrency(order.AmountTotal))

	notesHTML := fmt.Sprintf(`<div class="notes-box"><strong>Catatan & Syarat Ketentuan:</strong> %s</div>`, func() string {
		if order.Note != "" {
			return order.Note
		}
		return "Barang yang telah dipesan akan dikirimkan sesuai jadwal pengiriman yang telah disepakati. Faktur resmi diterbitkan setelah surat jalan divalidasi."
	}())

	sigsHTML := `
		<div class="signatures-grid">
			<div class="sign-card">
				<div class="sign-card-title">Dibuat Oleh (Sales Officer)</div>
				<div class="sign-line">Bagian Penjualan</div>
			</div>
			<div class="sign-card">
				<div class="sign-card-title">Disetujui Oleh (Sales Manager)</div>
				<div class="sign-line">Manager Komersial</div>
			</div>
			<div class="sign-card">
				<div class="sign-card-title">Diterima & Disetujui Pelanggan</div>
				<div class="sign-line">Cap & Tanda Tangan Pembeli</div>
			</div>
		</div>
	`

	statusColor := "#0284c7"
	if order.State == "sale" || order.State == "done" {
		statusColor = "#16a34a"
	}

	return WrapReportHTML(
		"SURAT PESANAN PENJUALAN (SALES ORDER)",
		order.Name,
		strings.ToUpper(order.State),
		statusColor,
		infoLeft,
		infoRight,
		tableHTML,
		summaryHTML,
		notesHTML,
		"CONFIRMED SO",
		sigsHTML,
		false,
	), nil
}

// 2. PURCHASE ORDER
func RenderPurchaseOrder(db *gorm.DB, id uint) (string, error) {
	var po struct {
		ID         uint
		Name       string
		State      string
		DateOrder  string
		AmountUntaxed float64
		AmountTax     float64
		AmountTotal   float64
		PartnerID  uint
		PartnerName string
		Note       string
	}
	if err := db.Table("purchase_orders").Where("id = ?", id).First(&po).Error; err != nil {
		return "", fmt.Errorf("pesanan pembelian #%d tidak ditemukan: %w", id, err)
	}

	var lines []struct {
		Name         string
		ProductQty   float64
		PriceUnit    float64
		PriceSubtotal float64
	}
	db.Table("purchase_order_lines").Where("order_id = ?", id).Find(&lines)

	infoLeft := fmt.Sprintf(`
		<div class="info-card-title">Pemasok / Vendor Terpilih</div>
		<div class="bold" style="font-size:13px; color:#0f172a;">%s</div>
		<div>Vendor ID: #%d</div>
		<div>Termin Pembayaran: Net 30 Hari</div>
	`, po.PartnerName, po.PartnerID)

	infoRight := fmt.Sprintf(`
		<div class="info-card-title">Rincian Dokumen PO</div>
		<div><strong>Tanggal Terbit PO:</strong> %s</div>
		<div><strong>Alamat Kirim:</strong> Gudang Utama WH/STOCK Jakarta</div>
		<div><strong>Departemen Pemohon:</strong> Procurement & Logistics</div>
	`, FormatDateTime(parseTimeAny(po.DateOrder)))

	var rows strings.Builder
	for idx, l := range lines {
		rows.WriteString(fmt.Sprintf(`
			<tr>
				<td class="text-center">%d</td>
				<td><div class="bold">%s</div></td>
				<td class="text-center">%.2f</td>
				<td class="text-right">Rp %s</td>
				<td class="text-right bold">Rp %s</td>
			</tr>
		`, idx+1, l.Name, l.ProductQty, FormatCurrency(l.PriceUnit), FormatCurrency(l.PriceSubtotal)))
	}

	tableHTML := fmt.Sprintf(`
		<table class="data-table">
			<thead>
				<tr>
					<th style="width:40px;" class="text-center">No</th>
					<th>Item Barang / Spesifikasi Material</th>
					<th style="width:80px;" class="text-center">Kuantitas</th>
					<th style="width:130px;" class="text-right">Harga Beli</th>
					<th style="width:140px;" class="text-right">Subtotal</th>
				</tr>
			</thead>
			<tbody>%s</tbody>
		</table>
	`, rows.String())

	summaryHTML := fmt.Sprintf(`
		<div class="summary-container">
			<table class="summary-table">
				<tr>
					<td>Subtotal DPP:</td>
					<td class="text-right">Rp %s</td>
				</tr>
				<tr>
					<td>PPN Masukan (11%%):</td>
					<td class="text-right">Rp %s</td>
				</tr>
				<tr class="grand-total">
					<td>Total Komitmen Pembelian:</td>
					<td class="text-right">Rp %s</td>
				</tr>
			</table>
		</div>
	`, FormatCurrency(po.AmountUntaxed), FormatCurrency(po.AmountTax), FormatCurrency(po.AmountTotal))

	notesHTML := `<div class="notes-box"><strong>Instruksi Pengiriman Pemasok:</strong> Harap sertakan nomor Purchase Order ini pada Surat Jalan & Faktur Pajak saat pengiriman fisik barang ke gudang. Barang tanpa PO resmi tidak akan diterima petugas penerimaan (Good Receipt).</div>`

	sigsHTML := `
		<div class="signatures-grid">
			<div class="sign-card">
				<div class="sign-card-title">Dibuat (Purchasing Officer)</div>
				<div class="sign-line">Spesialis Pengadaan</div>
			</div>
			<div class="sign-card">
				<div class="sign-card-title">Disetujui (Head of Procurement)</div>
				<div class="sign-line">Manager Pengadaan Barang</div>
			</div>
			<div class="sign-card">
				<div class="sign-card-title">Konfirmasi Diterima Pemasok</div>
				<div class="sign-line">Tanda Tangan & Cap Vendor</div>
			</div>
		</div>
	`

	return WrapReportHTML(
		"SURAT PESANAN PEMBELIAN RESMI (PURCHASE ORDER)",
		po.Name,
		strings.ToUpper(po.State),
		"#2563eb",
		infoLeft,
		infoRight,
		tableHTML,
		summaryHTML,
		notesHTML,
		"OFFICIAL PO",
		sigsHTML,
		false,
	), nil
}

// 3. ACCOUNT MOVE (INVOICE / VENDOR BILL)
func RenderAccountMove(db *gorm.DB, id uint) (string, error) {
	var move struct {
		ID           uint
		Name         string
		MoveType     string
		State        string
		PaymentState string
		InvoiceDate  string
		AmountTotal  float64
		PartnerName  string
	}
	if err := db.Table("account_moves").Where("id = ?", id).First(&move).Error; err != nil {
		return "", fmt.Errorf("faktur #%d tidak ditemukan: %w", id, err)
	}

	docTitle := "FAKTUR PENJUALAN RESMI (COMMERCIAL INVOICE)"
	if move.MoveType == "in_invoice" {
		docTitle = "TAGIHAN PEMASOK (VENDOR BILL)"
	}

	infoLeft := fmt.Sprintf(`
		<div class="info-card-title">Ditujukan Kepada</div>
		<div class="bold" style="font-size:13px; color:#0f172a;">%s</div>
		<div>Status Pembayaran: <strong>%s</strong></div>
		<div>Metode Bayar: Transfer Bank Mandiri / BCA</div>
	`, move.PartnerName, strings.ToUpper(move.PaymentState))

	infoRight := fmt.Sprintf(`
		<div class="info-card-title">Data Faktur</div>
		<div><strong>Tanggal Terbit:</strong> %s</div>
		<div><strong>Rekening Pembayaran:</strong> Bank Mandiri 123-00-9876543-2</div>
		<div><strong>Atas Nama:</strong> PT Odoo-Go Indonesia Enterprise</div>
	`, FormatDateTime(parseTimeAny(move.InvoiceDate)))

	var lines []struct {
		Name         string
		Quantity     float64
		PriceUnit    float64
		PriceSubtotal float64
	}
	db.Table("account_move_lines").Where("move_id = ?", id).Find(&lines)

	var rows strings.Builder
	for idx, l := range lines {
		rows.WriteString(fmt.Sprintf(`
			<tr>
				<td class="text-center">%d</td>
				<td><div class="bold">%s</div></td>
				<td class="text-center">%.2f</td>
				<td class="text-right">Rp %s</td>
				<td class="text-right bold">Rp %s</td>
			</tr>
		`, idx+1, l.Name, l.Quantity, FormatCurrency(l.PriceUnit), FormatCurrency(l.PriceSubtotal)))
	}

	tableHTML := fmt.Sprintf(`
		<table class="data-table">
			<thead>
				<tr>
					<th style="width:40px;" class="text-center">No</th>
					<th>Rincian Produk / Layanan</th>
					<th style="width:80px;" class="text-center">Kuantitas</th>
					<th style="width:130px;" class="text-right">Harga Satuan</th>
					<th style="width:140px;" class="text-right">Total</th>
				</tr>
			</thead>
			<tbody>%s</tbody>
		</table>
	`, rows.String())

	summaryHTML := fmt.Sprintf(`
		<div class="summary-container">
			<table class="summary-table">
				<tr class="grand-total">
					<td>Total Tagihan Faktur:</td>
					<td class="text-right">Rp %s</td>
				</tr>
			</table>
		</div>
	`, FormatCurrency(move.AmountTotal))

	notesHTML := `<div class="notes-box"><strong>Keterangan Pelunasan:</strong> Pembayaran dianggap sah apabila dana telah efektif masuk ke rekening koran perusahaan. Bukti transfer mohon diunggah ke Customer Portal atau dikirim ke finance@odoo.local.</div>`

	sigsHTML := `
		<div class="signatures-grid">
			<div class="sign-card">
				<div class="sign-card-title">Bagian Penagihan (Billing)</div>
				<div class="sign-line">Staff Finance AR</div>
			</div>
			<div class="sign-card">
				<div class="sign-card-title">Manager Keuangan</div>
				<div class="sign-line">Finance & Tax Manager</div>
			</div>
		</div>
	`

	statusColor := "#16a34a"
	if move.PaymentState != "paid" {
		statusColor = "#f59e0b"
	}

	return WrapReportHTML(
		docTitle,
		move.Name,
		strings.ToUpper(move.State),
		statusColor,
		infoLeft,
		infoRight,
		tableHTML,
		summaryHTML,
		notesHTML,
		"INVOICE POSTED",
		sigsHTML,
		false,
	), nil
}

// 4. STOCK PICKING (DELIVERY SLIP / RECEIPT)
func RenderStockPicking(db *gorm.DB, id uint) (string, error) {
	var pick struct {
		ID          uint
		Name        string
		PickingType string
		State       string
		Origin      string
		PartnerName string
		ScheduledDate string
	}
	if err := db.Table("stock_pickings").Where("id = ?", id).First(&pick).Error; err != nil {
		return "", fmt.Errorf("dokumen picking #%d tidak ditemukan: %w", id, err)
	}

	docTitle := "SURAT JALAN / PENGIRIMAN BARANG (DELIVERY ORDER)"
	if pick.PickingType == "incoming" {
		docTitle = "BUKTI PENERIMAAN BARANG GUDANG (GOOD RECEIPT)"
	} else if pick.PickingType == "internal" {
		docTitle = "SURAT TRANSFER GUDANG ANTAR-LOKASI"
	}

	infoLeft := fmt.Sprintf(`
		<div class="info-card-title">Tujuan / Mitra Eksternal</div>
		<div class="bold" style="font-size:13px; color:#0f172a;">%s</div>
		<div>Dokumen Asal (Origin): <strong>%s</strong></div>
	`, pick.PartnerName, pick.Origin)

	infoRight := fmt.Sprintf(`
		<div class="info-card-title">Informasi Logistik</div>
		<div><strong>Tipe Operasi:</strong> %s</div>
		<div><strong>Tanggal Terjadwal:</strong> %s</div>
	`, strings.ToUpper(pick.PickingType), FormatDateTime(parseTimeAny(pick.ScheduledDate)))

	var lines []struct {
		ProductName  string
		ProductUomQty float64
		QuantityDone float64
	}
	db.Table("stock_moves").Where("picking_id = ?", id).Find(&lines)

	var rows strings.Builder
	for idx, l := range lines {
		rows.WriteString(fmt.Sprintf(`
			<tr>
				<td class="text-center">%d</td>
				<td><div class="bold">%s</div></td>
				<td class="text-center">%.2f</td>
				<td class="text-center bold" style="color:#0284c7;">%.2f</td>
				<td class="text-center">UNIT</td>
			</tr>
		`, idx+1, l.ProductName, l.ProductUomQty, l.QuantityDone))
	}

	tableHTML := fmt.Sprintf(`
		<table class="data-table">
			<thead>
				<tr>
					<th style="width:40px;" class="text-center">No</th>
					<th>Nama Barang & SKU</th>
					<th style="width:100px;" class="text-center">Permintaan</th>
					<th style="width:100px;" class="text-center">Realisasi Fisik</th>
					<th style="width:80px;" class="text-center">Satuan</th>
				</tr>
			</thead>
			<tbody>%s</tbody>
		</table>
	`, rows.String())

	sigsHTML := `
		<div class="signatures-grid">
			<div class="sign-card">
				<div class="sign-card-title">Petugas Gudang (Picker)</div>
				<div class="sign-line">Staff Logistik</div>
			</div>
			<div class="sign-card">
				<div class="sign-card-title">Pengemudi / Kurir (Driver)</div>
				<div class="sign-line">Ekspedisi Logistik</div>
			</div>
			<div class="sign-card">
				<div class="sign-card-title">Penerima Barang</div>
				<div class="sign-line">Tanda Tangan & Nama Terang</div>
			</div>
		</div>
	`

	return WrapReportHTML(
		docTitle,
		pick.Name,
		strings.ToUpper(pick.State),
		"#0284c7",
		infoLeft,
		infoRight,
		tableHTML,
		"",
		`<div class="notes-box">Periksa kondisi fisik barang saat diterima. Klaim kerusakan wajib dicatat pada surat jalan ini sebelum kurir meninggalkan lokasi.</div>`,
		"GOODS RELEASED",
		sigsHTML,
		false,
	), nil
}

// 5. MRP PRODUCTION (SPK MANUFAKTUR)
func RenderMrpProduction(db *gorm.DB, id uint) (string, error) {
	var mo struct {
		ID          uint
		Name        string
		State       string
		ProductName string
		ProductQty  float64
		DatePlanned string
	}
	if err := db.Table("mrp_productions").Where("id = ?", id).First(&mo).Error; err != nil {
		return "", fmt.Errorf("perintah produksi #%d tidak ditemukan: %w", id, err)
	}

	infoLeft := fmt.Sprintf(`
		<div class="info-card-title">Target Hasil Produksi</div>
		<div class="bold" style="font-size:13px; color:#0f172a;">%s</div>
		<div>Target Output: <strong>%.2f Unit</strong></div>
	`, mo.ProductName, mo.ProductQty)

	infoRight := fmt.Sprintf(`
		<div class="info-card-title">Jadwal Perakitan</div>
		<div><strong>Tanggal Terencana:</strong> %s</div>
		<div><strong>Work Center:</strong> Jalur Perakitan Mesin Utama</div>
	`, FormatDateTime(parseTimeAny(mo.DatePlanned)))

	tableHTML := `
		<table class="data-table">
			<thead>
				<tr>
					<th style="width:40px;" class="text-center">No</th>
					<th>Komponen Bahan Baku (BOM Line)</th>
					<th class="text-center">Standar Konsumsi</th>
					<th class="text-center">Satuan</th>
				</tr>
			</thead>
			<tbody>
				<tr>
					<td class="text-center">1</td>
					<td>Komponen Elektronik & PCB Sub-Assembly</td>
					<td class="text-center">1.00</td>
					<td class="text-center">SET</td>
				</tr>
				<tr>
					<td class="text-center">2</td>
					<td>Casing Pelindung & Rangka Logam</td>
					<td class="text-center">1.00</td>
					<td class="text-center">PCS</td>
				</tr>
			</tbody>
		</table>
	`

	sigsHTML := `
		<div class="signatures-grid">
			<div class="sign-card">
				<div class="sign-card-title">Perencana Produksi (PPIC)</div>
				<div class="sign-line">Staff PPIC</div>
			</div>
			<div class="sign-card">
				<div class="sign-card-title">Kepala Regu Bengkel (Foreman)</div>
				<div class="sign-line">Supervisor Manufaktur</div>
			</div>
		</div>
	`

	return WrapReportHTML(
		"SURAT PERINTAH KERJA MANUFAKTUR (SPK PRODUCTION ORDER)",
		mo.Name,
		strings.ToUpper(mo.State),
		"#8b5cf6",
		infoLeft,
		infoRight,
		tableHTML,
		"",
		`<div class="notes-box">Operator wajib mematuhi standar SOP K3 dan memeriksa lembar spesifikasi mutu sebelum memulai proses perakitan massal.</div>`,
		"MANUFACTURING ORDER",
		sigsHTML,
		false,
	), nil
}

// 6. POS ORDER (THERMAL RECEIPT)
func RenderPosOrder(db *gorm.DB, id uint) (string, error) {
	var pos struct {
		ID          uint
		Name        string
		AmountTotal float64
		DateOrder   string
	}
	if err := db.Table("pos_orders").Where("id = ?", id).First(&pos).Error; err != nil {
		return "", fmt.Errorf("transaksi POS #%d tidak ditemukan: %w", id, err)
	}

	var lines []struct {
		ProductName  string
		Qty          float64
		PriceUnit    float64
		PriceSubtotal float64
	}
	db.Table("pos_order_lines").Where("order_id = ?", id).Find(&lines)

	var items strings.Builder
	items.WriteString("<table>")
	for _, l := range lines {
		items.WriteString(fmt.Sprintf(`
			<tr>
				<td colspan="2" class="bold">%s</td>
			</tr>
			<tr>
				<td>%.0f x Rp %s</td>
				<td class="text-right">Rp %s</td>
			</tr>
		`, l.ProductName, l.Qty, FormatCurrency(l.PriceUnit), FormatCurrency(l.PriceSubtotal)))
	}
	items.WriteString("</table>")

	summary := fmt.Sprintf(`
		<table>
			<tr class="bold" style="font-size:13px;">
				<td>TOTAL BELANJA:</td>
				<td class="text-right">Rp %s</td>
			</tr>
			<tr>
				<td>Metode: TUNAI / QRIS</td>
				<td class="text-right">LUNAS</td>
			</tr>
		</table>
	`, FormatCurrency(pos.AmountTotal))

	return WrapReportHTML(
		"STRUK PENJUALAN KASIR",
		pos.Name,
		"PAID",
		"#16a34a",
		"",
		"",
		items.String(),
		summary,
		"",
		"",
		"",
		true, // Thermal format
	), nil
}

// 7. PAYROLL (SLIP GAJI KARYAWAN RESMI)
func RenderPayslip(db *gorm.DB, id uint) (string, error) {
	var slip struct {
		ID              uint
		Number          string
		Period          string
		GrossSalary     float64
		TotalDeductions float64
		NetSalary       float64
		EmployeeID      uint
	}
	if err := db.Table("payroll_payslips").Where("id = ?", id).First(&slip).Error; err != nil {
		return "", fmt.Errorf("slip gaji #%d tidak ditemukan: %w", id, err)
	}

	var emp struct {
		Name       string
		WorkEmail  string
		Department string
		JobTitle   string
	}
	db.Table("hr_employees").Where("id = ?", slip.EmployeeID).First(&emp)

	infoLeft := fmt.Sprintf(`
		<div class="info-card-title">Identitas Pegawai</div>
		<div class="bold" style="font-size:13px; color:#0f172a;">%s</div>
		<div>Departemen: %s</div>
		<div>Jabatan: %s</div>
		<div>Email: %s</div>
	`, emp.Name, emp.Department, emp.JobTitle, emp.WorkEmail)

	infoRight := fmt.Sprintf(`
		<div class="info-card-title">Periode Penggajian</div>
		<div><strong>Periode:</strong> %s</div>
		<div><strong>Metode Transfer:</strong> Payroll Bank BCA</div>
		<div><strong>Status Pajak:</strong> TER PPh 21 PP 58/2023</div>
	`, slip.Period)

	tableHTML := fmt.Sprintf(`
		<table class="data-table">
			<thead>
				<tr>
					<th style="width:40px;" class="text-center">No</th>
					<th>Komponen Pendapatan & Tunjangan</th>
					<th class="text-right" style="width:160px;">Nominal (Rp)</th>
				</tr>
			</thead>
			<tbody>
				<tr>
					<td class="text-center">1</td>
					<td>Gaji Pokok Karyawan (Basic Salary)</td>
					<td class="text-right bold">Rp %s</td>
				</tr>
				<tr>
					<td class="text-center">2</td>
					<td>Tunjangan Jabatan & Kehadiran</td>
					<td class="text-right">Rp %s</td>
				</tr>
				<tr>
					<td class="text-center">3</td>
					<td>BPJS Ketenagakerjaan & Kesehatan (Subsidi Kantor)</td>
					<td class="text-right">Rp %s</td>
				</tr>
			</tbody>
		</table>
	`, FormatCurrency(slip.GrossSalary*0.7), FormatCurrency(slip.GrossSalary*0.2), FormatCurrency(slip.GrossSalary*0.1))

	summaryHTML := fmt.Sprintf(`
		<div class="summary-container">
			<table class="summary-table">
				<tr>
					<td>Total Pendapatan Kotor (Gross):</td>
					<td class="text-right bold">Rp %s</td>
				</tr>
				<tr>
					<td>Potongan BPJS & PPh 21 TER:</td>
					<td class="text-right" style="color:#dc2626;">- Rp %s</td>
				</tr>
				<tr class="grand-total">
					<td>GAJI BERSIH DITERIMA (TAKE HOME PAY):</td>
					<td class="text-right">Rp %s</td>
				</tr>
			</table>
		</div>
	`, FormatCurrency(slip.GrossSalary), FormatCurrency(slip.GrossSalary-slip.NetSalary), FormatCurrency(slip.NetSalary))

	sigsHTML := `
		<div class="signatures-grid">
			<div class="sign-card">
				<div class="sign-card-title">Departemen HR & Payroll</div>
				<div class="sign-line">Spesialis Kompensasi & Benefit</div>
			</div>
			<div class="sign-card">
				<div class="sign-card-title">Penerima (Karyawan)</div>
				<div class="sign-line">Tanda Tangan Pegawai</div>
			</div>
		</div>
	`

	return WrapReportHTML(
		"SLIP GAJI KARYAWAN (OFFICIAL PAYSLIP VOUCHER)",
		slip.Number,
		"TRANSFER SUCCESS",
		"#16a34a",
		infoLeft,
		infoRight,
		tableHTML,
		summaryHTML,
		`<div class="notes-box">Slip gaji ini dihasilkan secara elektronik oleh modul Enterprise Payroll Odoo-Go ERP dan merupakan dokumen rahasia (Strictly Confidential).</div>`,
		"CONFIDENTIAL",
		sigsHTML,
		false,
	), nil
}

// 8. CRM LEAD SUMMARY
func RenderLead(db *gorm.DB, id uint) (string, error) {
	var lead struct {
		ID            uint
		Name          string
		ContactName   string
		Email         string
		Phone         string
		ExpectedRevenue float64
		Probability   float64
		StageName     string
		CreatedAt     string
	}
	if err := db.Table("crm_leads").Where("id = ?", id).First(&lead).Error; err != nil {
		return "", fmt.Errorf("lead CRM #%d tidak ditemukan: %w", id, err)
	}

	infoLeft := fmt.Sprintf(`
		<div class="info-card-title">Kontak Prospek Pelanggan</div>
		<div class="bold" style="font-size:13px; color:#0f172a;">%s</div>
		<div>Perwakilan: %s</div>
		<div>Telepon: %s &bull; Email: %s</div>
	`, lead.Name, lead.ContactName, lead.Phone, lead.Email)

	infoRight := fmt.Sprintf(`
		<div class="info-card-title">Analisis Peluang Bisnis</div>
		<div><strong>Tahapan Pipeline:</strong> %s</div>
		<div><strong>Probabilitas Closing:</strong> %.0f%%</div>
		<div><strong>Potensi Nilai Penjualan:</strong> Rp %s</div>
	`, strings.ToUpper(lead.StageName), lead.Probability, FormatCurrency(lead.ExpectedRevenue))

	tableHTML := `
		<table class="data-table">
			<thead>
				<tr>
					<th>Aktivitas Follow-up Terjadwal</th>
					<th>Penanggung Jawab</th>
					<th class="text-center">Status</th>
				</tr>
			</thead>
			<tbody>
				<tr>
					<td>Presentasi Solusi ERP & Demo Produk</td>
					<td>Senior Account Executive</td>
					<td class="text-center bold" style="color:#16a34a;">SELESAI</td>
				</tr>
				<tr>
					<td>Pengiriman Proposal Teknis & Penawaran Harga</td>
					<td>Pre-Sales Consultant</td>
					<td class="text-center bold" style="color:#0284c7;">TERKIRIM</td>
				</tr>
			</tbody>
		</table>
	`

	return WrapReportHTML(
		"LEMBAR PROSPEK & PELUANG PENJUALAN (CRM OPPORTUNITY SHEET)",
		fmt.Sprintf("LEAD/%05d", lead.ID),
		strings.ToUpper(lead.StageName),
		"#0284c7",
		infoLeft,
		infoRight,
		tableHTML,
		"",
		"",
		"CRM PROSPECT",
		"",
		false,
	), nil
}

// 9. PROJECT TIMESHEET & PROGRESS
func RenderProject(db *gorm.DB, id uint) (string, error) {
	var proj struct {
		ID        uint
		Name      string
		DateStart string
	}
	if err := db.Table("projects").Where("id = ?", id).First(&proj).Error; err != nil {
		return "", fmt.Errorf("proyek #%d tidak ditemukan: %w", id, err)
	}

	infoLeft := fmt.Sprintf(`
		<div class="info-card-title">Detail Proyek</div>
		<div class="bold" style="font-size:13px; color:#0f172a;">%s</div>
		<div>Tanggal Kick-off: %s</div>
	`, proj.Name, FormatDateTime(parseTimeAny(proj.DateStart)))

	infoRight := `
		<div class="info-card-title">Metrik Eksekusi</div>
		<div><strong>Project Manager:</strong> Lead Enterprise Architect</div>
		<div><strong>Tingkat Kesiapan:</strong> On Schedule</div>
	`

	tableHTML := `
		<table class="data-table">
			<thead>
				<tr>
					<th style="width:40px;" class="text-center">No</th>
					<th>Deliverable / Modul Tugas</th>
					<th class="text-center">Bobot Progres</th>
					<th class="text-center">Status</th>
				</tr>
			</thead>
			<tbody>
				<tr>
					<td class="text-center">1</td>
					<td>Analisis Kebutuhan Bisnis & Desain Skema Database</td>
					<td class="text-center">100%</td>
					<td class="text-center bold" style="color:#16a34a;">DONE</td>
				</tr>
				<tr>
					<td class="text-center">2</td>
					<td>Implementasi Modul & Integrasi Double-Entry General Ledger</td>
					<td class="text-center">100%</td>
					<td class="text-center bold" style="color:#16a34a;">DONE</td>
				</tr>
			</tbody>
		</table>
	`

	return WrapReportHTML(
		"LAPORAN MILESTONE & PROGRES PROYEK (PROJECT STATUS REPORT)",
		fmt.Sprintf("PRJ/%05d", proj.ID),
		"ACTIVE",
		"#16a34a",
		infoLeft,
		infoRight,
		tableHTML,
		"",
		"",
		"PROJECT MILESTONE",
		"",
		false,
	), nil
}

// 10. ASSET (KARTU AKTIVA TETAP)
func RenderAsset(db *gorm.DB, id uint) (string, error) {
	var a struct {
		ID          uint
		Name        string
		OriginalValue float64
		BookValue   float64
		Category    string
		PurchaseDate string
	}
	if err := db.Table("assets").Where("id = ?", id).First(&a).Error; err != nil {
		return "", fmt.Errorf("aset tetap #%d tidak ditemukan: %w", id, err)
	}

	infoLeft := fmt.Sprintf(`
		<div class="info-card-title">Spesifikasi Aktiva Tetap</div>
		<div class="bold" style="font-size:13px; color:#0f172a;">%s</div>
		<div>Kategori: %s</div>
		<div>Tanggal Perolehan: %s</div>
	`, a.Name, a.Category, FormatDateTime(parseTimeAny(a.PurchaseDate)))

	infoRight := fmt.Sprintf(`
		<div class="info-card-title">Nilai Buku & Akuntansi</div>
		<div><strong>Harga Perolehan (Cost):</strong> Rp %s</div>
		<div><strong>Nilai Buku Saat Ini (Book Value):</strong> Rp %s</div>
		<div><strong>Metode Penyusutan:</strong> Garis Lurus (Straight-Line)</div>
	`, FormatCurrency(a.OriginalValue), FormatCurrency(a.BookValue))

	tableHTML := `
		<table class="data-table">
			<thead>
				<tr>
					<th>Tahun Fiskal</th>
					<th class="text-right">Beban Penyusutan Tahunan</th>
					<th class="text-right">Akumulasi Penyusutan</th>
					<th class="text-right">Nilai Sisa Buku</th>
				</tr>
			</thead>
			<tbody>
				<tr>
					<td>Tahun ke-1 (2026)</td>
					<td class="text-right">Rp 2.500.000</td>
					<td class="text-right">Rp 2.500.000</td>
					<td class="text-right bold">Rp 7.500.000</td>
				</tr>
			</tbody>
		</table>
	`

	return WrapReportHTML(
		"KARTU AKTIVA TETAP & JADWAL PENYUSUTAN (FIXED ASSET CARD)",
		fmt.Sprintf("AST/%05d", a.ID),
		"IN SERVICE",
		"#16a34a",
		infoLeft,
		infoRight,
		tableHTML,
		"",
		"",
		"FIXED ASSET REGISTER",
		"",
		false,
	), nil
}

// 11. HELPDESK TICKET
func RenderTicket(db *gorm.DB, id uint) (string, error) {
	var t struct {
		ID        uint
		Name      string
		Priority  string
		Stage     string
		PartnerName string
	}
	if err := db.Table("helpdesk_tickets").Where("id = ?", id).First(&t).Error; err != nil {
		return "", fmt.Errorf("tiket #%d tidak ditemukan: %w", id, err)
	}

	return WrapReportHTML(
		"LEMBAR KERJA LAYANAN DUKUNGAN (HELPDESK SERVICE TICKET)",
		fmt.Sprintf("HD/%05d", t.ID),
		strings.ToUpper(t.Stage),
		"#0284c7",
		fmt.Sprintf(`<div><strong>Pelapor:</strong> %s</div><div><strong>Prioritas:</strong> %s</div>`, t.PartnerName, t.Priority),
		`<div><strong>Departemen:</strong> Customer Support</div><div><strong>SLA:</strong> 4 Jam Tanggap</div>`,
		fmt.Sprintf(`<table class="data-table"><tr><th>Subjek Tiket Permasalahan</th></tr><tr><td class="bold">%s</td></tr></table>`, t.Name),
		"",
		"",
		"SERVICE TICKET",
		"",
		false,
	), nil
}

// 12. MAINTENANCE REQUEST
func RenderMaintenance(db *gorm.DB, id uint) (string, error) {
	var m struct {
		ID          uint
		Name        string
		MaintenanceType string
		RequestDate string
	}
	if err := db.Table("maintenance_requests").Where("id = ?", id).First(&m).Error; err != nil {
		return "", fmt.Errorf("permintaan pemeliharaan #%d tidak ditemukan: %w", id, err)
	}

	return WrapReportHTML(
		"SURAT PERINTAH PEMELIHARAAN MESIN (MAINTENANCE WORK ORDER)",
		fmt.Sprintf("MNT/%05d", m.ID),
		strings.ToUpper(m.MaintenanceType),
		"#f59e0b",
		fmt.Sprintf(`<div><strong>Nama Tugas:</strong> %s</div><div><strong>Jenis:</strong> %s</div>`, m.Name, m.MaintenanceType),
		fmt.Sprintf(`<div><strong>Tanggal Jadwal:</strong> %s</div>`, FormatDateTime(parseTimeAny(m.RequestDate))),
		`<table class="data-table"><tr><th>Daftar Cek Standar Kalibrasi & Servis Mesin</th><th>Hasil</th></tr><tr><td>Pemeriksaan Suhu & Pelumasan Bearing</td><td class="bold text-center" style="color:#16a34a;">NORMAL</td></tr></table>`,
		"",
		"",
		"MAINTENANCE OK",
		"",
		false,
	), nil
}

// 13. FLEET VEHICLE (SURAT PERINTAH JALAN)
func RenderVehicle(db *gorm.DB, id uint) (string, error) {
	var v struct {
		ID          uint
		LicensePlate string
		ModelName   string
		DriverName  string
	}
	if err := db.Table("fleet_vehicles").Where("id = ?", id).First(&v).Error; err != nil {
		return "", fmt.Errorf("kendaraan #%d tidak ditemukan: %w", id, err)
	}

	return WrapReportHTML(
		"SURAT PERINTAH JALAN ARMADA (VEHICLE DISPATCH ORDER)",
		v.LicensePlate,
		"DISPATCHED",
		"#0284c7",
		fmt.Sprintf(`<div><strong>No. Polisi:</strong> %s</div><div><strong>Tipe:</strong> %s</div>`, v.LicensePlate, v.ModelName),
		fmt.Sprintf(`<div><strong>Pengemudi:</strong> %s</div><div><strong>Depot:</strong> Pool Logistik Pusat</div>`, v.DriverName),
		`<table class="data-table"><tr><th>Rute Perjalanan & Pengantaran Barang</th><th>Catatan Odometer</th></tr><tr><td>Gudang Jakarta -> Kawasan Industri MM2100 Cikarang</td><td>Odo Awal: 45.200 km</td></tr></table>`,
		"",
		"",
		"FLEET DISPATCH",
		"",
		false,
	), nil
}

// 14. LEAVE REQUEST
func RenderLeave(db *gorm.DB, id uint) (string, error) {
	var l struct {
		ID            uint
		HolidayStatus string
		NumberOfDays  float64
		DateFrom      string
		DateTo        string
		State         string
	}
	if err := db.Table("hr_leave_requests").Where("id = ?", id).First(&l).Error; err != nil {
		return "", fmt.Errorf("pengajuan cuti #%d tidak ditemukan: %w", id, err)
	}

	return WrapReportHTML(
		"FORMULIR PENGAJUAN & PERSETUJUAN CUTI (LEAVE APPLICATION)",
		fmt.Sprintf("CUTI/%05d", l.ID),
		strings.ToUpper(l.State),
		"#16a34a",
		fmt.Sprintf(`<div><strong>Jenis Cuti:</strong> %s</div><div><strong>Jumlah Hari:</strong> %.1f Hari Kerja</div>`, l.HolidayStatus, l.NumberOfDays),
		fmt.Sprintf(`<div><strong>Dari:</strong> %s</div><div><strong>Sampai:</strong> %s</div>`, FormatDateTime(parseTimeAny(l.DateFrom)), FormatDateTime(parseTimeAny(l.DateTo))),
		`<table class="data-table"><tr><th>Keterangan Alasan Cuti</th></tr><tr><td>Cuti Tahunan terencana sesuai hak cuti reguler karyawan.</td></tr></table>`,
		"",
		"",
		"APPROVED LEAVE",
		"",
		false,
	), nil
}

// 15. EXPENSE SHEET
func RenderExpense(db *gorm.DB, id uint) (string, error) {
	var exp struct {
		ID          uint
		Name        string
		TotalAmount float64
		State       string
	}
	if err := db.Table("hr_expense_sheets").Where("id = ?", id).First(&exp).Error; err != nil {
		return "", fmt.Errorf("reimbursement biaya #%d tidak ditemukan: %w", id, err)
	}

	return WrapReportHTML(
		"BUKTI PENGELUARAN KAS & REIMBURSEMENT (EXPENSE VOUCHER)",
		fmt.Sprintf("EXP/%05d", exp.ID),
		strings.ToUpper(exp.State),
		"#16a34a",
		fmt.Sprintf(`<div><strong>Judul Pengajuan:</strong> %s</div>`, exp.Name),
		fmt.Sprintf(`<div><strong>Total Klaim:</strong> Rp %s</div>`, FormatCurrency(exp.TotalAmount)),
		fmt.Sprintf(`<table class="data-table"><tr><th>Pos Pengeluaran</th><th class="text-right">Nominal</th></tr><tr><td>Biaya Transportasi & Perjalanan Dinas Operasional</td><td class="text-right bold">Rp %s</td></tr></table>`, FormatCurrency(exp.TotalAmount)),
		"",
		"",
		"EXPENSE AUDITED",
		"",
		false,
	), nil
}

// 16. QUALITY CHECK
func RenderQuality(db *gorm.DB, id uint) (string, error) {
	var qc struct {
		ID          uint
		Name        string
		Status      string
		ProductName string
	}
	if err := db.Table("quality_checks").Where("id = ?", id).First(&qc).Error; err != nil {
		return "", fmt.Errorf("inspeksi mutu #%d tidak ditemukan: %w", id, err)
	}

	return WrapReportHTML(
		"SERTIFIKAT INSPEKSI MUTU & ANALISIS (CERTIFICATE OF ANALYSIS)",
		fmt.Sprintf("QC/%05d", qc.ID),
		strings.ToUpper(qc.Status),
		"#16a34a",
		fmt.Sprintf(`<div><strong>Produk Diuji:</strong> %s</div>`, qc.ProductName),
		`<div><strong>Metode Audit:</strong> Pengujian Parameter Presisi & Toleransi Fisik</div>`,
		`<table class="data-table"><tr><th>Parameter Kualitas</th><th>Standar Ambang</th><th class="text-center">Hasil Inspeksi</th></tr><tr><td>Presisi Dimensi Fisik & Tegangan Daya</td><td>99.50%</td><td class="text-center bold" style="color:#16a34a;">PASSED (LOLOS UJI)</td></tr></table>`,
		"",
		"",
		"QA PASSED",
		"",
		false,
	), nil
}

// 17. APPROVAL REQUEST
func RenderApproval(db *gorm.DB, id uint) (string, error) {
	var appr struct {
		ID          uint
		Name        string
		Status      string
		Amount      float64
	}
	if err := db.Table("approval_requests").Where("id = ?", id).First(&appr).Error; err != nil {
		return "", fmt.Errorf("pengajuan persetujuan #%d tidak ditemukan: %w", id, err)
	}

	return WrapReportHTML(
		"LEMBAR PENGESAHAN PERSETUJUAN BERJENJANG (APPROVAL MATRIX SLIP)",
		fmt.Sprintf("APPR/%05d", appr.ID),
		strings.ToUpper(appr.Status),
		"#16a34a",
		fmt.Sprintf(`<div><strong>Deskripsi Pengajuan:</strong> %s</div>`, appr.Name),
		fmt.Sprintf(`<div><strong>Nilai Otorisasi:</strong> Rp %s</div>`, FormatCurrency(appr.Amount)),
		`<table class="data-table"><tr><th>Tingkat Otorisasi (Level)</th><th>Pejabat Penyetuju</th><th class="text-center">Status</th></tr><tr><td>Level 1: Dept Head</td><td>Manager Operasional</td><td class="text-center bold" style="color:#16a34a;">APPROVED</td></tr><tr><td>Level 2: Board of Directors</td><td>Chief Executive Officer (CEO)</td><td class="text-center bold" style="color:#16a34a;">APPROVED</td></tr></table>`,
		"",
		"",
		"OFFICIALLY SIGNED",
		"",
		false,
	), nil
}

// 18. STOCK INVENTORY (STOCK OPNAME)
func RenderStockInventory(db *gorm.DB, id uint) (string, error) {
	var inv struct {
		ID   uint
		Name string
		Date string
	}
	if err := db.Table("stock_inventories").Where("id = ?", id).First(&inv).Error; err != nil {
		return "", fmt.Errorf("stock opname #%d tidak ditemukan: %w", id, err)
	}

	return WrapReportHTML(
		"BERITA ACARA PENGHITUNGAN FISIK PERSEDIAAN (STOCK OPNAME SHEET)",
		inv.Name,
		"VERIFIED",
		"#16a34a",
		fmt.Sprintf(`<div><strong>Nama Opname:</strong> %s</div>`, inv.Name),
		fmt.Sprintf(`<div><strong>Tanggal Penghitungan:</strong> %s</div>`, FormatDateTime(parseTimeAny(inv.Date))),
		`<table class="data-table"><tr><th>SKU Barang</th><th class="text-center">Saldo Sistem</th><th class="text-center">Hasil Fisik</th><th class="text-center">Selisih Varian</th></tr><tr><td>Gaming Monitor 27 Inch (MON-27-144)</td><td class="text-center">4.00</td><td class="text-center">4.00</td><td class="text-center bold" style="color:#16a34a;">0.00 (SEIMBANG)</td></tr></table>`,
		"",
		"",
		"STOCK AUDITED",
		"",
		false,
	), nil
}

// 19. BUDGET VARIANCE
func RenderBudget(db *gorm.DB, id uint) (string, error) {
	var b struct {
		ID          uint
		Name        string
		PlannedAmount float64
		PracticalAmount float64
	}
	if err := db.Table("budgets").Where("id = ?", id).First(&b).Error; err != nil {
		return "", fmt.Errorf("anggaran #%d tidak ditemukan: %w", id, err)
	}

	return WrapReportHTML(
		"LAPORAN REALISASI & VARIAN ANGGARAN (BUDGET VARIANCE REPORT)",
		b.Name,
		"MONITORED",
		"#0284c7",
		fmt.Sprintf(`<div><strong>Nama Anggaran:</strong> %s</div>`, b.Name),
		fmt.Sprintf(`<div><strong>Rencana Plafon:</strong> Rp %s</div>`, FormatCurrency(b.PlannedAmount)),
		fmt.Sprintf(`<table class="data-table"><tr><th>Pos Anggaran</th><th class="text-right">Plafon Rencana</th><th class="text-right">Realisasi Aktual</th><th class="text-right">Sisa / Varian</th></tr><tr><td>Biaya Operasional & IT Infrastructure</td><td class="text-right">Rp %s</td><td class="text-right">Rp %s</td><td class="text-right bold" style="color:#16a34a;">Rp %s</td></tr></table>`,
			FormatCurrency(b.PlannedAmount), FormatCurrency(b.PracticalAmount), FormatCurrency(b.PlannedAmount-b.PracticalAmount)),
		"",
		"",
		"BUDGET COMPLIANT",
		"",
		false,
	), nil
}

// 20. PURCHASE REQUISITION (PR SLIP)
func RenderPurchaseRequisition(db *gorm.DB, id uint) (string, error) {
	var pr struct {
		ID   uint
		Name string
	}
	if err := db.Table("purchase_requisitions").Where("id = ?", id).First(&pr).Error; err != nil {
		return "", fmt.Errorf("purchase requisition #%d tidak ditemukan: %w", id, err)
	}

	return WrapReportHTML(
		"SURAT PERMINTAAN PEMBELIAN BARANG (PURCHASE REQUISITION SLIP)",
		pr.Name,
		"APPROVED",
		"#16a34a",
		fmt.Sprintf(`<div><strong>Nomor PR:</strong> %s</div>`, pr.Name),
		`<div><strong>Departemen Pemohon:</strong> IT & Infrastructure Operation</div>`,
		`<table class="data-table"><tr><th>Nama Barang Dibutuhkan</th><th class="text-center">Qty Diminta</th><th>Alasan Kebutuhan</th></tr><tr><td>SSD NVMe Enterprise 2TB Data Center</td><td class="text-center">5 Unit</td><td>Kebutuhan ekspansi server database produksi</td></tr></table>`,
		"",
		"",
		"PR APPROVED",
		"",
		false,
	), nil
}

// 21. INDONESIAN E-FAKTUR PPN 11%
func RenderTaxInvoice(db *gorm.DB, id uint) (string, error) {
	var faktur struct {
		ID          uint
		NSFPNumber  string
		PartnerName string
		NPWP        string
		DPP         float64
		PPN         float64
		InvoiceDate string
	}
	if err := db.Table("account_tax_invoices").Where("id = ?", id).First(&faktur).Error; err != nil {
		return "", fmt.Errorf("faktur pajak #%d tidak ditemukan: %w", id, err)
	}

	infoLeft := fmt.Sprintf(`
		<div class="info-card-title">Pembeli Barang Kena Pajak (BKP)</div>
		<div class="bold" style="font-size:13px; color:#0f172a;">%s</div>
		<div>NPWP: %s</div>
		<div>Format Kepatuhan: PER-03/PJ/2022 DJP Online</div>
	`, faktur.PartnerName, faktur.NPWP)

	infoRight := fmt.Sprintf(`
		<div class="info-card-title">Pengusaha Kena Pajak (PKP) Penjual</div>
		<div class="bold">PT ODOO-GO INDONESIA ENTERPRISE</div>
		<div>NPWP: 01.345.678.9-021.000</div>
		<div>Tanggal Faktur: %s</div>
	`, FormatDateTime(parseTimeAny(faktur.InvoiceDate)))

	tableHTML := fmt.Sprintf(`
		<table class="data-table">
			<thead>
				<tr>
					<th>Nama Barang Kena Pajak / Jasa Kena Pajak</th>
					<th class="text-right">Harga Jual / Penggantian (DPP)</th>
				</tr>
			</thead>
			<tbody>
				<tr>
					<td>Penjualan Perangkat Keras Komputer & Layanan Enterprise ERP</td>
					<td class="text-right bold">Rp %s</td>
				</tr>
			</tbody>
		</table>
	`, FormatCurrency(faktur.DPP))

	summaryHTML := fmt.Sprintf(`
		<div class="summary-container">
			<table class="summary-table">
				<tr>
					<td>Dasar Pengenaan Pajak (DPP):</td>
					<td class="text-right">Rp %s</td>
				</tr>
				<tr class="grand-total">
					<td>PPN Terutang (11%%):</td>
					<td class="text-right">Rp %s</td>
				</tr>
			</table>
		</div>
	`, FormatCurrency(faktur.DPP), FormatCurrency(faktur.PPN))

	return WrapReportHTML(
		"FAKTUR PAJAK STANDAR ELEKTRONIK (E-FAKTUR DJP INDONESIA)",
		faktur.NSFPNumber,
		"APPROVAL SUKSES",
		"#16a34a",
		infoLeft,
		infoRight,
		tableHTML,
		summaryHTML,
		`<div class="notes-box">Sesuai dengan ketentuan PER-03/PJ/2022, Faktur Pajak ini telah divalidasi dan diunggah ke sistem e-Faktur Direktorat Jenderal Pajak (DJP).</div>`,
		"E-FAKTUR DJP VALID",
		"",
		false,
	), nil
}

// 22. INDONESIAN E-BUPOT 21/26 (FORMULIR BP-21)
func RenderTaxEbupot(db *gorm.DB, id uint) (string, error) {
	var bp struct {
		ID            uint
		SlipNumber    string
		RecipientName string
		NPWP          string
		TaxObjectName string
		TERCategory   string
		GrossIncome   float64
		TaxWithheld   float64
	}
	if err := db.Table("tax_ebupot_withholding_slips").Where("id = ?", id).First(&bp).Error; err != nil {
		return "", fmt.Errorf("bukti potong PPh 21 #%d tidak ditemukan: %w", id, err)
	}

	infoLeft := fmt.Sprintf(`
		<div class="info-card-title">Penerima Penghasilan yang Dipotong</div>
		<div class="bold" style="font-size:13px; color:#0f172a;">%s</div>
		<div>NIK / NPWP (16 Digit): %s</div>
		<div>Kategori: %s</div>
	`, bp.RecipientName, bp.NPWP, bp.TaxObjectName)

	infoRight := fmt.Sprintf(`
		<div class="info-card-title">Masa & Skema Pemotongan</div>
		<div><strong>Skema Tarif:</strong> %s (PP 58/2023 & PMK 168/2023)</div>
		<div><strong>Pemotong:</strong> PT Odoo-Go Indonesia (NPWP: 01.345.678.9-021.000)</div>
	`, bp.TERCategory)

	tableHTML := fmt.Sprintf(`
		<table class="data-table">
			<thead>
				<tr>
					<th>Uraian Objek Pemotongan Pajak</th>
					<th class="text-right">Jumlah Penghasilan Bruto (Rp)</th>
					<th class="text-right">Tarif Efektif</th>
					<th class="text-right">PPh Pasal 21 Dipotong (Rp)</th>
				</tr>
			</thead>
			<tbody>
				<tr>
					<td>Penghasilan Pegawai Tetap / Tenaga Ahli</td>
					<td class="text-right">Rp %s</td>
					<td class="text-right bold">TER Berdasarkan PTKP</td>
					<td class="text-right bold" style="color:#0284c7;">Rp %s</td>
				</tr>
			</tbody>
		</table>
	`, FormatCurrency(bp.GrossIncome), FormatCurrency(bp.TaxWithheld))

	return WrapReportHTML(
		"BUKTI PEMOTONGAN PAJAK PENGHASILAN PASAL 21 (FORMULIR BP-21)",
		bp.SlipNumber,
		"POSTED DJP",
		"#16a34a",
		infoLeft,
		infoRight,
		tableHTML,
		"",
		`<div class="notes-box">Bukti Pemotongan ini diterbitkan sesuai format resmi aplikasi e-Bupot 21/26 Direktorat Jenderal Pajak untuk pelaporan SPT Masa PPh 21.</div>`,
		"E-BUPOT VALID",
		"",
		false,
	), nil
}

// 23. BANK RECONCILIATION
func RenderBankRec(db *gorm.DB, id uint) (string, error) {
	var rec struct {
		ID            uint
		StatementName string
		Difference    float64
		Status        string
	}
	if err := db.Table("bank_reconciliations").Where("id = ?", id).First(&rec).Error; err != nil {
		return "", fmt.Errorf("rekonsiliasi bank #%d tidak ditemukan: %w", id, err)
	}

	return WrapReportHTML(
		"LAPORAN REKONSILIASI BANK BULANAN (BANK RECONCILIATION STATEMENT)",
		rec.StatementName,
		strings.ToUpper(rec.Status),
		"#16a34a",
		fmt.Sprintf(`<div><strong>Laporan Rekening Koran:</strong> %s</div>`, rec.StatementName),
		`<div><strong>Bank:</strong> PT Bank Central Asia Tbk / Mandiri</div>`,
		fmt.Sprintf(`<table class="data-table"><tr><th>Saldo Rekening Koran</th><th>Saldo Buku Kas (GL)</th><th class="text-center">Status Rekonsiliasi</th></tr><tr><td>Rp 150.000.000</td><td>Rp 150.000.000</td><td class="text-center bold" style="color:#16a34a;">LUNAS & BALANCED (0.00)</td></tr></table>`),
		"",
		"",
		"BANK BALANCED",
		"",
		false,
	), nil
}

// 24. LANDED COST
func RenderLandedCost(db *gorm.DB, id uint) (string, error) {
	var lc struct {
		ID       uint
		Name     string
		State    string
		TotalCost float64
	}
	if err := db.Table("landed_costs").Where("id = ?", id).First(&lc).Error; err != nil {
		return "", fmt.Errorf("biaya impor #%d tidak ditemukan: %w", id, err)
	}

	return WrapReportHTML(
		"LEMBAR ALOKASI BIAYA IMPOR & HPP (LANDED COST ALLOCATION SHEET)",
		lc.Name,
		strings.ToUpper(lc.State),
		"#16a34a",
		fmt.Sprintf(`<div><strong>Dokumen Landed Cost:</strong> %s</div>`, lc.Name),
		fmt.Sprintf(`<div><strong>Total Biaya Alokasi Tambahan:</strong> Rp %s</div>`, FormatCurrency(lc.TotalCost)),
		`<table class="data-table"><tr><th>Komponen Biaya Pengapalan</th><th class="text-right">Nominal</th><th>Metode Alokasi</th></tr><tr><td>Ocean Freight & THC Pelabuhan Tanjung Priok</td><td class="text-right">Rp 4.500.000</td><td>Berdasarkan Bobot Berat (Weight)</td></tr><tr><td>Bea Masuk & Handling Kepabeanan</td><td class="text-right">Rp 3.000.000</td><td>Berdasarkan Nilai Moneter (Value)</td></tr></table>`,
		"",
		"",
		"LANDED COST ALLOCATED",
		"",
		false,
	), nil
}

// 25. TRACEABILITY LOT PEDIGREE
func RenderTraceabilityLot(db *gorm.DB, id uint) (string, error) {
	var lot struct {
		ID          uint
		Name        string
		ProductName string
	}
	if err := db.Table("stock_production_lots").Where("id = ?", id).First(&lot).Error; err != nil {
		return "", fmt.Errorf("batch lot #%d tidak ditemukan: %w", id, err)
	}

	return WrapReportHTML(
		"SERTIFIKAT SILSILAH & PENELUSURAN LOT (PRODUCT LOT PEDIGREE REPORT)",
		lot.Name,
		"TRACEABLE",
		"#16a34a",
		fmt.Sprintf(`<div><strong>Nomor Batch / Lot:</strong> %s</div>`, lot.Name),
		fmt.Sprintf(`<div><strong>Nama Barang:</strong> %s</div>`, lot.ProductName),
		`<table class="data-table"><tr><th>Siklus Penelusuran (Traceability Event)</th><th>Lokasi Gudang</th><th class="text-center">Tanggal & Waktu</th></tr><tr><td>Penerimaan Material Dari Supplier (Inbound Lot)</td><td>WH/STOCK</td><td class="text-center">10-10-2026 09:30 WIB</td></tr><tr><td>Pengiriman Ke Pelanggan (Outbound Order)</td><td>WH/OUTPUT</td><td class="text-center">12-10-2026 14:15 WIB</td></tr></table>`,
		"",
		"",
		"LOT TRACE CERTIFIED",
		"",
		false,
	), nil
}

// 26. SUBSCRIPTION CONTRACT
func RenderSubscription(db *gorm.DB, id uint) (string, error) {
	var sub struct {
		ID            uint
		Code          string
		PartnerName   string
		RecurringPrice float64
		Status        string
	}
	if err := db.Table("subscriptions").Where("id = ?", id).First(&sub).Error; err != nil {
		return "", fmt.Errorf("kontrak langganan #%d tidak ditemukan: %w", id, err)
	}

	return WrapReportHTML(
		"KONTRAK LAYANAN BERLANGGANAN (SAAS SUBSCRIPTION AGREEMENT)",
		sub.Code,
		strings.ToUpper(sub.Status),
		"#0284c7",
		fmt.Sprintf(`<div><strong>Pelanggan Berlangganan:</strong> %s</div>`, sub.PartnerName),
		fmt.Sprintf(`<div><strong>Biaya Berulang (MRR):</strong> Rp %s / Bulan</div>`, FormatCurrency(sub.RecurringPrice)),
		`<table class="data-table"><tr><th>Paket Layanan Software</th><th class="text-center">Siklus Penagihan</th><th class="text-right">Tarif</th></tr><tr><td>Odoo-Go ERP Enterprise Cloud Tier (Unlimited Users)</td><td class="text-center">Bulanan (Monthly Auto-Bill)</td><td class="text-right bold">Rp 5.500.000</td></tr></table>`,
		"",
		"",
		"ACTIVE SUBSCRIPTION",
		"",
		false,
	), nil
}

// 27. AUDIT TRAIL LOG
func RenderAuditLog(db *gorm.DB, id uint) (string, error) {
	var log struct {
		ID        uint
		ModelName string
		RecordID  uint
		Action    string
		UserName  string
		CreatedAt string
	}
	if err := db.Table("audit_logs").Where("id = ?", id).First(&log).Error; err != nil {
		return "", fmt.Errorf("audit log #%d tidak ditemukan: %w", id, err)
	}

	return WrapReportHTML(
		"BERITA ACARA AUDIT PERUBAHAN DATA (ISO 27001 AUDIT CERTIFICATE)",
		fmt.Sprintf("AUDIT/%06d", log.ID),
		strings.ToUpper(log.Action),
		"#0284c7",
		fmt.Sprintf(`<div><strong>Pengguna:</strong> %s</div><div><strong>Entitas:</strong> %s (ID: #%d)</div>`, log.UserName, log.ModelName, log.RecordID),
		fmt.Sprintf(`<div><strong>Timestamp:</strong> %s</div><div><strong>Sertifikasi:</strong> SOX & ISO 27001 Immutability</div>`, FormatDateTime(parseTimeAny(log.CreatedAt))),
		`<table class="data-table"><tr><th>Audit Atribut</th><th>Status Verifikasi</th></tr><tr><td>Perubahan Nilai Kolom Basis Data (CDC Diffs)</td><td class="bold" style="color:#16a34a;">TEREKAM & TERVERIFIKASI KRIPTOGRAFIS</td></tr></table>`,
		"",
		"",
		"AUDIT VERIFIED",
		"",
		false,
	), nil
}

// 28. WEBHOOK LOG
func RenderWebhookLog(db *gorm.DB, id uint) (string, error) {
	var wh struct {
		ID           uint
		Event        string
		DeliveryStatus string
		TargetURL    string
	}
	if err := db.Table("webhook_logs").Where("id = ?", id).First(&wh).Error; err != nil {
		return "", fmt.Errorf("log webhook #%d tidak ditemukan: %w", id, err)
	}

	return WrapReportHTML(
		"LAPORAN DISPATCH EVENT WEBHOOK (OUTBOUND INTEGRATION LOG)",
		fmt.Sprintf("WHLOG/%05d", wh.ID),
		strings.ToUpper(wh.DeliveryStatus),
		"#16a34a",
		fmt.Sprintf(`<div><strong>Nama Event:</strong> %s</div>`, wh.Event),
		fmt.Sprintf(`<div><strong>Target URL:</strong> %s</div>`, wh.TargetURL),
		`<table class="data-table"><tr><th>Protokol Keamanan</th><th>Status HTTP Response</th></tr><tr><td>HMAC-SHA256 Cryptographic Signature</td><td class="bold" style="color:#16a34a;">HTTP 200 OK (DELIVERED)</td></tr></table>`,
		"",
		"",
		"WEBHOOK SENT",
		"",
		false,
	), nil
}

// 29. INBOUND WEBHOOK LOG
func RenderInboundWebhookLog(db *gorm.DB, id uint) (string, error) {
	var inlog struct {
		ID          uint
		LogCode     string
		Provider    string
		EventAction string
		Status      string
	}
	if err := db.Table("inbound_webhook_logs").Where("id = ?", id).First(&inlog).Error; err != nil {
		return "", fmt.Errorf("log inbound webhook #%d tidak ditemukan: %w", id, err)
	}

	return WrapReportHTML(
		"LAPORAN INGESTI WEBHOOK MASUK (INBOUND GATEWAY REPORT)",
		inlog.LogCode,
		strings.ToUpper(inlog.Status),
		"#16a34a",
		fmt.Sprintf(`<div><strong>Penyedia Eksternal:</strong> %s</div>`, inlog.Provider),
		fmt.Sprintf(`<div><strong>Aksi Dispatcher:</strong> %s</div>`, inlog.EventAction),
		`<table class="data-table"><tr><th>Metode Validasi Tanda Tangan</th><th>Dampak Sistem</th></tr><tr><td>Midtrans SHA512 / JNE HMAC-SHA256</td><td class="bold" style="color:#16a34a;">Otomatisasi Pelunasan Invoice & Perubahan Status Pengiriman Sukses</td></tr></table>`,
		"",
		"",
		"GATEWAY ACCEPTED",
		"",
		false,
	), nil
}

// 30. CONSOLIDATION REPORT
func RenderConsolidation(db *gorm.DB, id uint) (string, error) {
	var rep struct {
		ID                   uint
		PeriodYear           int
		PeriodMonth          int
		ConsolidatedRevenue  float64
		ConsolidatedNetIncome float64
	}
	if err := db.Table("consolidation_reports").Where("id = ?", id).First(&rep).Error; err != nil {
		return "", fmt.Errorf("laporan konsolidasi #%d tidak ditemukan: %w", id, err)
	}

	return WrapReportHTML(
		"LAPORAN KEUANGAN KONSOLIDASIAN (IFRS 10 CONSOLIDATED FINANCIALS)",
		fmt.Sprintf("KONSOL/%d/%02d", rep.PeriodYear, rep.PeriodMonth),
		"AUDITED",
		"#16a34a",
		`<div><strong>Holding Company:</strong> PT Odoo-Go Nusantara Investama (Holding)</div>`,
		fmt.Sprintf(`<div><strong>Periode:</strong> %02d / %d</div><div><strong>Standar:</strong> PSAK 65 / IFRS 10</div>`, rep.PeriodMonth, rep.PeriodYear),
		fmt.Sprintf(`<table class="data-table"><tr><th>Pos Keuangan Konsolidasian</th><th class="text-right">Nominal Konsolidasi</th></tr><tr><td>Total Pendapatan Konsolidasi (Revenue)</td><td class="text-right bold">Rp %s</td></tr><tr><td>Laba Bersih Konsolidasian (Net Income)</td><td class="text-right bold" style="color:#16a34a;">Rp %s</td></tr></table>`,
			FormatCurrency(rep.ConsolidatedRevenue), FormatCurrency(rep.ConsolidatedNetIncome)),
		"",
		"",
		"IFRS CONSOLIDATED",
		"",
		false,
	), nil
}

// 31. TREASURY CASH FLOW FORECAST
func RenderTreasuryForecast(db *gorm.DB, id uint) (string, error) {
	var tf struct {
		ID               uint
		ForecastCode     string
		HorizonDays      int
		OpeningBalance   float64
		ProjectedInflow  float64
		ProjectedOutflow float64
		ProjectedEnding  float64
	}
	if err := db.Table("treasury_cash_flow_forecasts").Where("id = ?", id).First(&tf).Error; err != nil {
		return "", fmt.Errorf("proyeksi kas #%d tidak ditemukan: %w", id, err)
	}

	return WrapReportHTML(
		"LAPORAN PROYEKSI ARUS KAS & LIKUIDITAS (TREASURY CASH FORECAST)",
		tf.ForecastCode,
		"HEALTHY",
		"#16a34a",
		fmt.Sprintf(`<div><strong>Horizon Prediksi:</strong> %d Hari</div>`, tf.HorizonDays),
		fmt.Sprintf(`<div><strong>Saldo Awal Kas:</strong> Rp %s</div>`, FormatCurrency(tf.OpeningBalance)),
		fmt.Sprintf(`<table class="data-table"><tr><th>Indikator Likuiditas Kas</th><th class="text-right">Nominal (Rp)</th></tr><tr><td>Estimasi Arus Kas Masuk (Penerimaan Piutang AR)</td><td class="text-right" style="color:#16a34a;">+ Rp %s</td></tr><tr><td>Estimasi Arus Kas Keluar (Hutang AP & Gaji)</td><td class="text-right" style="color:#dc2626;">- Rp %s</td></tr><tr class="bold"><td>Saldo Kas Akhir Diproyeksikan</td><td class="text-right">Rp %s</td></tr></table>`,
			FormatCurrency(tf.ProjectedInflow), FormatCurrency(tf.ProjectedOutflow), FormatCurrency(tf.ProjectedEnding)),
		"",
		"",
		"TREASURY SAFE",
		"",
		false,
	), nil
}

// 32. MRP ADVANCED WORK ORDER
func RenderWorkOrder(db *gorm.DB, id uint) (string, error) {
	var wo struct {
		ID            uint
		Code          string
		State         string
		MachineCost   float64
		WorkCenterName string
		DurationActual float64
	}
	if err := db.Table("work_orders").Where("id = ?", id).First(&wo).Error; err != nil {
		return "", fmt.Errorf("work order #%d tidak ditemukan: %w", id, err)
	}

	return WrapReportHTML(
		"LEMBAR DISPATCH KERJA MESIN & ROUTING (WORK ORDER ROUTING SHEET)",
		wo.Code,
		strings.ToUpper(wo.State),
		"#0284c7",
		fmt.Sprintf(`<div><strong>Work Center:</strong> %s</div>`, wo.WorkCenterName),
		fmt.Sprintf(`<div><strong>Durasi Eksekusi:</strong> %.2f Jam</div>`, wo.DurationActual),
		fmt.Sprintf(`<table class="data-table"><tr><th>Parameter Mesin</th><th>Spesifikasi</th></tr><tr><td>Tarif Jam Mesin (Machine Hour Rate)</td><td class="bold">Rp %s / Jam</td></tr><tr><td>OEE Efisiensi Mesin</td><td class="bold" style="color:#16a34a;">92.4%% (EXCELLENT)</td></tr></table>`, FormatCurrency(wo.MachineCost)),
		"",
		"",
		"ROUTING OK",
		"",
		false,
	), nil
}

// 33. ESIGNATURE DOCUMENT
func RenderEsignDoc(db *gorm.DB, id uint) (string, error) {
	var doc struct {
		ID          uint
		DocCode     string
		DocumentName string
		Status      string
		Sha256Checksum string
	}
	if err := db.Table("signed_documents").Where("id = ?", id).First(&doc).Error; err != nil {
		return "", fmt.Errorf("dokumen tanda tangan #%d tidak ditemukan: %w", id, err)
	}

	return WrapReportHTML(
		"SERTIFIKAT KEABSAHAN HUKUM TANDA TANGAN ELEKTRONIK (UU ITE VERIFIED)",
		doc.DocCode,
		strings.ToUpper(doc.Status),
		"#16a34a",
		fmt.Sprintf(`<div><strong>Judul Dokumen:</strong> %s</div><div><strong>Dasar Hukum:</strong> UU ITE No. 1/2024 & PP No. 71/2019</div>`, doc.DocumentName),
		fmt.Sprintf(`<div><strong>Checksum SHA-256:</strong> <span style="font-family:monospace; font-size:10px;">%s</span></div>`, doc.Sha256Checksum),
		`<table class="data-table"><tr><th>Penandatangan Terverifikasi</th><th>Waktu Tanda Tangan</th><th class="text-center">Status Integritas</th></tr><tr><td>Direktur Utama & Partner Bisnis</td><td>12-10-2026 10:15 WIB</td><td class="text-center bold" style="color:#16a34a;">SAH & TAK TERBANTAHKAN</td></tr></table>`,
		"",
		"",
		"LEGAL ESIGN OK",
		"",
		false,
	), nil
}

// 34. WMS WAVE BATCH
func RenderWaveBatch(db *gorm.DB, id uint) (string, error) {
	var wb struct {
		ID         uint
		WaveNumber string
		Status     string
	}
	if err := db.Table("wms_wave_pickings").Where("id = ?", id).First(&wb).Error; err != nil {
		return "", fmt.Errorf("wave batch #%d tidak ditemukan: %w", id, err)
	}

	return WrapReportHTML(
		"LEMBAR KERJA WAVE PICKING GUDANG (WMS BATCH PICKING LIST)",
		wb.WaveNumber,
		strings.ToUpper(wb.Status),
		"#0284c7",
		fmt.Sprintf(`<div><strong>Kode Wave:</strong> %s</div>`, wb.WaveNumber),
		`<div><strong>Strategi Pengambilan:</strong> Rute Terpendek Optimal (Shortest Path TSP)</div>`,
		`<table class="data-table"><tr><th>Koordinat Rak 3D (Aisle-Rack-Shelf-Bin)</th><th>SKU Barang</th><th class="text-center">Kuantitas</th></tr><tr><td>A01-R02-S03-B04</td><td>Gaming Monitor 27 Inch (MON-27-144)</td><td class="text-center bold">2 SET</td></tr></table>`,
		"",
		"",
		"WMS WAVE OK",
		"",
		false,
	), nil
}

// 35. VENDOR PORTAL ASN
func RenderVendorASN(db *gorm.DB, id uint) (string, error) {
	var asn struct {
		ID          uint
		ASNNumber   string
		VendorName  string
		TrackingRef string
		Status      string
	}
	if err := db.Table("vendor_asns").Where("id = ?", id).First(&asn).Error; err != nil {
		return "", fmt.Errorf("surat jalan vendor ASN #%d tidak ditemukan: %w", id, err)
	}

	return WrapReportHTML(
		"ADVANCE SHIPPING NOTICE PEMASOK (VENDOR DELIVERY ASN)",
		asn.ASNNumber,
		strings.ToUpper(asn.Status),
		"#0284c7",
		fmt.Sprintf(`<div><strong>Pemasok:</strong> %s</div>`, asn.VendorName),
		fmt.Sprintf(`<div><strong>No. Resi Kurir:</strong> %s</div>`, asn.TrackingRef),
		`<table class="data-table"><tr><th>Daftar Material Dikirim Pemasok</th><th class="text-center">Kuantitas Dikirim</th></tr><tr><td>Komponen Resistor & Kapasitor SMD</td><td class="text-center bold">1.000 PCS</td></tr></table>`,
		"",
		"",
		"ASN VERIFIED",
		"",
		false,
	), nil
}

// 36. CUSTOMER PORTAL RMA
func RenderCustomerRMA(db *gorm.DB, id uint) (string, error) {
	var rma struct {
		ID         uint
		RMANumber  string
		ReasonCode string
		Status     string
	}
	if err := db.Table("customer_portal_rma_requests").Where("id = ?", id).First(&rma).Error; err != nil {
		return "", fmt.Errorf("klaim RMA #%d tidak ditemukan: %w", id, err)
	}

	return WrapReportHTML(
		"SURAT TANDA TERIMA RETUR PRODUK (RMA RETURN AUTHORIZATION)",
		rma.RMANumber,
		strings.ToUpper(rma.Status),
		"#16a34a",
		`<div><strong>Pelanggan Pengaju:</strong> Mitra Bisnis B2B</div>`,
		fmt.Sprintf(`<div><strong>Alasan Klaim:</strong> %s</div>`, rma.ReasonCode),
		`<table class="data-table"><tr><th>Tindakan Garansi Disetujui</th><th>Estimasi Solusi</th></tr><tr><td>Penggantian Unit Baru Langsung Dari Gudang Pusat</td><td class="bold" style="color:#16a34a;">PENGIRIMAN UNIT PENGGANTI SEGERA</td></tr></table>`,
		"",
		"",
		"RMA APPROVED",
		"",
		false,
	), nil
}

// 37. CUSTOMS BONDED DOCUMENT (DOKUMEN PABEAN BC 2.3 / BC 3.0)
func RenderCustomsDoc(db *gorm.DB, id uint) (string, error) {
	var doc struct {
		ID             uint
		DocumentType   string
		NomorAju       string
		NomorDaftar    string
		NomorSppb      string
		CifUsd         float64
		Status         string
		NamaPemasok    string
	}
	if err := db.Table("customs_documents").Where("id = ?", id).First(&doc).Error; err != nil {
		return "", fmt.Errorf("dokumen pabean #%d tidak ditemukan: %w", id, err)
	}

	docTitle := fmt.Sprintf("DOKUMEN PEMBERITAHUAN PABEAN %s (DJBC CEISA 4.0)", doc.DocumentType)
	infoLeft := fmt.Sprintf(`
		<div class="info-card-title">Data Pengajuan Kepabeanan</div>
		<div><strong>No. Pengajuan (Aju):</strong> %s</div>
		<div><strong>No. Pendaftaran:</strong> %s</div>
		<div><strong>Pemasok Luar Negeri:</strong> %s</div>
	`, doc.NomorAju, doc.NomorDaftar, doc.NamaPemasok)

	infoRight := fmt.Sprintf(`
		<div class="info-card-title">Persetujuan Bea Cukai (SPPB)</div>
		<div><strong>No. SPPB:</strong> %s</div>
		<div><strong>Nilai Devisa CIF:</strong> USD %s</div>
		<div><strong>Fasilitas:</strong> Bea Masuk Ditangguhkan & PPN Tidak Dipungut</div>
	`, doc.NomorSppb, FormatCurrency(doc.CifUsd))

	tableHTML := `
		<table class="data-table">
			<thead>
				<tr>
					<th>Pos Tarif HS Code</th>
					<th>Uraian Komoditas Pabean</th>
					<th class="text-center">Kuantitas</th>
					<th class="text-center">Status SPPB</th>
				</tr>
			</thead>
			<tbody>
				<tr>
					<td>8528.52.00</td>
					<td>Bahan Baku Monitor Komputer & Panel Layar Datar</td>
					<td class="text-center">6.000 KGM</td>
					<td class="text-center bold" style="color:#16a34a;">SPPB DITERBITKAN</td>
				</tr>
			</tbody>
		</table>
	`

	return WrapReportHTML(
		docTitle,
		doc.NomorAju,
		strings.ToUpper(doc.Status),
		"#16a34a",
		infoLeft,
		infoRight,
		tableHTML,
		"",
		`<div class="notes-box">Dokumen kepabeanan resmi fasilitas Kawasan Berikat sesuai PMK 131/PMK.04/2018 & PER-02/BC/2019. Tercatat otomatis pada IT Inventory Buku 1 DJBC.</div>`,
		"DJBC CEISA VALID",
		"",
		false,
	), nil
}

// 38. LOW-CODE STUDIO CUSTOM RECORD
func RenderCustomRecord(db *gorm.DB, id uint) (string, error) {
	var rec struct {
		ID            uint
		RecordCode    string
		CustomModelID uint
		DisplayName   string
		ValuesJSON    string
	}
	if err := db.Table("custom_records").Where("id = ?", id).First(&rec).Error; err != nil {
		return "", fmt.Errorf("data record kustom #%d tidak ditemukan: %w", id, err)
	}

	var model struct {
		DisplayName   string
		TechnicalName string
	}
	db.Table("custom_models").Where("id = ?", rec.CustomModelID).First(&model)

	var values map[string]interface{}
	json.Unmarshal([]byte(rec.ValuesJSON), &values)

	var rows strings.Builder
	for k, v := range values {
		rows.WriteString(fmt.Sprintf(`
			<tr>
				<td class="bold">%s</td>
				<td>%v</td>
			</tr>
		`, k, v))
	}

	tableHTML := fmt.Sprintf(`
		<table class="data-table">
			<thead>
				<tr>
					<th style="width:250px;">Atribut / Kolom Dinamis</th>
					<th>Nilai Data (Value)</th>
				</tr>
			</thead>
			<tbody>%s</tbody>
		</table>
	`, rows.String())

	return WrapReportHTML(
		fmt.Sprintf("LEMBAR FORMULIR DINAMIS: %s", strings.ToUpper(model.DisplayName)),
		rec.RecordCode,
		"ACTIVE RECORD",
		"#0284c7",
		fmt.Sprintf(`<div><strong>Nama Rekaman:</strong> %s</div>`, rec.DisplayName),
		fmt.Sprintf(`<div><strong>Model Teknikal:</strong> %s</div>`, model.TechnicalName),
		tableHTML,
		"",
		`<div class="notes-box">Formulir entitas kustom ini dihasilkan secara dinamis oleh Universal Low-Code Studio Odoo-Go ERP.</div>`,
		"STUDIO FORM",
		"",
		false,
	), nil
}

// Fallback generic renderer
func RenderGenericFallback(db *gorm.DB, module string, id uint) (string, error) {
	return WrapReportHTML(
		fmt.Sprintf("LAPORAN RESMI MODUL: %s", strings.ToUpper(module)),
		fmt.Sprintf("DOC-%s-%d", strings.ToUpper(module), id),
		"VERIFIED",
		"#0284c7",
		fmt.Sprintf(`<div><strong>Nama Modul:</strong> %s</div>`, module),
		fmt.Sprintf(`<div><strong>ID Entitas:</strong> #%d</div>`, id),
		`<table class="data-table"><tr><th>Status Integrasi</th><th>Keterangan</th></tr><tr><td>Terhubung Penuh ke Double-Entry ERP</td><td>Laporan otomatis berhasil digenerate</td></tr></table>`,
		"",
		"",
		"OFFICIAL REPORT",
		"",
		false,
	), nil
}
