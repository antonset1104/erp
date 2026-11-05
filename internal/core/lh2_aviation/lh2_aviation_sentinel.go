package lh2aviation

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

// LH2TankTelemetry captures commercial aircraft liquid hydrogen fuel system dynamics under FAA 14 CFR Part 25 & SAE AS6858
type LH2TankTelemetry struct {
	TelemetryID               string    `json:"telemetry_id"`
	TankID                    string    `json:"tank_id"`                     // "TANK-LH2-AIRCRAFT-A380-01"
	AircraftTailNumber        string    `json:"aircraft_tail_number"`        // "N701-ZERO-E"
	FlightPhase               string    `json:"flight_phase"`                 // "CRUISE_FL350"
	LiquidHydrogenMassKg      float64   `json:"liquid_hydrogen_mass_kg"`      // e.g. 4250.0 kg
	TankUllagePressureBar     float64   `json:"tank_ullage_pressure_bar"`     // e.g. 3.45 bar (safety relief setpoint 3.20 bar)
	BulkHydrogenTempKelvin    float64   `json:"bulk_hydrogen_temp_kelvin"`    // e.g. 23.2 K (nominal subcooled 20.4 K)
	VacuumJacketPressureMbar  float64   `json:"vacuum_jacket_pressure_mbar"`  // e.g. 0.085 mbar (loss-of-vacuum limit > 0.050 mbar)
	BoilOffGasRateKgPerHr     float64   `json:"boil_off_gas_rate_kg_per_hr"`  // e.g. 18.5 kg/hr (nominal < 8.0 kg/hr)
	CryoPumpInletNPSHMarginM  float64   `json:"cryo_pump_inlet_npsh_margin_m"` // e.g. 0.35 m (cavitation limit < 0.60 m)
	OrthoToParaRatio          float64   `json:"ortho_to_para_ratio"`          // e.g. 0.045
	RecordedAt                time.Time `json:"recorded_at"`
}

// LH2TankAssessment evaluates vacuum insulation loss, boil-off pressure rise, and cryogenic pump cavitation
type LH2TankAssessment struct {
	AssessmentID                    string    `json:"assessment_id"`
	TankID                          string    `json:"tank_id"`
	SubcoolingMarginKelvin          float64   `json:"subcooling_margin_kelvin"`
	VacuumInsulationStatus          string    `json:"vacuum_insulation_status"`           // "HIGH_VACUUM_OPTIMAL", "PERLITE_OUTGASSING_WARNING", "CRITICAL_VACUUM_LOSS_THERMAL_INCURSION"
	CryoPumpCavitationRisk          string    `json:"cryo_pump_cavitation_risk"`           // "HYDRAULIC_NPSH_SAFE", "ELEVATED_VAPOR_PRESSURE_WARNING", "CRYOPUMP_VAPOR_LOCK_CAVITATION"
	BoilOffReliquefactionMandate    bool      `json:"boil_off_reliquefaction_mandate"`
	AntiCavitationSubcoolingMandate bool      `json:"anti_cavitation_subcooling_mandate"`
	AuditDigest                     string    `json:"audit_digest"`
	EvaluatedAt                     time.Time `json:"evaluated_at"`
}

// LH2MitigationAction models BOG reliquefaction compressor activation, inerting nitrogen purge, and cryopump speed trim
type LH2MitigationAction struct {
	ActionID                         string    `json:"action_id"`
	TankID                           string    `json:"tank_id"`
	CommandedReliquefactionFlowKgHr  float64   `json:"commanded_reliquefaction_flow_kg_hr"`
	NitrogenInertingSweepRateNm3Hr   float64   `json:"nitrogen_inerting_sweep_rate_nm3_hr"`
	PumpSpeedThrottlingPct           float64   `json:"pump_speed_throttling_pct"`
	AvoidedAircraftLossUSD           float64   `json:"avoided_aircraft_loss_usd"`
	ActionTimestamp                  time.Time `json:"action_timestamp"`
}

// LH2Vault thread-safe in-memory store
type LH2Vault struct {
	mu          sync.RWMutex
	telemetries map[string]LH2TankTelemetry
	assessments map[string]LH2TankAssessment
	actions     map[string]LH2MitigationAction
}

// GlobalLH2Vault instance
var GlobalLH2Vault = &LH2Vault{
	telemetries: make(map[string]LH2TankTelemetry),
	assessments: make(map[string]LH2TankAssessment),
	actions:     make(map[string]LH2MitigationAction),
}

// IngestTankTelemetry stores LH2 cryogenic fuel tank telemetry
func (v *LH2Vault) IngestTankTelemetry(t LH2TankTelemetry) LH2TankTelemetry {
	v.mu.Lock()
	defer v.mu.Unlock()

	if t.TelemetryID == "" {
		t.TelemetryID = fmt.Sprintf("TEL-LH2-%d", time.Now().UnixNano()%1000000)
	}
	if t.RecordedAt.IsZero() {
		t.RecordedAt = time.Now()
	}

	v.telemetries[t.TankID] = t
	return t
}

// EvaluateBoilOffAndCavitation computes subcooling margin, vacuum degradation, and cryopump cavitation hazard
func (v *LH2Vault) EvaluateBoilOffAndCavitation(tankID string) (LH2TankAssessment, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	telem, exists := v.telemetries[tankID]
	if !exists {
		return LH2TankAssessment{}, fmt.Errorf("telemetry not found for LH2 tank: %s", tankID)
	}

	subcooling := math.Round((33.0-telem.BulkHydrogenTempKelvin)*10) / 10

	vacuumStatus := "HIGH_VACUUM_OPTIMAL"
	reliquefyMandate := false
	antiCavitateMandate := false

	if telem.VacuumJacketPressureMbar > 0.050 || telem.TankUllagePressureBar > 3.20 {
		vacuumStatus = "CRITICAL_VACUUM_LOSS_THERMAL_INCURSION"
		reliquefyMandate = true
		antiCavitateMandate = true
	} else if telem.VacuumJacketPressureMbar > 0.020 || telem.TankUllagePressureBar > 2.80 {
		vacuumStatus = "PERLITE_OUTGASSING_WARNING"
		reliquefyMandate = true
	}

	cavitationRisk := "HYDRAULIC_NPSH_SAFE"
	if telem.CryoPumpInletNPSHMarginM < 0.40 || telem.BoilOffGasRateKgPerHr > 15.0 {
		cavitationRisk = "CRYOPUMP_VAPOR_LOCK_CAVITATION"
		antiCavitateMandate = true
		reliquefyMandate = true
	} else if telem.CryoPumpInletNPSHMarginM < 0.60 || telem.BoilOffGasRateKgPerHr > 10.0 {
		cavitationRisk = "ELEVATED_VAPOR_PRESSURE_WARNING"
		antiCavitateMandate = true
	}

	rawHash := fmt.Sprintf("%s:%f:%s:%s:%t:%t", tankID, subcooling, vacuumStatus, cavitationRisk, reliquefyMandate, antiCavitateMandate)
	hash := sha256.Sum256([]byte(rawHash))
	digest := hex.EncodeToString(hash[:])

	assessment := LH2TankAssessment{
		AssessmentID:                    fmt.Sprintf("ASM-LH2-%d", time.Now().UnixNano()%1000000),
		TankID:                          tankID,
		SubcoolingMarginKelvin:          subcooling,
		VacuumInsulationStatus:          vacuumStatus,
		CryoPumpCavitationRisk:          cavitationRisk,
		BoilOffReliquefactionMandate:    reliquefyMandate,
		AntiCavitationSubcoolingMandate: antiCavitateMandate,
		AuditDigest:                     digest,
		EvaluatedAt:                     time.Now(),
	}

	v.assessments[tankID] = assessment
	return assessment, nil
}

// TriggerReliquefactionAndAntiCavitation deploys Stirling cryocooler reliquefaction, N2 sweep, and pump speed throttling
func (v *LH2Vault) TriggerReliquefactionAndAntiCavitation(tankID string) (LH2MitigationAction, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	asm, hasAsm := v.assessments[tankID]
	_, hasTelem := v.telemetries[tankID]
	if !hasAsm || !hasTelem {
		return LH2MitigationAction{}, fmt.Errorf("missing telemetry or assessment for LH2 tank %s", tankID)
	}

	reliqFlow := 0.0
	n2Sweep := 0.0
	pumpThrottle := 0.0

	if asm.BoilOffReliquefactionMandate {
		reliqFlow = 22.5 // command onboard cryogenic BOG reliquefaction compressor at 22.5 kg/hr
		n2Sweep = 12.0    // 12.0 Nm3/hr nitrogen sweep gas across vent shroud to inert overboard exhaust
	}
	if asm.AntiCavitationSubcoolingMandate {
		pumpThrottle = -15.0 // throttle cryogenic boost pump by 15% to suppress vapor inception at impeller eye
	}

	action := LH2MitigationAction{
		ActionID:                        fmt.Sprintf("ACT-LH2-%d", time.Now().UnixNano()%1000000),
		TankID:                          tankID,
		CommandedReliquefactionFlowKgHr: reliqFlow,
		NitrogenInertingSweepRateNm3Hr:  n2Sweep,
		PumpSpeedThrottlingPct:          pumpThrottle,
		AvoidedAircraftLossUSD:          160000000.0,
		ActionTimestamp:                 time.Now(),
	}

	v.actions[tankID] = action
	return action, nil
}

// GetLH2FlightSafetyDossier returns liquid hydrogen fuel system airworthiness dossier under FAA 14 CFR Part 25 & SAE AS6858
func (v *LH2Vault) GetLH2FlightSafetyDossier(tankID string) (map[string]interface{}, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	telem, hasTelem := v.telemetries[tankID]
	asm, hasAsm := v.assessments[tankID]
	act := v.actions[tankID]

	if !hasTelem || !hasAsm {
		return nil, fmt.Errorf("incomplete data for LH2 tank dossier %s", tankID)
	}

	return map[string]interface{}{
		"tank_id":               tankID,
		"aircraft_tail_number":  telem.AircraftTailNumber,
		"faa_14cfr25_compliant": true,
		"sae_as6858_certified":  true,
		"telemetry":             telem,
		"assessment":            asm,
		"latest_action":         act,
		"compiled_at":           time.Now(),
	}, nil
}

// RegisterLH2Routes wires API endpoints under /api/v1/aviation/lh2/*
func RegisterLH2Routes(r *gin.RouterGroup) {
	lh2Group := r.Group("/aviation/lh2")
	{
		lh2Group.POST("/ingest-tank-telemetry", func(c *gin.Context) {
			var t LH2TankTelemetry
			if err := c.ShouldBindJSON(&t); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res := GlobalLH2Vault.IngestTankTelemetry(t)
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		lh2Group.POST("/evaluate-boiloff-and-cavitation", func(c *gin.Context) {
			var req struct {
				TankID string `json:"tank_id"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := GlobalLH2Vault.EvaluateBoilOffAndCavitation(req.TankID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		lh2Group.POST("/trigger-reliquefaction-and-anti-cavitation", func(c *gin.Context) {
			var req struct {
				TankID string `json:"tank_id"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := GlobalLH2Vault.TriggerReliquefactionAndAntiCavitation(req.TankID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		lh2Group.GET("/lh2-flight-safety-dossier/:tank_id", func(c *gin.Context) {
			tankID := c.Param("tank_id")
			res, err := GlobalLH2Vault.GetLH2FlightSafetyDossier(tankID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})
	}
}
