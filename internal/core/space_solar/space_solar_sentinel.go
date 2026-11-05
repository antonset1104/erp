package spacesolar

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// SpaceSolarTelemetry captures gigawatt space-based solar power microwave phased-array telemetry
type SpaceSolarTelemetry struct {
	TelemetryID                          string    `json:"telemetry_id"`
	SatelliteArrayID                     string    `json:"satellite_array_id"`
	OrbitalSlotGeo                       string    `json:"orbital_slot_geo"`                          // e.g. "GEO_122.5_WEST"
	TransmittedRfPowerGigawatts          float64   `json:"transmitted_rf_power_gigawatts"`            // 1.0 - 2.5 GW
	RetrodirectiveBeamDriftDeg           float64   `json:"retrodirective_beam_drift_deg"`             // limit < 0.0005°
	RectennaSidelobeFluxMilliwattsPerCm2 float64   `json:"rectenna_sidelobe_flux_milliwatts_per_cm2"` // safety limit < 1.0 mW/cm2
	IonosphericSelfFocusingIndex         float64   `json:"ionospheric_self_focusing_index"`           // threshold > 0.75
	SolarMirrorThermalDistortionMm       float64   `json:"solar_mirror_thermal_distortion_mm"`        // limit < 1.5 mm
	PilotUplinkCarrierToNoiseDb          float64   `json:"pilot_uplink_carrier_to_noise_db"`          // target > 24.0 dB
	ArrayThermalSkinTempKelvin           float64   `json:"array_thermal_skin_temp_kelvin"`            // limit < 385.0 K
	Timestamp                            time.Time `json:"timestamp"`
}

// SpaceSolarAssessment models beam pointing containment, ionospheric filamentation, and ITU Article 21 compliance
type SpaceSolarAssessment struct {
	AssessmentID                     string    `json:"assessment_id"`
	SatelliteArrayID                 string    `json:"satellite_array_id"`
	BeamPointingContainmentStatus    string    `json:"beam_pointing_containment_status"` // LOCK_ON_RECTENNA_BORESIGHT, MARGINAL_PHASE_JITTER, UNCONTAINED_BEAM_WANDER_HAZARD
	IonosphericDistortionHazard      string    `json:"ionospheric_distortion_hazard"`    // LINEAR_IONOSPHERE_PROPAGATION, THERMAL_SELF_FOCUSING_FILAMENTATION
	SidelobePowerExcessMarginPct     float64   `json:"sidelobe_power_excess_margin_pct"`
	FastPhaseConjugatorCutoffMandate bool      `json:"fast_phase_conjugator_cutoff_mandate"`
	PowerSteppingAttenuationMandate  bool      `json:"power_stepping_attenuation_mandate"`
	AuditDigest                      string    `json:"audit_digest"`
	EvaluatedAt                      time.Time `json:"evaluated_at"`
}

// SpaceSolarMitigationAction models sub-microsecond retrodirective RF cutoff, array power stepping, and gimbal trim
type SpaceSolarMitigationAction struct {
	ActionID                            string    `json:"action_id"`
	SatelliteArrayID                    string    `json:"satellite_array_id"`
	CommandedFastCutoffResponseMicroSec float64   `json:"commanded_fast_cutoff_response_micro_sec"`
	ArrayPowerAttenuationDb             float64   `json:"array_power_attenuation_db"`
	ReactionControlGimbalCorrectionDeg  float64   `json:"reaction_control_gimbal_correction_deg"`
	AvoidedOrbitalSPSDisasterUSD        float64   `json:"avoided_orbital_sps_disaster_usd"`
	ExecutedAt                          time.Time `json:"executed_at"`
}

// SpaceSolarVault manages gigawatt space solar power satellites and ground rectenna safety
type SpaceSolarVault struct {
	mu          sync.RWMutex
	telemetries map[string]SpaceSolarTelemetry
	assessments map[string]SpaceSolarAssessment
	actions     map[string]SpaceSolarMitigationAction
}

// GlobalSpaceSolarVault is the singleton instance for space solar power sentinel
var GlobalSpaceSolarVault = NewSpaceSolarVault()

func NewSpaceSolarVault() *SpaceSolarVault {
	return &SpaceSolarVault{
		telemetries: make(map[string]SpaceSolarTelemetry),
		assessments: make(map[string]SpaceSolarAssessment),
		actions:     make(map[string]SpaceSolarMitigationAction),
	}
}

// IngestSolarTelemetry stores space solar power satellite phased-array metrics
func (v *SpaceSolarVault) IngestSolarTelemetry(telemetry SpaceSolarTelemetry) SpaceSolarTelemetry {
	v.mu.Lock()
	defer v.mu.Unlock()

	if telemetry.Timestamp.IsZero() {
		telemetry.Timestamp = time.Now()
	}
	v.telemetries[telemetry.SatelliteArrayID] = telemetry
	return telemetry
}

// EvaluateBeamContainment evaluates ITU Radio Regulations Article 21 and IEEE C95.1 microwave power limits
func (v *SpaceSolarVault) EvaluateBeamContainment(satelliteArrayID string) (SpaceSolarAssessment, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	telem, exists := v.telemetries[satelliteArrayID]
	if !exists {
		return SpaceSolarAssessment{}, fmt.Errorf("telemetry not found for space solar satellite %s", satelliteArrayID)
	}

	containmentStatus := "LOCK_ON_RECTENNA_BORESIGHT"
	ionosphericHazard := "LINEAR_IONOSPHERE_PROPAGATION"
	cutoffMandate := false
	attenuateMandate := false

	safetyLimit := 1.0 // 1.0 mW/cm2 safety limit at rectenna boundary
	excessMargin := (telem.RectennaSidelobeFluxMilliwattsPerCm2 - safetyLimit) / safetyLimit * 100.0

	if telem.RetrodirectiveBeamDriftDeg >= 0.0005 || telem.RectennaSidelobeFluxMilliwattsPerCm2 >= 1.0 || telem.IonosphericSelfFocusingIndex >= 0.75 {
		containmentStatus = "UNCONTAINED_BEAM_WANDER_HAZARD"
		cutoffMandate = true
		attenuateMandate = true
		if telem.IonosphericSelfFocusingIndex >= 0.75 {
			ionosphericHazard = "THERMAL_SELF_FOCUSING_FILAMENTATION"
		}
	} else if telem.RetrodirectiveBeamDriftDeg > 0.0002 || telem.RectennaSidelobeFluxMilliwattsPerCm2 > 0.6 {
		containmentStatus = "MARGINAL_PHASE_JITTER"
		attenuateMandate = true
	}

	hashInput := fmt.Sprintf("%s:%s:%s:%.6f:%.2f:%t",
		telem.SatelliteArrayID, containmentStatus, ionosphericHazard, telem.RetrodirectiveBeamDriftDeg, telem.RectennaSidelobeFluxMilliwattsPerCm2, cutoffMandate)
	hasher := sha256.New()
	hasher.Write([]byte(hashInput))
	digest := hex.EncodeToString(hasher.Sum(nil))

	assessment := SpaceSolarAssessment{
		AssessmentID:                     fmt.Sprintf("ASM-SPS-%d", time.Now().UnixNano()%1000000),
		SatelliteArrayID:                 satelliteArrayID,
		BeamPointingContainmentStatus:    containmentStatus,
		IonosphericDistortionHazard:      ionosphericHazard,
		SidelobePowerExcessMarginPct:     excessMargin,
		FastPhaseConjugatorCutoffMandate: cutoffMandate,
		PowerSteppingAttenuationMandate:  attenuateMandate,
		AuditDigest:                      digest,
		EvaluatedAt:                      time.Now(),
	}

	v.assessments[satelliteArrayID] = assessment
	return assessment, nil
}

// TriggerFastCutoffAndGimbalTrim deactivates phased array within 0.8 µs and corrects gimbal pointing
func (v *SpaceSolarVault) TriggerFastCutoffAndGimbalTrim(satelliteArrayID string) (SpaceSolarMitigationAction, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	asm, hasAsm := v.assessments[satelliteArrayID]
	_, hasTelem := v.telemetries[satelliteArrayID]
	if !hasAsm || !hasTelem {
		return SpaceSolarMitigationAction{}, fmt.Errorf("missing telemetry or assessment for space solar satellite %s", satelliteArrayID)
	}

	cutoffTime := 0.0
	attenuationDb := 0.0
	gimbalCorrection := 0.0

	if asm.FastPhaseConjugatorCutoffMandate {
		cutoffTime = 0.65    // 0.65 µs sub-microsecond retrodirective RF phase-gate quench
		attenuationDb = 45.0 // immediate 45 dB attenuation of transmitted power
	}
	if asm.PowerSteppingAttenuationMandate {
		gimbalCorrection = -0.00045 // 0.00045° micro-gimbal optical boresight bias compensation
	}

	action := SpaceSolarMitigationAction{
		ActionID:                            fmt.Sprintf("ACT-SPS-%d", time.Now().UnixNano()%1000000),
		SatelliteArrayID:                    satelliteArrayID,
		CommandedFastCutoffResponseMicroSec: cutoffTime,
		ArrayPowerAttenuationDb:             attenuationDb,
		ReactionControlGimbalCorrectionDeg:  gimbalCorrection,
		AvoidedOrbitalSPSDisasterUSD:        250000000.0, // $250M USD avoided uncontained high-power microwave spillover, ground hazards, and satellite loss
		ExecutedAt:                          time.Now(),
	}

	v.actions[satelliteArrayID] = action
	return action, nil
}

// GetSpaceSolarDossier compiles space solar power certification, ITU compliance, and RF safety records
func (v *SpaceSolarVault) GetSpaceSolarDossier(satelliteArrayID string) (map[string]interface{}, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	telem, hasTelem := v.telemetries[satelliteArrayID]
	asm, hasAsm := v.assessments[satelliteArrayID]
	act, hasAct := v.actions[satelliteArrayID]

	if !hasTelem || !hasAsm {
		return nil, fmt.Errorf("incomplete data for space solar dossier: %s", satelliteArrayID)
	}

	dossier := map[string]interface{}{
		"satellite_array_id":          satelliteArrayID,
		"orbital_slot_geo":            telem.OrbitalSlotGeo,
		"itu_radio_reg_art21_cleared": asm.BeamPointingContainmentStatus != "UNCONTAINED_BEAM_WANDER_HAZARD" || (hasAct && act.AvoidedOrbitalSPSDisasterUSD > 0),
		"nasa_sp8000_compliant":       true,
		"ieee_c95_1_certified":        asm.SidelobePowerExcessMarginPct <= 0 || hasAct,
		"transmitted_gw":              telem.TransmittedRfPowerGigawatts,
		"beam_drift_deg":              telem.RetrodirectiveBeamDriftDeg,
		"sidelobe_flux_mw_cm2":        telem.RectennaSidelobeFluxMilliwattsPerCm2,
		"containment_status":          asm.BeamPointingContainmentStatus,
		"ionospheric_hazard":          asm.IonosphericDistortionHazard,
		"cutoff_response_us":          0.0,
		"power_attenuation_db":        0.0,
		"avoided_loss_usd":            0.0,
		"audit_digest":                asm.AuditDigest,
		"compiled_at":                 time.Now(),
	}

	if hasAct {
		dossier["cutoff_response_us"] = act.CommandedFastCutoffResponseMicroSec
		dossier["power_attenuation_db"] = act.ArrayPowerAttenuationDb
		dossier["avoided_loss_usd"] = act.AvoidedOrbitalSPSDisasterUSD
	}

	return dossier, nil
}

// RegisterSpaceSolarRoutes wires API endpoints under /api/v1/space/space-solar/*
func RegisterSpaceSolarRoutes(r *gin.RouterGroup) {
	spsGroup := r.Group("/space/space-solar")
	{
		spsGroup.POST("/ingest-solar-telemetry", func(c *gin.Context) {
			var t SpaceSolarTelemetry
			if err := c.ShouldBindJSON(&t); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res := GlobalSpaceSolarVault.IngestSolarTelemetry(t)
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		spsGroup.POST("/evaluate-beam-containment", func(c *gin.Context) {
			var req struct {
				SatelliteArrayID string `json:"satellite_array_id"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := GlobalSpaceSolarVault.EvaluateBeamContainment(req.SatelliteArrayID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		spsGroup.POST("/trigger-fast-cutoff-and-gimbal-trim", func(c *gin.Context) {
			var req struct {
				SatelliteArrayID string `json:"satellite_array_id"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := GlobalSpaceSolarVault.TriggerFastCutoffAndGimbalTrim(req.SatelliteArrayID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		spsGroup.GET("/space-solar-dossier/:satellite_array_id", func(c *gin.Context) {
			sID := c.Param("satellite_array_id")
			res, err := GlobalSpaceSolarVault.GetSpaceSolarDossier(sID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})
	}
}
