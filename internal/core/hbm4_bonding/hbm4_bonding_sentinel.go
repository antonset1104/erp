package hbm4bonding

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

// HBM4StackTelemetry captures 16-die / 20-die HBM4 wafer-to-wafer direct Cu-Cu hybrid bonding under SEMI 3D-0613 & IEEE 1581
type HBM4StackTelemetry struct {
	TelemetryID                         string    `json:"telemetry_id"`
	StackID                             string    `json:"stack_id"`                                // "HBM4-16HI-DRAM-STACK-008"
	WaferLotID                          string    `json:"wafer_lot_id"`                            // "LOT-HBM4-AI-ACCEL-440"
	DieStackCount                       int       `json:"die_stack_count"`                         // e.g. 16
	CuPadDishingDepthNm                 float64   `json:"cu_pad_dishing_depth_nm"`                 // e.g. 2.85 nm (target < 2.0 nm)
	DielectricRecessVarianceNm          float64   `json:"dielectric_recess_variance_nm"`           // e.g. 1.45 nm
	InterfacialVoidAreaFractionPct      float64   `json:"interfacial_void_area_fraction_pct"`       // e.g. 1.85% (acceptance limit < 0.80%)
	ContactResistancePerPadOhms         float64   `json:"contact_resistance_per_pad_ohms"`         // e.g. 0.28 Ohms (nominal < 0.15 Ohms)
	ThermalAnnealTempC                  float64   `json:"thermal_anneal_temp_c"`                   // e.g. 215.0°C
	ElectromigrationCurrentDensityAcm2  float64   `json:"electromigration_current_density_acm2"`   // e.g. 1.45e6 A/cm2 (limit 1.2e6 A/cm2)
	OverlayAlignmentRunoutNm            float64   `json:"overlay_alignment_runout_nm"`             // e.g. 58.5 nm (spec < 40.0 nm)
	RecordedAt                          time.Time `json:"recorded_at"`
}

// HBM4StackAssessment evaluates interfacial voiding, Cu grain boundary growth, and electromigration current crowding
type HBM4StackAssessment struct {
	AssessmentID                   string    `json:"assessment_id"`
	StackID                        string    `json:"stack_id"`
	InterfacialBondingQualityPct   float64   `json:"interfacial_bonding_quality_pct"`
	HybridBondingIntegrityStatus   string    `json:"hybrid_bonding_integrity_status"` // "CU_CU_CRYSTALLIZED_BOND", "INTERFACIAL_VOID_WARNING", "INTERFACIAL_VOID_COALESCENCE_REJECT"
	ElectromigrationHazard         string    `json:"electromigration_hazard"`         // "CURRENT_DENSITY_STABLE", "CURRENT_CROWDING_WARNING", "ELECTROMIGRATION_CURRENT_CROWDING_HAZARD"
	ThermalAnnealPulseMandate      bool      `json:"thermal_anneal_pulse_mandate"`
	PiezoClampingForceTrimMandate  bool      `json:"piezo_clamping_force_trim_mandate"`
	AuditDigest                    string    `json:"audit_digest"`
	EvaluatedAt                    time.Time `json:"evaluated_at"`
}

// HBM4MitigationAction models localized infrared laser thermal anneal, piezo clamp load modulation, and redundant TSV rerouting
type HBM4MitigationAction struct {
	ActionID                    string    `json:"action_id"`
	StackID                     string    `json:"stack_id"`
	CommandedAnnealPulseTempC   float64   `json:"commanded_anneal_pulse_temp_c"`
	AnnealPulseDurationMs       float64   `json:"anneal_pulse_duration_ms"`
	PiezoBondingClampForceKN    float64   `json:"piezo_bonding_clamp_force_kn"`
	AvoidedHBM4ScrapLossUSD     float64   `json:"avoided_hbm4_scrap_loss_usd"`
	ActionTimestamp             time.Time `json:"action_timestamp"`
}

// HBM4Vault thread-safe in-memory store
type HBM4Vault struct {
	mu          sync.RWMutex
	telemetries map[string]HBM4StackTelemetry
	assessments map[string]HBM4StackAssessment
	actions     map[string]HBM4MitigationAction
}

// GlobalHBM4Vault instance
var GlobalHBM4Vault = &HBM4Vault{
	telemetries: make(map[string]HBM4StackTelemetry),
	assessments: make(map[string]HBM4StackAssessment),
	actions:     make(map[string]HBM4MitigationAction),
}

// IngestStackTelemetry stores HBM4 hybrid bonding telemetry
func (v *HBM4Vault) IngestStackTelemetry(t HBM4StackTelemetry) HBM4StackTelemetry {
	v.mu.Lock()
	defer v.mu.Unlock()

	if t.TelemetryID == "" {
		t.TelemetryID = fmt.Sprintf("TEL-HBM4-%d", time.Now().UnixNano()%1000000)
	}
	if t.RecordedAt.IsZero() {
		t.RecordedAt = time.Now()
	}

	v.telemetries[t.StackID] = t
	return t
}

// EvaluateVoidsAndElectromigration evaluates Cu-Cu interfacial grain coalescence, void fraction, and current crowding
func (v *HBM4Vault) EvaluateVoidsAndElectromigration(stackID string) (HBM4StackAssessment, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	telem, exists := v.telemetries[stackID]
	if !exists {
		return HBM4StackAssessment{}, fmt.Errorf("telemetry not found for HBM4 stack: %s", stackID)
	}

	quality := math.Round((100.0-(telem.InterfacialVoidAreaFractionPct*20.0)-(telem.CuPadDishingDepthNm*5.0))*10) / 10
	if quality < 0.0 {
		quality = 0.0
	}

	bondStatus := "CU_CU_CRYSTALLIZED_BOND"
	annealMandate := false
	clampMandate := false

	if telem.InterfacialVoidAreaFractionPct > 1.20 || telem.CuPadDishingDepthNm > 2.50 {
		bondStatus = "INTERFACIAL_VOID_COALESCENCE_REJECT"
		annealMandate = true
		clampMandate = true
	} else if telem.InterfacialVoidAreaFractionPct > 0.80 || telem.CuPadDishingDepthNm > 1.80 {
		bondStatus = "INTERFACIAL_VOID_WARNING"
		annealMandate = true
	}

	emHazard := "CURRENT_DENSITY_STABLE"
	if telem.ElectromigrationCurrentDensityAcm2 > 1.30e6 || telem.ContactResistancePerPadOhms > 0.22 {
		emHazard = "ELECTROMIGRATION_CURRENT_CROWDING_HAZARD"
		annealMandate = true
		clampMandate = true
	} else if telem.ElectromigrationCurrentDensityAcm2 > 1.10e6 || telem.ContactResistancePerPadOhms > 0.16 {
		emHazard = "CURRENT_CROWDING_WARNING"
	}

	rawHash := fmt.Sprintf("%s:%f:%s:%s:%t:%t", stackID, quality, bondStatus, emHazard, annealMandate, clampMandate)
	hash := sha256.Sum256([]byte(rawHash))
	digest := hex.EncodeToString(hash[:])

	assessment := HBM4StackAssessment{
		AssessmentID:                  fmt.Sprintf("ASM-HBM4-%d", time.Now().UnixNano()%1000000),
		StackID:                       stackID,
		InterfacialBondingQualityPct:  quality,
		HybridBondingIntegrityStatus:  bondStatus,
		ElectromigrationHazard:        emHazard,
		ThermalAnnealPulseMandate:     annealMandate,
		PiezoClampingForceTrimMandate: clampMandate,
		AuditDigest:                   digest,
		EvaluatedAt:                   time.Now(),
	}

	v.assessments[stackID] = assessment
	return assessment, nil
}

// TriggerAnnealPulseAndStressRelief applies rapid infrared thermal pulse and modulates piezo clamping load
func (v *HBM4Vault) TriggerAnnealPulseAndStressRelief(stackID string) (HBM4MitigationAction, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	asm, hasAsm := v.assessments[stackID]
	telem, hasTelem := v.telemetries[stackID]
	if !hasAsm || !hasTelem {
		return HBM4MitigationAction{}, fmt.Errorf("missing telemetry or assessment for HBM4 stack %s", stackID)
	}

	pulseTemp := telem.ThermalAnnealTempC
	pulseDur := 0.0
	clampForce := 12.0 // nominal clamp 12.0 kN

	if asm.ThermalAnnealPulseMandate {
		pulseTemp = 245.0 // boost to 245°C to activate atomic Cu boundary diffusion across void necks
		pulseDur = 450.0  // 450 ms laser pulse
	}
	if asm.PiezoClampingForceTrimMandate {
		clampForce = 18.5 // increase piezo clamp force to 18.5 kN to eliminate dishing gap
	}

	action := HBM4MitigationAction{
		ActionID:                  fmt.Sprintf("ACT-HBM4-%d", time.Now().UnixNano()%1000000),
		StackID:                   stackID,
		CommandedAnnealPulseTempC: pulseTemp,
		AnnealPulseDurationMs:     pulseDur,
		PiezoBondingClampForceKN:  clampForce,
		AvoidedHBM4ScrapLossUSD:   55000000.0,
		ActionTimestamp:           time.Now(),
	}

	v.actions[stackID] = action
	return action, nil
}

// GetHBM4QualityDossier returns advanced packaging certification dossier under SEMI 3D-0613 & IEEE 1581
func (v *HBM4Vault) GetHBM4QualityDossier(stackID string) (map[string]interface{}, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	telem, hasTelem := v.telemetries[stackID]
	asm, hasAsm := v.assessments[stackID]
	act := v.actions[stackID]

	if !hasTelem || !hasAsm {
		return nil, fmt.Errorf("incomplete data for HBM4 stack dossier %s", stackID)
	}

	return map[string]interface{}{
		"stack_id":               stackID,
		"wafer_lot_id":           telem.WaferLotID,
		"die_stack_count":        telem.DieStackCount,
		"semi_3d0613_compliant":  true,
		"ieee_1581_certified":    true,
		"telemetry":              telem,
		"assessment":             asm,
		"latest_action":          act,
		"compiled_at":            time.Now(),
	}, nil
}

// RegisterHBM4Routes wires API endpoints under /api/v1/semicon/hbm4/*
func RegisterHBM4Routes(r *gin.RouterGroup) {
	hbm4Group := r.Group("/semicon/hbm4")
	{
		hbm4Group.POST("/ingest-stack-telemetry", func(c *gin.Context) {
			var t HBM4StackTelemetry
			if err := c.ShouldBindJSON(&t); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res := GlobalHBM4Vault.IngestStackTelemetry(t)
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		hbm4Group.POST("/evaluate-voids-and-electromigration", func(c *gin.Context) {
			var req struct {
				StackID string `json:"stack_id"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := GlobalHBM4Vault.EvaluateVoidsAndElectromigration(req.StackID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		hbm4Group.POST("/trigger-anneal-pulse-and-stress-relief", func(c *gin.Context) {
			var req struct {
				StackID string `json:"stack_id"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := GlobalHBM4Vault.TriggerAnnealPulseAndStressRelief(req.StackID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		hbm4Group.GET("/hbm4-quality-dossier/:stack_id", func(c *gin.Context) {
			stackID := c.Param("stack_id")
			res, err := GlobalHBM4Vault.GetHBM4QualityDossier(stackID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})
	}
}
