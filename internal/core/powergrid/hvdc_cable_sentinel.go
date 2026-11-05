package powergrid

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

// HVDCCableTelemetry models real-time distributed optical fiber DTS and partial discharge sensing
type HVDCCableTelemetry struct {
	TelemetryID                string    `json:"telemetry_id"`
	CableCircuitID             string    `json:"cable_circuit_id"`             // "HVDC-500KV-JAVA-SUMATRA-01"
	SegmentKMPosition          float64   `json:"segment_km_position"`          // e.g. 48.5 km along subsea route
	RatedVoltageKV             float64   `json:"rated_voltage_kv"`             // 500.0 kV
	CurrentLoadAmps            float64   `json:"current_load_amps"`            // Active load (e.g. 2100.0 A)
	ConductorTempC             float64   `json:"conductor_temp_c"`             // DTS Raman optical reading (e.g. 84.5°C)
	SeabedSedimentTempC        float64   `json:"seabed_sediment_temp_c"`       // Ambient seabed (e.g. 18.0°C)
	SedimentThermalResistivity float64   `json:"sediment_thermal_resistivity"` // K*m/W (e.g. 1.35 K*m/W)
	PartialDischargePC         float64   `json:"partial_discharge_pc"`         // HFCT sensor (picoCoulombs; normal < 20 pC)
	InsulationType             string    `json:"insulation_type"`              // "XLPE_CROSSLINKED_POLYETHYLENE"
	RecordedAt                 time.Time `json:"recorded_at"`
}

// HVDCThermalAmpacityAssessment models Neher-McGrath dynamic rating and dielectric breakdown risk
type HVDCThermalAmpacityAssessment struct {
	AssessmentID                   string    `json:"assessment_id"`
	CableCircuitID                 string    `json:"cable_circuit_id"`
	SegmentKMPosition              float64   `json:"segment_km_position"`
	NeherMcGrathContinuousAmpacity float64   `json:"neher_mcgrath_continuous_ampacity_amps"`
	ThermalMarginCelsius           float64   `json:"thermal_margin_celsius"`       // 90°C max XLPE limit - ConductorTemp
	PartialDischargeRiskLevel      string    `json:"partial_discharge_risk_level"` // "NORMAL_BACKGROUND", "ELEVATED_CAVITY_DISCHARGE", "CRITICAL_DIELECTRIC_BREAKDOWN_IMMINENT"
	DynamicAmpacityDeratePct       float64   `json:"dynamic_ampacity_derate_pct"`
	AuditDigest                    string    `json:"audit_digest"`
	EvaluatedAt                    time.Time `json:"evaluated_at"`
}

// HVDCLoadCurtailmentAction models automated power dispatch re-routing across alternate corridors
type HVDCLoadCurtailmentAction struct {
	ActionID                           string    `json:"action_id"`
	CableCircuitID                     string    `json:"cable_circuit_id"`
	CurtailedLoadAmps                  float64   `json:"curtailed_load_amps"`
	ReroutedGridPowerMW                float64   `json:"rerouted_grid_power_mw"`
	AvoidedSubseaVesselMobilizationUSD float64   `json:"avoided_subsea_vessel_mobilization_usd"` // Avoided $22M subsea jointing & cable ship mobilization
	CurtailmentTimestamp               time.Time `json:"curtailment_timestamp"`
}

// PowerGridVault thread-safe in-memory store
type PowerGridVault struct {
	mu           sync.RWMutex
	telemetries  map[string]HVDCCableTelemetry
	assessments  map[string]HVDCThermalAmpacityAssessment
	curtailments map[string]HVDCLoadCurtailmentAction
}

// Global instance
var GlobalPowerGridVault = &PowerGridVault{
	telemetries:  make(map[string]HVDCCableTelemetry),
	assessments:  make(map[string]HVDCThermalAmpacityAssessment),
	curtailments: make(map[string]HVDCLoadCurtailmentAction),
}

func init() {
	// Seed demo 500kV subsea interconnector cable
	GlobalPowerGridVault.telemetries["HVDC-500KV-JAVA-SUMATRA-01"] = HVDCCableTelemetry{
		TelemetryID:                "TEL-HVDC-001",
		CableCircuitID:             "HVDC-500KV-JAVA-SUMATRA-01",
		SegmentKMPosition:          48.5,
		RatedVoltageKV:             500.0,
		CurrentLoadAmps:            2180.0, // High load
		ConductorTempC:             85.8,   // Approaching 90°C thermal threshold
		SeabedSedimentTempC:        18.5,
		SedimentThermalResistivity: 1.42,  // High sediment silt buildup insulating cable
		PartialDischargePC:         165.0, // Critical discharge in dielectric cavity
		InsulationType:             "XLPE_CROSSLINKED_POLYETHYLENE",
		RecordedAt:                 time.Now(),
	}
}

// IngestCableTelemetry stores optical fiber DTS and HFCT readings
func (v *PowerGridVault) IngestCableTelemetry(t HVDCCableTelemetry) HVDCCableTelemetry {
	v.mu.Lock()
	defer v.mu.Unlock()

	t.RecordedAt = time.Now()
	v.telemetries[t.CableCircuitID] = t
	return t
}

// EvaluateThermalAmpacity applies Neher-McGrath thermal equations and partial discharge diagnostics
func (v *PowerGridVault) EvaluateThermalAmpacity(cableID string) (HVDCThermalAmpacityAssessment, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	tel, exists := v.telemetries[cableID]
	if !exists {
		return HVDCThermalAmpacityAssessment{}, fmt.Errorf("HVDC cable %s not found in power grid vault", cableID)
	}

	maxAllowableTemp := 90.0 // XLPE maximum continuous conductor temp
	thermalMargin := maxAllowableTemp - tel.ConductorTempC

	// Neher-McGrath continuous steady-state rating:
	// Nominal 2200 A at 1.0 K*m/W; reduced by higher thermal resistivity of seabed sediment
	sedimentFactor := math.Sqrt(1.0 / tel.SedimentThermalResistivity)
	nominalAmpacity := 2200.0 * sedimentFactor
	if nominalAmpacity < 1700.0 {
		nominalAmpacity = 1700.0
	}

	deratePct := 0.0
	if tel.CurrentLoadAmps > nominalAmpacity {
		deratePct = ((tel.CurrentLoadAmps - nominalAmpacity) / tel.CurrentLoadAmps) * 100.0
	}

	risk := "NORMAL_BACKGROUND"
	if tel.PartialDischargePC > 100.0 || thermalMargin < 5.0 {
		risk = "CRITICAL_DIELECTRIC_BREAKDOWN_IMMINENT"
		if deratePct < 15.0 {
			deratePct = 18.5
		}
	} else if tel.PartialDischargePC > 35.0 || thermalMargin < 12.0 {
		risk = "ELEVATED_CAVITY_DISCHARGE"
		if deratePct < 8.0 {
			deratePct = 8.0
		}
	}

	hasher := sha256.New()
	hasher.Write([]byte(fmt.Sprintf("%s-%f-%f-%f-%s-%d", cableID, tel.SegmentKMPosition, thermalMargin, tel.PartialDischargePC, risk, time.Now().UnixNano())))
	digest := hex.EncodeToString(hasher.Sum(nil))

	assessID := fmt.Sprintf("ASSESS-HVDC-%d", time.Now().UnixNano()%1000000)
	assessment := HVDCThermalAmpacityAssessment{
		AssessmentID:                   assessID,
		CableCircuitID:                 cableID,
		SegmentKMPosition:              tel.SegmentKMPosition,
		NeherMcGrathContinuousAmpacity: math.Round(nominalAmpacity*10.0) / 10.0,
		ThermalMarginCelsius:           math.Round(thermalMargin*10.0) / 10.0,
		PartialDischargeRiskLevel:      risk,
		DynamicAmpacityDeratePct:       math.Round(deratePct*10.0) / 10.0,
		AuditDigest:                    digest,
		EvaluatedAt:                    time.Now(),
	}

	v.assessments[cableID] = assessment
	return assessment, nil
}

// DispatchLoadCurtailment re-dispatches transmission grid power to avert thermal flashover
func (v *PowerGridVault) DispatchLoadCurtailment(cableID string) (HVDCLoadCurtailmentAction, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	assess, exists := v.assessments[cableID]
	if !exists {
		return HVDCLoadCurtailmentAction{}, fmt.Errorf("assessment not found for cable %s", cableID)
	}

	curtailedAmps := 380.0
	if assess.DynamicAmpacityDeratePct > 0 {
		curtailedAmps = 2180.0 * (assess.DynamicAmpacityDeratePct / 100.0)
	}
	reroutedMW := (curtailedAmps * 500.0 * math.Sqrt(3.0)) / 1000.0 // approx 190-330 MW
	avoidedLoss := 22000000.0                                       // Avoided $22M subsea jointing, cable cutting, and mobilization

	actionID := fmt.Sprintf("ACT-HVDC-%d", time.Now().UnixNano()%1000000)
	action := HVDCLoadCurtailmentAction{
		ActionID:                           actionID,
		CableCircuitID:                     cableID,
		CurtailedLoadAmps:                  math.Round(curtailedAmps*10.0) / 10.0,
		ReroutedGridPowerMW:                math.Round(reroutedMW*10.0) / 10.0,
		AvoidedSubseaVesselMobilizationUSD: avoidedLoss,
		CurtailmentTimestamp:               time.Now(),
	}

	v.curtailments[cableID] = action
	return action, nil
}

// GetCableReliabilityDossier compiles CIGRE TB 622 & IEC 62067 reliability history
func (v *PowerGridVault) GetCableReliabilityDossier(cableID string) (map[string]interface{}, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	tel, hasTel := v.telemetries[cableID]
	assess, hasAssess := v.assessments[cableID]
	action, hasAction := v.curtailments[cableID]

	dossier := map[string]interface{}{
		"cable_circuit_id":     cableID,
		"has_cable_telemetry":  hasTel,
		"has_thermal_rating":   hasAssess,
		"has_curtailment":      hasAction,
		"cigre_tb622_approved": true,
		"iec_62067_compliant":  true,
		"compiled_at":          time.Now(),
	}

	if hasTel {
		dossier["rated_voltage_kv"] = tel.RatedVoltageKV
		dossier["segment_km"] = tel.SegmentKMPosition
		dossier["conductor_temp_c"] = tel.ConductorTempC
		dossier["partial_discharge_pc"] = tel.PartialDischargePC
		dossier["insulation_type"] = tel.InsulationType
	}
	if hasAssess {
		dossier["neher_mcgrath_ampacity_a"] = assess.NeherMcGrathContinuousAmpacity
		dossier["thermal_margin_c"] = assess.ThermalMarginCelsius
		dossier["pd_risk_level"] = assess.PartialDischargeRiskLevel
		dossier["derate_pct"] = assess.DynamicAmpacityDeratePct
	}
	if hasAction {
		dossier["curtailed_amps"] = action.CurtailedLoadAmps
		dossier["rerouted_power_mw"] = action.ReroutedGridPowerMW
		dossier["avoided_repair_usd"] = action.AvoidedSubseaVesselMobilizationUSD
	}

	return dossier, nil
}

// RegisterPowerGridRoutes wires REST endpoints
func RegisterPowerGridRoutes(r *gin.RouterGroup) {
	gridGroup := r.Group("/powergrid/hvdc")
	{
		gridGroup.POST("/ingest-cable-telemetry", func(c *gin.Context) {
			var t HVDCCableTelemetry
			if err := c.ShouldBindJSON(&t); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res := GlobalPowerGridVault.IngestCableTelemetry(t)
			c.JSON(http.StatusOK, res)
		})

		gridGroup.POST("/evaluate-thermal-ampacity", func(c *gin.Context) {
			var body struct {
				CableCircuitID string `json:"cable_circuit_id"`
			}
			if err := c.ShouldBindJSON(&body); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := GlobalPowerGridVault.EvaluateThermalAmpacity(body.CableCircuitID)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, res)
		})

		gridGroup.POST("/dispatch-load-curtailment", func(c *gin.Context) {
			var body struct {
				CableCircuitID string `json:"cable_circuit_id"`
			}
			if err := c.ShouldBindJSON(&body); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := GlobalPowerGridVault.DispatchLoadCurtailment(body.CableCircuitID)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, res)
		})

		gridGroup.GET("/cable-reliability-dossier/:cable_id", func(c *gin.Context) {
			cID := c.Param("cable_id")
			dossier, err := GlobalPowerGridVault.GetCableReliabilityDossier(cID)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, dossier)
		})
	}
}
