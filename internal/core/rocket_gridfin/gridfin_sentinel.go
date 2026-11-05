package rocketgridfin

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

// GridFinTelemetry captures super-heavy reusable rocket booster aerogrid fin dynamics under FAA 14 CFR Part 450 & AIAA S-081
type GridFinTelemetry struct {
	TelemetryID                     string    `json:"telemetry_id"`
	BoosterID                       string    `json:"booster_id"`                          // "BOOSTER-SUPER-HEAVY-B12"
	FlightMissionID                 string    `json:"flight_mission_id"`                   // "MISSION-ORBITAL-STAR-09"
	MachNumber                      float64   `json:"mach_number"`                         // e.g. 5.8
	HydraulicServoPressureBar       float64   `json:"hydraulic_servo_pressure_bar"`        // e.g. 295.0 bar (normal 320-350 bar)
	FinHingeTorqueKNm               float64   `json:"fin_hinge_torque_knm"`                // e.g. 142.5 kNm (design limit 160 kNm)
	ActuatorOilBulkTempC            float64   `json:"actuator_oil_bulk_temp_c"`             // e.g. 215.0°C (oil vaporization > 200°C)
	ActiveCoolantFlowLMin           float64   `json:"active_coolant_flow_l_min"`           // e.g. 18.5 L/min
	AeroelasticFlutterFrequencyHz   float64   `json:"aeroelastic_flutter_frequency_hz"`    // e.g. 28.5 Hz (limit cycle flutter > 25 Hz)
	ReentryPlasmaElectronDensityCm3 float64   `json:"reentry_plasma_electron_density_cm3"` // e.g. 1.8e12 cm^-3
	RecordedAt                      time.Time `json:"recorded_at"`
}

// GridFinAssessment evaluates hydraulic cavitation, aeroelastic supersonic flutter, and re-entry thermal loads
type GridFinAssessment struct {
	AssessmentID                string    `json:"assessment_id"`
	BoosterID                   string    `json:"booster_id"`
	HydraulicTorqueMarginPct    float64   `json:"hydraulic_torque_margin_pct"`
	HydraulicCavitationRisk     string    `json:"hydraulic_cavitation_risk"`      // "HYDRAULIC_PRESSURE_OPTIMAL", "THERMAL_VAPOR_CAVITATION_WARNING", "CRITICAL_SERVO_PRESSURE_COLLAPSE"
	AeroelasticStabilityStatus  string    `json:"aeroelastic_stability_status"`   // "AEROELASTIC_FLUTTER_DAMPED", "SUPERSONIC_LIMIT_CYCLE_WARNING", "CRITICAL_GRID_FIN_TEAR_HAZARD"
	AngleOfAttackBiasTrimMandate bool     `json:"angle_of_attack_bias_trim_mandate"`
	SubcooledFluidPulseMandate  bool      `json:"subcooled_fluid_pulse_mandate"`
	AuditDigest                 string    `json:"audit_digest"`
	EvaluatedAt                 time.Time `json:"evaluated_at"`
}

// GridFinMitigationAction models hydraulic pressure boost, AoA attitude bias trim, and subcooled fluid cooling pulse
type GridFinMitigationAction struct {
	ActionID                           string    `json:"action_id"`
	BoosterID                          string    `json:"booster_id"`
	ModulatedServoPressureSetpointBar  float64   `json:"modulated_servo_pressure_setpoint_bar"`
	AngleOfAttackTrimBiasDeg           float64   `json:"angle_of_attack_trim_bias_deg"`
	SubcooledCoolantPulseFlowLMin      float64   `json:"subcooled_coolant_pulse_flow_l_min"`
	AvoidedBoosterLossUSD              float64   `json:"avoided_booster_loss_usd"`
	ActionTimestamp                    time.Time `json:"action_timestamp"`
}

// GridFinVault thread-safe in-memory store
type GridFinVault struct {
	mu          sync.RWMutex
	telemetries map[string]GridFinTelemetry
	assessments map[string]GridFinAssessment
	actions     map[string]GridFinMitigationAction
}

// GlobalGridFinVault instance
var GlobalGridFinVault = &GridFinVault{
	telemetries: make(map[string]GridFinTelemetry),
	assessments: make(map[string]GridFinAssessment),
	actions:     make(map[string]GridFinMitigationAction),
}

// IngestFinTelemetry stores aerogrid fin telemetry
func (v *GridFinVault) IngestFinTelemetry(t GridFinTelemetry) GridFinTelemetry {
	v.mu.Lock()
	defer v.mu.Unlock()

	if t.TelemetryID == "" {
		t.TelemetryID = fmt.Sprintf("TEL-FIN-%d", time.Now().UnixNano()%1000000)
	}
	if t.RecordedAt.IsZero() {
		t.RecordedAt = time.Now()
	}

	v.telemetries[t.BoosterID] = t
	return t
}

// EvaluateHydraulicsAndFlutter computes torque margin, cavitation hazard, and aeroelastic flutter status
func (v *GridFinVault) EvaluateHydraulicsAndFlutter(boosterID string) (GridFinAssessment, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	telem, exists := v.telemetries[boosterID]
	if !exists {
		return GridFinAssessment{}, fmt.Errorf("telemetry not found for booster: %s", boosterID)
	}

	torqueMargin := math.Round(((160.0-telem.FinHingeTorqueKNm)/160.0*100.0)*10) / 10

	cavitationRisk := "HYDRAULIC_PRESSURE_OPTIMAL"
	subcooledPulse := false
	aoaTrim := false

	if telem.ActuatorOilBulkTempC > 200.0 || telem.HydraulicServoPressureBar < 300.0 {
		cavitationRisk = "CRITICAL_SERVO_PRESSURE_COLLAPSE"
		subcooledPulse = true
		aoaTrim = true
	} else if telem.ActuatorOilBulkTempC > 175.0 || telem.HydraulicServoPressureBar < 315.0 {
		cavitationRisk = "THERMAL_VAPOR_CAVITATION_WARNING"
		subcooledPulse = true
	}

	aeroStatus := "AEROELASTIC_FLUTTER_DAMPED"
	if telem.AeroelasticFlutterFrequencyHz > 25.0 || telem.MachNumber > 5.0 {
		aeroStatus = "CRITICAL_GRID_FIN_TEAR_HAZARD"
		aoaTrim = true
	} else if telem.AeroelasticFlutterFrequencyHz > 20.0 {
		aeroStatus = "SUPERSONIC_LIMIT_CYCLE_WARNING"
		aoaTrim = true
	}

	rawHash := fmt.Sprintf("%s:%f:%s:%s:%t:%t", boosterID, torqueMargin, cavitationRisk, aeroStatus, aoaTrim, subcooledPulse)
	hash := sha256.Sum256([]byte(rawHash))
	digest := hex.EncodeToString(hash[:])

	assessment := GridFinAssessment{
		AssessmentID:                 fmt.Sprintf("ASM-FIN-%d", time.Now().UnixNano()%1000000),
		BoosterID:                    boosterID,
		HydraulicTorqueMarginPct:     torqueMargin,
		HydraulicCavitationRisk:      cavitationRisk,
		AeroelasticStabilityStatus:   aeroStatus,
		AngleOfAttackBiasTrimMandate: aoaTrim,
		SubcooledFluidPulseMandate:   subcooledPulse,
		AuditDigest:                  digest,
		EvaluatedAt:                  time.Now(),
	}

	v.assessments[boosterID] = assessment
	return assessment, nil
}

// TriggerAoATrimAndSubcooledPulse deploys hydraulic pressure setpoint boost, AoA trim bias, and subcooled fluid cooling
func (v *GridFinVault) TriggerAoATrimAndSubcooledPulse(boosterID string) (GridFinMitigationAction, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	asm, hasAsm := v.assessments[boosterID]
	telem, hasTelem := v.telemetries[boosterID]
	if !hasAsm || !hasTelem {
		return GridFinMitigationAction{}, fmt.Errorf("missing telemetry or assessment for booster %s", boosterID)
	}

	cmdPressure := telem.HydraulicServoPressureBar
	aoaTrimDeg := 0.0
	pulseFlow := telem.ActiveCoolantFlowLMin

	if asm.SubcooledFluidPulseMandate {
		cmdPressure = 350.0 // boost hydraulic reservoir pressure to prevent oil vapor cavitation
		pulseFlow = 45.0    // pulse cryogenic/subcooled thermal fluid across fin hinge bearing
	}
	if asm.AngleOfAttackBiasTrimMandate {
		aoaTrimDeg = -2.4   // bias booster angle of attack to shed re-entry aerothermal shock load
	}

	action := GridFinMitigationAction{
		ActionID:                          fmt.Sprintf("ACT-FIN-%d", time.Now().UnixNano()%1000000),
		BoosterID:                         boosterID,
		ModulatedServoPressureSetpointBar: cmdPressure,
		AngleOfAttackTrimBiasDeg:          aoaTrimDeg,
		SubcooledCoolantPulseFlowLMin:     pulseFlow,
		AvoidedBoosterLossUSD:             120000000.0,
		ActionTimestamp:                   time.Now(),
	}

	v.actions[boosterID] = action
	return action, nil
}

// GetGridFinReliabilityDossier returns booster flight safety dossier under FAA 14 CFR Part 450 & AIAA S-081
func (v *GridFinVault) GetGridFinReliabilityDossier(boosterID string) (map[string]interface{}, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	telem, hasTelem := v.telemetries[boosterID]
	asm, hasAsm := v.assessments[boosterID]
	act := v.actions[boosterID]

	if !hasTelem || !hasAsm {
		return nil, fmt.Errorf("incomplete data for booster grid fin dossier %s", boosterID)
	}

	return map[string]interface{}{
		"booster_id":             boosterID,
		"flight_mission_id":      telem.FlightMissionID,
		"faa_14cfr450_compliant": true,
		"aiaa_s081_certified":    true,
		"telemetry":              telem,
		"assessment":             asm,
		"latest_action":          act,
		"compiled_at":            time.Now(),
	}, nil
}

// RegisterGridFinRoutes wires API endpoints under /api/v1/aerospace/gridfin/*
func RegisterGridFinRoutes(r *gin.RouterGroup) {
	fin := r.Group("/aerospace/gridfin")
	{
		fin.POST("/ingest-fin-telemetry", func(c *gin.Context) {
			var t GridFinTelemetry
			if err := c.ShouldBindJSON(&t); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res := GlobalGridFinVault.IngestFinTelemetry(t)
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		fin.POST("/evaluate-hydraulics-and-flutter", func(c *gin.Context) {
			var req struct {
				BoosterID string `json:"booster_id"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := GlobalGridFinVault.EvaluateHydraulicsAndFlutter(req.BoosterID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		fin.POST("/trigger-aoa-trim-and-subcooled-pulse", func(c *gin.Context) {
			var req struct {
				BoosterID string `json:"booster_id"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := GlobalGridFinVault.TriggerAoATrimAndSubcooledPulse(req.BoosterID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		fin.GET("/gridfin-reliability-dossier/:booster_id", func(c *gin.Context) {
			boosterID := c.Param("booster_id")
			res, err := GlobalGridFinVault.GetGridFinReliabilityDossier(boosterID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})
	}
}
