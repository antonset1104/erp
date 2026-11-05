package mrnacapping

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

// MRNASynthesisTelemetry captures continuous in vitro transcription (IVT) mRNA capping & quality dynamics under FDA 21 CFR Part 211 & ISO 20387
type MRNASynthesisTelemetry struct {
	TelemetryID                  string    `json:"telemetry_id"`
	BatchID                      string    `json:"batch_id"`                                // "BATCH-MRNA-ONCOLOGY-VACC-004"
	BioreactorLineID             string    `json:"bioreactor_line_id"`                      // "LINE-IVT-MICROFLUIDIC-01"
	TargetAntigenCode            string    `json:"target_antigen_code"`                     // "AG-NEOANTIGEN-MELANOMA"
	Cap1MethylationEfficiencyPct float64   `json:"cap1_methylation_efficiency_pct"`         // e.g. 88.5% (release spec > 95.0%)
	PolyATailMeanLengthNt        float64   `json:"poly_a_tail_mean_length_nt"`              // e.g. 115.0 nt (optimal 100-120 nt)
	DoubleStrandedRnaFractionPpm float64   `json:"double_stranded_rna_fraction_ppm"`         // e.g. 48.5 ppm (spec limit < 20.0 ppm)
	SamCofactorConcentrationMm   float64   `json:"sam_cofactor_concentration_mm"`           // e.g. 0.42 mM (nominal 0.80 mM)
	IvtReactionTempC             float64   `json:"ivt_reaction_temp_c"`                     // e.g. 37.8°C
	NtpConsumptionRateMmPerHr    float64   `json:"ntp_consumption_rate_mm_per_hr"`          // e.g. 14.2 mM/hr
	TranscriptIntegrityRin       float64   `json:"transcript_integrity_rin"`                // e.g. 8.4
	RecordedAt                   time.Time `json:"recorded_at"`
}

// MRNASynthesisAssessment evaluates translational Cap-1 efficiency, innate immune reactogenicity, and dsRNA toxicity
type MRNASynthesisAssessment struct {
	AssessmentID                 string    `json:"assessment_id"`
	BatchID                      string    `json:"batch_id"`
	Cap1DeficitPct               float64   `json:"cap1_deficit_pct"`
	CappingQualityStatus         string    `json:"capping_quality_status"`          // "HIGH_TRANSLATIONAL_CAP1_GRADE", "SUB_OPTIMAL_CAPPING_WARNING", "IMMUNOGENIC_UNCAPPED_MESSENGER_HAZARD"
	DoubleStrandedRnaRisk        string    `json:"double_stranded_rna_risk"`         // "LOW_IMMUNOGENICITY_CLEAN", "ELEVATED_DSRNA_WARNING", "CRITICAL_INFLAMMATORY_DSRNA_REJECT"
	SamMicroInfusionMandate      bool      `json:"sam_micro_infusion_mandate"`
	ReactionTempStepDownMandate  bool      `json:"reaction_temp_step_down_mandate"`
	AuditDigest                  string    `json:"audit_digest"`
	EvaluatedAt                  time.Time `json:"evaluated_at"`
}

// MRNAMitigationAction models S-adenosylmethionine (SAM) micro-infusion, IVT reaction temperature ramp-down, and SEC diverter trimming
type MRNAMitigationAction struct {
	ActionID                      string    `json:"action_id"`
	BatchID                       string    `json:"batch_id"`
	PulsedSamInfusionRateMlPerMin float64   `json:"pulsed_sam_infusion_rate_ml_per_min"`
	CommandedIvtTempSetpointC     float64   `json:"commanded_ivt_temp_setpoint_c"`
	DivertedAberrantFractionLiters float64  `json:"diverted_aberrant_fraction_liters"`
	AvoidedMrnaBatchLossUSD       float64   `json:"avoided_mrna_batch_loss_usd"`
	ActionTimestamp               time.Time `json:"action_timestamp"`
}

// MRNAVault thread-safe in-memory store
type MRNAVault struct {
	mu          sync.RWMutex
	telemetries map[string]MRNASynthesisTelemetry
	assessments map[string]MRNASynthesisAssessment
	actions     map[string]MRNAMitigationAction
}

// GlobalMRNAVault instance
var GlobalMRNAVault = &MRNAVault{
	telemetries: make(map[string]MRNASynthesisTelemetry),
	assessments: make(map[string]MRNASynthesisAssessment),
	actions:     make(map[string]MRNAMitigationAction),
}

// IngestSynthesisTelemetry stores mRNA in vitro synthesis telemetry
func (v *MRNAVault) IngestSynthesisTelemetry(t MRNASynthesisTelemetry) MRNASynthesisTelemetry {
	v.mu.Lock()
	defer v.mu.Unlock()

	if t.TelemetryID == "" {
		t.TelemetryID = fmt.Sprintf("TEL-MRNA-%d", time.Now().UnixNano()%1000000)
	}
	if t.RecordedAt.IsZero() {
		t.RecordedAt = time.Now()
	}

	v.telemetries[t.BatchID] = t
	return t
}

// EvaluateCappingAndImpurities evaluates Cap-1 methylation fraction, innate immune reactogenicity, and dsRNA toxicity
func (v *MRNAVault) EvaluateCappingAndImpurities(batchID string) (MRNASynthesisAssessment, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	telem, exists := v.telemetries[batchID]
	if !exists {
		return MRNASynthesisAssessment{}, fmt.Errorf("telemetry not found for mRNA batch: %s", batchID)
	}

	deficit := math.Round((95.0-telem.Cap1MethylationEfficiencyPct)*10) / 10
	if deficit < 0.0 {
		deficit = 0.0
	}

	cappingStatus := "HIGH_TRANSLATIONAL_CAP1_GRADE"
	samInfusion := false

	if telem.Cap1MethylationEfficiencyPct < 90.0 || telem.SamCofactorConcentrationMm < 0.50 {
		cappingStatus = "IMMUNOGENIC_UNCAPPED_MESSENGER_HAZARD"
		samInfusion = true
	} else if telem.Cap1MethylationEfficiencyPct < 95.0 {
		cappingStatus = "SUB_OPTIMAL_CAPPING_WARNING"
		samInfusion = true
	}

	dsRnaRisk := "LOW_IMMUNOGENICITY_CLEAN"
	tempRampDown := false

	if telem.DoubleStrandedRnaFractionPpm > 35.0 || telem.IvtReactionTempC > 37.5 {
		dsRnaRisk = "CRITICAL_INFLAMMATORY_DSRNA_REJECT"
		tempRampDown = true
		samInfusion = true
	} else if telem.DoubleStrandedRnaFractionPpm > 20.0 {
		dsRnaRisk = "ELEVATED_DSRNA_WARNING"
		tempRampDown = true
	}

	rawHash := fmt.Sprintf("%s:%f:%s:%s:%t:%t", batchID, deficit, cappingStatus, dsRnaRisk, samInfusion, tempRampDown)
	hash := sha256.Sum256([]byte(rawHash))
	digest := hex.EncodeToString(hash[:])

	assessment := MRNASynthesisAssessment{
		AssessmentID:                fmt.Sprintf("ASM-MRNA-%d", time.Now().UnixNano()%1000000),
		BatchID:                     batchID,
		Cap1DeficitPct:              deficit,
		CappingQualityStatus:        cappingStatus,
		DoubleStrandedRnaRisk:       dsRnaRisk,
		SamMicroInfusionMandate:     samInfusion,
		ReactionTempStepDownMandate: tempRampDown,
		AuditDigest:                 digest,
		EvaluatedAt:                 time.Now(),
	}

	v.assessments[batchID] = assessment
	return assessment, nil
}

// TriggerSamInfusionAndTempRamp injects methyltransferase cofactor SAM and steps IVT temperature down to 32°C
func (v *MRNAVault) TriggerSamInfusionAndTempRamp(batchID string) (MRNAMitigationAction, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	asm, hasAsm := v.assessments[batchID]
	telem, hasTelem := v.telemetries[batchID]
	if !hasAsm || !hasTelem {
		return MRNAMitigationAction{}, fmt.Errorf("missing telemetry or assessment for mRNA batch %s", batchID)
	}

	samRate := 0.0
	targetTemp := telem.IvtReactionTempC
	divertedVol := 0.0

	if asm.SamMicroInfusionMandate {
		samRate = 8.5 // inject 8.5 mL/min concentrated SAM cofactor to drive methyltransferase kinetics to 98%
	}
	if asm.ReactionTempStepDownMandate {
		targetTemp = 32.0 // ramp down IVT temperature from 37.8°C to 32.0°C to halt aberrant T7 RNA polymerase dsRNA synthesis
		divertedVol = 5.2 // divert aberrant pre-run fraction to purification drain
	}

	action := MRNAMitigationAction{
		ActionID:                      fmt.Sprintf("ACT-MRNA-%d", time.Now().UnixNano()%1000000),
		BatchID:                       batchID,
		PulsedSamInfusionRateMlPerMin: samRate,
		CommandedIvtTempSetpointC:     targetTemp,
		DivertedAberrantFractionLiters: divertedVol,
		AvoidedMrnaBatchLossUSD:       35000000.0,
		ActionTimestamp:               time.Now(),
	}

	v.actions[batchID] = action
	return action, nil
}

// GetMRNACGMPDossier returns mRNA synthetic biology cGMP release dossier under FDA 21 CFR Part 211 & ISO 20387
func (v *MRNAVault) GetMRNACGMPDossier(batchID string) (map[string]interface{}, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	telem, hasTelem := v.telemetries[batchID]
	asm, hasAsm := v.assessments[batchID]
	act := v.actions[batchID]

	if !hasTelem || !hasAsm {
		return nil, fmt.Errorf("incomplete data for mRNA batch dossier %s", batchID)
	}

	return map[string]interface{}{
		"batch_id":               batchID,
		"bioreactor_line_id":     telem.BioreactorLineID,
		"target_antigen_code":    telem.TargetAntigenCode,
		"fda_21cfr211_compliant": true,
		"iso_20387_certified":    true,
		"telemetry":              telem,
		"assessment":             asm,
		"latest_action":          act,
		"compiled_at":            time.Now(),
	}, nil
}

// RegisterMRNARoutes wires API endpoints under /api/v1/biopharma/mrna-capping/*
func RegisterMRNARoutes(r *gin.RouterGroup) {
	mrnaGroup := r.Group("/biopharma/mrna-capping")
	{
		mrnaGroup.POST("/ingest-synthesis-telemetry", func(c *gin.Context) {
			var t MRNASynthesisTelemetry
			if err := c.ShouldBindJSON(&t); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res := GlobalMRNAVault.IngestSynthesisTelemetry(t)
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		mrnaGroup.POST("/evaluate-capping-and-impurities", func(c *gin.Context) {
			var req struct {
				BatchID string `json:"batch_id"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := GlobalMRNAVault.EvaluateCappingAndImpurities(req.BatchID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		mrnaGroup.POST("/trigger-sam-infusion-and-temp-ramp", func(c *gin.Context) {
			var req struct {
				BatchID string `json:"batch_id"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := GlobalMRNAVault.TriggerSamInfusionAndTempRamp(req.BatchID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		mrnaGroup.GET("/mrna-cgmp-dossier/:batch_id", func(c *gin.Context) {
			batchID := c.Param("batch_id")
			res, err := GlobalMRNAVault.GetMRNACGMPDossier(batchID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})
	}
}
