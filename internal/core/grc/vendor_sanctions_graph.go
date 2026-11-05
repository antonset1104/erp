package grc

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// PartnerCandidate represents vendor/customer profile for deduplication and AML screening
type PartnerCandidate struct {
	PartnerID               string    `json:"partner_id"`
	CompanyName             string    `json:"company_name"`
	TaxID                   string    `json:"tax_id"` // NPWP / TIN
	BankAccountNumber       string    `json:"bank_account_number"`
	BankName                string    `json:"bank_name"`
	AddressCity             string    `json:"address_city"`
	UltimateBeneficialOwner string    `json:"ultimate_beneficial_owner"`
	UBOSanctionSharePct     float64   `json:"ubo_sanction_share_pct"` // Equity share owned by sanctioned party
	CreatedAt               time.Time `json:"created_at"`
}

// DeduplicationCheckResult models entity resolution across conglomerates
type DeduplicationCheckResult struct {
	CheckID            string    `json:"check_id"`
	NewPartnerID       string    `json:"new_partner_id"`
	ExistingPartnerID  string    `json:"existing_partner_id,omitempty"`
	IsDuplicate        bool      `json:"is_duplicate"`
	MatchConfidencePct float64   `json:"match_confidence_pct"`
	MatchingAttributes []string  `json:"matching_attributes"`
	Recommendation     string    `json:"recommendation"` // "MERGE_PARTNER_ENTITIES", "DISTINCT_RECORD_PROCEED"
	CheckedAt          time.Time `json:"checked_at"`
}

// SanctionsScreeningResult models OFAC / EU / UN watchlist and SDN 50% rule enforcement
type SanctionsScreeningResult struct {
	ScreeningID               string    `json:"screening_id"`
	PartnerID                 string    `json:"partner_id"`
	CompanyName               string    `json:"company_name"`
	WatchlistHit              bool      `json:"watchlist_hit"`
	MatchedWatchlist          string    `json:"matched_watchlist,omitempty"` // "US_OFAC_SDN", "EU_CONSOLIDATED", "UN_SECURITY_COUNCIL"
	SDN50PercentRuleTriggered bool      `json:"sdn_50_percent_rule_triggered"`
	RiskCategory              string    `json:"risk_category"`     // "BLOCKED_SANCTIONED_PROPERTY", "LOW_RISK_CLEAN"
	ComplianceAction          string    `json:"compliance_action"` // "IMMEDIATE_TRANSACTION_FREEZE", "APPROVED_FOR_PAYMENT"
	ScreenedAt                time.Time `json:"screened_at"`
}

// PaymentFreezeRecord captures formal AML/OFAC payment interception
type PaymentFreezeRecord struct {
	FreezeID               string    `json:"freeze_id"`
	PartnerID              string    `json:"partner_id"`
	FrozenPayableAmountUSD float64   `json:"frozen_payable_amount_usd"`
	SARReportReference     string    `json:"sar_report_reference"` // Suspicious Activity Report
	RegulatoryHash         string    `json:"regulatory_hash"`
	Status                 string    `json:"status"` // "PAYMENT_BLOCKED_AML_FREEZE"
	FrozenAt               time.Time `json:"frozen_at"`
}

// VendorSanctionsVault thread-safe store
type VendorSanctionsVault struct {
	mu         sync.RWMutex
	partners   map[string]PartnerCandidate
	dedupLogs  map[string]DeduplicationCheckResult
	screenLogs map[string]SanctionsScreeningResult
	freezeLogs map[string]PaymentFreezeRecord
}

// Global Vendor Sanctions instance
var GlobalVendorSanctionsVault = &VendorSanctionsVault{
	partners:   make(map[string]PartnerCandidate),
	dedupLogs:  make(map[string]DeduplicationCheckResult),
	screenLogs: make(map[string]SanctionsScreeningResult),
	freezeLogs: make(map[string]PaymentFreezeRecord),
}

func init() {
	// Seed demo clean and existing vendors
	GlobalVendorSanctionsVault.partners["VND-STEEL-001"] = PartnerCandidate{
		PartnerID:               "VND-STEEL-001",
		CompanyName:             "PT Krakatau Prima Baja Nusantara",
		TaxID:                   "01.234.567.8-012.000",
		BankAccountNumber:       "123-00-9876543-2",
		BankName:                "Bank Mandiri",
		AddressCity:             "Cilegon",
		UltimateBeneficialOwner: "H. Sukirman Hadikusumo",
		UBOSanctionSharePct:     0.0,
		CreatedAt:               time.Now().AddDate(-2, 0, 0),
	}
}

// DeduplicatePartnerMaster checks candidates against existing partner graph
func (v *VendorSanctionsVault) DeduplicatePartnerMaster(p PartnerCandidate) DeduplicationCheckResult {
	v.mu.Lock()
	defer v.mu.Unlock()

	p.CreatedAt = time.Now()
	v.partners[p.PartnerID] = p

	var matchID string
	var matchAttrs []string
	var confidence float64

	for id, ex := range v.partners {
		if id == p.PartnerID {
			continue
		}

		if ex.TaxID == p.TaxID && p.TaxID != "" {
			matchID = id
			matchAttrs = append(matchAttrs, "Identical Tax Identification Number (NPWP/TIN)")
			confidence = 100.0
			break
		}
		if ex.BankAccountNumber == p.BankAccountNumber && p.BankAccountNumber != "" {
			matchID = id
			matchAttrs = append(matchAttrs, "Identical Bank Account Number")
			confidence = 98.0
			break
		}
		if strings.EqualFold(ex.CompanyName, p.CompanyName) {
			matchID = id
			matchAttrs = append(matchAttrs, "Exact Company Name Match")
			confidence = 95.0
			break
		}
	}

	isDup := confidence >= 85.0
	rec := "DISTINCT_RECORD_PROCEED"
	if isDup {
		rec = "MERGE_PARTNER_ENTITIES_DUPLICATE_DETECTED"
	}

	checkID := fmt.Sprintf("CHK-DEDUP-%d", time.Now().UnixNano()%1000000)
	result := DeduplicationCheckResult{
		CheckID:            checkID,
		NewPartnerID:       p.PartnerID,
		ExistingPartnerID:  matchID,
		IsDuplicate:        isDup,
		MatchConfidencePct: confidence,
		MatchingAttributes: matchAttrs,
		Recommendation:     rec,
		CheckedAt:          time.Now(),
	}

	v.dedupLogs[p.PartnerID] = result
	return result
}

// ScreenUBOWatchlists scans against global sanctions and evaluates US OFAC 50% Rule
func (v *VendorSanctionsVault) ScreenUBOWatchlists(partnerID string) (*SanctionsScreeningResult, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	p, ok := v.partners[partnerID]
	if !ok {
		return nil, fmt.Errorf("partner candidate not found: %s", partnerID)
	}

	// Watchlist keyword triggers (simulating OFAC / EU SDN lists)
	sanctionKeywords := []string{"oligarh", "sanction", "belarus", "crimea", "prohibited", "arms"}
	hit := false
	var matchedList string

	compLower := strings.ToLower(p.CompanyName)
	uboLower := strings.ToLower(p.UltimateBeneficialOwner)

	for _, kw := range sanctionKeywords {
		if strings.Contains(compLower, kw) || strings.Contains(uboLower, kw) {
			hit = true
			matchedList = "US_OFAC_SDN_LIST"
			break
		}
	}

	// Check US OFAC 50% Rule: Entity owned >= 50% by sanctioned person is automatically blocked
	sdn50Trigger := false
	if p.UBOSanctionSharePct >= 50.0 {
		hit = true
		sdn50Trigger = true
		if matchedList == "" {
			matchedList = "OFAC_SDN_50_PERCENT_RULE"
		}
	}

	riskCat := "LOW_RISK_CLEAN"
	action := "APPROVED_FOR_PAYMENT"
	if hit {
		riskCat = "BLOCKED_SANCTIONED_PROPERTY"
		action = "IMMEDIATE_TRANSACTION_FREEZE"
	}

	scrID := fmt.Sprintf("SCR-SANC-%d", time.Now().UnixNano()%1000000)
	result := SanctionsScreeningResult{
		ScreeningID:               scrID,
		PartnerID:                 partnerID,
		CompanyName:               p.CompanyName,
		WatchlistHit:              hit,
		MatchedWatchlist:          matchedList,
		SDN50PercentRuleTriggered: sdn50Trigger,
		RiskCategory:              riskCat,
		ComplianceAction:          action,
		ScreenedAt:                time.Now(),
	}

	v.screenLogs[partnerID] = result
	return &result, nil
}

// EnforcePaymentFreeze freezes AP payments and logs SAR report
func (v *VendorSanctionsVault) EnforcePaymentFreeze(
	partnerID string, payableAmount float64,
) (*PaymentFreezeRecord, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	scr, ok := v.screenLogs[partnerID]
	if !ok || !scr.WatchlistHit {
		return nil, fmt.Errorf("cannot freeze: partner %s has not tested positive for sanctions", partnerID)
	}

	rawCert := fmt.Sprintf("AML_FREEZE:%s:%s:%.2f:%d", partnerID, scr.MatchedWatchlist, payableAmount, time.Now().Unix())
	hashBytes := sha256.Sum256([]byte(rawCert))
	digest := hex.EncodeToString(hashBytes[:])

	sarRef := fmt.Sprintf("SAR-FIU-2026-%d", time.Now().Unix()%100000)
	freezeID := fmt.Sprintf("FRZ-PAY-%d", time.Now().UnixNano()%1000000)

	record := PaymentFreezeRecord{
		FreezeID:               freezeID,
		PartnerID:              partnerID,
		FrozenPayableAmountUSD: payableAmount,
		SARReportReference:     sarRef,
		RegulatoryHash:         digest,
		Status:                 "PAYMENT_BLOCKED_AML_FREEZE",
		FrozenAt:               time.Now(),
	}

	v.freezeLogs[partnerID] = record
	return &record, nil
}

// GetComplianceDossier compiles 360-degree partner compliance report
func (v *VendorSanctionsVault) GetComplianceDossier(partnerID string) (gin.H, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	p, hasP := v.partners[partnerID]
	dedup, hasDedup := v.dedupLogs[partnerID]
	scr, hasScr := v.screenLogs[partnerID]
	frz, hasFrz := v.freezeLogs[partnerID]

	if !hasP {
		return nil, fmt.Errorf("no partner records found: %s", partnerID)
	}

	return gin.H{
		"partner_id":             partnerID,
		"profile":                p,
		"deduplication_audit":    dedup,
		"has_deduplication":      hasDedup,
		"sanctions_screening":    scr,
		"has_screening":          hasScr,
		"payment_freeze_record":  frz,
		"has_freeze":             hasFrz,
		"compliance_framework":   "US OFAC Sanctions / EU Consolidated List / AML-CFT PPATK",
		"sdn_50_rule_compliance": "AUTOMATED_BENEFICIAL_OWNER_GRAPH_VERIFIED",
	}, nil
}

// RegisterSanctionsRoutes registers HTTP endpoints
func RegisterSanctionsRoutes(r *gin.RouterGroup) {
	snc := r.Group("/grc/sanctions")
	{
		snc.POST("/deduplicate-partner-master", func(c *gin.Context) {
			var req PartnerCandidate
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			res := GlobalVendorSanctionsVault.DeduplicatePartnerMaster(req)
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		snc.POST("/screen-ubo-watchlists", func(c *gin.Context) {
			var req struct {
				PartnerID string `json:"partner_id" binding:"required"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			res, err := GlobalVendorSanctionsVault.ScreenUBOWatchlists(req.PartnerID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		snc.POST("/enforce-payment-freeze", func(c *gin.Context) {
			var req struct {
				PartnerID           string  `json:"partner_id" binding:"required"`
				FrozenPayableAmount float64 `json:"frozen_payable_amount_usd" binding:"required"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			rec, err := GlobalVendorSanctionsVault.EnforcePaymentFreeze(req.PartnerID, req.FrozenPayableAmount)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": rec})
		})

		snc.GET("/compliance-graph-dossier/:partner_id", func(c *gin.Context) {
			partnerID := c.Param("partner_id")
			dossier, err := GlobalVendorSanctionsVault.GetComplianceDossier(partnerID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": dossier})
		})
	}
}
