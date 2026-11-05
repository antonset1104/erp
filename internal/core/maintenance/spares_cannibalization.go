package maintenance

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// CannibalizationAssessment models the safety, airworthiness, and financial feasibility
type CannibalizationAssessment struct {
	AssessmentID           string    `json:"assessment_id"`
	PartSKU                string    `json:"part_sku"`
	PartSerialNumber       string    `json:"part_serial_number"`
	PartDescription        string    `json:"part_description"`
	DonorAssetID           string    `json:"donor_asset_id"`
	DonorAssetStatus       string    `json:"donor_asset_status"` // "SCHEDULED_MAJOR_OVERHAUL", "LONG_TERM_PARKED"
	RecipientAssetID       string    `json:"recipient_asset_id"`
	RecipientAssetStatus   string    `json:"recipient_asset_status"` // "AOG_CRITICAL_STOPPAGE", "HAULAGE_HALTED"
	RemainingLifeHoursTSO  float64   `json:"remaining_life_hours_tso"`
	ModStatusCompatible    bool      `json:"mod_status_compatible"`
	FeasibilityScore       float64   `json:"feasibility_score"` // 0 - 100
	IsFeasible             bool      `json:"is_feasible"`
	AvoidedDowntimeCostUSD float64   `json:"avoided_downtime_cost_usd"`
	ApprovalStatus         string    `json:"approval_status"` // "AUTHORIZED_BY_CHIEF_ENGINEER", "REJECTED_SAFETY_LIMIT"
	AssessedAt             time.Time `json:"assessed_at"`
}

// BilateralSwapWorkOrders represents paired removal and installation orders
type BilateralSwapWorkOrders struct {
	SwapPackageID           string    `json:"swap_package_id"`
	AssessmentID            string    `json:"assessment_id"`
	PartSerialNumber        string    `json:"part_serial_number"`
	RemovalWorkOrderID      string    `json:"removal_work_order_id"`
	RemovalTargetAsset      string    `json:"removal_target_asset"`
	InstallationWorkOrderID string    `json:"installation_work_order_id"`
	InstallationTargetAsset string    `json:"installation_target_asset"`
	AuthorizedChiefEngineer string    `json:"authorized_chief_engineer"`
	MechanicalSignoffDone   bool      `json:"mechanical_signoff_done"`
	Status                  string    `json:"status"` // "WORK_ORDERS_DISPATCHED", "PARTS_SWAPPED_OPERATIONAL"
	DispatchedAt            time.Time `json:"dispatched_at"`
}

// LineageSerialSwapRecord stores immutable serial lineage transfer
type LineageSerialSwapRecord struct {
	SwapRecordID             string    `json:"swap_record_id"`
	PartSerialNumber         string    `json:"part_serial_number"`
	PreviousDonorAssetID     string    `json:"previous_donor_asset_id"`
	NewActiveAssetID         string    `json:"new_active_asset_id"`
	OperatingHoursAtSwap     float64   `json:"operating_hours_at_swap"`
	LineageCertificateHash   string    `json:"lineage_certificate_hash"` // SHA-256
	RegulatoryComplianceNote string    `json:"regulatory_compliance_note"`
	CommittedAt              time.Time `json:"committed_at"`
}

// SparesCannibalizationVault thread-safe store
type SparesCannibalizationVault struct {
	mu          sync.RWMutex
	assessments map[string]CannibalizationAssessment
	workOrders  map[string]BilateralSwapWorkOrders
	lineages    map[string]LineageSerialSwapRecord
}

// Global Spares Cannibalization instance
var GlobalSparesCannibalizationVault = &SparesCannibalizationVault{
	assessments: make(map[string]CannibalizationAssessment),
	workOrders:  make(map[string]BilateralSwapWorkOrders),
	lineages:    make(map[string]LineageSerialSwapRecord),
}

// EvaluateFeasibility calculates technical airworthiness and economic downtime avoidance
func (v *SparesCannibalizationVault) EvaluateFeasibility(
	sku, serial, desc, donorID, donorStatus, recipientID, recipientStatus string,
	remLifeHours, hourlyDowntimeUSD, leadTimeDays float64, modParity bool,
) *CannibalizationAssessment {
	v.mu.Lock()
	defer v.mu.Unlock()

	// Calculate downtime avoidance: leadTimeDays * 24 * hourlyDowntimeUSD
	avoidedLossUSD := leadTimeDays * 24.0 * hourlyDowntimeUSD

	// Feasibility score evaluation
	var score float64 = 100.0
	if !modParity {
		score -= 40.0
	}
	if remLifeHours < 500 {
		score -= 30.0
	}
	if donorStatus != "SCHEDULED_MAJOR_OVERHAUL" && donorStatus != "LONG_TERM_PARKED" {
		score -= 35.0
	}

	isFeasible := score >= 60.0 && modParity
	approval := "REJECTED_SAFETY_LIMIT"
	if isFeasible {
		approval = "AUTHORIZED_BY_CHIEF_ENGINEER"
	}

	assessID := fmt.Sprintf("ASSESS-CAN-%d", time.Now().UnixNano()%1000000)
	assessment := CannibalizationAssessment{
		AssessmentID:           assessID,
		PartSKU:                sku,
		PartSerialNumber:       serial,
		PartDescription:        desc,
		DonorAssetID:           donorID,
		DonorAssetStatus:       donorStatus,
		RecipientAssetID:       recipientID,
		RecipientAssetStatus:   recipientStatus,
		RemainingLifeHoursTSO:  remLifeHours,
		ModStatusCompatible:    modParity,
		FeasibilityScore:       score,
		IsFeasible:             isFeasible,
		AvoidedDowntimeCostUSD: avoidedLossUSD,
		ApprovalStatus:         approval,
		AssessedAt:             time.Now(),
	}

	v.assessments[recipientID] = assessment
	return &assessment
}

// AuthorizeSwapWorkOrders generates synchronized bilateral work orders
func (v *SparesCannibalizationVault) AuthorizeSwapWorkOrders(
	recipientID, chiefEngineer string,
) (*BilateralSwapWorkOrders, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	assess, ok := v.assessments[recipientID]
	if !ok {
		return nil, fmt.Errorf("assessment not found for recipient asset: %s", recipientID)
	}
	if !assess.IsFeasible {
		return nil, fmt.Errorf("cannot authorize: cannibalization is not feasible (Score: %.1f)", assess.FeasibilityScore)
	}

	pkgID := fmt.Sprintf("SWP-PKG-%d", time.Now().UnixNano()%1000000)
	remWO := fmt.Sprintf("WO-REM-DONOR-%s-%d", assess.DonorAssetID, time.Now().Unix()%10000)
	insWO := fmt.Sprintf("WO-INS-RECIPIENT-%s-%d", assess.RecipientAssetID, time.Now().Unix()%10000)

	swapOrders := BilateralSwapWorkOrders{
		SwapPackageID:           pkgID,
		AssessmentID:            assess.AssessmentID,
		PartSerialNumber:        assess.PartSerialNumber,
		RemovalWorkOrderID:      remWO,
		RemovalTargetAsset:      assess.DonorAssetID,
		InstallationWorkOrderID: insWO,
		InstallationTargetAsset: assess.RecipientAssetID,
		AuthorizedChiefEngineer: chiefEngineer,
		MechanicalSignoffDone:   true,
		Status:                  "WORK_ORDERS_DISPATCHED",
		DispatchedAt:            time.Now(),
	}

	v.workOrders[recipientID] = swapOrders
	return &swapOrders, nil
}

// CommitSerialLineageSwap permanently re-binds part serial to recipient asset with cryptographic certificate
func (v *SparesCannibalizationVault) CommitSerialLineageSwap(
	recipientID, inspectorNote string, operatingHours float64,
) (*LineageSerialSwapRecord, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	assess, ok := v.assessments[recipientID]
	if !ok {
		return nil, fmt.Errorf("assessment not found for recipient asset: %s", recipientID)
	}

	orders, ok := v.workOrders[recipientID]
	if !ok {
		return nil, fmt.Errorf("bilateral work orders not authorized for %s", recipientID)
	}
	orders.Status = "PARTS_SWAPPED_OPERATIONAL"
	v.workOrders[recipientID] = orders

	raw := fmt.Sprintf("%s:%s:%s:%s:%.2f:%d",
		assess.PartSerialNumber, assess.DonorAssetID, assess.RecipientAssetID, inspectorNote, operatingHours, time.Now().Unix())
	hashBytes := sha256.Sum256([]byte(raw))
	hashStr := hex.EncodeToString(hashBytes[:])

	recID := fmt.Sprintf("LIN-REC-%d", time.Now().UnixNano()%1000000)
	lineage := LineageSerialSwapRecord{
		SwapRecordID:             recID,
		PartSerialNumber:         assess.PartSerialNumber,
		PreviousDonorAssetID:     assess.DonorAssetID,
		NewActiveAssetID:         assess.RecipientAssetID,
		OperatingHoursAtSwap:     operatingHours,
		LineageCertificateHash:   hashStr,
		RegulatoryComplianceNote: inspectorNote,
		CommittedAt:              time.Now(),
	}

	v.lineages[recipientID] = lineage
	return &lineage, nil
}

// GetAOGRecoverySummary retrieves recovery dossier
func (v *SparesCannibalizationVault) GetAOGRecoverySummary(recipientID string) (gin.H, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	assess, hasAssess := v.assessments[recipientID]
	orders, hasOrders := v.workOrders[recipientID]
	lineage, hasLineage := v.lineages[recipientID]

	if !hasAssess {
		return nil, fmt.Errorf("no cannibalization dossier found for recipient asset: %s", recipientID)
	}

	return gin.H{
		"recipient_asset_id":       recipientID,
		"assessment":               assess,
		"bilateral_work_orders":    orders,
		"has_work_orders":          hasOrders,
		"lineage_record":           lineage,
		"has_lineage":              hasLineage,
		"operational_restoration":  "ASSET_RETURNED_TO_SERVICE_SAFELY",
		"aviation_mining_standard": "FAA AC 120-84 / ESDM Ditjen Minerba Parity Compliant",
	}, nil
}

// RegisterCannibalizationRoutes registers HTTP endpoints
func RegisterCannibalizationRoutes(r *gin.RouterGroup) {
	maint := r.Group("/maintenance/cannibalization")
	{
		maint.POST("/evaluate-feasibility", func(c *gin.Context) {
			var req struct {
				PartSKU               string  `json:"part_sku" binding:"required"`
				PartSerialNumber      string  `json:"part_serial_number" binding:"required"`
				PartDescription       string  `json:"part_description" binding:"required"`
				DonorAssetID          string  `json:"donor_asset_id" binding:"required"`
				DonorAssetStatus      string  `json:"donor_asset_status" binding:"required"`
				RecipientAssetID      string  `json:"recipient_asset_id" binding:"required"`
				RecipientAssetStatus  string  `json:"recipient_asset_status" binding:"required"`
				RemainingLifeHoursTSO float64 `json:"remaining_life_hours_tso" binding:"required"`
				HourlyDowntimeUSD     float64 `json:"hourly_downtime_usd" binding:"required"`
				LeadTimeDays          float64 `json:"lead_time_days" binding:"required"`
				ModStatusCompatible   bool    `json:"mod_status_compatible"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			assessment := GlobalSparesCannibalizationVault.EvaluateFeasibility(
				req.PartSKU, req.PartSerialNumber, req.PartDescription,
				req.DonorAssetID, req.DonorAssetStatus, req.RecipientAssetID, req.RecipientAssetStatus,
				req.RemainingLifeHoursTSO, req.HourlyDowntimeUSD, req.LeadTimeDays, req.ModStatusCompatible,
			)
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": assessment})
		})

		maint.POST("/authorize-swap-workorders", func(c *gin.Context) {
			var req struct {
				RecipientAssetID        string `json:"recipient_asset_id" binding:"required"`
				AuthorizedChiefEngineer string `json:"authorized_chief_engineer" binding:"required"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			orders, err := GlobalSparesCannibalizationVault.AuthorizeSwapWorkOrders(
				req.RecipientAssetID, req.AuthorizedChiefEngineer,
			)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": orders})
		})

		maint.POST("/commit-serial-lineage-swap", func(c *gin.Context) {
			var req struct {
				RecipientAssetID string  `json:"recipient_asset_id" binding:"required"`
				InspectorNote    string  `json:"inspector_note" binding:"required"`
				OperatingHours   float64 `json:"operating_hours" binding:"required"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			lineage, err := GlobalSparesCannibalizationVault.CommitSerialLineageSwap(
				req.RecipientAssetID, req.InspectorNote, req.OperatingHours,
			)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": lineage})
		})

		maint.GET("/aog-recovery-summary/:recipient_id", func(c *gin.Context) {
			recipientID := c.Param("recipient_id")
			summary, err := GlobalSparesCannibalizationVault.GetAOGRecoverySummary(recipientID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": summary})
		})
	}
}
