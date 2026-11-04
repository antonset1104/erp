package cpq

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ConfigurationRule defines propositional logic constraints between product options.
type ConfigurationRule struct {
	RuleID          string `json:"rule_id"`
	Name            string `json:"name"`
	SourceFeatureID string `json:"source_feature_id"`
	TargetFeatureID string `json:"target_feature_id"`
	ConditionType   string `json:"condition_type"` // "REQUIRES", "EXCLUDES", "RECOMMENDS"
	ErrorMessage    string `json:"error_message"`
}

// SelectedFeature represents an option, module, or add-on selected by customer/sales.
type SelectedFeature struct {
	FeatureID  string  `json:"feature_id"`
	Name       string  `json:"name"`
	Category   string  `json:"category"` // e.g. "ENGINE", "TRANSMISSION", "SENSORS", "INTERIOR"
	Cost       float64 `json:"cost"`
	BasePrice  float64 `json:"base_price"`
	LaborHours float64 `json:"labor_hours"`
}

// ConfigureProductRequest holds input for rule evaluation and pricing calculation.
type ConfigureProductRequest struct {
	BaseProductID     string              `json:"base_product_id"`
	BaseProductName   string              `json:"base_product_name"`
	BaseCost          float64             `json:"base_cost"`
	BasePrice         float64             `json:"base_price"`
	TargetMarginFloor float64             `json:"target_margin_floor"` // e.g. 0.20 for 20%
	SelectedOptions   []SelectedFeature   `json:"selected_options"`
	CustomRules       []ConfigurationRule `json:"custom_rules"`
}

// ConfigureProductResult represents the validation and financial verdict of a configured product.
type ConfigureProductResult struct {
	Valid           bool     `json:"valid"`
	Violations      []string `json:"violations"`
	Warnings        []string `json:"warnings"`
	TotalCost       float64  `json:"total_cost"`
	TotalPrice      float64  `json:"total_price"`
	AchievedMargin  float64  `json:"achieved_margin"`
	MarginFloorMet  bool     `json:"margin_floor_met"`
	TotalLaborHours float64  `json:"total_labor_hours"`
	SummaryMessage  string   `json:"summary_message"`
}

// GenerateBOMRequest holds parameters to synthesize a multi-level Bill of Materials.
type GenerateBOMRequest struct {
	ConfigurationID string            `json:"configuration_id"`
	BaseProductID   string            `json:"base_product_id"`
	BatchSize       int               `json:"batch_size"`
	SelectedOptions []SelectedFeature `json:"selected_options"`
}

// BOMItem represents a component, subassembly, or labor operation in the synthesized BOM.
type BOMItem struct {
	ItemCode        string  `json:"item_code"`
	Description     string  `json:"description"`
	Category        string  `json:"category"` // "RAW_MATERIAL", "SUB_ASSEMBLY", "LABOR_ROUTING"
	QuantityPerUnit float64 `json:"quantity_per_unit"`
	TotalQuantity   float64 `json:"total_quantity"`
	UnitCost        float64 `json:"unit_cost"`
	ExtendedCost    float64 `json:"extended_cost"`
}

// GenerateBOMResult holds the synthesized production BOM.
type GenerateBOMResult struct {
	BOMID           string    `json:"bom_id"`
	BaseProductID   string    `json:"base_product_id"`
	BatchSize       int       `json:"batch_size"`
	Items           []BOMItem `json:"items"`
	TotalRollupCost float64   `json:"total_rollup_cost"`
	TotalLaborHours float64   `json:"total_labor_hours"`
	GeneratedAt     time.Time `json:"generated_at"`
}

// ConfigureProductLogic evaluates compatibility rules, checks margin floors, and computes totals.
func ConfigureProductLogic(req ConfigureProductRequest) ConfigureProductResult {
	selectedMap := make(map[string]bool)
	for _, opt := range req.SelectedOptions {
		selectedMap[opt.FeatureID] = true
	}

	var violations []string
	var warnings []string

	for _, rule := range req.CustomRules {
		srcPresent := selectedMap[rule.SourceFeatureID]
		tgtPresent := selectedMap[rule.TargetFeatureID]

		switch strings.ToUpper(rule.ConditionType) {
		case "REQUIRES":
			if srcPresent && !tgtPresent {
				msg := rule.ErrorMessage
				if msg == "" {
					msg = fmt.Sprintf("Feature '%s' requires feature '%s' to be selected.", rule.SourceFeatureID, rule.TargetFeatureID)
				}
				violations = append(violations, msg)
			}
		case "EXCLUDES":
			if srcPresent && tgtPresent {
				msg := rule.ErrorMessage
				if msg == "" {
					msg = fmt.Sprintf("Feature '%s' cannot be combined with feature '%s' (mutual exclusion).", rule.SourceFeatureID, rule.TargetFeatureID)
				}
				violations = append(violations, msg)
			}
		case "RECOMMENDS":
			if srcPresent && !tgtPresent {
				warnings = append(warnings, fmt.Sprintf("Recommended: Selecting '%s' performs best when paired with '%s'.", rule.SourceFeatureID, rule.TargetFeatureID))
			}
		}
	}

	totalCost := req.BaseCost
	totalPrice := req.BasePrice
	totalLaborHours := 0.0

	for _, opt := range req.SelectedOptions {
		totalCost += opt.Cost
		totalPrice += opt.BasePrice
		totalLaborHours += opt.LaborHours
	}

	var achievedMargin float64
	if totalPrice > 0 {
		achievedMargin = (totalPrice - totalCost) / totalPrice
	}

	marginFloorMet := true
	if req.TargetMarginFloor > 0 && achievedMargin < req.TargetMarginFloor {
		marginFloorMet = false
		violations = append(violations, fmt.Sprintf("Target margin floor of %.1f%% violated (Achieved: %.2f%%). Approval or price escalation required.", req.TargetMarginFloor*100, achievedMargin*100))
	}

	valid := len(violations) == 0

	var summary string
	if valid {
		summary = fmt.Sprintf("Configuration VALID. Total Price: $%.2f, Total Cost: $%.2f, Margin: %.2f%%, Labor: %.1f hrs.", totalPrice, totalCost, achievedMargin*100, totalLaborHours)
	} else {
		summary = fmt.Sprintf("Configuration INVALID with %d constraint violation(s).", len(violations))
	}

	return ConfigureProductResult{
		Valid:           valid,
		Violations:      violations,
		Warnings:        warnings,
		TotalCost:       totalCost,
		TotalPrice:      totalPrice,
		AchievedMargin:  achievedMargin,
		MarginFloorMet:  marginFloorMet,
		TotalLaborHours: totalLaborHours,
		SummaryMessage:  summary,
	}
}

// GenerateDynamicBOMLogic synthesizes production BOM lines and labor routing from configured options.
func GenerateDynamicBOMLogic(req GenerateBOMRequest) GenerateBOMResult {
	batch := req.BatchSize
	if batch <= 0 {
		batch = 1
	}

	var items []BOMItem
	var totalRollupCost float64
	var totalLabor float64

	// Base Chassis / Core Assembly
	baseItem := BOMItem{
		ItemCode:        req.BaseProductID + "-CHASSIS",
		Description:     "Core Frame and Master Mechanical Base Assembly",
		Category:        "SUB_ASSEMBLY",
		QuantityPerUnit: 1.0,
		TotalQuantity:   1.0 * float64(batch),
		UnitCost:        12500.0,
		ExtendedCost:    12500.0 * float64(batch),
	}
	items = append(items, baseItem)
	totalRollupCost += baseItem.ExtendedCost

	// Selected Options synthesis
	for _, opt := range req.SelectedOptions {
		item := BOMItem{
			ItemCode:        "MOD-" + opt.FeatureID,
			Description:     opt.Name + " (" + opt.Category + ")",
			Category:        "SUB_ASSEMBLY",
			QuantityPerUnit: 1.0,
			TotalQuantity:   1.0 * float64(batch),
			UnitCost:        opt.Cost,
			ExtendedCost:    opt.Cost * float64(batch),
		}
		items = append(items, item)
		totalRollupCost += item.ExtendedCost
		totalLabor += opt.LaborHours * float64(batch)
	}

	// Labor Routing Line (Standard manufacturing rate: $65/hr)
	laborRate := 65.0
	laborCost := totalLabor * laborRate
	laborItem := BOMItem{
		ItemCode:        "ROUT-ASSY-LABOR",
		Description:     fmt.Sprintf("Precision Robotic & Manual Assembly Routing (%.1f Total Hours)", totalLabor),
		Category:        "LABOR_ROUTING",
		QuantityPerUnit: totalLabor / float64(batch),
		TotalQuantity:   totalLabor,
		UnitCost:        laborRate,
		ExtendedCost:    laborCost,
	}
	items = append(items, laborItem)
	totalRollupCost += laborItem.ExtendedCost

	bomID := fmt.Sprintf("BOM-%s-%d", req.BaseProductID, time.Now().UnixNano()%100000)

	return GenerateBOMResult{
		BOMID:           bomID,
		BaseProductID:   req.BaseProductID,
		BatchSize:       batch,
		Items:           items,
		TotalRollupCost: totalRollupCost,
		TotalLaborHours: totalLabor,
		GeneratedAt:     time.Now().UTC(),
	}
}

// RegisterRoutes registers the CPQ engine routes.
func RegisterRoutes(r *gin.RouterGroup) {
	group := r.Group("/cpq")
	{
		group.POST("/configure-product", func(c *gin.Context) {
			var req ConfigureProductRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
				return
			}
			result := ConfigureProductLogic(req)
			c.JSON(http.StatusOK, result)
		})

		group.POST("/generate-bom", func(c *gin.Context) {
			var req GenerateBOMRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
				return
			}
			result := GenerateDynamicBOMLogic(req)
			c.JSON(http.StatusOK, result)
		})
	}
}
