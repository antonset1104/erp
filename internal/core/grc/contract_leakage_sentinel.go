package grc

import (
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// ContractObligation represents structured SLA, rebate, and auto-renewal terms
type ContractObligation struct {
	ContractID               string    `json:"contract_id"`
	ContractTitle            string    `json:"contract_title"`
	CounterpartyID           string    `json:"counterparty_id"`
	CounterpartyName         string    `json:"counterparty_name"`
	ContractType             string    `json:"contract_type"` // "SUPPLIER_PROCUREMENT", "CUSTOMER_SLA_SALES"
	EffectiveDate            time.Time `json:"effective_date"`
	ExpirationDate           time.Time `json:"expiration_date"`
	AutoRenewalNoticeDays    int       `json:"auto_renewal_notice_days"`    // e.g. 60 days prior to expiry
	SLAAvailabilityPctTarget float64   `json:"sla_availability_pct_target"` // e.g. 99.5%
	SLAPenaltyRateHourlyUSD  float64   `json:"sla_penalty_rate_hourly_usd"` // e.g. $500/hr
	RebateThresholdSpendUSD  float64   `json:"rebate_threshold_spend_usd"`  // e.g. $1,000,000
	RebateRatePercent        float64   `json:"rebate_rate_percent"`         // e.g. 5.0%
	Status                   string    `json:"status"`                      // "ACTIVE_MONITORED", "TERMINATION_PENDING", "RENEWED"
	CreatedAt                time.Time `json:"created_at"`
}

// OperationalPerformanceLog captures real-time operational delivery and uptime data
type OperationalPerformanceLog struct {
	LogID                 string    `json:"log_id"`
	ContractID            string    `json:"contract_id"`
	ReportingPeriod       string    `json:"reporting_period"`        // e.g. "2026-Q4"
	ActualAvailabilityPct float64   `json:"actual_availability_pct"` // e.g. 98.2%
	ActualOutageHours     float64   `json:"actual_outage_hours"`     // e.g. 14.5 hrs
	CumulativeSpendUSD    float64   `json:"cumulative_spend_usd"`    // e.g. $1,250,000 USD
	ActualDeliveryOTIFPct float64   `json:"actual_delivery_otif_pct"`
	LoggedAt              time.Time `json:"logged_at"`
}

// LeakageAuditResult captures financial recovery and evergreen renewal warnings
type LeakageAuditResult struct {
	AuditID                  string    `json:"audit_id"`
	ContractID               string    `json:"contract_id"`
	SLAMet                   bool      `json:"sla_met"`
	SLAPenaltyRecoverableUSD float64   `json:"sla_penalty_recoverable_usd"`
	RebateEligible           bool      `json:"rebate_eligible"`
	RebateRecoverableUSD     float64   `json:"rebate_recoverable_usd"`
	TotalValueRecoveredUSD   float64   `json:"total_value_recovered_usd"`
	DaysToRenewalNotice      int       `json:"days_to_renewal_notice"`
	EvergreenNoticeAlert     string    `json:"evergreen_notice_alert"` // "CRITICAL_ACTION_REQUIRED", "NOTICE_WINDOW_CLEAR"
	DebitNoteReference       string    `json:"debit_note_reference,omitempty"`
	AuditedAt                time.Time `json:"audited_at"`
}

// ContractLeakageVault thread-safe store
type ContractLeakageVault struct {
	mu          sync.RWMutex
	obligations map[string]ContractObligation
	perfLogs    map[string]OperationalPerformanceLog
	auditLogs   map[string]LeakageAuditResult
}

// Global Contract Leakage Sentinel instance
var GlobalContractLeakageVault = &ContractLeakageVault{
	obligations: make(map[string]ContractObligation),
	perfLogs:    make(map[string]OperationalPerformanceLog),
	auditLogs:   make(map[string]LeakageAuditResult),
}

func init() {
	// Seed demo enterprise SLA contract
	expDate := time.Now().AddDate(0, 1, 15) // Expires in ~45 days (within 60-day renewal notice window)
	GlobalContractLeakageVault.obligations["CTR-CLOUD-SLA-9901"] = ContractObligation{
		ContractID:               "CTR-CLOUD-SLA-9901",
		ContractTitle:            "Managed Multi-Cloud Enterprise SLA & Data Center Hosting",
		CounterpartyID:           "VND-DATACENTER-SG-01",
		CounterpartyName:         "SingaCloud Enterprise Infra Pte Ltd",
		ContractType:             "SUPPLIER_PROCUREMENT",
		EffectiveDate:            time.Now().AddDate(-1, 0, 0),
		ExpirationDate:           expDate,
		AutoRenewalNoticeDays:    60,
		SLAAvailabilityPctTarget: 99.9,
		SLAPenaltyRateHourlyUSD:  750.0,
		RebateThresholdSpendUSD:  1000000.0,
		RebateRatePercent:        4.5,
		Status:                   "ACTIVE_MONITORED",
		CreatedAt:                time.Now().AddDate(-1, 0, 0),
	}
}

// RegisterObligation registers a new contract with governance parameters
func (v *ContractLeakageVault) RegisterObligation(
	contractID, title, counterpartyID, counterpartyName, contractType string,
	effDate, expDate time.Time, noticeDays int,
	slaTarget, penaltyRate, rebateSpend, rebatePct float64,
) ContractObligation {
	v.mu.Lock()
	defer v.mu.Unlock()

	ob := ContractObligation{
		ContractID:               contractID,
		ContractTitle:            title,
		CounterpartyID:           counterpartyID,
		CounterpartyName:         counterpartyName,
		ContractType:             contractType,
		EffectiveDate:            effDate,
		ExpirationDate:           expDate,
		AutoRenewalNoticeDays:    noticeDays,
		SLAAvailabilityPctTarget: slaTarget,
		SLAPenaltyRateHourlyUSD:  penaltyRate,
		RebateThresholdSpendUSD:  rebateSpend,
		RebateRatePercent:        rebatePct,
		Status:                   "ACTIVE_MONITORED",
		CreatedAt:                time.Now(),
	}

	v.obligations[contractID] = ob
	return ob
}

// AuditLeakagePerformance cross-references actual performance against terms
func (v *ContractLeakageVault) AuditLeakagePerformance(
	contractID, period string, actualAvail, actualOutageHours, cumSpend, actualOTIF float64,
) (*LeakageAuditResult, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	ob, ok := v.obligations[contractID]
	if !ok {
		return nil, fmt.Errorf("contract obligation not found: %s", contractID)
	}

	// 1. SLA Penalty Calculation
	slaMet := actualAvail >= ob.SLAAvailabilityPctTarget
	var penaltyUSD float64
	if !slaMet && actualOutageHours > 0 {
		penaltyUSD = actualOutageHours * ob.SLAPenaltyRateHourlyUSD
	}

	// 2. Volume Rebate Calculation
	rebateEligible := cumSpend >= ob.RebateThresholdSpendUSD
	var rebateUSD float64
	if rebateEligible && ob.RebateRatePercent > 0 {
		rebateUSD = cumSpend * (ob.RebateRatePercent / 100.0)
	}

	totalRecovered := penaltyUSD + rebateUSD
	totalRecovered = math.Round(totalRecovered*100) / 100

	// 3. Evergreen Renewal Window Calculation
	daysToExpiry := int(time.Until(ob.ExpirationDate).Hours() / 24)
	daysToNotice := daysToExpiry - ob.AutoRenewalNoticeDays
	noticeAlert := "NOTICE_WINDOW_CLEAR"
	if daysToNotice <= 0 {
		noticeAlert = "CRITICAL_ACTION_REQUIRED_EVERGREEN_NOTICE_IMMINENT"
	}

	var debitNoteRef string
	if totalRecovered > 0 {
		debitNoteRef = fmt.Sprintf("DBN-LEAKAGE-%s-%d", contractID, time.Now().Unix()%10000)
	}

	auditID := fmt.Sprintf("AUD-LEAK-%d", time.Now().UnixNano()%1000000)
	result := LeakageAuditResult{
		AuditID:                  auditID,
		ContractID:               contractID,
		SLAMet:                   slaMet,
		SLAPenaltyRecoverableUSD: penaltyUSD,
		RebateEligible:           rebateEligible,
		RebateRecoverableUSD:     rebateUSD,
		TotalValueRecoveredUSD:   totalRecovered,
		DaysToRenewalNotice:      daysToNotice,
		EvergreenNoticeAlert:     noticeAlert,
		DebitNoteReference:       debitNoteRef,
		AuditedAt:                time.Now(),
	}

	v.perfLogs[contractID] = OperationalPerformanceLog{
		LogID:                 fmt.Sprintf("LOG-PERF-%d", time.Now().UnixNano()%1000000),
		ContractID:            contractID,
		ReportingPeriod:       period,
		ActualAvailabilityPct: actualAvail,
		ActualOutageHours:     actualOutageHours,
		CumulativeSpendUSD:    cumSpend,
		ActualDeliveryOTIFPct: actualOTIF,
		LoggedAt:              time.Now(),
	}

	v.auditLogs[contractID] = result
	return &result, nil
}

// IssuePenaltyDebitNote commits formal recovery posting
func (v *ContractLeakageVault) IssuePenaltyDebitNote(contractID string) (gin.H, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	audit, ok := v.auditLogs[contractID]
	if !ok {
		return nil, fmt.Errorf("audit result not found for contract: %s", contractID)
	}

	if audit.TotalValueRecoveredUSD <= 0 {
		return nil, fmt.Errorf("no leakage value recoverable for contract: %s", contractID)
	}

	return gin.H{
		"contract_id":            contractID,
		"debit_note_number":      audit.DebitNoteReference,
		"total_recovered_amount": audit.TotalValueRecoveredUSD,
		"currency":               "USD",
		"gl_posting_debit":       "110200 - Accounts Payable Contra / Debit Note Receivable",
		"gl_posting_credit":      "510900 - SLA Penalties & Rebate Income",
		"status":                 "POSTED_AND_APPLIED_TO_NEXT_BILLING",
		"issued_at":              time.Now(),
	}, nil
}

// GetContractDossier retrieves 360-degree contract intelligence
func (v *ContractLeakageVault) GetContractDossier(contractID string) (gin.H, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	ob, hasOb := v.obligations[contractID]
	audit, hasAudit := v.auditLogs[contractID]
	perf, hasPerf := v.perfLogs[contractID]

	if !hasOb {
		return nil, fmt.Errorf("contract obligation not found: %s", contractID)
	}

	return gin.H{
		"contract_id":                 contractID,
		"obligation":                  ob,
		"audit_result":                audit,
		"has_audit":                   hasAudit,
		"performance_telemetry":       perf,
		"has_telemetry":               hasPerf,
		"contract_leakage_mitigation": "PROTECTED_AGAINST_VALUE_DRAIN",
	}, nil
}

// RegisterContractLeakageRoutes registers HTTP endpoints
func RegisterContractLeakageRoutes(r *gin.RouterGroup) {
	ctr := r.Group("/grc/contract")
	{
		ctr.POST("/register-obligation", func(c *gin.Context) {
			var req struct {
				ContractID               string  `json:"contract_id" binding:"required"`
				ContractTitle            string  `json:"contract_title" binding:"required"`
				CounterpartyID           string  `json:"counterparty_id" binding:"required"`
				CounterpartyName         string  `json:"counterparty_name" binding:"required"`
				ContractType             string  `json:"contract_type" binding:"required"`
				ExpirationDateDays       int     `json:"expiration_date_days" binding:"required"`
				AutoRenewalNoticeDays    int     `json:"auto_renewal_notice_days" binding:"required"`
				SLAAvailabilityPctTarget float64 `json:"sla_availability_pct_target" binding:"required"`
				SLAPenaltyRateHourlyUSD  float64 `json:"sla_penalty_rate_hourly_usd" binding:"required"`
				RebateThresholdSpendUSD  float64 `json:"rebate_threshold_spend_usd"`
				RebateRatePercent        float64 `json:"rebate_rate_percent"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			effDate := time.Now()
			expDate := effDate.AddDate(0, 0, req.ExpirationDateDays)

			ob := GlobalContractLeakageVault.RegisterObligation(
				req.ContractID, req.ContractTitle, req.CounterpartyID, req.CounterpartyName,
				req.ContractType, effDate, expDate, req.AutoRenewalNoticeDays,
				req.SLAAvailabilityPctTarget, req.SLAPenaltyRateHourlyUSD,
				req.RebateThresholdSpendUSD, req.RebateRatePercent,
			)
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": ob})
		})

		ctr.POST("/audit-leakage-performance", func(c *gin.Context) {
			var req struct {
				ContractID            string  `json:"contract_id" binding:"required"`
				ReportingPeriod       string  `json:"reporting_period" binding:"required"`
				ActualAvailabilityPct float64 `json:"actual_availability_pct" binding:"required"`
				ActualOutageHours     float64 `json:"actual_outage_hours"`
				CumulativeSpendUSD    float64 `json:"cumulative_spend_usd"`
				ActualDeliveryOTIFPct float64 `json:"actual_delivery_otif_pct"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			audit, err := GlobalContractLeakageVault.AuditLeakagePerformance(
				req.ContractID, req.ReportingPeriod, req.ActualAvailabilityPct,
				req.ActualOutageHours, req.CumulativeSpendUSD, req.ActualDeliveryOTIFPct,
			)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": audit})
		})

		ctr.POST("/issue-penalty-debit-note", func(c *gin.Context) {
			var req struct {
				ContractID string `json:"contract_id" binding:"required"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			res, err := GlobalContractLeakageVault.IssuePenaltyDebitNote(req.ContractID)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		ctr.GET("/contract-dossier/:contract_id", func(c *gin.Context) {
			contractID := c.Param("contract_id")
			dossier, err := GlobalContractLeakageVault.GetContractDossier(contractID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": dossier})
		})
	}
}
