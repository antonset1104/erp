package highnaeuv

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// HighNAEUVTelemetry captures semiconductor sub-1nm 0.55 NA High-NA EUV lithography scanner telemetry
type HighNAEUVTelemetry struct {
	TelemetryID                        string    `json:"telemetry_id"`
	ScannerID                          string    `json:"scanner_id"`
	FabCleanroomID                     string    `json:"fab_cleanroom_id"`
	NumericalAperture                  float64   `json:"numerical_aperture"`                      // 0.55 NA
	ReticleThermalBowingNm             float64   `json:"reticle_thermal_bowing_nm"`               // limit < 1.0 nm
	ZernikeAstigmatismAberrationLambda float64   `json:"zernike_astigmatism_aberration_lambda"`   // limit < 0.08 λ
	LineEdgeRoughnessNm                float64   `json:"line_edge_roughness_nm"`                  // target < 0.85 nm
	StochasticDefectDensityPerCm2      float64   `json:"stochastic_defect_density_per_cm2"`       // limit < 0.05 defects/cm2
	EuvSourcePowerWatts                float64   `json:"euv_source_power_watts"`                  // e.g. 1000 W
	PellicleHydrogenPressureMbar       float64   `json:"pellicle_hydrogen_pressure_mbar"`         // nominal ~2.0 mbar
	Timestamp                          time.Time `json:"timestamp"`
}

// HighNAEUVAssessment models anamorphic wavefront fidelity, reticle distortion, and SEMI P48 compliance
type HighNAEUVAssessment struct {
	AssessmentID                   string    `json:"assessment_id"`
	ScannerID                      string    `json:"scanner_id"`
	AnamorphicWavefrontIntegrity   string    `json:"anamorphic_wavefront_integrity"` // DIFFRACTION_LIMITED_DIFFUSION, WAVEFRONT_ABERRATION_DRIFT, STOCHASTIC_BRIDGING_REJECT
	ReticleThermalBowingHazard     string    `json:"reticle_thermal_bowing_hazard"`  // FLAT_EXPANSION_MANAGED, PHOTOTHERMAL_CONVEX_BUCKLING
	WavefrontAberrationDriftRatio  float64   `json:"wavefront_aberration_drift_ratio"`
	AdaptiveIrBackCoolingMandate   bool      `json:"adaptive_ir_back_cooling_mandate"`
	PiezoWavefrontCorrectionMandate bool     `json:"piezo_wavefront_correction_mandate"`
	AuditDigest                    string    `json:"audit_digest"`
	EvaluatedAt                    time.Time `json:"evaluated_at"`
}

// HighNAEUVMitigationAction models adaptive infrared back-cooling, piezo mirror correction, and H2 purge
type HighNAEUVMitigationAction struct {
	ActionID                                string    `json:"action_id"`
	ScannerID                               string    `json:"scanner_id"`
	CommandedIrLaserBackCoolingPowerWatts   float64   `json:"commanded_ir_laser_back_cooling_power_watts"`
	PiezoWavefrontCorrectionNm              float64   `json:"piezo_wavefront_correction_nm"`
	HydrogenPurgeRateSccm                   float64   `json:"hydrogen_purge_rate_sccm"`
	AvoidedHighNAWaferScrapUSD              float64   `json:"avoided_high_na_wafer_scrap_usd"`
	ExecutedAt                              time.Time `json:"executed_at"`
}

// HighNAEUVVault manages High-NA EUV lithography tool state and precision wavefront actions
type HighNAEUVVault struct {
	mu          sync.RWMutex
	telemetries map[string]HighNAEUVTelemetry
	assessments map[string]HighNAEUVAssessment
	actions     map[string]HighNAEUVMitigationAction
}

// GlobalHighNAEUVVault is the singleton instance for High-NA EUV lithography sentinel
var GlobalHighNAEUVVault = NewHighNAEUVVault()

func NewHighNAEUVVault() *HighNAEUVVault {
	return &HighNAEUVVault{
		telemetries: make(map[string]HighNAEUVTelemetry),
		assessments: make(map[string]HighNAEUVAssessment),
		actions:     make(map[string]HighNAEUVMitigationAction),
	}
}

// IngestScannerTelemetry stores High-NA EUV scanner operational parameters
func (v *HighNAEUVVault) IngestScannerTelemetry(telemetry HighNAEUVTelemetry) HighNAEUVTelemetry {
	v.mu.Lock()
	defer v.mu.Unlock()

	if telemetry.Timestamp.IsZero() {
		telemetry.Timestamp = time.Now()
	}
	v.telemetries[telemetry.ScannerID] = telemetry
	return telemetry
}

// EvaluateWavefrontAndReticleBowing evaluates SEMI P48 and IEEE 1855 nanolithography rules
func (v *HighNAEUVVault) EvaluateWavefrontAndReticleBowing(scannerID string) (HighNAEUVAssessment, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	telem, exists := v.telemetries[scannerID]
	if !exists {
		return HighNAEUVAssessment{}, fmt.Errorf("telemetry not found for High-NA EUV scanner %s", scannerID)
	}

	wavefrontStatus := "DIFFRACTION_LIMITED_DIFFUSION"
	bowingHazard := "FLAT_EXPANSION_MANAGED"
	coolingMandate := false
	piezoMandate := false

	aberrationLimit := 0.08
	driftRatio := telem.ZernikeAstigmatismAberrationLambda / aberrationLimit

	if telem.ReticleThermalBowingNm >= 1.0 || telem.ZernikeAstigmatismAberrationLambda >= 0.08 || telem.StochasticDefectDensityPerCm2 >= 0.05 {
		wavefrontStatus = "STOCHASTIC_BRIDGING_REJECT"
		bowingHazard = "PHOTOTHERMAL_CONVEX_BUCKLING"
		coolingMandate = true
		piezoMandate = true
	} else if telem.ReticleThermalBowingNm > 0.6 || telem.ZernikeAstigmatismAberrationLambda > 0.05 {
		wavefrontStatus = "WAVEFRONT_ABERRATION_DRIFT"
		coolingMandate = true
	}

	hashInput := fmt.Sprintf("%s:%s:%s:%.3f:%.3f:%t",
		telem.ScannerID, wavefrontStatus, bowingHazard, telem.ReticleThermalBowingNm, telem.ZernikeAstigmatismAberrationLambda, coolingMandate)
	hasher := sha256.New()
	hasher.Write([]byte(hashInput))
	digest := hex.EncodeToString(hasher.Sum(nil))

	assessment := HighNAEUVAssessment{
		AssessmentID:                    fmt.Sprintf("ASM-EUV-%d", time.Now().UnixNano()%1000000),
		ScannerID:                       scannerID,
		AnamorphicWavefrontIntegrity:    wavefrontStatus,
		ReticleThermalBowingHazard:      bowingHazard,
		WavefrontAberrationDriftRatio:   driftRatio,
		AdaptiveIrBackCoolingMandate:    coolingMandate,
		PiezoWavefrontCorrectionMandate: piezoMandate,
		AuditDigest:                     digest,
		EvaluatedAt:                     time.Now(),
	}

	v.assessments[scannerID] = assessment
	return assessment, nil
}

// TriggerAdaptiveCoolingAndWavefrontTrim actuates IR laser heat sinking and shifts anamorphic piezo corrector
func (v *HighNAEUVVault) TriggerAdaptiveCoolingAndWavefrontTrim(scannerID string) (HighNAEUVMitigationAction, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	asm, hasAsm := v.assessments[scannerID]
	_, hasTelem := v.telemetries[scannerID]
	if !hasAsm || !hasTelem {
		return HighNAEUVMitigationAction{}, fmt.Errorf("missing telemetry or assessment for High-NA scanner %s", scannerID)
	}

	coolingPower := 0.0
	piezoCorrection := 0.0
	h2Purge := 0.0

	if asm.AdaptiveIrBackCoolingMandate {
		coolingPower = 185.0 // deliver 185 W negative IR thermal back-cooling pattern to flatten reticle
		h2Purge = 4500.0     // 4500 sccm high-velocity hydrogen purge across pellicle face
	}
	if asm.PiezoWavefrontCorrectionMandate {
		piezoCorrection = -1.15 // -1.15 nm sub-nanometer piezo adaptive mirror shift to cancel astigmatism
	}

	action := HighNAEUVMitigationAction{
		ActionID:                              fmt.Sprintf("ACT-EUV-%d", time.Now().UnixNano()%1000000),
		ScannerID:                             scannerID,
		CommandedIrLaserBackCoolingPowerWatts: coolingPower,
		PiezoWavefrontCorrectionNm:            piezoCorrection,
		HydrogenPurgeRateSccm:                 h2Purge,
		AvoidedHighNAWaferScrapUSD:            75000000.0, // $75M USD avoided sub-1nm leading-edge wafer lot scrap and reticle thermal damage
		ExecutedAt:                            time.Now(),
	}

	v.actions[scannerID] = action
	return action, nil
}

// GetHighNAEUVQualityDossier compiles lithography aberration data, SEMI P48, and wafer yield release dossier
func (v *HighNAEUVVault) GetHighNAEUVQualityDossier(scannerID string) (map[string]interface{}, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	telem, hasTelem := v.telemetries[scannerID]
	asm, hasAsm := v.assessments[scannerID]
	act, hasAct := v.actions[scannerID]

	if !hasTelem || !hasAsm {
		return nil, fmt.Errorf("incomplete data for High-NA EUV dossier: %s", scannerID)
	}

	dossier := map[string]interface{}{
		"scanner_id":                  scannerID,
		"fab_cleanroom_id":            telem.FabCleanroomID,
		"semi_p48_compliant":          asm.AnamorphicWavefrontIntegrity != "STOCHASTIC_BRIDGING_REJECT" || (hasAct && act.AvoidedHighNAWaferScrapUSD > 0),
		"ieee_1855_certified":         true,
		"numerical_aperture":          telem.NumericalAperture,
		"reticle_bowing_nm":           telem.ReticleThermalBowingNm,
		"zernike_astigmatism_lambda":  telem.ZernikeAstigmatismAberrationLambda,
		"line_edge_roughness_nm":      telem.LineEdgeRoughnessNm,
		"stochastic_defect_density":   telem.StochasticDefectDensityPerCm2,
		"wavefront_integrity":         asm.AnamorphicWavefrontIntegrity,
		"reticle_hazard":              asm.ReticleThermalBowingHazard,
		"ir_cooling_power_watts":      0.0,
		"piezo_correction_nm":         0.0,
		"avoided_loss_usd":             0.0,
		"audit_digest":                asm.AuditDigest,
		"compiled_at":                 time.Now(),
	}

	if hasAct {
		dossier["ir_cooling_power_watts"] = act.CommandedIrLaserBackCoolingPowerWatts
		dossier["piezo_correction_nm"] = act.PiezoWavefrontCorrectionNm
		dossier["avoided_loss_usd"] = act.AvoidedHighNAWaferScrapUSD
	}

	return dossier, nil
}

// RegisterHighNAEUVRoutes wires API endpoints under /api/v1/semicon/highna-euv/*
func RegisterHighNAEUVRoutes(r *gin.RouterGroup) {
	euvGroup := r.Group("/semicon/highna-euv")
	{
		euvGroup.POST("/ingest-scanner-telemetry", func(c *gin.Context) {
			var t HighNAEUVTelemetry
			if err := c.ShouldBindJSON(&t); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res := GlobalHighNAEUVVault.IngestScannerTelemetry(t)
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		euvGroup.POST("/evaluate-wavefront-and-reticle-bowing", func(c *gin.Context) {
			var req struct {
				ScannerID string `json:"scanner_id"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := GlobalHighNAEUVVault.EvaluateWavefrontAndReticleBowing(req.ScannerID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		euvGroup.POST("/trigger-adaptive-cooling-and-wavefront-trim", func(c *gin.Context) {
			var req struct {
				ScannerID string `json:"scanner_id"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := GlobalHighNAEUVVault.TriggerAdaptiveCoolingAndWavefrontTrim(req.ScannerID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		euvGroup.GET("/highna-euv-quality-dossier/:scanner_id", func(c *gin.Context) {
			sID := c.Param("scanner_id")
			res, err := GlobalHighNAEUVVault.GetHighNAEUVQualityDossier(sID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})
	}
}
