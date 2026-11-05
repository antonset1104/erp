package finance

import (
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// CustomerBehavioralCreditProfile represents dynamic scoring profile
type CustomerBehavioralCreditProfile struct {
	CustomerID              string    `json:"customer_id"`
	CustomerName            string    `json:"customer_name"`
	BaseCreditLimitUSD      float64   `json:"base_credit_limit_usd"`
	CurrentARBalanceUSD     float64   `json:"current_ar_balance_usd"`
	OverdueBalanceUSD       float64   `json:"overdue_balance_usd"`
	HistoricalAverageDSO    float64   `json:"historical_average_dso_days"`
	AgreedTermsDays         int       `json:"agreed_terms_days"`
	DisputeFrequencyRate    float64   `json:"dispute_frequency_rate"` // e.g. 0.04 (4%)
	DynamicScore            float64   `json:"dynamic_score"`          // 0.0 - 100.0
	RiskTier                string    `json:"risk_tier"`              // "TIER_A_PRIME", "TIER_B_MODERATE", "TIER_C_ELEVATED_RISK", "TIER_D_DEFAULT_HOLD"
	AdjustedDynamicLimitUSD float64   `json:"adjusted_dynamic_limit_usd"`
	RecommendedAction       string    `json:"recommended_action"`
	LastEvaluatedAt         time.Time `json:"last_evaluated_at"`
}

// TradeCreditInsuranceCoverage represents external underwriter policy binding
type TradeCreditInsuranceCoverage struct {
	BindingID            string    `json:"binding_id"`
	CustomerID           string    `json:"customer_id"`
	PolicyNumber         string    `json:"policy_number"`
	UnderwriterName      string    `json:"underwriter_name"` // "Allianz Trade (Euler Hermes)", "Atradius", "PT Askrindo"
	InsuredLimitUSD      float64   `json:"insured_limit_usd"`
	CoverageRatioPercent float64   `json:"coverage_ratio_percent"` // e.g. 90.0%
	MaxDeductibleUSD     float64   `json:"max_deductible_usd"`
	PolicyStatus         string    `json:"policy_status"` // "ACTIVE_POLICY_BOUND", "SUSPENDED"
	BoundAt              time.Time `json:"bound_at"`
}

// OrderCreditHoldAssessment represents the real-time order clearance decision
type OrderCreditHoldAssessment struct {
	AssessmentID           string    `json:"assessment_id"`
	SalesOrderID           string    `json:"sales_order_id"`
	CustomerID             string    `json:"customer_id"`
	OrderAmountUSD         float64   `json:"order_amount_usd"`
	ProjectedTotalExposure float64   `json:"projected_total_exposure_usd"`
	DynamicLimitUSD        float64   `json:"dynamic_limit_usd"`
	InsuredLimitUSD        float64   `json:"insured_limit_usd"`
	UninsuredExposureUSD   float64   `json:"uninsured_exposure_usd"`
	Decision               string    `json:"decision"` // "APPROVED_DISPATCH_PERMITTED", "HOLD_DYNAMIC_LIMIT_EXCEEDED", "HOLD_OVERDUE_INVOICES_EXIST"
	BlockReason            string    `json:"block_reason,omitempty"`
	OverrideAllowedByCFO   bool      `json:"override_allowed_by_cfo"`
	EvaluatedAt            time.Time `json:"evaluated_at"`
}

// DynamicCreditVault thread-safe store
type DynamicCreditVault struct {
	mu          sync.RWMutex
	profiles    map[string]CustomerBehavioralCreditProfile
	insurances  map[string]TradeCreditInsuranceCoverage
	assessments map[string]OrderCreditHoldAssessment
}

// Global Dynamic Credit instance
var GlobalDynamicCreditVault = &DynamicCreditVault{
	profiles:    make(map[string]CustomerBehavioralCreditProfile),
	insurances:  make(map[string]TradeCreditInsuranceCoverage),
	assessments: make(map[string]OrderCreditHoldAssessment),
}

// ComputeDynamicScore evaluates behavioral payment velocity and updates credit capacity
func (v *DynamicCreditVault) ComputeDynamicScore(
	customerID, name string, baseLimit, currentAR, overdueAR, dso float64,
	termsDays int, disputeRate float64,
) CustomerBehavioralCreditProfile {
	v.mu.Lock()
	defer v.mu.Unlock()

	// Base score 100
	score := 100.0

	// Penalize DSO overshoot
	dsoOvershoot := dso - float64(termsDays)
	if dsoOvershoot > 0 {
		score -= dsoOvershoot * 1.5
	} else {
		score += 5.0 // Bonus for paying early
	}

	// Penalize overdue ratio
	if currentAR > 0 {
		overdueRatio := overdueAR / currentAR
		score -= overdueRatio * 40.0
	}

	// Penalize dispute frequency
	score -= disputeRate * 50.0

	if score > 100.0 {
		score = 100.0
	}
	if score < 0.0 {
		score = 0.0
	}
	score = math.Round(score*10) / 10

	var tier string
	var adjustedLimit float64
	var recAction string

	switch {
	case score >= 80.0:
		tier = "TIER_A_PRIME"
		adjustedLimit = baseLimit * 1.25 // Up to +25% expansion
		recAction = "Eligible for automatic credit expansion up to +25%."
	case score >= 60.0:
		tier = "TIER_B_MODERATE"
		adjustedLimit = baseLimit // Maintain base limit
		recAction = "Maintain current credit terms; monitor payment cycles."
	case score >= 40.0:
		tier = "TIER_C_ELEVATED_RISK"
		adjustedLimit = baseLimit * 0.70 // Constrain limit by 30%
		recAction = "Reduce credit limit by 30%; mandate trade credit insurance."
	default:
		tier = "TIER_D_DEFAULT_HOLD"
		adjustedLimit = 0.0 // Cut off credit
		recAction = "Immediate credit freeze; switch customer to Cash-On-Delivery (COD)."
	}

	profile := CustomerBehavioralCreditProfile{
		CustomerID:              customerID,
		CustomerName:            name,
		BaseCreditLimitUSD:      baseLimit,
		CurrentARBalanceUSD:     currentAR,
		OverdueBalanceUSD:       overdueAR,
		HistoricalAverageDSO:    dso,
		AgreedTermsDays:         termsDays,
		DisputeFrequencyRate:    disputeRate,
		DynamicScore:            score,
		RiskTier:                tier,
		AdjustedDynamicLimitUSD: adjustedLimit,
		RecommendedAction:       recAction,
		LastEvaluatedAt:         time.Now(),
	}

	v.profiles[customerID] = profile
	return profile
}

// BindInsuranceCoverage records underwriter policy limit
func (v *DynamicCreditVault) BindInsuranceCoverage(
	customerID, policyNo, underwriter string,
	insuredLimit, covRatio, deductible float64,
) TradeCreditInsuranceCoverage {
	v.mu.Lock()
	defer v.mu.Unlock()

	bindID := fmt.Sprintf("COV-BIND-%d", time.Now().UnixNano()%1000000)
	cov := TradeCreditInsuranceCoverage{
		BindingID:            bindID,
		CustomerID:           customerID,
		PolicyNumber:         policyNo,
		UnderwriterName:      underwriter,
		InsuredLimitUSD:      insuredLimit,
		CoverageRatioPercent: covRatio,
		MaxDeductibleUSD:     deductible,
		PolicyStatus:         "ACTIVE_POLICY_BOUND",
		BoundAt:              time.Now(),
	}

	v.insurances[customerID] = cov
	return cov
}

// EvaluateOrderHold performs automated pre-dispatch credit intercept
func (v *DynamicCreditVault) EvaluateOrderHold(
	salesOrderID, customerID string, orderAmount float64,
) (*OrderCreditHoldAssessment, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	profile, ok := v.profiles[customerID]
	if !ok {
		return nil, fmt.Errorf("credit profile not found for customer: %s", customerID)
	}

	insurance, hasInsurance := v.insurances[customerID]
	insuredLimit := 0.0
	if hasInsurance && insurance.PolicyStatus == "ACTIVE_POLICY_BOUND" {
		insuredLimit = insurance.InsuredLimitUSD
	}

	totalExposure := profile.CurrentARBalanceUSD + orderAmount
	uninsuredExp := 0.0
	if totalExposure > insuredLimit {
		uninsuredExp = totalExposure - insuredLimit
	}

	decision := "APPROVED_DISPATCH_PERMITTED"
	var blockReason string
	overrideAllowed := false

	if profile.OverdueBalanceUSD > 0 && profile.HistoricalAverageDSO > float64(profile.AgreedTermsDays+15) {
		decision = "HOLD_OVERDUE_INVOICES_EXIST"
		blockReason = fmt.Sprintf("Customer has overdue invoices totaling $%.2f USD exceeding grace period", profile.OverdueBalanceUSD)
		overrideAllowed = true
	} else if totalExposure > profile.AdjustedDynamicLimitUSD {
		decision = "HOLD_DYNAMIC_LIMIT_EXCEEDED"
		blockReason = fmt.Sprintf("Total exposure ($%.2f USD) exceeds dynamic credit ceiling ($%.2f USD)", totalExposure, profile.AdjustedDynamicLimitUSD)
		overrideAllowed = true
	}

	assessID := fmt.Sprintf("HOLD-DEC-%d", time.Now().UnixNano()%1000000)
	assessment := OrderCreditHoldAssessment{
		AssessmentID:           assessID,
		SalesOrderID:           salesOrderID,
		CustomerID:             customerID,
		OrderAmountUSD:         orderAmount,
		ProjectedTotalExposure: totalExposure,
		DynamicLimitUSD:        profile.AdjustedDynamicLimitUSD,
		InsuredLimitUSD:        insuredLimit,
		UninsuredExposureUSD:   uninsuredExp,
		Decision:               decision,
		BlockReason:            blockReason,
		OverrideAllowedByCFO:   overrideAllowed,
		EvaluatedAt:            time.Now(),
	}

	v.assessments[salesOrderID] = assessment
	return &assessment, nil
}

// GetCustomerRiskProfile retrieves 360-degree credit dossier
func (v *DynamicCreditVault) GetCustomerRiskProfile(customerID string) (gin.H, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	profile, hasProfile := v.profiles[customerID]
	insurance, hasInsurance := v.insurances[customerID]

	if !hasProfile {
		return nil, fmt.Errorf("no profile found for customer: %s", customerID)
	}

	return gin.H{
		"customer_id":          customerID,
		"credit_profile":       profile,
		"insurance_coverage":   insurance,
		"has_insurance":        hasInsurance,
		"underwriter_security": "TRADE_CREDIT_DEFAULT_PROTECTED",
		"credit_engine":        "GoERP AI Dynamic Credit Scoring v1.62.0",
	}, nil
}

// RegisterCreditRoutes registers HTTP endpoints
func RegisterCreditRoutes(r *gin.RouterGroup) {
	cred := r.Group("/finance/credit")
	{
		cred.POST("/compute-dynamic-score", func(c *gin.Context) {
			var req struct {
				CustomerID           string  `json:"customer_id" binding:"required"`
				CustomerName         string  `json:"customer_name" binding:"required"`
				BaseCreditLimitUSD   float64 `json:"base_credit_limit_usd" binding:"required"`
				CurrentARBalanceUSD  float64 `json:"current_ar_balance_usd"`
				OverdueBalanceUSD    float64 `json:"overdue_balance_usd"`
				HistoricalAverageDSO float64 `json:"historical_average_dso_days" binding:"required"`
				AgreedTermsDays      int     `json:"agreed_terms_days" binding:"required"`
				DisputeFrequencyRate float64 `json:"dispute_frequency_rate"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			profile := GlobalDynamicCreditVault.ComputeDynamicScore(
				req.CustomerID, req.CustomerName, req.BaseCreditLimitUSD,
				req.CurrentARBalanceUSD, req.OverdueBalanceUSD, req.HistoricalAverageDSO,
				req.AgreedTermsDays, req.DisputeFrequencyRate,
			)
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": profile})
		})

		cred.POST("/bind-insurance-coverage", func(c *gin.Context) {
			var req struct {
				CustomerID           string  `json:"customer_id" binding:"required"`
				PolicyNumber         string  `json:"policy_number" binding:"required"`
				UnderwriterName      string  `json:"underwriter_name" binding:"required"`
				InsuredLimitUSD      float64 `json:"insured_limit_usd" binding:"required"`
				CoverageRatioPercent float64 `json:"coverage_ratio_percent" binding:"required"`
				MaxDeductibleUSD     float64 `json:"max_deductible_usd"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			coverage := GlobalDynamicCreditVault.BindInsuranceCoverage(
				req.CustomerID, req.PolicyNumber, req.UnderwriterName,
				req.InsuredLimitUSD, req.CoverageRatioPercent, req.MaxDeductibleUSD,
			)
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": coverage})
		})

		cred.POST("/evaluate-order-hold", func(c *gin.Context) {
			var req struct {
				SalesOrderID   string  `json:"sales_order_id" binding:"required"`
				CustomerID     string  `json:"customer_id" binding:"required"`
				OrderAmountUSD float64 `json:"order_amount_usd" binding:"required"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			assessment, err := GlobalDynamicCreditVault.EvaluateOrderHold(
				req.SalesOrderID, req.CustomerID, req.OrderAmountUSD,
			)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": assessment})
		})

		cred.GET("/customer-risk-profile/:customer_id", func(c *gin.Context) {
			customerID := c.Param("customer_id")
			profile, err := GlobalDynamicCreditVault.GetCustomerRiskProfile(customerID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": profile})
		})
	}
}
