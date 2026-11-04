package scm

import (
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// NetworkNode represents an echelon facility in the supply chain (Central DC, Regional Hub, Retail Spoke).
type NetworkNode struct {
	NodeID                 string  `json:"node_id"`
	NodeName               string  `json:"node_name"`
	TierLevel              int     `json:"tier_level"` // 1 = CENTRAL_DC, 2 = REGIONAL_HUB, 3 = SPOKE_STORE
	LeadTimeDays           float64 `json:"lead_time_days"`
	DailyDemandMean        float64 `json:"daily_demand_mean"`
	DailyDemandStdDev      float64 `json:"daily_demand_std_dev"`
	HoldingCostPerUnitYear float64 `json:"holding_cost_per_unit_year"`
	UnitCostIDR            float64 `json:"unit_cost_idr"`
}

// MEIOOptimizationRequest holds nodes and target customer service levels.
type MEIOOptimizationRequest struct {
	NetworkID          string        `json:"network_id"`
	NetworkName        string        `json:"network_name"`
	TargetServiceLevel float64       `json:"target_service_level"` // e.g. 0.95 for 95%
	Nodes              []NetworkNode `json:"nodes"`
}

// NodeInventoryRecommendation holds optimized stocking parameters for a single node.
type NodeInventoryRecommendation struct {
	NodeID                     string  `json:"node_id"`
	NodeName                   string  `json:"node_name"`
	TierLevel                  int     `json:"tier_level"`
	SingleEchelonSafetyStock   float64 `json:"single_echelon_safety_stock"`
	MultiEchelonSafetyStock    float64 `json:"multi_echelon_safety_stock"`
	RecommendedReorderPoint    float64 `json:"recommended_reorder_point"`
	WorkingCapitalSavedIDR     float64 `json:"working_capital_saved_idr"`
	AnnualHoldingCostSavedIDR  float64 `json:"annual_holding_cost_saved_idr"`
	InventoryReductionPct      float64 `json:"inventory_reduction_pct"`
}

// MEIOOptimizationResult summarizes total network inventory reduction and working capital unlocked.
type MEIOOptimizationResult struct {
	NetworkID                  string                        `json:"network_id"`
	NetworkName                string                        `json:"network_name"`
	TargetServiceLevelPct      float64                       `json:"target_service_level_pct"`
	ZScoreApplied              float64                       `json:"z_score_applied"`
	TotalNodesOptimized        int                           `json:"total_nodes_optimized"`
	TotalDecoupledStockUnits   float64                       `json:"total_decoupled_stock_units"`
	TotalMEIOStockUnits        float64                       `json:"total_meio_stock_units"`
	NetUnitsSaved              float64                       `json:"net_units_saved"`
	TotalCapitalUnlockedIDR    float64                       `json:"total_capital_unlocked_idr"`
	TotalAnnualHoldingSavings  float64                       `json:"total_annual_holding_savings"`
	OverallInventorySavingsPct float64                       `json:"overall_inventory_savings_pct"`
	Recommendations            []NodeInventoryRecommendation `json:"recommendations"`
	ComputedAt                 string                        `json:"computed_at"`
}

// ServiceLevelTradeoffPoint models safety stock investment across varying service levels.
type ServiceLevelTradeoffPoint struct {
	ServiceLevelPct            float64 `json:"service_level_pct"`
	ZScore                     float64 `json:"z_score"`
	RequiredSafetyStockUnits   float64 `json:"required_safety_stock_units"`
	TotalCapitalRequirementIDR float64 `json:"total_capital_requirement_idr"`
	AnnualHoldingCostIDR       float64 `json:"annual_holding_cost_idr"`
}

var (
	meioMutex sync.RWMutex
	meioStore = make(map[string]*MEIOOptimizationResult)
)

// getZScore approximates normal inverse cumulative distribution for standard service levels.
func getZScore(sl float64) float64 {
	switch {
	case sl >= 0.999:
		return 3.090
	case sl >= 0.99:
		return 2.326
	case sl >= 0.98:
		return 2.054
	case sl >= 0.975:
		return 1.960
	case sl >= 0.95:
		return 1.645
	case sl >= 0.90:
		return 1.282
	case sl >= 0.85:
		return 1.036
	default:
		return 1.645 // Default to 95%
	}
}

// OptimizeNetworkInventoryLogic runs multi-echelon risk pooling and buffer positioning calculations.
func OptimizeNetworkInventoryLogic(req MEIOOptimizationRequest) MEIOOptimizationResult {
	if req.NetworkID == "" {
		req.NetworkID = fmt.Sprintf("NET-MEIO-%s-%03d", time.Now().Format("20060102"), time.Now().Unix()%1000)
	}
	if req.NetworkName == "" {
		req.NetworkName = "Java-Bali Multi-Tier Consumer Goods Distribution Network"
	}
	if req.TargetServiceLevel <= 0 {
		req.TargetServiceLevel = 0.95
	}
	z := getZScore(req.TargetServiceLevel)

	if len(req.Nodes) == 0 {
		// Provide default demo 3-tier network topology
		req.Nodes = []NetworkNode{
			{NodeID: "DC-CENTRAL-CKR", NodeName: "Central Mega DC Cikarang", TierLevel: 1, LeadTimeDays: 14.0, DailyDemandMean: 1200.0, DailyDemandStdDev: 240.0, HoldingCostPerUnitYear: 45000.0, UnitCostIDR: 250000.0},
			{NodeID: "HUB-REGIONAL-SBY", NodeName: "Regional Hub Surabaya", TierLevel: 2, LeadTimeDays: 5.0, DailyDemandMean: 500.0, DailyDemandStdDev: 130.0, HoldingCostPerUnitYear: 48000.0, UnitCostIDR: 250000.0},
			{NodeID: "HUB-REGIONAL-SMG", NodeName: "Regional Hub Semarang", TierLevel: 2, LeadTimeDays: 4.0, DailyDemandMean: 400.0, DailyDemandStdDev: 110.0, HoldingCostPerUnitYear: 48000.0, UnitCostIDR: 250000.0},
			{NodeID: "SPOKE-RETAIL-MLG", NodeName: "Spoke Retail Malang", TierLevel: 3, LeadTimeDays: 2.0, DailyDemandMean: 150.0, DailyDemandStdDev: 55.0, HoldingCostPerUnitYear: 52000.0, UnitCostIDR: 250000.0},
			{NodeID: "SPOKE-RETAIL-DPS", NodeName: "Spoke Retail Denpasar", TierLevel: 3, LeadTimeDays: 3.0, DailyDemandMean: 180.0, DailyDemandStdDev: 60.0, HoldingCostPerUnitYear: 55000.0, UnitCostIDR: 250000.0},
		}
	}

	var recs []NodeInventoryRecommendation
	var totalDecoupledUnits, totalMEIOUnits float64
	var totalCapitalUnlocked, totalAnnualHoldingSavings float64

	for _, node := range req.Nodes {
		lt := node.LeadTimeDays
		if lt <= 0 {
			lt = 1.0
		}
		// Classic decoupled single-echelon safety stock: SS = Z * sigma * sqrt(L)
		ssDecoupled := math.Round(z * node.DailyDemandStdDev * math.Sqrt(lt))

		// Multi-echelon pooling & risk diversification benefit factor:
		// Upstream echelons gain pooling efficiency from aggregated variances; downstream buffers are positioned smartly.
		var poolingFactor float64
		switch node.TierLevel {
		case 1: // Central DC: High pooling efficiency from diverse downstream demand
			poolingFactor = 0.68 // 32% reduction from centralization
		case 2: // Regional Hub: Moderate pooling
			poolingFactor = 0.78 // 22% reduction
		default: // Tier 3 Spoke: Localized buffer with high responsiveness
			poolingFactor = 0.86 // 14% reduction
		}

		ssMEIO := math.Round(ssDecoupled * poolingFactor)
		unitDiff := ssDecoupled - ssMEIO
		if unitDiff < 0 {
			unitDiff = 0
		}

		capSaved := unitDiff * node.UnitCostIDR
		holdingSaved := unitDiff * node.HoldingCostPerUnitYear
		pctReduction := 0.0
		if ssDecoupled > 0 {
			pctReduction = math.Round((unitDiff/ssDecoupled)*10000) / 100.0
		}

		rop := math.Round((node.DailyDemandMean * lt) + ssMEIO)

		rec := NodeInventoryRecommendation{
			NodeID:                    node.NodeID,
			NodeName:                  node.NodeName,
			TierLevel:                 node.TierLevel,
			SingleEchelonSafetyStock:  ssDecoupled,
			MultiEchelonSafetyStock:   ssMEIO,
			RecommendedReorderPoint:   rop,
			WorkingCapitalSavedIDR:    capSaved,
			AnnualHoldingCostSavedIDR: holdingSaved,
			InventoryReductionPct:     pctReduction,
		}
		recs = append(recs, rec)

		totalDecoupledUnits += ssDecoupled
		totalMEIOUnits += ssMEIO
		totalCapitalUnlocked += capSaved
		totalAnnualHoldingSavings += holdingSaved
	}

	overallPct := 0.0
	if totalDecoupledUnits > 0 {
		overallPct = math.Round(((totalDecoupledUnits-totalMEIOUnits)/totalDecoupledUnits)*10000) / 100.0
	}

	result := MEIOOptimizationResult{
		NetworkID:                  req.NetworkID,
		NetworkName:                req.NetworkName,
		TargetServiceLevelPct:      math.Round(req.TargetServiceLevel*10000) / 100.0,
		ZScoreApplied:              math.Round(z*1000) / 1000.0,
		TotalNodesOptimized:        len(req.Nodes),
		TotalDecoupledStockUnits:   totalDecoupledUnits,
		TotalMEIOStockUnits:        totalMEIOUnits,
		NetUnitsSaved:              totalDecoupledUnits - totalMEIOUnits,
		TotalCapitalUnlockedIDR:    totalCapitalUnlocked,
		TotalAnnualHoldingSavings:  totalAnnualHoldingSavings,
		OverallInventorySavingsPct: overallPct,
		Recommendations:            recs,
		ComputedAt:                 time.Now().Format(time.RFC3339),
	}

	meioMutex.Lock()
	meioStore[req.NetworkID] = &result
	meioMutex.Unlock()

	return result
}

// GetServiceLevelTradeoffLogic computes safety stock capital requirements across 90%, 95%, 98%, and 99% service levels.
func GetServiceLevelTradeoffLogic(unitCostIDR, holdingCostYear float64) []ServiceLevelTradeoffPoint {
	if unitCostIDR <= 0 {
		unitCostIDR = 250000.0
	}
	if holdingCostYear <= 0 {
		holdingCostYear = 48000.0
	}

	levels := []float64{0.90, 0.95, 0.98, 0.99}
	var curve []ServiceLevelTradeoffPoint

	baseUnits := 2500.0 // Base representative demand variance standard units
	for _, sl := range levels {
		z := getZScore(sl)
		units := math.Round(baseUnits * (z / 1.645))
		capReq := units * unitCostIDR
		holding := units * holdingCostYear

		curve = append(curve, ServiceLevelTradeoffPoint{
			ServiceLevelPct:            math.Round(sl * 100.0),
			ZScore:                     math.Round(z*1000) / 1000.0,
			RequiredSafetyStockUnits:   units,
			TotalCapitalRequirementIDR: capReq,
			AnnualHoldingCostIDR:       holding,
		})
	}
	return curve
}

// RegisterRoutes registers the MEIO routes.
func RegisterRoutes(r *gin.RouterGroup) {
	group := r.Group("/scm/meio")
	{
		group.POST("/optimize-network", func(c *gin.Context) {
			var req MEIOOptimizationRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				// Allow empty request to run with default topology
				req = MEIOOptimizationRequest{}
			}
			res := OptimizeNetworkInventoryLogic(req)
			c.JSON(http.StatusOK, res)
		})

		group.GET("/tradeoff-curve", func(c *gin.Context) {
			curve := GetServiceLevelTradeoffLogic(250000.0, 48000.0)
			c.JSON(http.StatusOK, gin.H{
				"tradeoff_curve": curve,
				"description":    "Exponential capital requirement curve as customer service level approaches 99.9%",
			})
		})
	}
}
