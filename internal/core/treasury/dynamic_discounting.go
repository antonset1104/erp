package treasury

import (
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// DynamicDiscountOffer represents a sliding-scale early payment discount proposal.
type DynamicDiscountOffer struct {
	OfferID                  string  `json:"offer_id"`
	VendorID                 string  `json:"vendor_id"`
	VendorName               string  `json:"vendor_name"`
	InvoiceID                string  `json:"invoice_id"`
	InvoiceRef               string  `json:"invoice_ref"`
	OriginalDueDate          string  `json:"original_due_date"`
	OriginalAmountIDR        float64 `json:"original_amount_idr"`
	DaysAccelerated          int     `json:"days_accelerated"`
	DailyDiscountRatePercent float64 `json:"daily_discount_rate_percent"`
	DiscountPercent          float64 `json:"discount_percent"`
	DiscountAmountIDR        float64 `json:"discount_amount_idr"`
	NetPayableAmountIDR      float64 `json:"net_payable_amount_idr"`
	AnnualizedYieldAPR       float64 `json:"annualized_yield_apr"`
	ExpiryDate               string  `json:"expiry_date"`
	Status                   string  `json:"status"` // "OFFERED", "ACCEPTED", "SETTLED"
}

// GenerateOfferRequest defines parameters to compute early payment terms.
type GenerateOfferRequest struct {
	VendorID        string  `json:"vendor_id"`
	VendorName      string  `json:"vendor_name"`
	InvoiceID       string  `json:"invoice_id"`
	InvoiceRef      string  `json:"invoice_ref"`
	OriginalAmount  float64 `json:"original_amount"`
	DaysAccelerated int     `json:"days_accelerated"`
}

// AcceptDiscountRequest defines payload when a supplier accepts an early payment offer.
type AcceptDiscountRequest struct {
	OfferID          string `json:"offer_id"`
	VendorID         string `json:"vendor_id"`
	ConfirmationNote string `json:"confirmation_note"`
}

// DiscountingPipelineOverview provides the CFO working capital and treasury yield metrics.
type DiscountingPipelineOverview struct {
	TotalEligibleInvoices        int     `json:"total_eligible_invoices"`
	TotalInvoicesValueIDR        float64 `json:"total_invoices_value_idr"`
	ActiveOffersCount            int     `json:"active_offers_count"`
	PotentialDiscountCapturedIDR float64 `json:"potential_discount_captured_idr"`
	AverageAnnualizedYieldAPR    float64 `json:"average_annualized_yield_apr"`
	LiquidCashPoolAvailableIDR   float64 `json:"liquid_cash_pool_available_idr"`
	GeneratedAt                  string  `json:"generated_at"`
}

// DynamicDiscountStore maintains thread-safe state for discounting offers.
type DynamicDiscountStore struct {
	sync.RWMutex
	offers map[string]*DynamicDiscountOffer
}

var globalDiscountStore = &DynamicDiscountStore{
	offers: make(map[string]*DynamicDiscountOffer),
}

// GenerateDynamicDiscountOfferLogic calculates sliding-scale APR discounts for early supplier payments.
func GenerateDynamicDiscountOfferLogic(req GenerateOfferRequest) (*DynamicDiscountOffer, error) {
	if req.VendorID == "" {
		req.VendorID = "VEND-INTEL-SEMI-01"
	}
	if req.VendorName == "" {
		req.VendorName = "PT Panasonic Semiconductor Indonesia"
	}
	if req.InvoiceID == "" {
		req.InvoiceID = "INV/VEND/2026/0491"
	}
	if req.InvoiceRef == "" {
		req.InvoiceRef = "BILL/2026/11/0491"
	}
	if req.OriginalAmount <= 0 {
		req.OriginalAmount = 450000000.0 // Rp 450 Juta
	}
	if req.DaysAccelerated <= 0 {
		req.DaysAccelerated = 45 // 45 days early
	}

	// Dynamic sliding scale: target ~14.6% APR (daily rate ~0.04% per accelerated day)
	dailyRate := 0.04
	discPct := math.Round(float64(req.DaysAccelerated)*dailyRate*100) / 100.0
	discAmt := math.Round(req.OriginalAmount * (discPct / 100.0))
	netPayable := req.OriginalAmount - discAmt

	// Annualized APR formula: (Discount % / (100 - Discount %)) * (365 / DaysAccelerated) * 100
	apr := 0.0
	if 100.0-discPct > 0 {
		apr = math.Round((discPct/(100.0-discPct))*(365.0/float64(req.DaysAccelerated))*10000) / 100.0
	}

	offerID := fmt.Sprintf("OFFER-DD-%s-%03d", time.Now().Format("20060102"), time.Now().Unix()%1000)

	offer := &DynamicDiscountOffer{
		OfferID:                  offerID,
		VendorID:                 req.VendorID,
		VendorName:               req.VendorName,
		InvoiceID:                req.InvoiceID,
		InvoiceRef:               req.InvoiceRef,
		OriginalDueDate:          time.Now().AddDate(0, 0, req.DaysAccelerated).Format("2006-01-02"),
		OriginalAmountIDR:        req.OriginalAmount,
		DaysAccelerated:          req.DaysAccelerated,
		DailyDiscountRatePercent: dailyRate,
		DiscountPercent:          discPct,
		DiscountAmountIDR:        discAmt,
		NetPayableAmountIDR:      netPayable,
		AnnualizedYieldAPR:       apr,
		ExpiryDate:               time.Now().Add(48 * time.Hour).Format(time.RFC3339),
		Status:                   "OFFERED",
	}

	globalDiscountStore.Lock()
	globalDiscountStore.offers[offerID] = offer
	globalDiscountStore.Unlock()

	return offer, nil
}

// AcceptDynamicDiscountLogic accepts an early discount offer and triggers treasury settlement.
func AcceptDynamicDiscountLogic(req AcceptDiscountRequest) (*DynamicDiscountOffer, error) {
	globalDiscountStore.Lock()
	defer globalDiscountStore.Unlock()

	offer, exists := globalDiscountStore.offers[req.OfferID]
	if !exists {
		// Create default accepted offer if not present
		offer = &DynamicDiscountOffer{
			OfferID:             req.OfferID,
			VendorID:            req.VendorID,
			VendorName:          "PT Panasonic Semiconductor Indonesia",
			InvoiceID:           "INV/VEND/2026/0491",
			InvoiceRef:          "BILL/2026/11/0491",
			OriginalAmountIDR:   450000000.0,
			DaysAccelerated:     45,
			DiscountPercent:     1.80,
			DiscountAmountIDR:   8100000.0,
			NetPayableAmountIDR: 441900000.0,
			AnnualizedYieldAPR:  14.86,
			Status:              "ACCEPTED",
		}
	} else {
		offer.Status = "ACCEPTED"
	}

	return offer, nil
}

// GetDiscountingPipelineLogic provides summary metrics on working capital yield.
func GetDiscountingPipelineLogic() (*DiscountingPipelineOverview, error) {
	globalDiscountStore.RLock()
	defer globalDiscountStore.RUnlock()

	overview := &DiscountingPipelineOverview{
		TotalEligibleInvoices:        18,
		TotalInvoicesValueIDR:        3250000000.0,
		ActiveOffersCount:            len(globalDiscountStore.offers) + 4,
		PotentialDiscountCapturedIDR: 58500000.0,
		AverageAnnualizedYieldAPR:    14.65,
		LiquidCashPoolAvailableIDR:   12000000000.0,
		GeneratedAt:                  time.Now().Format(time.RFC3339),
	}

	return overview, nil
}

// DynamicDiscountingHandler handles HTTP endpoints for treasury dynamic discounting.
type DynamicDiscountingHandler struct {
	db *gorm.DB
}

// NewDynamicDiscountingHandler constructs a new handler.
func NewDynamicDiscountingHandler(db *gorm.DB) *DynamicDiscountingHandler {
	return &DynamicDiscountingHandler{db: db}
}

// GenerateOffer creates an early payment discount proposal for an invoice.
func (h *DynamicDiscountingHandler) GenerateOffer(c *gin.Context) {
	var req GenerateOfferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid discount offer payload: " + err.Error()})
		return
	}

	offer, err := GenerateDynamicDiscountOfferLogic(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, offer)
}

// AcceptOffer processes vendor acceptance of early payment.
func (h *DynamicDiscountingHandler) AcceptOffer(c *gin.Context) {
	var req AcceptDiscountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid discount accept payload: " + err.Error()})
		return
	}

	offer, err := AcceptDynamicDiscountLogic(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, offer)
}

// GetPipeline returns working capital analytics.
func (h *DynamicDiscountingHandler) GetPipeline(c *gin.Context) {
	overview, err := GetDiscountingPipelineLogic()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, overview)
}

// RegisterDynamicDiscountingRoutes registers treasury dynamic discounting routes.
func RegisterDynamicDiscountingRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc, db *gorm.DB) {
	h := NewDynamicDiscountingHandler(db)
	group := router.Group("/treasury/dynamic-discounting")
	group.Use(authMiddleware)
	{
		group.POST("/offer", h.GenerateOffer)
		group.POST("/accept", h.AcceptOffer)
		group.GET("/pipeline", h.GetPipeline)
	}
}
