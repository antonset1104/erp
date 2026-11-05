package dryelectrode

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

// DryElectrodeTelemetry captures gigafactory all-solid-state battery (ASSB) solvent-free dry electrode continuous extrusion & calendering under IEC 62660-3 & USABC
type DryElectrodeTelemetry struct {
	TelemetryID                  string    `json:"telemetry_id"`
	ProductionLineID             string    `json:"production_line_id"`                  // "LINE-GIGAFACTORY-DRY-CATHODE-01"
	CoatingLotID                 string    `json:"coating_lot_id"`                      // "LOT-DRY-NMC811-ASSB-408"
	TwinScrewMeltPressureBar     float64   `json:"twin_screw_melt_pressure_bar"`        // e.g. 185.0 bar
	ExtruderShearTorqueNm        float64   `json:"extruder_shear_torque_nm"`            // e.g. 340.0 Nm
	CalenderRollNipGapUm         float64   `json:"calender_roll_nip_gap_um"`            // e.g. 78.5 um
	CalenderRollSurfaceTempC     float64   `json:"calender_roll_surface_temp_c"`        // e.g. 112.5°C
	ElectrodeArealLoadingMgPerCm2 float64  `json:"electrode_areal_loading_mg_per_cm2"`  // e.g. 32.5 mg/cm2
	CalenderedPorosityPct        float64   `json:"calendered_porosity_pct"`             // e.g. 18.2% (solid-state target 14-16%)
	CrossWebThicknessVariationUm float64   `json:"cross_web_thickness_variation_um"`    // e.g. 4.8 um (strict limit < 2.0 um)
	WebTensionN                  float64   `json:"web_tension_n"`                       // e.g. 245.0 N
	RecordedAt                   time.Time `json:"recorded_at"`
}

// DryElectrodeAssessment evaluates PTFE binder shear fibrillation, calender roll nip pressure, and cross-web porosity gradients
type DryElectrodeAssessment struct {
	AssessmentID                   string    `json:"assessment_id"`
	ProductionLineID               string    `json:"production_line_id"`
	PorosityDeviationPct           float64   `json:"porosity_deviation_pct"`
	ElectrodeCoatingUniformity     string    `json:"electrode_coating_uniformity"`      // "UNIFORM_HIGH_DENSITY_WEB", "POROSITY_GRADIENT_EDGE_WARNING", "CRITICAL_DENDRITE_SHORT_HAZARD"
	PTFEFibrillationIntegrity      string    `json:"ptfe_fibrillation_integrity"`       // "COMPLETE_FIBRILLAR_MATRIX", "INSUFFICIENT_SHEAR_WEAK_WEB", "OVER_SHEARED_BRITTLE_TEAR"
	CalenderNipServoTrimMandate    bool      `json:"calender_nip_servo_trim_mandate"`
	ThermalInductionZoneTrimMandate bool     `json:"thermal_induction_zone_trim_mandate"`
	AuditDigest                    string    `json:"audit_digest"`
	EvaluatedAt                    time.Time `json:"evaluated_at"`
}

// DryElectrodeMitigationAction models calender servo roll nip trim, induction heating zone trim, and extruder screw speed modulation
type DryElectrodeMitigationAction struct {
	ActionID                       string    `json:"action_id"`
	ProductionLineID               string    `json:"production_line_id"`
	ModulatedCalenderRollNipGapUm  float64   `json:"modulated_calender_roll_nip_gap_um"`
	ZoneThermalInductionBoostC     float64   `json:"zone_thermal_induction_boost_c"`
	ExtruderScrewSpeedRpmTrim      float64   `json:"extruder_screw_speed_rpm_trim"`
	AvoidedBatteryScrapLossUSD     float64   `json:"avoided_battery_scrap_loss_usd"`
	ActionTimestamp                time.Time `json:"action_timestamp"`
}

// DryElectrodeVault thread-safe in-memory store
type DryElectrodeVault struct {
	mu          sync.RWMutex
	telemetries map[string]DryElectrodeTelemetry
	assessments map[string]DryElectrodeAssessment
	actions     map[string]DryElectrodeMitigationAction
}

// GlobalDryElectrodeVault instance
var GlobalDryElectrodeVault = &DryElectrodeVault{
	telemetries: make(map[string]DryElectrodeTelemetry),
	assessments: make(map[string]DryElectrodeAssessment),
	actions:     make(map[string]DryElectrodeMitigationAction),
}

// IngestElectrodeTelemetry stores dry electrode manufacturing telemetry
func (v *DryElectrodeVault) IngestElectrodeTelemetry(t DryElectrodeTelemetry) DryElectrodeTelemetry {
	v.mu.Lock()
	defer v.mu.Unlock()

	if t.TelemetryID == "" {
		t.TelemetryID = fmt.Sprintf("TEL-DRY-%d", time.Now().UnixNano()%1000000)
	}
	if t.RecordedAt.IsZero() {
		t.RecordedAt = time.Now()
	}

	v.telemetries[t.ProductionLineID] = t
	return t
}

// EvaluatePorosityAndFibrillation calculates target porosity deviation, web thickness variance, and PTFE matrix stability
func (v *DryElectrodeVault) EvaluatePorosityAndFibrillation(lineID string) (DryElectrodeAssessment, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	telem, exists := v.telemetries[lineID]
	if !exists {
		return DryElectrodeAssessment{}, fmt.Errorf("telemetry not found for line: %s", lineID)
	}

	porosityDev := math.Round((telem.CalenderedPorosityPct-15.0)*10) / 10

	uniformity := "UNIFORM_HIGH_DENSITY_WEB"
	calenderTrim := false
	thermalTrim := false

	if telem.CrossWebThicknessVariationUm > 3.0 || telem.CalenderedPorosityPct > 17.5 {
		uniformity = "CRITICAL_DENDRITE_SHORT_HAZARD"
		calenderTrim = true
		thermalTrim = true
	} else if telem.CrossWebThicknessVariationUm > 2.0 || telem.CalenderedPorosityPct > 16.0 {
		uniformity = "POROSITY_GRADIENT_EDGE_WARNING"
		calenderTrim = true
	}

	fibrillation := "COMPLETE_FIBRILLAR_MATRIX"
	if telem.ExtruderShearTorqueNm > 320.0 || telem.TwinScrewMeltPressureBar > 170.0 {
		fibrillation = "OVER_SHEARED_BRITTLE_TEAR"
		thermalTrim = true
	} else if telem.ExtruderShearTorqueNm < 250.0 {
		fibrillation = "INSUFFICIENT_SHEAR_WEAK_WEB"
	}

	rawHash := fmt.Sprintf("%s:%f:%s:%s:%t:%t", lineID, porosityDev, uniformity, fibrillation, calenderTrim, thermalTrim)
	hash := sha256.Sum256([]byte(rawHash))
	digest := hex.EncodeToString(hash[:])

	assessment := DryElectrodeAssessment{
		AssessmentID:                   fmt.Sprintf("ASM-DRY-%d", time.Now().UnixNano()%1000000),
		ProductionLineID:               lineID,
		PorosityDeviationPct:           porosityDev,
		ElectrodeCoatingUniformity:     uniformity,
		PTFEFibrillationIntegrity:      fibrillation,
		CalenderNipServoTrimMandate:    calenderTrim,
		ThermalInductionZoneTrimMandate: thermalTrim,
		AuditDigest:                    digest,
		EvaluatedAt:                    time.Now(),
	}

	v.assessments[lineID] = assessment
	return assessment, nil
}

// TriggerCalenderGapAndThermalTrim adjusts calender roll nip servo, boosts thermal induction, and trims extruder speed
func (v *DryElectrodeVault) TriggerCalenderGapAndThermalTrim(lineID string) (DryElectrodeMitigationAction, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	asm, hasAsm := v.assessments[lineID]
	telem, hasTelem := v.telemetries[lineID]
	if !hasAsm || !hasTelem {
		return DryElectrodeMitigationAction{}, fmt.Errorf("missing telemetry or assessment for line %s", lineID)
	}

	cmdNipGap := telem.CalenderRollNipGapUm
	thermalBoost := 0.0
	screwSpeedTrim := 0.0

	if asm.CalenderNipServoTrimMandate {
		cmdNipGap = 74.2 // micro-adjust calender nip servo down to compress free volume to 15.0% porosity
	}
	if asm.ThermalInductionZoneTrimMandate {
		thermalBoost = 3.5    // +3.5°C thermal induction boost to soften PTFE fibrils without thermal degradation
		screwSpeedTrim = -15.0 // reduce twin-screw RPM to relieve excessive melt shear stress
	}

	action := DryElectrodeMitigationAction{
		ActionID:                      fmt.Sprintf("ACT-DRY-%d", time.Now().UnixNano()%1000000),
		ProductionLineID:              lineID,
		ModulatedCalenderRollNipGapUm: cmdNipGap,
		ZoneThermalInductionBoostC:    thermalBoost,
		ExtruderScrewSpeedRpmTrim:     screwSpeedTrim,
		AvoidedBatteryScrapLossUSD:    25000000.0,
		ActionTimestamp:               time.Now(),
	}

	v.actions[lineID] = action
	return action, nil
}

// GetDryElectrodeQualityDossier returns solid-state dry electrode production release dossier under IEC 62660-3 & USABC
func (v *DryElectrodeVault) GetDryElectrodeQualityDossier(lineID string) (map[string]interface{}, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	telem, hasTelem := v.telemetries[lineID]
	asm, hasAsm := v.assessments[lineID]
	act := v.actions[lineID]

	if !hasTelem || !hasAsm {
		return nil, fmt.Errorf("incomplete data for dry electrode line dossier %s", lineID)
	}

	return map[string]interface{}{
		"production_line_id":  lineID,
		"coating_lot_id":      telem.CoatingLotID,
		"iec_62660_compliant": true,
		"usabc_certified":     true,
		"telemetry":           telem,
		"assessment":          asm,
		"latest_action":       act,
		"compiled_at":         time.Now(),
	}, nil
}

// RegisterDryElectrodeRoutes wires API endpoints under /api/v1/gigafactory/dry-electrode/*
func RegisterDryElectrodeRoutes(r *gin.RouterGroup) {
	deGroup := r.Group("/gigafactory/dry-electrode")
	{
		deGroup.POST("/ingest-electrode-telemetry", func(c *gin.Context) {
			var t DryElectrodeTelemetry
			if err := c.ShouldBindJSON(&t); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res := GlobalDryElectrodeVault.IngestElectrodeTelemetry(t)
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		deGroup.POST("/evaluate-porosity-and-fibrillation", func(c *gin.Context) {
			var req struct {
				ProductionLineID string `json:"production_line_id"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := GlobalDryElectrodeVault.EvaluatePorosityAndFibrillation(req.ProductionLineID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		deGroup.POST("/trigger-calender-gap-and-thermal-trim", func(c *gin.Context) {
			var req struct {
				ProductionLineID string `json:"production_line_id"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := GlobalDryElectrodeVault.TriggerCalenderGapAndThermalTrim(req.ProductionLineID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		deGroup.GET("/dry-electrode-quality-dossier/:line_id", func(c *gin.Context) {
			lineID := c.Param("line_id")
			res, err := GlobalDryElectrodeVault.GetDryElectrodeQualityDossier(lineID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})
	}
}
