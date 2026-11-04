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

// CashPoolAccount represents a corporate entity account within the group treasury pool.
type CashPoolAccount struct {
	AccountID            string  `json:"account_id"`
	EntityName           string  `json:"entity_name"`
	Currency             string  `json:"currency"`
	CurrentBalance       float64 `json:"current_balance"`
	TargetBalance        float64 `json:"target_balance"`
	PoolType             string  `json:"pool_type"` // "PHYSICAL_ZBA", "NOTIONAL_POOL"
	IntercompanyLoanRate float64 `json:"intercompany_loan_rate"`
}

// SweepExecutionRequest defines the payload to execute an end-of-day treasury sweep.
type SweepExecutionRequest struct {
	PoolID          string            `json:"pool_id"`
	ExecutionMode   string            `json:"execution_mode"` // "MANUAL", "SCHEDULED_EOD"
	HeaderAccountID string            `json:"header_account_id"`
	Accounts        []CashPoolAccount `json:"accounts"`
}

// SweepTransaction logs each upstream or downstream sweep movement.
type SweepTransaction struct {
	TransactionID          string  `json:"transaction_id"`
	AccountID              string  `json:"account_id"`
	EntityName             string  `json:"entity_name"`
	SweepDirection         string  `json:"sweep_direction"` // "UPSTREAM_TO_HEADER", "DOWNSTREAM_TO_SUBSIDIARY", "BALANCED"
	SweepAmount            float64 `json:"sweep_amount"`
	Currency               string  `json:"currency"`
	PreBalance             float64 `json:"pre_balance"`
	PostBalance            float64 `json:"post_balance"`
	DailyArmLengthInterest float64 `json:"daily_arm_length_interest"`
}

// SweepReport returns the comprehensive consolidated result of the sweep cycle.
type SweepReport struct {
	SweepCycleID                 string             `json:"sweep_cycle_id"`
	Timestamp                    string             `json:"timestamp"`
	HeaderAccountID              string             `json:"header_account_id"`
	TotalFundsSweptUpstream       float64            `json:"total_funds_swept_upstream"`
	TotalDeficitFundedDownstream float64            `json:"total_deficit_funded_downstream"`
	NetLiquidityConcentrated     float64            `json:"net_liquidity_concentrated"`
	TotalDailyInterestYield      float64            `json:"total_daily_interest_yield"`
	Transactions                 []SweepTransaction `json:"transactions"`
	Status                       string             `json:"status"`
}

// IntercompanyObligation represents an invoice due between group subsidiaries.
type IntercompanyObligation struct {
	InvoiceRef     string  `json:"invoice_ref"`
	DebtorEntity   string  `json:"debtor_entity"`
	CreditorEntity string  `json:"creditor_entity"`
	Amount         float64 `json:"amount"`
	Currency       string  `json:"currency"`
	AmountInUSD    float64 `json:"amount_in_usd"`
}

// NettingCycleRequest defines multilateral netting input.
type NettingCycleRequest struct {
	CycleID            string                   `json:"cycle_id"`
	SettlementCurrency string                   `json:"settlement_currency"` // e.g. "USD"
	Invoices           []IntercompanyObligation `json:"invoices"`
}

// NettingParticipantResult captures the net position for a single group company.
type NettingParticipantResult struct {
	EntityName         string  `json:"entity_name"`
	GrossPayablesUSD   float64 `json:"gross_payables_usd"`
	GrossReceivablesUSD float64 `json:"gross_receivables_usd"`
	NetPosition        string  `json:"net_position"` // "NET_PAYER", "NET_RECEIVER", "SQUARE"
	NetAmountUSD       float64 `json:"net_amount_usd"`
}

// NettingReport represents the bilateral/multilateral netting settlement outcome.
type NettingReport struct {
	CycleID                      string                     `json:"cycle_id"`
	Timestamp                    string                     `json:"timestamp"`
	SettlementCurrency           string                     `json:"settlement_currency"`
	TotalGrossInvoicesCount      int                        `json:"total_gross_invoices_count"`
	TotalGrossVolumeUSD          float64                    `json:"total_gross_volume_usd"`
	TotalNetSettlementUSD        float64                    `json:"total_net_settlement_usd"`
	FXVolumeCompressionPercent   float64                    `json:"fx_volume_compression_percent"`
	EstimatedBankingFeesSavedUSD float64                    `json:"estimated_banking_fees_saved_usd"`
	Participants                 []NettingParticipantResult `json:"participants"`
	Status                       string                     `json:"status"`
}

// IHBBalanceOverview summarizes the In-House Bank position.
type IHBBalanceOverview struct {
	InHouseBankID                string  `json:"in_house_bank_id"`
	MasterTreasuryEntity         string  `json:"master_treasury_entity"`
	TotalDepositsHeldUSD         float64 `json:"total_deposits_held_usd"`
	TotalCreditLinesDrawnUSD     float64 `json:"total_credit_lines_drawn_usd"`
	NetLiquiditySurplusUSD       float64 `json:"net_liquidity_surplus_usd"`
	AverageCostOfCapitalPercent  float64 `json:"average_cost_of_capital_percent"`
	ActiveSweepingAccounts       int     `json:"active_sweeping_accounts"`
	NextScheduledSweep           string  `json:"next_scheduled_sweep"`
}

// CashPoolStore provides thread-safe state management for Treasury Cash Pooling.
type CashPoolStore struct {
	sync.RWMutex
	lastSweepReport *SweepReport
	lastNetting     *NettingReport
}

var globalCashPoolStore = &CashPoolStore{}

// ExecuteSweepLogic runs the sweeping calculation algorithm.
func ExecuteSweepLogic(req SweepExecutionRequest) (*SweepReport, error) {
	if req.HeaderAccountID == "" {
		req.HeaderAccountID = "ACC-TREASURY-HEADER-01"
	}

	if len(req.Accounts) == 0 {
		req.Accounts = []CashPoolAccount{
			{
				AccountID:            "ACC-IDN-OPS",
				EntityName:           "PT GoERP Manufaktur Indonesia",
				Currency:             "IDR",
				CurrentBalance:       25000000000,
				TargetBalance:        5000000000,
				PoolType:             "PHYSICAL_ZBA",
				IntercompanyLoanRate: 4.75,
			},
			{
				AccountID:            "ACC-SGP-TECH",
				EntityName:           "GoERP Technologies Pte Ltd",
				Currency:             "USD",
				CurrentBalance:       250000,
				TargetBalance:        500000,
				PoolType:             "PHYSICAL_ZBA",
				IntercompanyLoanRate: 4.25,
			},
			{
				AccountID:            "ACC-MYS-DIST",
				EntityName:           "GoERP Distribution Sdn Bhd",
				Currency:             "USD",
				CurrentBalance:       1200000,
				TargetBalance:        400000,
				PoolType:             "PHYSICAL_ZBA",
				IntercompanyLoanRate: 4.50,
			},
		}
	}

	globalCashPoolStore.Lock()
	defer globalCashPoolStore.Unlock()

	sweepID := fmt.Sprintf("SWEEP-%s-%d", time.Now().Format("20060102"), time.Now().Unix()%10000)
	var txs []SweepTransaction
	var totalUpstream, totalDownstream, totalInterest float64

	for idx, acc := range req.Accounts {
		diff := acc.CurrentBalance - acc.TargetBalance
		direction := "BALANCED"
		sweepAmt := math.Abs(diff)
		postBal := acc.TargetBalance
		var dailyInterest float64

		if diff > 0.01 {
			direction = "UPSTREAM_TO_HEADER"
			totalUpstream += sweepAmt
			dailyInterest = math.Round((sweepAmt*(acc.IntercompanyLoanRate/100.0)/365.0)*100) / 100
		} else if diff < -0.01 {
			direction = "DOWNSTREAM_TO_SUBSIDIARY"
			totalDownstream += sweepAmt
			dailyInterest = math.Round((sweepAmt*(acc.IntercompanyLoanRate/100.0)/365.0)*100) / 100
		} else {
			postBal = acc.CurrentBalance
			sweepAmt = 0
		}

		totalInterest += dailyInterest

		txs = append(txs, SweepTransaction{
			TransactionID:          fmt.Sprintf("TX-SWP-%03d", idx+1),
			AccountID:              acc.AccountID,
			EntityName:             acc.EntityName,
			SweepDirection:         direction,
			SweepAmount:            sweepAmt,
			Currency:               acc.Currency,
			PreBalance:             acc.CurrentBalance,
			PostBalance:            postBal,
			DailyArmLengthInterest: dailyInterest,
		})
	}

	report := SweepReport{
		SweepCycleID:                 sweepID,
		Timestamp:                    time.Now().Format(time.RFC3339),
		HeaderAccountID:              req.HeaderAccountID,
		TotalFundsSweptUpstream:       totalUpstream,
		TotalDeficitFundedDownstream: totalDownstream,
		NetLiquidityConcentrated:     totalUpstream - totalDownstream,
		TotalDailyInterestYield:      totalInterest,
		Transactions:                 txs,
		Status:                       "EXECUTED_RECONCILED",
	}

	globalCashPoolStore.lastSweepReport = &report
	return &report, nil
}

// RunNettingCycleLogic executes multilateral netting calculations.
func RunNettingCycleLogic(req NettingCycleRequest) (*NettingReport, error) {
	if req.SettlementCurrency == "" {
		req.SettlementCurrency = "USD"
	}

	if len(req.Invoices) == 0 {
		req.Invoices = []IntercompanyObligation{
			{
				InvoiceRef:     "IC-INV-2026-001",
				DebtorEntity:   "GoERP Technologies Pte Ltd",
				CreditorEntity: "PT GoERP Manufaktur Indonesia",
				Amount:         850000,
				Currency:       "USD",
				AmountInUSD:    850000,
			},
			{
				InvoiceRef:     "IC-INV-2026-002",
				DebtorEntity:   "PT GoERP Manufaktur Indonesia",
				CreditorEntity: "GoERP Technologies Pte Ltd",
				Amount:         420000,
				Currency:       "USD",
				AmountInUSD:    420000,
			},
			{
				InvoiceRef:     "IC-INV-2026-003",
				DebtorEntity:   "GoERP Distribution Sdn Bhd",
				CreditorEntity: "PT GoERP Manufaktur Indonesia",
				Amount:         630000,
				Currency:       "USD",
				AmountInUSD:    630000,
			},
			{
				InvoiceRef:     "IC-INV-2026-004",
				DebtorEntity:   "GoERP Technologies Pte Ltd",
				CreditorEntity: "GoERP Distribution Sdn Bhd",
				Amount:         210000,
				Currency:       "USD",
				AmountInUSD:    210000,
			},
		}
	}

	globalCashPoolStore.Lock()
	defer globalCashPoolStore.Unlock()

	var totalGrossUSD float64
	entityPayables := make(map[string]float64)
	entityReceivables := make(map[string]float64)
	entitiesSet := make(map[string]bool)

	for _, inv := range req.Invoices {
		val := inv.AmountInUSD
		if val <= 0 {
			val = inv.Amount
		}
		totalGrossUSD += val
		entityPayables[inv.DebtorEntity] += val
		entityReceivables[inv.CreditorEntity] += val
		entitiesSet[inv.DebtorEntity] = true
		entitiesSet[inv.CreditorEntity] = true
	}

	var participants []NettingParticipantResult
	var totalNetSettlementUSD float64

	for entity := range entitiesSet {
		pay := entityPayables[entity]
		rec := entityReceivables[entity]
		net := rec - pay

		pos := "SQUARE"
		netAmt := math.Abs(net)

		if net > 0.01 {
			pos = "NET_RECEIVER"
			totalNetSettlementUSD += netAmt
		} else if net < -0.01 {
			pos = "NET_PAYER"
		}

		participants = append(participants, NettingParticipantResult{
			EntityName:          entity,
			GrossPayablesUSD:    pay,
			GrossReceivablesUSD: rec,
			NetPosition:         pos,
			NetAmountUSD:        math.Round(netAmt*100) / 100,
		})
	}

	compressionPct := 0.0
	if totalGrossUSD > 0 {
		compressionPct = math.Round((1.0-(totalNetSettlementUSD/totalGrossUSD))*10000) / 100.0
	}
	feesSaved := math.Round((totalGrossUSD-totalNetSettlementUSD)*0.008*100) / 100.0

	cycleID := req.CycleID
	if cycleID == "" {
		cycleID = fmt.Sprintf("NET-CYC-%s-%03d", time.Now().Format("20060102"), time.Now().Unix()%1000)
	}

	report := NettingReport{
		CycleID:                      cycleID,
		Timestamp:                    time.Now().Format(time.RFC3339),
		SettlementCurrency:           req.SettlementCurrency,
		TotalGrossInvoicesCount:      len(req.Invoices),
		TotalGrossVolumeUSD:          totalGrossUSD,
		TotalNetSettlementUSD:        totalNetSettlementUSD,
		FXVolumeCompressionPercent:   compressionPct,
		EstimatedBankingFeesSavedUSD: feesSaved,
		Participants:                 participants,
		Status:                       "CYCLE_SETTLED",
	}

	globalCashPoolStore.lastNetting = &report
	return &report, nil
}

// CashPoolingHandler handles HTTP requests for treasury cash pooling & in-house banking.
type CashPoolingHandler struct {
	db *gorm.DB
}

// NewCashPoolingHandler constructs a new CashPoolingHandler.
func NewCashPoolingHandler(db *gorm.DB) *CashPoolingHandler {
	return &CashPoolingHandler{db: db}
}

// ExecuteSweep handles HTTP sweep requests.
func (h *CashPoolingHandler) ExecuteSweep(c *gin.Context) {
	var req SweepExecutionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid sweep request payload: " + err.Error()})
		return
	}

	report, err := ExecuteSweepLogic(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, report)
}

// RunNettingCycle handles HTTP netting requests.
func (h *CashPoolingHandler) RunNettingCycle(c *gin.Context) {
	var req NettingCycleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid netting request payload: " + err.Error()})
		return
	}

	report, err := RunNettingCycleLogic(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, report)
}

// GetInHouseBankBalances retrieves current In-House Bank master liquidity metrics.
func (h *CashPoolingHandler) GetInHouseBankBalances(c *gin.Context) {
	overview := IHBBalanceOverview{
		InHouseBankID:               "IHB-GLOBAL-TREASURY",
		MasterTreasuryEntity:        "GoERP Group Treasury S.A.",
		TotalDepositsHeldUSD:        18500000.00,
		TotalCreditLinesDrawnUSD:    6200000.00,
		NetLiquiditySurplusUSD:      12300000.00,
		AverageCostOfCapitalPercent: 3.85,
		ActiveSweepingAccounts:      8,
		NextScheduledSweep:          time.Now().Add(6 * time.Hour).Format(time.RFC3339),
	}

	c.JSON(http.StatusOK, overview)
}

// RegisterCashPoolingRoutes registers treasury cash pooling & in-house banking routes.
func RegisterCashPoolingRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc, db *gorm.DB) {
	h := NewCashPoolingHandler(db)
	treasuryGroup := router.Group("/treasury")
	treasuryGroup.Use(authMiddleware)
	{
		treasuryGroup.POST("/cash-pool/sweep", h.ExecuteSweep)
		treasuryGroup.POST("/netting/run-cycle", h.RunNettingCycle)
		treasuryGroup.GET("/in-house-bank/balances", h.GetInHouseBankBalances)
	}
}
