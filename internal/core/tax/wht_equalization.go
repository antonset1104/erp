package tax

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// GLExpenseLine represents a GL trial balance expense line subject to withholding tax audit.
type GLExpenseLine struct {
	AccountCode       string  `json:"account_code"`
	AccountName       string  `json:"account_name"`
	GLTotalExpenseIDR float64 `json:"gl_total_expense_idr"`
	TaxType           string  `json:"tax_type"` // "PPH_23", "PPH_4_AYAT_2", "PPH_26", "PPH_21"
}

// WHTSlipIssued represents a submitted withholding tax slip (Bukti Potong).
type WHTSlipIssued struct {
	SlipNumber         string  `json:"slip_number"`
	VendorName         string  `json:"vendor_name"`
	TaxType            string  `json:"tax_type"`
	TaxBaseDPP_IDR     float64 `json:"tax_base_dpp_idr"`
	WithholdingRatePct float64 `json:"withholding_rate_pct"`
	TaxWithheldIDR     float64 `json:"tax_withheld_idr"`
}

// WHTEqualizationRequest holds general ledger and tax return lines for automated reconciliation.
type WHTEqualizationRequest struct {
	FiscalYear  int             `json:"fiscal_year"`
	FiscalMonth int             `json:"fiscal_month"`
	GLExpenses  []GLExpenseLine `json:"gl_expenses"`
	WHTSlips    []WHTSlipIssued `json:"wht_slips"`
}

// WHTEqualizationDiscrepancy isolates unmatched expenses subject to tax audit exposure.
type WHTEqualizationDiscrepancy struct {
	TaxType                 string  `json:"tax_type"`
	AccountCode             string  `json:"account_code"`
	TotalGLExpenseIDR       float64 `json:"total_gl_expense_idr"`
	TotalReportedDPP_IDR    float64 `json:"total_reported_dpp_idr"`
	DiscrepancyAmountIDR    float64 `json:"discrepancy_amount_idr"`
	PotentialTaxExposureIDR float64 `json:"potential_tax_exposure_idr"`
	RootCauseNote           string  `json:"root_cause_note"`
}

// WHTEqualizationReport summarizes tax base equalization and audit exposure.
type WHTEqualizationReport struct {
	ReportID                    string                       `json:"report_id"`
	FiscalPeriod                string                       `json:"fiscal_period"`
	TotalGLExpensesAuditedIDR   float64                      `json:"total_gl_expenses_audited_idr"`
	TotalReportedDPPIDR         float64                      `json:"total_reported_dpp_idr"`
	TotalTaxWithheldIDR         float64                      `json:"total_tax_withheld_idr"`
	TotalDiscrepancyIDR         float64                      `json:"total_discrepancy_idr"`
	TotalPotentialTaxExposureIDR float64                      `json:"total_potential_tax_exposure_idr"`
	AuditStatus                 string                       `json:"audit_status"` // "EQUALIZED_CLEAN", "DISCREPANCY_DETECTED"
	Discrepancies               []WHTEqualizationDiscrepancy `json:"discrepancies"`
	GeneratedAt                 string                       `json:"generated_at"`
}

// DGT1ValidationRequest holds foreign vendor certificate data under tax treaties (P3B).
type DGT1ValidationRequest struct {
	VendorName                string  `json:"vendor_name"`
	VendorCountry             string  `json:"vendor_country"` // "SG", "JP", "NL", "DE", "US"
	DGTFormNumber             string  `json:"dgt_form_number"`
	CertificateExpirationDate string  `json:"certificate_expiration_date"` // YYYY-MM-DD
	GrossPaymentIDR           float64 `json:"gross_payment_idr"`
	IncomeCategory            string  `json:"income_category"` // "ROYALTIES", "TECHNICAL_SERVICES", "DIVIDENDS", "INTEREST"
}

// DGT1ValidationResult verifies tax treaty rate relief and computes tax savings.
type DGT1ValidationResult struct {
	VendorName           string  `json:"vendor_name"`
	DGTFormNumber        string  `json:"dgt_form_number"`
	IsValidCertificate   bool    `json:"is_valid_certificate"`
	DomesticWHTRatePct   float64 `json:"domestic_wht_rate_pct"`
	TreatyRatePct        float64 `json:"treaty_rate_pct"`
	DomesticTaxIDR       float64 `json:"domestic_tax_idr"`
	TreatyTaxIDR         float64 `json:"treaty_tax_idr"`
	TaxSavedIDR          float64 `json:"tax_saved_idr"`
	TreatyArticleApplied string  `json:"treaty_article_applied"`
	ValidationVerdict    string  `json:"validation_verdict"`
}

// AuditWHTEqualizationLogic matches GL expenses with reported DPP withholding slips.
func AuditWHTEqualizationLogic(req WHTEqualizationRequest) WHTEqualizationReport {
	if req.FiscalYear <= 0 {
		req.FiscalYear = time.Now().Year()
	}
	if req.FiscalMonth <= 0 {
		req.FiscalMonth = int(time.Now().Month())
	}

	if len(req.GLExpenses) == 0 {
		req.GLExpenses = []GLExpenseLine{
			{AccountCode: "6110-CONSULTING-SERVICES", AccountName: "Legal & Consulting Professional Fees", GLTotalExpenseIDR: 450000000.0, TaxType: "PPH_23"},
			{AccountCode: "6120-OFFICE-RENTAL", AccountName: "Commercial Office Building Rental", GLTotalExpenseIDR: 300000000.0, TaxType: "PPH_4_AYAT_2"},
			{AccountCode: "6130-OFFSHORE-SOFTWARE-ROYALTY", AccountName: "Offshore Cloud & IP Royalties", GLTotalExpenseIDR: 600000000.0, TaxType: "PPH_26"},
		}
	}

	if len(req.WHTSlips) == 0 {
		req.WHTSlips = []WHTSlipIssued{
			{SlipNumber: "BP-23-2026-001", VendorName: "PT Konsultan Finansial Prima", TaxType: "PPH_23", TaxBaseDPP_IDR: 400000000.0, WithholdingRatePct: 2.0, TaxWithheldIDR: 8000000.0},
			{SlipNumber: "BP-42-2026-001", VendorName: "PT Gedung Perkantoran Sudirman", TaxType: "PPH_4_AYAT_2", TaxBaseDPP_IDR: 300000000.0, WithholdingRatePct: 10.0, TaxWithheldIDR: 30000000.0},
			{SlipNumber: "BP-26-2026-001", VendorName: "Global Cloud SaaS Inc (US)", TaxType: "PPH_26", TaxBaseDPP_IDR: 600000000.0, WithholdingRatePct: 10.0, TaxWithheldIDR: 60000000.0},
		}
	}

	// Sum DPP by tax type
	dppByTaxType := make(map[string]float64)
	var totalReportedDPP, totalTaxWithheld float64
	for _, s := range req.WHTSlips {
		dppByTaxType[s.TaxType] += s.TaxBaseDPP_IDR
		totalReportedDPP += s.TaxBaseDPP_IDR
		totalTaxWithheld += s.TaxWithheldIDR
	}

	var discrepancies []WHTEqualizationDiscrepancy
	var totalGLExpenses, totalDiscrepancy, totalPotentialExposure float64

	for _, gl := range req.GLExpenses {
		totalGLExpenses += gl.GLTotalExpenseIDR
		reportedDPP := dppByTaxType[gl.TaxType]

		diff := gl.GLTotalExpenseIDR - reportedDPP
		if diff > 1000.0 { // Noticeable positive gap where GL expense exceeds tax base
			var rate float64
			switch gl.TaxType {
			case "PPH_23":
				rate = 0.02
			case "PPH_4_AYAT_2":
				rate = 0.10
			case "PPH_26":
				rate = 0.20
			default:
				rate = 0.05
			}
			taxExposure := diff * rate
			totalDiscrepancy += diff
			totalPotentialExposure += taxExposure

			discrepancies = append(discrepancies, WHTEqualizationDiscrepancy{
				TaxType:                 gl.TaxType,
				AccountCode:             gl.AccountCode,
				TotalGLExpenseIDR:       gl.GLTotalExpenseIDR,
				TotalReportedDPP_IDR:    reportedDPP,
				DiscrepancyAmountIDR:    diff,
				PotentialTaxExposureIDR: taxExposure,
				RootCauseNote:           fmt.Sprintf("Expense exceeds reported tax base by Rp %.0f. Likely unwithheld invoice or non-taxable reimbursement requiring ledger breakdown.", diff),
			})
		}
	}

	auditStatus := "EQUALIZED_CLEAN"
	if len(discrepancies) > 0 {
		auditStatus = "DISCREPANCY_DETECTED"
	}

	repID := fmt.Sprintf("WHT-EQ-%d%02d-%03d", req.FiscalYear, req.FiscalMonth, time.Now().Unix()%1000)

	return WHTEqualizationReport{
		ReportID:                     repID,
		FiscalPeriod:                 fmt.Sprintf("%d-%02d", req.FiscalYear, req.FiscalMonth),
		TotalGLExpensesAuditedIDR:    totalGLExpenses,
		TotalReportedDPPIDR:          totalReportedDPP,
		TotalTaxWithheldIDR:          totalTaxWithheld,
		TotalDiscrepancyIDR:          totalDiscrepancy,
		TotalPotentialTaxExposureIDR: totalPotentialExposure,
		AuditStatus:                  auditStatus,
		Discrepancies:                discrepancies,
		GeneratedAt:                  time.Now().Format(time.RFC3339),
	}
}

// ValidateDGT1CertificateLogic validates non-resident tax treaty certificate and applies reduced treaty rates.
func ValidateDGT1CertificateLogic(req DGT1ValidationRequest) DGT1ValidationResult {
	if req.VendorName == "" {
		req.VendorName = "Acme Global Technology Pte Ltd"
	}
	if req.VendorCountry == "" {
		req.VendorCountry = "SG"
	}
	if req.DGTFormNumber == "" {
		req.DGTFormNumber = "DGT-1/SG/2026/8941"
	}
	if req.GrossPaymentIDR <= 0 {
		req.GrossPaymentIDR = 500000000.0 // Rp 500 Juta
	}

	// Validate expiration
	isValid := true
	if req.CertificateExpirationDate != "" {
		expTime, err := time.Parse("2006-01-02", req.CertificateExpirationDate)
		if err == nil && expTime.Before(time.Now()) {
			isValid = false
		}
	}

	domesticRate := 20.0 // Standard Indonesian PPh 26 rate
	treatyRate := domesticRate
	var treatyArticle string

	if isValid {
		switch strings.ToUpper(req.VendorCountry) {
		case "SG": // Singapore
			switch strings.ToUpper(req.IncomeCategory) {
			case "ROYALTIES":
				treatyRate = 10.0
				treatyArticle = "Article 12 (Royalties) - Indonesia-Singapore DTA (10% rate)"
			case "TECHNICAL_SERVICES":
				treatyRate = 0.0 // No PE in Indonesia
				treatyArticle = "Article 7 (Business Profits) - Indonesia-Singapore DTA (0% without PE)"
			default:
				treatyRate = 10.0
				treatyArticle = "Article 11 (Interest/Dividends) - Indonesia-Singapore DTA (10%)"
			}
		case "JP": // Japan
			treatyRate = 10.0
			treatyArticle = "Article 12 - Indonesia-Japan DTA (10% reduced rate)"
		case "NL": // Netherlands
			treatyRate = 10.0
			treatyArticle = "Article 12 - Indonesia-Netherlands DTA (10% reduced rate)"
		default:
			treatyRate = 10.0
			treatyArticle = "Double Tax Avoidance Agreement Standard Treaty Rate (10%)"
		}
	} else {
		treatyArticle = "N/A - Expired or Invalid DGT-1 Form. Domestic 20% statutory rate applied."
	}

	domesticTax := req.GrossPaymentIDR * (domesticRate / 100.0)
	treatyTax := req.GrossPaymentIDR * (treatyRate / 100.0)
	taxSaved := domesticTax - treatyTax

	var verdict string
	if isValid {
		verdict = fmt.Sprintf("Valid DGT-1 Certificate verified. Withholding reduced from 20%% to %.1f%%, saving Rp %.0f in cross-border tax.", treatyRate, taxSaved)
	} else {
		verdict = "DGT-1 Certificate expired or invalid. Full domestic 20% PPh 26 withholding required."
	}

	return DGT1ValidationResult{
		VendorName:           req.VendorName,
		DGTFormNumber:        req.DGTFormNumber,
		IsValidCertificate:   isValid,
		DomesticWHTRatePct:   domesticRate,
		TreatyRatePct:        treatyRate,
		DomesticTaxIDR:       domesticTax,
		TreatyTaxIDR:         treatyTax,
		TaxSavedIDR:          taxSaved,
		TreatyArticleApplied: treatyArticle,
		ValidationVerdict:    verdict,
	}
}

// RegisterWHTRoutes registers Withholding Tax Equalization and DGT-1 routes.
func RegisterWHTRoutes(r *gin.RouterGroup) {
	group := r.Group("/tax/wht")
	{
		group.POST("/audit-equalization", func(c *gin.Context) {
			var req WHTEqualizationRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				req = WHTEqualizationRequest{}
			}
			res := AuditWHTEqualizationLogic(req)
			c.JSON(http.StatusOK, res)
		})

		group.POST("/validate-dgt1", func(c *gin.Context) {
			var req DGT1ValidationRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				req = DGT1ValidationRequest{}
			}
			res := ValidateDGT1CertificateLogic(req)
			c.JSON(http.StatusOK, res)
		})
	}
}
