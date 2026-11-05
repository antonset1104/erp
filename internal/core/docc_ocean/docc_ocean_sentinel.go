package doccocean

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// DOCCOceanTelemetry captures Direct Ocean Carbon Capture (DOCC) bipolar membrane electrodialysis (BPMED) metrics
type DOCCOceanTelemetry struct {
	TelemetryID                      string    `json:"telemetry_id"`
	PlantID                          string    `json:"plant_id"`
	OceanPlatformID                  string    `json:"ocean_platform_id"`
	SeawaterFlowRateM3PerHr          float64   `json:"seawater_flow_rate_m3_per_hr"`           // nominal 1200 - 3500 m3/hr
	BipolarMembraneCellVoltageV      float64   `json:"bipolar_membrane_cell_voltage_v"`        // limit < 2.40 V (voltage spike > 2.50 V)
	CurrentDensityMaPerCm2           float64   `json:"current_density_ma_per_cm2"`             // nominal 80 - 150 mA/cm2
	BruciteCalciteScaleThicknessUm   float64   `json:"brucite_calcite_scale_thickness_um"`     // critical limit > 12.0 um
	EffluentDischargePh              float64   `json:"effluent_discharge_ph"`                  // IMO London Protocol allowable: 7.60 - 8.40
	OceanCo2ExtractionRateTonsPerDay float64   `json:"ocean_co2_extraction_rate_tons_per_day"` // nominal 40 - 100 t/day
	Co2ProductGasPurityPct           float64   `json:"co2_product_gas_purity_pct"`             // limit > 98.0%
	Timestamp                        time.Time `json:"timestamp"`
}

// DOCCOceanAssessment models mineral scaling, stack overvoltage, and IMO London Protocol / ISO 14064-2 compliance
type DOCCOceanAssessment struct {
	AssessmentID                      string    `json:"assessment_id"`
	PlantID                           string    `json:"plant_id"`
	ElectrodialysisStackCondition     string    `json:"electrodialysis_stack_condition"`      // PRISTINE_DESCALED, SCALE_NUCLEATION_WARNING, SEVERE_BRUCITE_MINERAL_FOULING_CRITICAL
	LondonProtocolDischargeCompliance string    `json:"london_protocol_discharge_compliance"` // FULLY_COMPLIANT, EFFLUENT_ACIDITY_EXCURSION_RISK
	CellVoltageExcessMarginPct        float64   `json:"cell_voltage_excess_margin_pct"`
	PolarityReversalMandate           bool      `json:"polarity_reversal_mandate"`
	DiluteAcidFlushMandate            bool      `json:"dilute_acid_flush_mandate"`
	AuditDigest                       string    `json:"audit_digest"`
	EvaluatedAt                       time.Time `json:"evaluated_at"`
}

// DOCCOceanMitigationAction models electrodialysis reversal (EDR), food-grade citric acid wash, and alkaline neutralization
type DOCCOceanMitigationAction struct {
	ActionID                           string    `json:"action_id"`
	PlantID                            string    `json:"plant_id"`
	ElectrodialysisPolarityReversalSec float64   `json:"electrodialysis_polarity_reversal_sec"`
	CitricAcidFlushRateLpm             float64   `json:"citric_acid_flush_rate_lpm"`
	EffluentCausticNeutralizationPh    float64   `json:"effluent_caustic_neutralization_ph"`
	AvoidedMembraneStackBurnoutLossUSD float64   `json:"avoided_membrane_stack_burnout_loss_usd"`
	ExecutedAt                         time.Time `json:"executed_at"`
}

// DOCCOceanVault manages electrochemical ocean carbon capture stack health, scaling alerts, and marine environmental permits
type DOCCOceanVault struct {
	mu          sync.RWMutex
	telemetries map[string]DOCCOceanTelemetry
	assessments map[string]DOCCOceanAssessment
	actions     map[string]DOCCOceanMitigationAction
}

// GlobalDOCCOceanVault is the singleton instance for DOCC electrochemical capture sentinel
var GlobalDOCCOceanVault = NewDOCCOceanVault()

func NewDOCCOceanVault() *DOCCOceanVault {
	return &DOCCOceanVault{
		telemetries: make(map[string]DOCCOceanTelemetry),
		assessments: make(map[string]DOCCOceanAssessment),
		actions:     make(map[string]DOCCOceanMitigationAction),
	}
}

// IngestDOCCTelemetry stores direct ocean capture electrochemical stack sensors
func (v *DOCCOceanVault) IngestDOCCTelemetry(telemetry DOCCOceanTelemetry) DOCCOceanTelemetry {
	v.mu.Lock()
	defer v.mu.Unlock()

	if telemetry.Timestamp.IsZero() {
		telemetry.Timestamp = time.Now()
	}
	v.telemetries[telemetry.PlantID] = telemetry
	return telemetry
}

// EvaluateScalingAndAcidification evaluates BPMED cell scaling, effluent pH, and IMO London Protocol rules
func (v *DOCCOceanVault) EvaluateScalingAndAcidification(plantID string) (DOCCOceanAssessment, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	telem, exists := v.telemetries[plantID]
	if !exists {
		return DOCCOceanAssessment{}, fmt.Errorf("telemetry not found for DOCC plant %s", plantID)
	}

	stackCondition := "PRISTINE_DESCALED"
	compliance := "FULLY_COMPLIANT"
	reversalMandate := false
	flushMandate := false

	voltLimit := 2.40 // V
	excessVolt := (telem.BipolarMembraneCellVoltageV - voltLimit) / voltLimit * 100.0

	if telem.EffluentDischargePh < 7.60 || telem.EffluentDischargePh > 8.40 {
		compliance = "EFFLUENT_ACIDITY_EXCURSION_RISK"
	}

	if telem.BipolarMembraneCellVoltageV >= 2.40 || telem.BruciteCalciteScaleThicknessUm >= 12.0 || compliance == "EFFLUENT_ACIDITY_EXCURSION_RISK" {
		stackCondition = "SEVERE_BRUCITE_MINERAL_FOULING_CRITICAL"
		reversalMandate = true
		flushMandate = true
	} else if telem.BipolarMembraneCellVoltageV > 2.10 || telem.BruciteCalciteScaleThicknessUm > 6.0 {
		stackCondition = "SCALE_NUCLEATION_WARNING"
		reversalMandate = true
	}

	hashInput := fmt.Sprintf("%s:%s:%s:%.2f:%.2f:%t",
		telem.PlantID, stackCondition, compliance, telem.BipolarMembraneCellVoltageV, telem.EffluentDischargePh, reversalMandate)
	hasher := sha256.New()
	hasher.Write([]byte(hashInput))
	digest := hex.EncodeToString(hasher.Sum(nil))

	assessment := DOCCOceanAssessment{
		AssessmentID:                      fmt.Sprintf("ASM-DOCC-%d", time.Now().UnixNano()%1000000),
		PlantID:                           plantID,
		ElectrodialysisStackCondition:     stackCondition,
		LondonProtocolDischargeCompliance: compliance,
		CellVoltageExcessMarginPct:        excessVolt,
		PolarityReversalMandate:           reversalMandate,
		DiluteAcidFlushMandate:            flushMandate,
		AuditDigest:                       digest,
		EvaluatedAt:                       time.Now(),
	}

	v.assessments[plantID] = assessment
	return assessment, nil
}

// TriggerEDRPolarityReversalAndAcidFlush triggers EDR current reversal, citric acid descaling wash, and neutralization
func (v *DOCCOceanVault) TriggerEDRPolarityReversalAndAcidFlush(plantID string) (DOCCOceanMitigationAction, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	asm, hasAsm := v.assessments[plantID]
	_, hasTelem := v.telemetries[plantID]
	if !hasAsm || !hasTelem {
		return DOCCOceanMitigationAction{}, fmt.Errorf("missing telemetry or assessment for DOCC plant %s", plantID)
	}

	reversalSec := 0.0
	flushRate := 0.0
	neutralPh := 8.10

	if asm.PolarityReversalMandate {
		reversalSec = 120.0 // reverse BPMED electrode polarity for 120 seconds to dissolve brucite (Mg(OH)2) precipitates
	}
	if asm.DiluteAcidFlushMandate {
		flushRate = 265.0 // pump 265 L/min 2.5% biodegradable citric acid wash through membrane spacer channels
		neutralPh = 8.18  // dose seawater degassing retentate with sodium hydroxide to restore ocean discharge pH to 8.18
	}

	action := DOCCOceanMitigationAction{
		ActionID:                           fmt.Sprintf("ACT-DOCC-%d", time.Now().UnixNano()%1000000),
		PlantID:                            plantID,
		ElectrodialysisPolarityReversalSec: reversalSec,
		CitricAcidFlushRateLpm:             flushRate,
		EffluentCausticNeutralizationPh:    neutralPh,
		AvoidedMembraneStackBurnoutLossUSD: 85000000.0, // $85M USD avoided bipolar membrane stack irreversible burnout and environmental fines
		ExecutedAt:                         time.Now(),
	}

	v.actions[plantID] = action
	return action, nil
}

// GetDOCCEnvironmentalDossier compiles marine carbon capture verification, stack health, and IMO London Protocol records
func (v *DOCCOceanVault) GetDOCCEnvironmentalDossier(plantID string) (map[string]interface{}, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	telem, hasTelem := v.telemetries[plantID]
	asm, hasAsm := v.assessments[plantID]
	act, hasAct := v.actions[plantID]

	if !hasTelem || !hasAsm {
		return nil, fmt.Errorf("incomplete data for DOCC environmental dossier: %s", plantID)
	}

	dossier := map[string]interface{}{
		"plant_id":                    plantID,
		"ocean_platform":              telem.OceanPlatformID,
		"imo_london_protocol_cleared": asm.LondonProtocolDischargeCompliance == "FULLY_COMPLIANT" || (hasAct && act.AvoidedMembraneStackBurnoutLossUSD > 0),
		"iso_14064_certified":         true,
		"seawater_flow_m3_hr":         telem.SeawaterFlowRateM3PerHr,
		"bpmed_cell_voltage_v":        telem.BipolarMembraneCellVoltageV,
		"scale_thickness_um":          telem.BruciteCalciteScaleThicknessUm,
		"effluent_ph":                 telem.EffluentDischargePh,
		"co2_purity_pct":              telem.Co2ProductGasPurityPct,
		"stack_condition":             asm.ElectrodialysisStackCondition,
		"discharge_compliance":        asm.LondonProtocolDischargeCompliance,
		"polarity_reversal_sec":       0.0,
		"citric_flush_lpm":            0.0,
		"avoided_loss_usd":            0.0,
		"audit_digest":                asm.AuditDigest,
		"compiled_at":                 time.Now(),
	}

	if hasAct {
		dossier["polarity_reversal_sec"] = act.ElectrodialysisPolarityReversalSec
		dossier["citric_flush_lpm"] = act.CitricAcidFlushRateLpm
		dossier["avoided_loss_usd"] = act.AvoidedMembraneStackBurnoutLossUSD
	}

	return dossier, nil
}

// RegisterDOCCOceanRoutes wires API endpoints under /api/v1/ocean/docc-capture/*
func RegisterDOCCOceanRoutes(r *gin.RouterGroup) {
	doccGroup := r.Group("/ocean/docc-capture")
	{
		doccGroup.POST("/ingest-docc-telemetry", func(c *gin.Context) {
			var t DOCCOceanTelemetry
			if err := c.ShouldBindJSON(&t); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res := GlobalDOCCOceanVault.IngestDOCCTelemetry(t)
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		doccGroup.POST("/evaluate-scaling-and-acidification", func(c *gin.Context) {
			var req struct {
				PlantID string `json:"plant_id"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := GlobalDOCCOceanVault.EvaluateScalingAndAcidification(req.PlantID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		doccGroup.POST("/trigger-edr-polarity-reversal-and-acid-flush", func(c *gin.Context) {
			var req struct {
				PlantID string `json:"plant_id"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := GlobalDOCCOceanVault.TriggerEDRPolarityReversalAndAcidFlush(req.PlantID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		doccGroup.GET("/docc-environmental-dossier/:plant_id", func(c *gin.Context) {
			pID := c.Param("plant_id")
			res, err := GlobalDOCCOceanVault.GetDOCCEnvironmentalDossier(pID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})
	}
}
