package mrp

import (
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// JobWorkOrder models the subcontracting toll manufacturing agreement
type JobWorkOrder struct {
	JobWorkID                   string    `json:"job_work_id"`
	VendorID                    string    `json:"vendor_id"`
	VendorName                  string    `json:"vendor_name"`
	FinishedProductSKU          string    `json:"finished_product_sku"`
	PlannedFinishedUnits        int       `json:"planned_finished_units"`
	BOMMaterialPerUnitKG        float64   `json:"bom_material_per_unit_kg"`      // e.g. 2.5 kg raw material per unit
	RawMaterialPricePerKGUSD    float64   `json:"raw_material_price_per_kg_usd"` // e.g. $12.50/kg
	AllowableScrapPct           float64   `json:"allowable_scrap_pct"`           // e.g. 2.0%
	ProcessingFeePerUnitUSD     float64   `json:"processing_fee_per_unit_usd"`   // e.g. $15.00/unit
	CustomsBonded180DayDeadline time.Time `json:"customs_bonded_180day_deadline"`
	Status                      string    `json:"status"` // "ORDER_ISSUED", "MATERIALS_DISPATCHED", "RECONCILED_CLOSED"
	CreatedAt                   time.Time `json:"created_at"`
}

// MaterialOutwardChallan tracks physical movement of raw materials to job-worker
type MaterialOutwardChallan struct {
	ChallanNumber            string    `json:"challan_number"`
	JobWorkID                string    `json:"job_work_id"`
	DispatchedMaterialKG     float64   `json:"dispatched_material_kg"`
	VehiclePlateNumber       string    `json:"vehicle_plate_number"`
	CustomsPassSlipReference string    `json:"customs_pass_slip_reference"`
	DispatchedAt             time.Time `json:"dispatched_at"`
}

// SubcontractingReconciliationResult models inward yield audit and excess scrap contra recovery
type SubcontractingReconciliationResult struct {
	ReconciliationID            string    `json:"reconciliation_id"`
	JobWorkID                   string    `json:"job_work_id"`
	ReceivedFinishedUnits       int       `json:"received_finished_units"`
	TheoreticalMaterialNeededKG float64   `json:"theoretical_material_needed_kg"`
	ReturnedScrapKG             float64   `json:"returned_scrap_kg"`
	AllowableScrapKG            float64   `json:"allowable_scrap_kg"`
	UnexplainedExcessScrapKG    float64   `json:"unexplained_excess_scrap_kg"`
	ExcessScrapPenaltyUSD       float64   `json:"excess_scrap_penalty_usd"`
	GrossProcessingFeeUSD       float64   `json:"gross_processing_fee_usd"`
	NetVendorPayableUSD         float64   `json:"net_vendor_payable_usd"`
	ContraDebitNoteReference    string    `json:"contra_debit_note_reference,omitempty"`
	CustomsBondedCompliant      bool      `json:"customs_bonded_compliant"`
	ReconciledAt                time.Time `json:"reconciled_at"`
}

// TollManufacturingVault thread-safe store
type TollManufacturingVault struct {
	mu         sync.RWMutex
	orders     map[string]*JobWorkOrder
	challans   map[string]MaterialOutwardChallan
	reconciles map[string]SubcontractingReconciliationResult
}

// Global Toll Manufacturing instance
var GlobalTollManufacturingVault = &TollManufacturingVault{
	orders:     make(map[string]*JobWorkOrder),
	challans:   make(map[string]MaterialOutwardChallan),
	reconciles: make(map[string]SubcontractingReconciliationResult),
}

// IssueJobWorkOrder creates toll manufacturing order
func (v *TollManufacturingVault) IssueJobWorkOrder(
	jobWorkID, vendorID, vendorName, sku string,
	units int, bomKG, rawPrice, scrapPct, feeUSD float64,
) *JobWorkOrder {
	v.mu.Lock()
	defer v.mu.Unlock()

	deadline := time.Now().AddDate(0, 6, 0) // 180 days customs bonded limit
	order := &JobWorkOrder{
		JobWorkID:                   jobWorkID,
		VendorID:                    vendorID,
		VendorName:                  vendorName,
		FinishedProductSKU:          sku,
		PlannedFinishedUnits:        units,
		BOMMaterialPerUnitKG:        bomKG,
		RawMaterialPricePerKGUSD:    rawPrice,
		AllowableScrapPct:           scrapPct,
		ProcessingFeePerUnitUSD:     feeUSD,
		CustomsBonded180DayDeadline: deadline,
		Status:                      "ORDER_ISSUED",
		CreatedAt:                   time.Now(),
	}

	v.orders[jobWorkID] = order
	return order
}

// DispatchRawMaterials generates outward challan
func (v *TollManufacturingVault) DispatchRawMaterials(
	jobWorkID, plate, customsRef string, dispatchedKG float64,
) (*MaterialOutwardChallan, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	order, ok := v.orders[jobWorkID]
	if !ok {
		return nil, fmt.Errorf("job work order not found: %s", jobWorkID)
	}

	order.Status = "MATERIALS_DISPATCHED"
	challanNo := fmt.Sprintf("CHL-OUT-%d", time.Now().UnixNano()%1000000)

	challan := MaterialOutwardChallan{
		ChallanNumber:            challanNo,
		JobWorkID:                jobWorkID,
		DispatchedMaterialKG:     dispatchedKG,
		VehiclePlateNumber:       plate,
		CustomsPassSlipReference: customsRef,
		DispatchedAt:             time.Now(),
	}

	v.challans[jobWorkID] = challan
	return &challan, nil
}

// ReconcileInwardScrap audits yield and deducts excess scrap penalties
func (v *TollManufacturingVault) ReconcileInwardScrap(
	jobWorkID string, receivedUnits int, returnedScrapKG float64,
) (*SubcontractingReconciliationResult, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	order, ok := v.orders[jobWorkID]
	if !ok {
		return nil, fmt.Errorf("job work order not found: %s", jobWorkID)
	}

	challan, hasChallan := v.challans[jobWorkID]
	if !hasChallan {
		return nil, fmt.Errorf("no material outward challan found for order: %s", jobWorkID)
	}

	theoreticalNeededKG := float64(receivedUnits) * order.BOMMaterialPerUnitKG
	allowableScrapKG := theoreticalNeededKG * (order.AllowableScrapPct / 100.0)

	// Material accounted for = theoretical + returned scrap
	materialAccountedFor := theoreticalNeededKG + returnedScrapKG
	// Material allowed = challan total - allowable scrap buffer
	unexplainedScrapKG := challan.DispatchedMaterialKG - materialAccountedFor - (allowableScrapKG - returnedScrapKG)
	if unexplainedScrapKG < 0 {
		unexplainedScrapKG = 0
	}
	unexplainedScrapKG = math.Round(unexplainedScrapKG*10) / 10

	penaltyUSD := unexplainedScrapKG * order.RawMaterialPricePerKGUSD
	penaltyUSD = math.Round(penaltyUSD*100) / 100

	grossFee := float64(receivedUnits) * order.ProcessingFeePerUnitUSD
	netPayable := grossFee - penaltyUSD

	var contraRef string
	if penaltyUSD > 0 {
		contraRef = fmt.Sprintf("DBN-SCRAP-CONTRA-%s-%d", jobWorkID, time.Now().Unix()%10000)
	}

	order.Status = "RECONCILED_CLOSED"
	customsOK := time.Now().Before(order.CustomsBonded180DayDeadline)

	reconID := fmt.Sprintf("REC-TOLL-%d", time.Now().UnixNano()%1000000)
	result := SubcontractingReconciliationResult{
		ReconciliationID:            reconID,
		JobWorkID:                   jobWorkID,
		ReceivedFinishedUnits:       receivedUnits,
		TheoreticalMaterialNeededKG: theoreticalNeededKG,
		ReturnedScrapKG:             returnedScrapKG,
		AllowableScrapKG:            allowableScrapKG,
		UnexplainedExcessScrapKG:    unexplainedScrapKG,
		ExcessScrapPenaltyUSD:       penaltyUSD,
		GrossProcessingFeeUSD:       grossFee,
		NetVendorPayableUSD:         netPayable,
		ContraDebitNoteReference:    contraRef,
		CustomsBondedCompliant:      customsOK,
		ReconciledAt:                time.Now(),
	}

	v.reconciles[jobWorkID] = result
	return &result, nil
}

// GetReconciliationSummary retrieves subcontracting audit dossier
func (v *TollManufacturingVault) GetReconciliationSummary(jobWorkID string) (gin.H, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	order, hasOrder := v.orders[jobWorkID]
	challan, hasChallan := v.challans[jobWorkID]
	recon, hasRecon := v.reconciles[jobWorkID]

	if !hasOrder {
		return nil, fmt.Errorf("no subcontracting records found for: %s", jobWorkID)
	}

	return gin.H{
		"job_work_id":             jobWorkID,
		"order":                   order,
		"outward_challan":         challan,
		"has_challan":             hasChallan,
		"reconciliation":          recon,
		"has_reconciliation":      hasRecon,
		"customs_bonded_facility": "BC 2.6 / 2.7 Subcontracting Permitted Within 180 Days",
	}, nil
}

// RegisterTollManufacturingRoutes registers HTTP endpoints
func RegisterTollManufacturingRoutes(r *gin.RouterGroup) {
	toll := r.Group("/mrp/subcontracting")
	{
		toll.POST("/issue-jobwork-order", func(c *gin.Context) {
			var req struct {
				JobWorkID                string  `json:"job_work_id" binding:"required"`
				VendorID                 string  `json:"vendor_id" binding:"required"`
				VendorName               string  `json:"vendor_name" binding:"required"`
				FinishedProductSKU       string  `json:"finished_product_sku" binding:"required"`
				PlannedFinishedUnits     int     `json:"planned_finished_units" binding:"required"`
				BOMMaterialPerUnitKG     float64 `json:"bom_material_per_unit_kg" binding:"required"`
				RawMaterialPricePerKGUSD float64 `json:"raw_material_price_per_kg_usd" binding:"required"`
				AllowableScrapPct        float64 `json:"allowable_scrap_pct" binding:"required"`
				ProcessingFeePerUnitUSD  float64 `json:"processing_fee_per_unit_usd" binding:"required"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			order := GlobalTollManufacturingVault.IssueJobWorkOrder(
				req.JobWorkID, req.VendorID, req.VendorName, req.FinishedProductSKU,
				req.PlannedFinishedUnits, req.BOMMaterialPerUnitKG, req.RawMaterialPricePerKGUSD,
				req.AllowableScrapPct, req.ProcessingFeePerUnitUSD,
			)
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": order})
		})

		toll.POST("/dispatch-raw-materials", func(c *gin.Context) {
			var req struct {
				JobWorkID                string  `json:"job_work_id" binding:"required"`
				VehiclePlateNumber       string  `json:"vehicle_plate_number" binding:"required"`
				CustomsPassSlipReference string  `json:"customs_pass_slip_reference" binding:"required"`
				DispatchedMaterialKG     float64 `json:"dispatched_material_kg" binding:"required"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			challan, err := GlobalTollManufacturingVault.DispatchRawMaterials(
				req.JobWorkID, req.VehiclePlateNumber, req.CustomsPassSlipReference, req.DispatchedMaterialKG,
			)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": challan})
		})

		toll.POST("/reconcile-inward-scrap", func(c *gin.Context) {
			var req struct {
				JobWorkID             string  `json:"job_work_id" binding:"required"`
				ReceivedFinishedUnits int     `json:"received_finished_units" binding:"required"`
				ReturnedScrapKG       float64 `json:"returned_scrap_kg"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			recon, err := GlobalTollManufacturingVault.ReconcileInwardScrap(
				req.JobWorkID, req.ReceivedFinishedUnits, req.ReturnedScrapKG,
			)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": recon})
		})

		toll.GET("/reconciliation-summary/:jobwork_id", func(c *gin.Context) {
			jobWorkID := c.Param("jobwork_id")
			summary, err := GlobalTollManufacturingVault.GetReconciliationSummary(jobWorkID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": summary})
		})
	}
}
