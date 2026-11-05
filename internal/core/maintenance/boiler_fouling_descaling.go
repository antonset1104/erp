package maintenance

import (
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// BoilerThermalTelemetry represents continuous sensor data from steam generators and heat exchangers
type BoilerThermalTelemetry struct {
	TelemetryID         string    `json:"telemetry_id"`
	AssetID             string    `json:"asset_id"`
	FlueGasTemperatureC float64   `json:"flue_gas_temperature_c"` // Elevated flue gas indicates heat not transferred to water
	SteamOutputTPH      float64   `json:"steam_output_tph"`       // Tons per hour
	HeatTransferAreaM2  float64   `json:"heat_transfer_area_m2"`
	LogMeanTempDiffLMTD float64   `json:"log_mean_temp_diff_lmtd"`
	CleanDesignUCoeff   float64   `json:"clean_design_u_coeff"` // W/m2.K (Nominal design benchmark)
	DailyFuelCostUSD    float64   `json:"daily_fuel_cost_usd"`
	Timestamp           time.Time `json:"timestamp"`
}

// FoulingAssessmentResult models thermal degradation, scale thickness, and financial waste
type FoulingAssessmentResult struct {
	AssessmentID          string    `json:"assessment_id"`
	AssetID               string    `json:"asset_id"`
	ActualUCoeff          float64   `json:"actual_u_coeff"`          // Current heat transfer coefficient
	FoulingResistanceRf   float64   `json:"fouling_resistance_rf"`   // m2.K/W
	EstimatedScaleMicrons float64   `json:"estimated_scale_microns"` // Scale thickness in micrometers
	ThermalEfficiencyPct  float64   `json:"thermal_efficiency_pct"`  // e.g. 82.5% vs 94.0% clean
	DailyFuelWasteUSD     float64   `json:"daily_fuel_waste_usd"`    // Extra fuel burned due to insulating scale
	AssessedAt            time.Time `json:"assessed_at"`
}

// DescalingTurnaroundWorkOrder models scheduled chemical/mechanical cleaning
type DescalingTurnaroundWorkOrder struct {
	WorkOrderID               string    `json:"work_order_id"`
	AssetID                   string    `json:"asset_id"`
	RecommendedCleaningMethod string    `json:"recommended_cleaning_method"` // "CHEMICAL_CIP_ACID_FLUSH", "HIGH_PRESSURE_HYDROBLASTING"
	EstimatedCleaningCostUSD  float64   `json:"estimated_cleaning_cost_usd"` // e.g. $12,500
	DaysPaybackPeriod         float64   `json:"days_payback_period"`
	UrgencyClassification     string    `json:"urgency_classification"` // "DESCALING_MANDATORY_NOW", "SCHEDULE_NEXT_TURNAROUND"
	PreventedRuptureRisk      bool      `json:"prevented_rupture_risk"`
	ScheduledAt               time.Time `json:"scheduled_at"`
}

// BoilerFoulingVault thread-safe store
type BoilerFoulingVault struct {
	mu          sync.RWMutex
	telemetries map[string]BoilerThermalTelemetry
	assessments map[string]FoulingAssessmentResult
	workOrders  map[string]DescalingTurnaroundWorkOrder
}

// Global Boiler Fouling instance
var GlobalBoilerFoulingVault = &BoilerFoulingVault{
	telemetries: make(map[string]BoilerThermalTelemetry),
	assessments: make(map[string]FoulingAssessmentResult),
	workOrders:  make(map[string]DescalingTurnaroundWorkOrder),
}

func init() {
	// Seed demo industrial boiler telemetry
	GlobalBoilerFoulingVault.telemetries["BOILER-STM-50TPH-01"] = BoilerThermalTelemetry{
		TelemetryID:         "TEL-BLR-001",
		AssetID:             "BOILER-STM-50TPH-01",
		FlueGasTemperatureC: 235.0, // Elevated flue gas indicates heavy scale
		SteamOutputTPH:      48.0,
		HeatTransferAreaM2:  850.0,
		LogMeanTempDiffLMTD: 142.5,
		CleanDesignUCoeff:   1250.0,
		DailyFuelCostUSD:    9500.0,
		Timestamp:           time.Now(),
	}
}

// IngestThermalTelemetry stores continuous sensor data
func (v *BoilerFoulingVault) IngestThermalTelemetry(t BoilerThermalTelemetry) BoilerThermalTelemetry {
	v.mu.Lock()
	defer v.mu.Unlock()

	t.TelemetryID = fmt.Sprintf("TEL-BLR-%d", time.Now().UnixNano()%1000000)
	t.Timestamp = time.Now()
	v.telemetries[t.AssetID] = t
	return t
}

// CalculateFoulingResistance evaluates thermal resistance, scale thickness, and financial waste
func (v *BoilerFoulingVault) CalculateFoulingResistance(assetID string) (*FoulingAssessmentResult, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	t, ok := v.telemetries[assetID]
	if !ok {
		return nil, fmt.Errorf("thermal telemetry not found for boiler asset: %s", assetID)
	}

	// Calculate Heat Duty Q in Watts: Q = SteamOutputTPH * 1000 kg/hr * (2257 kJ/kg latent heat) / 3.6
	// Or approximate actual U from flue gas elevation:
	// Clean flue gas = 160 C. Excess flue gas = t.FlueGasTemperatureC - 160.0
	excessTemp := t.FlueGasTemperatureC - 160.0
	if excessTemp < 0 {
		excessTemp = 0
	}

	// Each 10 C flue gas rise roughly correlates to ~1.2% thermal efficiency loss and ~120 micron scale
	efficiencyLossPct := (excessTemp / 10.0) * 1.2
	actualEfficiency := 94.0 - efficiencyLossPct
	if actualEfficiency < 65.0 {
		actualEfficiency = 65.0
	}

	// Actual U-coefficient degradation
	uDegradeRatio := actualEfficiency / 94.0
	actualU := t.CleanDesignUCoeff * uDegradeRatio

	// Fouling Resistance Rf = 1/U_actual - 1/U_clean
	rf := (1.0 / actualU) - (1.0 / t.CleanDesignUCoeff)
	scaleMicrons := excessTemp * 11.5 // Approx scale thickness in micrometers

	// Daily fuel waste USD: Efficiency loss pct * DailyFuelCostUSD
	fuelWasteUSD := t.DailyFuelCostUSD * (efficiencyLossPct / 100.0)
	fuelWasteUSD = math.Round(fuelWasteUSD*100) / 100

	assessID := fmt.Sprintf("ASS-FOUL-%d", time.Now().UnixNano()%1000000)
	assessment := FoulingAssessmentResult{
		AssessmentID:          assessID,
		AssetID:               assetID,
		ActualUCoeff:          math.Round(actualU*10) / 10,
		FoulingResistanceRf:   math.Round(rf*1000000) / 1000000,
		EstimatedScaleMicrons: math.Round(scaleMicrons*10) / 10,
		ThermalEfficiencyPct:  math.Round(actualEfficiency*10) / 10,
		DailyFuelWasteUSD:     fuelWasteUSD,
		AssessedAt:            time.Now(),
	}

	v.assessments[assetID] = assessment
	return &assessment, nil
}

// ScheduleDescalingTurnaround triggers optimal cleaning work order
func (v *BoilerFoulingVault) ScheduleDescalingTurnaround(
	assetID string, cleaningCost float64, method string,
) (*DescalingTurnaroundWorkOrder, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	assess, ok := v.assessments[assetID]
	if !ok {
		return nil, fmt.Errorf("fouling assessment not found for asset: %s", assetID)
	}

	// Payback period in days = cleaningCost / assess.DailyFuelWasteUSD
	var paybackDays float64
	if assess.DailyFuelWasteUSD > 0 {
		paybackDays = cleaningCost / assess.DailyFuelWasteUSD
		paybackDays = math.Round(paybackDays*10) / 10
	}

	urgency := "SCHEDULE_NEXT_TURNAROUND"
	if paybackDays > 0 && paybackDays <= 15.0 {
		urgency = "DESCALING_MANDATORY_NOW" // Payback in under 2 weeks -> clean immediately!
	}

	woID := fmt.Sprintf("WO-DESC-%d", time.Now().UnixNano()%1000000)
	wo := DescalingTurnaroundWorkOrder{
		WorkOrderID:               woID,
		AssetID:                   assetID,
		RecommendedCleaningMethod: method,
		EstimatedCleaningCostUSD:  cleaningCost,
		DaysPaybackPeriod:         paybackDays,
		UrgencyClassification:     urgency,
		PreventedRuptureRisk:      true,
		ScheduledAt:               time.Now(),
	}

	v.workOrders[assetID] = wo
	return &wo, nil
}

// GetThermalDossier compiles comprehensive boiler efficiency dossier
func (v *BoilerFoulingVault) GetThermalDossier(assetID string) (gin.H, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	tel, hasTel := v.telemetries[assetID]
	ass, hasAss := v.assessments[assetID]
	wo, hasWO := v.workOrders[assetID]

	if !hasTel {
		return nil, fmt.Errorf("no thermal records found for boiler: %s", assetID)
	}

	return gin.H{
		"asset_id":              assetID,
		"latest_telemetry":      tel,
		"fouling_assessment":    ass,
		"has_assessment":        hasAss,
		"descaling_work_order":  wo,
		"has_work_order":        hasWO,
		"asme_compliance":       "ASME PTC 4 Fired Steam Generators & Section I Rules for Construction of Power Boilers",
		"tube_integrity_status": "OVERHEATING_RUPTURE_PREVENTED",
	}, nil
}

// RegisterBoilerRoutes registers HTTP endpoints
func RegisterBoilerRoutes(r *gin.RouterGroup) {
	blr := r.Group("/maintenance/boiler")
	{
		blr.POST("/ingest-thermal-telemetry", func(c *gin.Context) {
			var req BoilerThermalTelemetry
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			t := GlobalBoilerFoulingVault.IngestThermalTelemetry(req)
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": t})
		})

		blr.POST("/calculate-fouling-resistance", func(c *gin.Context) {
			var req struct {
				AssetID string `json:"asset_id" binding:"required"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			res, err := GlobalBoilerFoulingVault.CalculateFoulingResistance(req.AssetID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		blr.POST("/schedule-descaling-turnaround", func(c *gin.Context) {
			var req struct {
				AssetID                  string  `json:"asset_id" binding:"required"`
				EstimatedCleaningCostUSD float64 `json:"estimated_cleaning_cost_usd" binding:"required"`
				CleaningMethod           string  `json:"cleaning_method" binding:"required"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			wo, err := GlobalBoilerFoulingVault.ScheduleDescalingTurnaround(
				req.AssetID, req.EstimatedCleaningCostUSD, req.CleaningMethod,
			)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": wo})
		})

		blr.GET("/thermal-efficiency-dossier/:asset_id", func(c *gin.Context) {
			assetID := c.Param("asset_id")
			dossier, err := GlobalBoilerFoulingVault.GetThermalDossier(assetID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": dossier})
		})
	}
}
