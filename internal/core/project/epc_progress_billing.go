package project

import (
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// EPCContractMilestone models macro FIDIC construction contract terms and escrow covenants
type EPCContractMilestone struct {
	ProjectID                    string    `json:"project_id"`
	ProjectName                  string    `json:"project_name"`
	ClientEmployerName           string    `json:"client_employer_name"`
	LeadContractorName           string    `json:"lead_contractor_name"`
	TotalContractValueUSD        float64   `json:"total_contract_value_usd"`
	CumulativeCertifiedProgressPct float64 `json:"cumulative_certified_progress_pct"` // %
	RetentionPercentage          float64   `json:"retention_percentage"`               // e.g. 5.0% or 10.0%
	MobilizationAdvanceTotalUSD  float64   `json:"mobilization_advance_total_usd"`
	AdvanceAmortizationPct       float64   `json:"advance_amortization_pct"`          // % deducted per IPC
	EscrowAccountIBAN            string    `json:"escrow_account_iban"`
	CreatedAt                    time.Time `json:"created_at"`
}

// InterimPaymentCertificateIPC models monthly progress claim certified by FIDIC Engineer
type InterimPaymentCertificateIPC struct {
	CertificateNumber              string    `json:"certificate_number"`
	ProjectID                      string    `json:"project_id"`
	BillingPeriodMonth             string    `json:"billing_period_month"`
	CurrentPeriodProgressPct       float64   `json:"current_period_progress_pct"`
	GrossValuationUSD              float64   `json:"gross_valuation_usd"`
	MaterialsOnSiteAllowanceUSD    float64   `json:"materials_on_site_allowance_usd"`
	RetentionDeductedUSD           float64   `json:"retention_deducted_usd"`
	AdvanceRecoveryDeductedUSD     float64   `json:"advance_recovery_deducted_usd"`
	NetCertifiedAmountUSD          float64   `json:"net_certified_amount_usd"`
	EngineerCertificationStatus    string    `json:"engineer_certification_status"` // "CERTIFIED_APPROVED_FIDIC_CLAUSE_14", "DISPUTED_VARIATION"
	CertifiedAt                    time.Time `json:"certified_at"`
}

// SubcontractorEscrowDisbursement captures direct ring-fenced payout from project escrow to subcons
type SubcontractorEscrowDisbursement struct {
	DisbursementID      string    `json:"disbursement_id"`
	CertificateNumber   string    `json:"certificate_number"`
	SubcontractorID     string    `json:"subcontractor_id"`
	SubcontractorName   string    `json:"subcontractor_name"`
	TierLevel           int       `json:"tier_level"` // 1 = Tier 1 MEP/Civil, 2 = Tier 2 Specialist
	AllocatedAmountUSD  float64   `json:"allocated_amount_usd"`
	DisbursementStatus  string    `json:"disbursement_status"` // "ESCROW_FUNDS_RELEASED_DIRECT", "HELD_PENDING_QC_SIGN_OFF"
	BankWireReference   string    `json:"bank_wire_reference"`
	DisbursedAt         time.Time `json:"disbursed_at"`
}

// EPCProgressVault thread-safe store
type EPCProgressVault struct {
	mu           sync.RWMutex
	contracts    map[string]*EPCContractMilestone
	certificates map[string]InterimPaymentCertificateIPC
	escrowPours  map[string][]SubcontractorEscrowDisbursement
	evmMetrics   map[string]gin.H
}

// Global EPC Progress instance
var GlobalEPCProgressVault = &EPCProgressVault{
	contracts:    make(map[string]*EPCContractMilestone),
	certificates: make(map[string]InterimPaymentCertificateIPC),
	escrowPours:  make(map[string][]SubcontractorEscrowDisbursement),
	evmMetrics:   make(map[string]gin.H),
}

func init() {
	// Seed demo mega-refinery EPC project
	GlobalEPCProgressVault.contracts["PRJ-EPC-REFINERY-09"] = &EPCContractMilestone{
		ProjectID:                      "PRJ-EPC-REFINERY-09",
		ProjectName:                    "Balongan Petrochemical Hydrocracker Expansion",
		ClientEmployerName:             "PT Kilang Pertamina Internasional",
		LeadContractorName:             "Consortium Chiyoda - Wijaya Karya KSO",
		TotalContractValueUSD:          45000000.0,
		CumulativeCertifiedProgressPct: 28.0,
		RetentionPercentage:            5.0,
		MobilizationAdvanceTotalUSD:    4500000.0,
		AdvanceAmortizationPct:         10.0,
		EscrowAccountIBAN:              "ID88-BNI-ESCROW-9912001",
		CreatedAt:                      time.Now().AddDate(-1, 0, 0),
	}
}

// RegisterContract creates or updates an EPC project contract
func (v *EPCProgressVault) RegisterContract(c EPCContractMilestone) *EPCContractMilestone {
	v.mu.Lock()
	defer v.mu.Unlock()

	c.CreatedAt = time.Now()
	v.contracts[c.ProjectID] = &c
	return &c
}

// GenerateIPC computes progressive gross billings, FIDIC retention, and advance recovery
func (v *EPCProgressVault) GenerateIPC(
	projectID string, progressPct, materialsOnSiteUSD float64,
) (*InterimPaymentCertificateIPC, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	c, ok := v.contracts[projectID]
	if !ok {
		return nil, fmt.Errorf("epc contract not found for project: %s", projectID)
	}

	grossBilling := (progressPct / 100.0) * c.TotalContractValueUSD
	grossWithMaterials := grossBilling + materialsOnSiteUSD

	// Retention deduction (e.g. 5% of gross work done)
	retentionUSD := grossWithMaterials * (c.RetentionPercentage / 100.0)
	retentionUSD = math.Round(retentionUSD*100) / 100

	// Advance recovery deduction
	advanceDeductionUSD := grossBilling * (c.AdvanceAmortizationPct / 100.0)
	advanceDeductionUSD = math.Round(advanceDeductionUSD*100) / 100

	netCertified := grossWithMaterials - retentionUSD - advanceDeductionUSD
	netCertified = math.Round(netCertified*100) / 100

	c.CumulativeCertifiedProgressPct += progressPct

	ipcNo := fmt.Sprintf("IPC-%s-%03d", time.Now().Format("2006"), time.Now().Unix()%1000)
	ipc := InterimPaymentCertificateIPC{
		CertificateNumber:           ipcNo,
		ProjectID:                   projectID,
		BillingPeriodMonth:          time.Now().Format("2006-01"),
		CurrentPeriodProgressPct:    progressPct,
		GrossValuationUSD:           grossBilling,
		MaterialsOnSiteAllowanceUSD: materialsOnSiteUSD,
		RetentionDeductedUSD:        retentionUSD,
		AdvanceRecoveryDeductedUSD:  advanceDeductionUSD,
		NetCertifiedAmountUSD:       netCertified,
		EngineerCertificationStatus: "CERTIFIED_APPROVED_FIDIC_CLAUSE_14",
		CertifiedAt:                 time.Now(),
	}

	v.certificates[ipcNo] = ipc
	return &ipc, nil
}

// DisburseEscrowWaterfall executes direct bank waterfall payments to tier 1/2 subcontractors
func (v *EPCProgressVault) DisburseEscrowWaterfall(
	certNo string, subAllocations []SubcontractorEscrowDisbursement,
) ([]SubcontractorEscrowDisbursement, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	ipc, ok := v.certificates[certNo]
	if !ok {
		return nil, fmt.Errorf("interim payment certificate not found: %s", certNo)
	}

	var disbursed []SubcontractorEscrowDisbursement
	for i, sub := range subAllocations {
		sub.DisbursementID = fmt.Sprintf("DISB-SUB-%d-%d", time.Now().Unix()%10000, i+1)
		sub.CertificateNumber = ipc.CertificateNumber
		sub.DisbursementStatus = "ESCROW_FUNDS_RELEASED_DIRECT"
		sub.BankWireReference = fmt.Sprintf("SWIFT-ESCROW-REL-%d", time.Now().UnixNano()%1000000)
		sub.DisbursedAt = time.Now()
		disbursed = append(disbursed, sub)
	}

	v.escrowPours[certNo] = disbursed
	return disbursed, nil
}

// RecordEarnedValueMilestone computes CPI, SPI, and EAC under Earned Value Management (EVM)
func (v *EPCProgressVault) RecordEarnedValueMilestone(
	projectID string, pv, ev, ac float64,
) gin.H {
	v.mu.Lock()
	defer v.mu.Unlock()

	// Cost Performance Index: CPI = EV / AC
	var cpi float64 = 1.0
	if ac > 0 {
		cpi = ev / ac
	}
	cpi = math.Round(cpi*1000) / 1000

	// Schedule Performance Index: SPI = EV / PV
	var spi float64 = 1.0
	if pv > 0 {
		spi = ev / pv
	}
	spi = math.Round(spi*1000) / 1000

	status := "ON_TRACK_GOOD_PERFORMANCE"
	if cpi < 0.95 || spi < 0.95 {
		status = "WARNING_BUDGET_OR_SCHEDULE_OVERRUN"
	}

	evm := gin.H{
		"project_id":             projectID,
		"planned_value_pv_usd":   pv,
		"earned_value_ev_usd":    ev,
		"actual_cost_ac_usd":     ac,
		"cost_performance_index": cpi,
		"sched_performance_idx":  spi,
		"project_evm_status":     status,
		"updated_at":             time.Now(),
	}

	v.evmMetrics[projectID] = evm
	return evm
}

// GetFinancialProgressDossier compiles 360-degree EPC contract dossier
func (v *EPCProgressVault) GetFinancialProgressDossier(projectID string) (gin.H, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	c, hasC := v.contracts[projectID]
	evm, hasEVM := v.evmMetrics[projectID]

	if !hasC {
		return nil, fmt.Errorf("no contract records found for project: %s", projectID)
	}

	return gin.H{
		"project_id":             projectID,
		"contract":               c,
		"has_contract":           hasC,
		"earned_value_metrics":   evm,
		"has_evm":                hasEVM,
		"accounting_standard":    "IFRS 15 / PSAK 72 Over-Time Percentage of Completion",
		"contractual_framework":  "FIDIC Conditions of Contract for Construction (Red Book Cl. 14)",
		"lien_prevention_status": "DIRECT_SUBCONTRACTOR_ESCROW_RINGFENCED",
	}, nil
}

// RegisterEPCRoutes registers HTTP endpoints
func RegisterEPCRoutes(r *gin.RouterGroup) {
	epc := r.Group("/project/epc")
	{
		epc.POST("/generate-interim-payment-certificate", func(c *gin.Context) {
			var req struct {
				ProjectID           string  `json:"project_id" binding:"required"`
				ProgressPct         float64 `json:"progress_pct" binding:"required"`
				MaterialsOnSiteUSD float64 `json:"materials_on_site_usd"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			ipc, err := GlobalEPCProgressVault.GenerateIPC(
				req.ProjectID, req.ProgressPct, req.MaterialsOnSiteUSD,
			)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": ipc})
		})

		epc.POST("/disburse-escrow-waterfall", func(c *gin.Context) {
			var req struct {
				CertificateNumber string                            `json:"certificate_number" binding:"required"`
				Subcontractors    []SubcontractorEscrowDisbursement `json:"subcontractors" binding:"required"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			disbursed, err := GlobalEPCProgressVault.DisburseEscrowWaterfall(
				req.CertificateNumber, req.Subcontractors,
			)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": disbursed})
		})

		epc.POST("/record-earned-value-milestone", func(c *gin.Context) {
			var req struct {
				ProjectID    string  `json:"project_id" binding:"required"`
				PlannedValue float64 `json:"planned_value_usd" binding:"required"`
				EarnedValue  float64 `json:"earned_value_usd" binding:"required"`
				ActualCost   float64 `json:"actual_cost_usd" binding:"required"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			evm := GlobalEPCProgressVault.RecordEarnedValueMilestone(
				req.ProjectID, req.PlannedValue, req.EarnedValue, req.ActualCost,
			)
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": evm})
		})

		epc.GET("/financial-progress-dossier/:project_id", func(c *gin.Context) {
			projectID := c.Param("project_id")
			dossier, err := GlobalEPCProgressVault.GetFinancialProgressDossier(projectID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": dossier})
		})
	}
}
