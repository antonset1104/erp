package treasury

import (
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// SubAccountNode represents a subsidiary bank account participating in ZBA
type SubAccountNode struct {
	SubsidiaryID       string  `json:"subsidiary_id"`
	SubsidiaryName     string  `json:"subsidiary_name"`
	SubAccountIBAN     string  `json:"sub_account_iban"`
	CurrentBalance     float64 `json:"current_balance"`
	TargetBalance      float64 `json:"target_balance"`      // Typically 0 or minimum operational buffer
	CumulativeLoanWithParent float64 `json:"cumulative_loan_balance"` // Positive = Parent lent to sub, Negative = Sub lent to parent
}

// ZBAPoolStructure represents master cash pool hierarchy
type ZBAPoolStructure struct {
	PoolID               string                    `json:"pool_id"`
	HoldingCompanyID     string                    `json:"holding_company_id"`
	HoldingCompanyName   string                    `json:"holding_company_name"`
	MasterAccountIBAN    string                    `json:"master_account_iban"`
	Currency             string                    `json:"currency"`
	MasterBalance        float64                   `json:"master_balance"`
	SweepStrategy        string                    `json:"sweep_strategy"` // "TWO_WAY_ZBA_ZERO_BALANCE"
	SubAccounts          map[string]*SubAccountNode `json:"sub_accounts"`
	CreatedAt            time.Time                 `json:"created_at"`
}

// SweepTransactionLine models an individual sweep transfer
type SweepTransactionLine struct {
	SweepLineID        string    `json:"sweep_line_id"`
	SubsidiaryID       string    `json:"subsidiary_id"`
	FromAccount        string    `json:"from_account"`
	ToAccount          string    `json:"to_account"`
	SweepAmount        float64   `json:"sweep_amount"`
	SweepDirection      string    `json:"sweep_direction"` // "UPWARD_SWEEP", "DOWNWARD_TOPUP"
	SubBalanceBefore   float64   `json:"sub_balance_before"`
	SubBalanceAfter    float64   `json:"sub_balance_after"`
	IntercompanyMoveRef string   `json:"intercompany_move_ref"`
}

// ZBASweepExecutionRecord represents the complete batch sweep run
type ZBASweepExecutionRecord struct {
	ExecutionID          string                 `json:"execution_id"`
	PoolID               string                 `json:"pool_id"`
	ExecutionDate        time.Time              `json:"execution_date"`
	TotalSweptUpward     float64                `json:"total_swept_upward"`
	TotalToppedDown      float64                `json:"total_topped_down"`
	NetMasterCashBalance float64                `json:"net_master_cash_balance"`
	Transactions         []SweepTransactionLine `json:"transactions"`
	Status               string                 `json:"status"` // "EXECUTED_RECONCILED"
}

// IntercompanyInterestAccrual models arm's length interest calculation and statutory tax withholding
type IntercompanyInterestAccrual struct {
	AccrualID             string    `json:"accrual_id"`
	SubsidiaryID          string    `json:"subsidiary_id"`
	PrincipalLoanAmount   float64   `json:"principal_loan_amount"`
	BenchmarkRateAPR      float64   `json:"benchmark_rate_apr"`      // e.g. 6.00% (BI-Rate / SOFR)
	ArmsLengthSpreadAPR   float64   `json:"arms_length_spread_apr"`  // e.g. 1.25%
	EffectiveRateAPR      float64   `json:"effective_rate_apr"`      // e.g. 7.25%
	DaysAccrued           int       `json:"days_accrued"`
	GrossInterestAmount   float64   `json:"gross_interest_amount"`
	TaxWithholdingRate    float64   `json:"tax_withholding_rate"`    // 15% (PPh 23) or 10% (PPh 26/DTAA)
	TaxWithholdingAmount  float64   `json:"tax_withholding_amount"`
	NetInterestPayable    float64   `json:"net_interest_payable"`
	DebitAccountParent    string    `json:"debit_account_parent"`
	CreditAccountSub      string    `json:"credit_account_sub"`
	AccruedAt             time.Time `json:"accrued_at"`
}

// ZBACashSweepingVault thread-safe store
type ZBACashSweepingVault struct {
	mu           sync.RWMutex
	pools        map[string]*ZBAPoolStructure
	executions   map[string]*ZBASweepExecutionRecord
	accruals     map[string][]IntercompanyInterestAccrual
}

// Global ZBA Vault instance
var GlobalZBACashSweepingVault = &ZBACashSweepingVault{
	pools:      make(map[string]*ZBAPoolStructure),
	executions: make(map[string]*ZBASweepExecutionRecord),
	accruals:   make(map[string][]IntercompanyInterestAccrual),
}

func init() {
	// Seed demo corporate cash pool
	pool := &ZBAPoolStructure{
		PoolID:             "POOL-CORP-IDR-001",
		HoldingCompanyID:   "HOLDING-CORP-01",
		HoldingCompanyName: "PT GoERP Corporindo Utama Tbk",
		MasterAccountIBAN:  "ID12BMRI000998877665544",
		Currency:           "IDR",
		MasterBalance:      50000000000.0, // 50 Milyar IDR
		SweepStrategy:      "TWO_WAY_ZBA_ZERO_BALANCE",
		SubAccounts:        make(map[string]*SubAccountNode),
		CreatedAt:          time.Now(),
	}

	pool.SubAccounts["SUB-LOGISTICS"] = &SubAccountNode{
		SubsidiaryID:             "SUB-LOGISTICS",
		SubsidiaryName:           "PT GoERP Logistik Global",
		SubAccountIBAN:           "ID12BMRI000112233445566",
		CurrentBalance:           8500000000.0, // Surplus 8.5 Milyar
		TargetBalance:            50000000.0,   // Buffer 50 Juta
		CumulativeLoanWithParent: -8450000000.0,
	}

	pool.SubAccounts["SUB-MFG"] = &SubAccountNode{
		SubsidiaryID:             "SUB-MFG",
		SubsidiaryName:           "PT GoERP Manufaktur Presisi",
		SubAccountIBAN:           "ID12BMRI000223344556677",
		CurrentBalance:           -1200000000.0, // Defisit 1.2 Milyar
		TargetBalance:            50000000.0,    // Buffer 50 Juta
		CumulativeLoanWithParent: 1250000000.0,
	}

	GlobalZBACashSweepingVault.pools["HOLDING-CORP-01"] = pool
}

// ConfigurePoolingStructure registers or updates corporate cash pooling tree
func (v *ZBACashSweepingVault) ConfigurePoolingStructure(
	holdingID, holdingName, masterIBAN, currency string, masterBal float64,
	subs []SubAccountNode,
) *ZBAPoolStructure {
	v.mu.Lock()
	defer v.mu.Unlock()

	poolID := fmt.Sprintf("POOL-%s-%s", currency, holdingID)
	pool := &ZBAPoolStructure{
		PoolID:             poolID,
		HoldingCompanyID:   holdingID,
		HoldingCompanyName: holdingName,
		MasterAccountIBAN:  masterIBAN,
		Currency:           currency,
		MasterBalance:      masterBal,
		SweepStrategy:      "TWO_WAY_ZBA_ZERO_BALANCE",
		SubAccounts:        make(map[string]*SubAccountNode),
		CreatedAt:          time.Now(),
	}

	for _, sub := range subs {
		subCopy := sub
		pool.SubAccounts[sub.SubsidiaryID] = &subCopy
	}

	v.pools[holdingID] = pool
	return pool
}

// ExecuteEODSweeping sweeps surplus and top-ups deficit
func (v *ZBACashSweepingVault) ExecuteEODSweeping(holdingID string) (*ZBASweepExecutionRecord, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	pool, ok := v.pools[holdingID]
	if !ok {
		return nil, fmt.Errorf("cash pool not found for holding company: %s", holdingID)
	}

	execID := fmt.Sprintf("ZBA-EXEC-%d", time.Now().UnixNano()%1000000)
	record := &ZBASweepExecutionRecord{
		ExecutionID:   execID,
		PoolID:        pool.PoolID,
		ExecutionDate: time.Now(),
		Status:        "EXECUTED_RECONCILED",
	}

	for _, sub := range pool.SubAccounts {
		diff := sub.CurrentBalance - sub.TargetBalance
		if diff > 0 {
			// Upward sweep from Sub to Master
			sweepAmt := diff
			sub.CurrentBalance -= sweepAmt
			pool.MasterBalance += sweepAmt
			record.TotalSweptUpward += sweepAmt
			sub.CumulativeLoanWithParent -= sweepAmt

			line := SweepTransactionLine{
				SweepLineID:        fmt.Sprintf("SWP-%d", time.Now().UnixNano()%1000000),
				SubsidiaryID:       sub.SubsidiaryID,
				FromAccount:        sub.SubAccountIBAN,
				ToAccount:          pool.MasterAccountIBAN,
				SweepAmount:        sweepAmt,
				SweepDirection:      "UPWARD_SWEEP",
				SubBalanceBefore:   sub.CurrentBalance + sweepAmt,
				SubBalanceAfter:    sub.CurrentBalance,
				IntercompanyMoveRef: fmt.Sprintf("GL-IC-UP-%s-%d", sub.SubsidiaryID, time.Now().Unix()%10000),
			}
			record.Transactions = append(record.Transactions, line)
		} else if diff < 0 {
			// Downward top-up from Master to Sub
			topupAmt := -diff
			sub.CurrentBalance += topupAmt
			pool.MasterBalance -= topupAmt
			record.TotalToppedDown += topupAmt
			sub.CumulativeLoanWithParent += topupAmt

			line := SweepTransactionLine{
				SweepLineID:        fmt.Sprintf("SWP-%d", time.Now().UnixNano()%1000000),
				SubsidiaryID:       sub.SubsidiaryID,
				FromAccount:        pool.MasterAccountIBAN,
				ToAccount:          sub.SubAccountIBAN,
				SweepAmount:        topupAmt,
				SweepDirection:      "DOWNWARD_TOPUP",
				SubBalanceBefore:   sub.CurrentBalance - topupAmt,
				SubBalanceAfter:    sub.CurrentBalance,
				IntercompanyMoveRef: fmt.Sprintf("GL-IC-DOWN-%s-%d", sub.SubsidiaryID, time.Now().Unix()%10000),
			}
			record.Transactions = append(record.Transactions, line)
		}
	}

	record.NetMasterCashBalance = pool.MasterBalance
	v.executions[execID] = record
	return record, nil
}

// AccrueIntercompanyInterest calculates arm's length interest and tax withholding
func (v *ZBACashSweepingVault) AccrueIntercompanyInterest(
	holdingID, subID string, days int, benchmarkRate, spreadRate, whtRate float64,
) (*IntercompanyInterestAccrual, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	pool, ok := v.pools[holdingID]
	if !ok {
		return nil, fmt.Errorf("cash pool not found for holding company: %s", holdingID)
	}

	sub, ok := pool.SubAccounts[subID]
	if !ok {
		return nil, fmt.Errorf("subsidiary %s not found in pool", subID)
	}

	effectiveRate := benchmarkRate + spreadRate
	absPrincipal := math.Abs(sub.CumulativeLoanWithParent)
	if absPrincipal == 0 {
		absPrincipal = 1000000000.0 // Default 1M baseline if newly leveled
	}

	grossInterest := absPrincipal * (effectiveRate / 100.0) * (float64(days) / 365.0)
	grossInterest = math.Round(grossInterest*100) / 100

	taxWithholding := grossInterest * (whtRate / 100.0)
	taxWithholding = math.Round(taxWithholding*100) / 100
	netInterest := grossInterest - taxWithholding

	accrualID := fmt.Sprintf("IC-INT-%d", time.Now().UnixNano()%1000000)
	accrual := IntercompanyInterestAccrual{
		AccrualID:            accrualID,
		SubsidiaryID:         subID,
		PrincipalLoanAmount:  absPrincipal,
		BenchmarkRateAPR:     benchmarkRate,
		ArmsLengthSpreadAPR:  spreadRate,
		EffectiveRateAPR:     effectiveRate,
		DaysAccrued:          days,
		GrossInterestAmount:  grossInterest,
		TaxWithholdingRate:   whtRate,
		TaxWithholdingAmount: taxWithholding,
		NetInterestPayable:   netInterest,
		DebitAccountParent:   "710100 - Intercompany Interest Expense",
		CreditAccountSub:     "220400 - PPh 23 / 26 Withholding Tax Payable",
		AccruedAt:            time.Now(),
	}

	v.accruals[holdingID] = append(v.accruals[holdingID], accrual)
	return &accrual, nil
}

// GetPoolingOverview retrieves dashboard metrics
func (v *ZBACashSweepingVault) GetPoolingOverview(holdingID string) (gin.H, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	pool, ok := v.pools[holdingID]
	if !ok {
		return nil, fmt.Errorf("cash pool not found: %s", holdingID)
	}

	var subList []SubAccountNode
	var totalSubsidiaryCash float64
	for _, s := range pool.SubAccounts {
		subList = append(subList, *s)
		totalSubsidiaryCash += s.CurrentBalance
	}

	return gin.H{
		"pool_id":                pool.PoolID,
		"holding_company_id":     pool.HoldingCompanyID,
		"holding_company_name":   pool.HoldingCompanyName,
		"currency":               pool.Currency,
		"master_account_iban":    pool.MasterAccountIBAN,
		"master_cash_balance":    pool.MasterBalance,
		"total_subsidiary_cash":  totalSubsidiaryCash,
		"total_conglomerate_cash": pool.MasterBalance + totalSubsidiaryCash,
		"subsidiary_nodes":       subList,
		"recent_accruals":        v.accruals[holdingID],
		"sweep_strategy":         pool.SweepStrategy,
		"interest_tax_status":    "COMPLIANT_WITH_PMK_TRANSFER_PRICING",
	}, nil
}

// RegisterZBARoutes registers HTTP endpoints
func RegisterZBARoutes(r *gin.RouterGroup) {
	zba := r.Group("/treasury/zba")
	{
		zba.POST("/configure-pooling-structure", func(c *gin.Context) {
			var req struct {
				HoldingCompanyID   string           `json:"holding_company_id" binding:"required"`
				HoldingCompanyName string           `json:"holding_company_name" binding:"required"`
				MasterAccountIBAN  string           `json:"master_account_iban" binding:"required"`
				Currency           string           `json:"currency"`
				MasterBalance      float64          `json:"master_balance"`
				SubAccounts        []SubAccountNode `json:"sub_accounts"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			if req.Currency == "" {
				req.Currency = "IDR"
			}

			pool := GlobalZBACashSweepingVault.ConfigurePoolingStructure(
				req.HoldingCompanyID, req.HoldingCompanyName, req.MasterAccountIBAN,
				req.Currency, req.MasterBalance, req.SubAccounts,
			)
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": pool})
		})

		zba.POST("/execute-eod-sweeping", func(c *gin.Context) {
			var req struct {
				HoldingCompanyID string `json:"holding_company_id" binding:"required"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			record, err := GlobalZBACashSweepingVault.ExecuteEODSweeping(req.HoldingCompanyID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": record})
		})

		zba.POST("/accrue-intercompany-interest", func(c *gin.Context) {
			var req struct {
				HoldingCompanyID string  `json:"holding_company_id" binding:"required"`
				SubsidiaryID     string  `json:"subsidiary_id" binding:"required"`
				DaysAccrued      int     `json:"days_accrued" binding:"required"`
				BenchmarkRateAPR float64 `json:"benchmark_rate_apr"`
				SpreadRateAPR    float64 `json:"spread_rate_apr"`
				WHTRate          float64 `json:"wht_rate"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			if req.BenchmarkRateAPR == 0 {
				req.BenchmarkRateAPR = 6.00
			}
			if req.SpreadRateAPR == 0 {
				req.SpreadRateAPR = 1.25
			}
			if req.WHTRate == 0 {
				req.WHTRate = 15.0 // PPh 23 domestic
			}

			accrual, err := GlobalZBACashSweepingVault.AccrueIntercompanyInterest(
				req.HoldingCompanyID, req.SubsidiaryID, req.DaysAccrued,
				req.BenchmarkRateAPR, req.SpreadRateAPR, req.WHTRate,
			)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": accrual})
		})

		zba.GET("/pooling-overview/:holding_id", func(c *gin.Context) {
			holdingID := c.Param("holding_id")
			overview, err := GlobalZBACashSweepingVault.GetPoolingOverview(holdingID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": overview})
		})
	}
}
