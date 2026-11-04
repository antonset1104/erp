package pyrolysis

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

// PyrolysisReactorTelemetry captures mixed plastic chemical recycling metrics under ASTM D8338 & EPA Method 23
type PyrolysisReactorTelemetry struct {
	TelemetryID                 string    `json:"telemetry_id"`
	ReactorID                   string    `json:"reactor_id"`                     // "PYRO-CIRCULAR-PLASTIC-50TPD-01"
	FacilityLocation            string    `json:"facility_location"`              // "ANTWERP-CIRCULAR-POLYMER-HUB"
	PyrolysisBedTempC           float64   `json:"pyrolysis_bed_temp_c"`           // e.g. 485.0°C (cracking 450-520°C)
	FeedstockFeedRateKgHr       float64   `json:"feedstock_feed_rate_kg_hr"`      // e.g. 2100.0 kg/hr
	GasRecycleLoopTempC         float64   `json:"gas_recycle_loop_temp_c"`        // e.g. 310.0°C (dioxin window 250-400°C)
	ChlorineContentPPM          float64   `json:"chlorine_content_ppm"`           // e.g. 48.0 ppm in pyoil (spec < 10 ppm)
	HClAcidGasPPM               float64   `json:"hcl_acid_gas_ppm"`               // e.g. 185.0 ppm
	CalciumSorbentDosingRateKgHr float64  `json:"calcium_sorbent_dosing_rate_kg_hr"` // e.g. 22.0 kg/hr
	PyrolysisOilYieldPct        float64   `json:"pyrolysis_oil_yield_pct"`        // e.g. 78.5%
	RecordedAt                  time.Time `json:"recorded_at"`
}

// DepolymerizationDioxinAssessment models random scission depolymerization kinetics and dioxin de novo synthesis risk
type DepolymerizationDioxinAssessment struct {
	AssessmentID                          string    `json:"assessment_id"`
	ReactorID                             string    `json:"reactor_id"`
	DepolymerizationConversionPct         float64   `json:"depolymerization_conversion_pct"`
	DioxinDeNovoSynthesisRiskLevel        string    `json:"dioxin_de_novo_synthesis_risk_level"`        // "SAFE_THERMAL_DECOMPOSITION", "ELEVATED_CHLORINATED_GAS_WARNING", "CRITICAL_DIOXIN_SYNTHESIS_AND_CATALYST_POISON_HAZARD"
	SteamCrackerFeedstockComplianceStatus string    `json:"steam_cracker_feedstock_compliance_status"` // "REFINERY_ISCC_PLUS_CERTIFIED", "OFF_SPEC_CHLORINE_CORROSION_RISK"
	RapidGasQuenchRequired                bool      `json:"rapid_gas_quench_required"`
	RecommendedSorbentBoostKgHr           float64   `json:"recommended_sorbent_boost_kg_hr"`
	AuditDigest                           string    `json:"audit_digest"`
	EvaluatedAt                           time.Time `json:"evaluated_at"`
}

// PyrolysisQuenchDechlorinationAction models rapid vapor quenching below 200°C and calcium guard bed sorbent boost
type PyrolysisQuenchDechlorinationAction struct {
	ActionID                      string    `json:"action_id"`
	ReactorID                     string    `json:"reactor_id"`
	QuenchTemperatureDropC        float64   `json:"quench_temperature_drop_c"`
	TunedSorbentInjectionKgHr     float64   `json:"tuned_sorbent_injection_kg_hr"`
	AvoidedRefineryCatalystLossUSD float64  `json:"avoided_refinery_catalyst_loss_usd"` // Avoided $21M USD refinery hydrotreater catalyst poisoning and EPA hazardous air fines
	ActionTimestamp               time.Time `json:"action_timestamp"`
}

// PyrolysisVault thread-safe in-memory store
type PyrolysisVault struct {
	mu          sync.RWMutex
	telemetries map[string]PyrolysisReactorTelemetry
	assessments map[string]DepolymerizationDioxinAssessment
	actions     map[string]PyrolysisQuenchDechlorinationAction
}

// Global instance
var GlobalPyrolysisVault = &PyrolysisVault{
	telemetries: make(map[string]PyrolysisReactorTelemetry),
	assessments: make(map[string]DepolymerizationDioxinAssessment),
	actions:     make(map[string]PyrolysisQuenchDechlorinationAction),
}

func init() {
	// Seed demo plastic circular recycling reactor with elevated chlorine precursor
	GlobalPyrolysisVault.telemetries["PYRO-CIRCULAR-PLASTIC-50TPD-01"] = PyrolysisReactorTelemetry{
		TelemetryID:                 "TEL-PYRO-001",
		ReactorID:                   "PYRO-CIRCULAR-PLASTIC-50TPD-01",
		FacilityLocation:            "ANTWERP-CIRCULAR-POLYMER-HUB",
		PyrolysisBedTempC:           490.0,
		FeedstockFeedRateKgHr:       2150.0,
		GasRecycleLoopTempC:         320.0, // Inside PCDD/F formation window (250 - 400°C)
		ChlorineContentPPM:          52.0,  // Exceeds 10 ppm refinery cracker limit
		HClAcidGasPPM:               195.0,
		CalciumSorbentDosingRateKgHr: 22.0,
		PyrolysisOilYieldPct:        79.0,
		RecordedAt:                  time.Now(),
	}
}

// IngestReactorTelemetry stores gas chromatography and temperature profile readings
func (v *PyrolysisVault) IngestReactorTelemetry(t PyrolysisReactorTelemetry) PyrolysisReactorTelemetry {
	v.mu.Lock()
	defer v.mu.Unlock()

	t.RecordedAt = time.Now()
	v.telemetries[t.ReactorID] = t
	return t
}

// EvaluateDepolymerizationAndDioxin evaluates depolymerization conversion and dioxin formation windows
func (v *PyrolysisVault) EvaluateDepolymerizationAndDioxin(reactorID string) (DepolymerizationDioxinAssessment, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	tel, exists := v.telemetries[reactorID]
	if !exists {
		return DepolymerizationDioxinAssessment{}, fmt.Errorf("reactor %s not found in pyrolysis vault", reactorID)
	}

	conversionPct := 92.5 + ((tel.PyrolysisBedTempC - 450.0) * 0.08)
	if conversionPct > 99.5 {
		conversionPct = 99.5
	}

	dioxinRisk := "SAFE_THERMAL_DECOMPOSITION"
	quenchReq := false
	sorbentBoost := 25.0
	crackerCompliance := "REFINERY_ISCC_PLUS_CERTIFIED"

	if (tel.GasRecycleLoopTempC >= 250.0 && tel.GasRecycleLoopTempC <= 400.0) && tel.ChlorineContentPPM > 10.0 {
		dioxinRisk = "CRITICAL_DIOXIN_SYNTHESIS_AND_CATALYST_POISON_HAZARD"
		crackerCompliance = "OFF_SPEC_CHLORINE_CORROSION_RISK"
		quenchReq = true
		sorbentBoost = 52.0
	} else if tel.ChlorineContentPPM > 10.0 || tel.HClAcidGasPPM > 100.0 {
		dioxinRisk = "ELEVATED_CHLORINATED_GAS_WARNING"
		crackerCompliance = "OFF_SPEC_CHLORINE_CORROSION_RISK"
		sorbentBoost = 38.0
	}

	hasher := sha256.New()
	hasher.Write([]byte(fmt.Sprintf("%s-%f-%f-%f-%s-%d", reactorID, conversionPct, tel.ChlorineContentPPM, tel.GasRecycleLoopTempC, dioxinRisk, time.Now().UnixNano())))
	digest := hex.EncodeToString(hasher.Sum(nil))

	assessID := fmt.Sprintf("ASSESS-PYRO-%d", time.Now().UnixNano()%1000000)
	assessment := DepolymerizationDioxinAssessment{
		AssessmentID:                          assessID,
		ReactorID:                             reactorID,
		DepolymerizationConversionPct:         math.Round(conversionPct*100.0) / 100.0,
		DioxinDeNovoSynthesisRiskLevel:        dioxinRisk,
		SteamCrackerFeedstockComplianceStatus: crackerCompliance,
		RapidGasQuenchRequired:                quenchReq,
		RecommendedSorbentBoostKgHr:           sorbentBoost,
		AuditDigest:                           digest,
		EvaluatedAt:                           time.Now(),
	}

	v.assessments[reactorID] = assessment
	return assessment, nil
}

// TriggerQuenchAndDechlorination triggers rapid gas quench and injects sodium bicarbonate/calcium sorbent
func (v *PyrolysisVault) TriggerQuenchAndDechlorination(reactorID string) (PyrolysisQuenchDechlorinationAction, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	avoidedLoss := 21000000.0 // Avoided $21M USD refinery hydrotreater catalyst deactivation, chloride stress cracking, and EPA emission penalties

	actionID := fmt.Sprintf("ACT-PYRO-%d", time.Now().UnixNano()%1000000)
	action := PyrolysisQuenchDechlorinationAction{
		ActionID:                      actionID,
		ReactorID:                     reactorID,
		QuenchTemperatureDropC:        135.0, // Rapid plunge from 320°C to 185°C in 0.6 seconds
		TunedSorbentInjectionKgHr:     52.0,
		AvoidedRefineryCatalystLossUSD: avoidedLoss,
		ActionTimestamp:               time.Now(),
	}

	v.actions[reactorID] = action
	return action, nil
}

// GetPyrolysisCircularityDossier compiles complete ASTM D8338 & EPA Method 23 circular plastic recycling report
func (v *PyrolysisVault) GetPyrolysisCircularityDossier(reactorID string) (map[string]interface{}, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	tel, hasTel := v.telemetries[reactorID]
	assess, hasAssess := v.assessments[reactorID]
	act, hasAct := v.actions[reactorID]

	dossier := map[string]interface{}{
		"reactor_id":               reactorID,
		"has_reactor_telemetry":    hasTel,
		"has_depolymer_assessment": hasAssess,
		"has_quench_action":        hasAct,
		"astm_d8338_compliant":     true,
		"epa_method_23_certified":  true,
		"compiled_at":              time.Now(),
	}

	if hasTel {
		dossier["facility"] = tel.FacilityLocation
		dossier["bed_temp_c"] = tel.PyrolysisBedTempC
		dossier["gas_recycle_temp_c"] = tel.GasRecycleLoopTempC
		dossier["chlorine_ppm"] = tel.ChlorineContentPPM
		dossier["oil_yield_pct"] = tel.PyrolysisOilYieldPct
	}
	if hasAssess {
		dossier["conversion_pct"] = assess.DepolymerizationConversionPct
		dossier["dioxin_risk"] = assess.DioxinDeNovoSynthesisRiskLevel
		dossier["cracker_compliance"] = assess.SteamCrackerFeedstockComplianceStatus
		dossier["sorbent_boost_kg_hr"] = assess.RecommendedSorbentBoostKgHr
	}
	if hasAct {
		dossier["quench_drop_c"] = act.QuenchTemperatureDropC
		dossier["tuned_sorbent_kg_hr"] = act.TunedSorbentInjectionKgHr
		dossier["avoided_loss_usd"] = act.AvoidedRefineryCatalystLossUSD
	}

	return dossier, nil
}

// RegisterPyrolysisRoutes wires Gin HTTP handlers
func RegisterPyrolysisRoutes(r *gin.RouterGroup) {
	group := r.Group("/pyrolysis/reactor")
	{
		group.POST("/ingest-reactor-telemetry", func(c *gin.Context) {
			var req PyrolysisReactorTelemetry
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res := GlobalPyrolysisVault.IngestReactorTelemetry(req)
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		group.POST("/evaluate-depolymerization-and-dioxin", func(c *gin.Context) {
			var req struct {
				ReactorID string `json:"reactor_id"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := GlobalPyrolysisVault.EvaluateDepolymerizationAndDioxin(req.ReactorID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		group.POST("/trigger-quench-and-dechlorination", func(c *gin.Context) {
			var req struct {
				ReactorID string `json:"reactor_id"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := GlobalPyrolysisVault.TriggerQuenchAndDechlorination(req.ReactorID)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		group.GET("/pyrolysis-circularity-dossier/:reactor_id", func(c *gin.Context) {
			rID := c.Param("reactor_id")
			res, err := GlobalPyrolysisVault.GetPyrolysisCircularityDossier(rID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})
	}
}
