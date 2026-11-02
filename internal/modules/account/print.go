package account

import (
	"fmt"
	"strings"
)

// RenderInvoiceHTML generates a professional, clean printable HTML invoice / bill
func RenderInvoiceHTML(move *AccountMove) string {
	docTitle := "FAKTUR PENJUALAN / INVOICE"
	partnerTitle := "Ditagihkan Kepada (Customer):"
	if move.MoveType == "in_invoice" {
		docTitle = "TAGIHAN PEMASOK / VENDOR BILL"
		partnerTitle = "Pemasok / Vendor:"
	} else if move.MoveType == "entry" {
		docTitle = "BUKTI JURNAL / JOURNAL ENTRY"
		partnerTitle = "Partner Terkait:"
	}

	statusColor := "#10b981"
	statusText := "LUNAS (PAID)"
	if move.PaymentState != "paid" {
		statusColor = "#f59e0b"
		statusText = "BELUM LUNAS (UNPAID)"
	}
	if move.State == "draft" {
		statusColor = "#6b7280"
		statusText = "DRAFT"
	}

	var linesHTML strings.Builder
	for idx, line := range move.LineIDs {
		// Only display lines that belong to receivable, payable, or revenue/expense for standard invoice items
		if line.Credit > 0 || line.Debit > 0 {
			lineTotal := line.Debit
			if move.MoveType == "out_invoice" {
				lineTotal = line.Credit
			}
			if lineTotal == 0 {
				lineTotal = line.Debit
			}
			linesHTML.WriteString(fmt.Sprintf(`
			<tr>
				<td style="padding:10px; border-bottom:1px solid #e5e7eb; text-align:center;">%d</td>
				<td style="padding:10px; border-bottom:1px solid #e5e7eb;">
					<strong>%s</strong>
					<div style="font-size:11px; color:#6b7280;">Akun: %s - %s</div>
				</td>
				<td style="padding:10px; border-bottom:1px solid #e5e7eb; text-align:right;">%.2f</td>
				<td style="padding:10px; border-bottom:1px solid #e5e7eb; text-align:right;">Rp %s</td>
				<td style="padding:10px; border-bottom:1px solid #e5e7eb; text-align:right;">Rp %s</td>
			</tr>
			`, idx+1, line.Name, line.Account.Code, line.Account.Name,
				line.Quantity, formatCurrency(line.PriceUnit), formatCurrency(lineTotal)))
		}
	}

	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="id">
<head>
	<meta charset="UTF-8">
	<title>%s - %s</title>
	<style>
		body { font-family: 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; color: #1f2937; margin: 0; padding: 40px; background: #fff; }
		.container { max-width: 800px; margin: auto; }
		.header { display: flex; justify-content: space-between; align-items: flex-start; border-bottom: 2px solid #3b82f6; padding-bottom: 20px; }
		.company-name { font-size: 24px; font-weight: bold; color: #1e40af; }
		.company-info { font-size: 12px; color: #4b5563; line-height: 1.5; margin-top: 5px; }
		.doc-title { text-align: right; }
		.doc-title h1 { margin: 0; font-size: 20px; color: #111827; }
		.doc-number { font-size: 16px; font-weight: bold; color: #3b82f6; margin-top: 4px; }
		.status-badge { display: inline-block; padding: 4px 10px; border-radius: 4px; color: #fff; font-weight: bold; font-size: 11px; margin-top: 6px; }
		.info-grid { display: flex; justify-content: space-between; margin: 25px 0; font-size: 13px; line-height: 1.6; }
		.info-box { width: 48%%; }
		.info-title { font-weight: bold; color: #374151; border-bottom: 1px solid #e5e7eb; padding-bottom: 4px; margin-bottom: 8px; }
		table { width: 100%%; border-collapse: collapse; margin-top: 15px; font-size: 13px; }
		th { background: #f3f4f6; color: #374151; padding: 10px; text-align: left; border-bottom: 2px solid #e5e7eb; }
		.total-section { display: flex; justify-content: flex-end; margin-top: 20px; }
		.total-table { width: 320px; font-size: 13px; }
		.total-table td { padding: 6px 10px; }
		.grand-total { font-size: 16px; font-weight: bold; color: #1e40af; border-top: 2px solid #3b82f6; border-bottom: 2px solid #3b82f6; }
		.footer-notes { margin-top: 30px; font-size: 12px; color: #6b7280; border-top: 1px solid #e5e7eb; padding-top: 15px; }
		.signatures { display: flex; justify-content: space-between; margin-top: 50px; text-align: center; font-size: 12px; }
		.sign-box { width: 200px; }
		.sign-line { margin-top: 60px; border-top: 1px solid #9ca3af; padding-top: 4px; }
		@media print {
			body { padding: 0; }
			.no-print { display: none; }
		}
	</style>
</head>
<body>
	<div class="container">
		<div class="no-print" style="margin-bottom: 20px; text-align: right;">
			<button onclick="window.print()" style="background:#2563eb; color:#fff; border:none; padding:10px 20px; border-radius:6px; cursor:pointer; font-weight:bold; font-size:14px;">🖨️ Cetak / Simpan PDF</button>
		</div>

		<div class="header">
			<div>
				<div class="company-name">🏢 PT GOLANG ERP INDONESIA</div>
				<div class="company-info">
					Kawasan Bisnis Sudirman, Tower B Lt. 12, Jakarta Selatan<br>
					NPWP: 01.234.567.8-012.000 | Telp: +62 21 555-8888<br>
					Email: billing@goerplang.co.id | www.goerplang.co.id
				</div>
			</div>
			<div class="doc-title">
				<h1>%s</h1>
				<div class="doc-number">%s</div>
				<div class="status-badge" style="background-color: %s;">%s</div>
			</div>
		</div>

		<div class="info-grid">
			<div class="info-box">
				<div class="info-title">%s</div>
				<strong>%s</strong><br>
				Email: %s<br>
				Telepon: %s<br>
				Alamat: %s, %s
			</div>
			<div class="info-box">
				<div class="info-title">Detail Tagihan:</div>
				Tanggal Faktur: <strong>%s</strong><br>
				Jatuh Tempo: <strong>%s</strong><br>
				Referensi Order: <strong>%s</strong><br>
				Metode Pembayaran: Bank Transfer (BCA: 123-456-7890 a/n PT GoERP)
			</div>
		</div>

		<table>
			<thead>
				<tr>
					<th style="text-align:center; width:30px;">#</th>
					<th>Deskripsi / Item Barang</th>
					<th style="text-align:right; width:70px;">Jumlah</th>
					<th style="text-align:right; width:120px;">Harga Satuan</th>
					<th style="text-align:right; width:130px;">Subtotal</th>
				</tr>
			</thead>
			<tbody>
				%s
			</tbody>
		</table>

		<div class="total-section">
			<table class="total-table">
				<tr>
					<td>Subtotal (Sebelum Pajak):</td>
					<td style="text-align:right;">Rp %s</td>
				</tr>
				<tr>
					<td>PPN / Pajak:</td>
					<td style="text-align:right;">Rp %s</td>
				</tr>
				<tr class="grand-total">
					<td>Total Pembayaran:</td>
					<td style="text-align:right;">Rp %s</td>
				</tr>
				<tr>
					<td>Sisa Tagihan (Residual):</td>
					<td style="text-align:right; font-weight:bold; color:%s;">Rp %s</td>
				</tr>
			</table>
		</div>

		<div class="footer-notes">
			<strong>Catatan Pembayaran:</strong><br>
			1. Mohon sertakan nomor invoice pada berita transfer bank.<br>
			2. Bukti transfer mohon diupload atau dikirimkan ke billing@goerplang.co.id.<br>
			Catatan Dokumen: %s
		</div>

		<div class="signatures">
			<div class="sign-box">
				Diterima Oleh,<br>
				<div class="sign-line">%s</div>
			</div>
			<div class="sign-box">
				Hormat Kami,<br>
				<div class="sign-line">PT GoERP Indonesia</div>
			</div>
		</div>
	</div>
</body>
</html>`,
		docTitle, move.Name,
		docTitle, move.Name, statusColor, statusText,
		partnerTitle, move.Partner.Name, move.Partner.Email, move.Partner.Phone, move.Partner.Street, move.Partner.City,
		move.InvoiceDate, move.InvoiceDueDate, move.Ref,
		linesHTML.String(),
		formatCurrency(move.AmountUntaxed),
		formatCurrency(move.AmountTax),
		formatCurrency(move.AmountTotal),
		statusColor, formatCurrency(move.AmountResidual),
		move.Note,
		move.Partner.Name,
	)

	return html
}

func formatCurrency(val float64) string {
	parts := fmt.Sprintf("%.2f", val)
	subparts := strings.Split(parts, ".")
	intPart := subparts[0]
	decPart := subparts[1]

	var res []string
	length := len(intPart)
	for i := length; i > 0; i -= 3 {
		start := i - 3
		if start < 0 {
			start = 0
		}
		res = append([]string{intPart[start:i]}, res...)
	}

	formattedInt := strings.Join(res, ".")
	if decPart == "00" {
		return formattedInt
	}
	return formattedInt + "," + decPart
}
