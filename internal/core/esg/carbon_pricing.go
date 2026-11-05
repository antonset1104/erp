package esg

import (
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// CapexCarbonEvaluationRequest holds investment proposal data for internal shadow carbon pricing.
type CapexCarbonEvaluationRequest struct {
	ProposalID                     string  `json:"proposal_id"`
	ProjectName                    string  `json:"project_name"`
	BusinessUnit                   string  `json:"business_unit"`
	BaselineCapexIDR               float64 `json:"baseline_capex_idr"`
	AnnualOpexIDR                  float64 `json:"annual_opex_idr"`
	ProjectLifeYears               int     `json:"project_life_years"`
	ProjectedEmissionsTonsCO2eYear float64 `json:"projected_emissions_tons_co2e_year"`
	ShadowCarbonPriceUSDPerTon     float64 `json:"shadow_carbon_price_usd_per_ton"` // e.g. $75/ton CO2e
	FXRateUSDToIDR                 float64 `json:"fx_rate_usd_to_idr"`              // e.g. 16300 IDR/USD
	IsGreenInitiative              bool    `json:"is_green_initiative"`
}

// CapexCarbonEvaluationResult provides the carbon-adjusted TCO and green fund allocation.
type CapexCarbonEvaluationResult struct {
	ProposalID                 string  `json:"proposal_id"`
	ProjectName                string  `json:"project_name"`
	BusinessUnit               string  `json:"business_unit"`
	BaselineCapexIDR           float64 `json:"baseline_capex_idr"`
	AnnualShadowCarbonFeeIDR   float64 `json:"annual_shadow_carbon_fee_idr"`
	LifetimeCarbonFeeIDR       float64 `json:"lifetime_carbon_fee_idr"`
	CarbonAdjustedTCOIDR       float64 `json:"carbon_adjusted_tco_idr"`
	GreenFundSubsidyGrantedIDR float64 `json:"green_fund_subsidy_granted_idr"`
	NetApprovedInvestmentIDR   float64 `json:"net_approved_investment_idr"`
	ApprovalStatus             string  `json:"approval_status"` // "APPROVED_WITH_CARBON_LEVY", "APPROVED_WITH_GREEN_SUBSIDY", "REJECTED_CARBON_EXCESSIVE"
	CarbonIntensityVerdict     string  `json:"carbon_intensity_verdict"`
	EvaluatedAt                string  `json:"evaluated_at"`
}

// GreenTaxPoolSummary manages internal carbon levies collected for corporate sustainability reinvestment.
type GreenTaxPoolSummary struct {
	TotalCarbonLeviesCollectedIDR   float64                       `json:"total_carbon_levies_collected_idr"`
	TotalGreenSubsidiesDisbursedIDR float64                       `json:"total_green_subsidies_disbursed_idr"`
	AvailableGreenFundBalanceIDR    float64                       `json:"available_green_fund_balance_idr"`
	ShadowPriceAppliedUSD           float64                       `json:"shadow_price_applied_usd"`
	EvaluatedProposals              []CapexCarbonEvaluationResult `json:"evaluated_proposals"`
}

var (
	carbonMutex sync.RWMutex
	carbonPool  = &GreenTaxPoolSummary{
		TotalCarbonLeviesCollectedIDR:   1450000000.0, // Rp 1.45 Miliar collected
		TotalGreenSubsidiesDisbursedIDR: 620000000.0,  // Rp 620 Juta disbursed
		AvailableGreenFundBalanceIDR:    830000000.0,  // Rp 830 Juta available
		ShadowPriceAppliedUSD:           75.0,
		EvaluatedProposals:              []CapexCarbonEvaluationResult{},
	}
)

// EvaluateCapexCarbonLogic applies internal shadow carbon pricing on capital expenditure proposals.
func EvaluateCapexCarbonLogic(req CapexCarbonEvaluationRequest) CapexCarbonEvaluationResult {
	carbonMutex.Lock()
	defer carbonMutex.Unlock()

	if req.ProposalID == "" {
		req.ProposalID = fmt.Sprintf("CPX-CARB-%s-%03d", time.Now().Format("20060102"), time.Now().Unix()%1000)
	}
	if req.ProjectName == "" {
		req.ProjectName = "Solar PV Rooftop & Fleet Electrification Phase 1"
	}
	if req.BusinessUnit == "" {
		req.BusinessUnit = "Supply Chain & Factory Logistics"
	}
	if req.BaselineCapexIDR <= 0 {
		req.BaselineCapexIDR = 2500000000.0 // Rp 2.5 Miliar
	}
	if req.ProjectLifeYears <= 0 {
		req.ProjectLifeYears = 5
	}
	if req.ShadowCarbonPriceUSDPerTon <= 0 {
		req.ShadowCarbonPriceUSDPerTon = 75.0 // $75/ton shadow carbon price
	}
	if req.FXRateUSDToIDR <= 0 {
		req.FXRateUSDToIDR = 16300.0
	}

	pricePerTonIDR := req.ShadowCarbonPriceUSDPerTon * req.FXRateUSDToIDR
	annualCarbonFee := req.ProjectedEmissionsTonsCO2eYear * pricePerTonIDR
	lifetimeCarbonFee := annualCarbonFee * float64(req.ProjectLifeYears)

	nominalTCO := req.BaselineCapexIDR + (req.AnnualOpexIDR * float64(req.ProjectLifeYears))
	carbonAdjustedTCO := nominalTCO + lifetimeCarbonFee

	var subsidyGranted float64
	approvalStatus := "APPROVED_WITH_CARBON_LEVY"
	verdict := ""

	if req.IsGreenInitiative || req.ProjectedEmissionsTonsCO2eYear < 50.0 {
		// Green initiative receives up to 25% Capex grant from internal green pool
		subsidyGranted = math.Min(req.BaselineCapexIDR*0.25, carbonPool.AvailableGreenFundBalanceIDR*0.5)
		approvalStatus = "APPROVED_WITH_GREEN_SUBSIDY"
		carbonPool.TotalGreenSubsidiesDisbursedIDR += subsidyGranted
		carbonPool.AvailableGreenFundBalanceIDR -= subsidyGranted
		verdict = fmt.Sprintf("High-priority decarbonization project. Awarded Rp %.0f internal green subsidy.", subsidyGranted)
	} else if lifetimeCarbonFee > req.BaselineCapexIDR*0.5 {
		approvalStatus = "REJECTED_CARBON_EXCESSIVE"
		verdict = fmt.Sprintf("Proposal shadow carbon penalty (Rp %.0f) exceeds 50%% of Capex. Redesign required.", lifetimeCarbonFee)
	} else {
		carbonPool.TotalCarbonLeviesCollectedIDR += lifetimeCarbonFee
		carbonPool.AvailableGreenFundBalanceIDR += lifetimeCarbonFee
		verdict = fmt.Sprintf("Approved with shadow carbon fee of Rp %.0f/year transferred to Green Innovation Fund.", annualCarbonFee)
	}

	netInvestment := req.BaselineCapexIDR - subsidyGranted
	if netInvestment < 0 {
		netInvestment = 0
	}

	res := CapexCarbonEvaluationResult{
		ProposalID:                 req.ProposalID,
		ProjectName:                req.ProjectName,
		BusinessUnit:               req.BusinessUnit,
		BaselineCapexIDR:           req.BaselineCapexIDR,
		AnnualShadowCarbonFeeIDR:   math.Round(annualCarbonFee),
		LifetimeCarbonFeeIDR:       math.Round(lifetimeCarbonFee),
		CarbonAdjustedTCOIDR:       math.Round(carbonAdjustedTCO),
		GreenFundSubsidyGrantedIDR: math.Round(subsidyGranted),
		NetApprovedInvestmentIDR:   math.Round(netInvestment),
		ApprovalStatus:             approvalStatus,
		CarbonIntensityVerdict:     verdict,
		EvaluatedAt:                time.Now().Format(time.RFC3339),
	}

	carbonPool.EvaluatedProposals = append(carbonPool.EvaluatedProposals, res)
	return res
}

// GetGreenTaxPoolSummaryLogic returns corporate carbon tax reserves and green subsidy statistics.
func GetGreenTaxPoolSummaryLogic() GreenTaxPoolSummary {
	carbonMutex.RLock()
	defer carbonMutex.RUnlock()
	return *carbonPool
}

// RegisterCarbonPricingRoutes registers the Internal Carbon Pricing routes.
func RegisterCarbonPricingRoutes(r *gin.RouterGroup) {
	group := r.Group("/esg/carbon-pricing")
	{
		group.POST("/evaluate-capex", func(c *gin.Context) {
			var req CapexCarbonEvaluationRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				req = CapexCarbonEvaluationRequest{}
			}
			res := EvaluateCapexCarbonLogic(req)
			c.JSON(http.StatusOK, res)
		})

		group.GET("/internal-tax-pool", func(c *gin.Context) {
			summary := GetGreenTaxPoolSummaryLogic()
			c.JSON(http.StatusOK, summary)
		})
	}
}
