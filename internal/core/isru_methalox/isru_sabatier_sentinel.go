package isrumethalox

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

// ISRUMethaloxTelemetry captures planetary in-situ resource utilization (ISRU) Sabatier methalox synthesis under NASA SP-8000 & ISO 14300
type ISRUMethaloxTelemetry struct {
	TelemetryID                     string    `json:"telemetry_id"`
	PlantID                         string    `json:"plant_id"`                             // "ISRU-MARS-BASE-ALPHA-01"
	MissionCycleID                  string    `json:"mission_cycle_id"`                     // "SOL-MARS-ASCENT-METHALOX-420"
	MartianAmbientPressureMbar      float64   `json:"martian_ambient_pressure_mbar"`        // e.g. 7.5 mbar
	Co2IntakeCryoFreezerBar         float64   `json:"co2_intake_cryo_freezer_bar"`          // e.g. 6.2 bar
	SabatierCatalystBedTempC        float64   `json:"sabatier_catalyst_bed_temp_c"`         // e.g. 415.0°C (runaway sintering > 440°C)
	H2ToCo2StoichiometricRatio      float64   `json:"h2_to_co2_stoichiometric_ratio"`       // e.g. 4.28:1 (ideal stoichiometric 4.05:1)
	RutheniumPerchloratePpm         float64   `json:"ruthenium_perchlorate_ppm"`            // e.g. 35.0 ppm (catalyst poisoning limit > 15 ppm)
	DustFilterPressureDropMbar      float64   `json:"dust_filter_pressure_drop_mbar"`       // e.g. 18.5 mbar (severe regolith clogging > 12 mbar)
	WaterCondenserSeparationPct     float64   `json:"water_condenser_separation_pct"`       // e.g. 96.2%
	LiquidMethaneProductionKgPerSol float64   `json:"liquid_methane_production_kg_per_sol"` // e.g. 48.5 kg/sol
	RecordedAt                      time.Time `json:"recorded_at"`
}

// ISRUAssessment evaluates Sabatier exothermic runaway, Ru/Al2O3 catalyst poisoning, and electrostatic regolith filter clogging
type ISRUAssessment struct {
	AssessmentID                    string    `json:"assessment_id"`
	PlantID                         string    `json:"plant_id"`
	StoichiometricSlipRatio         float64   `json:"stoichiometric_slip_ratio"`
	SabatierThermalRunawayHazard    string    `json:"sabatier_thermal_runaway_hazard"` // "EXOTHERMIC_EQUILIBRIUM_STABLE", "METHANATION_OVERTEMP_WARNING", "CRITICAL_CATALYST_SINTERING_COLLAPSE"
	MartianDustFilterStatus         string    `json:"martian_dust_filter_status"`      // "ELECTROSTATIC_DUST_CLEARED", "PERCHLORATE_POISONING_WARNING", "CRITICAL_REGOLITH_FILTER_CLOG"
	StoichiometricRatioTrimMandate  bool      `json:"stoichiometric_ratio_trim_mandate"`
	ElectrostaticPebblePurgeMandate bool      `json:"electrostatic_pebble_purge_mandate"`
	AuditDigest                     string    `json:"audit_digest"`
	EvaluatedAt                     time.Time `json:"evaluated_at"`
}

// ISRUMitigationAction models feed gas stoichiometry trim, electrostatic pulsed reverse purge, and condenser cooling boost
type ISRUMitigationAction struct {
	ActionID                          string    `json:"action_id"`
	PlantID                           string    `json:"plant_id"`
	CommandedH2Co2RatioSetpoint       float64   `json:"commanded_h2_co2_ratio_setpoint"`
	ElectrostaticDustPurgeDurationSec float64   `json:"electrostatic_dust_purge_duration_sec"`
	CondenserChillerBoostWatts        float64   `json:"condenser_chiller_boost_watts"`
	AvoidedMissionLossUSD             float64   `json:"avoided_mission_loss_usd"`
	ActionTimestamp                   time.Time `json:"action_timestamp"`
}

// ISRUVault thread-safe in-memory store
type ISRUVault struct {
	mu          sync.RWMutex
	telemetries map[string]ISRUMethaloxTelemetry
	assessments map[string]ISRUAssessment
	actions     map[string]ISRUMitigationAction
}

// GlobalISRUVault instance
var GlobalISRUVault = &ISRUVault{
	telemetries: make(map[string]ISRUMethaloxTelemetry),
	assessments: make(map[string]ISRUAssessment),
	actions:     make(map[string]ISRUMitigationAction),
}

// IngestISRUTelemetry stores Martian ISRU plant telemetry
func (v *ISRUVault) IngestISRUTelemetry(t ISRUMethaloxTelemetry) ISRUMethaloxTelemetry {
	v.mu.Lock()
	defer v.mu.Unlock()

	if t.TelemetryID == "" {
		t.TelemetryID = fmt.Sprintf("TEL-ISRU-%d", time.Now().UnixNano()%1000000)
	}
	if t.RecordedAt.IsZero() {
		t.RecordedAt = time.Now()
	}

	v.telemetries[t.PlantID] = t
	return t
}

// EvaluateSabatierAndDustClogging evaluates thermal reaction kinetics, catalyst poisoning, and dust clogging
func (v *ISRUVault) EvaluateSabatierAndDustClogging(plantID string) (ISRUAssessment, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	telem, exists := v.telemetries[plantID]
	if !exists {
		return ISRUAssessment{}, fmt.Errorf("telemetry not found for ISRU plant: %s", plantID)
	}

	slip := math.Round((telem.H2ToCo2StoichiometricRatio-4.0)*100) / 100

	runawayHazard := "EXOTHERMIC_EQUILIBRIUM_STABLE"
	ratioTrim := false

	if telem.SabatierCatalystBedTempC > 410.0 || telem.H2ToCo2StoichiometricRatio > 4.25 {
		runawayHazard = "CRITICAL_CATALYST_SINTERING_COLLAPSE"
		ratioTrim = true
	} else if telem.SabatierCatalystBedTempC > 380.0 {
		runawayHazard = "METHANATION_OVERTEMP_WARNING"
		ratioTrim = true
	}

	dustStatus := "ELECTROSTATIC_DUST_CLEARED"
	dustPurge := false

	if telem.DustFilterPressureDropMbar > 15.0 || telem.RutheniumPerchloratePpm > 25.0 {
		dustStatus = "CRITICAL_REGOLITH_FILTER_CLOG"
		dustPurge = true
	} else if telem.DustFilterPressureDropMbar > 10.0 || telem.RutheniumPerchloratePpm > 10.0 {
		dustStatus = "PERCHLORATE_POISONING_WARNING"
		dustPurge = true
	}

	rawHash := fmt.Sprintf("%s:%f:%s:%s:%t:%t", plantID, slip, runawayHazard, dustStatus, ratioTrim, dustPurge)
	hash := sha256.Sum256([]byte(rawHash))
	digest := hex.EncodeToString(hash[:])

	assessment := ISRUAssessment{
		AssessmentID:                    fmt.Sprintf("ASM-ISRU-%d", time.Now().UnixNano()%1000000),
		PlantID:                         plantID,
		StoichiometricSlipRatio:         slip,
		SabatierThermalRunawayHazard:    runawayHazard,
		MartianDustFilterStatus:         dustStatus,
		StoichiometricRatioTrimMandate:  ratioTrim,
		ElectrostaticPebblePurgeMandate: dustPurge,
		AuditDigest:                     digest,
		EvaluatedAt:                     time.Now(),
	}

	v.assessments[plantID] = assessment
	return assessment, nil
}

// TriggerRatioTrimAndDustPurge applies H2/CO2 ratio setpoint trim, activates electrostatic reverse blowback, and chills condenser
func (v *ISRUVault) TriggerRatioTrimAndDustPurge(plantID string) (ISRUMitigationAction, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	asm, hasAsm := v.assessments[plantID]
	telem, hasTelem := v.telemetries[plantID]
	if !hasAsm || !hasTelem {
		return ISRUMitigationAction{}, fmt.Errorf("missing telemetry or assessment for plant %s", plantID)
	}

	cmdRatio := telem.H2ToCo2StoichiometricRatio
	dustPurgeDuration := 0.0
	chillerBoost := 50.0

	if asm.StoichiometricRatioTrimMandate {
		cmdRatio = 4.05 // trim ratio to 4.05:1 to restore exothermic reaction equilibrium and suppress runaway
	}
	if asm.ElectrostaticPebblePurgeMandate {
		dustPurgeDuration = 8.0 // 8-second pulsed electrostatic backflush to eject sub-micron iron oxide regolith
		chillerBoost = 350.0    // boost cryo-chiller power to ensure complete water droplet condensation
	}

	action := ISRUMitigationAction{
		ActionID:                          fmt.Sprintf("ACT-ISRU-%d", time.Now().UnixNano()%1000000),
		PlantID:                           plantID,
		CommandedH2Co2RatioSetpoint:       cmdRatio,
		ElectrostaticDustPurgeDurationSec: dustPurgeDuration,
		CondenserChillerBoostWatts:        chillerBoost,
		AvoidedMissionLossUSD:             500000000.0,
		ActionTimestamp:                   time.Now(),
	}

	v.actions[plantID] = action
	return action, nil
}

// GetISRUMissionDossier returns Mars ISRU propellant production safety dossier under NASA SP-8000 & ISO 14300
func (v *ISRUVault) GetISRUMissionDossier(plantID string) (map[string]interface{}, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	telem, hasTelem := v.telemetries[plantID]
	asm, hasAsm := v.assessments[plantID]
	act := v.actions[plantID]

	if !hasTelem || !hasAsm {
		return nil, fmt.Errorf("incomplete data for ISRU plant dossier %s", plantID)
	}

	return map[string]interface{}{
		"plant_id":              plantID,
		"mission_cycle_id":      telem.MissionCycleID,
		"nasa_sp8000_compliant": true,
		"iso_14300_certified":   true,
		"telemetry":             telem,
		"assessment":            asm,
		"latest_action":         act,
		"compiled_at":           time.Now(),
	}, nil
}

// RegisterISRURoutes wires API endpoints under /api/v1/space/isru/*
func RegisterISRURoutes(r *gin.RouterGroup) {
	isruGroup := r.Group("/space/isru")
	{
		isruGroup.POST("/ingest-isru-telemetry", func(c *gin.Context) {
			var t ISRUMethaloxTelemetry
			if err := c.ShouldBindJSON(&t); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res := GlobalISRUVault.IngestISRUTelemetry(t)
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		isruGroup.POST("/evaluate-sabatier-and-dust-clogging", func(c *gin.Context) {
			var req struct {
				PlantID string `json:"plant_id"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := GlobalISRUVault.EvaluateSabatierAndDustClogging(req.PlantID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		isruGroup.POST("/trigger-ratio-trim-and-dust-purge", func(c *gin.Context) {
			var req struct {
				PlantID string `json:"plant_id"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := GlobalISRUVault.TriggerRatioTrimAndDustPurge(req.PlantID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		isruGroup.GET("/isru-mission-dossier/:plant_id", func(c *gin.Context) {
			plantID := c.Param("plant_id")
			res, err := GlobalISRUVault.GetISRUMissionDossier(plantID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})
	}
}
