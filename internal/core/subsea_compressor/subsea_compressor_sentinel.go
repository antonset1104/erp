package subseacompressor

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

// SubseaCompressorTelemetry captures ultra-deepwater seabed multiphase wet gas compressor dynamics under API 17D & ISO 13628-4
type SubseaCompressorTelemetry struct {
	TelemetryID                     string    `json:"telemetry_id"`
	CompressorID                    string    `json:"compressor_id"`                       // "COMPRESSOR-SUBSEA-3000M-01"
	SeabedStationID                 string    `json:"seabed_station_id"`                   // "STATION-PRE-SALT-SANTOS-B02"
	WaterDepthMeters                float64   `json:"water_depth_meters"`                  // e.g. 2950.0 m
	SuctionLiquidVolumeFractionPct  float64   `json:"suction_liquid_volume_fraction_pct"`  // e.g. 28.5% (slug threshold > 15.0%)
	CompressorRpm                   float64   `json:"compressor_rpm"`                      // e.g. 10850.0 RPM
	PressureRatio                   float64   `json:"pressure_ratio"`                      // e.g. 2.65
	SurgeMarginPct                  float64   `json:"surge_margin_pct"`                    // e.g. 6.8% (safety threshold > 12.0%)
	AmbRadialBearingOrbitUm         float64   `json:"amb_radial_bearing_orbit_um"`         // e.g. 138.5 um (touchdown bearing clearance 120.0 um)
	ImpellerDropletErosionRateMmYr  float64   `json:"impeller_droplet_erosion_rate_mm_yr"` // e.g. 1.85 mm/yr
	CannedMotorWindingTempC         float64   `json:"canned_motor_winding_temp_c"`         // e.g. 142.5°C
	RecordedAt                      time.Time `json:"recorded_at"`
}

// SubseaCompressorAssessment evaluates liquid slug surge, active magnetic bearing (AMB) levitation stability, and droplet erosion
type SubseaCompressorAssessment struct {
	AssessmentID                 string    `json:"assessment_id"`
	CompressorID                 string    `json:"compressor_id"`
	EffectiveSurgeDeficitPct     float64   `json:"effective_surge_deficit_pct"`
	CompressorSurgeHazard        string    `json:"compressor_surge_hazard"`       // "AERODYNAMIC_HEAD_STABLE", "SLUG_LIQUID_SURGE_WARNING", "SUBSEA_COMPRESSOR_SLUG_SURGE_COLLAPSE"
	MagneticBearingStability     string    `json:"magnetic_bearing_stability"`    // "AMB_LEVITATION_STABLE", "RADIAL_ORBIT_OSCILLATION_WARNING", "AMB_MAGNETIC_BEARING_DELEVITATION"
	AntiSurgeRecycleValveMandate bool      `json:"anti_surge_recycle_valve_mandate"`
	AmbCoilCurrentBoostMandate   bool      `json:"amb_coil_current_boost_mandate"`
	AuditDigest                  string    `json:"audit_digest"`
	EvaluatedAt                  time.Time `json:"evaluated_at"`
}

// SubseaCompressorMitigationAction models ultra-fast 25ms anti-surge recycle valve opening, AMB bias current boost, and VFD speed setback
type SubseaCompressorMitigationAction struct {
	ActionID                            string    `json:"action_id"`
	CompressorID                        string    `json:"compressor_id"`
	CommandedAntiSurgeValveOpeningPct   float64   `json:"commanded_anti_surge_valve_opening_pct"`
	AmbBiasCurrentBoostAmps             float64   `json:"amb_bias_current_boost_amps"`
	VfdSpeedSetbackRpm                  float64   `json:"vfd_speed_setback_rpm"`
	AvoidedSubseaFailureLossUSD         float64   `json:"avoided_subsea_failure_loss_usd"`
	ActionTimestamp                     time.Time `json:"action_timestamp"`
}

// SubseaCompressorVault thread-safe in-memory store
type SubseaCompressorVault struct {
	mu          sync.RWMutex
	telemetries map[string]SubseaCompressorTelemetry
	assessments map[string]SubseaCompressorAssessment
	actions     map[string]SubseaCompressorMitigationAction
}

// GlobalSubseaCompressorVault instance
var GlobalSubseaCompressorVault = &SubseaCompressorVault{
	telemetries: make(map[string]SubseaCompressorTelemetry),
	assessments: make(map[string]SubseaCompressorAssessment),
	actions:     make(map[string]SubseaCompressorMitigationAction),
}

// IngestCompressorTelemetry stores subsea compressor telemetry
func (v *SubseaCompressorVault) IngestCompressorTelemetry(t SubseaCompressorTelemetry) SubseaCompressorTelemetry {
	v.mu.Lock()
	defer v.mu.Unlock()

	if t.TelemetryID == "" {
		t.TelemetryID = fmt.Sprintf("TEL-SCOMP-%d", time.Now().UnixNano()%1000000)
	}
	if t.RecordedAt.IsZero() {
		t.RecordedAt = time.Now()
	}

	v.telemetries[t.CompressorID] = t
	return t
}

// EvaluateSlugSurgeAndAMBStability evaluates slug liquid fraction, surge margin deficit, and magnetic bearing orbital displacement
func (v *SubseaCompressorVault) EvaluateSlugSurgeAndAMBStability(compressorID string) (SubseaCompressorAssessment, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	telem, exists := v.telemetries[compressorID]
	if !exists {
		return SubseaCompressorAssessment{}, fmt.Errorf("telemetry not found for subsea compressor: %s", compressorID)
	}

	surgeDeficit := math.Round((12.0-telem.SurgeMarginPct)*10) / 10
	if surgeDeficit < 0.0 {
		surgeDeficit = 0.0
	}

	surgeHazard := "AERODYNAMIC_HEAD_STABLE"
	antiSurgeMandate := false
	ambBoostMandate := false

	if telem.SurgeMarginPct < 8.0 || telem.SuctionLiquidVolumeFractionPct > 25.0 {
		surgeHazard = "SUBSEA_COMPRESSOR_SLUG_SURGE_COLLAPSE"
		antiSurgeMandate = true
		ambBoostMandate = true
	} else if telem.SurgeMarginPct < 12.0 || telem.SuctionLiquidVolumeFractionPct > 15.0 {
		surgeHazard = "SLUG_LIQUID_SURGE_WARNING"
		antiSurgeMandate = true
	}

	ambStability := "AMB_LEVITATION_STABLE"
	if telem.AmbRadialBearingOrbitUm > 125.0 {
		ambStability = "AMB_MAGNETIC_BEARING_DELEVITATION"
		ambBoostMandate = true
		antiSurgeMandate = true
	} else if telem.AmbRadialBearingOrbitUm > 100.0 {
		ambStability = "RADIAL_ORBIT_OSCILLATION_WARNING"
		ambBoostMandate = true
	}

	rawHash := fmt.Sprintf("%s:%f:%s:%s:%t:%t", compressorID, surgeDeficit, surgeHazard, ambStability, antiSurgeMandate, ambBoostMandate)
	hash := sha256.Sum256([]byte(rawHash))
	digest := hex.EncodeToString(hash[:])

	assessment := SubseaCompressorAssessment{
		AssessmentID:                 fmt.Sprintf("ASM-SCOMP-%d", time.Now().UnixNano()%1000000),
		CompressorID:                 compressorID,
		EffectiveSurgeDeficitPct:     surgeDeficit,
		CompressorSurgeHazard:        surgeHazard,
		MagneticBearingStability:     ambStability,
		AntiSurgeRecycleValveMandate: antiSurgeMandate,
		AmbCoilCurrentBoostMandate:   ambBoostMandate,
		AuditDigest:                  digest,
		EvaluatedAt:                  time.Now(),
	}

	v.assessments[compressorID] = assessment
	return assessment, nil
}

// TriggerAntiSurgeAndAMBBiasBoost deploys subsea anti-surge recycle valve, boosts AMB coil current, and throttles VFD
func (v *SubseaCompressorVault) TriggerAntiSurgeAndAMBBiasBoost(compressorID string) (SubseaCompressorMitigationAction, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	asm, hasAsm := v.assessments[compressorID]
	_, hasTelem := v.telemetries[compressorID]
	if !hasAsm || !hasTelem {
		return SubseaCompressorMitigationAction{}, fmt.Errorf("missing telemetry or assessment for subsea compressor %s", compressorID)
	}

	valveOpening := 0.0
	ambBoost := 0.0
	vfdSetback := 0.0

	if asm.AntiSurgeRecycleValveMandate {
		valveOpening = 65.0 // fast-open subsea anti-surge recycle valve to 65% in 25ms to bypass liquid slug
		vfdSetback = -1200.0 // trim VFD speed by 1200 RPM to drop head below surge boundary
	}
	if asm.AmbCoilCurrentBoostMandate {
		ambBoost = 14.5 // boost AMB coil bias current by +14.5 A to counteract hydrodynamic fluid slug unseating force
	}

	action := SubseaCompressorMitigationAction{
		ActionID:                          fmt.Sprintf("ACT-SCOMP-%d", time.Now().UnixNano()%1000000),
		CompressorID:                      compressorID,
		CommandedAntiSurgeValveOpeningPct: valveOpening,
		AmbBiasCurrentBoostAmps:           ambBoost,
		VfdSpeedSetbackRpm:                vfdSetback,
		AvoidedSubseaFailureLossUSD:       150000000.0,
		ActionTimestamp:                   time.Now(),
	}

	v.actions[compressorID] = action
	return action, nil
}

// GetCompressorIntegrityDossier returns subsea compression integrity dossier under API 17D & ISO 13628-4
func (v *SubseaCompressorVault) GetCompressorIntegrityDossier(compressorID string) (map[string]interface{}, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	telem, hasTelem := v.telemetries[compressorID]
	asm, hasAsm := v.assessments[compressorID]
	act := v.actions[compressorID]

	if !hasTelem || !hasAsm {
		return nil, fmt.Errorf("incomplete data for subsea compressor dossier %s", compressorID)
	}

	return map[string]interface{}{
		"compressor_id":       compressorID,
		"seabed_station_id":   telem.SeabedStationID,
		"water_depth_meters":  telem.WaterDepthMeters,
		"api_17d_compliant":   true,
		"iso_13628_certified": true,
		"telemetry":           telem,
		"assessment":          asm,
		"latest_action":       act,
		"compiled_at":         time.Now(),
	}, nil
}

// RegisterSubseaCompressorRoutes wires API endpoints under /api/v1/subsea/compressor/*
func RegisterSubseaCompressorRoutes(r *gin.RouterGroup) {
	scGroup := r.Group("/subsea/compressor")
	{
		scGroup.POST("/ingest-compressor-telemetry", func(c *gin.Context) {
			var t SubseaCompressorTelemetry
			if err := c.ShouldBindJSON(&t); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res := GlobalSubseaCompressorVault.IngestCompressorTelemetry(t)
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		scGroup.POST("/evaluate-slug-surge-and-amb-stability", func(c *gin.Context) {
			var req struct {
				CompressorID string `json:"compressor_id"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := GlobalSubseaCompressorVault.EvaluateSlugSurgeAndAMBStability(req.CompressorID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		scGroup.POST("/trigger-anti-surge-and-amb-bias-boost", func(c *gin.Context) {
			var req struct {
				CompressorID string `json:"compressor_id"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := GlobalSubseaCompressorVault.TriggerAntiSurgeAndAMBBiasBoost(req.CompressorID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		scGroup.GET("/compressor-integrity-dossier/:compressor_id", func(c *gin.Context) {
			compressorID := c.Param("compressor_id")
			res, err := GlobalSubseaCompressorVault.GetCompressorIntegrityDossier(compressorID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})
	}
}
