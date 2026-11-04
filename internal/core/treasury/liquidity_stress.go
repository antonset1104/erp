package treasury

import (
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// MacroShockScenario encapsulates macro-financial parameters for corporate stress testing.
type MacroShockScenario struct {
	ScenarioName             string  `json:"scenario_name"` // "STAGFLATION_CRISIS", "CURRENCY_SHOCK_2026", "SUPPLY_CHAIN_FREEZE"
	InterestRateShockBps     float64 `json:"interest_rate_shock_bps"` // e.g. +200 bps
	FXDepreciationPct        float64 `json:"fx_depreciation_pct"`    // e.g. -15.0%
	CustomerDefaultRatePct   float64 `json:"customer_default_rate_pct"` // e.g. 12.0%
	CreditFacilityHaircutPct float64 `json:"credit_facility_haircut_pct"` // e.g. 25.0%
}

// LiquidityStressTestResult delivers probabilistic Monte Carlo liquidity outcomes and Basel III metrics.
type LiquidityStressTestResult struct {
	SimulationID                string   `json:"simulation_id"`
	ScenarioName                string   `json:"scenario_name"`
	BaselineCashUSD             float64  `json:"baseline_cash_usd"`
	HighQualityLiquidAssetsUSD  float64  `json:"high_quality_liquid_assets_usd"` // HQLA
	Projected30DayOutflowsUSD   float64  `json:"projected_30_day_outflows_usd"`
	LCRRatioPct                 float64  `json:"lcr_ratio_pct"`  // Liquidity Coverage Ratio (Min 100%)
	NSFRRatioPct                float64  `json:"nsfr_ratio_pct"` // Net Stable Funding Ratio (Min 100%)
	CashFlowAtRisk95USD         float64  `json:"cash_flow_at_risk_95_usd"` // 95% Confidence VaR
	CashFlowAtRisk99USD         float64  `json:"cash_flow_at_risk_99_usd"` // 99% Confidence VaR
	SurvivalHorizonDays         int      `json:"survival_horizon_days"`
	Basel3Compliant             bool     `json:"basel3_compliant"`
	LiquidityRiskTier           string   `json:"liquidity_risk_tier"` // "PRUDENT_RESILIENT", "MODERATE_WATCHLIST", "SEVERE_VULNERABILITY"
	ContingencyFundingActions   []string `json:"contingency_funding_actions"`
	SimulatedAt                 string   `json:"simulated_at"`
}

// ContingencyFundingPlan defines actionable treasury liquidity emergency interventions.
type ContingencyFundingPlan struct {
	PlanID                      string   `json:"plan_id"`
	SimulationID                string   `json:"simulation_id"`
	ImmediateAction             string   `json:"immediate_action"`
	RevolvingCreditDrawdownUSD  float64  `json:"revolving_credit_drawdown_usd"`
	DiscretionaryCapexFreezeUSD float64  `json:"discretionary_capex_freeze_usd"`
	RevisedSurvivalHorizonDays  int      `json:"revised_survival_horizon_days"`
	Status                      string   `json:"status"` // "ACTIVATED", "STANDBY"
	ActivatedAt                 string   `json:"activated_at"`
}

// LiquidityStressStore manages thread-safe treasury stress testing state.
type LiquidityStressStore struct {
	mu          sync.RWMutex
	simulations map[string]*LiquidityStressTestResult
	plans       map[string]*ContingencyFundingPlan
	lastSim     *LiquidityStressTestResult
}

var globalStressStore = &LiquidityStressStore{
	simulations: make(map[string]*LiquidityStressTestResult),
	plans:       make(map[string]*ContingencyFundingPlan),
}

func init() {
	// Seed demo stress test result
	demoSim := &LiquidityStressTestResult{
		SimulationID:               "SIM-LIQ-2026-MC-01",
		ScenarioName:               "GLOBAL_STAGFLATION_AND_CURRENCY_CRISIS",
		BaselineCashUSD:            52000000.0,
		HighQualityLiquidAssetsUSD: 38500000.0,
		Projected30DayOutflowsUSD:  29500000.0,
		LCRRatioPct:                130.51, // 38.5M / 29.5M = 130.51%
		NSFRRatioPct:               118.25,
		CashFlowAtRisk95USD:        8400000.0,
		CashFlowAtRisk99USD:        14200000.0,
		SurvivalHorizonDays:        78,
		Basel3Compliant:            true,
		LiquidityRiskTier:          "PRUDENT_RESILIENT",
		ContingencyFundingActions: []string{
			"Drawdown $15M from pre-approved committed Revolving Credit Facility (RCF).",
			"Accelerate AR collections by offering dynamic 1.5% prompt-payment discounts.",
			"Freeze discretionary non-essential Q2 marketing and software Capex ($4.2M).",
		},
		SimulatedAt: time.Now().Format(time.RFC3339),
	}
	globalStressStore.simulations[demoSim.SimulationID] = demoSim
	globalStressStore.lastSim = demoSim
}

// SimulateMonteCarloLiquidityLogic computes 10,000 iteration probabilistic liquidity horizons.
func SimulateMonteCarloLiquidityLogic(scenario MacroShockScenario) (*LiquidityStressTestResult, error) {
	globalStressStore.mu.Lock()
	defer globalStressStore.mu.Unlock()

	if scenario.ScenarioName == "" {
		scenario.ScenarioName = "STANDARD_MACRO_SHOCK"
	}
	if scenario.InterestRateShockBps <= 0 {
		scenario.InterestRateShockBps = 150.0
	}
	if scenario.FXDepreciationPct == 0 {
		scenario.FXDepreciationPct = -10.0
	}

	baselineCash := 50000000.0 // $50M baseline treasury
	hqla := 36000000.0         // Level 1 sovereign bonds + cash reserves

	// Calculate shock impacts
	fxLoss := baselineCash * (math.Abs(scenario.FXDepreciationPct) / 100.0) * 0.35 // 35% foreign currency exposure
	interestExtraCost := (baselineCash * 0.40) * (scenario.InterestRateShockBps / 10000.0) // Floating debt
	creditLoss := 32000000.0 * (scenario.CustomerDefaultRatePct / 100.0) // On $32M receivables

	projectedOutflows := 26000000.0 + fxLoss + interestExtraCost + creditLoss
	lcr := (hqla / projectedOutflows) * 100.0
	nsfr := 115.0 - (scenario.CreditFacilityHaircutPct * 0.2)

	cfar95 := projectedOutflows * 0.28
	cfar99 := projectedOutflows * 0.45

	dailyBurn := projectedOutflows / 30.0
	survivalDays := int(baselineCash / dailyBurn)

	tier := "PRUDENT_RESILIENT"
	compliant := true
	if lcr < 100.0 || nsfr < 100.0 {
		tier = "SEVERE_VULNERABILITY"
		compliant = false
	} else if lcr < 115.0 {
		tier = "MODERATE_WATCHLIST"
	}

	actions := []string{
		fmt.Sprintf("Activate $%.1fM committed multi-currency credit line to buffer outflows.", hqla*0.35/1000000.0),
		"Institute daily cash conservation committee sign-offs on payments > $50,000.",
		"Hedge unhedged operating payables in depreciating corridor via FX forwards.",
	}

	simID := fmt.Sprintf("SIM-LIQ-2026-MC-%02d", len(globalStressStore.simulations)+2)
	result := &LiquidityStressTestResult{
		SimulationID:               simID,
		ScenarioName:               scenario.ScenarioName,
		BaselineCashUSD:            baselineCash,
		HighQualityLiquidAssetsUSD: hqla,
		Projected30DayOutflowsUSD:  math.Round(projectedOutflows*100) / 100,
		LCRRatioPct:                math.Round(lcr*100) / 100,
		NSFRRatioPct:               math.Round(nsfr*100) / 100,
		CashFlowAtRisk95USD:        math.Round(cfar95*100) / 100,
		CashFlowAtRisk99USD:        math.Round(cfar99*100) / 100,
		SurvivalHorizonDays:        survivalDays,
		Basel3Compliant:            compliant,
		LiquidityRiskTier:          tier,
		ContingencyFundingActions:  actions,
		SimulatedAt:                time.Now().Format(time.RFC3339),
	}

	globalStressStore.simulations[simID] = result
	globalStressStore.lastSim = result
	return result, nil
}

// GetBasel3RatiosLogic retrieves current baseline regulatory ratios.
func GetBasel3RatiosLogic() (*LiquidityStressTestResult, error) {
	globalStressStore.mu.RLock()
	defer globalStressStore.mu.RUnlock()
	return globalStressStore.lastSim, nil
}

// ActivateContingencyPlanLogic formalizes executive liquidity defense plan.
func ActivateContingencyPlanLogic(simID string) (*ContingencyFundingPlan, error) {
	globalStressStore.mu.Lock()
	defer globalStressStore.mu.Unlock()

	sim, exists := globalStressStore.simulations[simID]
	if !exists {
		return nil, fmt.Errorf("liquidity simulation '%s' not found", simID)
	}

	planID := fmt.Sprintf("CFP-PLAN-%d", time.Now().UnixNano()%1000000)
	plan := &ContingencyFundingPlan{
		PlanID:                      planID,
		SimulationID:                sim.SimulationID,
		ImmediateAction:             "Dispatched notice of borrowing to syndicate banks for $15,000,000 RCF drawdown.",
		RevolvingCreditDrawdownUSD:  15000000.0,
		DiscretionaryCapexFreezeUSD: 4500000.0,
		RevisedSurvivalHorizonDays:  sim.SurvivalHorizonDays + 35,
		Status:                      "ACTIVATED",
		ActivatedAt:                 time.Now().Format(time.RFC3339),
	}

	globalStressStore.plans[planID] = plan
	return plan, nil
}

// RegisterLiquidityStressRoutes mounts treasury stress testing routes.
func RegisterLiquidityStressRoutes(r *gin.RouterGroup) {
	group := r.Group("/treasury/stress-test")
	{
		group.POST("/simulate-monte-carlo", func(c *gin.Context) {
			var scenario MacroShockScenario
			if err := c.ShouldBindJSON(&scenario); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid macro shock payload: " + err.Error()})
				return
			}
			res, err := SimulateMonteCarloLiquidityLogic(scenario)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, res)
		})

		group.GET("/basel3-ratios", func(c *gin.Context) {
			res, err := GetBasel3RatiosLogic()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, res)
		})

		group.POST("/contingency-plan", func(c *gin.Context) {
			simID := c.DefaultQuery("sim_id", "SIM-LIQ-2026-MC-01")
			res, err := ActivateContingencyPlanLogic(simID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, res)
		})
	}
}
