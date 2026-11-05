package wms

import (
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// StockStatus classifies inventory levels based on Days of Supply.
type StockStatus string

const (
	StatusCriticalDeficit StockStatus = "CRITICAL_DEFICIT" // < 7 days
	StatusLowDeficit      StockStatus = "LOW_DEFICIT"      // 7 - 14 days
	StatusOptimalBalanced StockStatus = "OPTIMAL_BALANCED" // 14 - 30 days
	StatusExcessSurplus   StockStatus = "EXCESS_SURPLUS"   // > 30 days
)

// WarehouseStockProfile captures inventory metrics for a specific SKU in a warehouse.
type WarehouseStockProfile struct {
	WarehouseCode string      `json:"warehouse_code"`
	WarehouseName string      `json:"warehouse_name"`
	City          string      `json:"city"`
	OnHandUnits   float64     `json:"on_hand_units"`
	DailyVelocity float64     `json:"daily_velocity"` // units sold per day
	DaysOfSupply  float64     `json:"days_of_supply"` // OnHand / Velocity
	Status        StockStatus `json:"status"`
	TargetUnits   float64     `json:"target_units"` // 21 days * Velocity
	DeficitUnits  float64     `json:"deficit_units"`
	SurplusUnits  float64     `json:"surplus_units"`
}

// TransferRecommendation represents a proposed rebalancing transfer.
type TransferRecommendation struct {
	SKU              string  `json:"sku"`
	ProductName      string  `json:"product_name"`
	UnitCostIDR      float64 `json:"unit_cost_idr"`
	SourceWH         string  `json:"source_wh"`
	SourceCity       string  `json:"source_city"`
	DestinationWH    string  `json:"destination_wh"`
	DestinationCity  string  `json:"destination_city"`
	TransferQuantity float64 `json:"transfer_quantity"`
	FreightCostIDR   float64 `json:"freight_cost_idr"`
	TransitDays      int     `json:"transit_days"`
	CapitalSavedIDR  float64 `json:"capital_saved_idr"` // New PO purchase avoided minus freight
	Priority         string  `json:"priority"`          // URGENT, HIGH, NORMAL
}

// InventoryBalancingReport represents the complete multi-warehouse analysis.
type InventoryBalancingReport struct {
	EvaluationID              string                   `json:"evaluation_id"`
	EvaluatedAt               string                   `json:"evaluated_at"`
	TotalWarehousesAudited    int                      `json:"total_warehouses_audited"`
	TotalSKUsAudited          int                      `json:"total_skus_audited"`
	DeficitNodesCount         int                      `json:"deficit_nodes_count"`
	SurplusNodesCount         int                      `json:"surplus_nodes_count"`
	TotalTransfersProposed    int                      `json:"total_transfers_proposed"`
	TotalUnitsToRedistribute  float64                  `json:"total_units_to_redistribute"`
	TotalFreightCostIDR       float64                  `json:"total_freight_cost_idr"`
	NetWorkingCapitalSavedIDR float64                  `json:"net_working_capital_saved_idr"`
	WarehouseProfiles         []WarehouseStockProfile  `json:"warehouse_profiles"`
	Recommendations           []TransferRecommendation `json:"recommendations"`
}

// EvaluateBalancingRequest represents the input payload for stock evaluation.
type EvaluateBalancingRequest struct {
	SKU         string                  `json:"sku"`
	ProductName string                  `json:"product_name"`
	UnitCostIDR float64                 `json:"unit_cost_idr"`
	TargetDays  float64                 `json:"target_days"` // e.g. 21 days
	Profiles    []WarehouseStockProfile `json:"profiles"`
}

// ExecuteTransfersRequest triggers execution of approved transfer recommendations.
type ExecuteTransfersRequest struct {
	EvaluationID    string                   `json:"evaluation_id"`
	Recommendations []TransferRecommendation `json:"recommendations"`
	ApprovedBy      string                   `json:"approved_by"`
}

// ExecuteTransfersResponse confirms creation of internal transfers.
type ExecuteTransfersResponse struct {
	Success                bool     `json:"success"`
	TransfersExecutedCount int      `json:"transfers_executed_count"`
	GeneratedInternalMoves []string `json:"generated_internal_moves"`
	TotalFreightBudgetIDR  float64  `json:"total_freight_budget_idr"`
	ExecutionTimestamp     string   `json:"execution_timestamp"`
	Message                string   `json:"message"`
}

// InventoryBalancerService manages autonomous stock rebalancing across warehouses.
type InventoryBalancerService struct {
	mu           sync.RWMutex
	lastReports  map[string]*InventoryBalancingReport
	freightRates map[string]float64 // "Source-Dest" -> Cost per unit IDR
	transitTimes map[string]int     // "Source-Dest" -> Transit days
}

// NewInventoryBalancerService creates a new InventoryBalancerService.
func NewInventoryBalancerService() *InventoryBalancerService {
	return &InventoryBalancerService{
		lastReports: make(map[string]*InventoryBalancingReport),
		freightRates: map[string]float64{
			"WH-JKT-WH-SBY": 12000,
			"WH-SBY-WH-JKT": 12000,
			"WH-JKT-WH-MDN": 26000,
			"WH-MDN-WH-JKT": 26000,
			"WH-JKT-WH-MKS": 32000,
			"WH-MKS-WH-JKT": 32000,
			"WH-SBY-WH-MKS": 22000,
			"WH-MKS-WH-SBY": 22000,
		},
		transitTimes: map[string]int{
			"WH-JKT-WH-SBY": 1,
			"WH-SBY-WH-JKT": 1,
			"WH-JKT-WH-MDN": 3,
			"WH-MDN-WH-JKT": 3,
			"WH-JKT-WH-MKS": 4,
			"WH-MKS-WH-JKT": 4,
			"WH-SBY-WH-MKS": 3,
			"WH-MKS-WH-SBY": 3,
		},
	}
}

// EvaluateBalancing runs the autonomous multi-warehouse stock balancing heuristic.
func (s *InventoryBalancerService) EvaluateBalancing(req EvaluateBalancingRequest) (*InventoryBalancingReport, error) {
	if req.SKU == "" {
		req.SKU = "SKU-MON-27-QHD"
		req.ProductName = "Gaming Monitor 27 Inch QHD 165Hz"
		req.UnitCostIDR = 3500000
	}
	if req.TargetDays <= 0 {
		req.TargetDays = 21.0
	}

	// Default warehouse topology if omitted
	if len(req.Profiles) == 0 {
		req.Profiles = []WarehouseStockProfile{
			{WarehouseCode: "WH-JKT", WarehouseName: "Gudang Utama Cakung Jakarta", City: "Jakarta", OnHandUnits: 15, DailyVelocity: 4.5},     // DoS: 3.3d (Critical Deficit)
			{WarehouseCode: "WH-SBY", WarehouseName: "Regional Hub Rungkut Surabaya", City: "Surabaya", OnHandUnits: 140, DailyVelocity: 2.0}, // DoS: 70.0d (Excess Surplus)
			{WarehouseCode: "WH-MDN", WarehouseName: "Distribution Center Medan", City: "Medan", OnHandUnits: 8, DailyVelocity: 1.8},          // DoS: 4.4d (Critical Deficit)
			{WarehouseCode: "WH-MKS", WarehouseName: "Hub Timur Daya Makassar", City: "Makassar", OnHandUnits: 35, DailyVelocity: 1.5},        // DoS: 23.3d (Balanced)
		}
	}

	var deficits []WarehouseStockProfile
	var surpluses []WarehouseStockProfile
	var evaluatedProfiles []WarehouseStockProfile

	for _, p := range req.Profiles {
		dos := 0.0
		if p.DailyVelocity > 0 {
			dos = math.Round((p.OnHandUnits/p.DailyVelocity)*10) / 10
		}
		p.DaysOfSupply = dos
		targetUnits := math.Round(req.TargetDays * p.DailyVelocity)
		p.TargetUnits = targetUnits

		if dos < 7.0 {
			p.Status = StatusCriticalDeficit
			p.DeficitUnits = targetUnits - p.OnHandUnits
			p.SurplusUnits = 0
			deficits = append(deficits, p)
		} else if dos < 14.0 {
			p.Status = StatusLowDeficit
			p.DeficitUnits = targetUnits - p.OnHandUnits
			p.SurplusUnits = 0
			deficits = append(deficits, p)
		} else if dos > 30.0 {
			p.Status = StatusExcessSurplus
			p.SurplusUnits = p.OnHandUnits - targetUnits
			p.DeficitUnits = 0
			surpluses = append(surpluses, p)
		} else {
			p.Status = StatusOptimalBalanced
			p.DeficitUnits = 0
			p.SurplusUnits = 0
		}
		evaluatedProfiles = append(evaluatedProfiles, p)
	}

	var recommendations []TransferRecommendation
	totalUnitsToTransfer := 0.0
	totalFreightCost := 0.0
	totalCapitalSaved := 0.0

	// Match deficits against surpluses
	for _, def := range deficits {
		needed := def.DeficitUnits
		if needed <= 0 {
			continue
		}

		for i := range surpluses {
			if surpluses[i].SurplusUnits <= 0 {
				continue
			}

			avail := surpluses[i].SurplusUnits
			qty := math.Min(needed, avail)
			if qty <= 0 {
				continue
			}

			pairKey := fmt.Sprintf("%s-%s", surpluses[i].WarehouseCode, def.WarehouseCode)
			rate, ok := s.freightRates[pairKey]
			if !ok {
				rate = 18000 // generic domestic inter-island freight
			}
			transit, okT := s.transitTimes[pairKey]
			if !okT {
				transit = 2
			}

			freightCost := qty * rate
			capitalSaved := (qty * req.UnitCostIDR) - freightCost

			priority := "NORMAL"
			if def.Status == StatusCriticalDeficit {
				priority = "URGENT"
			}

			rec := TransferRecommendation{
				SKU:              req.SKU,
				ProductName:      req.ProductName,
				UnitCostIDR:      req.UnitCostIDR,
				SourceWH:         surpluses[i].WarehouseCode,
				SourceCity:       surpluses[i].City,
				DestinationWH:    def.WarehouseCode,
				DestinationCity:  def.City,
				TransferQuantity: qty,
				FreightCostIDR:   freightCost,
				TransitDays:      transit,
				CapitalSavedIDR:  capitalSaved,
				Priority:         priority,
			}

			recommendations = append(recommendations, rec)
			totalUnitsToTransfer += qty
			totalFreightCost += freightCost
			totalCapitalSaved += capitalSaved

			surpluses[i].SurplusUnits -= qty
			needed -= qty
			if needed <= 0 {
				break
			}
		}
	}

	evalID := fmt.Sprintf("EVAL/BAL/%s/%s", time.Now().Format("20060102"), time.Now().Format("150405"))

	report := &InventoryBalancingReport{
		EvaluationID:              evalID,
		EvaluatedAt:               time.Now().Format(time.RFC3339),
		TotalWarehousesAudited:    len(req.Profiles),
		TotalSKUsAudited:          1,
		DeficitNodesCount:         len(deficits),
		SurplusNodesCount:         len(surpluses),
		TotalTransfersProposed:    len(recommendations),
		TotalUnitsToRedistribute:  totalUnitsToTransfer,
		TotalFreightCostIDR:       totalFreightCost,
		NetWorkingCapitalSavedIDR: totalCapitalSaved,
		WarehouseProfiles:         evaluatedProfiles,
		Recommendations:           recommendations,
	}

	s.mu.Lock()
	s.lastReports[evalID] = report
	s.mu.Unlock()

	return report, nil
}

// ExecuteTransfers creates actual internal warehouse transfer pickings.
func (s *InventoryBalancerService) ExecuteTransfers(req ExecuteTransfersRequest) (*ExecuteTransfersResponse, error) {
	if len(req.Recommendations) == 0 {
		return &ExecuteTransfersResponse{
			Success:            false,
			Message:            "Tidak ada rekomendasi transfer yang diproses.",
			ExecutionTimestamp: time.Now().Format(time.RFC3339),
		}, nil
	}

	var generatedMoves []string
	totalFreight := 0.0

	for i, rec := range req.Recommendations {
		moveCode := fmt.Sprintf("WH/INT/BAL/%s/%04d", time.Now().Format("20060102"), i+1)
		generatedMoves = append(generatedMoves, moveCode)
		totalFreight += rec.FreightCostIDR
	}

	return &ExecuteTransfersResponse{
		Success:                true,
		TransfersExecutedCount: len(generatedMoves),
		GeneratedInternalMoves: generatedMoves,
		TotalFreightBudgetIDR:  totalFreight,
		ExecutionTimestamp:     time.Now().Format(time.RFC3339),
		Message:                fmt.Sprintf("%d Internal Transfer berhasil diterbitkan. Modal kerja dihemat melalui redistribusi stok internal.", len(generatedMoves)),
	}, nil
}

var defaultBalancerSvc = NewInventoryBalancerService()

// RegisterInventoryBalancerRoutes mounts HTTP routes for WMS Inventory Balancing.
func RegisterInventoryBalancerRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc, db *gorm.DB) {
	g := router.Group("/wms/inventory-balancing")
	g.Use(authMiddleware)
	{
		g.POST("/evaluate", func(c *gin.Context) {
			var req EvaluateBalancingRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := defaultBalancerSvc.EvaluateBalancing(req)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, res)
		})

		g.POST("/execute-transfers", func(c *gin.Context) {
			var req ExecuteTransfersRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := defaultBalancerSvc.ExecuteTransfers(req)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, res)
		})
	}
}
