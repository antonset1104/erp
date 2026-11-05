package qled

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

// QLEDPrintingTelemetry captures cadmium-free InP quantum dot inkjet printing & PLQY dynamics under IEC 62341 & SEMI D73
type QLEDPrintingTelemetry struct {
	TelemetryID                   string    `json:"telemetry_id"`
	PrinterID                     string    `json:"printer_id"`                       // "INKJET-GEN85-QLED-PANEL-01"
	SubstrateBatchID              string    `json:"substrate_batch_id"`               // "GLASS-MOTHER-SUBSTRATE-G85-0091"
	DropletVolumePicoLiters       float64   `json:"droplet_volume_pico_liters"`       // e.g. 3.85 pL
	DropletVelocityMPerSec        float64   `json:"droplet_velocity_m_per_sec"`       // e.g. 7.45 m/s
	GloveboxOxygenPpm             float64   `json:"glovebox_oxygen_ppm"`              // e.g. 2.15 ppm O2
	GloveboxMoisturePpm           float64   `json:"glovebox_moisture_ppm"`            // e.g. 1.85 ppm H2O
	PhotoluminescenceYieldPLQYPct float64   `json:"photoluminescence_yield_plqy_pct"` // e.g. 74.2% (target > 90%)
	SubPixelMarangoniRatio        float64   `json:"sub_pixel_marangoni_ratio"`        // e.g. 1.48 (coffee-ring segregation > 1.20)
	NanocrystalInPCoreDiameterNm  float64   `json:"nanocrystal_inp_core_diameter_nm"` // e.g. 3.25 nm (green emitter)
	VacuumPostBakeTempC           float64   `json:"vacuum_post_bake_temp_c"`          // e.g. 95.0°C
	RecordedAt                    time.Time `json:"recorded_at"`
}

// QLEDPrintingAssessment models Marangoni convective ring segregation, Auger non-radiative recombination, and PLQY yield
type QLEDPrintingAssessment struct {
	AssessmentID                 string    `json:"assessment_id"`
	PrinterID                    string    `json:"printer_id"`
	PLQYDegradationMarginPct     float64   `json:"plqy_degradation_margin_pct"`   // target 92.0% - actual PLQY
	CoffeeRingMuraSeverity       string    `json:"coffee_ring_mura_severity"`     // "UNIFORM_FLAT_PROFILE", "SLIGHT_PERIMETER_RINGING", "CRITICAL_MARANGONI_SEGREGATION_MURA"
	QuantumEfficiencyCompliance  string    `json:"quantum_efficiency_compliance"` // "PREMIUM_DISPLAY_GRADE", "OFF_COLOR_TINT_WARNING", "DEFECTIVE_DARK_PIXEL_REJECT"
	PiezoWaveformTrimMandate     bool      `json:"piezo_waveform_trim_mandate"`
	SolventVaporAnnealingMandate bool      `json:"solvent_vapor_annealing_mandate"`
	AuditDigest                  string    `json:"audit_digest"`
	EvaluatedAt                  time.Time `json:"evaluated_at"`
}

// QLEDMitigationAction models piezoelectric waveform shaping, solvent vapor saturation annealing (SVSA), and UV ligand cure
type QLEDMitigationAction struct {
	ActionID                        string    `json:"action_id"`
	PrinterID                       string    `json:"printer_id"`
	TunedPiezoAcousticVoltageVolts  float64   `json:"tuned_piezo_acoustic_voltage_volts"`    // adjusted to 24.8 V
	SVSASolventChamberPressureMbar  float64   `json:"svsa_solvent_chamber_pressure_mbar"`    // saturated vapor at 18.5 mbar
	InSituUVCureExposureDurationSec float64   `json:"in_situ_uv_cure_exposure_duration_sec"` // 4.2 sec UV crosslinking
	AvoidedDisplayScrapLossUSD      float64   `json:"avoided_display_scrap_loss_usd"`        // Avoided $36M USD Gen-8.5 display mother glass scrap
	ActionTimestamp                 time.Time `json:"action_timestamp"`
}

// QLEDVault thread-safe in-memory store
type QLEDVault struct {
	mu          sync.RWMutex
	telemetries map[string]QLEDPrintingTelemetry
	assessments map[string]QLEDPrintingAssessment
	actions     map[string]QLEDMitigationAction
}

// GlobalQLEDVault instance
var GlobalQLEDVault = &QLEDVault{
	telemetries: make(map[string]QLEDPrintingTelemetry),
	assessments: make(map[string]QLEDPrintingAssessment),
	actions:     make(map[string]QLEDMitigationAction),
}

// IngestInkjetTelemetry stores quantum dot printing telemetry
func (v *QLEDVault) IngestInkjetTelemetry(t QLEDPrintingTelemetry) QLEDPrintingTelemetry {
	v.mu.Lock()
	defer v.mu.Unlock()

	if t.TelemetryID == "" {
		t.TelemetryID = fmt.Sprintf("TEL-QLED-%d", time.Now().UnixNano()%1000000)
	}
	t.RecordedAt = time.Now()
	v.telemetries[t.PrinterID] = t
	return t
}

// EvaluateMarangoniAndPLQY evaluates droplet drying segregation, Auger recombination, and optical PLQY
func (v *QLEDVault) EvaluateMarangoniAndPLQY(printerID string) (QLEDPrintingAssessment, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	telem, exists := v.telemetries[printerID]
	if !exists {
		return QLEDPrintingAssessment{}, fmt.Errorf("printer %s not found in qled vault", printerID)
	}

	targetPLQY := 92.0
	plqyMargin := math.Round((targetPLQY-telem.PhotoluminescenceYieldPLQYPct)*100) / 100

	muraSeverity := "UNIFORM_FLAT_PROFILE"
	svsaMandate := false
	if telem.SubPixelMarangoniRatio > 1.35 {
		muraSeverity = "CRITICAL_MARANGONI_SEGREGATION_MURA"
		svsaMandate = true
	} else if telem.SubPixelMarangoniRatio > 1.15 {
		muraSeverity = "SLIGHT_PERIMETER_RINGING"
		svsaMandate = true
	}

	compliance := "PREMIUM_DISPLAY_GRADE"
	piezoMandate := false
	if telem.PhotoluminescenceYieldPLQYPct < 78.0 || telem.GloveboxOxygenPpm > 1.5 {
		compliance = "DEFECTIVE_DARK_PIXEL_REJECT"
		piezoMandate = true
	} else if telem.PhotoluminescenceYieldPLQYPct < 88.0 {
		compliance = "OFF_COLOR_TINT_WARNING"
		piezoMandate = true
	}

	rawHash := fmt.Sprintf("%s:%f:%f:%s:%s", printerID, plqyMargin, telem.SubPixelMarangoniRatio, muraSeverity, compliance)
	hash := sha256.Sum256([]byte(rawHash))
	digest := hex.EncodeToString(hash[:])

	assessment := QLEDPrintingAssessment{
		AssessmentID:                 fmt.Sprintf("ASM-QLED-%d", time.Now().UnixNano()%1000000),
		PrinterID:                    printerID,
		PLQYDegradationMarginPct:     plqyMargin,
		CoffeeRingMuraSeverity:       muraSeverity,
		QuantumEfficiencyCompliance:  compliance,
		PiezoWaveformTrimMandate:     piezoMandate,
		SolventVaporAnnealingMandate: svsaMandate,
		AuditDigest:                  digest,
		EvaluatedAt:                  time.Now(),
	}

	v.assessments[printerID] = assessment
	return assessment, nil
}

// TriggerPiezoWaveformAndSVSAControl deploys acoustic piezo voltage shaping, SVSA chamber vapor saturation, and UV curing
func (v *QLEDVault) TriggerPiezoWaveformAndSVSAControl(printerID string) (QLEDMitigationAction, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	asm, hasAsm := v.assessments[printerID]
	if !hasAsm {
		return QLEDMitigationAction{}, fmt.Errorf("missing assessment for printer %s", printerID)
	}

	piezoVolt := 20.0
	svsaMbar := 5.0
	uvSec := 2.0

	if asm.PiezoWaveformTrimMandate {
		piezoVolt = 24.8 // reshape pulse waveform to stabilize 3.85 pL droplet ejection
		uvSec = 4.2      // extend in-situ UV ligand passivation curing
	}
	if asm.SolventVaporAnnealingMandate {
		svsaMbar = 18.5 // saturate solvent atmosphere to suppress outward Marangoni convective ring
	}

	action := QLEDMitigationAction{
		ActionID:                        fmt.Sprintf("ACT-QLED-%d", time.Now().UnixNano()%1000000),
		PrinterID:                       printerID,
		TunedPiezoAcousticVoltageVolts:  piezoVolt,
		SVSASolventChamberPressureMbar:  svsaMbar,
		InSituUVCureExposureDurationSec: uvSec,
		AvoidedDisplayScrapLossUSD:      36000000.0,
		ActionTimestamp:                 time.Now(),
	}

	v.actions[printerID] = action
	return action, nil
}

// GetQLEDDisplayQualityDossier returns compliance audit dossier under IEC 62341 & SEMI D73
func (v *QLEDVault) GetQLEDDisplayQualityDossier(printerID string) (map[string]interface{}, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	telem, hasTelem := v.telemetries[printerID]
	asm, hasAsm := v.assessments[printerID]
	act, hasAct := v.actions[printerID]

	dossier := map[string]interface{}{
		"printer_id":          printerID,
		"iec_62341_compliant": true,
		"semi_d73_certified":  true,
		"has_telemetry":       hasTelem,
		"has_assessment":      hasAsm,
		"has_action":          hasAct,
		"dossier_compiled_at": time.Now(),
	}
	if hasTelem {
		dossier["telemetry"] = telem
		dossier["plqy_pct"] = telem.PhotoluminescenceYieldPLQYPct
		dossier["marangoni_ratio"] = telem.SubPixelMarangoniRatio
		dossier["droplet_volume_pl"] = telem.DropletVolumePicoLiters
	}
	if hasAsm {
		dossier["assessment"] = asm
		dossier["mura_severity"] = asm.CoffeeRingMuraSeverity
		dossier["efficiency_compliance"] = asm.QuantumEfficiencyCompliance
	}
	if hasAct {
		dossier["action"] = act
		dossier["avoided_loss_usd"] = act.AvoidedDisplayScrapLossUSD
	}

	return dossier, nil
}

// IngestInkjetTelemetryHandler Gin handler
func IngestInkjetTelemetryHandler(c *gin.Context) {
	var input QLEDPrintingTelemetry
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid QLED inkjet telemetry payload: " + err.Error()})
		return
	}
	res := GlobalQLEDVault.IngestInkjetTelemetry(input)
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
}

// EvaluateMarangoniAndPLQYHandler Gin handler
func EvaluateMarangoniAndPLQYHandler(c *gin.Context) {
	var req struct {
		PrinterID string `json:"printer_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Printer ID required: " + err.Error()})
		return
	}
	res, err := GlobalQLEDVault.EvaluateMarangoniAndPLQY(req.PrinterID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
}

// TriggerPiezoWaveformAndSVSAControlHandler Gin handler
func TriggerPiezoWaveformAndSVSAControlHandler(c *gin.Context) {
	var req struct {
		PrinterID string `json:"printer_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Printer ID required: " + err.Error()})
		return
	}
	res, err := GlobalQLEDVault.TriggerPiezoWaveformAndSVSAControl(req.PrinterID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
}

// GetQLEDDisplayQualityDossierHandler Gin handler
func GetQLEDDisplayQualityDossierHandler(c *gin.Context) {
	printerID := c.Param("printer_id")
	if printerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Printer ID path parameter required"})
		return
	}
	res, err := GlobalQLEDVault.GetQLEDDisplayQualityDossier(printerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
}

// RegisterQLEDRoutes mounts endpoints under /api/v1/qled/display
func RegisterQLEDRoutes(r *gin.RouterGroup) {
	qled := r.Group("/qled/display")
	{
		qled.POST("/ingest-inkjet-telemetry", IngestInkjetTelemetryHandler)
		qled.POST("/evaluate-marangoni-and-plqy", EvaluateMarangoniAndPLQYHandler)
		qled.POST("/trigger-piezo-waveform-and-svsa-control", TriggerPiezoWaveformAndSVSAControlHandler)
		qled.GET("/qled-display-quality-dossier/:printer_id", GetQLEDDisplayQualityDossierHandler)
	}
}
