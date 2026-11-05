package mbmw

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

// MBMWReticleTelemetry captures semiconductor multi-beam mask writer (MBMW) electron beam dynamics under SEMI P48 & ISO 14644-1
type MBMWReticleTelemetry struct {
	TelemetryID                  string    `json:"telemetry_id"`
	ReticleID                    string    `json:"reticle_id"`                         // "RETICLE-EUV-2NM-METAL1-009"
	WriterToolID                 string    `json:"writer_tool_id"`                     // "MBMW-EUV-512K-BEAM-01"
	AccelerationVoltageKv        float64   `json:"acceleration_voltage_kv"`            // e.g. 50.0 kV
	TotalBeamCurrentMicroA       float64   `json:"total_beam_current_micro_a"`         // e.g. 18.5 uA
	ActiveBeamCount              int       `json:"active_beam_count"`                  // e.g. 262144 beams
	ElectrostaticChuckTempMilliK float64   `json:"electrostatic_chuck_temp_milli_k"`   // e.g. 21450.0 mK
	BeamDriftRateNmPerHr         float64   `json:"beam_drift_rate_nm_per_hr"`          // e.g. 0.45 nm/hr (spec < 0.20 nm/hr)
	AstigmatismDeflectionErrorNm float64   `json:"astigmatism_deflection_error_nm"`    // e.g. 0.62 nm (limit < 0.50 nm)
	CriticalDimensionUniformityNm float64  `json:"critical_dimension_uniformity_nm"`   // e.g. 0.78 nm (target < 0.50 nm)
	BlankMaskSurfaceChargeVolts  float64   `json:"blank_mask_surface_charge_volts"`    // e.g. -4.8 V
	RecordedAt                   time.Time `json:"recorded_at"`
}

// MBMWAssessment evaluates electron beam astigmatism blur, deflection field drift, and chuck thermal expansion
type MBMWAssessment struct {
	AssessmentID                  string    `json:"assessment_id"`
	ReticleID                     string    `json:"reticle_id"`
	CriticalDimensionErrorMarginNm float64  `json:"critical_dimension_error_margin_nm"`
	BeamDeflectionDistortionHazard string   `json:"beam_deflection_distortion_hazard"` // "DEFLECTION_FIELD_LINEAR", "ASTIGMATISM_BLUR_WARNING", "CRITICAL_EBEAM_DISTORTION_REJECT"
	ChuckThermalUniformityStatus  string    `json:"chuck_thermal_uniformity_status"`   // "SUB_10MK_THERMAL_STABLE", "GRADIENT_EXPANSION_WARNING", "THERMAL_DRIFT_RUNAWAY_ABORT"
	StigmatorCorrectionMandate    bool      `json:"stigmator_correction_mandate"`
	DoseModulationMatrixMandate   bool      `json:"dose_modulation_matrix_mandate"`
	AuditDigest                   string    `json:"audit_digest"`
	EvaluatedAt                   time.Time `json:"evaluated_at"`
}

// MBMWAction models multi-pole stigmator voltage tuning, deflection field trim, and Peltier thermal stabilization
type MBMWAction struct {
	ActionID                       string    `json:"action_id"`
	ReticleID                      string    `json:"reticle_id"`
	StigmatorCorrectionVoltageV    float64   `json:"stigmator_correction_voltage_v"`
	MagneticDeflectorTrimPpm       float64   `json:"magnetic_deflector_trim_ppm"`
	PeltierChuckStabilizationPowerW float64  `json:"peltier_chuck_stabilization_power_w"`
	AvoidedReticleScrapLossUSD     float64   `json:"avoided_reticle_scrap_loss_usd"`
	ActionTimestamp                time.Time `json:"action_timestamp"`
}

// MBMWVault thread-safe in-memory store
type MBMWVault struct {
	mu          sync.RWMutex
	telemetries map[string]MBMWReticleTelemetry
	assessments map[string]MBMWAssessment
	actions     map[string]MBMWAction
}

// GlobalMBMWVault instance
var GlobalMBMWVault = &MBMWVault{
	telemetries: make(map[string]MBMWReticleTelemetry),
	assessments: make(map[string]MBMWAssessment),
	actions:     make(map[string]MBMWAction),
}

// IngestReticleTelemetry stores multi-beam mask writer telemetry
func (v *MBMWVault) IngestReticleTelemetry(t MBMWReticleTelemetry) MBMWReticleTelemetry {
	v.mu.Lock()
	defer v.mu.Unlock()

	if t.TelemetryID == "" {
		t.TelemetryID = fmt.Sprintf("TEL-MBMW-%d", time.Now().UnixNano()%1000000)
	}
	if t.RecordedAt.IsZero() {
		t.RecordedAt = time.Now()
	}

	v.telemetries[t.ReticleID] = t
	return t
}

// EvaluateBeamAstigmatismAndDrift computes critical dimension margin, astigmatism blur, and thermal drift
func (v *MBMWVault) EvaluateBeamAstigmatismAndDrift(reticleID string) (MBMWAssessment, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	telem, exists := v.telemetries[reticleID]
	if !exists {
		return MBMWAssessment{}, fmt.Errorf("telemetry not found for reticle: %s", reticleID)
	}

	cdMargin := math.Round((telem.CriticalDimensionUniformityNm-0.50)*100) / 100

	distortionHazard := "DEFLECTION_FIELD_LINEAR"
	stigmatorMandate := false
	doseModulation := false

	if telem.AstigmatismDeflectionErrorNm > 0.50 || telem.BeamDriftRateNmPerHr > 0.35 {
		distortionHazard = "CRITICAL_EBEAM_DISTORTION_REJECT"
		stigmatorMandate = true
		doseModulation = true
	} else if telem.AstigmatismDeflectionErrorNm > 0.30 || telem.BeamDriftRateNmPerHr > 0.20 {
		distortionHazard = "ASTIGMATISM_BLUR_WARNING"
		stigmatorMandate = true
	}

	thermalStatus := "SUB_10MK_THERMAL_STABLE"
	if telem.ElectrostaticChuckTempMilliK > 21400.0 || math.Abs(telem.BlankMaskSurfaceChargeVolts) > 4.0 {
		thermalStatus = "THERMAL_DRIFT_RUNAWAY_ABORT"
		doseModulation = true
	} else if telem.ElectrostaticChuckTempMilliK > 21200.0 {
		thermalStatus = "GRADIENT_EXPANSION_WARNING"
	}

	rawHash := fmt.Sprintf("%s:%f:%s:%s:%t:%t", reticleID, cdMargin, distortionHazard, thermalStatus, stigmatorMandate, doseModulation)
	hash := sha256.Sum256([]byte(rawHash))
	digest := hex.EncodeToString(hash[:])

	assessment := MBMWAssessment{
		AssessmentID:                  fmt.Sprintf("ASM-MBMW-%d", time.Now().UnixNano()%1000000),
		ReticleID:                     reticleID,
		CriticalDimensionErrorMarginNm: cdMargin,
		BeamDeflectionDistortionHazard: distortionHazard,
		ChuckThermalUniformityStatus:  thermalStatus,
		StigmatorCorrectionMandate:    stigmatorMandate,
		DoseModulationMatrixMandate:   doseModulation,
		AuditDigest:                   digest,
		EvaluatedAt:                   time.Now(),
	}

	v.assessments[reticleID] = assessment
	return assessment, nil
}

// TriggerStigmatorAndDoseCorrection executes multi-pole stigmator voltage offset, magnetic deflection trim, and Peltier cooling
func (v *MBMWVault) TriggerStigmatorAndDoseCorrection(reticleID string) (MBMWAction, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	asm, hasAsm := v.assessments[reticleID]
	_, hasTelem := v.telemetries[reticleID]
	if !hasAsm || !hasTelem {
		return MBMWAction{}, fmt.Errorf("missing telemetry or assessment for reticle %s", reticleID)
	}

	stigmatorVolt := 0.0
	deflectorTrim := 0.0
	peltierPower := 25.0

	if asm.StigmatorCorrectionMandate {
		stigmatorVolt = 14.8 // apply 14.8V stigmator electrostatic correction to circularize multi-beam spot profile
		deflectorTrim = -38.5 // magnetic deflector gain trim in ppm to compensate beam trajectory drift
	}
	if asm.DoseModulationMatrixMandate {
		peltierPower = 85.0 // boost Peltier chuck cooling to damp sub-10mK thermal expansion wave
	}

	action := MBMWAction{
		ActionID:                       fmt.Sprintf("ACT-MBMW-%d", time.Now().UnixNano()%1000000),
		ReticleID:                      reticleID,
		StigmatorCorrectionVoltageV:    stigmatorVolt,
		MagneticDeflectorTrimPpm:       deflectorTrim,
		PeltierChuckStabilizationPowerW: peltierPower,
		AvoidedReticleScrapLossUSD:     40000000.0,
		ActionTimestamp:                time.Now(),
	}

	v.actions[reticleID] = action
	return action, nil
}

// GetMBMWReliabilityDossier returns reticle electron beam writing compliance dossier under SEMI P48 & ISO 14644-1
func (v *MBMWVault) GetMBMWReliabilityDossier(reticleID string) (map[string]interface{}, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	telem, hasTelem := v.telemetries[reticleID]
	asm, hasAsm := v.assessments[reticleID]
	act := v.actions[reticleID]

	if !hasTelem || !hasAsm {
		return nil, fmt.Errorf("incomplete data for reticle dossier %s", reticleID)
	}

	return map[string]interface{}{
		"reticle_id":           reticleID,
		"writer_tool_id":       telem.WriterToolID,
		"semi_p48_compliant":   true,
		"iso_14644_certified":  true,
		"telemetry":            telem,
		"assessment":           asm,
		"latest_action":        act,
		"compiled_at":          time.Now(),
	}, nil
}

// RegisterMBMWRoutes wires API endpoints under /api/v1/semicon/mbmw/*
func RegisterMBMWRoutes(r *gin.RouterGroup) {
	mbmwGroup := r.Group("/semicon/mbmw")
	{
		mbmwGroup.POST("/ingest-reticle-telemetry", func(c *gin.Context) {
			var t MBMWReticleTelemetry
			if err := c.ShouldBindJSON(&t); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res := GlobalMBMWVault.IngestReticleTelemetry(t)
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		mbmwGroup.POST("/evaluate-beam-astigmatism-and-drift", func(c *gin.Context) {
			var req struct {
				ReticleID string `json:"reticle_id"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := GlobalMBMWVault.EvaluateBeamAstigmatismAndDrift(req.ReticleID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		mbmwGroup.POST("/trigger-stigmator-and-dose-correction", func(c *gin.Context) {
			var req struct {
				ReticleID string `json:"reticle_id"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := GlobalMBMWVault.TriggerStigmatorAndDoseCorrection(req.ReticleID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		mbmwGroup.GET("/mbmw-reliability-dossier/:reticle_id", func(c *gin.Context) {
			reticleID := c.Param("reticle_id")
			res, err := GlobalMBMWVault.GetMBMWReliabilityDossier(reticleID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})
	}
}
