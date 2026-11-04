package cpq

import (
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// DealApprovalAction captures human or policy sign-off on a non-standard commercial deal.
type DealApprovalAction struct {
	ActionID     string `json:"action_id"`
	ApproverRole string `json:"approver_role"` // "SALES_DIRECTOR", "CFO", "GENERAL_COUNSEL"
	ApproverName string `json:"approver_name"`
	Decision     string `json:"decision"` // "APPROVED", "CONDITIONALLY_APPROVED", "REJECTED"
	Remarks      string `json:"remarks"`
	Timestamp    string `json:"timestamp"`
}

// DealQuote represents an enterprise B2B sales proposal subject to Deal Desk governance.
type DealQuote struct {
	QuoteID                string               `json:"quote_id"` // e.g. "QT-2026-ENT-771"
	CustomerName           string               `json:"customer_name"`
	IndustrySegment        string               `json:"industry_segment"`
	ContractTermMonths     int                  `json:"contract_term_months"`
	ListPrice              float64              `json:"list_price"`
	OfferedDiscountPct     float64              `json:"offered_discount_pct"`
	NetPrice               float64              `json:"net_price"`
	TotalEstimatedCost     float64              `json:"total_estimated_cost"`
	GrossMarginPct         float64              `json:"gross_margin_pct"`
	NonStandardTerms       []string             `json:"non_standard_terms"`
	DealDeskStatus         string               `json:"deal_desk_status"` // "AUTO_APPROVED", "PENDING_SALES_DIRECTOR", "PENDING_CFO_APPROVAL", "REJECTED_MARGIN_BREACH"
	ApprovalThresholdRule  string               `json:"approval_threshold_rule"`
	MarginLeakageRiskScore float64              `json:"margin_leakage_risk_score"` // 0.0 - 100.0
	ApprovalHistory        []DealApprovalAction `json:"approval_history"`
	CreatedAt              string               `json:"created_at"`
}

// EvaluateQuoteRequest holds input parameters to price and evaluate a quote against margin gates.
type EvaluateQuoteRequest struct {
	CustomerName       string   `json:"customer_name"`
	IndustrySegment    string   `json:"industry_segment"`
	ContractTermMonths int      `json:"contract_term_months"`
	ListPrice          float64  `json:"list_price"`
	OfferedDiscountPct float64  `json:"offered_discount_pct"`
	TotalEstimatedCost float64  `json:"total_estimated_cost"`
	NonStandardTerms   []string `json:"non_standard_terms"`
}

// ApproveDiscountRequest contains sign-off action parameters by authorized management.
type ApproveDiscountRequest struct {
	QuoteID      string `json:"quote_id"`
	ApproverRole string `json:"approver_role"`
	ApproverName string `json:"approver_name"`
	Decision     string `json:"decision"`
	Remarks      string `json:"remarks"`
}

// DealDeskStore provides thread-safe storage for deal quotes and audit logs.
type DealDeskStore struct {
	mu     sync.RWMutex
	quotes map[string]*DealQuote
}

var globalDealDeskStore = &DealDeskStore{
	quotes: make(map[string]*DealQuote),
}

func init() {
	// Seed demo enterprise quote
	demoQuote := &DealQuote{
		QuoteID:                "QT-2026-ENT-771",
		CustomerName:           "PT Global Petrokimia Mandiri Tbk",
		IndustrySegment:        "OIL_AND_GAS_ENERGY",
		ContractTermMonths:     36,
		ListPrice:              1200000.0,
		OfferedDiscountPct:     22.0,
		NetPrice:               936000.0,
		TotalEstimatedCost:     620000.0,
		GrossMarginPct:         33.76,
		NonStandardTerms:       []string{"Net 60 Days Payment", "Custom SLA 99.95%"},
		DealDeskStatus:         "PENDING_SALES_DIRECTOR",
		ApprovalThresholdRule:  "Discount > 10% requires Commercial Sales Director sign-off",
		MarginLeakageRiskScore: 35.5,
		ApprovalHistory:        make([]DealApprovalAction, 0),
		CreatedAt:              time.Now().Format(time.RFC3339),
	}
	globalDealDeskStore.quotes[demoQuote.QuoteID] = demoQuote
}

// EvaluateDealQuoteLogic applies margin leakage detection and routes to appropriate approval tier.
func EvaluateDealQuoteLogic(req EvaluateQuoteRequest) (*DealQuote, error) {
	globalDealDeskStore.mu.Lock()
	defer globalDealDeskStore.mu.Unlock()

	if req.ListPrice <= 0 {
		return nil, fmt.Errorf("list price must be greater than zero")
	}
	if req.TotalEstimatedCost <= 0 {
		req.TotalEstimatedCost = req.ListPrice * 0.50
	}
	if req.ContractTermMonths <= 0 {
		req.ContractTermMonths = 12
	}

	netPrice := req.ListPrice * (1.0 - (req.OfferedDiscountPct / 100.0))
	grossMargin := 0.0
	if netPrice > 0 {
		grossMargin = ((netPrice - req.TotalEstimatedCost) / netPrice) * 100.0
	}

	// Calculate Margin Leakage Risk Score (0 - 100)
	riskScore := (req.OfferedDiscountPct * 1.5)
	if grossMargin < 20.0 {
		riskScore += 30.0
	}
	hasUnlimitedLiability := false
	for _, term := range req.NonStandardTerms {
		riskScore += 10.0
		if term == "Unlimited Liability Cap" || term == "Penalty SLA 99.99%" {
			hasUnlimitedLiability = true
			riskScore += 25.0
		}
	}
	if riskScore > 100.0 {
		riskScore = 100.0
	}

	status := "AUTO_APPROVED"
	rule := "Standard commercial pricing within sales rep empowerment authority"

	if grossMargin < 12.0 {
		status = "REJECTED_MARGIN_BREACH"
		rule = fmt.Sprintf("Gross margin %.2f%% is below minimum company profitability threshold of 12%%", grossMargin)
	} else if req.OfferedDiscountPct > 25.0 || grossMargin < 25.0 || hasUnlimitedLiability {
		status = "PENDING_CFO_APPROVAL"
		rule = "High discount (>25%), low margin (<25%), or material contractual liability requires CFO & GC approval"
	} else if req.OfferedDiscountPct > 10.0 || grossMargin < 35.0 || len(req.NonStandardTerms) > 0 {
		status = "PENDING_SALES_DIRECTOR"
		rule = "Discount between 10%-25% or non-standard commercial terms requires Sales Director approval"
	}

	quoteID := fmt.Sprintf("QT-2026-ENT-%03d", len(globalDealDeskStore.quotes)+772)
	quote := &DealQuote{
		QuoteID:                quoteID,
		CustomerName:           req.CustomerName,
		IndustrySegment:        req.IndustrySegment,
		ContractTermMonths:     req.ContractTermMonths,
		ListPrice:              req.ListPrice,
		OfferedDiscountPct:     req.OfferedDiscountPct,
		NetPrice:               math.Round(netPrice*100) / 100,
		TotalEstimatedCost:     req.TotalEstimatedCost,
		GrossMarginPct:         math.Round(grossMargin*100) / 100,
		NonStandardTerms:       req.NonStandardTerms,
		DealDeskStatus:         status,
		ApprovalThresholdRule:  rule,
		MarginLeakageRiskScore: math.Round(riskScore*100) / 100,
		ApprovalHistory:        make([]DealApprovalAction, 0),
		CreatedAt:              time.Now().Format(time.RFC3339),
	}

	globalDealDeskStore.quotes[quoteID] = quote
	return quote, nil
}

// ApproveDiscountLogic records executive governance sign-off.
func ApproveDiscountLogic(req ApproveDiscountRequest) (*DealQuote, error) {
	globalDealDeskStore.mu.Lock()
	defer globalDealDeskStore.mu.Unlock()

	quote, exists := globalDealDeskStore.quotes[req.QuoteID]
	if !exists {
		return nil, fmt.Errorf("quote '%s' not found", req.QuoteID)
	}
	if quote.DealDeskStatus == "REJECTED_MARGIN_BREACH" {
		return nil, fmt.Errorf("cannot approve quote '%s' marked as REJECTED_MARGIN_BREACH", req.QuoteID)
	}

	action := DealApprovalAction{
		ActionID:     fmt.Sprintf("ACT-%d", time.Now().UnixNano()%1000000),
		ApproverRole: req.ApproverRole,
		ApproverName: req.ApproverName,
		Decision:     req.Decision,
		Remarks:      req.Remarks,
		Timestamp:    time.Now().Format(time.RFC3339),
	}

	quote.ApprovalHistory = append(quote.ApprovalHistory, action)

	if req.Decision == "APPROVED" || req.Decision == "CONDITIONALLY_APPROVED" {
		quote.DealDeskStatus = "APPROVED_BY_MANAGEMENT"
	} else if req.Decision == "REJECTED" {
		quote.DealDeskStatus = "REJECTED_BY_MANAGEMENT"
	}

	return quote, nil
}

// RegisterDealDeskRoutes registers CPQ Deal Desk API endpoints.
func RegisterDealDeskRoutes(r *gin.RouterGroup) {
	group := r.Group("/cpq/deal-desk")
	{
		group.POST("/evaluate-quote", func(c *gin.Context) {
			var req EvaluateQuoteRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid evaluate quote payload: " + err.Error()})
				return
			}
			res, err := EvaluateDealQuoteLogic(req)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, res)
		})

		group.POST("/approve-discount", func(c *gin.Context) {
			var req ApproveDiscountRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid approve discount payload: " + err.Error()})
				return
			}
			res, err := ApproveDiscountLogic(req)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, res)
		})

		group.GET("/audit-trail", func(c *gin.Context) {
			globalDealDeskStore.mu.RLock()
			defer globalDealDeskStore.mu.RUnlock()

			list := make([]*DealQuote, 0, len(globalDealDeskStore.quotes))
			for _, q := range globalDealDeskStore.quotes {
				list = append(list, q)
			}
			c.JSON(http.StatusOK, gin.H{
				"total_quotes": len(list),
				"quotes":       list,
			})
		})
	}
}
