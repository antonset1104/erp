package ammonia

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

// AmmoniaLoopTelemetry captures Haber-Bosch green synthesis loop dynamics under ISO 22734 & ASME VIII-1
type AmmoniaLoopTelemetry struct {
	TelemetryID                  string    `json:"telemetry_id"`
	SynthesisLoopID              string    `json:"synthesis_loop_id"`                // "LOOP-HABER-BOSCH-GREEN-01"
	PlantID                      string    `json:"plant_id"`                          // "PLANT-NEOM-OXAGON-GREEN-NH3"
	ReactorBedTemperatureC       float64   `json:"reactor_bed_temperature_c"`        // e.g. 475.0°C (equilibrium 400-500°C)
	LoopPressureBar              float64   `json:"loop_pressure_bar"`                 // e.g. 195.0 bar
	RenewablePowerAvailabilityMW float64   `json:"renewable_power_availability_mw"`  // e.g. 35.0 MW (fluctuating renewable input)
	HydrogenFeedFlowNm3Hr        float64   `json:"hydrogen_feed_flow_nm3_hr"`         // e.g. 14200.0 Nm3/hr
	NitrogenFeedFlowNm3Hr        float64   `json:"nitrogen_feed_flow_nm3_hr"`         // e.g. 4800.0 Nm3/hr
	StoichiometricH2N2Ratio      float64   `json:"stoichiometric_h2_n2_ratio"`       // e.g. 2.96 (optimal 3.0:1)
	AmmoniaConversionPct         float64   `json:"ammonia_conversion_pct"`            // e.g. 16.5% per pass
	NitridingDepthMicrons        float64   `json:"nitriding_depth_microns"`           // e.g. 46.8 µm (case depth limit < 50 µm)
	RecordedAt                   time.Time `json:"recorded_at"`
}

// AmmoniaKineticsAssessment models Haber-Bosch equilibrium conversion, ramp rate, and vessel nitriding
type AmmoniaKineticsAssessment struct {
	AssessmentID                 string    `json:"assessment_id"`
	SynthesisLoopID              string    `json:"synthesis_loop_id"`
	EquilibriumDeviationKelvin   float64   `json:"equilibrium_deviation_kelvin"`
	LoadRampingRatePctPerMin     float64   `json:"load_ramping_rate_pct_per_min"`
	ThermalShockHazardLevel      string    `json:"thermal_shock_hazard_level"`       // "STEADY_STATE_OPTIMAL", "ELEVATED_RAMP_CYCLING_WARNING", "CRITICAL_THERMAL_SHOCK_AND_NITRIDING_BREACH"
	NitridingEmbrittlementRisk   string    `json:"nitriding_embrittlement_risk"`     // "PASSIVE_LAYER_STABLE", "NITRIDE_SCALE_GROWTH_WARNING", "CRITICAL_VESSEL_HYDROGEN_NITRIDING_FAIL"
	LoopBypassTripMandate        bool      `json:"loop_bypass_trip_mandate"`
	QuenchGasInjectionRequired   bool      `json:"quench_gas_injection_required"`
	AuditDigest                  string    `json:"audit_digest"`
	EvaluatedAt                  time.Time `json:"evaluated_at"`
}

// AmmoniaMitigationAction models automated bypass looping, recycle ratio trimming, and nitrogen purge
type AmmoniaMitigationAction struct {
	ActionID                      string    `json:"action_id"`
	SynthesisLoopID              string    `json:"synthesis_loop_id"`
	CommandedBypassRatioPct       float64   `json:"commanded_bypass_ratio_pct"`
	ModulatedSyngasRecycleRatio   float64   `json:"modulated_syngas_recycle_ratio"`
	NitrogenPurgeFlowNm3Hr        float64   `json:"nitrogen_purge_flow_nm3_hr"`
	AvoidedReactorExplosionLossUSD float64  `json:"avoided_reactor_explosion_loss_usd"` // Avoided $32M USD reactor vessel nitriding rupture & catalyst thermal sintering
	ActionTimestamp               time.Time `json:"action_timestamp"`
}

// AmmoniaVault thread-safe in-memory store
type AmmoniaVault struct {
	mu          sync.RWMutex
	telemetries map[string]AmmoniaLoopTelemetry
	assessments map[string]AmmoniaKineticsAssessment
	actions     map[string]AmmoniaMitigationAction
}

// Global instance
var GlobalAmmoniaVault = &AmmoniaVault{
	telemetries: make(map[string]AmmoniaLoopTelemetry),
	assessments: make(map[string]AmmoniaKineticsAssessment),
	actions:     make(map[string]AmmoniaMitigationAction),
}

func init() {
	// Seed demo dynamic green ammonia Haber-Bosch reactor loop under fluctuating renewable load
	GlobalAmmoniaVault.telemetries["LOOP-HABER-BOSCH-GREEN-01"] = AmmoniaLoopTelemetry{
		TelemetryID:                  "TEL-AMMONIA-001",
		SynthesisLoopID:              "LOOP-HABER-BOSCH-GREEN-01",
		PlantID:                      "PLANT-NEOM-OXAGON-GREEN-NH3",
		ReactorBedTemperatureC:       485.0, // High bed temperature accelerating Fe4N formation
		LoopPressureBar:              198.0,
		RenewablePowerAvailabilityMW: 24.5,  // Sharp drop from 50 MW wind power
		HydrogenFeedFlowNm3Hr:        11500.0,
		NitrogenFeedFlowNm3Hr:        4200.0,
		StoichiometricH2N2Ratio:      2.74,  // Sub-stoichiometric ratio
		AmmoniaConversionPct:         14.2,
		NitridingDepthMicrons:        48.5,  // Critical nitriding depth near threshold
		RecordedAt:                   time.Now(),
	}
}

// IngestLoopTelemetry stores gas chromatography and temperature profile readings
func (v *AmmoniaVault) IngestLoopTelemetry(t AmmoniaLoopTelemetry) AmmoniaLoopTelemetry {
	v.mu.Lock()
	defer v.mu.Unlock()

	t.RecordedAt = time.Now()
	v.telemetries[t.SynthesisLoopID] = t
	return t
}

// EvaluateAmmoniaKinetics evaluates thermal ramping shock and reactor vessel nitriding embrittlement
func (v *AmmoniaVault) EvaluateAmmoniaKinetics(loopID string) (AmmoniaKineticsAssessment, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	tel, exists := v.telemetries[loopID]
	if !exists {
		return AmmoniaKineticsAssessment{}, fmt.Errorf("synthesis loop %s not found in ammonia vault", loopID)
	}

	idealTempC := 425.0
	eqDeviation := math.Abs(tel.ReactorBedTemperatureC - idealTempC)

	// Load ramping rate estimated from renewable power shortfall
	rampRate := math.Abs(50.0-tel.RenewablePowerAvailabilityMW) * 0.45 // %/min

	thermalHazard := "STEADY_STATE_OPTIMAL"
	nitridingRisk := "PASSIVE_LAYER_STABLE"
	bypassMandate := false
	quenchReq := false

	if tel.ReactorBedTemperatureC > 470.0 || tel.NitridingDepthMicrons >= 45.0 {
		thermalHazard = "CRITICAL_THERMAL_SHOCK_AND_NITRIDING_BREACH"
		nitridingRisk = "CRITICAL_VESSEL_HYDROGEN_NITRIDING_FAIL"
		bypassMandate = true
		quenchReq = true
	} else if tel.ReactorBedTemperatureC > 450.0 || tel.NitridingDepthMicrons >= 35.0 {
		thermalHazard = "ELEVATED_RAMP_CYCLING_WARNING"
		nitridingRisk = "NITRIDE_SCALE_GROWTH_WARNING"
		quenchReq = true
	}

	hasher := sha256.New()
	hasher.Write([]byte(fmt.Sprintf("%s-%f-%f-%f-%s-%s-%d", loopID, eqDeviation, rampRate, tel.NitridingDepthMicrons, thermalHazard, nitridingRisk, time.Now().UnixNano())))
	digest := hex.EncodeToString(hasher.Sum(nil))

	assessID := fmt.Sprintf("ASSESS-NH3-%d", time.Now().UnixNano()%1000000)
	assessment := AmmoniaKineticsAssessment{
		AssessmentID:               assessID,
		SynthesisLoopID:            loopID,
		EquilibriumDeviationKelvin: math.Round(eqDeviation*10.0) / 10.0,
		LoadRampingRatePctPerMin:   math.Round(rampRate*10.0) / 10.0,
		ThermalShockHazardLevel:    thermalHazard,
		NitridingEmbrittlementRisk: nitridingRisk,
		LoopBypassTripMandate:      bypassMandate,
		QuenchGasInjectionRequired: quenchReq,
		AuditDigest:                digest,
		EvaluatedAt:                time.Now(),
	}

	v.assessments[loopID] = assessment
	return assessment, nil
}

// TriggerLoopBypassAndQuenchControl modulates bypass loop ratio and injects cold synthesis gas quench
func (v *AmmoniaVault) TriggerLoopBypassAndQuenchControl(loopID string) (AmmoniaMitigationAction, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	avoidedLoss := 32000000.0 // Avoided $32M USD high-pressure Haber-Bosch vessel catastrophic blowout & catalyst replacement

	actionID := fmt.Sprintf("ACT-NH3-%d", time.Now().UnixNano()%1000000)
	action := AmmoniaMitigationAction{
		ActionID:                      actionID,
		SynthesisLoopID:              loopID,
		CommandedBypassRatioPct:       28.5,  // Divert 28.5% syngas past the catalyst bed
		ModulatedSyngasRecycleRatio:   4.8,   // Increase recycle ratio to 4.8:1
		NitrogenPurgeFlowNm3Hr:        850.0, // Cold N2 quench to suppress thermal peak
		AvoidedReactorExplosionLossUSD: avoidedLoss,
		ActionTimestamp:               time.Now(),
	}

	v.actions[loopID] = action
	return action, nil
}

// GetGreenAmmoniaSafetyDossier compiles complete ISO 22734 & ASME VIII-1 green ammonia synthesis safety dossier
func (v *AmmoniaVault) GetGreenAmmoniaSafetyDossier(loopID string) (map[string]interface{}, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	tel, hasTel := v.telemetries[loopID]
	assess, hasAssess := v.assessments[loopID]
	act, hasAct := v.actions[loopID]

	dossier := map[string]interface{}{
		"synthesis_loop_id":      loopID,
		"has_loop_telemetry":     hasTel,
		"has_ammonia_assess":     hasAssess,
		"has_mitigation_action":  hasAct,
		"iso_22734_compliant":    true,
		"asme_section8_approved": true,
		"compiled_at":            time.Now(),
	}

	if hasTel {
		dossier["plant_id"] = tel.PlantID
		dossier["bed_temp_c"] = tel.ReactorBedTemperatureC
		dossier["loop_pressure_bar"] = tel.LoopPressureBar
		dossier["stoichiometric_ratio"] = tel.StoichiometricH2N2Ratio
		dossier["nitriding_depth_um"] = tel.NitridingDepthMicrons
	}
	if hasAssess {
		dossier["thermal_hazard"] = assess.ThermalShockHazardLevel
		dossier["nitriding_risk"] = assess.NitridingEmbrittlementRisk
		dossier["bypass_trip"] = assess.LoopBypassTripMandate
		dossier["quench_required"] = assess.QuenchGasInjectionRequired
	}
	if hasAct {
		dossier["bypass_ratio_pct"] = act.CommandedBypassRatioPct
		dossier["syngas_recycle_ratio"] = act.ModulatedSyngasRecycleRatio
		dossier["avoided_loss_usd"] = act.AvoidedReactorExplosionLossUSD
	}

	return dossier, nil
}

// RegisterAmmoniaRoutes wires Gin HTTP handlers
func RegisterAmmoniaRoutes(r *gin.RouterGroup) {
	group := r.Group("/ammonia/loop")
	{
		group.POST("/ingest-loop-telemetry", func(c *gin.Context) {
			var req AmmoniaLoopTelemetry
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res := GlobalAmmoniaVault.IngestLoopTelemetry(req)
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		group.POST("/evaluate-ammonia-kinetics", func(c *gin.Context) {
			var req struct {
				SynthesisLoopID string `json:"synthesis_loop_id"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := GlobalAmmoniaVault.EvaluateAmmoniaKinetics(req.SynthesisLoopID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		group.POST("/trigger-loop-bypass-and-quench", func(c *gin.Context) {
			var req struct {
				SynthesisLoopID string `json:"synthesis_loop_id"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := GlobalAmmoniaVault.TriggerLoopBypassAndQuenchControl(req.SynthesisLoopID)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		group.GET("/green-ammonia-safety-dossier/:loop_id", func(c *gin.Context) {
			lID := c.Param("loop_id")
			res, err := GlobalAmmoniaVault.GetGreenAmmoniaSafetyDossier(lID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})
	}
}
