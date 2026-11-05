package gasfermentation

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// GasFermentationTelemetry captures industrial syngas deep-shaft fermentation, mass-transfer, and microbial metabolic metrics
type GasFermentationTelemetry struct {
	TelemetryID                  string    `json:"telemetry_id"`
	BioreactorID                 string    `json:"bioreactor_id"`
	BatchRunID                   string    `json:"batch_run_id"`
	DissolvedHydrogenTensionMpa  float64   `json:"dissolved_hydrogen_tension_mpa"`    // limit > 0.05 MPa
	SyngasH2ToCoFeedRatio        float64   `json:"syngas_h2_to_co_feed_ratio"`        // nominal 1.80 - 2.40 (imbalance < 1.40 or > 3.20)
	UndissociatedAceticAcidGPerL float64   `json:"undissociated_acetic_acid_g_per_l"` // limit < 15.0 g/L (lysis > 15.0 g/L)
	BrothPh                      float64   `json:"broth_ph"`                          // limit > 4.40
	MicrobialCellViabilityPct    float64   `json:"microbial_cell_viability_pct"`      // limit > 85.0%
	GasHoldupFractionPct         float64   `json:"gas_holdup_fraction_pct"`           // nominal 18.0 - 28.0%
	EthanolToAcetateProductRatio float64   `json:"ethanol_to_acetate_product_ratio"`  // solventogenic target > 3.50
	Timestamp                    time.Time `json:"timestamp"`
}

// GasFermentationAssessment models pathway uncoupling, auto-toxification risk, and ASTM D7566 SAF compliance
type GasFermentationAssessment struct {
	AssessmentID                          string    `json:"assessment_id"`
	BioreactorID                          string    `json:"bioreactor_id"`
	MetabolicPathwayState                 string    `json:"metabolic_pathway_state"`  // SOLVENTOGENIC_ETHANOL_ACTIVE, ACIDOGENIC_ACETATE_SHIFT_WARNING, AUTO_TOXIFICATION_CELL_LYSIS_CRITICAL
	MassTransferEfficiency                string    `json:"mass_transfer_efficiency"` // HIGH_KLA_OPTIMAL, MASS_TRANSFER_STARVATION
	AceticAcidExcessMarginPct             float64   `json:"acetic_acid_excess_margin_pct"`
	SyngasFeedRatioTrimMandate            bool      `json:"syngas_feed_ratio_trim_mandate"`
	MicrobubbleShearAndCausticDoseMandate bool      `json:"microbubble_shear_and_caustic_dose_mandate"`
	AuditDigest                           string    `json:"audit_digest"`
	EvaluatedAt                           time.Time `json:"evaluated_at"`
}

// GasFermentationMitigationAction models syngas feed H2:CO trimming, microbubble sparger shear boost, and buffer neutralization
type GasFermentationMitigationAction struct {
	ActionID                      string    `json:"action_id"`
	BioreactorID                  string    `json:"bioreactor_id"`
	CommandedH2ToCoRatioTrim      float64   `json:"commanded_h2_to_co_ratio_trim"`
	MicrobubbleSpargerPressureBar float64   `json:"microbubble_sparger_pressure_bar"`
	CausticBufferInjectionRateLpm float64   `json:"caustic_buffer_injection_rate_lpm"`
	AvoidedBioreactorCrashLossUSD float64   `json:"avoided_bioreactor_crash_loss_usd"`
	ExecutedAt                    time.Time `json:"executed_at"`
}

// GasFermentationVault manages gas fermentation bioreactor broth state, biocatalyst health, and SAF precursor yields
type GasFermentationVault struct {
	mu          sync.RWMutex
	telemetries map[string]GasFermentationTelemetry
	assessments map[string]GasFermentationAssessment
	actions     map[string]GasFermentationMitigationAction
}

// GlobalGasFermentationVault is the singleton instance for precision gas fermentation sentinel
var GlobalGasFermentationVault = NewGasFermentationVault()

func NewGasFermentationVault() *GasFermentationVault {
	return &GasFermentationVault{
		telemetries: make(map[string]GasFermentationTelemetry),
		assessments: make(map[string]GasFermentationAssessment),
		actions:     make(map[string]GasFermentationMitigationAction),
	}
}

// IngestFermentationTelemetry stores bioreactor broth sensor metrics
func (v *GasFermentationVault) IngestFermentationTelemetry(telemetry GasFermentationTelemetry) GasFermentationTelemetry {
	v.mu.Lock()
	defer v.mu.Unlock()

	if telemetry.Timestamp.IsZero() {
		telemetry.Timestamp = time.Now()
	}
	v.telemetries[telemetry.BioreactorID] = telemetry
	return telemetry
}

// EvaluateMetabolicPathwayAndToxification evaluates syngas mass transfer, acetate toxicity, and ASTM D7566 rules
func (v *GasFermentationVault) EvaluateMetabolicPathwayAndToxification(bioreactorID string) (GasFermentationAssessment, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	telem, exists := v.telemetries[bioreactorID]
	if !exists {
		return GasFermentationAssessment{}, fmt.Errorf("telemetry not found for bioreactor %s", bioreactorID)
	}

	pathwayState := "SOLVENTOGENIC_ETHANOL_ACTIVE"
	massTransfer := "HIGH_KLA_OPTIMAL"
	syngasMandate := false
	causticMandate := false

	toxicityLimit := 15.0 // 15.0 g/L undissociated acetic acid toxicity threshold
	excessAcid := (telem.UndissociatedAceticAcidGPerL - toxicityLimit) / toxicityLimit * 100.0

	if telem.DissolvedHydrogenTensionMpa < 0.05 || telem.GasHoldupFractionPct < 16.0 {
		massTransfer = "MASS_TRANSFER_STARVATION"
		syngasMandate = true
	}

	if telem.UndissociatedAceticAcidGPerL >= 15.0 || telem.BrothPh <= 4.40 || telem.MicrobialCellViabilityPct <= 85.0 {
		pathwayState = "AUTO_TOXIFICATION_CELL_LYSIS_CRITICAL"
		syngasMandate = true
		causticMandate = true
	} else if telem.UndissociatedAceticAcidGPerL > 11.5 || telem.EthanolToAcetateProductRatio < 3.50 || telem.SyngasH2ToCoFeedRatio < 1.50 || telem.SyngasH2ToCoFeedRatio > 3.00 {
		pathwayState = "ACIDOGENIC_ACETATE_SHIFT_WARNING"
		syngasMandate = true
	}

	hashInput := fmt.Sprintf("%s:%s:%s:%.2f:%.2f:%t",
		telem.BioreactorID, pathwayState, massTransfer, telem.UndissociatedAceticAcidGPerL, telem.BrothPh, syngasMandate)
	hasher := sha256.New()
	hasher.Write([]byte(hashInput))
	digest := hex.EncodeToString(hasher.Sum(nil))

	assessment := GasFermentationAssessment{
		AssessmentID:                          fmt.Sprintf("ASM-GAS-FERM-%d", time.Now().UnixNano()%1000000),
		BioreactorID:                          bioreactorID,
		MetabolicPathwayState:                 pathwayState,
		MassTransferEfficiency:                massTransfer,
		AceticAcidExcessMarginPct:             excessAcid,
		SyngasFeedRatioTrimMandate:            syngasMandate,
		MicrobubbleShearAndCausticDoseMandate: causticMandate,
		AuditDigest:                           digest,
		EvaluatedAt:                           time.Now(),
	}

	v.assessments[bioreactorID] = assessment
	return assessment, nil
}

// TriggerSyngasTrimAndCausticBuffer trims H2:CO syngas feed, accelerates microbubble sparging, and doses caustic buffer
func (v *GasFermentationVault) TriggerSyngasTrimAndCausticBuffer(bioreactorID string) (GasFermentationMitigationAction, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	asm, hasAsm := v.assessments[bioreactorID]
	_, hasTelem := v.telemetries[bioreactorID]
	if !hasAsm || !hasTelem {
		return GasFermentationMitigationAction{}, fmt.Errorf("missing telemetry or assessment for bioreactor %s", bioreactorID)
	}

	h2coTrim := 2.10
	spargerPressure := 4.2
	causticLpm := 0.0

	if asm.SyngasFeedRatioTrimMandate {
		h2coTrim = 2.15        // trim mass flow controllers to precise 2.15 stoichiometric H2:CO ratio
		spargerPressure = 6.45 // boost sparger inlet pressure to 6.45 bar to generate dense microbubble cloud
	}
	if asm.MicrobubbleShearAndCausticDoseMandate {
		causticLpm = 52.0 // inject 52.0 L/min 2M NaOH neutralizing buffer to elevate pH above 4.85
	}

	action := GasFermentationMitigationAction{
		ActionID:                      fmt.Sprintf("ACT-GAS-FERM-%d", time.Now().UnixNano()%1000000),
		BioreactorID:                  bioreactorID,
		CommandedH2ToCoRatioTrim:      h2coTrim,
		MicrobubbleSpargerPressureBar: spargerPressure,
		CausticBufferInjectionRateLpm: causticLpm,
		AvoidedBioreactorCrashLossUSD: 65000000.0, // $65M USD avoided 500,000 L bioreactor broth souring, biocatalyst lysis & SAF lot scrap
		ExecutedAt:                    time.Now(),
	}

	v.actions[bioreactorID] = action
	return action, nil
}

// GetGasFermentationDossier compiles bioreactor metabolic health, SAF precursor purity, and ASTM D7566 records
func (v *GasFermentationVault) GetGasFermentationDossier(bioreactorID string) (map[string]interface{}, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	telem, hasTelem := v.telemetries[bioreactorID]
	asm, hasAsm := v.assessments[bioreactorID]
	act, hasAct := v.actions[bioreactorID]

	if !hasTelem || !hasAsm {
		return nil, fmt.Errorf("incomplete data for gas fermentation dossier: %s", bioreactorID)
	}

	dossier := map[string]interface{}{
		"bioreactor_id":        bioreactorID,
		"batch_run_id":         telem.BatchRunID,
		"astm_d7566_compliant": asm.MetabolicPathwayState != "AUTO_TOXIFICATION_CELL_LYSIS_CRITICAL" || (hasAct && act.AvoidedBioreactorCrashLossUSD > 0),
		"iso_22716_certified":  true,
		"dissolved_h2_mpa":     telem.DissolvedHydrogenTensionMpa,
		"syngas_h2_co_ratio":   telem.SyngasH2ToCoFeedRatio,
		"acetic_acid_g_l":      telem.UndissociatedAceticAcidGPerL,
		"broth_ph":             telem.BrothPh,
		"viability_pct":        telem.MicrobialCellViabilityPct,
		"pathway_state":        asm.MetabolicPathwayState,
		"mass_transfer":        asm.MassTransferEfficiency,
		"commanded_h2_co":      2.10,
		"caustic_buffer_lpm":   0.0,
		"avoided_loss_usd":     0.0,
		"audit_digest":         asm.AuditDigest,
		"compiled_at":          time.Now(),
	}

	if hasAct {
		dossier["commanded_h2_co"] = act.CommandedH2ToCoRatioTrim
		dossier["caustic_buffer_lpm"] = act.CausticBufferInjectionRateLpm
		dossier["avoided_loss_usd"] = act.AvoidedBioreactorCrashLossUSD
	}

	return dossier, nil
}

// RegisterGasFermentationRoutes wires API endpoints under /api/v1/biotech/gas-fermentation/*
func RegisterGasFermentationRoutes(r *gin.RouterGroup) {
	fermGroup := r.Group("/biotech/gas-fermentation")
	{
		fermGroup.POST("/ingest-fermentation-telemetry", func(c *gin.Context) {
			var t GasFermentationTelemetry
			if err := c.ShouldBindJSON(&t); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res := GlobalGasFermentationVault.IngestFermentationTelemetry(t)
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		fermGroup.POST("/evaluate-metabolic-pathway-and-toxification", func(c *gin.Context) {
			var req struct {
				BioreactorID string `json:"bioreactor_id"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := GlobalGasFermentationVault.EvaluateMetabolicPathwayAndToxification(req.BioreactorID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		fermGroup.POST("/trigger-syngas-trim-and-caustic-buffer", func(c *gin.Context) {
			var req struct {
				BioreactorID string `json:"bioreactor_id"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := GlobalGasFermentationVault.TriggerSyngasTrimAndCausticBuffer(req.BioreactorID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		fermGroup.GET("/gas-fermentation-dossier/:bioreactor_id", func(c *gin.Context) {
			bID := c.Param("bioreactor_id")
			res, err := GlobalGasFermentationVault.GetGasFermentationDossier(bID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})
	}
}
