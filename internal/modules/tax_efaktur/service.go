package tax_efaktur

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"erp-golang/internal/core/ai"
	"erp-golang/internal/modules/account"
	"erp-golang/internal/modules/chatter"
	"gorm.io/gorm"
)

type Service struct {
	db         *gorm.DB
	repo       *Repository
	accountSvc *account.Service
	chatterSvc *chatter.Service
	aiClient   *ai.Client
}

func NewService(db *gorm.DB, repo *Repository, accountSvc *account.Service, chatterSvc *chatter.Service, aiClient *ai.Client) *Service {
	return &Service{
		db:         db,
		repo:       repo,
		accountSvc: accountSvc,
		chatterSvc: chatterSvc,
		aiClient:   aiClient,
	}
}

// CreateNSFP stores a new range of Nomor Seri Faktur Pajak
func (s *Service) CreateNSFP(req CreateNSFPRequest, companyID uint) (*TaxNSFP, error) {
	if req.StartNumber == "" || req.EndNumber == "" {
		return nil, errors.New("nomor awal dan nomor akhir NSFP wajib diisi")
	}
	if len(req.StartNumber) != 8 || len(req.EndNumber) != 8 {
		return nil, errors.New("nomor seri faktur pajak harus berformat 8 digit (e.g. 00000001)")
	}

	startInt, err1 := strconv.ParseInt(req.StartNumber, 10, 64)
	endInt, err2 := strconv.ParseInt(req.EndNumber, 10, 64)
	if err1 != nil || err2 != nil || endInt < startInt {
		return nil, errors.New("nomor akhir NSFP harus lebih besar atau sama dengan nomor awal")
	}

	prefix := req.Prefix
	if prefix == "" {
		prefix = "010"
	}
	year := req.Year
	if year == 0 {
		year = time.Now().Year()
	}

	totalQuota := int(endInt - startInt + 1)

	nsfp := &TaxNSFP{
		Prefix:         prefix,
		Year:           year,
		StartNumber:    req.StartNumber,
		EndNumber:      req.EndNumber,
		CurrentNumber:  req.StartNumber,
		TotalQuota:     totalQuota,
		UsedQuota:      0,
		RemainingQuota: totalQuota,
		State:          "active",
		CompanyID:      companyID,
	}

	if err := s.repo.CreateNSFP(nsfp); err != nil {
		return nil, err
	}

	return nsfp, nil
}

// ListNSFP returns all NSFP ranges
func (s *Service) ListNSFP(companyID uint) ([]TaxNSFP, error) {
	return s.repo.ListNSFP(companyID)
}

// GenerateOutTaxInvoiceFromMove generates an e-Faktur Pajak Keluaran from a posted Customer Invoice
func (s *Service) GenerateOutTaxInvoiceFromMove(req GenerateTaxInvoiceRequest, companyID uint, userID uint) (*TaxInvoice, error) {
	if req.MoveID == 0 {
		return nil, errors.New("move_id faktur komersial wajib diisi")
	}

	// 1. Fetch Move
	move, err := s.accountSvc.GetMove(req.MoveID, companyID)
	if err != nil {
		return nil, fmt.Errorf("faktur komersial tidak ditemukan: %v", err)
	}
	if move.MoveType != "out_invoice" {
		return nil, errors.New("faktur pajak keluaran hanya dapat diterbitkan dari Customer Invoice (out_invoice)")
	}
	if move.State != "posted" {
		return nil, errors.New("faktur komersial harus dalam status 'posted' sebelum dapat diterbitkan Faktur Pajak")
	}

	// Check existing
	existing, _ := s.repo.GetTaxInvoiceByMoveID(req.MoveID, companyID)
	if existing != nil && existing.State != "cancelled" {
		return nil, fmt.Errorf("faktur pajak telah diterbitkan sebelumnya untuk dokumen ini (NSFP: %s)", existing.NSFPNumber)
	}

	// 2. Determine Period & Year
	invDate := move.InvoiceDate
	if invDate == "" {
		invDate = time.Now().Format("2006-01-02")
	}
	parsedDate, _ := time.Parse("2006-01-02", invDate)
	taxPeriod := req.TaxPeriod
	if taxPeriod == 0 {
		taxPeriod = int(parsedDate.Month())
	}
	taxYear := req.TaxYear
	if taxYear == 0 {
		taxYear = parsedDate.Year()
	}

	// 3. Allocate Next NSFP Number
	nsfpNumber := req.CustomNSFP
	if nsfpNumber == "" {
		prefix := "010" // Penyerahan Dalam Negeri Standar
		allocated, err := s.repo.AllocateNextNSFP(prefix, taxYear, companyID)
		if err != nil {
			return nil, err
		}
		nsfpNumber = allocated
	}

	// 4. Partner Data
	partnerName := "Pelanggan Umum"
	npwp := "00.000.000.0-000.000"
	address := "Indonesia"

	if move.Partner.Name != "" {
		partnerName = move.Partner.Name
	}
	if move.Partner.TaxID != "" {
		npwp = cleanNPWP(move.Partner.TaxID)
	}
	if move.Partner.Street != "" {
		address = move.Partner.Street
	}

	// 5. Calculations
	dpp := move.AmountUntaxed
	if dpp <= 0 {
		dpp = move.AmountTotal
	}
	taxRate := req.TaxRatePPN
	if taxRate <= 0 {
		taxRate = 11.0 // PPN Indonesia UU HPP
	}
	ppn := math.Round(dpp*taxRate) / 100.0

	var pphAmount float64
	if req.PPhType != "" && req.PPhType != "none" && req.PPhRate > 0 {
		pphAmount = math.Round(dpp*req.PPhRate) / 100.0
	}

	totalAmount := dpp + ppn - pphAmount

	// DJP verification URL QR Code payload
	qrURL := fmt.Sprintf("https://efaktur.pajak.go.id/validasi?nsfp=%s&npwp=%s&dpp=%.0f&ppn=%.0f&date=%s",
		nsfpNumber, cleanNPWP(npwp), dpp, ppn, invDate)

	taxInv := &TaxInvoice{
		TaxInvoiceType: "out_tax",
		NSFPNumber:     nsfpNumber,
		InvoiceDate:    invDate,
		TaxPeriod:      taxPeriod,
		TaxYear:        taxYear,
		PartnerID:      move.PartnerID,
		PartnerName:    partnerName,
		NPWP:           npwp,
		Address:        address,
		MoveID:         &move.ID,
		DPP:            dpp,
		TaxRatePPN:     taxRate,
		PPN:            ppn,
		PPNBM:          0,
		PPhType:        req.PPhType,
		PPhRate:        req.PPhRate,
		PPhAmount:      pphAmount,
		TotalAmount:    totalAmount,
		State:          "generated",
		QRURL:          qrURL,
		Notes:          req.Notes,
		CompanyID:      companyID,
	}

	// Detail lines from move lines (sales lines)
	for _, l := range move.LineIDs {
		if l.Credit > 0 || l.Debit == 0 { // Income/Sales line
			if strings.Contains(strings.ToLower(l.Name), "ppn") || strings.Contains(strings.ToLower(l.Name), "pajak") {
				continue
			}
			lineDPP := l.Credit
			if lineDPP == 0 {
				lineDPP = l.Debit
			}
			linePPN := math.Round(lineDPP*taxRate) / 100.0
			taxInv.Lines = append(taxInv.Lines, TaxInvoiceLine{
				ProductName: l.Name,
				PriceUnit:   lineDPP,
				Quantity:    1,
				TotalPrice:  lineDPP,
				DPP:         lineDPP,
				PPN:         linePPN,
			})
		}
	}
	if len(taxInv.Lines) == 0 {
		taxInv.Lines = append(taxInv.Lines, TaxInvoiceLine{
			ProductName: fmt.Sprintf("Penyerahan BKP/JKP Faktur %s", move.Name),
			PriceUnit:   dpp,
			Quantity:    1,
			TotalPrice:  dpp,
			DPP:         dpp,
			PPN:         ppn,
		})
	}

	if err := s.repo.CreateTaxInvoice(taxInv); err != nil {
		return nil, fmt.Errorf("gagal membuat faktur pajak: %v", err)
	}

	// Chatter Audit Log
	if s.chatterSvc != nil {
		_ = s.chatterSvc.LogTracking(
			"account.tax.invoice",
			taxInv.ID,
			userID,
			"Sistem Pajak e-Faktur",
			fmt.Sprintf("Faktur Pajak Keluaran resmi diterbitkan: NSFP '%s' untuk Faktur Komersial '%s' (DPP: Rp %.0f, PPN %.0f%%: Rp %.0f)",
				taxInv.NSFPNumber, move.Name, taxInv.DPP, taxInv.TaxRatePPN, taxInv.PPN),
			"faktur_pajak_diterbitkan",
			companyID,
		)
	}

	return taxInv, nil
}

// RecordInTaxInvoiceFromMove records a vendor's tax invoice (e-Faktur Masukan)
func (s *Service) RecordInTaxInvoiceFromMove(req GenerateTaxInvoiceRequest, companyID uint, userID uint) (*TaxInvoice, error) {
	if req.MoveID == 0 {
		return nil, errors.New("move_id tagihan vendor wajib diisi")
	}
	if req.CustomNSFP == "" {
		return nil, errors.New("nomor seri faktur pajak (NSFP) dari vendor rekanan wajib diisi")
	}

	move, err := s.accountSvc.GetMove(req.MoveID, companyID)
	if err != nil {
		return nil, fmt.Errorf("tagihan vendor tidak ditemukan: %v", err)
	}
	if move.MoveType != "in_invoice" {
		return nil, errors.New("faktur pajak masukan hanya dapat dicatat dari Vendor Bill (in_invoice)")
	}

	invDate := move.InvoiceDate
	if invDate == "" {
		invDate = time.Now().Format("2006-01-02")
	}
	parsedDate, _ := time.Parse("2006-01-02", invDate)
	taxPeriod := req.TaxPeriod
	if taxPeriod == 0 {
		taxPeriod = int(parsedDate.Month())
	}
	taxYear := req.TaxYear
	if taxYear == 0 {
		taxYear = parsedDate.Year()
	}

	partnerName := "Vendor Rekanan"
	npwp := "00.000.000.0-000.000"
	address := "Indonesia"

	if move.Partner.Name != "" {
		partnerName = move.Partner.Name
	}
	if move.Partner.TaxID != "" {
		npwp = cleanNPWP(move.Partner.TaxID)
	}
	if move.Partner.Street != "" {
		address = move.Partner.Street
	}

	dpp := move.AmountUntaxed
	if dpp <= 0 {
		dpp = move.AmountTotal
	}
	taxRate := req.TaxRatePPN
	if taxRate <= 0 {
		taxRate = 11.0
	}
	ppn := math.Round(dpp*taxRate) / 100.0

	var pphAmount float64
	if req.PPhType != "" && req.PPhType != "none" && req.PPhRate > 0 {
		pphAmount = math.Round(dpp*req.PPhRate) / 100.0
	}
	totalAmount := dpp + ppn - pphAmount

	taxInv := &TaxInvoice{
		TaxInvoiceType: "in_tax",
		NSFPNumber:     cleanNSFP(req.CustomNSFP),
		InvoiceDate:    invDate,
		TaxPeriod:      taxPeriod,
		TaxYear:        taxYear,
		PartnerID:      move.PartnerID,
		PartnerName:    partnerName,
		NPWP:           npwp,
		Address:        address,
		MoveID:         &move.ID,
		DPP:            dpp,
		TaxRatePPN:     taxRate,
		PPN:            ppn,
		PPNBM:          0,
		PPhType:        req.PPhType,
		PPhRate:        req.PPhRate,
		PPhAmount:      pphAmount,
		TotalAmount:    totalAmount,
		State:          "generated",
		Notes:          req.Notes,
		CompanyID:      companyID,
	}

	taxInv.Lines = append(taxInv.Lines, TaxInvoiceLine{
		ProductName: fmt.Sprintf("Perolehan BKP/JKP Tagihan Vendor %s", move.Name),
		PriceUnit:   dpp,
		Quantity:    1,
		TotalPrice:  dpp,
		DPP:         dpp,
		PPN:         ppn,
	})

	if err := s.repo.CreateTaxInvoice(taxInv); err != nil {
		return nil, fmt.Errorf("gagal mencatat faktur pajak masukan: %v", err)
	}

	if s.chatterSvc != nil {
		_ = s.chatterSvc.LogTracking(
			"account.tax.invoice",
			taxInv.ID,
			userID,
			"Sistem Pajak e-Faktur",
			fmt.Sprintf("Faktur Pajak Masukan dicatat untuk pengkreditan PPN: NSFP '%s' dari Vendor '%s' (DPP: Rp %.0f, PPN: Rp %.0f)",
				taxInv.NSFPNumber, taxInv.PartnerName, taxInv.DPP, taxInv.PPN),
			"faktur_masukan_dicatat",
			companyID,
		)
	}

	return taxInv, nil
}

// GetTaxInvoiceByID retrieves a single tax invoice
func (s *Service) GetTaxInvoiceByID(id uint, companyID uint) (*TaxInvoice, error) {
	return s.repo.GetTaxInvoiceByID(id, companyID)
}

// ListTaxInvoices retrieves invoices by filter
func (s *Service) ListTaxInvoices(taxType string, period int, year int, state string, companyID uint) ([]TaxInvoice, error) {
	return s.repo.ListTaxInvoices(taxType, period, year, state, companyID)
}

// UpdateState modifies invoice state (reported, cancelled)
func (s *Service) UpdateState(id uint, state string, companyID uint, userID uint) (*TaxInvoice, error) {
	inv, err := s.repo.GetTaxInvoiceByID(id, companyID)
	if err != nil {
		return nil, err
	}

	inv.State = state
	if err := s.db.Save(inv).Error; err != nil {
		return nil, err
	}

	if s.chatterSvc != nil {
		_ = s.chatterSvc.LogTracking(
			"account.tax.invoice",
			inv.ID,
			userID,
			"Sistem Pajak e-Faktur",
			fmt.Sprintf("Status Faktur Pajak '%s' diperbarui menjadi '%s'", inv.NSFPNumber, state),
			"status_faktur_diubah",
			companyID,
		)
	}

	return inv, nil
}

// GetStats returns summary analytics
func (s *Service) GetStats(companyID uint) (*TaxStats, error) {
	return s.repo.GetStats(companyID)
}

// GetSPTPeriodSummary computes period 1111 SPT summary
func (s *Service) GetSPTPeriodSummary(period int, year int, companyID uint) (*SPTPeriodSummary, error) {
	return s.repo.GetSPTPeriodSummary(period, year, companyID)
}

// ExportEFakturCSV exports official DJP e-Faktur format CSV
func (s *Service) ExportEFakturCSV(taxType string, period int, year int, companyID uint) (string, error) {
	invoices, err := s.repo.ListTaxInvoices(taxType, period, year, "generated", companyID)
	if err != nil {
		return "", err
	}
	if len(invoices) == 0 {
		// Include reported as well
		invoices, err = s.repo.ListTaxInvoices(taxType, period, year, "", companyID)
		if err != nil {
			return "", err
		}
	}

	var sb strings.Builder

	// Standard DJP Header
	sb.WriteString("FK,KD_JENIS_TRANSAKSI,FG_PENGGANTI,NOMOR_FAKTUR,MASA_PAJAK,TAHUN_PAJAK,TANGGAL_FAKTUR,NPWP,NAMA,ALAMAT_LENGKAP,JUMLAH_DPP,JUMLAH_PPN,JUMLAH_PPNBM,ID_KETERANGAN_TAMBAHAN,FG_UANG_MUKA,UANG_MUKA_DPP,UANG_MUKA_PPN,UANG_MUKA_PPNBM,REFERENSI,KODE_DOKUMEN_PENDUKUNG\n")
	sb.WriteString("LT,NPWP,NAMA,JALAN,BLOK,NOMOR,RT,RW,KECAMATAN,KELURAHAN,KABUPATEN,PROPINSI,KODE_POS,NOMOR_TELEPON\n")
	sb.WriteString("OF,KODE_OBJEK,NAMA,HARGA_SATUAN,JUMLAH_BARANG,HARGA_TOTAL,DISKON,DPP,PPN,TARIF_PPNBM,PPNBM\n")

	for _, inv := range invoices {
		if inv.State == "cancelled" {
			continue
		}

		cleanNum := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(inv.NSFPNumber, ".", ""), "-", ""), " ", "")
		rawNPWP := cleanDigits(inv.NPWP)
		if len(rawNPWP) < 15 {
			rawNPWP = fmt.Sprintf("%015s", rawNPWP)
		}

		kdTransaksi := "01"
		if len(cleanNum) >= 2 {
			kdTransaksi = cleanNum[:2]
		}
		fgPengganti := "0"
		if inv.IsReplacement {
			fgPengganti = "1"
		}

		// Row FK
		sb.WriteString(fmt.Sprintf("FK,%s,%s,%s,%d,%d,%s,%s,\"%s\",\"%s\",%.0f,%.0f,%.0f,0,0,0,0,0,\"%s\",\"\"\n",
			kdTransaksi,
			fgPengganti,
			cleanNum,
			inv.TaxPeriod,
			inv.TaxYear,
			formatDateIndo(inv.InvoiceDate),
			rawNPWP,
			escapeCSV(inv.PartnerName),
			escapeCSV(inv.Address),
			inv.DPP,
			inv.PPN,
			inv.PPNBM,
			inv.NSFPNumber,
		))

		// Detail Rows OF
		for _, l := range inv.Lines {
			sb.WriteString(fmt.Sprintf("OF,\"\",\"%s\",%.0f,%.2f,%.0f,%.0f,%.0f,%.0f,0,0\n",
				escapeCSV(l.ProductName),
				l.PriceUnit,
				l.Quantity,
				l.TotalPrice,
				l.Discount,
				l.DPP,
				l.PPN,
			))
		}
	}

	return sb.String(), nil
}

// TaxComplianceAIAudit runs an audit on fiscal compliance
func (s *Service) TaxComplianceAIAudit(period int, year int, companyID uint) (*TaxAuditReport, error) {
	invoices, err := s.repo.ListTaxInvoices("", period, year, "", companyID)
	if err != nil {
		return nil, err
	}

	spt, _ := s.repo.GetSPTPeriodSummary(period, year, companyID)

	var invalidNPWP int
	var dateDiscrepancies int
	totalInvoices := len(invoices)

	for _, inv := range invoices {
		clean := cleanDigits(inv.NPWP)
		// Indonesian NPWP must be 15 digits (format lama) or 16 digits (NIK baru UU HPP)
		if len(clean) != 15 && len(clean) != 16 && clean != "000000000000000" {
			invalidNPWP++
		}

		parsed, err := time.Parse("2006-01-02", inv.InvoiceDate)
		if err == nil {
			if int(parsed.Month()) != inv.TaxPeriod || parsed.Year() != inv.TaxYear {
				dateDiscrepancies++
			}
		}
	}

	complianceScore := 100.0
	if totalInvoices > 0 {
		penalty := float64(invalidNPWP)*15.0 + float64(dateDiscrepancies)*10.0
		complianceScore = math.Max(30.0, 100.0-penalty)
	}

	riskLevel := "low"
	if complianceScore < 70 {
		riskLevel = "high"
	} else if complianceScore < 85 {
		riskLevel = "medium"
	}
	if invalidNPWP > 3 {
		riskLevel = "critical"
	}

	netTax := 0.0
	if spt != nil {
		netTax = spt.PPNKurangLebihBayar
	}

	prompt := fmt.Sprintf(`Lakukan audit kepatuhan perpajakan e-Faktur PPN Masa %d/%d untuk perusahaan di Indonesia.
Data Masa Pajak:
- Total Faktur Diperiksa: %d dokumen
- NPWP Tidak Valid / Format Salah: %d dokumen
- Anomali Tanggal Faktur Pajak vs Masa: %d dokumen
- Total Penyerahan PPN Keluaran: Rp %.0f
- Total Pengkreditan PPN Masukan: Rp %.0f
- Posisi Fiskal Akhir: %s (Rp %.0f)
- Skor Kepatuhan: %.1f%%
- Tingkat Risiko: %s

Analisis risiko denda keterlambatan Pasal 14 (4) KUP dan berikan rekomendasi pelaporan SPT Masa 1111 sebelum akhir bulan berikutnya.`,
		period, year, totalInvoices, invalidNPWP, dateDiscrepancies,
		spt.PPNKeluaranDipungut, spt.PPNMasukanDikreditkan, spt.StatusFiskal, math.Abs(netTax), complianceScore, riskLevel)

	report := &TaxAuditReport{
		TaxPeriod:            period,
		TaxYear:              year,
		ComplianceScore:      complianceScore,
		RiskLevel:            riskLevel,
		TotalInvoicesAudited: totalInvoices,
		InvalidNPWPCount:     invalidNPWP,
		DateDiscrepancies:    dateDiscrepancies,
		NetTaxPayable:        netTax,
		Engine:               "local_heuristic_engine",
	}

	if s.aiClient != nil && s.aiClient.IsCloudflareConfigured() {
		systemPrompt := "You are a senior Indonesian Tax Consultant & Auditor. Return JSON with risk_level, executive_summary, and action_recommendations."
		if aiResp, err := s.aiClient.CallRaw(prompt, systemPrompt); err == nil && aiResp != "" {
			var parsed struct {
				RiskLevel        string   `json:"risk_level"`
				ExecutiveSummary string   `json:"executive_summary"`
				Recommendations  []string `json:"action_recommendations"`
			}
			jsonStr := strings.TrimSpace(aiResp)
			if idx := strings.Index(jsonStr, "{"); idx != -1 {
				if endIdx := strings.LastIndex(jsonStr, "}"); endIdx != -1 {
					jsonStr = jsonStr[idx : endIdx+1]
				}
			}
			if err := json.Unmarshal([]byte(jsonStr), &parsed); err == nil {
				if parsed.ExecutiveSummary != "" {
					report.ExecutiveSummary = parsed.ExecutiveSummary
				}
				if len(parsed.Recommendations) > 0 {
					report.ActionRecommendations = parsed.Recommendations
				}
				if parsed.RiskLevel != "" {
					report.RiskLevel = parsed.RiskLevel
				}
				report.Engine = "@cf/meta/llama-3.1-8b-instruct"
			}
		}
	}

	if report.ExecutiveSummary == "" {
		monthNames := []string{"", "Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"}
		monthName := "Bulan Ini"
		if period >= 1 && period <= 12 {
			monthName = monthNames[period]
		}

		if netTax > 0 {
			report.ExecutiveSummary = fmt.Sprintf("Audit Pajak Masa %s %d: Posisi fiskal berada pada status Kurang Bayar sebesar Rp %.0f (PPN Keluaran Rp %.0f > PPN Masukan Rp %.0f). Skor kepatuhan %.1f%% (%s Risk).",
				monthName, year, netTax, spt.PPNKeluaranDipungut, spt.PPNMasukanDikreditkan, complianceScore, strings.ToUpper(riskLevel))
		} else if netTax < 0 {
			report.ExecutiveSummary = fmt.Sprintf("Audit Pajak Masa %s %d: Posisi fiskal berada pada status Lebih Bayar sebesar Rp %.0f. PPN Masukan dapat dikompensasikan ke masa pajak berikutnya atau diajukan restitusi.",
				monthName, year, math.Abs(netTax))
		} else {
			report.ExecutiveSummary = fmt.Sprintf("Audit Pajak Masa %s %d: Posisi fiskal Nihil (PPN Keluaran sama dengan PPN Masukan). Skor kepatuhan %.1f%%.",
				monthName, year, complianceScore)
		}

		report.ActionRecommendations = []string{
			"Segera lakukan pembayaran PPN Kurang Bayar via NTPN sebelum pelaporan SPT Masa 1111.",
			"Unduh file CSV e-Faktur dan lakukan impor ke aplikasi e-Faktur Desktop DJP untuk validasi barcode.",
			"Pastikan nomor NPWP pembeli telah tervalidasi dengan NIK 16 digit sesuai regulasi PMK 112/2022.",
		}
	}

	return report, nil
}

// Helpers
func cleanNPWP(s string) string {
	digits := cleanDigits(s)
	if len(digits) == 15 {
		return fmt.Sprintf("%s.%s.%s.%s-%s.%s",
			digits[0:2], digits[2:5], digits[5:8], digits[8:9], digits[9:12], digits[12:15])
	}
	return s
}

func cleanNSFP(s string) string {
	return strings.TrimSpace(s)
}

func cleanDigits(s string) string {
	reg := regexp.MustCompile("[^0-9]+")
	return reg.ReplaceAllString(s, "")
}

func escapeCSV(s string) string {
	return strings.ReplaceAll(s, "\"", "\"\"")
}

func formatDateIndo(dateStr string) string {
	// YYYY-MM-DD -> DD/MM/YYYY
	parts := strings.Split(dateStr, "-")
	if len(parts) == 3 {
		return fmt.Sprintf("%s/%s/%s", parts[2], parts[1], parts[0])
	}
	return dateStr
}
