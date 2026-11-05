package account

import (
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// PerformanceObligation represents a distinct good or service in an IFRS 15 bundled contract.
type PerformanceObligation struct {
	POBID                        string  `json:"pob_id"`
	Description                  string  `json:"description"`
	StandaloneSellingPriceIDR    float64 `json:"standalone_selling_price_idr"` // SSP
	AllocatedTransactionPriceIDR float64 `json:"allocated_transaction_price_idr"`
	DiscountAllocatedIDR         float64 `json:"discount_allocated_idr"`
	RecognitionPattern           string  `json:"recognition_pattern"` // "POINT_IN_TIME", "OVER_TIME_MONTHLY"
	DurationMonths               int     `json:"duration_months"`
	MonthlyAmortizationIDR       float64 `json:"monthly_amortization_idr"`
}

// SSPAllocationRequest holds bundled contracts to allocate transaction price according to relative SSP.
type SSPAllocationRequest struct {
	ContractID               string                  `json:"contract_id"`
	CustomerName             string                  `json:"customer_name"`
	TotalTransactionPriceIDR float64                 `json:"total_transaction_price_idr"` // Discounted bundle price
	Obligations              []PerformanceObligation `json:"obligations"`
}

// SSPAllocationResult summarizes relative standalone price allocations and monthly revenue schedule.
type SSPAllocationResult struct {
	ContractID               string                  `json:"contract_id"`
	CustomerName             string                  `json:"customer_name"`
	TotalStandalonePriceIDR  float64                 `json:"total_standalone_price_idr"`
	TotalTransactionPriceIDR float64                 `json:"total_transaction_price_idr"`
	TotalBundleDiscountIDR   float64                 `json:"total_bundle_discount_idr"`
	DiscountPercentage       float64                 `json:"discount_percentage"`
	AllocatedObligations     []PerformanceObligation `json:"allocated_obligations"`
	AllocatedAt              string                  `json:"allocated_at"`
}

// ContractModificationRequest represents an addendum altering scope, price, or seats mid-contract.
type ContractModificationRequest struct {
	ContractID                string  `json:"contract_id"`
	ModificationType          string  `json:"modification_type"` // "CUMULATIVE_CATCH_UP", "PROSPECTIVE_ADJUSTMENT"
	AdditionalPriceIDR        float64 `json:"additional_price_idr"`
	RemainingMonthsToAmortize int     `json:"remaining_months_to_amortize"`
	Reason                    string  `json:"reason"`
}

// ContractModificationResult details the catch-up adjustment or prospective monthly rate revision.
type ContractModificationResult struct {
	ContractID                  string  `json:"contract_id"`
	ModificationType            string  `json:"modification_type"`
	CumulativeCatchUpRevenueIDR float64 `json:"cumulative_catch_up_revenue_idr"`
	RevisedMonthlyRevenueIDR    float64 `json:"revised_monthly_revenue_idr"`
	RemainingMonths             int     `json:"remaining_months"`
	AccountingImpactNote        string  `json:"accounting_impact_note"`
	ModifiedAt                  string  `json:"modified_at"`
}

var (
	revRecMutex sync.RWMutex
	revRecStore = make(map[string]*SSPAllocationResult)
)

// AllocateSSPLogic calculates relative Standalone Selling Price (SSP) allocation under IFRS 15.
func AllocateSSPLogic(req SSPAllocationRequest) SSPAllocationResult {
	revRecMutex.Lock()
	defer revRecMutex.Unlock()

	if req.ContractID == "" {
		req.ContractID = fmt.Sprintf("IFRS15-CTR-%s-%03d", time.Now().Format("20060102"), time.Now().Unix()%1000)
	}
	if req.CustomerName == "" {
		req.CustomerName = "PT Telekomunikasi Enterprise Nusantara"
	}
	if req.TotalTransactionPriceIDR <= 0 {
		req.TotalTransactionPriceIDR = 1200000000.0 // Rp 1.2 Miliar bundle price
	}

	if len(req.Obligations) == 0 {
		req.Obligations = []PerformanceObligation{
			{POBID: "POB-01-HARDWARE", Description: "Industrial IoT Edge Gateways & Barcode Scanners", StandaloneSellingPriceIDR: 450000000.0, RecognitionPattern: "POINT_IN_TIME", DurationMonths: 1},
			{POBID: "POB-02-SOFTWARE-LICENSE", Description: "Enterprise Core ERP & WMS 3-Year Subscription", StandaloneSellingPriceIDR: 600000000.0, RecognitionPattern: "OVER_TIME_MONTHLY", DurationMonths: 36},
			{POBID: "POB-03-IMPLEMENTATION", Description: "Business Process Re-engineering & System Implementation", StandaloneSellingPriceIDR: 300000000.0, RecognitionPattern: "POINT_IN_TIME", DurationMonths: 3},
			{POBID: "POB-04-PREMIUM-SLA", Description: "24/7 Mission-Critical SLA & Database Maintenance", StandaloneSellingPriceIDR: 150000000.0, RecognitionPattern: "OVER_TIME_MONTHLY", DurationMonths: 36},
		}
	}

	var totalSSP float64
	for _, o := range req.Obligations {
		totalSSP += o.StandaloneSellingPriceIDR
	}

	totalDiscount := totalSSP - req.TotalTransactionPriceIDR
	if totalDiscount < 0 {
		totalDiscount = 0
	}
	discPct := 0.0
	if totalSSP > 0 {
		discPct = math.Round((totalDiscount/totalSSP)*10000) / 100.0
	}

	var allocated []PerformanceObligation
	for _, o := range req.Obligations {
		allocRatio := o.StandaloneSellingPriceIDR / totalSSP
		allocatedPrice := math.Round(req.TotalTransactionPriceIDR * allocRatio)
		discAlloc := o.StandaloneSellingPriceIDR - allocatedPrice

		monthlyAmort := 0.0
		if o.RecognitionPattern == "OVER_TIME_MONTHLY" && o.DurationMonths > 0 {
			monthlyAmort = math.Round(allocatedPrice / float64(o.DurationMonths))
		}

		o.AllocatedTransactionPriceIDR = allocatedPrice
		o.DiscountAllocatedIDR = discAlloc
		o.MonthlyAmortizationIDR = monthlyAmort
		allocated = append(allocated, o)
	}

	res := SSPAllocationResult{
		ContractID:               req.ContractID,
		CustomerName:             req.CustomerName,
		TotalStandalonePriceIDR:  totalSSP,
		TotalTransactionPriceIDR: req.TotalTransactionPriceIDR,
		TotalBundleDiscountIDR:   totalDiscount,
		DiscountPercentage:       discPct,
		AllocatedObligations:     allocated,
		AllocatedAt:              time.Now().Format(time.RFC3339),
	}

	revRecStore[req.ContractID] = &res
	return res
}

// ModifyContractLogic recalculates revenue recognition under contract modification rules.
func ModifyContractLogic(req ContractModificationRequest) ContractModificationResult {
	if req.ContractID == "" {
		req.ContractID = "IFRS15-CTR-20261104-001"
	}
	if req.ModificationType == "" {
		req.ModificationType = "PROSPECTIVE_ADJUSTMENT"
	}
	if req.AdditionalPriceIDR <= 0 {
		req.AdditionalPriceIDR = 240000000.0 // Rp 240 Juta additional consideration
	}
	if req.RemainingMonthsToAmortize <= 0 {
		req.RemainingMonthsToAmortize = 24
	}

	var catchUp float64
	var revisedMonthly float64
	var note string

	if req.ModificationType == "CUMULATIVE_CATCH_UP" {
		// Single performance obligation partially satisfied: immediate catch-up recognition
		catchUp = math.Round(req.AdditionalPriceIDR * 0.35)
		revisedMonthly = math.Round((req.AdditionalPriceIDR - catchUp) / float64(req.RemainingMonthsToAmortize))
		note = fmt.Sprintf("Cumulative catch-up of Rp %.0f recognized immediately in P&L. Remaining amortized at Rp %.0f/month.", catchUp, revisedMonthly)
	} else {
		// Distinct remaining goods/services treated prospectively: add to remaining performance obligations
		revisedMonthly = math.Round(req.AdditionalPriceIDR / float64(req.RemainingMonthsToAmortize))
		note = fmt.Sprintf("Contract modification treated prospectively. Additional revenue spread at Rp %.0f/month over remaining %d months.", revisedMonthly, req.RemainingMonthsToAmortize)
	}

	return ContractModificationResult{
		ContractID:                  req.ContractID,
		ModificationType:            req.ModificationType,
		CumulativeCatchUpRevenueIDR: catchUp,
		RevisedMonthlyRevenueIDR:    revisedMonthly,
		RemainingMonths:             req.RemainingMonthsToAmortize,
		AccountingImpactNote:        note,
		ModifiedAt:                  time.Now().Format(time.RFC3339),
	}
}

// RegisterRevRecRoutes registers the IFRS 15 Multi-Element Revenue Recognition routes.
func RegisterRevRecRoutes(r *gin.RouterGroup) {
	group := r.Group("/accounting/revrec")
	{
		group.POST("/allocate-ssp", func(c *gin.Context) {
			var req SSPAllocationRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				req = SSPAllocationRequest{}
			}
			res := AllocateSSPLogic(req)
			c.JSON(http.StatusOK, res)
		})

		group.POST("/modify-contract", func(c *gin.Context) {
			var req ContractModificationRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				req = ContractModificationRequest{}
			}
			res := ModifyContractLogic(req)
			c.JSON(http.StatusOK, res)
		})
	}
}
