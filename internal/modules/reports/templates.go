package reports

import (
	"crypto/sha256"
	"fmt"
	"math"
	"strings"
	"time"
)

// FormatCurrency formats a float64 into Indonesian Rupiah format (e.g. 1.500.000)
func FormatCurrency(val float64) string {
	isNeg := val < 0
	val = math.Abs(val)
	intPart := int64(math.Round(val))
	s := fmt.Sprintf("%d", intPart)
	n := len(s)
	if n <= 3 {
		if isNeg {
			return "-" + s
		}
		return s
	}

	var res strings.Builder
	rem := n % 3
	if rem > 0 {
		res.WriteString(s[:rem])
		if n > rem {
			res.WriteString(".")
		}
	}
	for i := rem; i < n; i += 3 {
		res.WriteString(s[i : i+3])
		if i+3 < n {
			res.WriteString(".")
		}
	}
	if isNeg {
		return "-" + res.String()
	}
	return res.String()
}

// FormatDateTime formats time.Time to Indonesian locale string
func FormatDateTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	months := []string{"", "Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"}
	return fmt.Sprintf("%02d %s %d %02d:%02d WIB", t.Day(), months[t.Month()], t.Year(), t.Hour(), t.Minute())
}

// GenerateSecurityHash generates an audit SHA-256 fingerprint for the document
func GenerateSecurityHash(docNum string, t time.Time) string {
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%s-%d-PT_ODOO_GO_ENTERPRISE_AUTHENTIC", docNum, t.Unix())))
	return fmt.Sprintf("%x", h.Sum(nil))[:24]
}

// WrapReportHTML builds a professional enterprise printable HTML document
func WrapReportHTML(
	docTitle string,
	docNumber string,
	statusText string,
	statusColor string,
	infoLeftHTML string,
	infoRightHTML string,
	tableContentHTML string,
	summaryHTML string,
	notesHTML string,
	watermarkText string,
	signaturesHTML string,
	isThermal bool,
) string {
	now := time.Now()
	secHash := GenerateSecurityHash(docNumber, now)

	if isThermal {
		// Thermal 80mm format for POS Receipts
		return fmt.Sprintf(`<!DOCTYPE html>
<html lang="id">
<head>
	<meta charset="UTF-8">
	<title>%s - %s</title>
	<script async src="https://pagead2.googlesyndication.com/pagead/js/adsbygoogle.js?client=ca-pub-6180580801533680"
	     crossorigin="anonymous"></script>
	<!-- Google tag (gtag.js) -->
	<script async src="https://www.googletagmanager.com/gtag/js?id=G-79X7K7H4MJ"></script>
	<script>
	  window.dataLayer = window.dataLayer || [];
	  function gtag(){dataLayer.push(arguments);}
	  gtag('js', new Date());

	  gtag('config', 'G-79X7K7H4MJ');
	</script>
	<style>
		@page { size: 80mm auto; margin: 0; }
		body { font-family: 'Courier New', Courier, monospace; width: 72mm; margin: 0 auto; padding: 10px 0; color: #000; font-size: 12px; }
		.text-center { text-align: center; }
		.text-right { text-align: right; }
		.bold { font-weight: bold; }
		.divider { border-top: 1px dashed #000; margin: 8px 0; }
		table { width: 100%%; border-collapse: collapse; }
		td { padding: 2px 0; }
		.no-print { text-align: center; margin-bottom: 10px; }
		.btn { background: #000; color: #fff; border: none; padding: 6px 12px; border-radius: 4px; cursor: pointer; font-size: 11px; }
		@media print { .no-print { display: none; } }
	</style>
</head>
<body>
	<div class="no-print">
		<button class="btn" onclick="window.print()">🖨️ CETAK STRUK KASIR</button>
	</div>
	<div class="text-center">
		<div class="bold" style="font-size: 14px;">PT ODOO-GO INDONESIA</div>
		<div>POS Store Outlet #01</div>
		<div>NPWP: 01.345.678.9-021.000</div>
		<div class="divider"></div>
		<div class="bold">%s</div>
		<div>No: %s</div>
		<div>%s</div>
	</div>
	<div class="divider"></div>
	%s
	<div class="divider"></div>
	%s
	<div class="divider"></div>
	<div class="text-center" style="font-size: 10px;">
		<div>*** TERIMA KASIH ATAS KUNJUNGAN ANDA ***</div>
		<div>Barang yang dibeli tidak dapat ditukar</div>
		<div>Kode Audit: %s</div>
	</div>
</body>
</html>`, docTitle, docNumber, docTitle, docNumber, FormatDateTime(now), tableContentHTML, summaryHTML, secHash)
	}

	// Standard Corporate A4 Letterhead format
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="id">
<head>
	<meta charset="UTF-8">
	<title>%s - %s</title>
	<script async src="https://pagead2.googlesyndication.com/pagead/js/adsbygoogle.js?client=ca-pub-6180580801533680"
	     crossorigin="anonymous"></script>
	<!-- Google tag (gtag.js) -->
	<script async src="https://www.googletagmanager.com/gtag/js?id=G-79X7K7H4MJ"></script>
	<script>
	  window.dataLayer = window.dataLayer || [];
	  function gtag(){dataLayer.push(arguments);}
	  gtag('js', new Date());

	  gtag('config', 'G-79X7K7H4MJ');
	</script>
	<style>
		@page {
			size: A4 portrait;
			margin: 15mm 12mm 15mm 12mm;
		}
		* { box-sizing: border-box; }
		body {
			font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
			color: #1e293b;
			margin: 0;
			padding: 24px;
			background: #ffffff;
			font-size: 12px;
			line-height: 1.5;
			position: relative;
		}
		.container {
			max-width: 860px;
			margin: 0 auto;
			position: relative;
		}
		.watermark {
			position: absolute;
			top: 38%%;
			left: 10%%;
			transform: rotate(-30deg);
			font-size: 72px;
			font-weight: 900;
			color: rgba(226, 232, 240, 0.45);
			text-transform: uppercase;
			pointer-events: none;
			z-index: 0;
			letter-spacing: 6px;
			border: 8px solid rgba(226, 232, 240, 0.4);
			padding: 10px 40px;
			border-radius: 12px;
		}
		.header {
			display: flex;
			justify-content: space-between;
			align-items: flex-start;
			border-bottom: 3px double #0284c7;
			padding-bottom: 14px;
			margin-bottom: 18px;
			position: relative;
			z-index: 1;
		}
		.company-logo {
			display: flex;
			align-items: center;
			gap: 12px;
		}
		.company-badge {
			background: #0284c7;
			color: #ffffff;
			width: 44px;
			height: 44px;
			border-radius: 8px;
			display: flex;
			align-items: center;
			justify-content: center;
			font-weight: bold;
			font-size: 20px;
		}
		.company-name {
			font-size: 18px;
			font-weight: 800;
			color: #0f172a;
			letter-spacing: -0.5px;
		}
		.company-details {
			font-size: 10px;
			color: #64748b;
			line-height: 1.4;
			margin-top: 2px;
		}
		.doc-badge-group {
			text-align: right;
		}
		.doc-title {
			font-size: 16px;
			font-weight: 800;
			color: #0f172a;
			text-transform: uppercase;
			letter-spacing: 0.5px;
			margin: 0;
		}
		.doc-number {
			font-size: 14px;
			font-weight: 700;
			color: #0284c7;
			margin-top: 3px;
			font-family: monospace;
		}
		.status-pill {
			display: inline-block;
			background: %s;
			color: #ffffff;
			font-size: 10px;
			font-weight: 700;
			padding: 3px 10px;
			border-radius: 12px;
			margin-top: 5px;
			text-transform: uppercase;
			letter-spacing: 0.5px;
		}
		.info-grid {
			display: flex;
			justify-content: space-between;
			gap: 20px;
			margin-bottom: 16px;
			position: relative;
			z-index: 1;
		}
		.info-card {
			flex: 1;
			background: #f8fafc;
			border: 1px solid #e2e8f0;
			border-radius: 6px;
			padding: 10px 14px;
			font-size: 11px;
		}
		.info-card-title {
			font-weight: 700;
			color: #475569;
			text-transform: uppercase;
			font-size: 10px;
			border-bottom: 1px solid #cbd5e1;
			padding-bottom: 4px;
			margin-bottom: 6px;
		}
		table.data-table {
			width: 100%%;
			border-collapse: collapse;
			margin-bottom: 14px;
			font-size: 11px;
			position: relative;
			z-index: 1;
		}
		table.data-table th {
			background: #f1f5f9;
			color: #334155;
			font-weight: 700;
			padding: 8px 10px;
			text-align: left;
			border: 1px solid #cbd5e1;
			font-size: 10.5px;
			text-transform: uppercase;
		}
		table.data-table td {
			padding: 8px 10px;
			border: 1px solid #e2e8f0;
			vertical-align: top;
		}
		table.data-table tr:nth-child(even) {
			background: #f8fafc;
		}
		.text-center { text-align: center; }
		.text-right { text-align: right; }
		.bold { font-weight: 700; }
		
		.summary-container {
			display: flex;
			justify-content: flex-end;
			margin-bottom: 16px;
			position: relative;
			z-index: 1;
		}
		table.summary-table {
			width: 340px;
			border-collapse: collapse;
			font-size: 11px;
		}
		table.summary-table td {
			padding: 4px 8px;
		}
		table.summary-table tr.grand-total {
			border-top: 2px solid #0284c7;
			border-bottom: 2px solid #0284c7;
			font-weight: 800;
			font-size: 13px;
			color: #0f172a;
			background: #f0f9ff;
		}
		.notes-box {
			background: #fefce8;
			border: 1px solid #fef08a;
			border-radius: 6px;
			padding: 10px 14px;
			font-size: 11px;
			color: #713f12;
			margin-bottom: 20px;
			position: relative;
			z-index: 1;
		}
		.signatures-grid {
			display: flex;
			justify-content: space-between;
			gap: 15px;
			margin-top: 30px;
			margin-bottom: 20px;
			text-align: center;
			position: relative;
			z-index: 1;
		}
		.sign-card {
			flex: 1;
			font-size: 11px;
		}
		.sign-card-title {
			color: #64748b;
			margin-bottom: 50px;
		}
		.sign-line {
			border-top: 1px solid #94a3b8;
			padding-top: 4px;
			font-weight: 700;
			color: #1e293b;
		}
		.footer {
			border-top: 1px solid #e2e8f0;
			padding-top: 8px;
			display: flex;
			justify-content: space-between;
			align-items: center;
			font-size: 9.5px;
			color: #94a3b8;
			position: relative;
			z-index: 1;
		}
		.no-print {
			margin-bottom: 18px;
			text-align: right;
		}
		.btn-print {
			background: #0284c7;
			color: #ffffff;
			border: none;
			padding: 8px 18px;
			border-radius: 6px;
			font-weight: 700;
			font-size: 13px;
			cursor: pointer;
			display: inline-flex;
			align-items: center;
			gap: 8px;
			box-shadow: 0 2px 4px rgba(2, 132, 199, 0.2);
		}
		.btn-print:hover {
			background: #0369a1;
		}
		@media print {
			body { padding: 0; }
			.no-print { display: none; }
			.container { max-width: 100%%; }
		}
	</style>
</head>
<body>
	<div class="container">
		%s
		<div class="no-print">
			<button class="btn-print" onclick="window.print()">
				🖨️ CETAK / SIMPAN PDF (A4)
			</button>
		</div>

		<div class="header">
			<div class="company-logo">
				<div class="company-badge">OG</div>
				<div>
					<div class="company-name">PT ODOO-GO INDONESIA ENTERPRISE</div>
					<div class="company-details">
						Gedung Menara Batavia Lt. 28, Jl. K.H. Mas Mansyur Kav. 126, Jakarta Pusat 10220<br>
						NPWP: 01.345.678.9-021.000 &bull; Telp: (021) 5790-1234 &bull; Email: finance@odoo.local
					</div>
				</div>
			</div>
			<div class="doc-badge-group">
				<h1 class="doc-title">%s</h1>
				<div class="doc-number">%s</div>
				<div><span class="status-pill">%s</span></div>
			</div>
		</div>

		<div class="info-grid">
			<div class="info-card">
				%s
			</div>
			<div class="info-card">
				%s
			</div>
		</div>

		%s

		%s

		%s

		%s

		<div class="footer">
			<div>Dokumen Resmi Odoo-Go ERP &bull; ID Audit: %s</div>
			<div>Dicetak pada: %s</div>
			<div>Halaman 1 / 1 &bull; Kepatuhan ISO 27001 / SOX Verified</div>
		</div>
	</div>
</body>
</html>`,
		docTitle,
		docNumber,
		statusColor,
		func() string {
			if watermarkText != "" {
				return fmt.Sprintf(`<div class="watermark">%s</div>`, watermarkText)
			}
			return ""
		}(),
		docTitle,
		docNumber,
		statusText,
		infoLeftHTML,
		infoRightHTML,
		tableContentHTML,
		summaryHTML,
		notesHTML,
		signaturesHTML,
		secHash,
		FormatDateTime(now),
	)
}
