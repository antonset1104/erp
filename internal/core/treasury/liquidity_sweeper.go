package treasury

import (
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// LiquiditySweepAccount models an overseas operating account subject to global treasury concentration.
type LiquiditySweepAccount struct {
	AccountID                string  `json:"account_id"`
	EntityName               string  `json:"entity_name"`
	CountryCode              string  `json:"country_code"` // "ID", "SG", "US", "DE", "JP"
	Currency                 string  `json:"currency"`     // "USD", "EUR", "SGD", "IDR", "JPY"
	OpeningBalance           float64 `json:"opening_balance"`
	TargetFloorBalance       float64 `json:"target_floor_balance"`
	ExchangeRateToUSD        float64 `json:"exchange_rate_to_usd"`       // e.g. 1.0 for USD, 1.08 for EUR, 0.000062 for IDR
	CapitalControlRestricted bool    `json:"capital_control_restricted"` // requires central bank offshore quota clearance
	DailyTransferQuotaUSD    float64 `json:"daily_transfer_quota_usd"`
}

// CrossBorderSweepRequest defines parameters for a group-wide concentration sweep run.
type CrossBorderSweepRequest struct {
	SweepPlanID                string                  `json:"sweep_plan_id"`
	ConcentrationMasterAccount string                  `json:"concentration_master_account"`
	MasterCurrency             string                  `json:"master_currency"` // "USD"
	Accounts                   []LiquiditySweepAccount `json:"accounts"`
	ExecuteViaRail             string                  `json:"execute_via_rail"` // "SWIFT_GPI", "WHOLESALE_CBDC_MBRIDGE", "LOCAL_RTGS"
}

// SweepMovement records each capital transfer movement executed or held by regulations.
type SweepMovement struct {
	MovementID              string  `json:"movement_id"`
	FromAccount             string  `json:"from_account"`
	ToAccount               string  `json:"to_account"`
	LocalCurrencyAmount     float64 `json:"local_currency_amount"`
	Currency                string  `json:"currency"`
	USDConvertedAmount      float64 `json:"usd_converted_amount"`
	Direction               string  `json:"direction"` // "SWEEP_UPSTREAM", "FUND_DOWNSTREAM", "HELD_CAPITAL_CONTROL", "BALANCED"
	SettlementRail          string  `json:"settlement_rail"`
	ArmLengthInterestFeeUSD float64 `json:"arm_length_interest_fee_usd"` // SOFR + spread annual/360 daily
	Status                  string  `json:"status"`                      // "SETTLED", "HELD_QUOTA_EXCEEDED"
}

// CrossBorderSweepResult is the consolidated output of the multi-currency liquidity run.
type CrossBorderSweepResult struct {
	SweepPlanID                   string          `json:"sweep_plan_id"`
	ExecutedAt                    string          `json:"executed_at"`
	MasterAccountID               string          `json:"master_account_id"`
	SettlementRailUsed            string          `json:"settlement_rail_used"`
	TotalSweptUpstreamUSD         float64         `json:"total_swept_upstream_usd"`
	TotalFundedDownstreamUSD      float64         `json:"total_funded_downstream_usd"`
	NetLiquidityGainUSD           float64         `json:"net_liquidity_gain_usd"`
	Movements                     []SweepMovement `json:"movements"`
	CapitalControlNotes           []string        `json:"capital_control_notes"`
	IntercompanyInterestEarnedUSD float64         `json:"intercompany_interest_earned_usd"`
}

// NotionalPoolSummary encapsulates multi-currency cash pooling without physical FX transfer.
type NotionalPoolSummary struct {
	PoolID                          string  `json:"pool_id"`
	MasterCurrency                  string  `json:"master_currency"`
	TotalConsolidatedBalanceUSD     float64 `json:"total_consolidated_balance_usd"`
	TotalActiveSurplusUSD           float64 `json:"total_active_surplus_usd"`
	TotalDeficitOverdraftUSD        float64 `json:"total_deficit_overdraft_usd"`
	NetEffectiveYieldPct            float64 `json:"net_effective_yield_pct"`
	InterestTaxOptimizationSavedUSD float64 `json:"interest_tax_optimization_saved_usd"`
	ParticipatingEntities           int     `json:"participating_entities"`
	LastValuationTimestamp          string  `json:"last_valuation_timestamp"`
}

// SettleCorridorRequest holds payload to execute direct wholesale CBDC settlement between corridors.
type SettleCorridorRequest struct {
	DebtorEntity   string  `json:"debtor_entity"`
	CreditorEntity string  `json:"creditor_entity"`
	AmountUSD      float64 `json:"amount_usd"`
	CorridorName   string  `json:"corridor_name"` // e.g. "mBridge-ASEAN-GCC"
	InvoiceRef     string  `json:"invoice_ref"`
}

// SettleCorridorResult represents instant atomic finality output.
type SettleCorridorResult struct {
	SettlementRef       string  `json:"settlement_ref"`
	CorridorName        string  `json:"corridor_name"`
	DebtorEntity        string  `json:"debtor_entity"`
	CreditorEntity      string  `json:"creditor_entity"`
	SettledAmountUSD    float64 `json:"settled_amount_usd"`
	AtomicFinalityProof string  `json:"atomic_finality_proof"`
	SettledAt           string  `json:"settled_at"`
	Status              string  `json:"status"`
}

// GlobalLiquidityStore provides thread-safe state for treasury liquidity sweeping.
type GlobalLiquidityStore struct {
	mu          sync.RWMutex
	lastResult  *CrossBorderSweepResult
	poolSummary *NotionalPoolSummary
	settledTxns []SettleCorridorResult
}

var globalLiquidityStore = &GlobalLiquidityStore{
	settledTxns: make([]SettleCorridorResult, 0),
}

func init() {
	// Initialize default Notional Pool Summary
	globalLiquidityStore.poolSummary = &NotionalPoolSummary{
		PoolID:                          "POOL-GLOBAL-TREASURY-01",
		MasterCurrency:                  "USD",
		TotalConsolidatedBalanceUSD:     48500000.0,
		TotalActiveSurplusUSD:           52000000.0,
		TotalDeficitOverdraftUSD:        3500000.0,
		NetEffectiveYieldPct:            4.85,
		InterestTaxOptimizationSavedUSD: 412500.0,
		ParticipatingEntities:           8,
		LastValuationTimestamp:          time.Now().Format(time.RFC3339),
	}
}

// ExecuteCrossBorderSweepLogic computes currency conversions, checks capital control restrictions, and sweeps funds.
func ExecuteCrossBorderSweepLogic(req CrossBorderSweepRequest) (*CrossBorderSweepResult, error) {
	globalLiquidityStore.mu.Lock()
	defer globalLiquidityStore.mu.Unlock()

	if req.SweepPlanID == "" {
		req.SweepPlanID = fmt.Sprintf("SWEEP-RUN-%d", time.Now().Unix())
	}
	if req.MasterCurrency == "" {
		req.MasterCurrency = "USD"
	}
	if req.ExecuteViaRail == "" {
		req.ExecuteViaRail = "WHOLESALE_CBDC_MBRIDGE"
	}

	var movements []SweepMovement
	var capitalControlNotes []string
	var totalSweptUpstreamUSD float64
	var totalFundedDownstreamUSD float64
	var totalIntercompanyInterestUSD float64

	const annualArmLengthRate = 0.0525 // SOFR 4.5% + 75 bps margin

	for _, acc := range req.Accounts {
		diff := acc.OpeningBalance - acc.TargetFloorBalance
		if math.Abs(diff) < 0.01 {
			continue
		}

		rateToUSD := acc.ExchangeRateToUSD
		if rateToUSD <= 0 {
			rateToUSD = 1.0
		}
		usdEquivalent := math.Abs(diff) * rateToUSD

		// Daily arm's length interest calculation (annualRate / 360)
		dailyInterest := (usdEquivalent * annualArmLengthRate) / 360.0

		if diff > 0 {
			// Surplus: Sweep upstream to Master Header
			if acc.CapitalControlRestricted && acc.DailyTransferQuotaUSD > 0 && usdEquivalent > acc.DailyTransferQuotaUSD {
				allowedLocal := acc.DailyTransferQuotaUSD / rateToUSD
				heldLocal := diff - allowedLocal
				heldUSD := heldLocal * rateToUSD

				capitalControlNotes = append(capitalControlNotes,
					fmt.Sprintf("Account %s (%s): Transfer capped by central bank quota at $%.2f. Held $%.2f in local escrow.",
						acc.AccountID, acc.CountryCode, acc.DailyTransferQuotaUSD, heldUSD))

				// Execute allowed portion
				movements = append(movements, SweepMovement{
					MovementID:              fmt.Sprintf("MOV-%d-01", time.Now().UnixNano()%1000000),
					FromAccount:             acc.AccountID,
					ToAccount:               req.ConcentrationMasterAccount,
					LocalCurrencyAmount:     allowedLocal,
					Currency:                acc.Currency,
					USDConvertedAmount:      acc.DailyTransferQuotaUSD,
					Direction:               "SWEEP_UPSTREAM",
					SettlementRail:          req.ExecuteViaRail,
					ArmLengthInterestFeeUSD: (acc.DailyTransferQuotaUSD * annualArmLengthRate) / 360.0,
					Status:                  "SETTLED",
				})
				totalSweptUpstreamUSD += acc.DailyTransferQuotaUSD

				// Record held movement
				movements = append(movements, SweepMovement{
					MovementID:              fmt.Sprintf("MOV-%d-02", time.Now().UnixNano()%1000000),
					FromAccount:             acc.AccountID,
					ToAccount:               "CENTRAL_BANK_ESCROW",
					LocalCurrencyAmount:     heldLocal,
					Currency:                acc.Currency,
					USDConvertedAmount:      heldUSD,
					Direction:               "HELD_CAPITAL_CONTROL",
					SettlementRail:          "LOCAL_RTGS",
					ArmLengthInterestFeeUSD: 0,
					Status:                  "HELD_QUOTA_EXCEEDED",
				})
			} else {
				movements = append(movements, SweepMovement{
					MovementID:              fmt.Sprintf("MOV-%d", time.Now().UnixNano()%1000000),
					FromAccount:             acc.AccountID,
					ToAccount:               req.ConcentrationMasterAccount,
					LocalCurrencyAmount:     diff,
					Currency:                acc.Currency,
					USDConvertedAmount:      usdEquivalent,
					Direction:               "SWEEP_UPSTREAM",
					SettlementRail:          req.ExecuteViaRail,
					ArmLengthInterestFeeUSD: dailyInterest,
					Status:                  "SETTLED",
				})
				totalSweptUpstreamUSD += usdEquivalent
				totalIntercompanyInterestUSD += dailyInterest
			}
		} else {
			// Deficit: Fund downstream from Master Header
			deficitLocal := math.Abs(diff)
			movements = append(movements, SweepMovement{
				MovementID:              fmt.Sprintf("MOV-%d", time.Now().UnixNano()%1000000),
				FromAccount:             req.ConcentrationMasterAccount,
				ToAccount:               acc.AccountID,
				LocalCurrencyAmount:     deficitLocal,
				Currency:                acc.Currency,
				USDConvertedAmount:      usdEquivalent,
				Direction:               "FUND_DOWNSTREAM",
				SettlementRail:          req.ExecuteViaRail,
				ArmLengthInterestFeeUSD: dailyInterest,
				Status:                  "SETTLED",
			})
			totalFundedDownstreamUSD += usdEquivalent
			totalIntercompanyInterestUSD += dailyInterest
		}
	}

	netLiquidity := totalSweptUpstreamUSD - totalFundedDownstreamUSD

	result := &CrossBorderSweepResult{
		SweepPlanID:                   req.SweepPlanID,
		ExecutedAt:                    time.Now().Format(time.RFC3339),
		MasterAccountID:               req.ConcentrationMasterAccount,
		SettlementRailUsed:            req.ExecuteViaRail,
		TotalSweptUpstreamUSD:         totalSweptUpstreamUSD,
		TotalFundedDownstreamUSD:      totalFundedDownstreamUSD,
		NetLiquidityGainUSD:           netLiquidity,
		Movements:                     movements,
		CapitalControlNotes:           capitalControlNotes,
		IntercompanyInterestEarnedUSD: totalIntercompanyInterestUSD,
	}

	globalLiquidityStore.lastResult = result
	return result, nil
}

// SettleCorridorLogic executes instantaneous cross-border wholesale CBDC clearing.
func SettleCorridorLogic(req SettleCorridorRequest) (*SettleCorridorResult, error) {
	globalLiquidityStore.mu.Lock()
	defer globalLiquidityStore.mu.Unlock()

	if req.DebtorEntity == "" || req.CreditorEntity == "" || req.AmountUSD <= 0 {
		return nil, fmt.Errorf("debtor, creditor, and positive amount are required")
	}
	if req.CorridorName == "" {
		req.CorridorName = "mBridge-ASEAN-GCC"
	}

	settlementRef := fmt.Sprintf("CBDC-SETTLE-%d", time.Now().UnixNano()%10000000)
	proof := fmt.Sprintf("HASH256::%s::%s::%.2f::%s", req.DebtorEntity, req.CreditorEntity, req.AmountUSD, req.CorridorName)

	res := SettleCorridorResult{
		SettlementRef:       settlementRef,
		CorridorName:        req.CorridorName,
		DebtorEntity:        req.DebtorEntity,
		CreditorEntity:      req.CreditorEntity,
		SettledAmountUSD:    req.AmountUSD,
		AtomicFinalityProof: proof,
		SettledAt:           time.Now().Format(time.RFC3339),
		Status:              "SETTLED",
	}

	globalLiquidityStore.settledTxns = append(globalLiquidityStore.settledTxns, res)
	return &res, nil
}

// RegisterLiquiditySweeperRoutes registers the global liquidity sweep endpoints.
func RegisterLiquiditySweeperRoutes(r *gin.RouterGroup) {
	group := r.Group("/treasury/liquidity")
	{
		group.POST("/cross-border-sweep", func(c *gin.Context) {
			var req CrossBorderSweepRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid sweep request payload: " + err.Error()})
				return
			}
			res, err := ExecuteCrossBorderSweepLogic(req)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, res)
		})

		group.GET("/notional-pool-summary", func(c *gin.Context) {
			globalLiquidityStore.mu.RLock()
			defer globalLiquidityStore.mu.RUnlock()
			c.JSON(http.StatusOK, globalLiquidityStore.poolSummary)
		})

		group.POST("/settle-corridor", func(c *gin.Context) {
			var req SettleCorridorRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid settle corridor payload: " + err.Error()})
				return
			}
			res, err := SettleCorridorLogic(req)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, res)
		})
	}
}
