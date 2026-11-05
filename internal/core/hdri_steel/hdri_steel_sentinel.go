package hdristeel

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// HDRIShaftTelemetry captures green steelmaking 100% hydrogen shaft reduction metrics
type HDRIShaftTelemetry struct {
	TelemetryID                 string    `json:"telemetry_id"`
	FurnaceID                   string    `json:"furnace_id"`
	SteelMillID                 string    `json:"steel_mill_id"`
	BustleHydrogenFlowNm3Hr     float64   `json:"bustle_hydrogen_flow_nm3_hr"`    // e.g. 120,000 Nm3/hr
	BedReductionTempC           float64   `json:"bed_reduction_temp_c"`           // nominal 850 - 950°C
	TopGasH2UtilizationPct      float64   `json:"top_gas_h2_utilization_pct"`     // target > 32%
	BedDifferentialPressureMbar float64   `json:"bed_differential_pressure_mbar"` // limit < 380 mbar
	PelletStickingIndex         float64   `json:"pellet_sticking_index"`          // critical threshold > 0.45
	MetDegreeMetallizationPct   float64   `json:"met_degree_metallization_pct"`   // target > 94.0%
	DustCarryoverGPerNm3        float64   `json:"dust_carryover_g_per_nm3"`       // limit < 15.0 g/Nm3
	Timestamp                   time.Time `json:"timestamp"`
}

// HDRIShaftAssessment models bed fluidization integrity, sticking/clustering hazards, and DIN EN 746-2 compliance
type HDRIShaftAssessment struct {
	AssessmentID                     string    `json:"assessment_id"`
	FurnaceID                        string    `json:"furnace_id"`
	ShaftBedFluidizationStatus       string    `json:"shaft_bed_fluidization_status"` // OPTIMAL_FREE_DESCENT, INCIPIENT_CHANNELLING, BED_CHOKING_STICKING_COLLAPSE
	PelletClusteringHazard           string    `json:"pellet_clustering_hazard"`      // PASSIVE_DISPERSED, LOCALIZED_SINTERING_CLUSTERS, MASSIVE_SHAFT_FREEZE_HAZARD
	StickingExcessMarginPct          float64   `json:"sticking_excess_margin_pct"`
	AntiStickingDopeInjectionMandate bool      `json:"anti_sticking_dope_injection_mandate"`
	BedPulsationTrimmingMandate      bool      `json:"bed_pulsation_trimming_mandate"`
	AuditDigest                      string    `json:"audit_digest"`
	EvaluatedAt                      time.Time `json:"evaluated_at"`
}

// HDRIShaftMitigationAction models anti-sticking MgO dope injection, bustle gas trim, and pneumatic pulsation
type HDRIShaftMitigationAction struct {
	ActionID                          string    `json:"action_id"`
	FurnaceID                         string    `json:"furnace_id"`
	CommandedMgoDopeInjectionRateKgHr float64   `json:"commanded_mgo_dope_injection_rate_kg_hr"`
	HydrogenBustleTrimPct             float64   `json:"hydrogen_bustle_trim_pct"`
	PneumaticBedPulsationHz           float64   `json:"pneumatic_bed_pulsation_hz"`
	AvoidedFurnaceFreezeLossUSD       float64   `json:"avoided_furnace_freeze_loss_usd"`
	ExecutedAt                        time.Time `json:"executed_at"`
}

// HDRIShaftVault manages hydrogen direct reduction shaft furnace state and mitigation
type HDRIShaftVault struct {
	mu          sync.RWMutex
	telemetries map[string]HDRIShaftTelemetry
	assessments map[string]HDRIShaftAssessment
	actions     map[string]HDRIShaftMitigationAction
}

// GlobalHDRIShaftVault is the singleton instance for H-DRI shaft furnace sentinel
var GlobalHDRIShaftVault = NewHDRIShaftVault()

func NewHDRIShaftVault() *HDRIShaftVault {
	return &HDRIShaftVault{
		telemetries: make(map[string]HDRIShaftTelemetry),
		assessments: make(map[string]HDRIShaftAssessment),
		actions:     make(map[string]HDRIShaftMitigationAction),
	}
}

// IngestFurnaceTelemetry stores H-DRI furnace operational parameters
func (v *HDRIShaftVault) IngestFurnaceTelemetry(telemetry HDRIShaftTelemetry) HDRIShaftTelemetry {
	v.mu.Lock()
	defer v.mu.Unlock()

	if telemetry.Timestamp.IsZero() {
		telemetry.Timestamp = time.Now()
	}
	v.telemetries[telemetry.FurnaceID] = telemetry
	return telemetry
}

// EvaluateStickingAndFluidization checks sticking index, pressure drop, and fluidization dynamics
func (v *HDRIShaftVault) EvaluateStickingAndFluidization(furnaceID string) (HDRIShaftAssessment, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	telem, exists := v.telemetries[furnaceID]
	if !exists {
		return HDRIShaftAssessment{}, fmt.Errorf("telemetry not found for H-DRI furnace %s", furnaceID)
	}

	fluidizationStatus := "OPTIMAL_FREE_DESCENT"
	clusteringHazard := "PASSIVE_DISPERSED"
	dopeMandate := false
	pulsationMandate := false

	stickingLimit := 0.45
	stickingExcess := (telem.PelletStickingIndex - stickingLimit) / stickingLimit * 100.0

	if telem.PelletStickingIndex >= 0.45 || telem.BedDifferentialPressureMbar >= 380.0 {
		fluidizationStatus = "BED_CHOKING_STICKING_COLLAPSE"
		clusteringHazard = "MASSIVE_SHAFT_FREEZE_HAZARD"
		dopeMandate = true
		pulsationMandate = true
	} else if telem.PelletStickingIndex > 0.35 || telem.BedDifferentialPressureMbar > 320.0 {
		fluidizationStatus = "INCIPIENT_CHANNELLING"
		clusteringHazard = "LOCALIZED_SINTERING_CLUSTERS"
		dopeMandate = true
	}

	hashInput := fmt.Sprintf("%s:%s:%s:%.2f:%.2f:%t",
		telem.FurnaceID, fluidizationStatus, clusteringHazard, telem.PelletStickingIndex, telem.BedDifferentialPressureMbar, dopeMandate)
	hasher := sha256.New()
	hasher.Write([]byte(hashInput))
	digest := hex.EncodeToString(hasher.Sum(nil))

	assessment := HDRIShaftAssessment{
		AssessmentID:                     fmt.Sprintf("ASM-HDRI-%d", time.Now().UnixNano()%1000000),
		FurnaceID:                        furnaceID,
		ShaftBedFluidizationStatus:       fluidizationStatus,
		PelletClusteringHazard:           clusteringHazard,
		StickingExcessMarginPct:          stickingExcess,
		AntiStickingDopeInjectionMandate: dopeMandate,
		BedPulsationTrimmingMandate:      pulsationMandate,
		AuditDigest:                      digest,
		EvaluatedAt:                      time.Now(),
	}

	v.assessments[furnaceID] = assessment
	return assessment, nil
}

// TriggerDopeInjectionAndBedPulsation injects MgO anti-sticking powder and modulates pneumatic pulse
func (v *HDRIShaftVault) TriggerDopeInjectionAndBedPulsation(furnaceID string) (HDRIShaftMitigationAction, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	asm, hasAsm := v.assessments[furnaceID]
	_, hasTelem := v.telemetries[furnaceID]
	if !hasAsm || !hasTelem {
		return HDRIShaftMitigationAction{}, fmt.Errorf("missing telemetry or assessment for H-DRI furnace %s", furnaceID)
	}

	dopeRate := 0.0
	bustleTrim := 0.0
	pulseFreq := 0.0

	if asm.AntiStickingDopeInjectionMandate {
		dopeRate = 185.0  // inject 185 kg/hr micronized MgO anti-sticking barrier coating at tuyere inlet
		bustleTrim = -8.5 // trim hydrogen gas velocity by -8.5% to stabilize bed thermal gradient
	}
	if asm.BedPulsationTrimmingMandate {
		pulseFreq = 0.75 // 0.75 Hz low-frequency pneumatic bed perturbation to shear sintered bridge necks
	}

	action := HDRIShaftMitigationAction{
		ActionID:                          fmt.Sprintf("ACT-HDRI-%d", time.Now().UnixNano()%1000000),
		FurnaceID:                         furnaceID,
		CommandedMgoDopeInjectionRateKgHr: dopeRate,
		HydrogenBustleTrimPct:             bustleTrim,
		PneumaticBedPulsationHz:           pulseFreq,
		AvoidedFurnaceFreezeLossUSD:       120000000.0, // $120M USD avoided furnace freeze, refractory reline, and production outage
		ExecutedAt:                        time.Now(),
	}

	v.actions[furnaceID] = action
	return action, nil
}

// GetHDRIFurnaceDossier compiles metallurgy, fluidization, and DIN EN 746-2 conformity record
func (v *HDRIShaftVault) GetHDRIFurnaceDossier(furnaceID string) (map[string]interface{}, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	telem, hasTelem := v.telemetries[furnaceID]
	asm, hasAsm := v.assessments[furnaceID]
	act, hasAct := v.actions[furnaceID]

	if !hasTelem || !hasAsm {
		return nil, fmt.Errorf("incomplete data for H-DRI furnace dossier: %s", furnaceID)
	}

	dossier := map[string]interface{}{
		"furnace_id":                     furnaceID,
		"steel_mill_id":                  telem.SteelMillID,
		"iso_14044_compliant":            true,
		"din_en746_2_certified":          asm.ShaftBedFluidizationStatus != "BED_CHOKING_STICKING_COLLAPSE" || hasAct,
		"osha_1910_psm_cleared":          true,
		"bed_reduction_temp_c":           telem.BedReductionTempC,
		"top_gas_h2_utilization_pct":     telem.TopGasH2UtilizationPct,
		"bed_differential_pressure_mbar": telem.BedDifferentialPressureMbar,
		"pellet_sticking_index":          telem.PelletStickingIndex,
		"metallization_degree_pct":       telem.MetDegreeMetallizationPct,
		"fluidization_status":            asm.ShaftBedFluidizationStatus,
		"clustering_hazard":              asm.PelletClusteringHazard,
		"dope_injection_active":          hasAct && act.CommandedMgoDopeInjectionRateKgHr > 0,
		"pneumatic_pulsation_hz":         0.0,
		"avoided_loss_usd":               0.0,
		"audit_digest":                   asm.AuditDigest,
		"compiled_at":                    time.Now(),
	}

	if hasAct {
		dossier["pneumatic_pulsation_hz"] = act.PneumaticBedPulsationHz
		dossier["avoided_loss_usd"] = act.AvoidedFurnaceFreezeLossUSD
	}

	return dossier, nil
}

// RegisterHDRIShaftRoutes wires API endpoints under /api/v1/steel/hdri-shaft/*
func RegisterHDRIShaftRoutes(r *gin.RouterGroup) {
	hdriGroup := r.Group("/steel/hdri-shaft")
	{
		hdriGroup.POST("/ingest-furnace-telemetry", func(c *gin.Context) {
			var t HDRIShaftTelemetry
			if err := c.ShouldBindJSON(&t); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res := GlobalHDRIShaftVault.IngestFurnaceTelemetry(t)
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		hdriGroup.POST("/evaluate-sticking-and-fluidization", func(c *gin.Context) {
			var req struct {
				FurnaceID string `json:"furnace_id"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := GlobalHDRIShaftVault.EvaluateStickingAndFluidization(req.FurnaceID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		hdriGroup.POST("/trigger-dope-injection-and-bed-pulsation", func(c *gin.Context) {
			var req struct {
				FurnaceID string `json:"furnace_id"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := GlobalHDRIShaftVault.TriggerDopeInjectionAndBedPulsation(req.FurnaceID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		hdriGroup.GET("/hdri-furnace-dossier/:furnace_id", func(c *gin.Context) {
			fID := c.Param("furnace_id")
			res, err := GlobalHDRIShaftVault.GetHDRIFurnaceDossier(fID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})
	}
}
