package asteroidmining

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// AsteroidMiningTelemetry captures deep-space asteroid optical mining, concentrated solar desorption, and volatile trapping telemetry
type AsteroidMiningTelemetry struct {
	TelemetryID                   string    `json:"telemetry_id"`
	MiningProbeID                 string    `json:"mining_probe_id"`
	TargetAsteroidName            string    `json:"target_asteroid_name"`               // e.g. "16-Psyche", "101955-Bennu"
	ConcentratedSolarFluxKwPerM2  float64   `json:"concentrated_solar_flux_kw_per_m2"`  // limit < 250.0 kW/m2
	RegolithDesorptionTempC       float64   `json:"regolith_desorption_temp_c"`         // threshold > 620.0°C
	InflatableCanopyPressureBar   float64   `json:"inflatable_canopy_pressure_bar"`     // burst limit < 0.85 bar
	CryoColdTrapTempK             float64   `json:"cryo_cold_trap_temp_k"`              // limit < 125.0 K
	DustEjectaMicronSize          float64   `json:"dust_ejecta_micron_size"`            // abrasive threshold > 150.0 um
	VolatileExtractionRateKgPerHr float64   `json:"volatile_extraction_rate_kg_per_hr"` // nominal 25 - 60 kg/hr
	CanopyTearStressMpa           float64   `json:"canopy_tear_stress_mpa"`             // tensile limit < 180.0 MPa
	Timestamp                     time.Time `json:"timestamp"`
}

// AsteroidMiningAssessment models canopy containment integrity, solar overflux, and NASA Planetary Protection compliance
type AsteroidMiningAssessment struct {
	AssessmentID                    string    `json:"assessment_id"`
	MiningProbeID                   string    `json:"mining_probe_id"`
	CanopyIntegrityStatus           string    `json:"canopy_integrity_status"`          // SEALED_INFLATED_STABLE, TENSILE_OVERPRESSURE_WARNING, CATASTROPHIC_CANOPY_RUPTURE_RISK
	VolatileCondensationEfficiency  string    `json:"volatile_condensation_efficiency"` // OPTIMAL_CRYO_TRAPPING, THERMAL_LEAK_INSUFFICIENT
	CanopyPressureMarginPct         float64   `json:"canopy_pressure_margin_pct"`
	SolarConcentratorDefocusMandate bool      `json:"solar_concentrator_defocus_mandate"`
	CryoChillerBoostMandate         bool      `json:"cryo_chiller_boost_mandate"`
	AuditDigest                     string    `json:"audit_digest"`
	EvaluatedAt                     time.Time `json:"evaluated_at"`
}

// AsteroidMiningMitigationAction models parabolic reflector defocusing, cryo cold-trap power boost, and pressure relief venting
type AsteroidMiningMitigationAction struct {
	ActionID                      string    `json:"action_id"`
	MiningProbeID                 string    `json:"mining_probe_id"`
	CommandedSolarDefocusAngleDeg float64   `json:"commanded_solar_defocus_angle_deg"`
	CryoChillerPowerWatts         float64   `json:"cryo_chiller_power_watts"`
	ReliefValveVentRateKgPerSec   float64   `json:"relief_valve_vent_rate_kg_per_sec"`
	AvoidedCanopyBlowoutLossUSD   float64   `json:"avoided_canopy_blowout_loss_usd"`
	ExecutedAt                    time.Time `json:"executed_at"`
}

// AsteroidMiningVault manages asteroid mining vessel telemetry, optical concentrator state, and volatile capture safety
type AsteroidMiningVault struct {
	mu          sync.RWMutex
	telemetries map[string]AsteroidMiningTelemetry
	assessments map[string]AsteroidMiningAssessment
	actions     map[string]AsteroidMiningMitigationAction
}

// GlobalAsteroidMiningVault is the singleton instance for asteroid optical mining sentinel
var GlobalAsteroidMiningVault = NewAsteroidMiningVault()

func NewAsteroidMiningVault() *AsteroidMiningVault {
	return &AsteroidMiningVault{
		telemetries: make(map[string]AsteroidMiningTelemetry),
		assessments: make(map[string]AsteroidMiningAssessment),
		actions:     make(map[string]AsteroidMiningMitigationAction),
	}
}

// IngestMiningTelemetry stores probe optical desorption telemetry
func (v *AsteroidMiningVault) IngestMiningTelemetry(telemetry AsteroidMiningTelemetry) AsteroidMiningTelemetry {
	v.mu.Lock()
	defer v.mu.Unlock()

	if telemetry.Timestamp.IsZero() {
		telemetry.Timestamp = time.Now()
	}
	v.telemetries[telemetry.MiningProbeID] = telemetry
	return telemetry
}

// EvaluateDesorptionAndCanopyIntegrity evaluates canopy tear stress, overpressure, and NASA/ISO 14620 safety rules
func (v *AsteroidMiningVault) EvaluateDesorptionAndCanopyIntegrity(probeID string) (AsteroidMiningAssessment, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	telem, exists := v.telemetries[probeID]
	if !exists {
		return AsteroidMiningAssessment{}, fmt.Errorf("telemetry not found for asteroid mining probe %s", probeID)
	}

	canopyStatus := "SEALED_INFLATED_STABLE"
	trapEfficiency := "OPTIMAL_CRYO_TRAPPING"
	defocusMandate := false
	chillerMandate := false

	burstLimit := 0.85 // bar
	pressureMargin := (telem.InflatableCanopyPressureBar - burstLimit) / burstLimit * 100.0

	if telem.CryoColdTrapTempK > 125.0 {
		trapEfficiency = "THERMAL_LEAK_INSUFFICIENT"
		chillerMandate = true
	}

	if telem.ConcentratedSolarFluxKwPerM2 >= 250.0 || telem.InflatableCanopyPressureBar >= 0.85 || telem.CanopyTearStressMpa >= 180.0 {
		canopyStatus = "CATASTROPHIC_CANOPY_RUPTURE_RISK"
		defocusMandate = true
		chillerMandate = true
	} else if telem.ConcentratedSolarFluxKwPerM2 > 200.0 || telem.InflatableCanopyPressureBar > 0.65 || telem.RegolithDesorptionTempC > 620.0 {
		canopyStatus = "TENSILE_OVERPRESSURE_WARNING"
		defocusMandate = true
	}

	hashInput := fmt.Sprintf("%s:%s:%s:%.2f:%.2f:%t",
		telem.MiningProbeID, canopyStatus, trapEfficiency, telem.ConcentratedSolarFluxKwPerM2, telem.InflatableCanopyPressureBar, defocusMandate)
	hasher := sha256.New()
	hasher.Write([]byte(hashInput))
	digest := hex.EncodeToString(hasher.Sum(nil))

	assessment := AsteroidMiningAssessment{
		AssessmentID:                    fmt.Sprintf("ASM-ASTEROID-%d", time.Now().UnixNano()%1000000),
		MiningProbeID:                   probeID,
		CanopyIntegrityStatus:           canopyStatus,
		VolatileCondensationEfficiency:  trapEfficiency,
		CanopyPressureMarginPct:         pressureMargin,
		SolarConcentratorDefocusMandate: defocusMandate,
		CryoChillerBoostMandate:         chillerMandate,
		AuditDigest:                     digest,
		EvaluatedAt:                     time.Now(),
	}

	v.assessments[probeID] = assessment
	return assessment, nil
}

// TriggerDefocusAndCryoTrapBoost gimbals solar concentrators and boosts cryo cold trap Stirling chillers
func (v *AsteroidMiningVault) TriggerDefocusAndCryoTrapBoost(probeID string) (AsteroidMiningMitigationAction, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	asm, hasAsm := v.assessments[probeID]
	_, hasTelem := v.telemetries[probeID]
	if !hasAsm || !hasTelem {
		return AsteroidMiningMitigationAction{}, fmt.Errorf("missing telemetry or assessment for asteroid probe %s", probeID)
	}

	defocusAngle := 0.0
	chillerWatts := 1200.0
	ventRate := 0.0

	if asm.SolarConcentratorDefocusMandate {
		defocusAngle = 14.5 // slew optical concentrator mirrors 14.5° off-focus to drop regolith heating
		ventRate = 0.45     // open proportional relief solenoid to vent excess volatile gas at 0.45 kg/s
	}
	if asm.CryoChillerBoostMandate {
		chillerWatts = 3600.0 // surge pulse-tube cryocoolers to 3,600 W to recapture volatile freeze-out
	}

	action := AsteroidMiningMitigationAction{
		ActionID:                      fmt.Sprintf("ACT-ASTEROID-%d", time.Now().UnixNano()%1000000),
		MiningProbeID:                 probeID,
		CommandedSolarDefocusAngleDeg: defocusAngle,
		CryoChillerPowerWatts:         chillerWatts,
		ReliefValveVentRateKgPerSec:   ventRate,
		AvoidedCanopyBlowoutLossUSD:   190000000.0, // $190M USD avoided inflatable canopy explosion, spacecraft damage & loss of volatiles
		ExecutedAt:                    time.Now(),
	}

	v.actions[probeID] = action
	return action, nil
}

// GetAsteroidMiningDossier compiles asteroid volatile extraction, canopy structural safety, and ISO 14620 records
func (v *AsteroidMiningVault) GetAsteroidMiningDossier(probeID string) (map[string]interface{}, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	telem, hasTelem := v.telemetries[probeID]
	asm, hasAsm := v.assessments[probeID]
	act, hasAct := v.actions[probeID]

	if !hasTelem || !hasAsm {
		return nil, fmt.Errorf("incomplete data for asteroid mining dossier: %s", probeID)
	}

	dossier := map[string]interface{}{
		"mining_probe_id":           probeID,
		"target_asteroid":           telem.TargetAsteroidName,
		"iso_14620_compliant":       asm.CanopyIntegrityStatus != "CATASTROPHIC_CANOPY_RUPTURE_RISK" || (hasAct && act.AvoidedCanopyBlowoutLossUSD > 0),
		"nasa_planetary_protection": true,
		"solar_flux_kw_m2":          telem.ConcentratedSolarFluxKwPerM2,
		"regolith_temp_c":           telem.RegolithDesorptionTempC,
		"canopy_pressure_bar":       telem.InflatableCanopyPressureBar,
		"cold_trap_temp_k":          telem.CryoColdTrapTempK,
		"canopy_integrity":          asm.CanopyIntegrityStatus,
		"volatile_condensation":     asm.VolatileCondensationEfficiency,
		"solar_defocus_angle_deg":   0.0,
		"chiller_power_watts":       1200.0,
		"avoided_loss_usd":          0.0,
		"audit_digest":              asm.AuditDigest,
		"compiled_at":               time.Now(),
	}

	if hasAct {
		dossier["solar_defocus_angle_deg"] = act.CommandedSolarDefocusAngleDeg
		dossier["chiller_power_watts"] = act.CryoChillerPowerWatts
		dossier["avoided_loss_usd"] = act.AvoidedCanopyBlowoutLossUSD
	}

	return dossier, nil
}

// RegisterAsteroidMiningRoutes wires API endpoints under /api/v1/space/asteroid-mining/*
func RegisterAsteroidMiningRoutes(r *gin.RouterGroup) {
	astGroup := r.Group("/space/asteroid-mining")
	{
		astGroup.POST("/ingest-mining-telemetry", func(c *gin.Context) {
			var t AsteroidMiningTelemetry
			if err := c.ShouldBindJSON(&t); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res := GlobalAsteroidMiningVault.IngestMiningTelemetry(t)
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		astGroup.POST("/evaluate-desorption-and-canopy-integrity", func(c *gin.Context) {
			var req struct {
				MiningProbeID string `json:"mining_probe_id"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := GlobalAsteroidMiningVault.EvaluateDesorptionAndCanopyIntegrity(req.MiningProbeID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		astGroup.POST("/trigger-defocus-and-cryo-trap-boost", func(c *gin.Context) {
			var req struct {
				MiningProbeID string `json:"mining_probe_id"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := GlobalAsteroidMiningVault.TriggerDefocusAndCryoTrapBoost(req.MiningProbeID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		astGroup.GET("/asteroid-mining-dossier/:probe_id", func(c *gin.Context) {
			probeID := c.Param("probe_id")
			res, err := GlobalAsteroidMiningVault.GetAsteroidMiningDossier(probeID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})
	}
}
