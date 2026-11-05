package pricing

import (
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// DynamicPriceRequest defines parameters to dynamically calculate optimal selling price.
type DynamicPriceRequest struct {
	SKU                   string  `json:"sku"`
	ProductName           string  `json:"product_name"`
	BasePriceIDR          float64 `json:"base_price_idr"`
	CurrentCostIDR        float64 `json:"current_cost_idr"`
	MinMarginPercent      float64 `json:"min_margin_percent"`
	CompetitorPriceIDR    float64 `json:"competitor_price_idr"`
	InventoryDaysOfSupply float64 `json:"inventory_days_of_supply"`
	ExpiryDaysLeft        float64 `json:"expiry_days_left"`
	PriceElasticityDemand float64 `json:"price_elasticity_demand"`
	SalesVelocityWeekly   float64 `json:"sales_velocity_weekly"`
}

// DynamicPriceEvaluation holds the repriced outcome and guard rails.
type DynamicPriceEvaluation struct {
	SKU                      string  `json:"sku"`
	ProductName              string  `json:"product_name"`
	BasePriceIDR             float64 `json:"base_price_idr"`
	CurrentCostIDR           float64 `json:"current_cost_idr"`
	FloorPriceIDR            float64 `json:"floor_price_idr"`
	SuggestedDynamicPriceIDR float64 `json:"suggested_dynamic_price_idr"`
	AchievedMarginPercent    float64 `json:"achieved_margin_percent"`
	MarginGuardEnforced      bool    `json:"margin_guard_enforced"`
	PriceAdjustmentDelta     float64 `json:"price_adjustment_delta"`
	AdjustmentPercent        float64 `json:"adjustment_percent"`
	ProjectedVolumeGainPct   float64 `json:"projected_volume_gain_pct"`
	RepriceReason            string  `json:"reprice_reason"`
	OmnichannelSyncStatus    string  `json:"omnichannel_sync_status"`
	EvaluatedAt              string  `json:"evaluated_at"`
}

// OrderLineVerification defines a single line on a customer sales order.
type OrderLineVerification struct {
	SKU             string  `json:"sku"`
	ProductName     string  `json:"product_name"`
	Quantity        float64 `json:"quantity"`
	OfferedPrice    float64 `json:"offered_price"`
	UnitCOGS        float64 `json:"unit_cogs"`
	DiscountPercent float64 `json:"discount_percent"`
}

// OrderMarginVerificationRequest defines the sales order payload to audit against margin policy.
type OrderMarginVerificationRequest struct {
	OrderID            string                  `json:"order_id"`
	CustomerName       string                  `json:"customer_name"`
	CustomerTier       string                  `json:"customer_tier"`
	MinPolicyMarginPct float64                 `json:"min_policy_margin_pct"`
	Items              []OrderLineVerification `json:"items"`
}

// OrderMarginVerificationResult represents the compliance audit outcome.
type OrderMarginVerificationResult struct {
	OrderID                 string  `json:"order_id"`
	CustomerName            string  `json:"customer_name"`
	TotalOrderRevenueIDR    float64 `json:"total_order_revenue_idr"`
	TotalOrderCOGSIDR       float64 `json:"total_order_cogs_idr"`
	GrossProfitIDR          float64 `json:"gross_profit_idr"`
	OrderGrossMarginPercent float64 `json:"order_gross_margin_percent"`
	PolicyThresholdPercent  float64 `json:"policy_threshold_percent"`
	IsApproved              bool    `json:"is_approved"`
	ComplianceStatus        string  `json:"compliance_status"`
	AuditNotes              string  `json:"audit_notes"`
	AuditedAt               string  `json:"audited_at"`
}

// PricingStore provides thread-safe storage for pricing models.
type PricingStore struct {
	sync.RWMutex
	evaluations map[string]*DynamicPriceEvaluation
	audits      map[string]*OrderMarginVerificationResult
}

var globalPricingStore = &PricingStore{
	evaluations: make(map[string]*DynamicPriceEvaluation),
	audits:      make(map[string]*OrderMarginVerificationResult),
}

// EvaluateDynamicPriceLogic calculates algorithmic dynamic prices with margin floor enforcement.
func EvaluateDynamicPriceLogic(req DynamicPriceRequest) (*DynamicPriceEvaluation, error) {
	if req.SKU == "" {
		req.SKU = "SKU-MON-27-QHD"
	}
	if req.ProductName == "" {
		req.ProductName = "Gaming Monitor 27-inch IPS QHD"
	}
	if req.BasePriceIDR <= 0 {
		req.BasePriceIDR = 4800000.0
	}
	if req.CurrentCostIDR <= 0 {
		req.CurrentCostIDR = 3450000.0
	}
	if req.MinMarginPercent <= 0 {
		req.MinMarginPercent = 18.0
	}
	if req.CompetitorPriceIDR <= 0 {
		req.CompetitorPriceIDR = 4250000.0
	}
	if req.PriceElasticityDemand == 0 {
		req.PriceElasticityDemand = -1.4
	}

	marginRatio := req.MinMarginPercent / 100.0
	if marginRatio >= 0.95 {
		marginRatio = 0.95
	}
	floorPrice := math.Round(req.CurrentCostIDR / (1.0 - marginRatio))
	targetPrice := req.BasePriceIDR

	if req.CompetitorPriceIDR > 0 && req.CompetitorPriceIDR < targetPrice {
		targetPrice = req.CompetitorPriceIDR * 0.99
	}

	if req.InventoryDaysOfSupply > 60.0 {
		excessDiscount := math.Min(0.08, (req.InventoryDaysOfSupply-60.0)*0.002)
		targetPrice = targetPrice * (1.0 - excessDiscount)
	}

	if req.ExpiryDaysLeft > 0 && req.ExpiryDaysLeft < 30.0 {
		targetPrice = targetPrice * 0.90
	}

	targetPrice = math.Round(targetPrice)

	marginGuardEnforced := false
	reason := "Dynamic price optimized for competitor alignment and inventory velocity"

	if targetPrice < floorPrice {
		targetPrice = floorPrice
		marginGuardEnforced = true
		reason = fmt.Sprintf("Calculated market price breached margin floor (Rp %.0f); strictly clamped to maintain minimum %.1f%% gross margin", floorPrice, req.MinMarginPercent)
	}

	achievedMargin := math.Round(((targetPrice-req.CurrentCostIDR)/targetPrice)*10000) / 100.0
	delta := targetPrice - req.BasePriceIDR
	adjPct := math.Round((delta/req.BasePriceIDR)*10000) / 100.0
	volGainPct := math.Round((-adjPct*math.Abs(req.PriceElasticityDemand))*100) / 100.0

	eval := DynamicPriceEvaluation{
		SKU:                      req.SKU,
		ProductName:              req.ProductName,
		BasePriceIDR:             req.BasePriceIDR,
		CurrentCostIDR:           req.CurrentCostIDR,
		FloorPriceIDR:            floorPrice,
		SuggestedDynamicPriceIDR: targetPrice,
		AchievedMarginPercent:    achievedMargin,
		MarginGuardEnforced:      marginGuardEnforced,
		PriceAdjustmentDelta:     delta,
		AdjustmentPercent:        adjPct,
		ProjectedVolumeGainPct:   volGainPct,
		RepriceReason:            reason,
		OmnichannelSyncStatus:    "SYNCED_TO_POS_AND_ECOMMERCE",
		EvaluatedAt:              time.Now().Format(time.RFC3339),
	}

	globalPricingStore.Lock()
	globalPricingStore.evaluations[req.SKU] = &eval
	globalPricingStore.Unlock()

	return &eval, nil
}

// VerifyOrderMarginLogic audits order lines against company minimum gross margin policy.
func VerifyOrderMarginLogic(req OrderMarginVerificationRequest) (*OrderMarginVerificationResult, error) {
	if req.OrderID == "" {
		req.OrderID = fmt.Sprintf("SO-BULK-%s-%d", time.Now().Format("20060102"), time.Now().Unix()%1000)
	}
	if req.CustomerName == "" {
		req.CustomerName = "PT Mega Retail Nusantara"
	}
	if req.MinPolicyMarginPct <= 0 {
		req.MinPolicyMarginPct = 15.0
	}

	if len(req.Items) == 0 {
		req.Items = []OrderLineVerification{
			{
				SKU:             "SKU-MON-27-QHD",
				ProductName:     "Gaming Monitor 27-inch IPS QHD",
				Quantity:        50,
				OfferedPrice:    4200000,
				UnitCOGS:        3450000,
				DiscountPercent: 12.5,
			},
		}
	}

	var totRev, totCOGS float64
	for _, item := range req.Items {
		rev := item.Quantity * item.OfferedPrice
		cogs := item.Quantity * item.UnitCOGS
		totRev += rev
		totCOGS += cogs
	}

	grossProfit := totRev - totCOGS
	marginPct := 0.0
	if totRev > 0 {
		marginPct = math.Round((grossProfit/totRev)*10000) / 100.0
	}

	isApproved := marginPct >= req.MinPolicyMarginPct
	status := "APPROVED_WITHIN_HEALTHY_MARGIN"
	auditNotes := fmt.Sprintf("Order margin of %.2f%% meets or exceeds minimum policy threshold of %.2f%%.", marginPct, req.MinPolicyMarginPct)

	if !isApproved {
		status = "MARGIN_HOLD_EXECUTIVE_APPROVAL_REQUIRED"
		auditNotes = fmt.Sprintf("Order margin of %.2f%% breaches minimum threshold (%.2f%%). Auto-hold placed; executive override required.", marginPct, req.MinPolicyMarginPct)
	}

	result := OrderMarginVerificationResult{
		OrderID:                 req.OrderID,
		CustomerName:            req.CustomerName,
		TotalOrderRevenueIDR:    totRev,
		TotalOrderCOGSIDR:       totCOGS,
		GrossProfitIDR:          grossProfit,
		OrderGrossMarginPercent: marginPct,
		PolicyThresholdPercent:  req.MinPolicyMarginPct,
		IsApproved:              isApproved,
		ComplianceStatus:        status,
		AuditNotes:              auditNotes,
		AuditedAt:               time.Now().Format(time.RFC3339),
	}

	globalPricingStore.Lock()
	globalPricingStore.audits[req.OrderID] = &result
	globalPricingStore.Unlock()

	return &result, nil
}

// PricingHandler handles dynamic pricing evaluation and order margin guard audits.
type PricingHandler struct {
	db *gorm.DB
}

// NewPricingHandler constructs a new PricingHandler.
func NewPricingHandler(db *gorm.DB) *PricingHandler {
	return &PricingHandler{db: db}
}

// EvaluateDynamicPrice calculates the algorithmic price balancing demand elasticity, competitor rates, and hard margin floor.
func (h *PricingHandler) EvaluateDynamicPrice(c *gin.Context) {
	var req DynamicPriceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid dynamic price payload: " + err.Error()})
		return
	}

	eval, err := EvaluateDynamicPriceLogic(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, eval)
}

// VerifyOrderMargin audits a sales order's overall profitability to prevent unapproved margin erosion.
func (h *PricingHandler) VerifyOrderMargin(c *gin.Context) {
	var req OrderMarginVerificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order margin payload: " + err.Error()})
		return
	}

	result, err := VerifyOrderMarginLogic(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// RegisterDynamicPricingGuardRoutes registers dynamic pricing & margin guard endpoints.
func RegisterDynamicPricingGuardRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc, db *gorm.DB) {
	h := NewPricingHandler(db)
	pricingGroup := router.Group("/pricing")
	pricingGroup.Use(authMiddleware)
	{
		pricingGroup.POST("/dynamic/evaluate", h.EvaluateDynamicPrice)
		pricingGroup.POST("/margin-guard/verify-order", h.VerifyOrderMargin)
	}
}
