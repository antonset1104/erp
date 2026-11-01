package stock

import (
	"fmt"
	"strings"
)

// RenderPickingHTML generates a professional, clean printable Delivery Order / Stock Receipt HTML
func RenderPickingHTML(p *StockPicking) string {
	docTitle := "SURAT JALAN / PENGIRIMAN BARANG"
	partnerTitle := "Tujuan Pengiriman (Customer):"
	actionText := "DIKIRIM KE"
	if p.PickingType == "incoming" {
		docTitle = "BUKTI PENERIMAAN BARANG / RECEIPT"
		partnerTitle = "Diterima Dari (Vendor):"
		actionText = "DITERIMA DARI"
	} else if p.PickingType == "internal" {
		docTitle = "SURAT TRANSFER GUDANG INTERNAL"
		partnerTitle = "Penanggung Jawab:"
		actionText = "TRANSFER INTERNAL"
	}

	statusColor := "#10b981"
	statusText := "SELESAI (DONE)"
	if p.State == "draft" {
		statusColor = "#6b7280"
		statusText = "DRAFT"
	} else if p.State == "assigned" {
		statusColor = "#3b82f6"
		statusText = "SIAP KIRIM (READY)"
	}

	partnerName := p.PartnerName
	if partnerName == "" {
		partnerName = "Umum / Internal"
	}

	var linesHTML strings.Builder
	for idx, line := range p.MoveLines {
		qtyDisplay := line.QuantityDone
		if p.State != "done" {
			qtyDisplay = line.ProductUomQty
		}
		linesHTML.WriteString(fmt.Sprintf(`
		<tr>
			<td style="padding:10px; border-bottom:1px solid #e5e7eb; text-align:center;">%d</td>
			<td style="padding:10px; border-bottom:1px solid #e5e7eb;">
				<strong>%s</strong>
			</td>
			<td style="padding:10px; border-bottom:1px solid #e5e7eb; text-align:center;">%.2f</td>
			<td style="padding:10px; border-bottom:1px solid #e5e7eb; text-align:center; font-weight:bold; color:#1e40af;">%.2f</td>
			<td style="padding:10px; border-bottom:1px solid #e5e7eb; text-align:center;">%s</td>
		</tr>
		`, idx+1, line.ProductName, line.ProductUomQty, qtyDisplay, p.State))
	}

	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="id">
<head>
	<meta charset="UTF-8">
	<title>%s - %s</title>
	<style>
		body { font-family: 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; color: #1f2937; margin: 0; padding: 40px; background: #fff; }
		.container { max-width: 800px; margin: auto; }
		.header { display: flex; justify-content: space-between; align-items: flex-start; border-bottom: 2px solid #2563eb; padding-bottom: 20px; }
		.company-name { font-size: 24px; font-weight: bold; color: #1e40af; }
		.company-info { font-size: 12px; color: #4b5563; line-height: 1.5; margin-top: 5px; }
		.doc-title { text-align: right; }
		.doc-title h1 { margin: 0; font-size: 18px; color: #111827; }
		.doc-number { font-size: 16px; font-weight: bold; color: #2563eb; margin-top: 4px; }
		.status-badge { display: inline-block; padding: 4px 10px; border-radius: 4px; color: #fff; font-weight: bold; font-size: 11px; margin-top: 6px; }
		.info-grid { display: flex; justify-content: space-between; margin: 25px 0; font-size: 13px; line-height: 1.6; }
		.info-box { width: 48%%; }
		.info-title { font-weight: bold; color: #374151; border-bottom: 1px solid #e5e7eb; padding-bottom: 4px; margin-bottom: 8px; }
		table { width: 100%%; border-collapse: collapse; margin-top: 15px; font-size: 13px; }
		th { background: #f3f4f6; color: #374151; padding: 10px; text-align: left; border-bottom: 2px solid #e5e7eb; }
		.footer-notes { margin-top: 30px; font-size: 12px; color: #6b7280; border-top: 1px solid #e5e7eb; padding-top: 15px; }
		.signatures { display: flex; justify-content: space-between; margin-top: 50px; text-align: center; font-size: 12px; }
		.sign-box { width: 180px; }
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
			<button onclick="window.print()" style="background:#2563eb; color:#fff; border:none; padding:10px 20px; border-radius:6px; cursor:pointer; font-weight:bold; font-size:14px;">🖨️ Cetak Surat Jalan</button>
		</div>

		<div class="header">
			<div>
				<div class="company-name">🏢 PT ODOO-GO INDONESIA</div>
				<div class="company-info">
					Logistics & Central Warehouse (WH/Stock)<br>
					Kawasan Pergudangan Logistik Blok C-4, Jakarta Barat<br>
					Telp: +62 21 555-9999 | Email: warehouse@odoogolang.co.id
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
				Status Logistik: <strong>%s</strong><br>
				Lokasi Asal: <strong>%s</strong><br>
				Lokasi Tujuan: <strong>%s</strong>
			</div>
			<div class="info-box">
				<div class="info-title">Detail Dokumen Pengiriman:</div>
				Referensi Order: <strong>%s</strong><br>
				Tanggal Dijadwalkan: <strong>%s</strong><br>
				Tanggal Realisasi: <strong>%s</strong><br>
				Instruksi: Segera periksa keutuhan fisik barang saat serah terima.
			</div>
		</div>

		<table>
			<thead>
				<tr>
					<th style="text-align:center; width:30px;">#</th>
					<th>Nama / Deskripsi Barang</th>
					<th style="text-align:center; width:100px;">Permintaan</th>
					<th style="text-align:center; width:100px;">Terkirim</th>
					<th style="text-align:center; width:90px;">Status</th>
				</tr>
			</thead>
			<tbody>
				%s
			</tbody>
		</table>

		<div class="footer-notes">
			<strong>Catatan Operasional Gudang:</strong><br>
			1. Barang telah dicek kualitas dan kuantitasnya sebelum meninggalkan area gudang.<br>
			2. Segala bentuk komplain kekurangan/kerusakan wajib menyertakan surat jalan bertanda tangan ini.<br>
			Catatan Khusus: %s
		</div>

		<div class="signatures">
			<div class="sign-box">
				Bagian Gudang,<br>
				<div class="sign-line">( Petugas Dispatch )</div>
			</div>
			<div class="sign-box">
				Kurir / Sopir Pengantar,<br>
				<div class="sign-line">( Ekspedisi / Driver )</div>
			</div>
			<div class="sign-box">
				Penerima Barang,<br>
				<div class="sign-line">%s</div>
			</div>
		</div>
	</div>
</body>
</html>`,
		docTitle, p.Name,
		docTitle, p.Name, statusColor, statusText,
		partnerTitle, partnerName, actionText, p.Location.CompleteName, p.LocationDest.CompleteName,
		p.Origin, p.ScheduledDate, p.DateDone,
		linesHTML.String(),
		p.Note,
		partnerName,
	)

	return html
}
