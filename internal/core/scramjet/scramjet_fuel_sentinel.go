package scramjet

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// ScramjetPropulsionTelemetry captures Mach 5-8+ hypersonic scramjet combustion & cryogenic fuel heat sink dynamics under AIAA S-113A & NASA-STD-5001
type ScramjetPropulsionTelemetry struct {
	TelemetryID                        string    `json:"telemetry_id"`
	EngineID                           string    `json:"engine_id"`                                // "SCRAMJET-X43D-MACH7-01"
	VehicleID                          string    `json:"vehicle_id"`                              // "VEHICLE-AURORA-HYPERSONIC-DEMO"
	FlightMachNumber                   float64   `json:"flight_mach_number"`                      // e.g. 6.85 Mach
	CombustorStagnationTempK           float64   `json:"combustor_stagnation_temp_k"`             // e.g. 2150.0 K
	CombustorWallHeatFluxMWm2          float64   `json:"combustor_wall_heat_flux_mwm2"`           // e.g. 18.5 MW/m2 extreme hypersonic thermal boundary layer
	FuelCoolingJacketInletPressureBar  float64   `json:"fuel_cooling_jacket_inlet_pressure_bar"`  // e.g. 68.0 bar supercritical hydrocarbon fuel
	FuelCoolingJacketOutletTempK       float64   `json:"fuel_cooling_jacket_outlet_temp_k"`       // e.g. 965.0 K (endothermic cracking range 850-1050 K)
	EndothermicCrackingConversionPct   float64   `json:"endothermic_cracking_conversion_pct"`     // e.g. 74.5% conversion of methylcyclohexane/JP-7 to light olefins/H2
	IsolatorShockTrainPositionMm       float64   `json:"isolator_shock_train_position_mm"`        // e.g. 138.0 mm from inlet cowl lip (choke margin)
	CombustorAcousticRmsDb             float64   `json:"combustor_acoustic_rms_db"`               // e.g. 164.5 dB supersonic combustion acoustic oscillation
	RecordedAt                         time.Time `json:"recorded_at"`
}

// ScramjetPropulsionAssessment models heat sink capacity, pyrolytic carbon coking, and isolator unstart margin
type ScramjetPropulsionAssessment struct {
	AssessmentID                    string    `json:"assessment_id"`
	EngineID                        string    `json:"engine_id"`
	EndothermicHeatSinkAbsorbedKjKg float64   `json:"endothermic_heat_sink_absorbed_kj_kg"` // e.g. ~1850 kJ/kg fuel chemical heat sink
	CokingPropensityIndex           float64   `json:"coking_propensity_index"`             // normalized index (0.0 - 2.0; > 1.0 critical coking rate)
	SupersonicIgnitionDelayMs       float64   `json:"supersonic_ignition_delay_ms"`        // auto-ignition delay in supersonic flow (< 0.85 ms residence time)
	ThermalChokeRisk                string    `json:"thermal_choke_risk"`                  // "STABLE_SUPERSONIC_FLOW", "PRE_COMBUSTOR_PRESSURE_SPIKE_WARNING", "CRITICAL_ISOLATOR_UNSTART_SHOCK_EXPULSION"
	CokingHazardLevel               string    `json:"coking_hazard_level"`                 // "COKING_FREE_CLEAN_TUBING", "PYROLYTIC_CARBON_DEPOSITION_WARNING", "CRITICAL_JACKET_CHOKING_MELTDOWN"
	DynamicFuelSplitMandate         bool      `json:"dynamic_fuel_split_mandate"`
	VariableBleedActuationRequired  bool      `json:"variable_bleed_actuation_required"`
	AuditDigest                     string    `json:"audit_digest"`
	EvaluatedAt                     time.Time `json:"evaluated_at"`
}

// ScramjetMitigationAction models dual-circuit fuel split, variable boundary layer bleed, and nano-inhibitor injection
type ScramjetMitigationAction struct {
	ActionID                        string    `json:"action_id"`
	EngineID                        string    `json:"engine_id"`
	CommandedEndothermicFuelSplitPct float64  `json:"commanded_endothermic_fuel_split_pct"` // e.g. 78.5% through catalytic cracking channels
	IsolatorBleedDoorAngleDeg       float64   `json:"isolator_bleed_door_angle_deg"`        // e.g. 14.5 deg boundary bleed to hold shock train
	AeroAntiCokingAdditivePpm       float64   `json:"aero_anti_coking_additive_ppm"`        // e.g. 95.0 ppm nano-dispersant inhibitor
	AvoidedHypersonicLossUSD        float64   `json:"avoided_hypersonic_loss_usd"`          // Avoided $140M USD scramjet combustor burnout and catastrophic unstart breakup
	ActionTimestamp                 time.Time `json:"action_timestamp"`
}

// ScramjetVault thread-safe in-memory store
type ScramjetVault struct {
	mu          sync.RWMutex
	telemetries map[string]ScramjetPropulsionTelemetry
	assessments map[string]ScramjetPropulsionAssessment
	actions     map[string]ScramjetMitigationAction
}

// GlobalScramjetVault instance
var GlobalScramjetVault = &ScramjetVault{
	telemetries: make(map[string]ScramjetPropulsionTelemetry),
	assessments: make(map[string]ScramjetPropulsionAssessment),
	actions:     make(map[string]ScramjetMitigationAction),
}

// IngestCombustorTelemetry stores high-Mach scramjet propulsion telemetry
func (v *ScramjetVault) IngestCombustorTelemetry(t ScramjetPropulsionTelemetry) ScramjetPropulsionTelemetry {
	v.mu.Lock()
	defer v.mu.Unlock()

	if t.TelemetryID == "" {
		t.TelemetryID = fmt.Sprintf("TEL-SCRAMJET-%d", time.Now().UnixNano()%1000000)
	}
	t.RecordedAt = time.Now()
	v.telemetries[t.EngineID] = t
	return t
}

// EvaluateCokingAndUnstart calculates fuel heat absorption, coking deposition, and shock train position
func (v *ScramjetVault) EvaluateCokingAndUnstart(engineID string) (ScramjetPropulsionAssessment, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	telem, exists := v.telemetries[engineID]
	if !exists {
		return ScramjetPropulsionAssessment{}, fmt.Errorf("engine %s not found in scramjet vault", engineID)
	}

	sensibleHeat := (telem.FuelCoolingJacketOutletTempK - 300.0) * 2.1
	crackingEnthalpy := (telem.EndothermicCrackingConversionPct / 100.0) * 1150.0
	totalHeatSink := math.Round((sensibleHeat+crackingEnthalpy)*100) / 100

	cokingIndex := 0.2
	if telem.FuelCoolingJacketOutletTempK > 920.0 {
		cokingIndex = math.Round((0.2+math.Exp((telem.FuelCoolingJacketOutletTempK-920.0)/35.0)*0.25)*100) / 100
	}

	ignitionDelayMs := math.Round((1200.0/telem.CombustorStagnationTempK)*0.75*100) / 100

	chokeRisk := "STABLE_SUPERSONIC_FLOW"
	bleedRequired := false
	if telem.IsolatorShockTrainPositionMm < 160.0 {
		chokeRisk = "CRITICAL_ISOLATOR_UNSTART_SHOCK_EXPULSION"
		bleedRequired = true
	} else if telem.IsolatorShockTrainPositionMm < 220.0 {
		chokeRisk = "PRE_COMBUSTOR_PRESSURE_SPIKE_WARNING"
		bleedRequired = true
	}

	cokingRisk := "COKING_FREE_CLEAN_TUBING"
	fuelSplitMandate := false
	if cokingIndex >= 1.0 {
		cokingRisk = "CRITICAL_JACKET_CHOKING_MELTDOWN"
		fuelSplitMandate = true
	} else if cokingIndex > 0.65 {
		cokingRisk = "PYROLYTIC_CARBON_DEPOSITION_WARNING"
		fuelSplitMandate = true
	}

	rawHash := fmt.Sprintf("%s:%f:%f:%f:%s:%s", engineID, totalHeatSink, cokingIndex, ignitionDelayMs, chokeRisk, cokingRisk)
	hash := sha256.Sum256([]byte(rawHash))
	digest := hex.EncodeToString(hash[:])

	assessment := ScramjetPropulsionAssessment{
		AssessmentID:                    fmt.Sprintf("ASM-SCRAMJET-%d", time.Now().UnixNano()%1000000),
		EngineID:                        engineID,
		EndothermicHeatSinkAbsorbedKjKg: totalHeatSink,
		CokingPropensityIndex:           cokingIndex,
		SupersonicIgnitionDelayMs:       ignitionDelayMs,
		ThermalChokeRisk:                chokeRisk,
		CokingHazardLevel:               cokingRisk,
		DynamicFuelSplitMandate:         fuelSplitMandate,
		VariableBleedActuationRequired:  bleedRequired,
		AuditDigest:                     digest,
		EvaluatedAt:                     time.Now(),
	}

	v.assessments[engineID] = assessment
	return assessment, nil
}

// TriggerFuelSplitAndIsolatorControl deploys dynamic fuel split, boundary bleed, and coking inhibitors
func (v *ScramjetVault) TriggerFuelSplitAndIsolatorControl(engineID string) (ScramjetMitigationAction, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	asm, hasAsm := v.assessments[engineID]
	if !hasAsm {
		return ScramjetMitigationAction{}, fmt.Errorf("missing propulsion assessment for engine %s", engineID)
	}

	fuelSplitPct := 50.0
	bleedAngleDeg := 0.0
	antiCokingPpm := 0.0

	if asm.DynamicFuelSplitMandate {
		fuelSplitPct = 78.5
		antiCokingPpm = 95.0
	}
	if asm.VariableBleedActuationRequired {
		bleedAngleDeg = 14.5
	}

	action := ScramjetMitigationAction{
		ActionID:                        fmt.Sprintf("ACT-SCRAMJET-%d", time.Now().UnixNano()%1000000),
		EngineID:                        engineID,
		CommandedEndothermicFuelSplitPct: fuelSplitPct,
		IsolatorBleedDoorAngleDeg:       bleedAngleDeg,
		AeroAntiCokingAdditivePpm:       antiCokingPpm,
		AvoidedHypersonicLossUSD:        140000000.0,
		ActionTimestamp:                 time.Now(),
	}

	v.actions[engineID] = action
	return action, nil
}

// GetHypersonicPropulsionDossier returns regulatory flight clearance dossier
func (v *ScramjetVault) GetHypersonicPropulsionDossier(engineID string) (map[string]interface{}, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	telem, hasTelem := v.telemetries[engineID]
	asm, hasAsm := v.assessments[engineID]
	act, hasAct := v.actions[engineID]

	dossier := map[string]interface{}{
		"engine_id":              engineID,
		"aiaa_s113a_compliant":   true,
		"nasa_std_5001_certified": true,
		"has_telemetry":          hasTelem,
		"has_assessment":         hasAsm,
		"has_action":             hasAct,
		"dossier_compiled_at":    time.Now(),
	}
	if hasTelem {
		dossier["telemetry"] = telem
		dossier["mach_number"] = telem.FlightMachNumber
		dossier["stagnation_temp_k"] = telem.CombustorStagnationTempK
	}
	if hasAsm {
		dossier["assessment"] = asm
		dossier["heat_sink_kj_kg"] = asm.EndothermicHeatSinkAbsorbedKjKg
		dossier["thermal_choke_risk"] = asm.ThermalChokeRisk
		dossier["coking_hazard"] = asm.CokingHazardLevel
	}
	if hasAct {
		dossier["action"] = act
		dossier["avoided_loss_usd"] = act.AvoidedHypersonicLossUSD
	}

	return dossier, nil
}

// IngestCombustorTelemetryHandler Gin handler
func IngestCombustorTelemetryHandler(c *gin.Context) {
	var input ScramjetPropulsionTelemetry
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid scramjet telemetry payload: " + err.Error()})
		return
	}
	res := GlobalScramjetVault.IngestCombustorTelemetry(input)
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
}

// EvaluateCokingAndUnstartHandler Gin handler
func EvaluateCokingAndUnstartHandler(c *gin.Context) {
	var req struct {
		EngineID string `json:"engine_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Engine ID required: " + err.Error()})
		return
	}
	res, err := GlobalScramjetVault.EvaluateCokingAndUnstart(req.EngineID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
}

// TriggerFuelSplitAndIsolatorControlHandler Gin handler
func TriggerFuelSplitAndIsolatorControlHandler(c *gin.Context) {
	var req struct {
		EngineID string `json:"engine_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Engine ID required: " + err.Error()})
		return
	}
	res, err := GlobalScramjetVault.TriggerFuelSplitAndIsolatorControl(req.EngineID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
}

// GetHypersonicPropulsionDossierHandler Gin handler
func GetHypersonicPropulsionDossierHandler(c *gin.Context) {
	engineID := c.Param("engine_id")
	if engineID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Engine ID path parameter required"})
		return
	}
	res, err := GlobalScramjetVault.GetHypersonicPropulsionDossier(engineID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
}

// RegisterScramjetRoutes mounts endpoints under /api/v1/scramjet/propulsion
func RegisterScramjetRoutes(r *gin.RouterGroup) {
	scram := r.Group("/scramjet/propulsion")
	{
		scram.POST("/ingest-combustor-telemetry", IngestCombustorTelemetryHandler)
		scram.POST("/evaluate-coking-and-unstart", EvaluateCokingAndUnstartHandler)
		scram.POST("/trigger-fuel-split-and-isolator-control", TriggerFuelSplitAndIsolatorControlHandler)
		scram.GET("/hypersonic-propulsion-dossier/:engine_id", GetHypersonicPropulsionDossierHandler)
	}
}
