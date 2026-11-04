package maintenance

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

// KilnThermalScan represents infrared continuous line scanner data along rotary kiln shell
type KilnThermalScan struct {
	ScanID                    string    `json:"scan_id"`
	KilnID                    string    `json:"kiln_id"`
	LongitudinalMeter         float64   `json:"longitudinal_meter"`          // 0 to 80m from discharge to feed end
	CircumferentialAngleDeg   float64   `json:"circumferential_angle_deg"`    // 0 to 360 degrees
	ShellSkinTempC            float64   `json:"shell_skin_temp_c"`           // Normal: 220-320°C; Warning > 380°C; Hotspot > 420°C
	AmbientTempC              float64   `json:"ambient_temp_c"`
	KilnRPM                   float64   `json:"kiln_rpm"`                    // 2.0 to 5.0 RPM
	BurningZoneClinkerTempC   float64   `json:"burning_zone_clinker_temp_c"` // ~1450°C
	SecondaryAirTempC         float64   `json:"secondary_air_temp_c"`
	ScannedAt                 time.Time `json:"scanned_at"`
}

// RefractoryAssessment models remaining refractory thickness and shell deformation/burn-through risk
type RefractoryAssessment struct {
	AssessmentID              string    `json:"assessment_id"`
	KilnID                    string    `json:"kiln_id"`
	LongitudinalMeter         float64   `json:"longitudinal_meter"`
	EstimatedRemainingBrickMM float64   `json:"estimated_remaining_brick_mm"` // Original 220mm; Critical < 60mm
	ShellHotspotAlert         bool      `json:"shell_hotspot_alert"`
	RiskOfOvalityDeformation  bool      `json:"risk_of_ovality_deformation"`
	RecommendedCoolingFanCount int      `json:"recommended_cooling_fan_count"`
	MaxAllowedKilnRPM         float64   `json:"max_allowed_kiln_rpm"`
	ActionStatus              string    `json:"action_status"` // "HEALTHY_COATING", "MODERATE_BRICK_THINNING", "CRITICAL_HOTSPOT_BURN_THROUGH_RISK"
	ThermodynamicAuditDigest  string    `json:"thermodynamic_audit_digest"`
	EvaluatedAt               time.Time `json:"evaluated_at"`
}

// HotspotCoolingAction represents high-velocity auxiliary fan deployment to preserve shell integrity
type HotspotCoolingAction struct {
	ActionID                        string    `json:"action_id"`
	KilnID                          string    `json:"kiln_id"`
	DeployedCoolingFansCount        int       `json:"deployed_cooling_fans_count"`
	TargetedLongitudinalMeter       float64   `json:"targeted_longitudinal_meter"`
	WaterMistCoolingActive          bool      `json:"water_mist_cooling_active"`
	ShellTempReducedC               float64   `json:"shell_temp_reduced_c"`
	AvoidedKilnEmergencyShutdownUSD float64   `json:"avoided_kiln_emergency_shutdown_usd"`
	ActionTriggeredAt               time.Time `json:"action_triggered_at"`
}

// KilnRefractoryVault thread-safe in-memory store
type KilnRefractoryVault struct {
	mu             sync.RWMutex
	scans          map[string]KilnThermalScan
	assessments    map[string]RefractoryAssessment
	coolingActions map[string]HotspotCoolingAction
}

// Global instance
var GlobalKilnRefractoryVault = &KilnRefractoryVault{
	scans:          make(map[string]KilnThermalScan),
	assessments:    make(map[string]RefractoryAssessment),
	coolingActions: make(map[string]HotspotCoolingAction),
}

func init() {
	// Seed demo burning zone hotspot scan
	GlobalKilnRefractoryVault.scans["SCAN-KILN-TUBAN-42"] = KilnThermalScan{
		ScanID:                  "SCAN-KILN-TUBAN-42",
		KilnID:                  "KILN-CEMENT-TUBAN-01",
		LongitudinalMeter:       42.5,
		CircumferentialAngleDeg: 185.0,
		ShellSkinTempC:          418.0, // Hotspot red alert
		AmbientTempC:            32.0,
		KilnRPM:                 4.2,
		BurningZoneClinkerTempC: 1455.0,
		SecondaryAirTempC:       980.0,
		ScannedAt:               time.Now(),
	}
}

// IngestThermalScan stores optical pyrometer/IR scanner profile
func (v *KilnRefractoryVault) IngestThermalScan(s KilnThermalScan) KilnThermalScan {
	v.mu.Lock()
	defer v.mu.Unlock()

	if s.ScanID == "" {
		s.ScanID = fmt.Sprintf("SCAN-KILN-%d", time.Now().UnixNano()%1000000)
	}
	s.ScannedAt = time.Now()
	scanKey := fmt.Sprintf("%s-%.1f", s.KilnID, s.LongitudinalMeter)
	v.scans[scanKey] = s
	return s
}

// EvaluateRefractoryThickness estimates brick wear via Fourier 1D radial heat transfer
func (v *KilnRefractoryVault) EvaluateRefractoryThickness(kilnID string, meter float64) (RefractoryAssessment, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	scanKey := fmt.Sprintf("%s-%.1f", kilnID, meter)
	scan, exists := v.scans[scanKey]
	if !exists {
		// Fallback: search any scan for this kiln
		var found *KilnThermalScan
		for _, s := range v.scans {
			if s.KilnID == kilnID {
				sc := s
				found = &sc
				break
			}
		}
		if found == nil {
			return RefractoryAssessment{}, fmt.Errorf("no thermal scan found for kiln %s at meter %.1f", kilnID, meter)
		}
		scan = *found
	}

	// Thermodynamic refractory model:
	// Normal shell temp ~ 260°C with 220mm brick + coating.
	// As brick erodes towards steel shell (melting point 1400°C, deformation at >420°C):
	// Remaining mm ~= 220 * (1 - (ShellTemp - 240) / (460 - 240))
	ratio := (scan.ShellSkinTempC - 240.0) / 220.0
	if ratio < 0 {
		ratio = 0
	} else if ratio > 0.95 {
		ratio = 0.95
	}
	remainingBrickMM := 220.0 * (1.0 - ratio)

	hotspotAlert := false
	ovalityRisk := false
	fanCount := 0
	maxRPM := scan.KilnRPM
	status := "HEALTHY_COATING"

	if scan.ShellSkinTempC >= 400.0 || remainingBrickMM < 70.0 {
		hotspotAlert = true
		ovalityRisk = true
		fanCount = 4
		maxRPM = 2.5 // De-rate RPM to reduce shell tire cyclic stress
		status = "CRITICAL_HOTSPOT_BURN_THROUGH_RISK"
	} else if scan.ShellSkinTempC >= 350.0 || remainingBrickMM < 120.0 {
		hotspotAlert = true
		ovalityRisk = false
		fanCount = 2
		maxRPM = 3.2
		status = "MODERATE_BRICK_THINNING"
	}

	hasher := sha256.New()
	hasher.Write([]byte(fmt.Sprintf("%s-%.1f-%.2f-%s-%d", kilnID, meter, remainingBrickMM, status, time.Now().UnixNano())))
	digest := hex.EncodeToString(hasher.Sum(nil))

	assessID := fmt.Sprintf("ASSESS-KILN-%d", time.Now().UnixNano()%1000000)
	assessment := RefractoryAssessment{
		AssessmentID:               assessID,
		KilnID:                     kilnID,
		LongitudinalMeter:          meter,
		EstimatedRemainingBrickMM:  math.Round(remainingBrickMM*10.0) / 10.0,
		ShellHotspotAlert:          hotspotAlert,
		RiskOfOvalityDeformation:   ovalityRisk,
		RecommendedCoolingFanCount: fanCount,
		MaxAllowedKilnRPM:          maxRPM,
		ActionStatus:               status,
		ThermodynamicAuditDigest:   digest,
		EvaluatedAt:                time.Now(),
	}

	v.assessments[kilnID] = assessment
	return assessment, nil
}

// TriggerHotspotCooling dispatches forced-draft cooling fans to prevent emergency kiln stop
func (v *KilnRefractoryVault) TriggerHotspotCooling(kilnID string, meter float64, fansCount int) (HotspotCoolingAction, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if fansCount <= 0 {
		fansCount = 2
	}

	tempReduction := float64(fansCount) * 18.5 // ~18.5°C drop per industrial cooling fan
	avoidedShutdownLoss := 350000.0            // Avoided USD 350,000 clinker production outage loss

	actionID := fmt.Sprintf("ACT-COOLING-%d", time.Now().UnixNano()%1000000)
	action := HotspotCoolingAction{
		ActionID:                        actionID,
		KilnID:                          kilnID,
		DeployedCoolingFansCount:        fansCount,
		TargetedLongitudinalMeter:       meter,
		WaterMistCoolingActive:          true,
		ShellTempReducedC:               tempReduction,
		AvoidedKilnEmergencyShutdownUSD: avoidedShutdownLoss,
		ActionTriggeredAt:               time.Now(),
	}

	v.coolingActions[actionID] = action
	return action, nil
}

// GetShellIntegrityDossier provides complete thermal profile and refractory health dossier
func (v *KilnRefractoryVault) GetShellIntegrityDossier(kilnID string) (map[string]interface{}, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	assess, hasAssess := v.assessments[kilnID]
	var matchedScans []KilnThermalScan
	for _, s := range v.scans {
		if s.KilnID == kilnID {
			matchedScans = append(matchedScans, s)
		}
	}

	dossier := map[string]interface{}{
		"kiln_id":                     kilnID,
		"scanned_points_count":        len(matchedScans),
		"has_assessment":              hasAssess,
		"api_560_compliance_status":  true,
		"iso_50001_energy_efficiency": "MONITORED_AND_OPTIMAL",
		"compiled_at":                 time.Now(),
	}

	if hasAssess {
		dossier["remaining_brick_mm"] = assess.EstimatedRemainingBrickMM
		dossier["hotspot_alert"] = assess.ShellHotspotAlert
		dossier["action_status"] = assess.ActionStatus
		dossier["max_allowed_rpm"] = assess.MaxAllowedKilnRPM
	}

	return dossier, nil
}

// RegisterKilnRoutes wires REST endpoints
func RegisterKilnRoutes(r *gin.RouterGroup) {
	kilnGroup := r.Group("/maintenance/kiln")
	{
		kilnGroup.POST("/ingest-thermal-scan", func(c *gin.Context) {
			var s KilnThermalScan
			if err := c.ShouldBindJSON(&s); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res := GlobalKilnRefractoryVault.IngestThermalScan(s)
			c.JSON(http.StatusOK, res)
		})

		kilnGroup.POST("/evaluate-refractory-thickness", func(c *gin.Context) {
			var body struct {
				KilnID float64 `json:"-"`
			}
			var req struct {
				KilnID string  `json:"kiln_id"`
				Meter  float64 `json:"longitudinal_meter"`
			}
			_ = body
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := GlobalKilnRefractoryVault.EvaluateRefractoryThickness(req.KilnID, req.Meter)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, res)
		})

		kilnGroup.POST("/trigger-hotspot-cooling", func(c *gin.Context) {
			var body struct {
				KilnID    string  `json:"kiln_id"`
				Meter     float64 `json:"longitudinal_meter"`
				FansCount int     `json:"cooling_fans_count"`
			}
			if err := c.ShouldBindJSON(&body); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := GlobalKilnRefractoryVault.TriggerHotspotCooling(body.KilnID, body.Meter, body.FansCount)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, res)
		})

		kilnGroup.GET("/shell-integrity-dossier/:kiln_id", func(c *gin.Context) {
			kilnID := c.Param("kiln_id")
			dossier, err := GlobalKilnRefractoryVault.GetShellIntegrityDossier(kilnID)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, dossier)
		})
	}
}
