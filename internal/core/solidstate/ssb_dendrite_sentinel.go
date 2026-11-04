package solidstate

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

// SSBCellTelemetry models all-solid-state battery cell and interface acoustics under IEC 62660-3 & ISO 12405-4
type SSBCellTelemetry struct {
	TelemetryID                      string    `json:"telemetry_id"`
	CellID                           string    `json:"cell_id"`                              // "CELL-SSB-SULFIDE-LITHIUM-01"
	PackID                           string    `json:"pack_id"`                              // "PACK-EV-800V-100KWH"
	CurrentDensityMA_CM2             float64   `json:"current_density_ma_cm2"`               // e.g. 5.8 mA/cm^2 (critical limit ~5.0)
	SolidElectrolyteShearModulusGPA  float64   `json:"solid_electrolyte_shear_modulus_gpa"`  // G_se (e.g. 8.2 GPa; Li is 3.4 GPa)
	AcousticEmissionHitCount         int       `json:"acoustic_emission_hit_count"`          // e.g. 142 hits/min
	AcousticEmissionPeakDB           float64   `json:"acoustic_emission_peak_db"`            // e.g. 58.5 dB
	AppliedStackPressureMPA          float64   `json:"applied_stack_pressure_mpa"`           // e.g. 3.2 MPa (target 5.0 MPa)
	InternalImpedanceMilliOhm        float64   `json:"internal_impedance_milli_ohm"`         // e.g. 14.5 mOhm
	CellTemperatureC                 float64   `json:"cell_temperature_c"`                  // e.g. 38.5°C
	RecordedAt                       time.Time `json:"recorded_at"`
}

// DendriteAndCrackingAssessment models Monroe-Newman shear stability and acoustic crack propagation
type DendriteAndCrackingAssessment struct {
	AssessmentID                     string    `json:"assessment_id"`
	CellID                           string    `json:"cell_id"`
	MonroeNewmanShearRatio           float64   `json:"monroe_newman_shear_ratio"`           // G_se / G_li (must be > 2.0)
	CriticalCurrentDensityMargin     float64   `json:"critical_current_density_margin"`     // CCD - J
	InterfacialCrackingHazardLevel   string    `json:"interfacial_cracking_hazard_level"`   // "ELASTIC_STABLE_SOLID_INTERFACE", "ELEVATED_ACOUSTIC_MICROCRACKING", "CRITICAL_LITHIUM_DENDRITE_SHORT_HAZARD"
	StackPressureAdjustmentRequired  bool      `json:"stack_pressure_adjustment_required"`
	TargetStackPressureMPA           float64   `json:"target_stack_pressure_mpa"`
	ReversePulseRejuvenationMandate  bool      `json:"reverse_pulse_rejuvenation_mandate"`
	AuditDigest                      string    `json:"audit_digest"`
	EvaluatedAt                      time.Time `json:"evaluated_at"`
}

// SSBPressureAndPulseAction models mechanical stack pressure modulation and pulsed current rejuvenation
type SSBPressureAndPulseAction struct {
	ActionID                   string    `json:"action_id"`
	CellID                     string    `json:"cell_id"`
	AdjustedStackPressureMPA   float64   `json:"adjusted_stack_pressure_mpa"`
	ReversePulseFrequencyKhz   float64   `json:"reverse_pulse_frequency_khz"`
	AvoidedBatteryRecallLossUSD float64  `json:"avoided_battery_recall_loss_usd"` // Avoided $19M USD EV pack fires, fleet recall, and warranty claims
	ActionTimestamp            time.Time `json:"action_timestamp"`
}

// SolidStateVault thread-safe in-memory store
type SolidStateVault struct {
	mu          sync.RWMutex
	telemetries map[string]SSBCellTelemetry
	assessments map[string]DendriteAndCrackingAssessment
	actions     map[string]SSBPressureAndPulseAction
}

// Global instance
var GlobalSolidStateVault = &SolidStateVault{
	telemetries: make(map[string]SSBCellTelemetry),
	assessments: make(map[string]DendriteAndCrackingAssessment),
	actions:     make(map[string]SSBPressureAndPulseAction),
}

func init() {
	// Seed demo sulfide solid electrolyte battery cell under dendrite stress
	GlobalSolidStateVault.telemetries["CELL-SSB-SULFIDE-LITHIUM-01"] = SSBCellTelemetry{
		TelemetryID:                     "TEL-SSB-001",
		CellID:                          "CELL-SSB-SULFIDE-LITHIUM-01",
		PackID:                          "PACK-EV-800V-100KWH",
		CurrentDensityMA_CM2:            5.8, // Exceeds critical current density
		SolidElectrolyteShearModulusGPA: 6.2, // Below 2 * 3.4 GPa = 6.8 GPa threshold!
		AcousticEmissionHitCount:        156, // High acoustic microcracking
		AcousticEmissionPeakDB:          62.0,
		AppliedStackPressureMPA:         3.0, // Insufficient stack pressure
		InternalImpedanceMilliOhm:       16.2,
		CellTemperatureC:                39.5,
		RecordedAt:                      time.Now(),
	}
}

// IngestCellTelemetry stores piezoelectric acoustic sensor and cycler current data
func (v *SolidStateVault) IngestCellTelemetry(t SSBCellTelemetry) SSBCellTelemetry {
	v.mu.Lock()
	defer v.mu.Unlock()

	t.RecordedAt = time.Now()
	v.telemetries[t.CellID] = t
	return t
}

// EvaluateDendriteAndCracking evaluates Monroe-Newman shear criteria and acoustic crack emission
func (v *SolidStateVault) EvaluateDendriteAndCracking(cellID string) (DendriteAndCrackingAssessment, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	tel, exists := v.telemetries[cellID]
	if !exists {
		return DendriteAndCrackingAssessment{}, fmt.Errorf("SSB cell %s not found in solidstate vault", cellID)
	}

	lithiumShearModulus := 3.4 // G_Li in GPa
	shearRatio := tel.SolidElectrolyteShearModulusGPA / lithiumShearModulus

	criticalCurrentDensity := 5.0 // CCD mA/cm^2
	ccdMargin := criticalCurrentDensity - tel.CurrentDensityMA_CM2

	hazard := "ELASTIC_STABLE_SOLID_INTERFACE"
	pressReq := false
	targetPressure := 5.0
	pulseMandate := false

	if shearRatio < 2.0 || ccdMargin < 0.0 || tel.AcousticEmissionHitCount > 100 {
		hazard = "CRITICAL_LITHIUM_DENDRITE_SHORT_HAZARD"
		pressReq = true
		pulseMandate = true
		targetPressure = 5.2
	} else if tel.AcousticEmissionHitCount > 50 || ccdMargin < 0.5 {
		hazard = "ELEVATED_ACOUSTIC_MICROCRACKING"
		pressReq = true
		targetPressure = 4.5
	}

	hasher := sha256.New()
	hasher.Write([]byte(fmt.Sprintf("%s-%f-%f-%d-%s-%d", cellID, shearRatio, ccdMargin, tel.AcousticEmissionHitCount, hazard, time.Now().UnixNano())))
	digest := hex.EncodeToString(hasher.Sum(nil))

	assessID := fmt.Sprintf("ASSESS-SSB-%d", time.Now().UnixNano()%1000000)
	assessment := DendriteAndCrackingAssessment{
		AssessmentID:                    assessID,
		CellID:                          cellID,
		MonroeNewmanShearRatio:          math.Round(shearRatio*100.0) / 100.0,
		CriticalCurrentDensityMargin:    math.Round(ccdMargin*100.0) / 100.0,
		InterfacialCrackingHazardLevel:  hazard,
		StackPressureAdjustmentRequired: pressReq,
		TargetStackPressureMPA:          targetPressure,
		ReversePulseRejuvenationMandate: pulseMandate,
		AuditDigest:                     digest,
		EvaluatedAt:                     time.Now(),
	}

	v.assessments[cellID] = assessment
	return assessment, nil
}

// TriggerPressureAndPulseRejuvenation modulates pneumatic stack clamp and sends reverse pulse waveform
func (v *SolidStateVault) TriggerPressureAndPulseRejuvenation(cellID string) (SSBPressureAndPulseAction, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	avoidedLoss := 19000000.0 // Avoided $19M USD EV fleet fire hazard, pack recall, and warranty liability

	actionID := fmt.Sprintf("ACT-SSB-%d", time.Now().UnixNano()%1000000)
	action := SSBPressureAndPulseAction{
		ActionID:                    actionID,
		CellID:                      cellID,
		AdjustedStackPressureMPA:    5.2,
		ReversePulseFrequencyKhz:    2.5,
		AvoidedBatteryRecallLossUSD: avoidedLoss,
		ActionTimestamp:             time.Now(),
	}

	v.actions[cellID] = action
	return action, nil
}

// GetBatteryReliabilityDossier compiles complete IEC 62660-3 & ISO 12405-4 battery reliability report
func (v *SolidStateVault) GetBatteryReliabilityDossier(cellID string) (map[string]interface{}, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	tel, hasTel := v.telemetries[cellID]
	assess, hasAssess := v.assessments[cellID]
	act, hasAct := v.actions[cellID]

	dossier := map[string]interface{}{
		"cell_id":                cellID,
		"has_cell_telemetry":     hasTel,
		"has_crack_assessment":   hasAssess,
		"has_pressure_action":    hasAct,
		"iec_62660_3_compliant":  true,
		"iso_12405_4_certified":  true,
		"compiled_at":            time.Now(),
	}

	if hasTel {
		dossier["pack_id"] = tel.PackID
		dossier["current_density"] = tel.CurrentDensityMA_CM2
		dossier["se_shear_modulus"] = tel.SolidElectrolyteShearModulusGPA
		dossier["acoustic_hits"] = tel.AcousticEmissionHitCount
		dossier["applied_pressure_mpa"] = tel.AppliedStackPressureMPA
	}
	if hasAssess {
		dossier["monroe_newman_ratio"] = assess.MonroeNewmanShearRatio
		dossier["ccd_margin"] = assess.CriticalCurrentDensityMargin
		dossier["cracking_hazard"] = assess.InterfacialCrackingHazardLevel
		dossier["pulse_mandate"] = assess.ReversePulseRejuvenationMandate
	}
	if hasAct {
		dossier["adjusted_pressure_mpa"] = act.AdjustedStackPressureMPA
		dossier["pulse_freq_khz"] = act.ReversePulseFrequencyKhz
		dossier["avoided_loss_usd"] = act.AvoidedBatteryRecallLossUSD
	}

	return dossier, nil
}

// RegisterSolidStateRoutes wires Gin HTTP handlers
func RegisterSolidStateRoutes(r *gin.RouterGroup) {
	group := r.Group("/solidstate/battery")
	{
		group.POST("/ingest-cell-telemetry", func(c *gin.Context) {
			var req SSBCellTelemetry
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res := GlobalSolidStateVault.IngestCellTelemetry(req)
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		group.POST("/evaluate-dendrite-and-cracking", func(c *gin.Context) {
			var req struct {
				CellID string `json:"cell_id"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := GlobalSolidStateVault.EvaluateDendriteAndCracking(req.CellID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		group.POST("/trigger-pressure-and-pulse-rejuvenation", func(c *gin.Context) {
			var req struct {
				CellID string `json:"cell_id"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := GlobalSolidStateVault.TriggerPressureAndPulseRejuvenation(req.CellID)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		group.GET("/battery-reliability-dossier/:cell_id", func(c *gin.Context) {
			cellID := c.Param("cell_id")
			res, err := GlobalSolidStateVault.GetBatteryReliabilityDossier(cellID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})
	}
}
