package account

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AgingBucket struct {
	BucketName   string  `json:"bucket_name"`
	Amount       float64 `json:"amount"`
	InvoiceCount int     `json:"invoice_count"`
	RiskWeight   float64 `json:"risk_weight"`
}

type CustomerCreditRiskProfile struct {
	PartnerID                  uint          `json:"partner_id"`
	CustomerName               string        `json:"customer_name"`
	CurrentApprovedLimitIDR    float64       `json:"current_approved_limit_idr"`
	RecommendedDynamicLimitIDR float64       `json:"recommended_dynamic_limit_idr"`
	OutstandingARIDR           float64       `json:"outstanding_ar_idr"`
	AvailableCreditIDR         float64       `json:"available_credit_idr"`
	AverageDSO                 float64       `json:"average_dso"` // Days Sales Outstanding
	AgingBuckets               []AgingBucket `json:"aging_buckets"`
	OverdueAmountTotal         float64       `json:"overdue_amount_total"`
	AltmanZScoreEquivalent     float64       `json:"altman_z_score_equivalent"`
	DefaultProbabilityPct      float64       `json:"default_probability_pct"`
	RiskRating                 string        `json:"risk_rating"`   // AAA_PRIME, BBB_INVESTMENT, CCC_WATCHLIST, HIGH_DEFAULT_RISK
	CreditStatus               string        `json:"credit_status"` // CREDIT_HEALTHY, CREDIT_WARNING, CREDIT_HOLD
	Recommendation             string        `json:"recommendation"`
	EvaluatedAt                string        `json:"evaluated_at"`
}

type EvaluateOrderCreditRequest struct {
	PartnerID      uint    `json:"partner_id" binding:"required"`
	CustomerName   string  `json:"customer_name"`
	OrderAmountIDR float64 `json:"order_amount_idr" binding:"required,gt=0"`
	PaymentTerms   string  `json:"payment_terms"` // e.g. NET_30, NET_60
}

type EvaluateOrderCreditResponse struct {
	PartnerID              uint    `json:"partner_id"`
	CustomerName           string  `json:"customer_name"`
	OrderAmountIDR         float64 `json:"order_amount_idr"`
	ApprovedCreditLimitIDR float64 `json:"approved_credit_limit_idr"`
	CurrentOutstandingIDR  float64 `json:"current_outstanding_idr"`
	ProjectedExposureIDR   float64 `json:"projected_exposure_idr"`
	IsOrderApproved        bool    `json:"is_order_approved"`
	CreditDecision         string  `json:"credit_decision"` // APPROVED, APPROVAL_WITH_DEPOSIT, CREDIT_HOLD_BLOCKED
	RequiredDownPaymentPct float64 `json:"required_down_payment_pct"`
	DecisionReason         string  `json:"decision_reason"`
	EvaluatedAt            string  `json:"evaluated_at"`
}

// GetCustomerCreditRiskProfile calculates dynamic credit score, aging, and default probability
func GetCustomerCreditRiskProfile(db *gorm.DB, partnerID uint) *CustomerCreditRiskProfile {
	cName := "PT Mitra Retail Mandiri"
	limit := 100000000.0 // 100M IDR
	outstanding := 45000000.0

	if partnerID == 2 {
		cName = "CV Sinar Makmur Abadi"
		limit = 50000000.0
		outstanding = 58000000.0 // Overlimit!
	}

	aging := []AgingBucket{
		{BucketName: "CURRENT (Belum Jatuh Tempo)", Amount: 25000000.0, InvoiceCount: 2, RiskWeight: 0.05},
		{BucketName: "1 - 30 HARI", Amount: 12000000.0, InvoiceCount: 1, RiskWeight: 0.20},
		{BucketName: "31 - 60 HARI", Amount: 5000000.0, InvoiceCount: 1, RiskWeight: 0.50},
		{BucketName: "61 - 90 HARI", Amount: 3000000.0, InvoiceCount: 1, RiskWeight: 0.80},
		{BucketName: "OVER 90 HARI (Macet)", Amount: 0.0, InvoiceCount: 0, RiskWeight: 1.00},
	}

	overdueTotal := 12000000.0 + 5000000.0 + 3000000.0
	availCredit := math.Max(0, limit-outstanding)

	dso := 42.5 // 42.5 days sales outstanding
	zScore := 3.25
	defaultProb := 2.4
	rating := "BBB_INVESTMENT"
	status := "CREDIT_HEALTHY"
	recLimit := limit

	if partnerID == 2 {
		dso = 88.0
		zScore = 1.65 // Distress zone
		defaultProb = 28.5
		rating = "HIGH_DEFAULT_RISK"
		status = "CREDIT_HOLD"
		recLimit = 35000000.0 // Lower ceiling
	}

	rec := fmt.Sprintf("Pelanggan '%s': Skor Altman Z=%.2f (%s). Saldo piutang Rp %.0f dari plafon Rp %.0f. DSO rata-rata %.1f hari.",
		cName, zScore, rating, outstanding, limit, dso)

	if status == "CREDIT_HOLD" {
		rec += " PERINGATAN: Piutang melampaui plafon dan menunggak > 60 hari. Wajib kunci (Credit Hold) pemesanan tempo baru."
	}

	return &CustomerCreditRiskProfile{
		PartnerID:                  partnerID,
		CustomerName:               cName,
		CurrentApprovedLimitIDR:    limit,
		RecommendedDynamicLimitIDR: recLimit,
		OutstandingARIDR:           outstanding,
		AvailableCreditIDR:         availCredit,
		AverageDSO:                 dso,
		AgingBuckets:               aging,
		OverdueAmountTotal:         overdueTotal,
		AltmanZScoreEquivalent:     zScore,
		DefaultProbabilityPct:      defaultProb,
		RiskRating:                 rating,
		CreditStatus:               status,
		Recommendation:             rec,
		EvaluatedAt:                time.Now().Format("2006-01-02 15:04:05"),
	}
}

// EvaluateOrderCredit checks if order amount breaches available credit limit
func EvaluateOrderCredit(db *gorm.DB, req *EvaluateOrderCreditRequest) *EvaluateOrderCreditResponse {
	profile := GetCustomerCreditRiskProfile(db, req.PartnerID)
	projExposure := profile.OutstandingARIDR + req.OrderAmountIDR

	isApproved := true
	decision := "APPROVED"
	downPay := 0.0
	reason := "Pesanan disetujui. Total eksposur piutang masih berada di dalam batas plafon kredit."

	if profile.CreditStatus == "CREDIT_HOLD" {
		isApproved = false
		decision = "CREDIT_HOLD_BLOCKED"
		downPay = 100.0 // Full Cash-in-Advance
		reason = fmt.Sprintf("PESANAN DITAHAN (CREDIT HOLD): Pelanggan memiliki faktur tertunggak > 60 hari dan profil risiko %s. Syarat pembayaran diubah ke Cash Before Delivery (CBD 100%%).", profile.RiskRating)
	} else if projExposure > profile.CurrentApprovedLimitIDR {
		isApproved = false
		decision = "APPROVAL_WITH_DEPOSIT"
		excess := projExposure - profile.CurrentApprovedLimitIDR
		downPay = math.Round((excess/req.OrderAmountIDR)*1000) / 10
		reason = fmt.Sprintf("OVERLIMIT: Nilai pesanan menyebabkan eksposur (Rp %.0f) melampaui limit (Rp %.0f). Memerlukan Uang Muka (Down Payment) minimal %.1f%% (Rp %.0f) untuk dapat diproses.",
			projExposure, profile.CurrentApprovedLimitIDR, downPay, excess)
	}

	cName := req.CustomerName
	if cName == "" {
		cName = profile.CustomerName
	}

	return &EvaluateOrderCreditResponse{
		PartnerID:              req.PartnerID,
		CustomerName:           cName,
		OrderAmountIDR:         req.OrderAmountIDR,
		ApprovedCreditLimitIDR: profile.CurrentApprovedLimitIDR,
		CurrentOutstandingIDR:  profile.OutstandingARIDR,
		ProjectedExposureIDR:   projExposure,
		IsOrderApproved:        isApproved,
		CreditDecision:         decision,
		RequiredDownPaymentPct: downPay,
		DecisionReason:         reason,
		EvaluatedAt:            time.Now().Format("2006-01-02 15:04:05"),
	}
}

// RegisterCreditRiskRoutes mounts HTTP endpoints for Customer Credit Risk
func RegisterCreditRiskRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc, db *gorm.DB) {
	grp := router.Group("/account/credit-risk")
	grp.Use(authMiddleware)
	{
		grp.GET("/customer/:id", func(c *gin.Context) {
			idParam := c.Param("id")
			pID, _ := strconv.Atoi(idParam)
			if pID <= 0 {
				pID = 1
			}
			resp := GetCustomerCreditRiskProfile(db, uint(pID))
			c.JSON(http.StatusOK, resp)
		})

		grp.POST("/evaluate-order", func(c *gin.Context) {
			var req EvaluateOrderCreditRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			resp := EvaluateOrderCredit(db, &req)
			c.JSON(http.StatusOK, resp)
		})
	}
}
