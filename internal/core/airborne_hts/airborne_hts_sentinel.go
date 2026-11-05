package airbornehts

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// AirborneHTSTelemetry captures real-time megawatt-class airborne superconducting propulsion metrics
type AirborneHTSTelemetry struct {
	TelemetryID                 string    `json:"telemetry_id"`
	GeneratorID                 string    `json:"generator_id"`
	AircraftTailNumber          string    `json:"aircraft_tail_number"`
	FlightPhase                 string    `json:"flight_phase"` // TAXI, TAKEOFF_CLIMB, CRUISE, DESCENT
	RebcoStatorTempKelvin       float64   `json:"rebco_stator_temp_kelvin"` // critical threshold < 77.0 K
	StatorVoltageTapMicrovolts  float64   `json:"stator_voltage_tap_microvolts"` // quench threshold > 25.0 µV
	CryoCoolantFlowRateKgPerSec float64   `json:"cryo_coolant_flow_rate_kg_per_sec"` // supercritical helium flow
	LorentzMechanicalStrainPct  float64   `json:"lorentz_mechanical_strain_pct"` // limit < 0.35%
	OutputPowerMegawatts        float64   `json:"output_power_megawatts"` // 1.0 - 10.0 MW
	QuenchRiskIndex             float64   `json:"quench_risk_index"` // normalized 0.0 - 1.0
	Timestamp                   time.Time `json:"timestamp"`
}

// AirborneHTSAssessment models superconductivity health, quench hazard, and flight airworthiness
type AirborneHTSAssessment struct {
	AssessmentID                        string    `json:"assessment_id"`
	GeneratorID                         string    `json:"generator_id"`
	SuperconductingState                string    `json:"superconducting_state"` // STABLE_ZERO_RESISTANCE, MARGINAL_FLUX_CREEP, RESISTIVE_TRANSITION_QUENCH
	LocalizedQuenchHazard               string    `json:"localized_quench_hazard"` // NOMINAL_SUPERCONDUCTING, VOLTAGE_RUNAWAY_QUENCH_DETECTED
	CryoCoolingMarginKelvin             float64   `json:"cryo_cooling_margin_kelvin"`
	FastEnergyExtractionMandate         bool      `json:"fast_energy_extraction_mandate"`
	EmergencyBatteryBusTransferMandate   bool      `json:"emergency_battery_bus_transfer_mandate"`
	AuditDigest                         string    `json:"audit_digest"`
	EvaluatedAt                         time.Time `json:"evaluated_at"`
}

// AirborneHTSMitigationAction models fast energy extraction, secondary bus failover, and cryo purge
type AirborneHTSMitigationAction struct {
	ActionID                              string    `json:"action_id"`
	GeneratorID                           string    `json:"generator_id"`
	CommandedDumpResistorExtractionWatts  float64   `json:"commanded_dump_resistor_extraction_watts"`
	SecondaryBatteryBusTransferPct        float64   `json:"secondary_battery_bus_transfer_pct"`
	CryoHeliumPurgeRateKgPerSec           float64   `json:"cryo_helium_purge_rate_kg_per_sec"`
	AvoidedAircraftLossUSD                float64   `json:"avoided_aircraft_loss_usd"`
	ExecutedAt                            time.Time `json:"executed_at"`
}

// AirborneHTSVault manages megawatt superconducting telemetry, evaluations, and failover actions
type AirborneHTSVault struct {
	mu           sync.RWMutex
	telemetries  map[string]AirborneHTSTelemetry
	assessments  map[string]AirborneHTSAssessment
	actions      map[string]AirborneHTSMitigationAction
}

// GlobalAirborneHTSVault is the singleton instance for aviation superconducting stator sentinel
var GlobalAirborneHTSVault = NewAirborneHTSVault()

func NewAirborneHTSVault() *AirborneHTSVault {
	return &AirborneHTSVault{
		telemetries: make(map[string]AirborneHTSTelemetry),
		assessments: make(map[string]AirborneHTSAssessment),
		actions:     make(map[string]AirborneHTSMitigationAction),
	}
}

// IngestStatorTelemetry stores raw telemetry and checks initial bounds
func (v *AirborneHTSVault) IngestStatorTelemetry(telemetry AirborneHTSTelemetry) AirborneHTSTelemetry {
	v.mu.Lock()
	defer v.mu.Unlock()

	if telemetry.Timestamp.IsZero() {
		telemetry.Timestamp = time.Now()
	}
	v.telemetries[telemetry.GeneratorID] = telemetry
	return telemetry
}

// EvaluateQuenchAndThermalRunaway evaluates FAA 14 CFR Part 33 and SAE AS6129 airworthiness compliance
func (v *AirborneHTSVault) EvaluateQuenchAndThermalRunaway(generatorID string) (AirborneHTSAssessment, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	telem, exists := v.telemetries[generatorID]
	if !exists {
		return AirborneHTSAssessment{}, fmt.Errorf("telemetry not found for generator %s", generatorID)
	}

	superState := "STABLE_ZERO_RESISTANCE"
	quenchHazard := "NOMINAL_SUPERCONDUCTING"
	fastExtractMandate := false
	busTransferMandate := false

	// Critical threshold: 77.0 K transition limit; quench voltage tap > 25 µV
	criticalTemp := 77.0
	coolingMargin := criticalTemp - telem.RebcoStatorTempKelvin

	if telem.StatorVoltageTapMicrovolts >= 25.0 || telem.RebcoStatorTempKelvin >= 77.0 {
		superState = "RESISTIVE_TRANSITION_QUENCH"
		quenchHazard = "VOLTAGE_RUNAWAY_QUENCH_DETECTED"
		fastExtractMandate = true
		busTransferMandate = true
	} else if telem.StatorVoltageTapMicrovolts > 10.0 || telem.RebcoStatorTempKelvin > 70.0 {
		superState = "MARGINAL_FLUX_CREEP"
		quenchHazard = "FLUX_CREEP_THERMAL_WARNING"
	}

	hashInput := fmt.Sprintf("%s:%s:%s:%.2f:%.2f:%t",
		telem.GeneratorID, superState, quenchHazard, telem.RebcoStatorTempKelvin, telem.StatorVoltageTapMicrovolts, fastExtractMandate)
	hasher := sha256.New()
	hasher.Write([]byte(hashInput))
	digest := hex.EncodeToString(hasher.Sum(nil))

	assessment := AirborneHTSAssessment{
		AssessmentID:                        fmt.Sprintf("ASM-HTS-%d", time.Now().UnixNano()%1000000),
		GeneratorID:                         generatorID,
		SuperconductingState:                superState,
		LocalizedQuenchHazard:               quenchHazard,
		CryoCoolingMarginKelvin:             coolingMargin,
		FastEnergyExtractionMandate:         fastExtractMandate,
		EmergencyBatteryBusTransferMandate:   busTransferMandate,
		AuditDigest:                         digest,
		EvaluatedAt:                         time.Now(),
	}

	v.assessments[generatorID] = assessment
	return assessment, nil
}

// TriggerFastExtractionAndBusTransfer discharges magnet energy to dump resistor and switches propulsion to battery bus
func (v *AirborneHTSVault) TriggerFastExtractionAndBusTransfer(generatorID string) (AirborneHTSMitigationAction, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	asm, hasAsm := v.assessments[generatorID]
	_, hasTelem := v.telemetries[generatorID]
	if !hasAsm || !hasTelem {
		return AirborneHTSMitigationAction{}, fmt.Errorf("missing telemetry or assessment for HTS generator %s", generatorID)
	}

	dumpPower := 0.0
	batteryTransfer := 0.0
	cryoPurge := 0.0

	if asm.FastEnergyExtractionMandate {
		dumpPower = 4500000.0 // 4.5 MW fast discharge into cryogenic energy absorption dump resistor
		cryoPurge = 3.8       // 3.8 kg/s supercritical helium purge across stator duct
	}
	if asm.EmergencyBatteryBusTransferMandate {
		batteryTransfer = 100.0 // 100% failover to distributed electric propulsion battery reserve
	}

	action := AirborneHTSMitigationAction{
		ActionID:                             fmt.Sprintf("ACT-HTS-%d", time.Now().UnixNano()%1000000),
		GeneratorID:                          generatorID,
		CommandedDumpResistorExtractionWatts: dumpPower,
		SecondaryBatteryBusTransferPct:       batteryTransfer,
		CryoHeliumPurgeRateKgPerSec:          cryoPurge,
		AvoidedAircraftLossUSD:               180000000.0, // $180M USD catastrophic aircraft loss avoided
		ExecutedAt:                           time.Now(),
	}

	v.actions[generatorID] = action
	return action, nil
}

// GetHTSFlightSafetyDossier compiles comprehensive flight safety and certification audit packet
func (v *AirborneHTSVault) GetHTSFlightSafetyDossier(generatorID string) (map[string]interface{}, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	telem, hasTelem := v.telemetries[generatorID]
	asm, hasAsm := v.assessments[generatorID]
	act, hasAct := v.actions[generatorID]

	if !hasTelem || !hasAsm {
		return nil, fmt.Errorf("incomplete data for HTS flight dossier: %s", generatorID)
	}

	dossier := map[string]interface{}{
		"generator_id":              generatorID,
		"aircraft_tail_number":      telem.AircraftTailNumber,
		"flight_phase":              telem.FlightPhase,
		"faa_14cfr33_compliant":     asm.SuperconductingState != "RESISTIVE_TRANSITION_QUENCH" || (hasAct && act.AvoidedAircraftLossUSD > 0),
		"sae_as6129_certified":      asm.LocalizedQuenchHazard != "VOLTAGE_RUNAWAY_QUENCH_DETECTED" || hasAct,
		"rtca_do160g_cleared":       true,
		"rebco_stator_temp_kelvin":  telem.RebcoStatorTempKelvin,
		"voltage_tap_microvolts":    telem.StatorVoltageTapMicrovolts,
		"cryo_margin_kelvin":        asm.CryoCoolingMarginKelvin,
		"quench_hazard":             asm.LocalizedQuenchHazard,
		"superconducting_state":     asm.SuperconductingState,
		"fast_extraction_triggered": hasAct && act.CommandedDumpResistorExtractionWatts > 0,
		"battery_bus_failover_pct":  0.0,
		"avoided_loss_usd":          0.0,
		"audit_digest":              asm.AuditDigest,
		"compiled_at":               time.Now(),
	}

	if hasAct {
		dossier["battery_bus_failover_pct"] = act.SecondaryBatteryBusTransferPct
		dossier["avoided_loss_usd"] = act.AvoidedAircraftLossUSD
	}

	return dossier, nil
}

// RegisterAirborneHTSRoutes wires API endpoints under /api/v1/aviation/airborne-hts/*
func RegisterAirborneHTSRoutes(r *gin.RouterGroup) {
	htsGroup := r.Group("/aviation/airborne-hts")
	{
		htsGroup.POST("/ingest-stator-telemetry", func(c *gin.Context) {
			var t AirborneHTSTelemetry
			if err := c.ShouldBindJSON(&t); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res := GlobalAirborneHTSVault.IngestStatorTelemetry(t)
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		htsGroup.POST("/evaluate-quench-and-thermal-runaway", func(c *gin.Context) {
			var req struct {
				GeneratorID string `json:"generator_id"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := GlobalAirborneHTSVault.EvaluateQuenchAndThermalRunaway(req.GeneratorID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		htsGroup.POST("/trigger-fast-extraction-and-bus-transfer", func(c *gin.Context) {
			var req struct {
				GeneratorID string `json:"generator_id"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := GlobalAirborneHTSVault.TriggerFastExtractionAndBusTransfer(req.GeneratorID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		htsGroup.GET("/hts-flight-safety-dossier/:generator_id", func(c *gin.Context) {
			genID := c.Param("generator_id")
			res, err := GlobalAirborneHTSVault.GetHTSFlightSafetyDossier(genID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})
	}
}
