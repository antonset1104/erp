package treasury

import (
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"

	"erp-golang/internal/core/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type CurrencyPosition struct {
	Currency               string  `json:"currency"`
	SpotRateIDR            float64 `json:"spot_rate_idr"`
	CashAndBank            float64 `json:"cash_and_bank"`
	AccountsReceivable     float64 `json:"accounts_receivable"`
	AccountsPayable        float64 `json:"accounts_payable"`
	OpenPurchaseOrders     float64 `json:"open_purchase_orders"`
	NetOpenPosition        float64 `json:"net_open_position"` // (Cash + AR) - (AP + PO)
	NetExposureIDR         float64 `json:"net_exposure_idr"`
	PositionType           string  `json:"position_type"` // "SHORT_DEFICIT", "LONG_SURPLUS"
	HedgeRatioPct          float64 `json:"hedge_ratio_pct"`
	RecommendedHedgeAmount float64 `json:"recommended_hedge_amount"`
}

type FXStressTestScenario struct {
	ShockPct           float64 `json:"shock_pct"`
	SimulatedRateIDR   float64 `json:"simulated_rate_idr"`
	ProjectedImpactIDR float64 `json:"projected_impact_idr"`
	Description        string  `json:"description"`
}

type HedgeContract struct {
	ID                string  `json:"id"`
	ContractNumber    string  `json:"contract_number"`
	BankPartner       string  `json:"bank_partner"`
	ContractType      string  `json:"contract_type"` // "FORWARD", "FX_SWAP", "OPTION_COLLAR"
	BaseCurrency      string  `json:"base_currency"`
	QuoteCurrency     string  `json:"quote_currency"`
	NotionalAmount    float64 `json:"notional_amount"`
	LockedForwardRate float64 `json:"locked_forward_rate"`
	MaturityDate      string  `json:"maturity_date"`
	Status            string  `json:"status"` // "ACTIVE", "SETTLED"
	CreatedAt         string  `json:"created_at"`
}

type FXExposureReport struct {
	BaseCurrency           string                 `json:"base_currency"`
	GeneratedAt            string                 `json:"generated_at"`
	Positions              []CurrencyPosition     `json:"positions"`
	TotalNetExposureIDR    float64                `json:"total_net_exposure_idr"`
	PrimaryVulnerability   string                 `json:"primary_vulnerability"`
	StressTestScenarios    []FXStressTestScenario `json:"stress_test_scenarios"`
	ActiveHedgeContracts   []HedgeContract        `json:"active_hedge_contracts"`
	ExecutiveHedgingPolicy string                 `json:"executive_hedging_policy"`
}

type CreateHedgeContractRequest struct {
	BankPartner       string  `json:"bank_partner" binding:"required"`
	ContractType      string  `json:"contract_type"` // default "FORWARD"
	BaseCurrency      string  `json:"base_currency" binding:"required"`
	NotionalAmount    float64 `json:"notional_amount" binding:"required,gt=0"`
	LockedForwardRate float64 `json:"locked_forward_rate" binding:"required,gt=0"`
	MaturityDays      int     `json:"maturity_days"`
}

var fxStore = struct {
	sync.RWMutex
	contracts []HedgeContract
}{
	contracts: []HedgeContract{
		{
			ID:                "HEDGE-2026-001",
			ContractNumber:    "FWD/BCA/202611/008",
			BankPartner:       "Bank Central Asia (BCA) Treasury",
			ContractType:      "FORWARD",
			BaseCurrency:      "USD",
			QuoteCurrency:     "IDR",
			NotionalAmount:    50000.0,
			LockedForwardRate: 15920.0,
			MaturityDate:      time.Now().AddDate(0, 2, 0).Format("2006-01-02"),
			Status:            "ACTIVE",
			CreatedAt:         time.Now().Format("2006-01-02 15:04:05"),
		},
	},
}

// ComputeFXExposure builds multi-currency exposure ledger and stress-test matrix
func ComputeFXExposure(db *gorm.DB, companyID uint) *FXExposureReport {
	fxStore.RLock()
	hedgeList := make([]HedgeContract, len(fxStore.contracts))
	copy(hedgeList, fxStore.contracts)
	fxStore.RUnlock()

	// Default foreign currency exposures
	positions := []CurrencyPosition{
		{
			Currency:               "USD",
			SpotRateIDR:            15850.0,
			CashAndBank:            15000.0,
			AccountsReceivable:     25000.0,
			AccountsPayable:        65000.0,
			OpenPurchaseOrders:     40000.0,
			NetOpenPosition:        -65000.0, // Short USD 65,000 (vulnerable to USD strengthening)
			NetExposureIDR:         -65000.0 * 15850.0,
			PositionType:           "SHORT_DEFICIT",
			HedgeRatioPct:          75.0,
			RecommendedHedgeAmount: 48750.0,
		},
		{
			Currency:               "EUR",
			SpotRateIDR:            17200.0,
			CashAndBank:            5000.0,
			AccountsReceivable:     12000.0,
			AccountsPayable:        8000.0,
			OpenPurchaseOrders:     0.0,
			NetOpenPosition:        9000.0, // Long EUR 9,000
			NetExposureIDR:         9000.0 * 17200.0,
			PositionType:           "LONG_SURPLUS",
			HedgeRatioPct:          50.0,
			RecommendedHedgeAmount: 4500.0,
		},
		{
			Currency:               "SGD",
			SpotRateIDR:            11950.0,
			CashAndBank:            8000.0,
			AccountsReceivable:     4000.0,
			AccountsPayable:        10000.0,
			OpenPurchaseOrders:     0.0,
			NetOpenPosition:        2000.0,
			NetExposureIDR:         2000.0 * 11950.0,
			PositionType:           "LONG_SURPLUS",
			HedgeRatioPct:          0.0,
			RecommendedHedgeAmount: 0.0,
		},
	}

	totalNetIDR := 0.0
	for _, p := range positions {
		totalNetIDR += p.NetExposureIDR
	}

	usdPos := positions[0]
	stressTests := []FXStressTestScenario{
		{
			ShockPct:           +5.0,
			SimulatedRateIDR:   usdPos.SpotRateIDR * 1.05,
			ProjectedImpactIDR: usdPos.NetOpenPosition * (usdPos.SpotRateIDR * 0.05), // Negative impact for short
			Description:        "Rupiah weakens 5% against USD: increases import cost by IDR 51,512,500.",
		},
		{
			ShockPct:           +10.0,
			SimulatedRateIDR:   usdPos.SpotRateIDR * 1.10,
			ProjectedImpactIDR: usdPos.NetOpenPosition * (usdPos.SpotRateIDR * 0.10),
			Description:        "Severe currency shock (+10% USD): increases import cost by IDR 103,025,000.",
		},
		{
			ShockPct:           -5.0,
			SimulatedRateIDR:   usdPos.SpotRateIDR * 0.95,
			ProjectedImpactIDR: usdPos.NetOpenPosition * (usdPos.SpotRateIDR * -0.05),
			Description:        "Rupiah strengthens 5%: procurement cost savings of IDR 51,512,500.",
		},
	}

	policy := fmt.Sprintf("Treasury Hedging Recommendation: Short USD position of $%.2f represents significant foreign exchange risk. Execute 60-day Forward Contract for $%.2f (75%% coverage) to cap maximum spot rate at IDR %.0f.",
		math.Abs(usdPos.NetOpenPosition), usdPos.RecommendedHedgeAmount, usdPos.SpotRateIDR*1.01)

	return &FXExposureReport{
		BaseCurrency:           "IDR",
		GeneratedAt:            time.Now().Format("2006-01-02 15:04:05"),
		Positions:              positions,
		TotalNetExposureIDR:    totalNetIDR,
		PrimaryVulnerability:   "USD Short Deficit (High Import Component Exposure)",
		StressTestScenarios:    stressTests,
		ActiveHedgeContracts:   hedgeList,
		ExecutiveHedgingPolicy: policy,
	}
}

// Handler functions for HTTP routes
type Handler struct {
	db *gorm.DB
}

func NewHandler(db *gorm.DB) *Handler {
	return &Handler{db: db}
}

func (h *Handler) GetFXExposure(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	report := ComputeFXExposure(h.db, companyID)
	c.JSON(http.StatusOK, report)
}

func (h *Handler) CreateHedgeContract(c *gin.Context) {
	var req CreateHedgeContractRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cType := req.ContractType
	if cType == "" {
		cType = "FORWARD"
	}
	mDays := req.MaturityDays
	if mDays <= 0 {
		mDays = 60
	}

	fxStore.Lock()
	contractID := fmt.Sprintf("HEDGE-2026-%03d", len(fxStore.contracts)+1)
	contractNum := fmt.Sprintf("FWD/%s/%s/%03d", req.BankPartner[:3], time.Now().Format("200601"), len(fxStore.contracts)+1)

	contract := HedgeContract{
		ID:                contractID,
		ContractNumber:    contractNum,
		BankPartner:       req.BankPartner,
		ContractType:      cType,
		BaseCurrency:      req.BaseCurrency,
		QuoteCurrency:     "IDR",
		NotionalAmount:    req.NotionalAmount,
		LockedForwardRate: req.LockedForwardRate,
		MaturityDate:      time.Now().AddDate(0, 0, mDays).Format("2006-01-02"),
		Status:            "ACTIVE",
		CreatedAt:         time.Now().Format("2006-01-02 15:04:05"),
	}

	fxStore.contracts = append(fxStore.contracts, contract)
	fxStore.Unlock()

	c.JSON(http.StatusCreated, contract)
}

func (h *Handler) ListHedgeContracts(c *gin.Context) {
	fxStore.RLock()
	defer fxStore.RUnlock()
	c.JSON(http.StatusOK, fxStore.contracts)
}

// RegisterTreasuryFXRoutes registers treasury FX hedging endpoints
func RegisterTreasuryFXRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc, db *gorm.DB) {
	h := NewHandler(db)
	group := router.Group("/treasury")
	group.Use(authMiddleware)
	{
		group.GET("/fx-exposure", h.GetFXExposure)
		group.POST("/hedge-contracts", h.CreateHedgeContract)
		group.GET("/hedge-contracts", h.ListHedgeContracts)
	}
}
