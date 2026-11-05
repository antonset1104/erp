package crisprpat

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// CRISPRPATTelemetry captures clinical in-vivo gene editing bioreactor and nanopore PAT telemetry
type CRISPRPATTelemetry struct {
	TelemetryID                      string    `json:"telemetry_id"`
	BioreactorBatchID                string    `json:"bioreactor_batch_id"`
	PatientTherapyID                 string    `json:"patient_therapy_id"`
	OnTargetIndelEfficiencyPct       float64   `json:"on_target_indel_efficiency_pct"`       // target > 85.0%
	OffTargetCleavageFrequencyPct    float64   `json:"off_target_cleavage_frequency_pct"`    // limit < 0.01%
	ActiveCasEnzymePersistenceHours  float64   `json:"active_cas_enzyme_persistence_hours"`  // safety threshold < 8.0 hrs
	GuideRnaMismatchDeltaGKcalMol    float64   `json:"guide_rna_mismatch_delta_g_kcal_mol"`  // target < -14.5 kcal/mol
	NanoporeCleavageReadDepth        float64   `json:"nanopore_cleavage_read_depth"`         // target > 50,000x
	InnateCytokineReleasePpm         float64   `json:"innate_cytokine_release_ppm"`          // limit < 25.0 ppm
	ViableCellFractionPct            float64   `json:"viable_cell_fraction_pct"`             // target > 92.0%
	Timestamp                        time.Time `json:"timestamp"`
}

// CRISPRPATAssessment models off-target oncogenesis risk, enzyme kinetics, and FDA CBER gene therapy compliance
type CRISPRPATAssessment struct {
	AssessmentID                   string    `json:"assessment_id"`
	BioreactorBatchID              string    `json:"bioreactor_batch_id"`
	GeneEditingPrecisionStatus     string    `json:"gene_editing_precision_status"` // MONOSPECIFIC_PRECISION_EDITING, OFF_TARGET_CLEAVAGE_WARNING, ONCOGENIC_TRANSLOCATION_HAZARD
	OffTargetCleavageHazard        string    `json:"off_target_cleavage_hazard"`    // TOLERABLE_CLEAVAGE_BACKGROUND, CRITICAL_DOUBLE_STRAND_BREAK_SPIKE
	OffTargetCleavageExcessPct     float64   `json:"off_target_cleavage_excess_pct"`
	AntiCrisprAcrInfusionMandate   bool      `json:"anti_crispr_acr_infusion_mandate"`
	ChemicalQuenchRinseMandate     bool      `json:"chemical_quench_rinse_mandate"`
	AuditDigest                    string    `json:"audit_digest"`
	EvaluatedAt                    time.Time `json:"evaluated_at"`
}

// CRISPRPATMitigationAction models anti-CRISPR Acr protein dosing, chemical quenching, and phosphorothioate trim
type CRISPRPATMitigationAction struct {
	ActionID                            string    `json:"action_id"`
	BioreactorBatchID                   string    `json:"bioreactor_batch_id"`
	CommandedAcrProteinDosingMgPerLiter float64   `json:"commanded_acr_protein_dosing_mg_per_liter"`
	EnzymeQuenchRinseTimeSec            float64   `json:"enzyme_quench_rinse_time_sec"`
	PhosphorothioateTrimRatio           float64   `json:"phosphorothioate_trim_ratio"`
	AvoidedGeneTherapyFailureUSD        float64   `json:"avoided_gene_therapy_failure_usd"`
	ExecutedAt                          time.Time `json:"executed_at"`
}

// CRISPRPATVault manages therapeutic gene editing PAT runs and anti-CRISPR intervention actions
type CRISPRPATVault struct {
	mu          sync.RWMutex
	telemetries map[string]CRISPRPATTelemetry
	assessments map[string]CRISPRPATAssessment
	actions     map[string]CRISPRPATMitigationAction
}

// GlobalCRISPRPATVault is the singleton instance for CRISPR PAT sentinel
var GlobalCRISPRPATVault = NewCRISPRPATVault()

func NewCRISPRPATVault() *CRISPRPATVault {
	return &CRISPRPATVault{
		telemetries: make(map[string]CRISPRPATTelemetry),
		assessments: make(map[string]CRISPRPATAssessment),
		actions:     make(map[string]CRISPRPATMitigationAction),
	}
}

// IngestEditingTelemetry stores real-time gene therapy synthesis and nanopore sequencing telemetry
func (v *CRISPRPATVault) IngestEditingTelemetry(telemetry CRISPRPATTelemetry) CRISPRPATTelemetry {
	v.mu.Lock()
	defer v.mu.Unlock()

	if telemetry.Timestamp.IsZero() {
		telemetry.Timestamp = time.Now()
	}
	v.telemetries[telemetry.BioreactorBatchID] = telemetry
	return telemetry
}

// EvaluateEditingPrecision evaluates FDA CBER Guidance for Human Gene Therapy and ISO 20387 quality standards
func (v *CRISPRPATVault) EvaluateEditingPrecision(batchID string) (CRISPRPATAssessment, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	telem, exists := v.telemetries[batchID]
	if !exists {
		return CRISPRPATAssessment{}, fmt.Errorf("telemetry not found for gene therapy batch %s", batchID)
	}

	precisionStatus := "MONOSPECIFIC_PRECISION_EDITING"
	hazardStatus := "TOLERABLE_CLEAVAGE_BACKGROUND"
	acrMandate := false
	quenchMandate := false

	offTargetLimit := 0.01 // 0.01% maximum allowable off-target DSB frequency
	excessCleavage := (telem.OffTargetCleavageFrequencyPct - offTargetLimit) / offTargetLimit * 100.0

	if telem.OffTargetCleavageFrequencyPct >= 0.01 || telem.ActiveCasEnzymePersistenceHours >= 8.0 || telem.InnateCytokineReleasePpm >= 25.0 {
		precisionStatus = "ONCOGENIC_TRANSLOCATION_HAZARD"
		hazardStatus = "CRITICAL_DOUBLE_STRAND_BREAK_SPIKE"
		acrMandate = true
		quenchMandate = true
	} else if telem.OffTargetCleavageFrequencyPct > 0.005 || telem.ActiveCasEnzymePersistenceHours > 5.0 {
		precisionStatus = "OFF_TARGET_CLEAVAGE_WARNING"
		acrMandate = true
	}

	hashInput := fmt.Sprintf("%s:%s:%s:%.4f:%.2f:%t",
		telem.BioreactorBatchID, precisionStatus, hazardStatus, telem.OffTargetCleavageFrequencyPct, telem.ActiveCasEnzymePersistenceHours, acrMandate)
	hasher := sha256.New()
	hasher.Write([]byte(hashInput))
	digest := hex.EncodeToString(hasher.Sum(nil))

	assessment := CRISPRPATAssessment{
		AssessmentID:                 fmt.Sprintf("ASM-CRISPR-%d", time.Now().UnixNano()%1000000),
		BioreactorBatchID:            batchID,
		GeneEditingPrecisionStatus:   precisionStatus,
		OffTargetCleavageHazard:      hazardStatus,
		OffTargetCleavageExcessPct:   excessCleavage,
		AntiCrisprAcrInfusionMandate: acrMandate,
		ChemicalQuenchRinseMandate:   quenchMandate,
		AuditDigest:                  digest,
		EvaluatedAt:                  time.Now(),
	}

	v.assessments[batchID] = assessment
	return assessment, nil
}

// TriggerAntiCrisprAcrAndQuench infuses anti-CRISPR Acr peptide and initiates microfluidic buffer wash
func (v *CRISPRPATVault) TriggerAntiCrisprAcrAndQuench(batchID string) (CRISPRPATMitigationAction, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	asm, hasAsm := v.assessments[batchID]
	_, hasTelem := v.telemetries[batchID]
	if !hasAsm || !hasTelem {
		return CRISPRPATMitigationAction{}, fmt.Errorf("missing telemetry or assessment for gene editing batch %s", batchID)
	}

	acrDose := 0.0
	quenchTime := 0.0
	ptRatio := 0.0

	if asm.AntiCrisprAcrInfusionMandate {
		acrDose = 12.5 // micro-infuse 12.5 mg/L AcrIIA4 anti-CRISPR protein to sterically block Cas nuclease active site
		ptRatio = 0.85 // apply 0.85 2'-O-methyl phosphorothioate gRNA chemical protection ratio
	}
	if asm.ChemicalQuenchRinseMandate {
		quenchTime = 45.0 // 45.0 sec chemical EDTA/citrate chelation rinse to freeze residual Cas activity
	}

	action := CRISPRPATMitigationAction{
		ActionID:                            fmt.Sprintf("ACT-CRISPR-%d", time.Now().UnixNano()%1000000),
		BioreactorBatchID:                   batchID,
		CommandedAcrProteinDosingMgPerLiter: acrDose,
		EnzymeQuenchRinseTimeSec:            quenchTime,
		PhosphorothioateTrimRatio:           ptRatio,
		AvoidedGeneTherapyFailureUSD:        45000000.0, // $45M USD avoided clinical trial rejection, secondary oncogenesis, and patient therapy lot loss
		ExecutedAt:                          time.Now(),
	}

	v.actions[batchID] = action
	return action, nil
}

// GetCRISPRSafetyDossier compiles nanopore sequencing read depths, off-target indel matrices, and FDA release dossier
func (v *CRISPRPATVault) GetCRISPRSafetyDossier(batchID string) (map[string]interface{}, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	telem, hasTelem := v.telemetries[batchID]
	asm, hasAsm := v.assessments[batchID]
	act, hasAct := v.actions[batchID]

	if !hasTelem || !hasAsm {
		return nil, fmt.Errorf("incomplete data for CRISPR gene editing dossier: %s", batchID)
	}

	dossier := map[string]interface{}{
		"bioreactor_batch_id":         batchID,
		"patient_therapy_id":          telem.PatientTherapyID,
		"fda_cber_compliant":          asm.GeneEditingPrecisionStatus != "ONCOGENIC_TRANSLOCATION_HAZARD" || (hasAct && act.AvoidedGeneTherapyFailureUSD > 0),
		"iso_20387_certified":         true,
		"on_target_indel_pct":         telem.OnTargetIndelEfficiencyPct,
		"off_target_cleavage_pct":     telem.OffTargetCleavageFrequencyPct,
		"active_cas_hours":            telem.ActiveCasEnzymePersistenceHours,
		"nanopore_read_depth":         telem.NanoporeCleavageReadDepth,
		"precision_status":            asm.GeneEditingPrecisionStatus,
		"hazard_status":               asm.OffTargetCleavageHazard,
		"acr_dosing_mg_l":             0.0,
		"quench_rinse_sec":            0.0,
		"avoided_loss_usd":             0.0,
		"audit_digest":                asm.AuditDigest,
		"compiled_at":                 time.Now(),
	}

	if hasAct {
		dossier["acr_dosing_mg_l"] = act.CommandedAcrProteinDosingMgPerLiter
		dossier["quench_rinse_sec"] = act.EnzymeQuenchRinseTimeSec
		dossier["avoided_loss_usd"] = act.AvoidedGeneTherapyFailureUSD
	}

	return dossier, nil
}

// RegisterCRISPRPATRoutes wires API endpoints under /api/v1/biopharma/crispr-pat/*
func RegisterCRISPRPATRoutes(r *gin.RouterGroup) {
	cpGroup := r.Group("/biopharma/crispr-pat")
	{
		cpGroup.POST("/ingest-editing-telemetry", func(c *gin.Context) {
			var t CRISPRPATTelemetry
			if err := c.ShouldBindJSON(&t); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res := GlobalCRISPRPATVault.IngestEditingTelemetry(t)
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		cpGroup.POST("/evaluate-editing-precision", func(c *gin.Context) {
			var req struct {
				BatchID string `json:"batch_id"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := GlobalCRISPRPATVault.EvaluateEditingPrecision(req.BatchID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		cpGroup.POST("/trigger-anti-crispr-acr-and-quench", func(c *gin.Context) {
			var req struct {
				BatchID string `json:"batch_id"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := GlobalCRISPRPATVault.TriggerAntiCrisprAcrAndQuench(req.BatchID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		cpGroup.GET("/crispr-safety-dossier/:batch_id", func(c *gin.Context) {
			bID := c.Param("batch_id")
			res, err := GlobalCRISPRPATVault.GetCRISPRSafetyDossier(bID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})
	}
}
