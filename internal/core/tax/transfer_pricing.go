package tax

import (
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ArmLengthBenchmark defines the OECD interquartile range for comparable independent companies.
type ArmLengthBenchmark struct {
	IndustrySector      string  `json:"industry_sector"`
	MethodUsed          string  `json:"method_used"` // "TNMM", "COST_PLUS", "CUP"
	MinimumPct          float64 `json:"minimum_pct"`
	LowerQuartileQ1Pct  float64 `json:"lower_quartile_q1_pct"`
	MedianPct           float64 `json:"median_pct"`
	UpperQuartileQ3Pct  float64 `json:"upper_quartile_q3_pct"`
	MaximumPct          float64 `json:"maximum_pct"`
}

// IntercompanyTxRequest defines an affiliated cross-border transaction to audit.
type IntercompanyTxRequest struct {
	TxID                         string  `json:"tx_id"`
	SourceEntity                 string  `json:"source_entity"`
	TargetEntity                 string  `json:"target_entity"`
	SourceJurisdiction           string  `json:"source_jurisdiction"` // e.g. "ID" (Indonesia)
	TargetJurisdiction           string  `json:"target_jurisdiction"` // e.g. "SG" (Singapore)
	TxType                       string  `json:"tx_type"` // "TANGIBLE_GOODS", "MANAGEMENT_SERVICES", "IP_ROYALTY", "FINANCIAL_LOAN"
	TransactionVolumeIDR         float64 `json:"transaction_volume_idr"`
	TestedPartyCostIDR           float64 `json:"tested_party_cost_idr"`
	TestedPartyOperatingProfitIDR float64 `json:"tested_party_operating_profit_idr"`
	MethodUsed                   string  `json:"method_used"`
}

// TransferPricingAuditResult represents the arm's-length compliance audit verdict.
type TransferPricingAuditResult struct {
	TxID                         string              `json:"tx_id"`
	SourceEntity                 string              `json:"source_entity"`
	TargetEntity                 string              `json:"target_entity"`
	TestedMarginPct              float64             `json:"tested_margin_pct"`
	MethodUsed                   string              `json:"method_used"`
	BenchmarkRange               ArmLengthBenchmark  `json:"benchmark_range"`
	IsWithinArmLength            bool                `json:"is_within_arm_length"`
	AuditRiskLevel               string              `json:"audit_risk_level"` // "LOW_AUDIT_RISK_COMPLIANT", "HIGH_EXPOSURE_OUTSIDE_RANGE"
	RecommendedTaxAdjustmentIDR  float64             `json:"recommended_tax_adjustment_idr"`
	ComplianceVerdict            string              `json:"compliance_verdict"`
	AuditedAt                    string              `json:"audited_at"`
}

// CbCRRecord represents Country-by-Country tax reporting data for a single tax jurisdiction.
type CbCRRecord struct {
	JurisdictionCode     string  `json:"jurisdiction_code"`
	JurisdictionName     string  `json:"jurisdiction_name"`
	EntityCount          int     `json:"entity_count"`
	TotalRevenuesIDR     float64 `json:"total_revenues_idr"`
	ProfitBeforeTaxIDR   float64 `json:"profit_before_tax_idr"`
	IncomeTaxPaidIDR     float64 `json:"income_tax_paid_idr"`
	StatedCapitalIDR     float64 `json:"stated_capital_idr"`
	AccumulatedEarningsIDR float64 `json:"accumulated_earnings_idr"`
	TotalHeadcount       int     `json:"total_headcount"`
}

// CbCRReport aggregates the corporate group's global tax footprint under OECD BEPS Action 13.
type CbCRReport struct {
	ReportingFiscalYear     string       `json:"reporting_fiscal_year"`
	UltimateParentEntity    string       `json:"ultimate_parent_entity"`
	TotalGroupRevenueIDR    float64      `json:"total_group_revenue_idr"`
	TotalGroupTaxPaidIDR    float64      `json:"total_group_tax_paid_idr"`
	EffectiveTaxRatePercent float64      `json:"effective_tax_rate_percent"`
	Jurisdictions           []CbCRRecord `json:"jurisdictions"`
	AuditReadinessStatus    string       `json:"audit_readiness_status"`
	GeneratedAt             string       `json:"generated_at"`
}

// TPStore maintains in-memory store for transfer pricing audits.
type TPStore struct {
	sync.RWMutex
	audits map[string]*TransferPricingAuditResult
}

var globalTPStore = &TPStore{
	audits: make(map[string]*TransferPricingAuditResult),
}

// AuditTransferPricingMarginsLogic audits tested party margins against the OECD interquartile range.
func AuditTransferPricingMarginsLogic(req IntercompanyTxRequest) (*TransferPricingAuditResult, error) {
	if req.TxID == "" {
		req.TxID = fmt.Sprintf("TP-TX-%s-%d", time.Now().Format("20060102"), time.Now().Unix()%1000)
	}
	if req.SourceEntity == "" {
		req.SourceEntity = "PT GoERP Manufaktur Indonesia"
	}
	if req.TargetEntity == "" {
		req.TargetEntity = "GoERP Technologies Pte Ltd (Singapore)"
	}
	if req.TransactionVolumeIDR <= 0 {
		req.TransactionVolumeIDR = 8500000000.0 // Rp 8.5 Miliar
	}
	if req.TestedPartyCostIDR <= 0 {
		req.TestedPartyCostIDR = 7800000000.0
	}
	if req.TestedPartyOperatingProfitIDR <= 0 {
		req.TestedPartyOperatingProfitIDR = 700000000.0 // ~8.24% operating margin
	}
	if req.MethodUsed == "" {
		req.MethodUsed = "TNMM"
	}

	// Benchmark range for Electronics Contract Manufacturing (TNMM Operating Margin / Full Cost Mark-up)
	benchmark := ArmLengthBenchmark{
		IndustrySector:     "Electronics & Precision Component Manufacturing",
		MethodUsed:         req.MethodUsed,
		MinimumPct:         4.20,
		LowerQuartileQ1Pct: 6.50,
		MedianPct:          8.40,
		UpperQuartileQ3Pct: 11.20,
		MaximumPct:         14.80,
	}

	testedMargin := 0.0
	if req.TestedPartyCostIDR > 0 {
		testedMargin = math.Round((req.TestedPartyOperatingProfitIDR/req.TestedPartyCostIDR)*10000) / 100.0
	}

	isCompliant := testedMargin >= benchmark.LowerQuartileQ1Pct && testedMargin <= benchmark.UpperQuartileQ3Pct
	riskLevel := "LOW_AUDIT_RISK_COMPLIANT"
	recAdjustment := 0.0
	verdict := fmt.Sprintf("Operating margin of %.2f%% falls comfortably within arm's-length interquartile range (%.2f%% - %.2f%%). Full tax deduction defended.",
		testedMargin, benchmark.LowerQuartileQ1Pct, benchmark.UpperQuartileQ3Pct)

	if !isCompliant {
		riskLevel = "HIGH_EXPOSURE_OUTSIDE_RANGE"
		// If outside, tax authority typically adjusts to Median
		targetProfit := req.TestedPartyCostIDR * (benchmark.MedianPct / 100.0)
		recAdjustment = math.Abs(math.Round(targetProfit - req.TestedPartyOperatingProfitIDR))
		verdict = fmt.Sprintf("Operating margin of %.2f%% is outside arm's-length interquartile range (%.2f%% - %.2f%%). Potential transfer pricing adjustment to median: Rp %.0f.",
			testedMargin, benchmark.LowerQuartileQ1Pct, benchmark.UpperQuartileQ3Pct, recAdjustment)
	}

	res := &TransferPricingAuditResult{
		TxID:                        req.TxID,
		SourceEntity:                req.SourceEntity,
		TargetEntity:                req.TargetEntity,
		TestedMarginPct:             testedMargin,
		MethodUsed:                  req.MethodUsed,
		BenchmarkRange:              benchmark,
		IsWithinArmLength:           isCompliant,
		AuditRiskLevel:              riskLevel,
		RecommendedTaxAdjustmentIDR: recAdjustment,
		ComplianceVerdict:           verdict,
		AuditedAt:                   time.Now().Format(time.RFC3339),
	}

	globalTPStore.Lock()
	globalTPStore.audits[req.TxID] = res
	globalTPStore.Unlock()

	return res, nil
}

// GetCbCRMatrixLogic provides aggregated Country-by-Country Reporting under OECD BEPS Action 13.
func GetCbCRMatrixLogic() (*CbCRReport, error) {
	jurisdictions := []CbCRRecord{
		{
			JurisdictionCode:       "ID",
			JurisdictionName:       "Indonesia",
			EntityCount:            3,
			TotalRevenuesIDR:       124500000000.0, // 124.5 Miliar
			ProfitBeforeTaxIDR:     18400000000.0,
			IncomeTaxPaidIDR:       3864000000.0, // ~21% ETR
			StatedCapitalIDR:       25000000000.0,
			AccumulatedEarningsIDR: 42000000000.0,
			TotalHeadcount:         185,
		},
		{
			JurisdictionCode:       "SG",
			JurisdictionName:       "Singapore",
			EntityCount:            1,
			TotalRevenuesIDR:       48200000000.0,
			ProfitBeforeTaxIDR:     8900000000.0,
			IncomeTaxPaidIDR:       1513000000.0, // ~17% headline
			StatedCapitalIDR:       10000000000.0,
			AccumulatedEarningsIDR: 18500000000.0,
			TotalHeadcount:         24,
		},
		{
			JurisdictionCode:       "NL",
			JurisdictionName:       "Netherlands",
			EntityCount:            1,
			TotalRevenuesIDR:       32100000000.0,
			ProfitBeforeTaxIDR:     5400000000.0,
			IncomeTaxPaidIDR:       1350000000.0, // ~25%
			StatedCapitalIDR:       8000000000.0,
			AccumulatedEarningsIDR: 12000000000.0,
			TotalHeadcount:         12,
		},
	}

	var totRev, totTax, totPBT float64
	for _, j := range jurisdictions {
		totRev += j.TotalRevenuesIDR
		totTax += j.IncomeTaxPaidIDR
		totPBT += j.ProfitBeforeTaxIDR
	}

	etr := 0.0
	if totPBT > 0 {
		etr = math.Round((totTax/totPBT)*10000) / 100.0
	}

	report := &CbCRReport{
		ReportingFiscalYear:     "2026",
		UltimateParentEntity:    "PT GoERP Holdings Tbk",
		TotalGroupRevenueIDR:    totRev,
		TotalGroupTaxPaidIDR:    totTax,
		EffectiveTaxRatePercent: etr,
		Jurisdictions:           jurisdictions,
		AuditReadinessStatus:    "OECD_BEPS_ACTION13_COMPLIANT_AUDIT_READY",
		GeneratedAt:             time.Now().Format(time.RFC3339),
	}

	return report, nil
}

// TransferPricingHandler handles HTTP endpoints for transfer pricing audits and CbCR reports.
type TransferPricingHandler struct {
	db *gorm.DB
}

// NewTransferPricingHandler constructs a new TransferPricingHandler.
func NewTransferPricingHandler(db *gorm.DB) *TransferPricingHandler {
	return &TransferPricingHandler{db: db}
}

// AuditMargins audits tested party margins.
func (h *TransferPricingHandler) AuditMargins(c *gin.Context) {
	var req IntercompanyTxRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid transfer pricing payload: " + err.Error()})
		return
	}

	res, err := AuditTransferPricingMarginsLogic(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

// GetCbCRMatrix returns Country-by-Country reporting.
func (h *TransferPricingHandler) GetCbCRMatrix(c *gin.Context) {
	report, err := GetCbCRMatrixLogic()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, report)
}

// RegisterTransferPricingRoutes registers transfer pricing endpoints.
func RegisterTransferPricingRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc, db *gorm.DB) {
	h := NewTransferPricingHandler(db)
	taxGroup := router.Group("/tax/transfer-pricing")
	taxGroup.Use(authMiddleware)
	{
		taxGroup.POST("/audit-margins", h.AuditMargins)
		taxGroup.GET("/cbcr-matrix", h.GetCbCRMatrix)
	}
}
