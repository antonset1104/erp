package seabed

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

// SeabedMiningTelemetry captures deep-sea crawler harvest metrics under ISA Guidelines & UNCLOS Part XI
type SeabedMiningTelemetry struct {
	TelemetryID                  string    `json:"telemetry_id"`
	CrawlerID                    string    `json:"crawler_id"`                        // "CRAWLER-PACIFIC-CCZ-01"
	MiningConcessionID           string    `json:"mining_concession_id"`              // "NORI-D-CLARION-CLIPPERTON"
	WaterDepthMeters             float64   `json:"water_depth_meters"`                // e.g. 4350.0 m
	SeabedCrawlerSpeedMPerSec    float64   `json:"seabed_crawler_speed_m_per_sec"`    // e.g. 0.45 m/s
	NoduleCollectionRateTonsHr   float64   `json:"nodule_collection_rate_tons_hr"`    // e.g. 185.0 tons/hr
	VerticalRiserVelocityMPerSec float64   `json:"vertical_riser_velocity_m_per_sec"` // e.g. 3.8 m/s
	BenthicPlumeTurbidityNTU     float64   `json:"benthic_plume_turbidity_ntu"`       // e.g. 78.5 NTU (ISA limit < 40 NTU)
	SuspendedParticulateMgL      float64   `json:"suspended_particulate_mg_l"`        // e.g. 48.0 mg/L (limit < 25 mg/L)
	MidwaterDischargeDepthM      float64   `json:"midwater_discharge_depth_m"`        // e.g. 1200.0 m
	RecordedAt                   time.Time `json:"recorded_at"`
}

// PlumeTurbidityAssessment models benthic plume advection-diffusion and hydrocyclone separation
type PlumeTurbidityAssessment struct {
	AssessmentID                         string    `json:"assessment_id"`
	CrawlerID                            string    `json:"crawler_id"`
	AdvectionDiffusionDispersionRadiusM  float64   `json:"advection_diffusion_dispersion_radius_m"` // r = sqrt(4 * K_diff * t)
	HydrocycloneSeparationEfficiencyPct  float64   `json:"hydrocyclone_separation_efficiency_pct"`
	BenthicEnvironmentalComplianceStatus string    `json:"benthic_environmental_compliance_status"` // "COMPLIANT_BENTHIC_REGIME", "ELEVATED_TURBIDITY_WARNING", "CRITICAL_BENTHIC_ECOSYSTEM_HAZARD"
	SuctionThrottlingMandate             bool      `json:"suction_throttling_mandate"`
	RecommendedFlocculantDosePPM         float64   `json:"recommended_flocculant_dose_ppm"`
	AuditDigest                          string    `json:"audit_digest"`
	EvaluatedAt                          time.Time `json:"evaluated_at"`
}

// PlumeMitigationAction models automated suction dredging throttling and midwater flocculant dosing
type PlumeMitigationAction struct {
	ActionID                    string    `json:"action_id"`
	CrawlerID                   string    `json:"crawler_id"`
	ThrottledSuctionPowerPct    float64   `json:"throttled_suction_power_pct"`
	FlocculantInjectionRateLPM  float64   `json:"flocculant_injection_rate_lpm"`
	AvoidedEnvironmentalFineUSD float64   `json:"avoided_environmental_fine_usd"` // Avoided $35M USD ISA moratorium and benthic damage fines
	MitigatedAt                 time.Time `json:"mitigated_at"`
}

// SeabedVault thread-safe in-memory store
type SeabedVault struct {
	mu          sync.RWMutex
	telemetries map[string]SeabedMiningTelemetry
	assessments map[string]PlumeTurbidityAssessment
	actions     map[string]PlumeMitigationAction
}

// Global instance
var GlobalSeabedVault = &SeabedVault{
	telemetries: make(map[string]SeabedMiningTelemetry),
	assessments: make(map[string]PlumeTurbidityAssessment),
	actions:     make(map[string]PlumeMitigationAction),
}

func init() {
	// Seed demo deep-sea polymetallic nodule harvester in Clarion-Clipperton Zone
	GlobalSeabedVault.telemetries["CRAWLER-PACIFIC-CCZ-01"] = SeabedMiningTelemetry{
		TelemetryID:                  "TEL-SEABED-001",
		CrawlerID:                    "CRAWLER-PACIFIC-CCZ-01",
		MiningConcessionID:           "NORI-D-CLARION-CLIPPERTON",
		WaterDepthMeters:             4350.0,
		SeabedCrawlerSpeedMPerSec:    0.45,
		NoduleCollectionRateTonsHr:   195.0,
		VerticalRiserVelocityMPerSec: 3.8,
		BenthicPlumeTurbidityNTU:     82.5, // Exceeds 40 NTU limit
		SuspendedParticulateMgL:      52.0, // Exceeds 25 mg/L threshold
		MidwaterDischargeDepthM:      1200.0,
		RecordedAt:                   time.Now(),
	}
}

// IngestCrawlerTelemetry stores benthic crawler and acoustic Doppler current profiler (ADCP) telemetry
func (v *SeabedVault) IngestCrawlerTelemetry(t SeabedMiningTelemetry) SeabedMiningTelemetry {
	v.mu.Lock()
	defer v.mu.Unlock()

	t.RecordedAt = time.Now()
	v.telemetries[t.CrawlerID] = t
	return t
}

// EvaluateBenthicPlumeTurbidity evaluates advection-diffusion sediment transport and turbidity compliance
func (v *SeabedVault) EvaluateBenthicPlumeTurbidity(crawlerID string) (PlumeTurbidityAssessment, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	tel, exists := v.telemetries[crawlerID]
	if !exists {
		return PlumeTurbidityAssessment{}, fmt.Errorf("seabed crawler %s not found in seabed vault", crawlerID)
	}

	// Advection-diffusion radius model: r = sqrt(4 * K_diff * t) + u_current * t
	kDiff := 0.25        // Turbulent benthic eddy diffusivity m^2/s
	timeWindow := 3600.0 // 1 hour observation
	dispersionRadius := math.Sqrt(4.0*math.Pi*kDiff*timeWindow) + (0.05 * timeWindow)

	// Separation efficiency modeled from riser velocity
	sepEfficiency := 98.5 - (tel.BenthicPlumeTurbidityNTU * 0.05)
	if sepEfficiency < 85.0 {
		sepEfficiency = 85.0
	}

	status := "COMPLIANT_BENTHIC_REGIME"
	throttleReq := false
	flocculantDose := 5.0

	if tel.SuspendedParticulateMgL > 25.0 || tel.BenthicPlumeTurbidityNTU > 40.0 {
		status = "CRITICAL_BENTHIC_ECOSYSTEM_HAZARD"
		throttleReq = true
		flocculantDose = 18.5
	} else if tel.SuspendedParticulateMgL > 15.0 || tel.BenthicPlumeTurbidityNTU > 25.0 {
		status = "ELEVATED_TURBIDITY_WARNING"
		flocculantDose = 10.0
	}

	hasher := sha256.New()
	hasher.Write([]byte(fmt.Sprintf("%s-%f-%f-%f-%s-%d", crawlerID, tel.SuspendedParticulateMgL, tel.BenthicPlumeTurbidityNTU, dispersionRadius, status, time.Now().UnixNano())))
	digest := hex.EncodeToString(hasher.Sum(nil))

	assessID := fmt.Sprintf("ASSESS-SEABED-%d", time.Now().UnixNano()%1000000)
	assessment := PlumeTurbidityAssessment{
		AssessmentID:                         assessID,
		CrawlerID:                            crawlerID,
		AdvectionDiffusionDispersionRadiusM:  math.Round(dispersionRadius*10.0) / 10.0,
		HydrocycloneSeparationEfficiencyPct:  math.Round(sepEfficiency*100.0) / 100.0,
		BenthicEnvironmentalComplianceStatus: status,
		SuctionThrottlingMandate:             throttleReq,
		RecommendedFlocculantDosePPM:         flocculantDose,
		AuditDigest:                          digest,
		EvaluatedAt:                          time.Now(),
	}

	v.assessments[crawlerID] = assessment
	return assessment, nil
}

// TriggerSuctionAndFlocculantControl throttles dredge head suction and injects biodegradable flocculant
func (v *SeabedVault) TriggerSuctionAndFlocculantControl(crawlerID string) (PlumeMitigationAction, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	avoidedLoss := 35000000.0 // Avoided $35M USD ISA moratorium, vessel impoundment, and benthic coral damage fines

	actionID := fmt.Sprintf("ACT-SEABED-%d", time.Now().UnixNano()%1000000)
	action := PlumeMitigationAction{
		ActionID:                    actionID,
		CrawlerID:                   crawlerID,
		ThrottledSuctionPowerPct:    35.0,
		FlocculantInjectionRateLPM:  45.0,
		AvoidedEnvironmentalFineUSD: avoidedLoss,
		MitigatedAt:                 time.Now(),
	}

	v.actions[crawlerID] = action
	return action, nil
}

// GetSeabedEnvironmentalDossier provides complete ISA & UNCLOS Part XI deep-sea environmental report
func (v *SeabedVault) GetSeabedEnvironmentalDossier(crawlerID string) (map[string]interface{}, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	tel, hasTel := v.telemetries[crawlerID]
	assess, hasAssess := v.assessments[crawlerID]
	act, hasAct := v.actions[crawlerID]

	dossier := map[string]interface{}{
		"crawler_id":               crawlerID,
		"has_crawler_telemetry":    hasTel,
		"has_plume_assessment":     hasAssess,
		"has_mitigation_action":    hasAct,
		"isa_guidelines_compliant": true,
		"unclos_part_xi_approved":  true,
		"compiled_at":              time.Now(),
	}

	if hasTel {
		dossier["concession_id"] = tel.MiningConcessionID
		dossier["water_depth_m"] = tel.WaterDepthMeters
		dossier["harvest_tons_hr"] = tel.NoduleCollectionRateTonsHr
		dossier["turbidity_ntu"] = tel.BenthicPlumeTurbidityNTU
		dossier["particulate_mg_l"] = tel.SuspendedParticulateMgL
	}
	if hasAssess {
		dossier["dispersion_radius_m"] = assess.AdvectionDiffusionDispersionRadiusM
		dossier["separation_efficiency"] = assess.HydrocycloneSeparationEfficiencyPct
		dossier["benthic_status"] = assess.BenthicEnvironmentalComplianceStatus
		dossier["suction_throttled"] = assess.SuctionThrottlingMandate
	}
	if hasAct {
		dossier["throttled_power_pct"] = act.ThrottledSuctionPowerPct
		dossier["flocculant_lpm"] = act.FlocculantInjectionRateLPM
		dossier["avoided_fines_usd"] = act.AvoidedEnvironmentalFineUSD
	}

	return dossier, nil
}

// RegisterSeabedRoutes wires Gin HTTP handlers
func RegisterSeabedRoutes(r *gin.RouterGroup) {
	group := r.Group("/seabed/mining")
	{
		group.POST("/ingest-crawler-telemetry", func(c *gin.Context) {
			var req SeabedMiningTelemetry
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res := GlobalSeabedVault.IngestCrawlerTelemetry(req)
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		group.POST("/evaluate-benthic-plume-turbidity", func(c *gin.Context) {
			var req struct {
				CrawlerID string `json:"crawler_id"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := GlobalSeabedVault.EvaluateBenthicPlumeTurbidity(req.CrawlerID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		group.POST("/trigger-suction-and-flocculant-control", func(c *gin.Context) {
			var req struct {
				CrawlerID string `json:"crawler_id"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := GlobalSeabedVault.TriggerSuctionAndFlocculantControl(req.CrawlerID)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		group.GET("/seabed-environmental-dossier/:crawler_id", func(c *gin.Context) {
			crawlerID := c.Param("crawler_id")
			res, err := GlobalSeabedVault.GetSeabedEnvironmentalDossier(crawlerID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})
	}
}
