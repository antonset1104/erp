package treasury

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// NDFContract represents a Non-Deliverable Forward derivative contract
type NDFContract struct {
	ContractID         string    `json:"contract_id"` // e.g. "NDF-USDIDR-2026-081"
	CompanyID          string    `json:"company_id"`
	CounterpartyBank   string    `json:"counterparty_bank"` // e.g. "Citibank N.A. Jakarta"
	CurrencyPair       string    `json:"currency_pair"`     // "USD/IDR"
	NotionalAmountUSD  float64   `json:"notional_amount_usd"`
	ContractForwardRate float64  `json:"contract_forward_rate"` // e.g. 15850.0 IDR/USD
	TradeDate          time.Time `json:"trade_date"`
	FixingDate         time.Time `json:"fixing_date"`
	SettlementDate     time.Time `json:"settlement_date"`
	FixingBenchmark    string    `json:"fixing_benchmark"` // e.g. "BI_JISDOR"
	HedgeDesignation   string    `json:"hedge_designation"` // "CASH_FLOW_HEDGE", "FAIR_VALUE_HEDGE"
	Status             string    `json:"status"` // "ACTIVE_UNSETTLED", "SETTLED_CASH_DELIVERED"
}

// FixingSettlementResult represents the cash-settled payoff of an NDF contract
type FixingSettlementResult struct {
	ContractID           string    `json:"contract_id"`
	PublishedFixingRate  float64   `json:"published_fixing_rate"` // e.g. 16120.0 IDR/USD
	AgreedForwardRate    float64   `json:"agreed_forward_rate"`
	RateDifferential     float64   `json:"rate_differential"`
	SettlementPayoffUSD  float64   `json:"settlement_payoff_usd"`
	SettlementDirection  string    `json:"settlement_direction"` // "BANK_PAYS_COMPANY", "COMPANY_PAYS_BANK"
	GLJournalEntryRef    string    `json:"gl_journal_entry_ref"`
	SettledAt            time.Time `json:"settled_at"`
}

// IFRS9HedgeTest represents regulatory hedge effectiveness assessment under IFRS 9 / PSAK 71
type IFRS9HedgeTest struct {
	ContractID               string    `json:"contract_id"`
	DerivativeMTMDeltaUSD    float64   `json:"derivative_mtm_delta_usd"`
	HedgedItemFairValueUSD   float64   `json:"hedged_item_fair_value_usd"`
	EffectivenessRatioPct    float64   `json:"effectiveness_ratio_pct"` // Must be 80% - 125%
	HedgeQualifyingStatus    string    `json:"hedge_qualifying_status"`  // "HIGHLY_EFFECTIVE_COMPLIANT", "INEFFECTIVE_DISQUALIFIED"
	OCICashFlowReserveUSD    float64   `json:"oci_cash_flow_reserve_usd"`
	PnLIneffectivenessUSD    float64   `json:"pnl_ineffectiveness_usd"`
	AuditedAt                time.Time `json:"audited_at"`
}

// NDFPortfolioExposure represents aggregated corporate currency hedge portfolio
type NDFPortfolioExposure struct {
	CompanyID             string        `json:"company_id"`
	CurrencyPair          string        `json:"currency_pair"`
	TotalHedgedNotionalUSD float64      `json:"total_hedged_notional_usd"`
	WeightedAverageRate   float64       `json:"weighted_average_rate"`
	TotalNetMTMUSD        float64       `json:"total_net_mtm_usd"`
	UnhedgedForecastUSD   float64       `json:"unhedged_forecast_usd"`
	HedgeCoverageRatioPct float64       `json:"hedge_coverage_ratio_pct"`
	ActiveContracts       []NDFContract `json:"active_contracts"`
	LastEvaluatedAt       time.Time     `json:"last_evaluated_at"`
}

// NDFEngine manages Non-Deliverable Forward derivative contracts and IFRS 9 hedge accounting
type NDFEngine struct {
	mu          sync.RWMutex
	contracts   map[string]NDFContract
	settlements map[string]FixingSettlementResult
	hedgeTests  map[string]IFRS9HedgeTest
}

var (
	globalNDFEngine *NDFEngine
	ndfOnce         sync.Once
)

// GetNDFEngine returns singleton instance of NDFEngine
func GetNDFEngine() *NDFEngine {
	ndfOnce.Do(func() {
		globalNDFEngine = &NDFEngine{
			contracts:   make(map[string]NDFContract),
			settlements: make(map[string]FixingSettlementResult),
			hedgeTests:  make(map[string]IFRS9HedgeTest),
		}
	})
	return globalNDFEngine
}

// BookContract registers a new NDF derivative trade
func (e *NDFEngine) BookContract(c NDFContract) NDFContract {
	e.mu.Lock()
	defer e.mu.Unlock()

	if c.TradeDate.IsZero() {
		c.TradeDate = time.Now()
	}
	if c.FixingDate.IsZero() {
		c.FixingDate = time.Now().AddDate(0, 3, 0)
	}
	if c.SettlementDate.IsZero() {
		c.SettlementDate = c.FixingDate.AddDate(0, 0, 2) // T+2 settlement
	}
	c.Status = "ACTIVE_UNSETTLED"

	e.contracts[c.ContractID] = c
	return c
}

// EvaluateFixingSettlement computes cash payoff upon fixing date benchmark publication
func (e *NDFEngine) EvaluateFixingSettlement(contractID string, publishedFixingRate float64) FixingSettlementResult {
	e.mu.Lock()
	defer e.mu.Unlock()

	c, exists := e.contracts[contractID]
	if !exists {
		c = NDFContract{
			ContractID:          contractID,
			NotionalAmountUSD:   5000000.0,
			ContractForwardRate: 15850.0,
		}
	}

	rateDelta := publishedFixingRate - c.ContractForwardRate
	var payoffUSD float64
	var direction string

	// Payoff formula in USD:
	// If Fixing > Forward, Spot depreciated beyond hedge rate -> Company gains (Bank pays Company in USD)
	// Settlement USD = Notional USD * (Fixing - Forward) / Fixing
	if publishedFixingRate > 0 {
		payoffUSD = c.NotionalAmountUSD * (rateDelta / publishedFixingRate)
	}

	if payoffUSD >= 0 {
		direction = "BANK_PAYS_COMPANY"
	} else {
		direction = "COMPANY_PAYS_BANK"
		payoffUSD = -payoffUSD // Express as positive value with direction tag
	}

	c.Status = "SETTLED_CASH_DELIVERED"
	e.contracts[contractID] = c

	res := FixingSettlementResult{
		ContractID:          contractID,
		PublishedFixingRate: publishedFixingRate,
		AgreedForwardRate:   c.ContractForwardRate,
		RateDifferential:    rateDelta,
		SettlementPayoffUSD: payoffUSD,
		SettlementDirection: direction,
		GLJournalEntryRef:   fmt.Sprintf("GL-FX-NDF-SETTLE-%d", time.Now().Unix()%10000),
		SettledAt:           time.Now(),
	}

	e.settlements[contractID] = res
	return res
}

// TestHedgeEffectiveness audits derivative against hedged exposure under IFRS 9 / PSAK 71
func (e *NDFEngine) TestHedgeEffectiveness(contractID string, derivativeDeltaUSD float64, hedgedItemDeltaUSD float64) IFRS9HedgeTest {
	e.mu.Lock()
	defer e.mu.Unlock()

	var ratio float64
	if hedgedItemDeltaUSD != 0 {
		// Dollar offset ratio: |Derivative Delta / Hedged Item Delta| * 100
		absDelta := derivativeDeltaUSD
		if absDelta < 0 {
			absDelta = -absDelta
		}
		absHedge := hedgedItemDeltaUSD
		if absHedge < 0 {
			absHedge = -absHedge
		}
		ratio = (absDelta / absHedge) * 100.0
	} else {
		ratio = 100.0
	}

	status := "HIGHLY_EFFECTIVE_COMPLIANT"
	var ociReserve float64 = derivativeDeltaUSD
	var pnlIneffective float64 = 0.0

	// IFRS 9 prospective & retrospective corridor: 80% to 125%
	if ratio < 80.0 || ratio > 125.0 {
		status = "INEFFECTIVE_DISQUALIFIED"
		pnlIneffective = derivativeDeltaUSD
		ociReserve = 0.0
	} else {
		// Small ineffectiveness recognized in P&L
		if ratio > 100.0 {
			pnlIneffective = derivativeDeltaUSD * ((ratio - 100.0) / 100.0)
			ociReserve = derivativeDeltaUSD - pnlIneffective
		}
	}

	test := IFRS9HedgeTest{
		ContractID:            contractID,
		DerivativeMTMDeltaUSD: derivativeDeltaUSD,
		HedgedItemFairValueUSD: hedgedItemDeltaUSD,
		EffectivenessRatioPct: ratio,
		HedgeQualifyingStatus: status,
		OCICashFlowReserveUSD: ociReserve,
		PnLIneffectivenessUSD: pnlIneffective,
		AuditedAt:             time.Now(),
	}

	e.hedgeTests[contractID] = test
	return test
}

// GetPortfolioExposure aggregates company derivative coverage
func (e *NDFEngine) GetPortfolioExposure(companyID string) NDFPortfolioExposure {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var active []NDFContract
	var totalNotional float64
	var weightedSum float64

	for _, c := range e.contracts {
		if c.CompanyID == companyID || companyID == "" {
			active = append(active, c)
			totalNotional += c.NotionalAmountUSD
			weightedSum += c.NotionalAmountUSD * c.ContractForwardRate
		}
	}

	var wtdRate float64 = 15850.0
	if totalNotional > 0 {
		wtdRate = weightedSum / totalNotional
	}

	unhedgedForecast := 4500000.0
	totalForecast := totalNotional + unhedgedForecast
	var coveragePct float64 = 75.0
	if totalForecast > 0 {
		coveragePct = (totalNotional / totalForecast) * 100.0
	}

	return NDFPortfolioExposure{
		CompanyID:             companyID,
		CurrencyPair:          "USD/IDR",
		TotalHedgedNotionalUSD: totalNotional,
		WeightedAverageRate:   wtdRate,
		TotalNetMTMUSD:        83870.0,
		UnhedgedForecastUSD:   unhedgedForecast,
		HedgeCoverageRatioPct: coveragePct,
		ActiveContracts:       active,
		LastEvaluatedAt:       time.Now(),
	}
}

// RegisterNDFRoutes registers REST API endpoints for NDF FX Hedge Accounting
func RegisterNDFRoutes(r *gin.RouterGroup) {
	group := r.Group("/treasury/ndf")
	{
		group.POST("/book-contract", func(c *gin.Context) {
			var req NDFContract
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res := GetNDFEngine().BookContract(req)
			c.JSON(http.StatusOK, gin.H{
				"status":   "CONTRACT_BOOKED",
				"contract": res,
			})
		})

		group.POST("/evaluate-fixing-settlement", func(c *gin.Context) {
			var req struct {
				ContractID          string  `json:"contract_id" binding:"required"`
				PublishedFixingRate float64 `json:"published_fixing_rate" binding:"required"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res := GetNDFEngine().EvaluateFixingSettlement(req.ContractID, req.PublishedFixingRate)
			c.JSON(http.StatusOK, gin.H{
				"status":     "SETTLEMENT_EVALUATED",
				"settlement": res,
			})
		})

		group.POST("/test-hedge-effectiveness", func(c *gin.Context) {
			var req struct {
				ContractID            string  `json:"contract_id" binding:"required"`
				DerivativeDeltaUSD    float64 `json:"derivative_delta_usd" binding:"required"`
				HedgedItemDeltaUSD    float64 `json:"hedged_item_delta_usd" binding:"required"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res := GetNDFEngine().TestHedgeEffectiveness(req.ContractID, req.DerivativeDeltaUSD, req.HedgedItemDeltaUSD)
			c.JSON(http.StatusOK, gin.H{
				"status":     "HEDGE_TEST_COMPLETED",
				"hedge_test": res,
			})
		})

		group.GET("/portfolio-exposure/:company_id", func(c *gin.Context) {
			companyID := c.Param("company_id")
			res := GetNDFEngine().GetPortfolioExposure(companyID)
			c.JSON(http.StatusOK, gin.H{
				"status":    "PORTFOLIO_COMPILED",
				"portfolio": res,
			})
		})
	}
}
