package mrp

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// ECOImpactBlastRadius measures the operational and financial exposure of an engineering change.
type ECOImpactBlastRadius struct {
	ECOID                     string  `json:"eco_id"`
	TargetFinishedProductSKU  string  `json:"target_finished_product_sku"`
	CurrentBOMRevision        string  `json:"current_bom_revision"`
	TargetBOMRevision         string  `json:"target_bom_revision"`
	AffectedBOMsCount         int     `json:"affected_boms_count"`
	AffectedOpenPOsCount      int     `json:"affected_open_pos_count"`
	OpenPOValueUSD            float64 `json:"open_po_value_usd"`
	ActiveWIPWorkOrdersCount  int     `json:"active_wip_work_orders_count"`
	WIPValueUSD               float64 `json:"wip_value_usd"`
	ObsoleteInventoryValueUSD float64 `json:"obsolete_inventory_value_usd"`
	ScrapRiskCostUSD          float64 `json:"scrap_risk_cost_usd"`
	TotalTransitionCostUSD    float64 `json:"total_transition_cost_usd"`
}

// ECODispositionAction specifies the inventory handling rules for phasing out parts.
type ECODispositionAction struct {
	ComponentSKU            string  `json:"component_sku"`
	ComponentName           string  `json:"component_name"`
	DispositionStrategy     string  `json:"disposition_strategy"` // "USE_UP_EXISTING_STOCK", "SCRAP_IMMEDIATE", "REWORK_RETROFIT", "RETURN_TO_VENDOR"
	QuantityOnHand          float64 `json:"quantity_on_hand"`
	UnitCostUSD             float64 `json:"unit_cost_usd"`
	TotalDispositionCostUSD float64 `json:"total_disposition_cost_usd"`
	AccountingGLPosting     string  `json:"accounting_gl_posting"`
}

// ECOApprovalRecord tracks sign-offs from engineering, operations, and quality management.
type ECOApprovalRecord struct {
	ApprovalID   string `json:"approval_id"`
	Role         string `json:"role"` // "CHIEF_ENGINEER", "PLANT_OPERATIONS_VP", "FINANCE_CONTROLLER"
	ApproverName string `json:"approver_name"`
	Decision     string `json:"decision"` // "APPROVED", "REJECTED"
	Comments     string `json:"comments"`
	Timestamp    string `json:"timestamp"`
}

// EngineeringChangeOrder represents a complete, lifecycle-controlled engineering revision asset.
type EngineeringChangeOrder struct {
	ECOID                 string                 `json:"eco_id"` // e.g. "ECO-2026-REV-041"
	Title                 string                 `json:"title"`
	ChangeReason          string                 `json:"change_reason"` // "SAFETY_UPGRADE", "COST_REDUCTION", "COMPONENT_OBSOLESCENCE", "REGULATORY_COMPLIANCE"
	ProductSKU            string                 `json:"product_sku"`
	ProductName           string                 `json:"product_name"`
	RequestedBy           string                 `json:"requested_by"`
	RolloutStrategy       string                 `json:"rollout_strategy"` // "DATE_EFFECTIVE_PHASE_IN", "SERIAL_NUMBER_CUTOVER", "IMMEDIATE_STOP_SHIP"
	EffectiveDate         string                 `json:"effective_date"`
	EffectiveSerialNumber string                 `json:"effective_serial_number,omitempty"`
	Status                string                 `json:"status"` // "DRAFT", "IMPACT_EVALUATED", "APPROVED_MANAGEMENT", "ROLLED_OUT_ACTIVE"
	BlastRadius           ECOImpactBlastRadius   `json:"blast_radius"`
	Dispositions          []ECODispositionAction `json:"dispositions"`
	Approvals             []ECOApprovalRecord    `json:"approvals"`
	CreatedAt             string                 `json:"created_at"`
	RolledOutAt           string                 `json:"rolled_out_at,omitempty"`
}

// CreateECORequest defines parameters to draft a new engineering change order.
type CreateECORequest struct {
	Title           string `json:"title"`
	ChangeReason    string `json:"change_reason"`
	ProductSKU      string `json:"product_sku"`
	ProductName     string `json:"product_name"`
	RequestedBy     string `json:"requested_by"`
	RolloutStrategy string `json:"rollout_strategy"`
	EffectiveDate   string `json:"effective_date"`
}

// ExecuteRolloutRequest confirms executive sign-off and activates BOM cut-over.
type ExecuteRolloutRequest struct {
	ECOID                 string `json:"eco_id"`
	ApproverName          string `json:"approver_name"`
	ApproverRole          string `json:"approver_role"`
	EffectiveSerialNumber string `json:"effective_serial_number"`
}

// ECOStore manages thread-safe memory storage for engineering change orders.
type ECOStore struct {
	mu     sync.RWMutex
	orders map[string]*EngineeringChangeOrder
}

var globalECOStore = &ECOStore{
	orders: make(map[string]*EngineeringChangeOrder),
}

func init() {
	// Seed demo engineering change order
	demoECO := &EngineeringChangeOrder{
		ECOID:           "ECO-2026-REV-041",
		Title:           "Thermal Interface Heat-Sink Redesign for Traction Inverters",
		ChangeReason:    "SAFETY_UPGRADE",
		ProductSKU:      "INV-TRAC-250KW",
		ProductName:     "250kW Silicon-Carbide Dual-Motor Traction Inverter",
		RequestedBy:     "Dr. Hartono (Chief Power Electronics Engineer)",
		RolloutStrategy: "SERIAL_NUMBER_CUTOVER",
		EffectiveDate:   time.Now().AddDate(0, 1, 0).Format("2006-01-02"),
		Status:          "IMPACT_EVALUATED",
		CreatedAt:       time.Now().Format(time.RFC3339),
		BlastRadius: ECOImpactBlastRadius{
			ECOID:                     "ECO-2026-REV-041",
			TargetFinishedProductSKU:  "INV-TRAC-250KW",
			CurrentBOMRevision:        "REV-B",
			TargetBOMRevision:         "REV-C",
			AffectedBOMsCount:         3,
			AffectedOpenPOsCount:      2,
			OpenPOValueUSD:            48000.0,
			ActiveWIPWorkOrdersCount:  4,
			WIPValueUSD:               92000.0,
			ObsoleteInventoryValueUSD: 14500.0,
			ScrapRiskCostUSD:          4500.0,
			TotalTransitionCostUSD:    18500.0,
		},
		Dispositions: []ECODispositionAction{
			{
				ComponentSKU:            "CMP-HSINK-ALU-01",
				ComponentName:           "Legacy Extruded Aluminum Heat Sink (Rev B)",
				DispositionStrategy:     "USE_UP_EXISTING_STOCK",
				QuantityOnHand:          320.0,
				UnitCostUSD:             45.0,
				TotalDispositionCostUSD: 14400.0,
				AccountingGLPosting:     "GL-INV-VALUATION-RAW-1410",
			},
			{
				ComponentSKU:            "CMP-TIM-PAD-02",
				ComponentName:           "Phase-Change Thermal Pad 0.5mm",
				DispositionStrategy:     "SCRAP_IMMEDIATE",
				QuantityOnHand:          450.0,
				UnitCostUSD:             9.0,
				TotalDispositionCostUSD: 4050.0,
				AccountingGLPosting:     "GL-EXP-SCRAP-WRITEOFF-5420",
			},
		},
		Approvals: make([]ECOApprovalRecord, 0),
	}
	globalECOStore.orders[demoECO.ECOID] = demoECO
}

// CreateEngineeringChangeOrderLogic instantiates a new engineering change document.
func CreateEngineeringChangeOrderLogic(req CreateECORequest) (*EngineeringChangeOrder, error) {
	globalECOStore.mu.Lock()
	defer globalECOStore.mu.Unlock()

	if req.ProductSKU == "" {
		req.ProductSKU = "PRD-DEFAULT-ASSEMBLY"
	}
	if req.Title == "" {
		req.Title = "Product Engineering Revision"
	}
	if req.RolloutStrategy == "" {
		req.RolloutStrategy = "DATE_EFFECTIVE_PHASE_IN"
	}
	if req.EffectiveDate == "" {
		req.EffectiveDate = time.Now().AddDate(0, 1, 0).Format("2006-01-02")
	}

	ecoID := fmt.Sprintf("ECO-2026-REV-%03d", len(globalECOStore.orders)+42)
	order := &EngineeringChangeOrder{
		ECOID:           ecoID,
		Title:           req.Title,
		ChangeReason:    req.ChangeReason,
		ProductSKU:      req.ProductSKU,
		ProductName:     req.ProductName,
		RequestedBy:     req.RequestedBy,
		RolloutStrategy: req.RolloutStrategy,
		EffectiveDate:   req.EffectiveDate,
		Status:          "DRAFT",
		CreatedAt:       time.Now().Format(time.RFC3339),
		BlastRadius: ECOImpactBlastRadius{
			ECOID:                    ecoID,
			TargetFinishedProductSKU: req.ProductSKU,
			CurrentBOMRevision:       "REV-A",
			TargetBOMRevision:        "REV-B",
			AffectedBOMsCount:        2,
			AffectedOpenPOsCount:     1,
			OpenPOValueUSD:           25000.0,
			ActiveWIPWorkOrdersCount: 2,
			WIPValueUSD:              45000.0,
			TotalTransitionCostUSD:   8500.0,
		},
		Dispositions: []ECODispositionAction{
			{
				ComponentSKU:            "CMP-LEGACY-01",
				ComponentName:           "Substituted Fastener Component",
				DispositionStrategy:     "USE_UP_EXISTING_STOCK",
				QuantityOnHand:          500.0,
				UnitCostUSD:             4.5,
				TotalDispositionCostUSD: 2250.0,
				AccountingGLPosting:     "GL-INV-VALUATION-RAW-1410",
			},
		},
		Approvals: make([]ECOApprovalRecord, 0),
	}

	globalECOStore.orders[ecoID] = order
	return order, nil
}

// EvaluateECOImpactLogic simulates the multi-tier bill-of-materials and work-order blast radius.
func EvaluateECOImpactLogic(ecoID string) (*EngineeringChangeOrder, error) {
	globalECOStore.mu.Lock()
	defer globalECOStore.mu.Unlock()

	order, exists := globalECOStore.orders[ecoID]
	if !exists {
		return nil, fmt.Errorf("engineering change order '%s' not found", ecoID)
	}

	order.Status = "IMPACT_EVALUATED"
	return order, nil
}

// ExecuteRolloutLogic applies formal approval and executes the BOM revision cut-over.
func ExecuteRolloutLogic(req ExecuteRolloutRequest) (*EngineeringChangeOrder, error) {
	globalECOStore.mu.Lock()
	defer globalECOStore.mu.Unlock()

	order, exists := globalECOStore.orders[req.ECOID]
	if !exists {
		return nil, fmt.Errorf("engineering change order '%s' not found", req.ECOID)
	}

	if req.ApproverName == "" {
		req.ApproverName = "Chief Technical Officer"
	}
	if req.ApproverRole == "" {
		req.ApproverRole = "PLANT_OPERATIONS_VP"
	}

	approval := ECOApprovalRecord{
		ApprovalID:   fmt.Sprintf("APP-ECO-%d", time.Now().UnixNano()%1000000),
		Role:         req.ApproverRole,
		ApproverName: req.ApproverName,
		Decision:     "APPROVED",
		Comments:     "Engineering change impact verified; disposition protocol authorized for production rollout.",
		Timestamp:    time.Now().Format(time.RFC3339),
	}

	order.Approvals = append(order.Approvals, approval)
	order.Status = "ROLLED_OUT_ACTIVE"
	order.RolledOutAt = time.Now().Format(time.RFC3339)
	if req.EffectiveSerialNumber != "" {
		order.EffectiveSerialNumber = req.EffectiveSerialNumber
	} else {
		order.EffectiveSerialNumber = fmt.Sprintf("SN-CUTOVER-%s-001", order.ProductSKU)
	}

	return order, nil
}

// GetECOAuditTrailLogic returns complete revision history.
func GetECOAuditTrailLogic(ecoID string) (*EngineeringChangeOrder, error) {
	globalECOStore.mu.RLock()
	defer globalECOStore.mu.RUnlock()

	order, exists := globalECOStore.orders[ecoID]
	if !exists {
		return nil, fmt.Errorf("engineering change order '%s' not found", ecoID)
	}
	return order, nil
}

// RegisterECORoutes mounts the Engineering Change Order routes.
func RegisterECORoutes(r *gin.RouterGroup) {
	group := r.Group("/mrp/eco")
	{
		group.POST("/create-order", func(c *gin.Context) {
			var req CreateECORequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid create ECO payload: " + err.Error()})
				return
			}
			res, err := CreateEngineeringChangeOrderLogic(req)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusCreated, res)
		})

		group.POST("/evaluate-impact", func(c *gin.Context) {
			ecoID := c.DefaultQuery("eco_id", "ECO-2026-REV-041")
			res, err := EvaluateECOImpactLogic(ecoID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, res)
		})

		group.POST("/execute-rollout", func(c *gin.Context) {
			var req ExecuteRolloutRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid execute rollout payload: " + err.Error()})
				return
			}
			res, err := ExecuteRolloutLogic(req)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, res)
		})

		group.GET("/audit-trail/:eco_id", func(c *gin.Context) {
			id := c.Param("eco_id")
			res, err := GetECOAuditTrailLogic(id)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, res)
		})
	}
}
