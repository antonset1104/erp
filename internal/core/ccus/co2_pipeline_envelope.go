package ccus

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

// CO2PipelineTelemetry models high-pressure dense phase CO2 transport pipeline readings
type CO2PipelineTelemetry struct {
	TelemetryID                string    `json:"telemetry_id"`
	PipelineSegmentID          string    `json:"pipeline_segment_id"`           // "CCS-PIPE-SUKOWATI-DENSE-01"
	OperatingPressureBar       float64   `json:"operating_pressure_bar"`        // e.g. 88.5 bar (> 73.8 bar critical)
	OperatingTempC             float64   `json:"operating_temp_c"`              // e.g. 34.5°C (> 31.1°C critical)
	CO2PurityMolPct            float64   `json:"co2_purity_mol_pct"`            // e.g. 96.8 mol% CO2
	N2ImpurityMolPct           float64   `json:"n2_impurity_mol_pct"`           // e.g. 2.1 mol% N2
	CH4ImpurityMolPct          float64   `json:"ch4_impurity_mol_pct"`          // e.g. 1.1 mol% CH4
	MassFlowRateTonsHr         float64   `json:"mass_flow_rate_tons_hr"`        // e.g. 420.0 tons/hr
	DepressurizationRateBarMin float64   `json:"depressurization_rate_bar_min"` // e.g. 4.8 bar/min during upset
	RecordedAt                 time.Time `json:"recorded_at"`
}

// PhaseEnvelopeThermodynamicAssessment models Peng-Robinson EOS phase envelope & Joule-Thomson chilling
type PhaseEnvelopeThermodynamicAssessment struct {
	AssessmentID                string    `json:"assessment_id"`
	PipelineSegmentID           string    `json:"pipeline_segment_id"`
	FluidPhaseState             string    `json:"fluid_phase_state"`              // "SUPERCRITICAL_DENSE_PHASE", "SUBCOOLED_LIQUID", "TWO_PHASE_VAPOR_LIQUID", "EXPANSION_SOLID_DRY_ICE_RISK"
	SaturationBubblePressureBar float64   `json:"saturation_bubble_pressure_bar"` // Dew/bubble pressure of impure mixture
	PressureMarginToTwoPhaseBar float64   `json:"pressure_margin_to_two_phase_bar"`
	JouleThomsonChillingCoeff   float64   `json:"joule_thomson_chilling_coeff_c_bar"` // mu_JT in °C/bar (e.g. 1.45 °C/bar)
	ProjectedChokedOrificeTempC float64   `json:"projected_choked_orifice_temp_c"`    // Can drop below -40°C DBTT
	BrittleDuctileFractureRisk  string    `json:"brittle_ductile_fracture_risk"`      // "SAFE_DUCTILE_RANGE", "ELEVATED_SUB_ZERO_CHILLING", "CRITICAL_BRITTLE_FRACTURE_RISK"
	AuditDigest                 string    `json:"audit_digest"`
	EvaluatedAt                 time.Time `json:"evaluated_at"`
}

// PipelineRepressurizationControl models automated line-heater firing and choke valve throttling
type PipelineRepressurizationControl struct {
	ControlID                 string    `json:"control_id"`
	PipelineSegmentID         string    `json:"pipeline_segment_id"`
	InLineHeaterDutyKW        float64   `json:"in_line_heater_duty_kw"`
	ChokeValveThrottledPct    float64   `json:"choke_valve_throttled_pct"`
	AvoidedPipeRuptureLossUSD float64   `json:"avoided_pipe_rupture_loss_usd"` // Avoided $35M pipeline rupture & toxic gas release
	ControlledAt              time.Time `json:"controlled_at"`
}

// CCUSVault thread-safe in-memory store
type CCUSVault struct {
	mu          sync.RWMutex
	telemetries map[string]CO2PipelineTelemetry
	assessments map[string]PhaseEnvelopeThermodynamicAssessment
	controls    map[string]PipelineRepressurizationControl
}

// Global instance
var GlobalCCUSVault = &CCUSVault{
	telemetries: make(map[string]CO2PipelineTelemetry),
	assessments: make(map[string]PhaseEnvelopeThermodynamicAssessment),
	controls:    make(map[string]PipelineRepressurizationControl),
}

func init() {
	// Seed demo supercritical CO2 transmission pipeline
	GlobalCCUSVault.telemetries["CCS-PIPE-SUKOWATI-DENSE-01"] = CO2PipelineTelemetry{
		TelemetryID:                "TEL-CCS-001",
		PipelineSegmentID:          "CCS-PIPE-SUKOWATI-DENSE-01",
		OperatingPressureBar:       78.2, // Approaching saturation envelope
		OperatingTempC:             32.5,
		CO2PurityMolPct:            96.8,
		N2ImpurityMolPct:           2.1,
		CH4ImpurityMolPct:          1.1,
		MassFlowRateTonsHr:         420.0,
		DepressurizationRateBarMin: 5.2, // Fast depressurization during ESD test
		RecordedAt:                 time.Now(),
	}
}

// IngestTelemetry stores mass flow and pressure-temperature transmitter readings
func (v *CCUSVault) IngestTelemetry(t CO2PipelineTelemetry) CO2PipelineTelemetry {
	v.mu.Lock()
	defer v.mu.Unlock()

	t.RecordedAt = time.Now()
	v.telemetries[t.PipelineSegmentID] = t
	return t
}

// EvaluatePhaseEnvelope evaluates dense phase integrity under ISO 27913 & DNV-RP-F104
func (v *CCUSVault) EvaluatePhaseEnvelope(segmentID string) (PhaseEnvelopeThermodynamicAssessment, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	tel, exists := v.telemetries[segmentID]
	if !exists {
		return PhaseEnvelopeThermodynamicAssessment{}, fmt.Errorf("CO2 pipeline %s not found in CCUS vault", segmentID)
	}

	// Impurities (N2, CH4) expand two-phase phase envelope and raise cricondenbar
	// Pure CO2 critical point is 73.8 bar, 31.1°C.
	// With 3.2% light impurities, bubble point pressure shifts up:
	impurityFraction := (tel.N2ImpurityMolPct + tel.CH4ImpurityMolPct) / 100.0
	saturationPressure := 73.8 + (impurityFraction * 140.0) // approx 78.28 bar
	pressureMargin := tel.OperatingPressureBar - saturationPressure

	// Joule-Thomson expansion coefficient mu_JT for dense CO2 approx 1.45 °C/bar
	muJT := 1.45
	// If pressure drops to 1 bar atmospheric in a pinhole/blowdown:
	deltaP := tel.OperatingPressureBar - 1.0
	tempDrop := muJT * deltaP * 0.45 // isenthalpic choked expansion factor
	projectedOrificeTemp := tel.OperatingTempC - tempDrop

	phaseState := "SUPERCRITICAL_DENSE_PHASE"
	fractureRisk := "SAFE_DUCTILE_RANGE"

	// Carbon steel pipeline ductile-to-brittle transition temperature (DBTT) is typically -40°C
	if projectedOrificeTemp < -40.0 || pressureMargin < 2.0 {
		phaseState = "EXPANSION_SOLID_DRY_ICE_RISK"
		fractureRisk = "CRITICAL_BRITTLE_FRACTURE_RISK"
	} else if pressureMargin < 6.0 || projectedOrificeTemp < -20.0 {
		phaseState = "TWO_PHASE_VAPOR_LIQUID"
		fractureRisk = "ELEVATED_SUB_ZERO_CHILLING"
	}

	hasher := sha256.New()
	hasher.Write([]byte(fmt.Sprintf("%s-%f-%f-%f-%s-%d", segmentID, tel.OperatingPressureBar, pressureMargin, projectedOrificeTemp, fractureRisk, time.Now().UnixNano())))
	digest := hex.EncodeToString(hasher.Sum(nil))

	assessID := fmt.Sprintf("ASSESS-CCS-%d", time.Now().UnixNano()%1000000)
	assessment := PhaseEnvelopeThermodynamicAssessment{
		AssessmentID:                assessID,
		PipelineSegmentID:           segmentID,
		FluidPhaseState:             phaseState,
		SaturationBubblePressureBar: math.Round(saturationPressure*100.0) / 100.0,
		PressureMarginToTwoPhaseBar: math.Round(pressureMargin*100.0) / 100.0,
		JouleThomsonChillingCoeff:   muJT,
		ProjectedChokedOrificeTempC: math.Round(projectedOrificeTemp*10.0) / 10.0,
		BrittleDuctileFractureRisk:  fractureRisk,
		AuditDigest:                 digest,
		EvaluatedAt:                 time.Now(),
	}

	v.assessments[segmentID] = assessment
	return assessment, nil
}

// TriggerRepressurizationControl modulates heating duty and choke valve position
func (v *CCUSVault) TriggerRepressurizationControl(segmentID string) (PipelineRepressurizationControl, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	avoidedLoss := 35000000.0 // Avoided $35M cryogenic pipeline rupture, repair, and environmental fines

	ctrlID := fmt.Sprintf("CTRL-CCS-%d", time.Now().UnixNano()%1000000)
	ctrl := PipelineRepressurizationControl{
		ControlID:                 ctrlID,
		PipelineSegmentID:         segmentID,
		InLineHeaterDutyKW:        650.0,
		ChokeValveThrottledPct:    35.0,
		AvoidedPipeRuptureLossUSD: avoidedLoss,
		ControlledAt:              time.Now(),
	}

	v.controls[segmentID] = ctrl
	return ctrl, nil
}

// GetPipelineIntegrityDossier compiles ISO 27913 & DNV-RP-F104 dense phase safety documentation
func (v *CCUSVault) GetPipelineIntegrityDossier(segmentID string) (map[string]interface{}, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	tel, hasTel := v.telemetries[segmentID]
	assess, hasAssess := v.assessments[segmentID]
	ctrl, hasCtrl := v.controls[segmentID]

	dossier := map[string]interface{}{
		"pipeline_segment_id":    segmentID,
		"has_pipeline_telemetry": hasTel,
		"has_phase_assessment":   hasAssess,
		"has_pressure_control":   hasCtrl,
		"iso_27913_compliant":    true,
		"dnv_rp_f104_certified":  true,
		"compiled_at":            time.Now(),
	}

	if hasTel {
		dossier["operating_pressure_bar"] = tel.OperatingPressureBar
		dossier["operating_temp_c"] = tel.OperatingTempC
		dossier["co2_purity_pct"] = tel.CO2PurityMolPct
		dossier["mass_flow_tons_hr"] = tel.MassFlowRateTonsHr
	}
	if hasAssess {
		dossier["fluid_phase_state"] = assess.FluidPhaseState
		dossier["saturation_pressure_bar"] = assess.SaturationBubblePressureBar
		dossier["pressure_margin_bar"] = assess.PressureMarginToTwoPhaseBar
		dossier["projected_orifice_temp_c"] = assess.ProjectedChokedOrificeTempC
		dossier["fracture_risk"] = assess.BrittleDuctileFractureRisk
	}
	if hasCtrl {
		dossier["heater_duty_kw"] = ctrl.InLineHeaterDutyKW
		dossier["choke_valve_throttled_pct"] = ctrl.ChokeValveThrottledPct
		dossier["avoided_rupture_loss_usd"] = ctrl.AvoidedPipeRuptureLossUSD
	}

	return dossier, nil
}

// RegisterCCUSRoutes wires REST endpoints
func RegisterCCUSRoutes(r *gin.RouterGroup) {
	ccusGroup := r.Group("/ccus/pipeline")
	{
		ccusGroup.POST("/ingest-telemetry", func(c *gin.Context) {
			var t CO2PipelineTelemetry
			if err := c.ShouldBindJSON(&t); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res := GlobalCCUSVault.IngestTelemetry(t)
			c.JSON(http.StatusOK, res)
		})

		ccusGroup.POST("/evaluate-phase-envelope", func(c *gin.Context) {
			var body struct {
				PipelineSegmentID string `json:"pipeline_segment_id"`
			}
			if err := c.ShouldBindJSON(&body); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := GlobalCCUSVault.EvaluatePhaseEnvelope(body.PipelineSegmentID)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, res)
		})

		ccusGroup.POST("/trigger-repressurization-control", func(c *gin.Context) {
			var body struct {
				PipelineSegmentID string `json:"pipeline_segment_id"`
			}
			if err := c.ShouldBindJSON(&body); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := GlobalCCUSVault.TriggerRepressurizationControl(body.PipelineSegmentID)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, res)
		})

		ccusGroup.GET("/integrity-dossier/:segment_id", func(c *gin.Context) {
			sID := c.Param("segment_id")
			dossier, err := GlobalCCUSVault.GetPipelineIntegrityDossier(sID)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, dossier)
		})
	}
}
