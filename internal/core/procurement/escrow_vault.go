package procurement

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// EscrowMilestone defines a performance milestone tied to escrow fund release.
type EscrowMilestone struct {
	MilestoneID         string   `json:"milestone_id"`
	Description         string   `json:"description"`
	AmountIDR           float64  `json:"amount_idr"`
	PercentageOfTotal   float64  `json:"percentage_of_total"`
	RequiredConditions  []string `json:"required_conditions"`  // e.g. ["IOT_FACTORY_ACCEPTANCE_TEST", "BAST_DUAL_ESIGNATURE"]
	FulfilledConditions []string `json:"fulfilled_conditions"` // conditions validated so far
	Status              string   `json:"status"`               // "LOCKED_IN_ESCROW", "DISBURSED_RELEASED"
	ReleasedAt          string   `json:"released_at,omitempty"`
	DisbursementTxID    string   `json:"disbursement_tx_id,omitempty"`
}

// EscrowContract holds the escrow fund balance and milestone schedule.
type EscrowContract struct {
	ContractID                string            `json:"contract_id"`
	VendorID                  string            `json:"vendor_id"`
	VendorName                string            `json:"vendor_name"`
	ProjectName               string            `json:"project_name"`
	TotalContractValueIDR     float64           `json:"total_contract_value_idr"`
	TotalEscrowFundedIDR      float64           `json:"total_escrow_funded_idr"`
	TotalDisbursedIDR         float64           `json:"total_disbursed_idr"`
	RemainingEscrowBalanceIDR float64           `json:"remaining_escrow_balance_idr"`
	Milestones                []EscrowMilestone `json:"milestones"`
	EscrowStatus              string            `json:"escrow_status"` // "ACTIVE_ESCROW_FUNDED", "PARTIALLY_RELEASED", "COMPLETED_CLOSED"
	CreatedAt                 string            `json:"created_at"`
}

// CreateMilestoneEscrowRequest initiates a condition-governed escrow contract.
type CreateMilestoneEscrowRequest struct {
	ContractID            string            `json:"contract_id"`
	VendorID              string            `json:"vendor_id"`
	VendorName            string            `json:"vendor_name"`
	ProjectName           string            `json:"project_name"`
	TotalContractValueIDR float64           `json:"total_contract_value_idr"`
	Milestones            []EscrowMilestone `json:"milestones"`
}

// VerifyMilestoneReleaseRequest validates condition evidence for fund disbursement.
type VerifyMilestoneReleaseRequest struct {
	ContractID             string `json:"contract_id"`
	MilestoneID            string `json:"milestone_id"`
	FulfilledConditionCode string `json:"fulfilled_condition_code"`
	VerificationProofRef   string `json:"verification_proof_ref"`
	ApproverID             string `json:"approver_id"`
}

// VerifyMilestoneReleaseResult delivers the payment release verdict.
type VerifyMilestoneReleaseResult struct {
	ContractID                string  `json:"contract_id"`
	MilestoneID               string  `json:"milestone_id"`
	ConditionVerified         string  `json:"condition_verified"`
	AllConditionsMet          bool    `json:"all_conditions_met"`
	AmountDisbursedIDR        float64 `json:"amount_disbursed_idr"`
	RemainingEscrowBalanceIDR float64 `json:"remaining_escrow_balance_idr"`
	DisbursementTxID          string  `json:"disbursement_tx_id,omitempty"`
	Status                    string  `json:"status"`
	Message                   string  `json:"message"`
}

// EscrowVaultSummary provides an executive overview across active escrow accounts.
type EscrowVaultSummary struct {
	TotalActiveContracts       int              `json:"total_active_contracts"`
	TotalEscrowUnderCustodyIDR float64          `json:"total_escrow_under_custody_idr"`
	TotalDisbursedToDateIDR    float64          `json:"total_disbursed_to_date_idr"`
	TotalRemainingLockedIDR    float64          `json:"total_remaining_locked_idr"`
	Contracts                  []EscrowContract `json:"contracts"`
}

var (
	escrowMutex sync.RWMutex
	escrowStore = make(map[string]*EscrowContract)
)

func init() {
	demoContract := &EscrowContract{
		ContractID:                "ESCROW-2026-EPC-001",
		VendorID:                  "VEND-SIEMENS-ID",
		VendorName:                "PT Siemens Industrial Automation",
		ProjectName:               "Turnkey SMT Assembly Line 4 EPC Installation",
		TotalContractValueIDR:     12000000000.0, // Rp 12 Miliar
		TotalEscrowFundedIDR:      12000000000.0,
		TotalDisbursedIDR:         3600000000.0,
		RemainingEscrowBalanceIDR: 8400000000.0,
		Milestones: []EscrowMilestone{
			{
				MilestoneID:         "MS-01-FAT-ACCEPTANCE",
				Description:         "Factory Acceptance Test (FAT) sign-off at German facility",
				AmountIDR:           3600000000.0,
				PercentageOfTotal:   30.0,
				RequiredConditions:  []string{"IOT_FAT_CERTIFICATION", "BAST_DUAL_ESIGNATURE"},
				FulfilledConditions: []string{"IOT_FAT_CERTIFICATION", "BAST_DUAL_ESIGNATURE"},
				Status:              "DISBURSED_RELEASED",
				ReleasedAt:          "2026-10-15T10:00:00Z",
				DisbursementTxID:    "TX-DISB-20261015-081",
			},
			{
				MilestoneID:         "MS-02-SITE-COMMISSIONING",
				Description:         "On-site mechanical installation, calibration, and power-up in Cikarang",
				AmountIDR:           4800000000.0,
				PercentageOfTotal:   40.0,
				RequiredConditions:  []string{"WMS_GOODS_RECEIPT", "CALIBRATION_CERTIFICATE"},
				FulfilledConditions: []string{"WMS_GOODS_RECEIPT"},
				Status:              "LOCKED_IN_ESCROW",
			},
			{
				MilestoneID:         "MS-03-FINAL-HANDOVER",
				Description:         "Continuous 72-hour trial run at >95% OEE and warranty transfer",
				AmountIDR:           3600000000.0,
				PercentageOfTotal:   30.0,
				RequiredConditions:  []string{"OEE_72HR_VERIFICATION", "FINAL_BAST_SIGNATURE"},
				FulfilledConditions: []string{},
				Status:              "LOCKED_IN_ESCROW",
			},
		},
		EscrowStatus: "PARTIALLY_RELEASED",
		CreatedAt:    "2026-10-01T08:00:00Z",
	}
	escrowStore[demoContract.ContractID] = demoContract
}

// CreateEscrowContractLogic registers and locks funds into a condition-governed escrow account.
func CreateEscrowContractLogic(req CreateMilestoneEscrowRequest) EscrowContract {
	escrowMutex.Lock()
	defer escrowMutex.Unlock()

	if req.ContractID == "" {
		req.ContractID = fmt.Sprintf("ESCROW-%s-%03d", time.Now().Format("20060102"), time.Now().Unix()%1000)
	}
	if req.VendorName == "" {
		req.VendorName = "PT ABB Industrial Automation"
	}
	if req.ProjectName == "" {
		req.ProjectName = "High-Voltage Substation Retrofit Project"
	}
	if req.TotalContractValueIDR <= 0 {
		req.TotalContractValueIDR = 8500000000.0 // Rp 8.5 Miliar
	}

	if len(req.Milestones) == 0 {
		req.Milestones = []EscrowMilestone{
			{
				MilestoneID:         "MS-01-ENGINEERING-APPROVAL",
				Description:         "Detailed Engineering Design (DED) Approval",
				AmountIDR:           req.TotalContractValueIDR * 0.30,
				PercentageOfTotal:   30.0,
				RequiredConditions:  []string{"ENGINEERING_DRAWING_APPROVAL"},
				FulfilledConditions: []string{},
				Status:              "LOCKED_IN_ESCROW",
			},
			{
				MilestoneID:         "MS-02-HARDWARE-DELIVERY",
				Description:         "Delivery of HV Transformers to plant site",
				AmountIDR:           req.TotalContractValueIDR * 0.40,
				PercentageOfTotal:   40.0,
				RequiredConditions:  []string{"WMS_GOODS_RECEIPT", "INSURANCE_POLICY"},
				FulfilledConditions: []string{},
				Status:              "LOCKED_IN_ESCROW",
			},
			{
				MilestoneID:         "MS-03-GRID-ENERGIZATION",
				Description:         "Grid Synchronization and Energization Testing",
				AmountIDR:           req.TotalContractValueIDR * 0.30,
				PercentageOfTotal:   30.0,
				RequiredConditions:  []string{"GRID_SYNCHRONIZATION_PASS", "FINAL_BAST"},
				FulfilledConditions: []string{},
				Status:              "LOCKED_IN_ESCROW",
			},
		}
	}

	contract := EscrowContract{
		ContractID:                req.ContractID,
		VendorID:                  req.VendorID,
		VendorName:                req.VendorName,
		ProjectName:               req.ProjectName,
		TotalContractValueIDR:     req.TotalContractValueIDR,
		TotalEscrowFundedIDR:      req.TotalContractValueIDR,
		TotalDisbursedIDR:         0.0,
		RemainingEscrowBalanceIDR: req.TotalContractValueIDR,
		Milestones:                req.Milestones,
		EscrowStatus:              "ACTIVE_ESCROW_FUNDED",
		CreatedAt:                 time.Now().Format(time.RFC3339),
	}

	escrowStore[req.ContractID] = &contract
	return contract
}

// VerifyMilestoneReleaseLogic validates milestone conditions and triggers instant fund release.
func VerifyMilestoneReleaseLogic(req VerifyMilestoneReleaseRequest) (*VerifyMilestoneReleaseResult, error) {
	escrowMutex.Lock()
	defer escrowMutex.Unlock()

	c, exists := escrowStore[req.ContractID]
	if !exists {
		return nil, fmt.Errorf("escrow contract '%s' not found", req.ContractID)
	}

	msIdx := -1
	for i, ms := range c.Milestones {
		if ms.MilestoneID == req.MilestoneID {
			msIdx = i
			break
		}
	}
	if msIdx == -1 {
		return nil, fmt.Errorf("milestone '%s' not found in contract", req.MilestoneID)
	}

	ms := &c.Milestones[msIdx]
	if ms.Status == "DISBURSED_RELEASED" {
		return nil, fmt.Errorf("milestone '%s' has already been disbursed on %s", ms.MilestoneID, ms.ReleasedAt)
	}

	// Add fulfilled condition if not present
	alreadyPresent := false
	for _, f := range ms.FulfilledConditions {
		if f == req.FulfilledConditionCode {
			alreadyPresent = true
			break
		}
	}
	if !alreadyPresent && req.FulfilledConditionCode != "" {
		ms.FulfilledConditions = append(ms.FulfilledConditions, req.FulfilledConditionCode)
	}

	// Check if all required conditions are satisfied
	allMet := true
	for _, reqCond := range ms.RequiredConditions {
		matched := false
		for _, f := range ms.FulfilledConditions {
			if f == reqCond {
				matched = true
				break
			}
		}
		if !matched {
			allMet = false
			break
		}
	}

	disbursed := 0.0
	txID := ""
	status := "CONDITION_RECORDED_WAITING_REMAINING"
	msg := fmt.Sprintf("Condition '%s' verified (Proof: %s). Waiting for remaining conditions.", req.FulfilledConditionCode, req.VerificationProofRef)

	if allMet {
		ms.Status = "DISBURSED_RELEASED"
		ms.ReleasedAt = time.Now().Format(time.RFC3339)
		txID = fmt.Sprintf("TX-DISB-%s-%03d", time.Now().Format("20060102"), time.Now().Unix()%1000)
		ms.DisbursementTxID = txID

		disbursed = ms.AmountIDR
		c.TotalDisbursedIDR += disbursed
		c.RemainingEscrowBalanceIDR -= disbursed
		if c.RemainingEscrowBalanceIDR <= 0 {
			c.EscrowStatus = "COMPLETED_CLOSED"
		} else {
			c.EscrowStatus = "PARTIALLY_RELEASED"
		}
		status = "DISBURSED_RELEASED"
		msg = fmt.Sprintf("All milestone conditions verified! Instant disbursement of Rp %.0f released to vendor (TxID: %s).", disbursed, txID)
	}

	return &VerifyMilestoneReleaseResult{
		ContractID:                c.ContractID,
		MilestoneID:               ms.MilestoneID,
		ConditionVerified:         req.FulfilledConditionCode,
		AllConditionsMet:          allMet,
		AmountDisbursedIDR:        disbursed,
		RemainingEscrowBalanceIDR: c.RemainingEscrowBalanceIDR,
		DisbursementTxID:          txID,
		Status:                    status,
		Message:                   msg,
	}, nil
}

// GetEscrowVaultSummaryLogic aggregates multi-contract escrow balances.
func GetEscrowVaultSummaryLogic() EscrowVaultSummary {
	escrowMutex.RLock()
	defer escrowMutex.RUnlock()

	var contracts []EscrowContract
	var totalFunded, totalDisbursed, totalRemaining float64

	for _, c := range escrowStore {
		contracts = append(contracts, *c)
		totalFunded += c.TotalEscrowFundedIDR
		totalDisbursed += c.TotalDisbursedIDR
		totalRemaining += c.RemainingEscrowBalanceIDR
	}

	return EscrowVaultSummary{
		TotalActiveContracts:       len(contracts),
		TotalEscrowUnderCustodyIDR: totalFunded,
		TotalDisbursedToDateIDR:    totalDisbursed,
		TotalRemainingLockedIDR:    totalRemaining,
		Contracts:                  contracts,
	}
}

// RegisterEscrowRoutes registers the Smart Milestone Escrow routes.
func RegisterEscrowRoutes(r *gin.RouterGroup) {
	group := r.Group("/procurement/escrow")
	{
		group.POST("/create-milestone", func(c *gin.Context) {
			var req CreateMilestoneEscrowRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				req = CreateMilestoneEscrowRequest{}
			}
			res := CreateEscrowContractLogic(req)
			c.JSON(http.StatusCreated, res)
		})

		group.POST("/verify-release", func(c *gin.Context) {
			var req VerifyMilestoneReleaseRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
				return
			}
			res, err := VerifyMilestoneReleaseLogic(req)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, res)
		})

		group.GET("/vault-summary", func(c *gin.Context) {
			summary := GetEscrowVaultSummaryLogic()
			c.JSON(http.StatusOK, summary)
		})
	}
}
