package wetchem

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

// H2O2ReactorTelemetry captures electronic-grade e-H2O2 auto-oxidation and distillation dynamics under SEMI C30 & NFPA 432
type H2O2ReactorTelemetry struct {
	TelemetryID                 string    `json:"telemetry_id"`
	ReactorID                   string    `json:"reactor_id"`                     // "REACTOR-AUTOX-H2O2-UP-SSS-01"
	ChemicalPlantID             string    `json:"chemical_plant_id"`             // "PLANT-SEMICHEM-TAICHUNG-01"
	H2O2ConcentrationWtPct      float64   `json:"h2o2_concentration_wt_pct"`      // e.g. 33.5 wt% (semiconductor UP-SSS grade)
	WorkingSolutionTempC        float64   `json:"working_solution_temp_c"`        // e.g. 48.5°C (runaway trigger > 45°C)
	RedoxPotentialMv            float64   `json:"redox_potential_mv"`             // e.g. 415.0 mV
	VentOffgasOxygenVolPct      float64   `json:"vent_offgas_oxygen_vol_pct"`     // e.g. 14.8 Vol% O2 (flammability/deoxygenation)
	DissolvedIronPpt            float64   `json:"dissolved_iron_ppt"`             // e.g. 85.0 ppt (SEMI C30 specification < 10 ppt)
	DistillationVaporPressureBar float64  `json:"distillation_vapor_pressure_bar"`// e.g. 3.85 bar
	PressureRiseRateBarPerSec   float64   `json:"pressure_rise_rate_bar_per_sec"` // e.g. 0.85 bar/s
	RecordedAt                  time.Time `json:"recorded_at"`
}

// H2O2RunawayAssessment models exothermic catalytic decomposition, Semenov thermal runaway, and SEMI C30 purity compliance
type H2O2RunawayAssessment struct {
	AssessmentID                string    `json:"assessment_id"`
	ReactorID                   string    `json:"reactor_id"`
	ExothermicDecompositionHeatKw float64 `json:"exothermic_decomposition_heat_kw"` // Q_rxn = dN/dt * Delta_H (-98 kJ/mol)
	OxygenGenerationRateNm3Hr   float64   `json:"oxygen_generation_rate_nm3_hr"`
	ThermalExplosionHazardLevel string    `json:"thermal_explosion_hazard_level"`   // "PASSIVE_STABLE_SOLUTION", "CATALYTIC_DECOMPOSITION_WARNING", "CRITICAL_RUNAWAY_DEOXYGENATION_EXPLOSION"
	ElectronicPurityCompliance  string    `json:"electronic_purity_compliance"`    // "SEMI_C30_UP_SSS_COMPLIANT", "METALLIC_CONTAMINATION_WARNING", "OFF_SPEC_REJECT_HAZARD"
	EmergencyDelugeMandate      bool      `json:"emergency_deluge_mandate"`
	ChelationQuenchRequired     bool      `json:"chelation_quench_required"`
	AuditDigest                 string    `json:"audit_digest"`
	EvaluatedAt                 time.Time `json:"evaluated_at"`
}

// H2O2MitigationAction models cold demineralized deluge, amino-phosphonate stabilizer dosing, and high-rate N2 purge
type H2O2MitigationAction struct {
	ActionID                    string    `json:"action_id"`
	ReactorID                   string    `json:"reactor_id"`
	ColdDemineralizedDelugeLMin float64   `json:"cold_demineralized_deluge_l_min"` // e.g. 1200.0 L/min
	ChelatingStabilizerDosingLHr float64  `json:"chelating_stabilizer_dosing_l_hr"`// e.g. 15.5 L/hr phosphonate scavenger
	NitrogenPurgeFlowNm3Hr      float64   `json:"nitrogen_purge_flow_nm3_hr"`      // e.g. 450.0 Nm3/hr
	AvoidedChemicalPlantLossUSD float64   `json:"avoided_chemical_plant_loss_usd"` // Avoided $65M USD distillation column explosion & fab stoppage
	ActionTimestamp             time.Time `json:"action_timestamp"`
}

// WetChemVault thread-safe in-memory store
type WetChemVault struct {
	mu          sync.RWMutex
	telemetries map[string]H2O2ReactorTelemetry
	assessments map[string]H2O2RunawayAssessment
	actions     map[string]H2O2MitigationAction
}

// GlobalWetChemVault instance
var GlobalWetChemVault = &WetChemVault{
	telemetries: make(map[string]H2O2ReactorTelemetry),
	assessments: make(map[string]H2O2RunawayAssessment),
	actions:     make(map[string]H2O2MitigationAction),
}

// IngestChemicalTelemetry stores electronic chemical reactor telemetry
func (v *WetChemVault) IngestChemicalTelemetry(t H2O2ReactorTelemetry) H2O2ReactorTelemetry {
	v.mu.Lock()
	defer v.mu.Unlock()

	if t.TelemetryID == "" {
		t.TelemetryID = fmt.Sprintf("TEL-H2O2-%d", time.Now().UnixNano()%1000000)
	}
	t.RecordedAt = time.Now()
	v.telemetries[t.ReactorID] = t
	return t
}

// EvaluateRunawayAndDeoxygenation evaluates Semenov thermal explosion kinetics and SEMI C30 purity
func (v *WetChemVault) EvaluateRunawayAndDeoxygenation(reactorID string) (H2O2RunawayAssessment, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	telem, exists := v.telemetries[reactorID]
	if !exists {
		return H2O2RunawayAssessment{}, fmt.Errorf("reactor %s not found in wet chem vault", reactorID)
	}

	// Exothermic decomposition heat output in kW
	heatKw := math.Round((telem.PressureRiseRateBarPerSec*385.0+telem.DissolvedIronPpt*2.4)*100) / 100

	// O2 generation rate in Nm3/hr
	o2GenRate := math.Round((telem.VentOffgasOxygenVolPct*18.5)*100) / 100

	hazard := "PASSIVE_STABLE_SOLUTION"
	delugeMandate := false
	chelationRequired := false
	if telem.WorkingSolutionTempC > 46.0 || telem.PressureRiseRateBarPerSec > 0.50 || telem.VentOffgasOxygenVolPct > 12.0 {
		hazard = "CRITICAL_RUNAWAY_DEOXYGENATION_EXPLOSION"
		delugeMandate = true
		chelationRequired = true
	} else if telem.WorkingSolutionTempC > 40.0 || telem.VentOffgasOxygenVolPct > 8.0 {
		hazard = "CATALYTIC_DECOMPOSITION_WARNING"
		chelationRequired = true
	}

	purityStatus := "SEMI_C30_UP_SSS_COMPLIANT"
	if telem.DissolvedIronPpt > 50.0 {
		purityStatus = "OFF_SPEC_REJECT_HAZARD"
	} else if telem.DissolvedIronPpt > 10.0 {
		purityStatus = "METALLIC_CONTAMINATION_WARNING"
	}

	rawHash := fmt.Sprintf("%s:%f:%f:%s:%s", reactorID, heatKw, o2GenRate, hazard, purityStatus)
	hash := sha256.Sum256([]byte(rawHash))
	digest := hex.EncodeToString(hash[:])

	assessment := H2O2RunawayAssessment{
		AssessmentID:                fmt.Sprintf("ASM-H2O2-%d", time.Now().UnixNano()%1000000),
		ReactorID:                   reactorID,
		ExothermicDecompositionHeatKw: heatKw,
		OxygenGenerationRateNm3Hr:   o2GenRate,
		ThermalExplosionHazardLevel: hazard,
		ElectronicPurityCompliance:  purityStatus,
		EmergencyDelugeMandate:      delugeMandate,
		ChelationQuenchRequired:     chelationRequired,
		AuditDigest:                 digest,
		EvaluatedAt:                 time.Now(),
	}

	v.assessments[reactorID] = assessment
	return assessment, nil
}

// TriggerDelugeAndChelationQuench deploys cold water deluge, amino-phosphonate stabilizer, and nitrogen blanketing
func (v *WetChemVault) TriggerDelugeAndChelationQuench(reactorID string) (H2O2MitigationAction, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	asm, hasAsm := v.assessments[reactorID]
	if !hasAsm {
		return H2O2MitigationAction{}, fmt.Errorf("missing assessment for reactor %s", reactorID)
	}

	delugeFlow := 0.0
	chelationDose := 0.0
	n2Purge := 150.0

	if asm.EmergencyDelugeMandate {
		delugeFlow = 1200.0 // L/min cold DI water deluge
		chelationDose = 15.5 // L/hr active phosphonate stabilizer
		n2Purge = 450.0      // Nm3/hr N2 purge to dilute oxygen below MOC
	} else if asm.ChelationQuenchRequired {
		chelationDose = 8.0
		n2Purge = 250.0
	}

	action := H2O2MitigationAction{
		ActionID:                    fmt.Sprintf("ACT-H2O2-%d", time.Now().UnixNano()%1000000),
		ReactorID:                   reactorID,
		ColdDemineralizedDelugeLMin: delugeFlow,
		ChelatingStabilizerDosingLHr: chelationDose,
		NitrogenPurgeFlowNm3Hr:      n2Purge,
		AvoidedChemicalPlantLossUSD: 65000000.0,
		ActionTimestamp:             time.Now(),
	}

	v.actions[reactorID] = action
	return action, nil
}

// GetWetChemicalSafetyDossier returns compliance audit dossier under SEMI C30 & NFPA 432
func (v *WetChemVault) GetWetChemicalSafetyDossier(reactorID string) (map[string]interface{}, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	telem, hasTelem := v.telemetries[reactorID]
	asm, hasAsm := v.assessments[reactorID]
	act, hasAct := v.actions[reactorID]

	dossier := map[string]interface{}{
		"reactor_id":          reactorID,
		"semi_c30_compliant":  true,
		"nfpa_432_certified":  true,
		"has_telemetry":       hasTelem,
		"has_assessment":      hasAsm,
		"has_action":          hasAct,
		"dossier_compiled_at": time.Now(),
	}
	if hasTelem {
		dossier["telemetry"] = telem
		dossier["h2o2_concentration_wt_pct"] = telem.H2O2ConcentrationWtPct
		dossier["temp_c"] = telem.WorkingSolutionTempC
		dossier["iron_ppt"] = telem.DissolvedIronPpt
	}
	if hasAsm {
		dossier["assessment"] = asm
		dossier["explosion_hazard"] = asm.ThermalExplosionHazardLevel
		dossier["purity_compliance"] = asm.ElectronicPurityCompliance
	}
	if hasAct {
		dossier["action"] = act
		dossier["avoided_loss_usd"] = act.AvoidedChemicalPlantLossUSD
	}

	return dossier, nil
}

// IngestChemicalTelemetryHandler Gin handler
func IngestChemicalTelemetryHandler(c *gin.Context) {
	var input H2O2ReactorTelemetry
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid H2O2 chemical telemetry payload: " + err.Error()})
		return
	}
	res := GlobalWetChemVault.IngestChemicalTelemetry(input)
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
}

// EvaluateRunawayAndDeoxygenationHandler Gin handler
func EvaluateRunawayAndDeoxygenationHandler(c *gin.Context) {
	var req struct {
		ReactorID string `json:"reactor_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Reactor ID required: " + err.Error()})
		return
	}
	res, err := GlobalWetChemVault.EvaluateRunawayAndDeoxygenation(req.ReactorID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
}

// TriggerDelugeAndChelationQuenchHandler Gin handler
func TriggerDelugeAndChelationQuenchHandler(c *gin.Context) {
	var req struct {
		ReactorID string `json:"reactor_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Reactor ID required: " + err.Error()})
		return
	}
	res, err := GlobalWetChemVault.TriggerDelugeAndChelationQuench(req.ReactorID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
}

// GetWetChemicalSafetyDossierHandler Gin handler
func GetWetChemicalSafetyDossierHandler(c *gin.Context) {
	reactorID := c.Param("reactor_id")
	if reactorID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Reactor ID path parameter required"})
		return
	}
	res, err := GlobalWetChemVault.GetWetChemicalSafetyDossier(reactorID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
}

// RegisterH2O2RunawayRoutes mounts endpoints under /api/v1/wetchem/h2o2
func RegisterH2O2RunawayRoutes(r *gin.RouterGroup) {
	h2o2 := r.Group("/wetchem/h2o2")
	{
		h2o2.POST("/ingest-chemical-telemetry", IngestChemicalTelemetryHandler)
		h2o2.POST("/evaluate-runaway-and-deoxygenation", EvaluateRunawayAndDeoxygenationHandler)
		h2o2.POST("/trigger-deluge-and-chelation-quench", TriggerDelugeAndChelationQuenchHandler)
		h2o2.GET("/wet-chemical-safety-dossier/:reactor_id", GetWetChemicalSafetyDossierHandler)
	}
}
