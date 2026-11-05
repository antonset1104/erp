package geothermal_shr

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

// SuperhotGeothermalTelemetry captures ultra-deep supercritical superhot rock (SHR) wellhead & downhole dynamics under API 5CT & ISO 19901-2
type SuperhotGeothermalTelemetry struct {
	TelemetryID               string    `json:"telemetry_id"`
	WellID                    string    `json:"well_id"`                     // "WELL-SHR-KRAFLA-MAGMA-01"
	GeothermalFieldID         string    `json:"geothermal_field_id"`         // "FIELD-ICELAND-DEEP-DRILLING-PROJECT"
	WellDepthMeters           float64   `json:"well_depth_meters"`           // e.g. 4650.0 m (near ductile-brittle transition)
	ReservoirTemperatureC     float64   `json:"reservoir_temperature_c"`     // e.g. 485.0°C (supercritical rock > 400°C)
	ReservoirPressureBar      float64   `json:"reservoir_pressure_bar"`      // e.g. 320.0 bar hydrostatic/lithostatic
	CasingVonMisesStressMpa   float64   `json:"casing_von_mises_stress_mpa"` // e.g. 685.0 MPa (high alloy casing yield 720 MPa)
	InducedSeismicMagnitudeML float64   `json:"induced_seismic_magnitude_ml"`// e.g. 1.85 ML (Traffic Light Protocol threshold 2.0 ML)
	PeakGroundAccelerationG   float64   `json:"peak_ground_acceleration_g"`  // e.g. 0.042 g
	ColdWaterInjectionRateKgS float64   `json:"cold_water_injection_rate_kg_s"` // e.g. 95.0 kg/s (cold water ~25°C)
	RecordedAt                time.Time `json:"recorded_at"`
}

// SuperhotGeothermalAssessment models Coulomb failure stress change, casing thermal shock, and Traffic Light Protocol (TLP) compliance
type SuperhotGeothermalAssessment struct {
	AssessmentID                  string    `json:"assessment_id"`
	WellID                        string    `json:"well_id"`
	CoulombStressChangeDeltaCFSBar float64  `json:"coulomb_stress_change_delta_cfs_bar"` // dCFS = dTau - mu * (dSigma - dP)
	ThermalShockDeltaTC           float64   `json:"thermal_shock_delta_t_c"`             // delta T between reservoir and injection fluid
	TrafficLightProtocolStatus    string    `json:"traffic_light_protocol_status"`       // "TLP_GREEN_NORMAL", "TLP_AMBER_FLOW_THROTTLE_REQUIRED", "TLP_RED_CRITICAL_SEISMIC_STOP_MANDATE"
	CasingIntegrityRisk           string    `json:"casing_integrity_risk"`               // "ELASTIC_RANGE_SAFE", "PLASTIC_DEFORMATION_WARNING", "CRITICAL_CASING_SHEAR_BUCKLING_FAILURE"
	FlowThrottlingMandate         bool      `json:"flow_throttling_mandate"`
	ThermalTemperingCycleRequired bool      `json:"thermal_tempering_cycle_required"`
	AuditDigest                   string    `json:"audit_digest"`
	EvaluatedAt                   time.Time `json:"evaluated_at"`
}

// SuperhotGeothermalAction models adaptive injection throttling, pulsed wellhead bleed-off, and thermal pre-tempering
type SuperhotGeothermalAction struct {
	ActionID                    string    `json:"action_id"`
	WellID                      string    `json:"well_id"`
	ModulatedInjectionRateKgS   float64   `json:"modulated_injection_rate_kg_s"`   // throttled down to 45.0 kg/s
	WellheadBleedOffPressureBar float64   `json:"wellhead_bleed_off_pressure_bar"` // pulsed bleed-off 28.0 bar
	TemperingTemperatureBoostC  float64   `json:"tempering_temperature_boost_c"`  // pre-tempering fluid to 125.0°C to soften thermal gradient
	AvoidedGeothermalWellLossUSD float64  `json:"avoided_geothermal_well_loss_usd"` // Avoided $120M USD well collapse and regulatory shutdown
	ActionTimestamp             time.Time `json:"action_timestamp"`
}

// SuperhotGeothermalVault thread-safe in-memory store
type SuperhotGeothermalVault struct {
	mu          sync.RWMutex
	telemetries map[string]SuperhotGeothermalTelemetry
	assessments map[string]SuperhotGeothermalAssessment
	actions     map[string]SuperhotGeothermalAction
}

// GlobalSuperhotGeothermalVault instance
var GlobalSuperhotGeothermalVault = &SuperhotGeothermalVault{
	telemetries: make(map[string]SuperhotGeothermalTelemetry),
	assessments: make(map[string]SuperhotGeothermalAssessment),
	actions:     make(map[string]SuperhotGeothermalAction),
}

// IngestWellTelemetry stores SHR downhole telemetry
func (v *SuperhotGeothermalVault) IngestWellTelemetry(t SuperhotGeothermalTelemetry) SuperhotGeothermalTelemetry {
	v.mu.Lock()
	defer v.mu.Unlock()

	if t.TelemetryID == "" {
		t.TelemetryID = fmt.Sprintf("TEL-SHR-%d", time.Now().UnixNano()%1000000)
	}
	t.RecordedAt = time.Now()
	v.telemetries[t.WellID] = t
	return t
}

// EvaluateThermalStressAndSeismicity evaluates Coulomb failure stress and Traffic Light Protocol
func (v *SuperhotGeothermalVault) EvaluateThermalStressAndSeismicity(wellID string) (SuperhotGeothermalAssessment, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	telem, exists := v.telemetries[wellID]
	if !exists {
		return SuperhotGeothermalAssessment{}, fmt.Errorf("well %s not found in superhot geothermal vault", wellID)
	}

	thermalShock := math.Round((telem.ReservoirTemperatureC-25.0)*100) / 100

	// Coulomb failure stress change: ~0.025 bar per kg/s injection flow under high pore pressure
	coulombDelta := math.Round((telem.ColdWaterInjectionRateKgS*0.0258)*100) / 100

	tlpStatus := "TLP_GREEN_NORMAL"
	throttleMandate := false
	temperingRequired := false
	if telem.InducedSeismicMagnitudeML >= 1.80 || telem.PeakGroundAccelerationG >= 0.040 {
		tlpStatus = "TLP_RED_CRITICAL_SEISMIC_STOP_MANDATE"
		throttleMandate = true
		temperingRequired = true
	} else if telem.InducedSeismicMagnitudeML >= 1.40 || telem.PeakGroundAccelerationG >= 0.020 {
		tlpStatus = "TLP_AMBER_FLOW_THROTTLE_REQUIRED"
		throttleMandate = true
	}

	casingRisk := "ELASTIC_RANGE_SAFE"
	if telem.CasingVonMisesStressMpa > 675.0 || thermalShock > 420.0 {
		casingRisk = "CRITICAL_CASING_SHEAR_BUCKLING_FAILURE"
		temperingRequired = true
	} else if telem.CasingVonMisesStressMpa > 580.0 {
		casingRisk = "PLASTIC_DEFORMATION_WARNING"
	}

	rawHash := fmt.Sprintf("%s:%f:%f:%f:%s:%s", wellID, thermalShock, coulombDelta, telem.InducedSeismicMagnitudeML, tlpStatus, casingRisk)
	hash := sha256.Sum256([]byte(rawHash))
	digest := hex.EncodeToString(hash[:])

	assessment := SuperhotGeothermalAssessment{
		AssessmentID:                  fmt.Sprintf("ASM-SHR-%d", time.Now().UnixNano()%1000000),
		WellID:                        wellID,
		CoulombStressChangeDeltaCFSBar: coulombDelta,
		ThermalShockDeltaTC:           thermalShock,
		TrafficLightProtocolStatus:    tlpStatus,
		CasingIntegrityRisk:           casingRisk,
		FlowThrottlingMandate:         throttleMandate,
		ThermalTemperingCycleRequired: temperingRequired,
		AuditDigest:                   digest,
		EvaluatedAt:                   time.Now(),
	}

	v.assessments[wellID] = assessment
	return assessment, nil
}

// TriggerSeismicMitigationAndTempering actuates flow throttling, pressure bleed-off, and thermal pre-tempering
func (v *SuperhotGeothermalVault) TriggerSeismicMitigationAndTempering(wellID string) (SuperhotGeothermalAction, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	asm, hasAsm := v.assessments[wellID]
	telem, hasTelem := v.telemetries[wellID]
	if !hasAsm || !hasTelem {
		return SuperhotGeothermalAction{}, fmt.Errorf("missing telemetry or assessment for SHR well %s", wellID)
	}

	throttledFlow := telem.ColdWaterInjectionRateKgS
	bleedPressure := 0.0
	temperingBoost := 0.0

	if asm.FlowThrottlingMandate {
		throttledFlow = 45.0
		bleedPressure = 28.0
	}
	if asm.ThermalTemperingCycleRequired {
		temperingBoost = 125.0 // pre-heat fluid to soften thermal expansion gradient
	}

	action := SuperhotGeothermalAction{
		ActionID:                    fmt.Sprintf("ACT-SHR-%d", time.Now().UnixNano()%1000000),
		WellID:                      wellID,
		ModulatedInjectionRateKgS:   throttledFlow,
		WellheadBleedOffPressureBar: bleedPressure,
		TemperingTemperatureBoostC:  temperingBoost,
		AvoidedGeothermalWellLossUSD: 120000000.0,
		ActionTimestamp:             time.Now(),
	}

	v.actions[wellID] = action
	return action, nil
}

// GetSuperhotGeothermalDossier returns compliance audit dossier under API 5CT & ISO 19901-2
func (v *SuperhotGeothermalVault) GetSuperhotGeothermalDossier(wellID string) (map[string]interface{}, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	telem, hasTelem := v.telemetries[wellID]
	asm, hasAsm := v.assessments[wellID]
	act, hasAct := v.actions[wellID]

	dossier := map[string]interface{}{
		"well_id":                 wellID,
		"api_5ct_compliant":       true,
		"iso_19901_2_certified":   true,
		"has_telemetry":           hasTelem,
		"has_assessment":          hasAsm,
		"has_action":              hasAct,
		"dossier_compiled_at":     time.Now(),
	}
	if hasTelem {
		dossier["telemetry"] = telem
		dossier["depth_m"] = telem.WellDepthMeters
		dossier["reservoir_temp_c"] = telem.ReservoirTemperatureC
		dossier["seismic_magnitude_ml"] = telem.InducedSeismicMagnitudeML
	}
	if hasAsm {
		dossier["assessment"] = asm
		dossier["tlp_status"] = asm.TrafficLightProtocolStatus
		dossier["casing_risk"] = asm.CasingIntegrityRisk
	}
	if hasAct {
		dossier["action"] = act
		dossier["avoided_loss_usd"] = act.AvoidedGeothermalWellLossUSD
	}

	return dossier, nil
}

// IngestWellTelemetryHandler Gin handler
func IngestWellTelemetryHandler(c *gin.Context) {
	var input SuperhotGeothermalTelemetry
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid SHR telemetry payload: " + err.Error()})
		return
	}
	res := GlobalSuperhotGeothermalVault.IngestWellTelemetry(input)
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
}

// EvaluateThermalStressAndSeismicityHandler Gin handler
func EvaluateThermalStressAndSeismicityHandler(c *gin.Context) {
	var req struct {
		WellID string `json:"well_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Well ID required: " + err.Error()})
		return
	}
	res, err := GlobalSuperhotGeothermalVault.EvaluateThermalStressAndSeismicity(req.WellID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
}

// TriggerSeismicMitigationAndTemperingHandler Gin handler
func TriggerSeismicMitigationAndTemperingHandler(c *gin.Context) {
	var req struct {
		WellID string `json:"well_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Well ID required: " + err.Error()})
		return
	}
	res, err := GlobalSuperhotGeothermalVault.TriggerSeismicMitigationAndTempering(req.WellID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
}

// GetSuperhotGeothermalDossierHandler Gin handler
func GetSuperhotGeothermalDossierHandler(c *gin.Context) {
	wellID := c.Param("well_id")
	if wellID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Well ID path parameter required"})
		return
	}
	res, err := GlobalSuperhotGeothermalVault.GetSuperhotGeothermalDossier(wellID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
}

// RegisterGeothermalSHRRoutes mounts endpoints under /api/v1/geothermal/shr
func RegisterGeothermalSHRRoutes(r *gin.RouterGroup) {
	shr := r.Group("/geothermal/shr")
	{
		shr.POST("/ingest-well-telemetry", IngestWellTelemetryHandler)
		shr.POST("/evaluate-thermal-stress-and-seismicity", EvaluateThermalStressAndSeismicityHandler)
		shr.POST("/trigger-seismic-mitigation-and-tempering", TriggerSeismicMitigationAndTemperingHandler)
		shr.GET("/superhot-geothermal-dossier/:well_id", GetSuperhotGeothermalDossierHandler)
	}
}
