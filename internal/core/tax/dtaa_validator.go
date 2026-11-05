package tax

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// VendorTaxResidency holds international vendor tax residency credentials
type VendorTaxResidency struct {
	VendorID                  string    `json:"vendor_id"`
	VendorName                string    `json:"vendor_name"`
	CountryOfResidence        string    `json:"country_of_residence"`
	ForeignTIN                string    `json:"foreign_tin"`
	CoRDocNumber              string    `json:"cor_doc_number"` // Certificate of Residence / Form DGT
	CoRExpiryDate             time.Time `json:"cor_expiry_date"`
	IsBeneficialOwner         bool      `json:"is_beneficial_owner"`
	HasNoAgentIntermediary    bool      `json:"has_no_agent_intermediary"`
	HasPermanentEstablishment bool      `json:"has_permanent_establishment"`
	ConsultantDaysInCountry   int       `json:"consultant_days_in_country"`
	VerifiedStatus            string    `json:"verified_status"` // "VALID_TREATY_ELIGIBLE", "EXPIRED", "NON_TREATY_DOMESTIC"
	LastValidatedAt           time.Time `json:"last_validated_at"`
}

// TreatyRateResult represents the calculated withholding tax rate based on DTAA
type TreatyRateResult struct {
	VendorID              string    `json:"vendor_id"`
	IncomeCategory        string    `json:"income_category"`        // "ROYALTY_SOFTWARE", "TECHNICAL_SERVICES", "DIVIDENDS", "INTEREST"
	StatutoryRatePercent  float64   `json:"statutory_rate_percent"` // Standard default non-treaty rate (e.g. 20.0%)
	TreatyRatePercent     float64   `json:"treaty_rate_percent"`    // Reduced treaty rate (e.g. 0%, 5%, 10%)
	GrossInvoiceAmountUSD float64   `json:"gross_invoice_amount_usd"`
	TaxWithheldAmountUSD  float64   `json:"tax_withheld_amount_usd"`
	NetPayableAmountUSD   float64   `json:"net_payable_amount_usd"`
	TreatyArticleCited    string    `json:"treaty_article_cited"`
	TaxSavingsUSD         float64   `json:"tax_savings_usd"`
	PEWarning             bool      `json:"pe_warning"`
	CalculatedAt          time.Time `json:"calculated_at"`
}

// TaxWithholdingCertificate represents the official tax deduction slip (e-Bupot equivalent)
type TaxWithholdingCertificate struct {
	CertificateNumber      string    `json:"certificate_number"`
	VendorID               string    `json:"vendor_id"`
	VendorName             string    `json:"vendor_name"`
	ResidenceCountry       string    `json:"residence_country"`
	IncomeCategory         string    `json:"income_category"`
	GrossAmountUSD         float64   `json:"gross_amount_usd"`
	WithholdingRate        float64   `json:"withholding_rate"`
	TaxWithheldUSD         float64   `json:"tax_withheld_usd"`
	NetRemittedUSD         float64   `json:"net_remitted_usd"`
	AccountingEntryRef     string    `json:"accounting_entry_ref"`
	DigitalCertificateHash string    `json:"digital_certificate_hash"`
	IssuedAt               time.Time `json:"issued_at"`
	Status                 string    `json:"status"` // "ISSUED", "REMITTED", "CANCELLED"
}

// TreatyAuditSummary provides audit trail for cross-border tax compliance
type TreatyAuditSummary struct {
	VendorID              string                      `json:"vendor_id"`
	Residency             VendorTaxResidency          `json:"residency"`
	TotalGrossInvoicedUSD float64                     `json:"total_gross_invoiced_usd"`
	TotalTaxWithheldUSD   float64                     `json:"total_tax_withheld_usd"`
	TotalTreatySavingsUSD float64                     `json:"total_treaty_savings_usd"`
	Certificates          []TaxWithholdingCertificate `json:"certificates"`
	AuditVerdict          string                      `json:"audit_verdict"`
	AuditedAt             time.Time                   `json:"audited_at"`
}

// DTAAEngine manages Double Tax Treaty validation and withholding calculations
type DTAAEngine struct {
	mu           sync.RWMutex
	vendors      map[string]VendorTaxResidency
	certificates map[string][]TaxWithholdingCertificate
}

var (
	globalDTAAEngine *DTAAEngine
	dtaaOnce         sync.Once
)

// GetDTAAEngine returns singleton instance of DTAAEngine
func GetDTAAEngine() *DTAAEngine {
	dtaaOnce.Do(func() {
		globalDTAAEngine = &DTAAEngine{
			vendors:      make(map[string]VendorTaxResidency),
			certificates: make(map[string][]TaxWithholdingCertificate),
		}
	})
	return globalDTAAEngine
}

// ValidateResidency validates vendor tax documentation and beneficial ownership
func (e *DTAAEngine) ValidateResidency(req VendorTaxResidency) VendorTaxResidency {
	e.mu.Lock()
	defer e.mu.Unlock()

	status := "VALID_TREATY_ELIGIBLE"
	if time.Now().After(req.CoRExpiryDate) {
		status = "EXPIRED_COR_REQUIRES_STATUTORY_20PCT"
	} else if !req.IsBeneficialOwner || !req.HasNoAgentIntermediary {
		status = "BENEFICIAL_OWNERSHIP_FAILED_DENIED_TREATY"
	}

	req.VerifiedStatus = status
	req.LastValidatedAt = time.Now()
	e.vendors[req.VendorID] = req
	return req
}

// ComputeWithholdingRate calculates the applicable WHT rate under bilateral treaty
func (e *DTAAEngine) ComputeWithholdingRate(vendorID string, incomeCategory string, grossAmount float64) TreatyRateResult {
	e.mu.RLock()
	defer e.mu.RUnlock()

	vendor, exists := e.vendors[vendorID]
	var statutoryRate float64 = 20.0 // Default non-treaty PPh 26 rate
	var treatyRate float64 = statutoryRate
	var articleCited string = "Domestic Tax Law Article 26 (Default Non-Treaty)"
	peWarning := false

	if exists && vendor.VerifiedStatus == "VALID_TREATY_ELIGIBLE" {
		// Time-test check for service Permanent Establishment (e.g., > 90 days)
		if vendor.ConsultantDaysInCountry > 90 {
			peWarning = true
		}

		country := vendor.CountryOfResidence
		switch incomeCategory {
		case "ROYALTY_SOFTWARE":
			switch country {
			case "SG", "SINGAPORE":
				treatyRate = 10.0
				articleCited = "DTAA Article 12 (Royalties - Reduced 10%)"
			case "JP", "JAPAN":
				treatyRate = 10.0
				articleCited = "DTAA Article 12 (Royalties - Reduced 10%)"
			case "DE", "GERMANY":
				treatyRate = 7.5
				articleCited = "DTAA Article 12 (Royalties - Special Treaty 7.5%)"
			case "US", "USA":
				treatyRate = 10.0
				articleCited = "DTAA Article 13 (Royalties - Reduced 10%)"
			default:
				treatyRate = 10.0
				articleCited = "DTAA Standard Reduced Treaty Article 12"
			}
		case "TECHNICAL_SERVICES":
			if peWarning {
				treatyRate = 20.0
				articleCited = "Article 5 PE Triggered (>90 days in-country). Subject to domestic PE taxation."
			} else {
				// No PE -> under OECD Article 7 (Business Profits), non-taxable at source
				treatyRate = 0.0
				articleCited = "DTAA Article 7 (Business Profits without Permanent Establishment - 0% WHT)"
			}
		case "DIVIDENDS":
			treatyRate = 10.0
			articleCited = "DTAA Article 10 (Substantial Holdings Reduced Dividends - 10%)"
		case "INTEREST":
			treatyRate = 10.0
			articleCited = "DTAA Article 11 (Cross-Border Intercompany Loan Interest - 10%)"
		default:
			treatyRate = 10.0
			articleCited = "DTAA General Bilateral Treaty Relief"
		}
	}

	taxWithheld := (treatyRate / 100.0) * grossAmount
	netPayable := grossAmount - taxWithheld
	statutoryTax := (statutoryRate / 100.0) * grossAmount
	savings := statutoryTax - taxWithheld
	if savings < 0 {
		savings = 0
	}

	return TreatyRateResult{
		VendorID:              vendorID,
		IncomeCategory:        incomeCategory,
		StatutoryRatePercent:  statutoryRate,
		TreatyRatePercent:     treatyRate,
		GrossInvoiceAmountUSD: grossAmount,
		TaxWithheldAmountUSD:  taxWithheld,
		NetPayableAmountUSD:   netPayable,
		TreatyArticleCited:    articleCited,
		TaxSavingsUSD:         savings,
		PEWarning:             peWarning,
		CalculatedAt:          time.Now(),
	}
}

// GenerateCertificate issues legal tax withholding voucher and ledger entry
func (e *DTAAEngine) GenerateCertificate(vendorID string, category string, grossAmount float64) TaxWithholdingCertificate {
	e.mu.Lock()
	defer e.mu.Unlock()

	rateRes := e.computeRateInternal(vendorID, category, grossAmount)
	vendorName := "Global Tech Vendor Inc."
	resCountry := "US"
	if v, ok := e.vendors[vendorID]; ok {
		vendorName = v.VendorName
		resCountry = v.CountryOfResidence
	}

	certNo := fmt.Sprintf("WHT-DTAA-%s-%d", resCountry, time.Now().UnixNano()%1000000)
	rawHash := fmt.Sprintf("%s|%s|%.2f|%.2f|%s", certNo, vendorID, grossAmount, rateRes.TaxWithheldAmountUSD, time.Now().Format(time.RFC3339))
	hashBytes := sha256.Sum256([]byte(rawHash))
	digitalHash := hex.EncodeToString(hashBytes[:])

	cert := TaxWithholdingCertificate{
		CertificateNumber:      certNo,
		VendorID:               vendorID,
		VendorName:             vendorName,
		ResidenceCountry:       resCountry,
		IncomeCategory:         category,
		GrossAmountUSD:         grossAmount,
		WithholdingRate:        rateRes.TreatyRatePercent,
		TaxWithheldUSD:         rateRes.TaxWithheldAmountUSD,
		NetRemittedUSD:         rateRes.NetPayableAmountUSD,
		AccountingEntryRef:     fmt.Sprintf("GL-AP-WHT26-%d", time.Now().Unix()%10000),
		DigitalCertificateHash: digitalHash,
		IssuedAt:               time.Now(),
		Status:                 "ISSUED",
	}

	e.certificates[vendorID] = append(e.certificates[vendorID], cert)
	return cert
}

func (e *DTAAEngine) computeRateInternal(vendorID string, incomeCategory string, grossAmount float64) TreatyRateResult {
	vendor, exists := e.vendors[vendorID]
	var statutoryRate float64 = 20.0
	var treatyRate float64 = statutoryRate
	var articleCited string = "Domestic Tax Law Article 26"
	peWarning := false

	if exists && vendor.VerifiedStatus == "VALID_TREATY_ELIGIBLE" {
		if vendor.ConsultantDaysInCountry > 90 {
			peWarning = true
		}
		if incomeCategory == "TECHNICAL_SERVICES" && !peWarning {
			treatyRate = 0.0
			articleCited = "DTAA Article 7 (0% WHT)"
		} else {
			treatyRate = 10.0
			articleCited = "DTAA Article 12 (10% WHT)"
		}
	}

	taxWithheld := (treatyRate / 100.0) * grossAmount
	netPayable := grossAmount - taxWithheld
	savings := ((statutoryRate - treatyRate) / 100.0) * grossAmount
	if savings < 0 {
		savings = 0
	}

	return TreatyRateResult{
		VendorID:              vendorID,
		IncomeCategory:        incomeCategory,
		StatutoryRatePercent:  statutoryRate,
		TreatyRatePercent:     treatyRate,
		GrossInvoiceAmountUSD: grossAmount,
		TaxWithheldAmountUSD:  taxWithheld,
		NetPayableAmountUSD:   netPayable,
		TreatyArticleCited:    articleCited,
		TaxSavingsUSD:         savings,
		PEWarning:             peWarning,
		CalculatedAt:          time.Now(),
	}
}

// GetAuditSummary returns complete tax compliance file for vendor
func (e *DTAAEngine) GetAuditSummary(vendorID string) TreatyAuditSummary {
	e.mu.RLock()
	defer e.mu.RUnlock()

	vendor := e.vendors[vendorID]
	certs := e.certificates[vendorID]

	var totalGross, totalWithheld, totalSavings float64
	for _, c := range certs {
		totalGross += c.GrossAmountUSD
		totalWithheld += c.TaxWithheldUSD
		statTax := (20.0 / 100.0) * c.GrossAmountUSD
		diff := statTax - c.TaxWithheldUSD
		if diff > 0 {
			totalSavings += diff
		}
	}

	verdict := "COMPLIANT_FULL_DTAA_RELIEF"
	if vendor.VerifiedStatus != "VALID_TREATY_ELIGIBLE" {
		verdict = "NON_TREATY_STATUTORY_ENFORCED"
	}

	return TreatyAuditSummary{
		VendorID:              vendorID,
		Residency:             vendor,
		TotalGrossInvoicedUSD: totalGross,
		TotalTaxWithheldUSD:   totalWithheld,
		TotalTreatySavingsUSD: totalSavings,
		Certificates:          certs,
		AuditVerdict:          verdict,
		AuditedAt:             time.Now(),
	}
}

// RegisterDTAARoutes registers REST API endpoints for DTAA Tax Validation
func RegisterDTAARoutes(r *gin.RouterGroup) {
	group := r.Group("/tax/dtaa")
	{
		group.POST("/validate-vendor-residency", func(c *gin.Context) {
			var req VendorTaxResidency
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res := GetDTAAEngine().ValidateResidency(req)
			c.JSON(http.StatusOK, gin.H{
				"status":    "RESIDENCY_VALIDATED",
				"residency": res,
			})
		})

		group.POST("/compute-withholding-rate", func(c *gin.Context) {
			var req struct {
				VendorID       string  `json:"vendor_id" binding:"required"`
				IncomeCategory string  `json:"income_category" binding:"required"`
				GrossAmountUSD float64 `json:"gross_amount_usd" binding:"required"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res := GetDTAAEngine().ComputeWithholdingRate(req.VendorID, req.IncomeCategory, req.GrossAmountUSD)
			c.JSON(http.StatusOK, gin.H{
				"status": "RATE_COMPUTED",
				"result": res,
			})
		})

		group.POST("/generate-tax-certificate", func(c *gin.Context) {
			var req struct {
				VendorID       string  `json:"vendor_id" binding:"required"`
				IncomeCategory string  `json:"income_category" binding:"required"`
				GrossAmountUSD float64 `json:"gross_amount_usd" binding:"required"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			cert := GetDTAAEngine().GenerateCertificate(req.VendorID, req.IncomeCategory, req.GrossAmountUSD)
			c.JSON(http.StatusOK, gin.H{
				"status":      "CERTIFICATE_ISSUED",
				"certificate": cert,
			})
		})

		group.GET("/treaty-compliance-audit/:vendor_id", func(c *gin.Context) {
			vendorID := c.Param("vendor_id")
			audit := GetDTAAEngine().GetAuditSummary(vendorID)
			c.JSON(http.StatusOK, gin.H{
				"status": "AUDIT_SUMMARY_COMPILED",
				"audit":  audit,
			})
		})
	}
}
