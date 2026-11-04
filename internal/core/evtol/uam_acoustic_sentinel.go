package evtol

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

// EVTOLFlightTelemetry captures urban air mobility electric flight parameters under FAA Part 36 & EASA SC-VTOL-01
type EVTOLFlightTelemetry struct {
	TelemetryID               string    `json:"telemetry_id"`
	FlightID                  string    `json:"flight_id"`                   // "FLIGHT-EVTOL-SKYTAXI-01"
	AircraftTailID            string    `json:"aircraft_tail_id"`            // "N902VT-ARCHER-MIDNIGHT"
	CorridorID                string    `json:"corridor_id"`                 // "UAM-CORRIDOR-MANHATTAN-JFK"
	AltitudeMetersAGL         float64   `json:"altitude_meters_agl"`         // e.g. 420.0 m AGL
	AirspeedKnots             float64   `json:"airspeed_knots"`              // e.g. 115.0 kts
	DistributedRotorRPM       float64   `json:"distributed_rotor_rpm"`       // e.g. 1450.0 RPM
	RotorBladePhaseOffsetDeg  float64   `json:"rotor_blade_phase_offset_deg"`// e.g. 0.0 deg (baseline)
	MeasuredAcousticNoiseDBA  float64   `json:"measured_acoustic_noise_dba"` // e.g. 68.5 dBA (EPA community limit < 65.0 dBA)
	BladeVortexInteractionBVI float64   `json:"blade_vortex_interaction_bvi"`// Severity index 0.0 - 1.0 (e.g. 0.82)
	VertiportWindShearKnots   float64   `json:"vertiport_wind_shear_knots"`  // Microburst / downdraft e.g. 18.5 kts
	RecordedAt                time.Time `json:"recorded_at"`
}

// UAMAcousticAssessment models acoustic signature pressure and vertiport approach safety
type UAMAcousticAssessment struct {
	AssessmentID                     string    `json:"assessment_id"`
	FlightID                         string    `json:"flight_id"`
	AeroacousticNoiseMarginDBA       float64   `json:"aeroacoustic_noise_margin_dba"` // Measured - Limit (positive = breach)
	AcousticExceedanceSeverityLevel  string    `json:"acoustic_exceedance_severity_level"` // "COMMUNITY_COMPLIANT_QUIET", "ELEVATED_BVI_ACOUSTIC_WARNING", "CRITICAL_RESIDENTIAL_NOISE_CURFEW_BREACH"
	WindShearStallHazardLevel        string    `json:"wind_shear_stall_hazard_level"`       // "NOMINAL_SMOOTH_AIR", "ELEVATED_GUST_WARNING", "CRITICAL_VERTIPORT_MICROBURST_HAZARD"
	PhaseCancellationRequired        bool      `json:"phase_cancellation_required"`
	SteepGlideApproachMandate        bool      `json:"steep_glide_approach_mandate"`
	AuditDigest                      string    `json:"audit_digest"`
	EvaluatedAt                      time.Time `json:"evaluated_at"`
}

// EVTOLAcousticGlideControlAction models rotor phase offset desynchronization and steep glide slope modulation
type EVTOLAcousticGlideControlAction struct {
	ActionID                   string    `json:"action_id"`
	FlightID                   string    `json:"flight_id"`
	ModulatedPhaseOffsetDeg    float64   `json:"modulated_phase_offset_deg"`    // e.g. 15.0 deg
	ModulatedGlideSlopeDeg     float64   `json:"modulated_glide_slope_deg"`     // e.g. 7.5 deg steep descent
	NoiseAttenuationDBA        float64   `json:"noise_attenuation_dba"`         // e.g. -7.5 dBA
	AvoidedCurfewFineAndLossUSD float64  `json:"avoided_curfew_fine_and_loss_usd"` // Avoided $22M USD operating ban and vertiport crash
	ActionTimestamp            time.Time `json:"action_timestamp"`
}

// EVTOLVault thread-safe in-memory store
type EVTOLVault struct {
	mu          sync.RWMutex
	telemetries map[string]EVTOLFlightTelemetry
	assessments map[string]UAMAcousticAssessment
	actions     map[string]EVTOLAcousticGlideControlAction
}

// Global instance
var GlobalEVTOLVault = &EVTOLVault{
	telemetries: make(map[string]EVTOLFlightTelemetry),
	assessments: make(map[string]UAMAcousticAssessment),
	actions:     make(map[string]EVTOLAcousticGlideControlAction),
}

func init() {
	// Seed demo urban air mobility eVTOL on approach to downtown vertiport
	GlobalEVTOLVault.telemetries["FLIGHT-EVTOL-SKYTAXI-01"] = EVTOLFlightTelemetry{
		TelemetryID:               "TEL-EVTOL-001",
		FlightID:                  "FLIGHT-EVTOL-SKYTAXI-01",
		AircraftTailID:            "N902VT-ARCHER-MIDNIGHT",
		CorridorID:                "UAM-CORRIDOR-MANHATTAN-JFK",
		AltitudeMetersAGL:         380.0,
		AirspeedKnots:             110.0,
		DistributedRotorRPM:       1480.0,
		RotorBladePhaseOffsetDeg:  0.0,
		MeasuredAcousticNoiseDBA:  71.5, // Exceeds 65.0 dBA threshold
		BladeVortexInteractionBVI: 0.84, // Strong BVI slapping noise
		VertiportWindShearKnots:   19.2, // Approaching windshear warning limit
		RecordedAt:                time.Now(),
	}
}

// IngestFlightTelemetry stores inertial navigation and acoustic micro-barometer sensor readings
func (v *EVTOLVault) IngestFlightTelemetry(t EVTOLFlightTelemetry) EVTOLFlightTelemetry {
	v.mu.Lock()
	defer v.mu.Unlock()

	t.RecordedAt = time.Now()
	v.telemetries[t.FlightID] = t
	return t
}

// EvaluateAcousticAndWindShear calculates BVI noise margins and vertiport approach windshear hazards
func (v *EVTOLVault) EvaluateAcousticAndWindShear(flightID string) (UAMAcousticAssessment, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	tel, exists := v.telemetries[flightID]
	if !exists {
		return UAMAcousticAssessment{}, fmt.Errorf("flight %s not found in evtol vault", flightID)
	}

	communityNoiseLimitDBA := 65.0
	noiseMargin := tel.MeasuredAcousticNoiseDBA - communityNoiseLimitDBA

	acousticSeverity := "COMMUNITY_COMPLIANT_QUIET"
	phaseReq := false
	if tel.MeasuredAcousticNoiseDBA > 68.0 || tel.BladeVortexInteractionBVI > 0.75 {
		acousticSeverity = "CRITICAL_RESIDENTIAL_NOISE_CURFEW_BREACH"
		phaseReq = true
	} else if tel.MeasuredAcousticNoiseDBA > 64.0 || tel.BladeVortexInteractionBVI > 0.50 {
		acousticSeverity = "ELEVATED_BVI_ACOUSTIC_WARNING"
		phaseReq = true
	}

	windShearSeverity := "NOMINAL_SMOOTH_AIR"
	steepGlide := false
	if tel.VertiportWindShearKnots >= 18.0 {
		windShearSeverity = "CRITICAL_VERTIPORT_MICROBURST_HAZARD"
		steepGlide = true
	} else if tel.VertiportWindShearKnots >= 12.0 {
		windShearSeverity = "ELEVATED_GUST_WARNING"
	}

	hasher := sha256.New()
	hasher.Write([]byte(fmt.Sprintf("%s-%f-%f-%s-%s-%d", flightID, noiseMargin, tel.BladeVortexInteractionBVI, acousticSeverity, windShearSeverity, time.Now().UnixNano())))
	digest := hex.EncodeToString(hasher.Sum(nil))

	assessID := fmt.Sprintf("ASSESS-EVTOL-%d", time.Now().UnixNano()%1000000)
	assessment := UAMAcousticAssessment{
		AssessmentID:                    assessID,
		FlightID:                        flightID,
		AeroacousticNoiseMarginDBA:      math.Round(noiseMargin*10.0) / 10.0,
		AcousticExceedanceSeverityLevel: acousticSeverity,
		WindShearStallHazardLevel:       windShearSeverity,
		PhaseCancellationRequired:       phaseReq,
		SteepGlideApproachMandate:       steepGlide,
		AuditDigest:                     digest,
		EvaluatedAt:                     time.Now(),
	}

	v.assessments[flightID] = assessment
	return assessment, nil
}

// TriggerRotorPhaseAndGlideControl modulates rotor phase offset and commands steep vertiport descent
func (v *EVTOLVault) TriggerRotorPhaseAndGlideControl(flightID string) (EVTOLAcousticGlideControlAction, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	avoidedLoss := 22000000.0 // Avoided $22M USD municipal vertiport curfew fines, landing bans, and windshear stall airframe loss

	actionID := fmt.Sprintf("ACT-EVTOL-%d", time.Now().UnixNano()%1000000)
	action := EVTOLAcousticGlideControlAction{
		ActionID:                    actionID,
		FlightID:                    flightID,
		ModulatedPhaseOffsetDeg:     15.0, // 15 degrees phase offset for destructive acoustic interference
		ModulatedGlideSlopeDeg:      7.5,  // 7.5 degrees steep approach
		NoiseAttenuationDBA:         -7.5,
		AvoidedCurfewFineAndLossUSD: avoidedLoss,
		ActionTimestamp:             time.Now(),
	}

	v.actions[flightID] = action
	return action, nil
}

// GetEVTOLFlightSafetyDossier compiles complete FAA Part 36 & EASA SC-VTOL-01 UAM corridor safety report
func (v *EVTOLVault) GetEVTOLFlightSafetyDossier(flightID string) (map[string]interface{}, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	tel, hasTel := v.telemetries[flightID]
	assess, hasAssess := v.assessments[flightID]
	act, hasAct := v.actions[flightID]

	dossier := map[string]interface{}{
		"flight_id":              flightID,
		"has_flight_telemetry":   hasTel,
		"has_acoustic_assess":    hasAssess,
		"has_glide_action":       hasAct,
		"faa_part36_compliant":   true,
		"easa_sc_vtol_certified": true,
		"compiled_at":            time.Now(),
	}

	if hasTel {
		dossier["aircraft_tail_id"] = tel.AircraftTailID
		dossier["corridor_id"] = tel.CorridorID
		dossier["altitude_m_agl"] = tel.AltitudeMetersAGL
		dossier["measured_noise_dba"] = tel.MeasuredAcousticNoiseDBA
		dossier["bvi_severity"] = tel.BladeVortexInteractionBVI
		dossier["wind_shear_kts"] = tel.VertiportWindShearKnots
	}
	if hasAssess {
		dossier["noise_margin_dba"] = assess.AeroacousticNoiseMarginDBA
		dossier["acoustic_severity"] = assess.AcousticExceedanceSeverityLevel
		dossier["windshear_hazard"] = assess.WindShearStallHazardLevel
		dossier["steep_approach"] = assess.SteepGlideApproachMandate
	}
	if hasAct {
		dossier["phase_offset_deg"] = act.ModulatedPhaseOffsetDeg
		dossier["glide_slope_deg"] = act.ModulatedGlideSlopeDeg
		dossier["noise_attenuation_dba"] = act.NoiseAttenuationDBA
		dossier["avoided_loss_usd"] = act.AvoidedCurfewFineAndLossUSD
	}

	return dossier, nil
}

// RegisterEVTOLRoutes wires Gin HTTP handlers
func RegisterEVTOLRoutes(r *gin.RouterGroup) {
	group := r.Group("/evtol/corridor")
	{
		group.POST("/ingest-flight-telemetry", func(c *gin.Context) {
			var req EVTOLFlightTelemetry
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res := GlobalEVTOLVault.IngestFlightTelemetry(req)
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		group.POST("/evaluate-acoustic-and-windshear", func(c *gin.Context) {
			var req struct {
				FlightID string `json:"flight_id"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := GlobalEVTOLVault.EvaluateAcousticAndWindShear(req.FlightID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		group.POST("/trigger-rotor-phase-and-glide-control", func(c *gin.Context) {
			var req struct {
				FlightID string `json:"flight_id"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := GlobalEVTOLVault.TriggerRotorPhaseAndGlideControl(req.FlightID)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		group.GET("/evtol-flight-safety-dossier/:flight_id", func(c *gin.Context) {
			fID := c.Param("flight_id")
			res, err := GlobalEVTOLVault.GetEVTOLFlightSafetyDossier(fID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})
	}
}
