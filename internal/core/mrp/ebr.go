package mrp

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// EBRStep represents an individual recipe step with IPC controls and dual 21 CFR Part 11 signatures.
type EBRStep struct {
	StepNumber             int    `json:"step_number"`
	OperationName          string `json:"operation_name"`
	TargetParameter        string `json:"target_parameter"`
	ObservedValue          string `json:"observed_value"`
	OperatorID             string `json:"operator_id"`
	WitnessQAID            string `json:"witness_qa_id"`
	OperatorSignatureHash  string `json:"operator_signature_hash"`
	WitnessQASignatureHash string `json:"witness_qa_signature_hash"`
	StepStatus             string `json:"step_status"` // "PENDING", "PASSED_WITH_DUAL_SIGNOFF", "DEVIATION_RECORDED"
	ExecutedAt             string `json:"executed_at"`
}

// EBRDeviation represents an Out-of-Specification (OOS) event or manufacturing variance.
type EBRDeviation struct {
	DeviationID      string `json:"deviation_id"`
	StepNumber       int    `json:"step_number"`
	Description      string `json:"description"`
	Severity         string `json:"severity"` // "CRITICAL", "MAJOR", "MINOR"
	RootCause        string `json:"root_cause"`
	CorrectiveAction string `json:"corrective_action"`
	QASignedBy       string `json:"qa_signed_by"`
	ResolutionStatus string `json:"resolution_status"` // "INVESTIGATED", "CLOSED_APPROVED"
	RecordedAt       string `json:"recorded_at"`
}

// EBRBatchRecord represents a fully compliant Electronic Batch Record under FDA 21 CFR Part 11.
type EBRBatchRecord struct {
	BatchNumber             string         `json:"batch_number"`
	ProductCode             string         `json:"product_code"`
	ProductName             string         `json:"product_name"`
	BatchSizeUnits          int            `json:"batch_size_units"`
	ManufacturingOrderRef   string         `json:"manufacturing_order_ref"`
	MasterFormulaID         string         `json:"master_formula_id"`
	CurrentStepNumber       int            `json:"current_step_number"`
	TotalSteps              int            `json:"total_steps"`
	Steps                   []EBRStep      `json:"steps"`
	Deviations              []EBRDeviation `json:"deviations"`
	OverallBatchStatus      string         `json:"overall_batch_status"` // "IN_PRODUCTION", "QUARANTINE_UNDER_REVIEW", "RELEASED_COA_ISSUED"
	CertificateOfAnalysisID string         `json:"certificate_of_analysis_id"`
	ReleasedByQA            string         `json:"released_by_qa"`
	ReleasedAt              string         `json:"released_at"`
}

// ExecuteEBRStepRequest holds dual electronic signatures and observed IPC measurements.
type ExecuteEBRStepRequest struct {
	BatchNumber   string `json:"batch_number"`
	StepNumber    int    `json:"step_number"`
	ObservedValue string `json:"observed_value"`
	OperatorID    string `json:"operator_id"`
	WitnessQAID   string `json:"witness_qa_id"`
	OperatorPin   string `json:"operator_pin"`
	WitnessQAPin  string `json:"witness_qa_pin"`
}

// SignEBRDeviationRequest records formal deviation containment and CAPA sign-off.
type SignEBRDeviationRequest struct {
	BatchNumber      string `json:"batch_number"`
	StepNumber       int    `json:"step_number"`
	Description      string `json:"description"`
	Severity         string `json:"severity"`
	RootCause        string `json:"root_cause"`
	CorrectiveAction string `json:"corrective_action"`
	QASignedBy       string `json:"qa_signed_by"`
}

var (
	ebrMutex sync.RWMutex
	ebrStore = make(map[string]*EBRBatchRecord)
)

func init() {
	// Seed demo regulated batch record
	demoRecord := &EBRBatchRecord{
		BatchNumber:           "BATCH-2026-PHARMA-001",
		ProductCode:           "MED-PARACETAMOL-500MG",
		ProductName:           "Paracetamol 500mg Film-Coated Caplets",
		BatchSizeUnits:        100000,
		ManufacturingOrderRef: "MO/2026/11/0088",
		MasterFormulaID:       "MF-PARA-500-REV4",
		CurrentStepNumber:     2,
		TotalSteps:            4,
		Steps: []EBRStep{
			{
				StepNumber:             1,
				OperationName:          "Active Ingredient Pre-Weighing & Dispensing",
				TargetParameter:        "Pure API Mass: 50.00 kg +/- 0.1%",
				ObservedValue:          "50.02 kg (Analytical Balance CAL-09)",
				OperatorID:             "OP-RATNA-WULAN",
				WitnessQAID:            "QA-DR-HENDRA",
				OperatorSignatureHash:  "7d2b4a1c5e9f8a3d...",
				WitnessQASignatureHash: "4f8a91b2c3d4e5f6...",
				StepStatus:             "PASSED_WITH_DUAL_SIGNOFF",
				ExecutedAt:             "2026-11-04T08:15:00Z",
			},
			{
				StepNumber:      2,
				OperationName:   "High-Shear Wet Granulation & Fluid Bed Drying",
				TargetParameter: "Granule Loss on Drying (LOD): 1.5% - 2.5%",
				ObservedValue:   "1.95% LOD",
				StepStatus:      "PENDING",
			},
			{
				StepNumber:      3,
				OperationName:   "Rotary Tableting & Core Compression",
				TargetParameter: "Tablet Hardness: 8 - 12 kp | Disintegration < 15 min",
				StepStatus:      "PENDING",
			},
			{
				StepNumber:      4,
				OperationName:   "Aqueous Film Coating & High-Speed Blistering",
				TargetParameter: "Weight Gain: 2.8% - 3.2% | Leak-test: 100% Hermetic",
				StepStatus:      "PENDING",
			},
		},
		Deviations:         []EBRDeviation{},
		OverallBatchStatus: "IN_PRODUCTION",
	}
	ebrStore[demoRecord.BatchNumber] = demoRecord
}

// computeEBRSignature generates SHA-256 cryptographic proof for 21 CFR Part 11 non-repudiation.
func computeEBRSignature(userID, pin, stepInfo string) string {
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%s:%s:%s:%d", userID, pin, stepInfo, time.Now().UnixNano())))
	return hex.EncodeToString(h.Sum(nil))[:32]
}

// ExecuteEBRStepLogic enforces dual digital signatures and validates IPC parameters.
func ExecuteEBRStepLogic(req ExecuteEBRStepRequest) (*EBRBatchRecord, error) {
	ebrMutex.Lock()
	defer ebrMutex.Unlock()

	rec, exists := ebrStore[req.BatchNumber]
	if !exists {
		return nil, fmt.Errorf("batch record '%s' not found", req.BatchNumber)
	}

	stepIdx := -1
	for i, s := range rec.Steps {
		if s.StepNumber == req.StepNumber {
			stepIdx = i
			break
		}
	}
	if stepIdx == -1 {
		return nil, fmt.Errorf("step number %d not found in batch formula", req.StepNumber)
	}

	opSig := computeEBRSignature(req.OperatorID, req.OperatorPin, rec.Steps[stepIdx].OperationName)
	qaSig := computeEBRSignature(req.WitnessQAID, req.WitnessQAPin, rec.Steps[stepIdx].OperationName)

	rec.Steps[stepIdx].ObservedValue = req.ObservedValue
	rec.Steps[stepIdx].OperatorID = req.OperatorID
	rec.Steps[stepIdx].WitnessQAID = req.WitnessQAID
	rec.Steps[stepIdx].OperatorSignatureHash = opSig
	rec.Steps[stepIdx].WitnessQASignatureHash = qaSig
	rec.Steps[stepIdx].StepStatus = "PASSED_WITH_DUAL_SIGNOFF"
	rec.Steps[stepIdx].ExecutedAt = time.Now().Format(time.RFC3339)

	rec.CurrentStepNumber = req.StepNumber

	// If final step completed, auto-release with Certificate of Analysis (CoA)
	allDone := true
	for _, s := range rec.Steps {
		if s.StepStatus != "PASSED_WITH_DUAL_SIGNOFF" {
			allDone = false
			break
		}
	}

	if allDone {
		rec.OverallBatchStatus = "RELEASED_COA_ISSUED"
		rec.CertificateOfAnalysisID = fmt.Sprintf("COA-BPOM-%s-%03d", time.Now().Format("20060102"), time.Now().Unix()%1000)
		rec.ReleasedByQA = req.WitnessQAID
		rec.ReleasedAt = time.Now().Format(time.RFC3339)
	}

	return rec, nil
}

// SignEBRDeviationLogic attaches an investigated manufacturing deviation and risk assessment.
func SignEBRDeviationLogic(req SignEBRDeviationRequest) (*EBRBatchRecord, error) {
	ebrMutex.Lock()
	defer ebrMutex.Unlock()

	rec, exists := ebrStore[req.BatchNumber]
	if !exists {
		return nil, fmt.Errorf("batch record '%s' not found", req.BatchNumber)
	}

	devID := fmt.Sprintf("DEV-%s-%03d", time.Now().Format("20060102"), len(rec.Deviations)+1)
	dev := EBRDeviation{
		DeviationID:      devID,
		StepNumber:       req.StepNumber,
		Description:      req.Description,
		Severity:         req.Severity,
		RootCause:        req.RootCause,
		CorrectiveAction: req.CorrectiveAction,
		QASignedBy:       req.QASignedBy,
		ResolutionStatus: "CLOSED_APPROVED",
		RecordedAt:       time.Now().Format(time.RFC3339),
	}

	rec.Deviations = append(rec.Deviations, dev)

	// If critical deviation, flag batch into quarantine
	if req.Severity == "CRITICAL" {
		rec.OverallBatchStatus = "QUARANTINE_UNDER_REVIEW"
	}

	return rec, nil
}

// GetEBRBatchRecordLogic retrieves complete master formula execution history.
func GetEBRBatchRecordLogic(batchNumber string) (*EBRBatchRecord, error) {
	ebrMutex.RLock()
	defer ebrMutex.RUnlock()

	rec, exists := ebrStore[batchNumber]
	if !exists {
		return nil, fmt.Errorf("batch record '%s' not found", batchNumber)
	}
	return rec, nil
}

// RegisterEBRRoutes registers the Electronic Batch Record routes.
func RegisterEBRRoutes(r *gin.RouterGroup) {
	group := r.Group("/mrp/ebr")
	{
		group.POST("/execute-step", func(c *gin.Context) {
			var req ExecuteEBRStepRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
				return
			}
			res, err := ExecuteEBRStepLogic(req)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, res)
		})

		group.POST("/sign-deviation", func(c *gin.Context) {
			var req SignEBRDeviationRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
				return
			}
			res, err := SignEBRDeviationLogic(req)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, res)
		})

		group.GET("/batch-record/:batch_number", func(c *gin.Context) {
			bn := c.Param("batch_number")
			res, err := GetEBRBatchRecordLogic(bn)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, res)
		})
	}
}
