package procurement

import (
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RawMaterialSupplier captures geopolitical and tariff parameters of raw material suppliers.
type RawMaterialSupplier struct {
	SupplierID             string  `json:"supplier_id"`
	SupplierName           string  `json:"supplier_name"`
	CountryCode            string  `json:"country_code"`
	GeopoliticalRiskIndex  float64 `json:"geopolitical_risk_index"` // 0.0 - 100.0 (Higher = more volatile/sanction-prone)
	CurrentTariffRatePct   float64 `json:"current_tariff_rate_pct"` // e.g. 25.0%
	UnitCostUSD            float64 `json:"unit_cost_usd"`
	LeadTimeDays           int     `json:"lead_time_days"`
	MaxMonthlyCapacityTons float64 `json:"max_monthly_capacity_tons"`
	SupplierTier           string  `json:"supplier_tier"` // "PRIMARY", "SECONDARY_NEARSHORE", "BACKUP"
}

// CriticalMaterialItem defines high-consequence raw materials requiring supply security.
type CriticalMaterialItem struct {
	MaterialID          string  `json:"material_id"` // e.g. "MAT-CRM-LITHIUM-01"
	MaterialName        string  `json:"material_name"`
	Category            string  `json:"category"` // "BATTERY_CATHODE", "SEMICONDUCTOR_SUBSTRATE", "MAGNET_ALLOY"
	MonthlyDemandTons   float64 `json:"monthly_demand_tons"`
	CriticalityScore    float64 `json:"criticality_score"` // 0.0 - 100.0
	PrimarySupplierID   string  `json:"primary_supplier_id"`
	SecondarySupplierID string  `json:"secondary_supplier_id"`
	ActiveSplitRatio    string  `json:"active_split_ratio"` // e.g. "70/30"
	EmbargoAlertActive  bool    `json:"embargo_alert_active"`
}

// EvaluateResilienceRequest holds input to evaluate supply security and tariff exposure.
type EvaluateResilienceRequest struct {
	MaterialID        string              `json:"material_id"`
	MaterialName      string              `json:"material_name"`
	MonthlyDemandTons float64             `json:"monthly_demand_tons"`
	PrimarySupplier   RawMaterialSupplier `json:"primary_supplier"`
	SecondarySupplier RawMaterialSupplier `json:"secondary_supplier"`
	TargetPrimaryPct  float64             `json:"target_primary_pct"` // e.g. 70.0 for 70%
}

// DualSourcingEvaluation represents the calculated resilience verdict and cost trade-off.
type DualSourcingEvaluation struct {
	MaterialID                 string              `json:"material_id"`
	MaterialName               string              `json:"material_name"`
	PrimarySupplier            RawMaterialSupplier `json:"primary_supplier"`
	SecondarySupplier          RawMaterialSupplier `json:"secondary_supplier"`
	RecommendedSplitRatio      string              `json:"recommended_split_ratio"`
	BlendedUnitCostUSD         float64             `json:"blended_unit_cost_usd"`
	GeopoliticalExposureScore  float64             `json:"geopolitical_exposure_score"` // Weighted 0-100
	SupplySecurityStatus       string              `json:"supply_security_status"`      // "RESILIENT_SECURED", "VULNERABLE_HIGH_RISK", "CRITICAL_ACTION_REQUIRED"
	EstimatedTariffExposureUSD float64             `json:"estimated_tariff_exposure_usd"`
	MitigationRecommendations  []string            `json:"mitigation_recommendations"`
	EvaluatedAt                string              `json:"evaluated_at"`
}

// SimulateSubstitutionRequest holds parameters to evaluate alternative material substitution.
type SimulateSubstitutionRequest struct {
	FinishedProductSKU  string  `json:"finished_product_sku"`
	OriginalMaterialID  string  `json:"original_material_id"`
	SubstituteMaterial  string  `json:"substitute_material"`
	MonthlyVolumeTons   float64 `json:"monthly_volume_tons"`
	SubstituteUnitCost  float64 `json:"substitute_unit_cost"`
	ReEngineeringCost   float64 `json:"re_engineering_cost"`
	QualWeeks           int     `json:"qual_weeks"`
	RegulatoryReapprove bool    `json:"regulatory_reapprove"`
}

// BOMSubstitutionSimulation represents engineering and financial impact of alternative materials.
type BOMSubstitutionSimulation struct {
	SimulationID             string   `json:"simulation_id"`
	FinishedProductSKU       string   `json:"finished_product_sku"`
	OriginalMaterialID       string   `json:"original_material_id"`
	SubstituteMaterial       string   `json:"substitute_material"`
	TechnicalFeasibilityPct  float64  `json:"technical_feasibility_pct"`
	AnnualCostDeltaUSD       float64  `json:"annual_cost_delta_usd"`
	PaybackMonths            float64  `json:"payback_months"`
	QualificationLeadTime    string   `json:"qualification_lead_time"`
	RegulatoryClearancePath  string   `json:"regulatory_clearance_path"`
	RiskReductionScore       float64  `json:"risk_reduction_score"`
	StrategicVerdict         string   `json:"strategic_verdict"`
	SimulatedAt              string   `json:"simulated_at"`
}

// CRMResilienceStore manages thread-safe storage for CRM resilience assets.
type CRMResilienceStore struct {
	mu           sync.RWMutex
	materials    map[string]*CriticalMaterialItem
	evaluations  map[string]*DualSourcingEvaluation
	simulations  map[string]*BOMSubstitutionSimulation
}

var globalCRMStore = &CRMResilienceStore{
	materials:   make(map[string]*CriticalMaterialItem),
	evaluations: make(map[string]*DualSourcingEvaluation),
	simulations: make(map[string]*BOMSubstitutionSimulation),
}

func init() {
	// Seed demo critical raw material
	demoMat := &CriticalMaterialItem{
		MaterialID:          "MAT-CRM-LITHIUM-01",
		MaterialName:        "Battery-Grade Lithium Hydroxide Monohydrate (56.5% LiOH)",
		Category:            "BATTERY_CATHODE",
		MonthlyDemandTons:   120.0,
		CriticalityScore:    92.5,
		PrimarySupplierID:   "VND-SICHUAN-LITHIUM",
		SecondarySupplierID: "VND-ALBEMARLE-CHILE",
		ActiveSplitRatio:    "70/30",
		EmbargoAlertActive:  false,
	}
	globalCRMStore.materials[demoMat.MaterialID] = demoMat
}

// EvaluateResilienceLogic performs geopolitical exposure modeling and dual-sourcing optimization.
func EvaluateResilienceLogic(req EvaluateResilienceRequest) (*DualSourcingEvaluation, error) {
	globalCRMStore.mu.Lock()
	defer globalCRMStore.mu.Unlock()

	if req.MaterialID == "" {
		req.MaterialID = "MAT-CRM-GENERIC"
	}
	if req.MonthlyDemandTons <= 0 {
		req.MonthlyDemandTons = 100.0
	}
	if req.TargetPrimaryPct <= 0 || req.TargetPrimaryPct > 100 {
		req.TargetPrimaryPct = 70.0
	}

	primaryWeight := req.TargetPrimaryPct / 100.0
	secondaryWeight := 1.0 - primaryWeight

	// Calculate blended unit cost
	blendedCost := (req.PrimarySupplier.UnitCostUSD * primaryWeight) + (req.SecondarySupplier.UnitCostUSD * secondaryWeight)

	// Calculate weighted geopolitical risk exposure
	blendedRisk := (req.PrimarySupplier.GeopoliticalRiskIndex * primaryWeight) + (req.SecondarySupplier.GeopoliticalRiskIndex * secondaryWeight)

	// Calculate tariff exposure
	primaryTariff := (req.PrimarySupplier.UnitCostUSD * (req.PrimarySupplier.CurrentTariffRatePct / 100.0)) * (req.MonthlyDemandTons * primaryWeight)
	secondaryTariff := (req.SecondarySupplier.UnitCostUSD * (req.SecondarySupplier.CurrentTariffRatePct / 100.0)) * (req.MonthlyDemandTons * secondaryWeight)
	monthlyTariffUSD := primaryTariff + secondaryTariff

	status := "RESILIENT_SECURED"
	var recs []string

	if blendedRisk >= 70.0 || req.PrimarySupplier.GeopoliticalRiskIndex >= 80.0 {
		status = "CRITICAL_ACTION_REQUIRED"
		recs = append(recs, "Primary origin country geopolitical risk index exceeds 80; accelerate nearshore secondary volume shift to 50/50.")
		recs = append(recs, "Initiate strategic safety stock buffer accumulation (+60 days demand in bonded customs zone).")
	} else if blendedRisk >= 45.0 {
		status = "VULNERABLE_HIGH_RISK"
		recs = append(recs, "Maintain active 70/30 dual-sourcing contract to keep secondary supplier qualified and warm.")
		recs = append(recs, "Monitor bilateral trade sanction lists for emerging tariff increases.")
	} else {
		recs = append(recs, "Supply routes well-diversified with low geopolitical friction; continue annual supplier performance reviews.")
	}

	eval := &DualSourcingEvaluation{
		MaterialID:                 req.MaterialID,
		MaterialName:               req.MaterialName,
		PrimarySupplier:            req.PrimarySupplier,
		SecondarySupplier:          req.SecondarySupplier,
		RecommendedSplitRatio:      fmt.Sprintf("%.0f/%.0f", req.TargetPrimaryPct, 100.0-req.TargetPrimaryPct),
		BlendedUnitCostUSD:         math.Round(blendedCost*100) / 100,
		GeopoliticalExposureScore:  math.Round(blendedRisk*100) / 100,
		SupplySecurityStatus:       status,
		EstimatedTariffExposureUSD: math.Round(monthlyTariffUSD*100) / 100,
		MitigationRecommendations:  recs,
		EvaluatedAt:                time.Now().Format(time.RFC3339),
	}

	globalCRMStore.evaluations[req.MaterialID] = eval
	return eval, nil
}

// SimulateSubstitutionLogic computes engineering feasibility and payback for alternative materials.
func SimulateSubstitutionLogic(req SimulateSubstitutionRequest) (*BOMSubstitutionSimulation, error) {
	globalCRMStore.mu.Lock()
	defer globalCRMStore.mu.Unlock()

	if req.FinishedProductSKU == "" {
		req.FinishedProductSKU = "PRD-EV-BATTERY-PACK"
	}
	if req.MonthlyVolumeTons <= 0 {
		req.MonthlyVolumeTons = 100.0
	}
	if req.QualWeeks <= 0 {
		req.QualWeeks = 16
	}

	annualVolume := req.MonthlyVolumeTons * 12.0
	annualCostDelta := (req.SubstituteUnitCost * annualVolume) - (25000.0 * annualVolume) // Benchmark $25k/ton

	paybackMonths := 0.0
	if annualCostDelta < 0 && req.ReEngineeringCost > 0 {
		monthlySavings := math.Abs(annualCostDelta) / 12.0
		paybackMonths = req.ReEngineeringCost / monthlySavings
	}

	simID := fmt.Sprintf("SIM-SUB-%d", time.Now().UnixNano()%1000000)
	clearance := "Internal Engineering Sign-Off (Direct Drop-in Replacement)"
	if req.RegulatoryReapprove {
		clearance = "Full OEM Homologation & UN 38.3 Transport Certification Required (6 Months Lead Time)"
	}

	verdict := "FEASIBLE_RECOMMENDED"
	if paybackMonths > 36.0 || req.QualWeeks > 24 {
		verdict = "STRATEGIC_CONTINGENCY_ONLY"
	}

	sim := &BOMSubstitutionSimulation{
		SimulationID:            simID,
		FinishedProductSKU:      req.FinishedProductSKU,
		OriginalMaterialID:      req.OriginalMaterialID,
		SubstituteMaterial:      req.SubstituteMaterial,
		TechnicalFeasibilityPct: 92.5,
		AnnualCostDeltaUSD:      math.Round(annualCostDelta*100) / 100,
		PaybackMonths:           math.Round(paybackMonths*10) / 10,
		QualificationLeadTime:   fmt.Sprintf("%d Weeks", req.QualWeeks),
		RegulatoryClearancePath: clearance,
		RiskReductionScore:      45.0,
		StrategicVerdict:        verdict,
		SimulatedAt:             time.Now().Format(time.RFC3339),
	}

	globalCRMStore.simulations[simID] = sim
	return sim, nil
}

// RegisterCRMResilienceRoutes mounts the procurement CRM resilience endpoints.
func RegisterCRMResilienceRoutes(r *gin.RouterGroup) {
	group := r.Group("/procurement/crm")
	{
		group.POST("/evaluate-resilience", func(c *gin.Context) {
			var req EvaluateResilienceRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid evaluate resilience payload: " + err.Error()})
				return
			}
			res, err := EvaluateResilienceLogic(req)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, res)
		})

		group.POST("/simulate-substitution", func(c *gin.Context) {
			var req SimulateSubstitutionRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid simulate substitution payload: " + err.Error()})
				return
			}
			res, err := SimulateSubstitutionLogic(req)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, res)
		})

		group.GET("/material-radar", func(c *gin.Context) {
			globalCRMStore.mu.RLock()
			defer globalCRMStore.mu.RUnlock()

			mats := make([]*CriticalMaterialItem, 0, len(globalCRMStore.materials))
			for _, m := range globalCRMStore.materials {
				mats = append(mats, m)
			}
			evals := make([]*DualSourcingEvaluation, 0, len(globalCRMStore.evaluations))
			for _, e := range globalCRMStore.evaluations {
				evals = append(evals, e)
			}
			c.JSON(http.StatusOK, gin.H{
				"total_monitored_critical_materials": len(mats),
				"materials":                          mats,
				"active_evaluations":                 evals,
			})
		})
	}
}
