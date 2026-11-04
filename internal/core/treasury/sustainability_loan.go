package treasury

import (
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// SustainabilityTargetKPI represents a contractual ESG milestone under the loan covenant.
type SustainabilityTargetKPI struct {
	KPICode         string  `json:"kpi_code"` // e.g. "SPT-GHG-01", "SPT-RENEWABLE-02", "SPT-K3-03"
	KPIName         string  `json:"kpi_name"`
	BaselineValue   float64 `json:"baseline_value"`
	TargetThreshold float64 `json:"target_threshold"`
	ActualAchieved  float64 `json:"actual_achieved"`
	TargetMet       bool    `json:"target_met"`
	WeightBps       float64 `json:"weight_bps"`
}

// SustainabilityLinkedFacility models a corporate syndicated green credit or SLL facility.
type SustainabilityLinkedFacility struct {
	FacilityID              string                    `json:"facility_id"` // e.g. "SLL-2026-SYNDICATE-01"
	FacilityName            string                    `json:"facility_name"`
	LenderSyndicateLead     string                    `json:"lender_syndicate_lead"`
	PrincipalAmountUSD      float64                   `json:"principal_amount_usd"`
	BenchmarkRateName       string                    `json:"benchmark_rate_name"` // "SOFR_3M"
	BaseMarginBps           float64                   `json:"base_margin_bps"`     // 175 bps
	StepDownDiscountBps     float64                   `json:"step_down_discount_bps"` // 25 bps
	StepUpPenaltyBps        float64                   `json:"step_up_penalty_bps"`    // 25 bps
	ActiveInterestRatePct   float64                   `json:"active_interest_rate_pct"`
	AnnualInterestSavingUSD float64                   `json:"annual_interest_saving_usd"`
	Targets                 []SustainabilityTargetKPI `json:"targets"`
	Status                  string                    `json:"status"` // "ACTIVE_DISCOUNTED", "ACTIVE_PENALTY", "STANDARD_BASE"
	CreatedAt               string                    `json:"created_at"`
}

// EvaluateSLLCouponRequest holds ESG actuals verified by accredited independent auditors.
type EvaluateSLLCouponRequest struct {
	FacilityID               string  `json:"facility_id"`
	EvaluationPeriod         string  `json:"evaluation_period"` // e.g. "2026-Q1"
	CurrentSOFRRatePct       float64 `json:"current_sofr_rate_pct"`
	AchievedGHGReductionPct  float64 `json:"achieved_ghg_reduction_pct"`
	AchievedRenewableSharePct float64 `json:"achieved_renewable_share_pct"`
	AchievedLTIFR            float64 `json:"achieved_ltifr"`
}

// SLLCouponAdjustmentResult reflects interest recalculation and General Ledger journal entry.
type SLLCouponAdjustmentResult struct {
	FacilityID               string  `json:"facility_id"`
	EvaluationPeriod         string  `json:"evaluation_period"`
	PrincipalAmountUSD       float64 `json:"principal_amount_usd"`
	TotalTargetsCount        int     `json:"total_targets_count"`
	TargetsMetCount          int     `json:"targets_met_count"`
	PriorCouponRatePct       float64 `json:"prior_coupon_rate_pct"`
	RevisedCouponRatePct     float64 `json:"revised_coupon_rate_pct"`
	MarginAdjustmentBps      float64 `json:"margin_adjustment_bps"` // -25.0, 0.0, or +25.0
	NetAnnualInterestDeltaUSD float64 `json:"net_annual_interest_delta_usd"`
	LedgerJournalEntryNo     string  `json:"ledger_journal_entry_no"`
	ESGPerformanceVerdict    string  `json:"esg_performance_verdict"`
	AuditCertificateRef      string  `json:"audit_certificate_ref"`
	AdjustedAt               string  `json:"adjusted_at"`
}

// SLLPortfolioSummary returns consolidated financing metrics for corporate treasury.
type SLLPortfolioSummary struct {
	TotalFacilitiesCount     int     `json:"total_facilities_count"`
	TotalPrincipalFinancedUSD float64 `json:"total_principal_financed_usd"`
	WeightedAverageCouponPct float64 `json:"weighted_average_coupon_pct"`
	TotalCumulativeSavingsUSD float64 `json:"total_cumulative_savings_usd"`
	Facilities               []*SustainabilityLinkedFacility `json:"facilities"`
	LastValuationDate        string  `json:"last_valuation_date"`
}

// SLLFacilityStore manages thread-safe storage for SLL instruments.
type SLLFacilityStore struct {
	mu         sync.RWMutex
	facilities map[string]*SustainabilityLinkedFacility
	history    []SLLCouponAdjustmentResult
}

var globalSLLStore = &SLLFacilityStore{
	facilities: make(map[string]*SustainabilityLinkedFacility),
	history:    make([]SLLCouponAdjustmentResult, 0),
}

func init() {
	// Seed demo syndicated SLL facility
	demoFacility := &SustainabilityLinkedFacility{
		FacilityID:              "SLL-2026-SYNDICATE-01",
		FacilityName:            "USD 75M ESG Transition Revolving Facility",
		LenderSyndicateLead:     "Standard Chartered & DBS Bank Consortium",
		PrincipalAmountUSD:      75000000.0,
		BenchmarkRateName:       "SOFR_3M",
		BaseMarginBps:           175.0,
		StepDownDiscountBps:     25.0,
		StepUpPenaltyBps:        25.0,
		ActiveInterestRatePct:   6.35, // 4.85% SOFR + (175 - 25) bps
		AnnualInterestSavingUSD: 187500.0,
		Status:                  "ACTIVE_DISCOUNTED",
		CreatedAt:               time.Now().Format(time.RFC3339),
		Targets: []SustainabilityTargetKPI{
			{
				KPICode:         "SPT-GHG-01",
				KPIName:         "Scope 1 & 2 Emissions Intensity Reduction %",
				BaselineValue:   45.0,
				TargetThreshold: 15.0,
				ActualAchieved:  18.5,
				TargetMet:       true,
				WeightBps:       10.0,
			},
			{
				KPICode:         "SPT-RENEWABLE-02",
				KPIName:         "Renewable Electricity Share of Factory Grid %",
				BaselineValue:   20.0,
				TargetThreshold: 40.0,
				ActualAchieved:  46.2,
				TargetMet:       true,
				WeightBps:       10.0,
			},
			{
				KPICode:         "SPT-K3-03",
				KPIName:         "Workplace LTIFR Injury Frequency Ceiling",
				BaselineValue:   1.20,
				TargetThreshold: 0.80,
				ActualAchieved:  0.45,
				TargetMet:       true,
				WeightBps:       5.0,
			},
		},
	}
	globalSLLStore.facilities[demoFacility.FacilityID] = demoFacility
}

// EvaluateSLLCouponLogic audits ESG KPIs against contractual covenants and adjusts interest margins.
func EvaluateSLLCouponLogic(req EvaluateSLLCouponRequest) (*SLLCouponAdjustmentResult, error) {
	globalSLLStore.mu.Lock()
	defer globalSLLStore.mu.Unlock()

	fac, exists := globalSLLStore.facilities[req.FacilityID]
	if !exists {
		return nil, fmt.Errorf("sustainability-linked facility '%s' not found", req.FacilityID)
	}

	if req.CurrentSOFRRatePct <= 0 {
		req.CurrentSOFRRatePct = 4.85 // 4.85% base
	}
	if req.EvaluationPeriod == "" {
		req.EvaluationPeriod = "2026-Q1"
	}

	targetsMet := 0
	for i, t := range fac.Targets {
		switch t.KPICode {
		case "SPT-GHG-01":
			fac.Targets[i].ActualAchieved = req.AchievedGHGReductionPct
			fac.Targets[i].TargetMet = req.AchievedGHGReductionPct >= t.TargetThreshold
		case "SPT-RENEWABLE-02":
			fac.Targets[i].ActualAchieved = req.AchievedRenewableSharePct
			fac.Targets[i].TargetMet = req.AchievedRenewableSharePct >= t.TargetThreshold
		case "SPT-K3-03":
			fac.Targets[i].ActualAchieved = req.AchievedLTIFR
			fac.Targets[i].TargetMet = req.AchievedLTIFR <= t.TargetThreshold
		}
		if fac.Targets[i].TargetMet {
			targetsMet++
		}
	}

	totalTargets := len(fac.Targets)
	var marginAdjBps float64
	var verdict string

	if targetsMet == totalTargets {
		// All met: Max discount
		marginAdjBps = -fac.StepDownDiscountBps
		verdict = "OUTPERFORMING_GREEN_MAX_DISCOUNT"
		fac.Status = "ACTIVE_DISCOUNTED"
	} else if targetsMet >= 2 {
		marginAdjBps = -10.0 // partial incentive
		verdict = "PARTIAL_TARGET_MET_MODERATE_DISCOUNT"
		fac.Status = "ACTIVE_DISCOUNTED"
	} else {
		marginAdjBps = fac.StepUpPenaltyBps
		verdict = "TARGET_BREACH_PENALTY_STEPUP"
		fac.Status = "ACTIVE_PENALTY"
	}

	effectiveMarginPct := (fac.BaseMarginBps + marginAdjBps) / 100.0
	revisedCouponRate := req.CurrentSOFRRatePct + effectiveMarginPct

	annualInterestDeltaUSD := (fac.PrincipalAmountUSD * (marginAdjBps / 10000.0))
	fac.ActiveInterestRatePct = math.Round(revisedCouponRate*100) / 100
	fac.AnnualInterestSavingUSD = math.Abs(annualInterestDeltaUSD)

	glEntry := fmt.Sprintf("GL-FIN-ADJ-%s-%d", req.EvaluationPeriod, time.Now().UnixNano()%10000)
	certRef := fmt.Sprintf("AUD-ESG-CERT-SGS-%d", time.Now().UnixNano()%1000000)

	result := SLLCouponAdjustmentResult{
		FacilityID:                fac.FacilityID,
		EvaluationPeriod:          req.EvaluationPeriod,
		PrincipalAmountUSD:        fac.PrincipalAmountUSD,
		TotalTargetsCount:         totalTargets,
		TargetsMetCount:           targetsMet,
		PriorCouponRatePct:        req.CurrentSOFRRatePct + (fac.BaseMarginBps / 100.0),
		RevisedCouponRatePct:      fac.ActiveInterestRatePct,
		MarginAdjustmentBps:       marginAdjBps,
		NetAnnualInterestDeltaUSD: annualInterestDeltaUSD,
		LedgerJournalEntryNo:      glEntry,
		ESGPerformanceVerdict:     verdict,
		AuditCertificateRef:       certRef,
		AdjustedAt:                time.Now().Format(time.RFC3339),
	}

	globalSLLStore.history = append(globalSLLStore.history, result)
	return &result, nil
}

// GetSLLPortfolioSummaryLogic compiles portfolio-wide green debt metrics.
func GetSLLPortfolioSummaryLogic() (*SLLPortfolioSummary, error) {
	globalSLLStore.mu.RLock()
	defer globalSLLStore.mu.RUnlock()

	var totalPrincipal float64
	var weightedRateSum float64
	var totalSavings float64

	facList := make([]*SustainabilityLinkedFacility, 0, len(globalSLLStore.facilities))
	for _, f := range globalSLLStore.facilities {
		facList = append(facList, f)
		totalPrincipal += f.PrincipalAmountUSD
		weightedRateSum += f.PrincipalAmountUSD * f.ActiveInterestRatePct
		totalSavings += f.AnnualInterestSavingUSD
	}

	weightedAvgRate := 0.0
	if totalPrincipal > 0 {
		weightedAvgRate = weightedRateSum / totalPrincipal
	}

	return &SLLPortfolioSummary{
		TotalFacilitiesCount:      len(facList),
		TotalPrincipalFinancedUSD: totalPrincipal,
		WeightedAverageCouponPct:  math.Round(weightedAvgRate*100) / 100,
		TotalCumulativeSavingsUSD: math.Round(totalSavings*100) / 100,
		Facilities:                facList,
		LastValuationDate:         time.Now().Format("2006-01-02"),
	}, nil
}

// RegisterSLLRoutes registers the Sustainability-Linked Loan routes.
func RegisterSLLRoutes(r *gin.RouterGroup) {
	group := r.Group("/treasury/sll")
	{
		group.POST("/evaluate-kpi-coupon", func(c *gin.Context) {
			var req EvaluateSLLCouponRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid evaluate coupon payload: " + err.Error()})
				return
			}
			res, err := EvaluateSLLCouponLogic(req)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, res)
		})

		group.GET("/portfolio-summary", func(c *gin.Context) {
			res, err := GetSLLPortfolioSummaryLogic()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, res)
		})
	}
}
