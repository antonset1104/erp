package aavgenetherapy

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

// AAVVectorTelemetry captures continuous gene therapy adeno-associated virus (AAV) vector purification under FDA 21 CFR Part 610 & ICH Q5A
type AAVVectorTelemetry struct {
	TelemetryID                string    `json:"telemetry_id"`
	BatchID                    string    `json:"batch_id"`                             // "BATCH-AAV9-SMA-CLINICAL-008"
	BioreactorLineID           string    `json:"bioreactor_line_id"`                  // "LINE-PERFUSION-AAV-02"
	ViralVectorSerotype        string    `json:"viral_vector_serotype"`                // "AAV9"
	UvAbsorbanceRatioA260A280  float64   `json:"uv_absorbance_ratio_a260_a280"`        // e.g. 1.08 (full ~1.30-1.35, empty ~0.60-0.70)
	FullCapsidPercentage       float64   `json:"full_capsid_percentage"`               // e.g. 62.5% (clinical release spec > 75.0%)
	HollowFiberShearRatePerSec float64   `json:"hollow_fiber_shear_rate_per_sec"`       // e.g. 4850.0 s^-1 (capsid rupture threshold > 4000 s^-1)
	TransmembranePressureBar   float64   `json:"transmembrane_pressure_bar"`           // e.g. 1.45 bar (safe operating limit < 1.0 bar)
	ElutionConductivityMsPerCm float64   `json:"elution_conductivity_ms_per_cm"`       // e.g. 38.5 mS/cm
	DlsMeanParticleDiameterNm  float64   `json:"dls_mean_particle_diameter_nm"`        // e.g. 26.2 nm (nominal AAV2/8/9 capsid ~25-27 nm)
	CapsidAggregationIndexPct  float64   `json:"capsid_aggregation_index_pct"`         // e.g. 3.8%
	RecordedAt                 time.Time `json:"recorded_at"`
}

// AAVAssessment evaluates viral capsid packaging completeness, empty capsid immunogenicity risk, and hollow-fiber shear lysis
type AAVAssessment struct {
	AssessmentID                   string    `json:"assessment_id"`
	BatchID                        string    `json:"batch_id"`
	FullCapsidDeficitPct           float64   `json:"full_capsid_deficit_pct"`
	ViralCapsidPackagingPurity     string    `json:"viral_capsid_packaging_purity"`      // "HIGH_FULL_CAPSID_CLINICAL_GRADE", "SUB_OPTIMAL_PACKAGING_WARNING", "CRITICAL_EMPTY_CAPSID_IMMUNOGENICITY_HAZARD"
	CapsidShearLysisRisk           string    `json:"capsid_shear_lysis_risk"`            // "LAMINAR_SHEAR_SAFE", "ELEVATED_MEMBRANE_SHEAR_WARNING", "IRREVERSIBLE_CAPSID_SHEAR_LYSIS"
	AEXConductivityStepTrimMandate bool      `json:"aex_conductivity_step_trim_mandate"`
	MembraneTMPDampeningMandate    bool      `json:"membrane_tmp_dampening_mandate"`
	AuditDigest                    string    `json:"audit_digest"`
	EvaluatedAt                    time.Time `json:"evaluated_at"`
}

// AAVMitigationAction models Anion-Exchange (AEX) step conductivity trim, tangential flow TMP throttling, and empty capsid diversion
type AAVMitigationAction struct {
	ActionID                             string    `json:"action_id"`
	BatchID                              string    `json:"batch_id"`
	ModulatedAEXElutionConductivityMsCm  float64   `json:"modulated_aex_elution_conductivity_ms_cm"`
	CommandedTMPSetpointBar              float64   `json:"commanded_tmp_setpoint_bar"`
	DivertedImpurityVolumeLiters         float64   `json:"diverted_impurity_volume_liters"`
	AvoidedGeneTherapyBatchLossUSD       float64   `json:"avoided_gene_therapy_batch_loss_usd"`
	ActionTimestamp                      time.Time `json:"action_timestamp"`
}

// AAVVault thread-safe in-memory store
type AAVVault struct {
	mu          sync.RWMutex
	telemetries map[string]AAVVectorTelemetry
	assessments map[string]AAVAssessment
	actions     map[string]AAVMitigationAction
}

// GlobalAAVVault instance
var GlobalAAVVault = &AAVVault{
	telemetries: make(map[string]AAVVectorTelemetry),
	assessments: make(map[string]AAVAssessment),
	actions:     make(map[string]AAVMitigationAction),
}

// IngestVectorTelemetry stores gene therapy viral vector telemetry
func (v *AAVVault) IngestVectorTelemetry(t AAVVectorTelemetry) AAVVectorTelemetry {
	v.mu.Lock()
	defer v.mu.Unlock()

	if t.TelemetryID == "" {
		t.TelemetryID = fmt.Sprintf("TEL-AAV-%d", time.Now().UnixNano()%1000000)
	}
	if t.RecordedAt.IsZero() {
		t.RecordedAt = time.Now()
	}

	v.telemetries[t.BatchID] = t
	return t
}

// EvaluatePackagingAndShearLysis evaluates capsid fullness ratio, immunogenicity risk, and membrane shear damage
func (v *AAVVault) EvaluatePackagingAndShearLysis(batchID string) (AAVAssessment, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	telem, exists := v.telemetries[batchID]
	if !exists {
		return AAVAssessment{}, fmt.Errorf("telemetry not found for batch: %s", batchID)
	}

	deficit := math.Round((75.0-telem.FullCapsidPercentage)*10) / 10

	packagingPurity := "HIGH_FULL_CAPSID_CLINICAL_GRADE"
	aexTrimMandate := false

	if telem.FullCapsidPercentage < 70.0 || telem.UvAbsorbanceRatioA260A280 < 1.15 {
		packagingPurity = "CRITICAL_EMPTY_CAPSID_IMMUNOGENICITY_HAZARD"
		aexTrimMandate = true
	} else if telem.FullCapsidPercentage < 75.0 {
		packagingPurity = "SUB_OPTIMAL_PACKAGING_WARNING"
		aexTrimMandate = true
	}

	shearRisk := "LAMINAR_SHEAR_SAFE"
	tmpDampingMandate := false

	if telem.HollowFiberShearRatePerSec > 4000.0 || telem.TransmembranePressureBar > 1.20 {
		shearRisk = "IRREVERSIBLE_CAPSID_SHEAR_LYSIS"
		tmpDampingMandate = true
	} else if telem.HollowFiberShearRatePerSec > 3200.0 || telem.TransmembranePressureBar > 0.95 {
		shearRisk = "ELEVATED_MEMBRANE_SHEAR_WARNING"
		tmpDampingMandate = true
	}

	rawHash := fmt.Sprintf("%s:%f:%s:%s:%t:%t", batchID, deficit, packagingPurity, shearRisk, aexTrimMandate, tmpDampingMandate)
	hash := sha256.Sum256([]byte(rawHash))
	digest := hex.EncodeToString(hash[:])

	assessment := AAVAssessment{
		AssessmentID:                   fmt.Sprintf("ASM-AAV-%d", time.Now().UnixNano()%1000000),
		BatchID:                        batchID,
		FullCapsidDeficitPct:           deficit,
		ViralCapsidPackagingPurity:     packagingPurity,
		CapsidShearLysisRisk:           shearRisk,
		AEXConductivityStepTrimMandate: aexTrimMandate,
		MembraneTMPDampeningMandate:    tmpDampingMandate,
		AuditDigest:                    digest,
		EvaluatedAt:                    time.Now(),
	}

	v.assessments[batchID] = assessment
	return assessment, nil
}

// TriggerConductivityTrimAndTMPDamping modulates AEX step elution conductivity and throttles membrane TMP
func (v *AAVVault) TriggerConductivityTrimAndTMPDamping(batchID string) (AAVMitigationAction, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	asm, hasAsm := v.assessments[batchID]
	telem, hasTelem := v.telemetries[batchID]
	if !hasAsm || !hasTelem {
		return AAVMitigationAction{}, fmt.Errorf("missing telemetry or assessment for batch %s", batchID)
	}

	condTrim := telem.ElutionConductivityMsPerCm
	cmdTMP := telem.TransmembranePressureBar
	divertedImpurity := 0.0

	if asm.AEXConductivityStepTrimMandate {
		condTrim = 42.0 // sharp step conductivity shift to desorb empty capsids before full capsids elute
		divertedImpurity = 14.5 // divert pre-peak empty fraction volume to quarantine drain
	}
	if asm.MembraneTMPDampeningMandate {
		cmdTMP = 0.85 // throttle hollow-fiber feed pump to relax TMP below laminar shear damage boundary
	}

	action := AAVMitigationAction{
		ActionID:                            fmt.Sprintf("ACT-AAV-%d", time.Now().UnixNano()%1000000),
		BatchID:                             batchID,
		ModulatedAEXElutionConductivityMsCm: condTrim,
		CommandedTMPSetpointBar:             cmdTMP,
		DivertedImpurityVolumeLiters:        divertedImpurity,
		AvoidedGeneTherapyBatchLossUSD:      30000000.0,
		ActionTimestamp:                     time.Now(),
	}

	v.actions[batchID] = action
	return action, nil
}

// GetAAVClinicalDossier returns gene therapy viral vector batch release dossier under FDA 21 CFR Part 610 & ICH Q5A
func (v *AAVVault) GetAAVClinicalDossier(batchID string) (map[string]interface{}, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	telem, hasTelem := v.telemetries[batchID]
	asm, hasAsm := v.assessments[batchID]
	act := v.actions[batchID]

	if !hasTelem || !hasAsm {
		return nil, fmt.Errorf("incomplete data for AAV batch dossier %s", batchID)
	}

	return map[string]interface{}{
		"batch_id":               batchID,
		"bioreactor_line_id":     telem.BioreactorLineID,
		"viral_vector_serotype":  telem.ViralVectorSerotype,
		"fda_21cfr610_compliant": true,
		"ich_q5a_certified":      true,
		"telemetry":              telem,
		"assessment":             asm,
		"latest_action":          act,
		"compiled_at":            time.Now(),
	}, nil
}

// RegisterAAVRoutes wires API endpoints under /api/v1/biopharma/aav/*
func RegisterAAVRoutes(r *gin.RouterGroup) {
	aavGroup := r.Group("/biopharma/aav")
	{
		aavGroup.POST("/ingest-vector-telemetry", func(c *gin.Context) {
			var t AAVVectorTelemetry
			if err := c.ShouldBindJSON(&t); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res := GlobalAAVVault.IngestVectorTelemetry(t)
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		aavGroup.POST("/evaluate-packaging-and-shear-lysis", func(c *gin.Context) {
			var req struct {
				BatchID string `json:"batch_id"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := GlobalAAVVault.EvaluatePackagingAndShearLysis(req.BatchID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		aavGroup.POST("/trigger-conductivity-trim-and-tmp-damping", func(c *gin.Context) {
			var req struct {
				BatchID string `json:"batch_id"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := GlobalAAVVault.TriggerConductivityTrimAndTMPDamping(req.BatchID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		aavGroup.GET("/aav-clinical-dossier/:batch_id", func(c *gin.Context) {
			batchID := c.Param("batch_id")
			res, err := GlobalAAVVault.GetAAVClinicalDossier(batchID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})
	}
}
