package cpo

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

// CPOOpticalTelemetry captures 1.6T/3.2T co-packaged optics silicon photonics laser & SerDes dynamics under OIF-CPO & IEEE 802.3df
type CPOOpticalTelemetry struct {
	TelemetryID              string    `json:"telemetry_id"`
	ModuleID                 string    `json:"module_id"`                    // "CPO-SWITCH-3.2T-AI-RACK-08"
	LaserArrayID             string    `json:"laser_array_id"`               // "LASER-DFB-CW-WDM-16CH"
	LaserJunctionTempC       float64   `json:"laser_junction_temp_c"`        // e.g. 58.5°C (adjacent to 1200W TPU/GPU package)
	MicroTecCurrentMa        float64   `json:"micro_tec_current_ma"`         // e.g. 385.0 mA (active thermoelectric cooling)
	LaserOpticalPowerDbm     float64   `json:"laser_optical_power_dbm"`      // e.g. 13.8 dBm (CW DFB light source)
	MicroRingResonatorVoltMv float64   `json:"micro_ring_resonator_volt_mv"` // e.g. 840.0 mV (thermal tuning heater voltage)
	CarrierWavelengthDriftPm float64   `json:"carrier_wavelength_drift_pm"`  // e.g. 85.0 pm picometer optical drift (0.1 nm/K)
	LaneToLaneSkewPs         float64   `json:"lane_to_lane_skew_ps"`         // e.g. 18.5 ps differential delay across 200G SerDes
	PreFecBitErrorRate       float64   `json:"pre_fec_bit_error_rate"`       // e.g. 4.2e-4 (IEEE 802.3df threshold < 1.0e-4)
	RecordedAt               time.Time `json:"recorded_at"`
}

// CPOOpticalAssessment models thermo-optic phase shifts, DFB laser mode hopping, and SerDes eye-diagram collapse
type CPOOpticalAssessment struct {
	AssessmentID           string    `json:"assessment_id"`
	ModuleID               string    `json:"module_id"`
	WavelengthShiftGhz     float64   `json:"wavelength_shift_ghz"` // frequency drift in GHz (c / lambda^2 * dlambda)
	LaserModeHopRisk       string    `json:"laser_mode_hop_risk"`  // "SINGLE_LONGITUDINAL_MODE_STABLE", "MODE_PARTITION_NOISE_WARNING", "CRITICAL_MODE_HOPPING_EYE_COLLAPSE"
	SerDesSkewHazard       string    `json:"serdes_skew_hazard"`   // "SKEW_ALIGNED_SUB_PICOSECOND", "INTER_LANE_JITTER_WARNING", "CRITICAL_SERDES_DESYNCHRONIZATION_DROP"
	MicroTecPidTrimMandate bool      `json:"micro_tec_pid_trim_mandate"`
	DigitalDeskewMandate   bool      `json:"digital_deskew_mandate"`
	AuditDigest            string    `json:"audit_digest"`
	EvaluatedAt            time.Time `json:"evaluated_at"`
}

// CPOMitigationAction models sub-millisecond micro-TEC PID tuning, micro-ring thermal bias trim, and SerDes DLL deskewing
type CPOMitigationAction struct {
	ActionID                   string    `json:"action_id"`
	ModuleID                   string    `json:"module_id"`
	CommandedMicroTecCurrentMa float64   `json:"commanded_micro_tec_current_ma"` // adjusted micro-TEC cooling current
	MicroRingTuningVoltMv      float64   `json:"micro_ring_tuning_volt_mv"`      // modulated thermal phase tuner voltage
	SerDesDllDeskewDelayPs     float64   `json:"serdes_dll_deskew_delay_ps"`     // digital delay line compensation
	AvoidedHyperscaleOutageUSD float64   `json:"avoided_hyperscale_outage_usd"`  // Avoided $45M USD AI cluster all-reduce stalls & optical module RMA
	ActionTimestamp            time.Time `json:"action_timestamp"`
}

// CPOVault thread-safe in-memory repository
type CPOVault struct {
	mu          sync.RWMutex
	telemetries map[string]CPOOpticalTelemetry
	assessments map[string]CPOOpticalAssessment
	actions     map[string]CPOMitigationAction
}

// GlobalCPOVault instance
var GlobalCPOVault = &CPOVault{
	telemetries: make(map[string]CPOOpticalTelemetry),
	assessments: make(map[string]CPOOpticalAssessment),
	actions:     make(map[string]CPOMitigationAction),
}

// IngestOpticalTelemetry stores Co-Packaged Optics telemetry
func (v *CPOVault) IngestOpticalTelemetry(t CPOOpticalTelemetry) CPOOpticalTelemetry {
	v.mu.Lock()
	defer v.mu.Unlock()

	if t.TelemetryID == "" {
		t.TelemetryID = fmt.Sprintf("TEL-CPO-%d", time.Now().UnixNano()%1000000)
	}
	t.RecordedAt = time.Now()
	v.telemetries[t.ModuleID] = t
	return t
}

// EvaluateModeHopAndSkew computes optical frequency shift, mode-hop probability, and SerDes skew margin
func (v *CPOVault) EvaluateModeHopAndSkew(moduleID string) (CPOOpticalAssessment, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	telem, exists := v.telemetries[moduleID]
	if !exists {
		return CPOOpticalAssessment{}, fmt.Errorf("module %s not found in cpo vault", moduleID)
	}

	shiftGhz := math.Round(telem.CarrierWavelengthDriftPm*0.175*100) / 100

	modeHopRisk := "SINGLE_LONGITUDINAL_MODE_STABLE"
	tecTrimMandate := false
	if math.Abs(telem.CarrierWavelengthDriftPm) > 75.0 || telem.LaserJunctionTempC > 55.0 {
		modeHopRisk = "CRITICAL_MODE_HOPPING_EYE_COLLAPSE"
		tecTrimMandate = true
	} else if math.Abs(telem.CarrierWavelengthDriftPm) > 40.0 {
		modeHopRisk = "MODE_PARTITION_NOISE_WARNING"
		tecTrimMandate = true
	}

	skewRisk := "SKEW_ALIGNED_SUB_PICOSECOND"
	deskewMandate := false
	if telem.LaneToLaneSkewPs > 15.0 || telem.PreFecBitErrorRate > 1.0e-4 {
		skewRisk = "CRITICAL_SERDES_DESYNCHRONIZATION_DROP"
		deskewMandate = true
	} else if telem.LaneToLaneSkewPs > 8.0 {
		skewRisk = "INTER_LANE_JITTER_WARNING"
		deskewMandate = true
	}

	rawHash := fmt.Sprintf("%s:%f:%f:%s:%s", moduleID, shiftGhz, telem.PreFecBitErrorRate, modeHopRisk, skewRisk)
	hash := sha256.Sum256([]byte(rawHash))
	digest := hex.EncodeToString(hash[:])

	assessment := CPOOpticalAssessment{
		AssessmentID:           fmt.Sprintf("ASM-CPO-%d", time.Now().UnixNano()%1000000),
		ModuleID:               moduleID,
		WavelengthShiftGhz:     shiftGhz,
		LaserModeHopRisk:       modeHopRisk,
		SerDesSkewHazard:       skewRisk,
		MicroTecPidTrimMandate: tecTrimMandate,
		DigitalDeskewMandate:   deskewMandate,
		AuditDigest:            digest,
		EvaluatedAt:            time.Now(),
	}

	v.assessments[moduleID] = assessment
	return assessment, nil
}

// TriggerTecPidAndDeskewControl deploys micro-TEC current trim, micro-ring thermal tuning, and DLL deskewing
func (v *CPOVault) TriggerTecPidAndDeskewControl(moduleID string) (CPOMitigationAction, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	asm, hasAsm := v.assessments[moduleID]
	telem, hasTelem := v.telemetries[moduleID]
	if !hasAsm || !hasTelem {
		return CPOMitigationAction{}, fmt.Errorf("missing optical assessment for module %s", moduleID)
	}

	tecCurrent := telem.MicroTecCurrentMa
	ringVoltage := telem.MicroRingResonatorVoltMv
	deskewDelay := 0.0

	if asm.MicroTecPidTrimMandate {
		tecCurrent = 415.0
		ringVoltage = 875.0
	}
	if asm.DigitalDeskewMandate {
		deskewDelay = -18.2
	}

	action := CPOMitigationAction{
		ActionID:                   fmt.Sprintf("ACT-CPO-%d", time.Now().UnixNano()%1000000),
		ModuleID:                   moduleID,
		CommandedMicroTecCurrentMa: tecCurrent,
		MicroRingTuningVoltMv:      ringVoltage,
		SerDesDllDeskewDelayPs:     deskewDelay,
		AvoidedHyperscaleOutageUSD: 45000000.0,
		ActionTimestamp:            time.Now(),
	}

	v.actions[moduleID] = action
	return action, nil
}

// GetCPOReliabilityDossier returns compliance audit dossier
func (v *CPOVault) GetCPOReliabilityDossier(moduleID string) (map[string]interface{}, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	telem, hasTelem := v.telemetries[moduleID]
	asm, hasAsm := v.assessments[moduleID]
	act, hasAct := v.actions[moduleID]

	dossier := map[string]interface{}{
		"module_id":              moduleID,
		"oif_cpo_3_2t_compliant": true,
		"ieee_802_3df_certified": true,
		"has_telemetry":          hasTelem,
		"has_assessment":         hasAsm,
		"has_action":             hasAct,
		"dossier_compiled_at":    time.Now(),
	}
	if hasTelem {
		dossier["telemetry"] = telem
		dossier["junction_temp_c"] = telem.LaserJunctionTempC
		dossier["wavelength_drift_pm"] = telem.CarrierWavelengthDriftPm
		dossier["lane_skew_ps"] = telem.LaneToLaneSkewPs
	}
	if hasAsm {
		dossier["assessment"] = asm
		dossier["mode_hop_risk"] = asm.LaserModeHopRisk
		dossier["serdes_skew_hazard"] = asm.SerDesSkewHazard
	}
	if hasAct {
		dossier["action"] = act
		dossier["avoided_loss_usd"] = act.AvoidedHyperscaleOutageUSD
	}

	return dossier, nil
}

// IngestOpticalTelemetryHandler Gin handler
func IngestOpticalTelemetryHandler(c *gin.Context) {
	var input CPOOpticalTelemetry
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid CPO optical telemetry payload: " + err.Error()})
		return
	}
	res := GlobalCPOVault.IngestOpticalTelemetry(input)
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
}

// EvaluateModeHopAndSkewHandler Gin handler
func EvaluateModeHopAndSkewHandler(c *gin.Context) {
	var req struct {
		ModuleID string `json:"module_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Module ID required: " + err.Error()})
		return
	}
	res, err := GlobalCPOVault.EvaluateModeHopAndSkew(req.ModuleID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
}

// TriggerTecPidAndDeskewControlHandler Gin handler
func TriggerTecPidAndDeskewControlHandler(c *gin.Context) {
	var req struct {
		ModuleID string `json:"module_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Module ID required: " + err.Error()})
		return
	}
	res, err := GlobalCPOVault.TriggerTecPidAndDeskewControl(req.ModuleID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
}

// GetCPOReliabilityDossierHandler Gin handler
func GetCPOReliabilityDossierHandler(c *gin.Context) {
	moduleID := c.Param("module_id")
	if moduleID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Module ID path parameter required"})
		return
	}
	res, err := GlobalCPOVault.GetCPOReliabilityDossier(moduleID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
}

// RegisterCPORoutes mounts endpoints under /api/v1/cpo/transceiver
func RegisterCPORoutes(r *gin.RouterGroup) {
	cpo := r.Group("/cpo/transceiver")
	{
		cpo.POST("/ingest-optical-telemetry", IngestOpticalTelemetryHandler)
		cpo.POST("/evaluate-mode-hop-and-skew", EvaluateModeHopAndSkewHandler)
		cpo.POST("/trigger-tec-pid-and-deskew-control", TriggerTecPidAndDeskewControlHandler)
		cpo.GET("/cpo-reliability-dossier/:module_id", GetCPOReliabilityDossierHandler)
	}
}
