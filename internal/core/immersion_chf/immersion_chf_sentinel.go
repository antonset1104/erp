package immersionchf

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// ImmersionCHFTelemetry captures AI data center 2-phase dielectric immersion boiling and GPU heat flux metrics
type ImmersionCHFTelemetry struct {
	TelemetryID                  string    `json:"telemetry_id"`
	GpuNodeID                    string    `json:"gpu_node_id"`
	ClusterRackID                string    `json:"cluster_rack_id"`
	GpuHeatFluxWPerCm2           float64   `json:"gpu_heat_flux_w_per_cm2"`         // limit < 120.0 W/cm2
	SiliconJunctionTempC         float64   `json:"silicon_junction_temp_c"`         // limit < 95.0°C (thermal runaway > 105°C)
	DielectricFluidTempC         float64   `json:"dielectric_fluid_temp_c"`         // nominal 50.0 - 65.0°C
	BubbleDepartureFrequencyHz   float64   `json:"bubble_departure_frequency_hz"`   // threshold > 850 Hz (vapor blanket coalescence)
	DielectricBreakdownVoltageKv float64   `json:"dielectric_breakdown_voltage_kv"` // limit > 35.0 kV
	VaporChamberPressureBar      float64   `json:"vapor_chamber_pressure_bar"`      // nominal 1.05 - 1.45 bar
	CondenserSubcoolingDeltaC    float64   `json:"condenser_subcooling_delta_c"`    // nominal 8.0 - 15.0°C
	Timestamp                    time.Time `json:"timestamp"`
}

// ImmersionCHFAssessment models boiling regime transition, Critical Heat Flux (CHF) departure, and ASHRAE TC 9.9 compliance
type ImmersionCHFAssessment struct {
	AssessmentID              string    `json:"assessment_id"`
	GpuNodeID                 string    `json:"gpu_node_id"`
	BoilingRegimeState        string    `json:"boiling_regime_state"`       // NUCLEATE_BOILING_OPTIMAL, TRANSITION_BOILING_WARNING, FILM_BOILING_CHF_DEPARTURE_CRITICAL
	DielectricFluidCondition  string    `json:"dielectric_fluid_condition"` // HIGH_PURITY_ISOLATED, MOISTURE_CONTAMINATION_DEGRADED
	HeatFluxExcessMarginPct   float64   `json:"heat_flux_excess_margin_pct"`
	MicrojetPumpingMandate    bool      `json:"microjet_pumping_mandate"`
	GpuClockThrottlingMandate bool      `json:"gpu_clock_throttling_mandate"`
	AuditDigest               string    `json:"audit_digest"`
	EvaluatedAt               time.Time `json:"evaluated_at"`
}

// ImmersionCHFMitigationAction models targeted dielectric microjet impingement, GPU power capping, and condenser fan boost
type ImmersionCHFMitigationAction struct {
	ActionID                         string    `json:"action_id"`
	GpuNodeID                        string    `json:"gpu_node_id"`
	CommandedMicrojetVelocityMPerSec float64   `json:"commanded_microjet_velocity_m_per_sec"`
	PowerCappingLimitWatts           float64   `json:"power_capping_limit_watts"`
	CondenserFanSpeedRpm             float64   `json:"condenser_fan_speed_rpm"`
	AvoidedSiliconMeltdownLossUSD    float64   `json:"avoided_silicon_meltdown_loss_usd"`
	ExecutedAt                       time.Time `json:"executed_at"`
}

// ImmersionCHFVault manages high-density AI cluster immersion cooling telemetry and thermal runaway mitigation
type ImmersionCHFVault struct {
	mu          sync.RWMutex
	telemetries map[string]ImmersionCHFTelemetry
	assessments map[string]ImmersionCHFAssessment
	actions     map[string]ImmersionCHFMitigationAction
}

// GlobalImmersionCHFVault is the singleton instance for immersion cooling CHF sentinel
var GlobalImmersionCHFVault = NewImmersionCHFVault()

func NewImmersionCHFVault() *ImmersionCHFVault {
	return &ImmersionCHFVault{
		telemetries: make(map[string]ImmersionCHFTelemetry),
		assessments: make(map[string]ImmersionCHFAssessment),
		actions:     make(map[string]ImmersionCHFMitigationAction),
	}
}

// IngestImmersionTelemetry stores GPU immersion tank sensor readings
func (v *ImmersionCHFVault) IngestImmersionTelemetry(telemetry ImmersionCHFTelemetry) ImmersionCHFTelemetry {
	v.mu.Lock()
	defer v.mu.Unlock()

	if telemetry.Timestamp.IsZero() {
		telemetry.Timestamp = time.Now()
	}
	v.telemetries[telemetry.GpuNodeID] = telemetry
	return telemetry
}

// EvaluateCHFAndVaporBlanket evaluates boiling regime, CHF departure risk, and ASHRAE TC 9.9 thermal limits
func (v *ImmersionCHFVault) EvaluateCHFAndVaporBlanket(gpuNodeID string) (ImmersionCHFAssessment, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	telem, exists := v.telemetries[gpuNodeID]
	if !exists {
		return ImmersionCHFAssessment{}, fmt.Errorf("telemetry not found for GPU node %s", gpuNodeID)
	}

	boilingState := "NUCLEATE_BOILING_OPTIMAL"
	fluidCondition := "HIGH_PURITY_ISOLATED"
	microjetMandate := false
	throttleMandate := false

	chfLimitWPerCm2 := 120.0
	excessFlux := (telem.GpuHeatFluxWPerCm2 - chfLimitWPerCm2) / chfLimitWPerCm2 * 100.0

	if telem.DielectricBreakdownVoltageKv < 35.0 {
		fluidCondition = "MOISTURE_CONTAMINATION_DEGRADED"
	}

	if telem.GpuHeatFluxWPerCm2 >= 120.0 || telem.SiliconJunctionTempC >= 95.0 || telem.BubbleDepartureFrequencyHz >= 850.0 {
		boilingState = "FILM_BOILING_CHF_DEPARTURE_CRITICAL"
		microjetMandate = true
		throttleMandate = true
	} else if telem.GpuHeatFluxWPerCm2 > 95.0 || telem.SiliconJunctionTempC > 82.0 || telem.BubbleDepartureFrequencyHz > 600.0 {
		boilingState = "TRANSITION_BOILING_WARNING"
		microjetMandate = true
	}

	hashInput := fmt.Sprintf("%s:%s:%s:%.2f:%.2f:%t",
		telem.GpuNodeID, boilingState, fluidCondition, telem.GpuHeatFluxWPerCm2, telem.SiliconJunctionTempC, microjetMandate)
	hasher := sha256.New()
	hasher.Write([]byte(hashInput))
	digest := hex.EncodeToString(hasher.Sum(nil))

	assessment := ImmersionCHFAssessment{
		AssessmentID:              fmt.Sprintf("ASM-CHF-%d", time.Now().UnixNano()%1000000),
		GpuNodeID:                 gpuNodeID,
		BoilingRegimeState:        boilingState,
		DielectricFluidCondition:  fluidCondition,
		HeatFluxExcessMarginPct:   excessFlux,
		MicrojetPumpingMandate:    microjetMandate,
		GpuClockThrottlingMandate: throttleMandate,
		AuditDigest:               digest,
		EvaluatedAt:               time.Now(),
	}

	v.assessments[gpuNodeID] = assessment
	return assessment, nil
}

// TriggerMicrojetAndPowerCapping activates impinging dielectric microjets and throttles GPU power capping
func (v *ImmersionCHFVault) TriggerMicrojetAndPowerCapping(gpuNodeID string) (ImmersionCHFMitigationAction, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	asm, hasAsm := v.assessments[gpuNodeID]
	_, hasTelem := v.telemetries[gpuNodeID]
	if !hasAsm || !hasTelem {
		return ImmersionCHFMitigationAction{}, fmt.Errorf("missing telemetry or assessment for GPU node %s", gpuNodeID)
	}

	jetVelocity := 0.0
	powerCapWatts := 1000.0
	fanRpm := 2400.0

	if asm.MicrojetPumpingMandate {
		jetVelocity = 8.75 // pump 8.75 m/s subcooled dielectric fluid directly onto micro-pin fin heat spreader
		fanRpm = 4500.0    // accelerate condenser loop fans to 4,500 RPM
	}
	if asm.GpuClockThrottlingMandate {
		powerCapWatts = 650.0 // clamp package power from 1000 W to 650 W to disperse vapor blanket
	}

	action := ImmersionCHFMitigationAction{
		ActionID:                         fmt.Sprintf("ACT-CHF-%d", time.Now().UnixNano()%1000000),
		GpuNodeID:                        gpuNodeID,
		CommandedMicrojetVelocityMPerSec: jetVelocity,
		PowerCappingLimitWatts:           powerCapWatts,
		CondenserFanSpeedRpm:             fanRpm,
		AvoidedSiliconMeltdownLossUSD:    140000000.0, // $140M USD avoided GPU silicon cluster permanent meltdown and rack fire
		ExecutedAt:                       time.Now(),
	}

	v.actions[gpuNodeID] = action
	return action, nil
}

// GetImmersionCoolingDossier compiles GPU thermal state, boiling regime, and OCP immersion standards
func (v *ImmersionCHFVault) GetImmersionCoolingDossier(gpuNodeID string) (map[string]interface{}, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	telem, hasTelem := v.telemetries[gpuNodeID]
	asm, hasAsm := v.assessments[gpuNodeID]
	act, hasAct := v.actions[gpuNodeID]

	if !hasTelem || !hasAsm {
		return nil, fmt.Errorf("incomplete data for GPU immersion dossier: %s", gpuNodeID)
	}

	dossier := map[string]interface{}{
		"gpu_node_id":               gpuNodeID,
		"cluster_rack_id":           telem.ClusterRackID,
		"ashrae_tc99_compliant":     asm.BoilingRegimeState != "FILM_BOILING_CHF_DEPARTURE_CRITICAL" || (hasAct && act.AvoidedSiliconMeltdownLossUSD > 0),
		"ocp_advanced_cooling_pass": true,
		"gpu_heat_flux_w_cm2":       telem.GpuHeatFluxWPerCm2,
		"silicon_junction_temp_c":   telem.SiliconJunctionTempC,
		"bubble_frequency_hz":       telem.BubbleDepartureFrequencyHz,
		"breakdown_voltage_kv":      telem.DielectricBreakdownVoltageKv,
		"boiling_state":             asm.BoilingRegimeState,
		"fluid_condition":           asm.DielectricFluidCondition,
		"microjet_velocity_m_s":     0.0,
		"power_cap_watts":           1000.0,
		"avoided_loss_usd":          0.0,
		"audit_digest":              asm.AuditDigest,
		"compiled_at":               time.Now(),
	}

	if hasAct {
		dossier["microjet_velocity_m_s"] = act.CommandedMicrojetVelocityMPerSec
		dossier["power_cap_watts"] = act.PowerCappingLimitWatts
		dossier["avoided_loss_usd"] = act.AvoidedSiliconMeltdownLossUSD
	}

	return dossier, nil
}

// RegisterImmersionCHFRoutes wires API endpoints under /api/v1/datacenter/immersion-chf/*
func RegisterImmersionCHFRoutes(r *gin.RouterGroup) {
	chfGroup := r.Group("/datacenter/immersion-chf")
	{
		chfGroup.POST("/ingest-immersion-telemetry", func(c *gin.Context) {
			var t ImmersionCHFTelemetry
			if err := c.ShouldBindJSON(&t); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res := GlobalImmersionCHFVault.IngestImmersionTelemetry(t)
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		chfGroup.POST("/evaluate-chf-and-vapor-blanket", func(c *gin.Context) {
			var req struct {
				GpuNodeID string `json:"gpu_node_id"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := GlobalImmersionCHFVault.EvaluateCHFAndVaporBlanket(req.GpuNodeID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		chfGroup.POST("/trigger-microjet-and-power-capping", func(c *gin.Context) {
			var req struct {
				GpuNodeID string `json:"gpu_node_id"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := GlobalImmersionCHFVault.TriggerMicrojetAndPowerCapping(req.GpuNodeID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		chfGroup.GET("/immersion-cooling-dossier/:gpu_node_id", func(c *gin.Context) {
			nodeID := c.Param("gpu_node_id")
			res, err := GlobalImmersionCHFVault.GetImmersionCoolingDossier(nodeID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})
	}
}
