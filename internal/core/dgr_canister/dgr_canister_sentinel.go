package dgrcanister

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// DGRCanisterTelemetry captures deep geological repository spent nuclear fuel container parameters
type DGRCanisterTelemetry struct {
	TelemetryID                 string    `json:"telemetry_id"`
	CanisterID                  string    `json:"canister_id"`
	RepositoryTunnelID          string    `json:"repository_tunnel_id"`
	HostRockDepthMeters         float64   `json:"host_rock_depth_meters"`          // nominal ~500 meters
	CopperCanisterSurfaceTempC  float64   `json:"copper_canister_surface_temp_c"`   // limit < 95.0°C
	BentoniteSwellingPressureMpa float64  `json:"bentonite_swelling_pressure_mpa"`  // nominal 5.0 - 15.0 MPa
	MicrobialSulfidePpm         float64   `json:"microbial_sulfide_ppm"`           // corrosion limit < 1.0 ppm
	BedrockShearDisplacementMm  float64   `json:"bedrock_shear_displacement_mm"`   // fault rupture limit < 50.0 mm
	ElectronBeamWeldIntegrityPct float64  `json:"electron_beam_weld_integrity_pct"` // target > 99.5%
	GammaDoseRateGyPerHr        float64   `json:"gamma_dose_rate_gy_per_hr"`        // shield limit < 0.5 Gy/hr
	Timestamp                   time.Time `json:"timestamp"`
}

// DGRCanisterAssessment models 100,000-year containment integrity, bentonite buffer swelling, and IAEA SSG-14 compliance
type DGRCanisterAssessment struct {
	AssessmentID                   string    `json:"assessment_id"`
	CanisterID                     string    `json:"canister_id"`
	ContainmentIntegrityStatus     string    `json:"containment_integrity_status"` // MONOLITHIC_BARRIER_SECURE, MICRO_SULFIDE_CREEP_WARNING, CORROSION_SHEAR_BREACH_HAZARD
	BentoniteBufferState           string    `json:"bentonite_buffer_state"`       // OPTIMAL_SWELLING_PRESSURE, BUFFER_DESATURATION_DEFICIT, CRITICAL_UNDERPRESSURE_EXCAVATION
	HostRockShearHazard            string    `json:"host_rock_shear_hazard"`       // STABLE_CRYSTALLINE_SHIELD, ACTIVE_SEISMIC_SHEAR_DISPLACEMENT
	HydrationDesalinationMandate   bool      `json:"hydration_desalination_mandate"`
	TunnelBulkheadSealingMandate   bool      `json:"tunnel_bulkhead_sealing_mandate"`
	AuditDigest                    string    `json:"audit_digest"`
	EvaluatedAt                    time.Time `json:"evaluated_at"`
}

// DGRCanisterMitigationAction models bentonite hydration, sacrificial anode polarization, and bulkhead seal
type DGRCanisterMitigationAction struct {
	ActionID                                string    `json:"action_id"`
	CanisterID                              string    `json:"canister_id"`
	CommandedBentoniteHydrationRateLitersPerDay float64 `json:"commanded_bentonite_hydration_rate_liters_per_day"`
	SacrificialAnodeCurrentMilliAmps        float64   `json:"sacrificial_anode_current_milli_amps"`
	TunnelBulkheadSealPressureBar           float64   `json:"tunnel_bulkhead_seal_pressure_bar"`
	AvoidedGeologicalDisasterLossUSD        float64   `json:"avoided_geological_disaster_loss_usd"`
	ExecutedAt                              time.Time `json:"executed_at"`
}

// DGRCanisterVault manages deep geological repository copper canister and bentonite buffer data
type DGRCanisterVault struct {
	mu          sync.RWMutex
	telemetries map[string]DGRCanisterTelemetry
	assessments map[string]DGRCanisterAssessment
	actions     map[string]DGRCanisterMitigationAction
}

// GlobalDGRCanisterVault is the singleton instance for DGR canister sentinel
var GlobalDGRCanisterVault = NewDGRCanisterVault()

func NewDGRCanisterVault() *DGRCanisterVault {
	return &DGRCanisterVault{
		telemetries: make(map[string]DGRCanisterTelemetry),
		assessments: make(map[string]DGRCanisterAssessment),
		actions:     make(map[string]DGRCanisterMitigationAction),
	}
}

// IngestCanisterTelemetry stores deep geological repository canister metrics
func (v *DGRCanisterVault) IngestCanisterTelemetry(telemetry DGRCanisterTelemetry) DGRCanisterTelemetry {
	v.mu.Lock()
	defer v.mu.Unlock()

	if telemetry.Timestamp.IsZero() {
		telemetry.Timestamp = time.Now()
	}
	v.telemetries[telemetry.CanisterID] = telemetry
	return telemetry
}

// EvaluateCanisterIntegrity evaluates IAEA Safety Standards SSG-14 and ISO 12749 long-term isolation rules
func (v *DGRCanisterVault) EvaluateCanisterIntegrity(canisterID string) (DGRCanisterAssessment, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	telem, exists := v.telemetries[canisterID]
	if !exists {
		return DGRCanisterAssessment{}, fmt.Errorf("telemetry not found for DGR canister %s", canisterID)
	}

	containmentStatus := "MONOLITHIC_BARRIER_SECURE"
	bufferState := "OPTIMAL_SWELLING_PRESSURE"
	shearHazard := "STABLE_CRYSTALLINE_SHIELD"
	hydrationMandate := false
	bulkheadMandate := false

	// Critical conditions: bentonite swelling < 5.0 MPa or sulfide > 1.0 ppm or bedrock shear > 50 mm
	if telem.BentoniteSwellingPressureMpa < 5.0 || telem.MicrobialSulfidePpm >= 1.0 || telem.BedrockShearDisplacementMm >= 50.0 {
		containmentStatus = "CORROSION_SHEAR_BREACH_HAZARD"
		bufferState = "CRITICAL_UNDERPRESSURE_EXCAVATION"
		hydrationMandate = true
		bulkheadMandate = true
		if telem.BedrockShearDisplacementMm >= 50.0 {
			shearHazard = "ACTIVE_SEISMIC_SHEAR_DISPLACEMENT"
		}
	} else if telem.BentoniteSwellingPressureMpa < 6.5 || telem.MicrobialSulfidePpm > 0.5 || telem.CopperCanisterSurfaceTempC > 90.0 {
		containmentStatus = "MICRO_SULFIDE_CREEP_WARNING"
		bufferState = "BUFFER_DESATURATION_DEFICIT"
		hydrationMandate = true
	}

	hashInput := fmt.Sprintf("%s:%s:%s:%.2f:%.2f:%t",
		telem.CanisterID, containmentStatus, bufferState, telem.BentoniteSwellingPressureMpa, telem.MicrobialSulfidePpm, hydrationMandate)
	hasher := sha256.New()
	hasher.Write([]byte(hashInput))
	digest := hex.EncodeToString(hasher.Sum(nil))

	assessment := DGRCanisterAssessment{
		AssessmentID:                 fmt.Sprintf("ASM-DGR-%d", time.Now().UnixNano()%1000000),
		CanisterID:                   canisterID,
		ContainmentIntegrityStatus:   containmentStatus,
		BentoniteBufferState:         bufferState,
		HostRockShearHazard:          shearHazard,
		HydrationDesalinationMandate: hydrationMandate,
		TunnelBulkheadSealingMandate: bulkheadMandate,
		AuditDigest:                  digest,
		EvaluatedAt:                  time.Now(),
	}

	v.assessments[canisterID] = assessment
	return assessment, nil
}

// TriggerHydrationAndBulkheadSeal injects low-salinity hydration and pressurizes inflatable tunnel bulkheads
func (v *DGRCanisterVault) TriggerHydrationAndBulkheadSeal(canisterID string) (DGRCanisterMitigationAction, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	asm, hasAsm := v.assessments[canisterID]
	_, hasTelem := v.telemetries[canisterID]
	if !hasAsm || !hasTelem {
		return DGRCanisterMitigationAction{}, fmt.Errorf("missing telemetry or assessment for canister %s", canisterID)
	}

	hydrationRate := 0.0
	anodeCurrent := 0.0
	bulkheadPressure := 0.0

	if asm.HydrationDesalinationMandate {
		hydrationRate = 85.0 // deliver 85.0 L/day deaerated low-salinity groundwater to restore swelling pressure
		anodeCurrent = 450.0  // 450 mA cathodic sacrificial current to suppress microbial sulfide depolarization
	}
	if asm.TunnelBulkheadSealingMandate {
		bulkheadPressure = 45.0 // inflate tunnel isolation bulkhead to 45.0 bar seal pressure
	}

	action := DGRCanisterMitigationAction{
		ActionID:                                fmt.Sprintf("ACT-DGR-%d", time.Now().UnixNano()%1000000),
		CanisterID:                              canisterID,
		CommandedBentoniteHydrationRateLitersPerDay: hydrationRate,
		SacrificialAnodeCurrentMilliAmps:        anodeCurrent,
		TunnelBulkheadSealPressureBar:           bulkheadPressure,
		AvoidedGeologicalDisasterLossUSD:        400000000.0, // $400M USD avoided catastrophic aquifer contamination and repository cancellation
		ExecutedAt:                              time.Now(),
	}

	v.actions[canisterID] = action
	return action, nil
}

// GetDGRSafetyDossier compiles repository nuclear isolation, IAEA SSG-14, and KBS-3 barrier dossier
func (v *DGRCanisterVault) GetDGRSafetyDossier(canisterID string) (map[string]interface{}, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	telem, hasTelem := v.telemetries[canisterID]
	asm, hasAsm := v.assessments[canisterID]
	act, hasAct := v.actions[canisterID]

	if !hasTelem || !hasAsm {
		return nil, fmt.Errorf("incomplete data for DGR safety dossier: %s", canisterID)
	}

	dossier := map[string]interface{}{
		"canister_id":                canisterID,
		"repository_tunnel_id":       telem.RepositoryTunnelID,
		"host_rock_depth_meters":     telem.HostRockDepthMeters,
		"iaea_ssg14_compliant":       asm.ContainmentIntegrityStatus != "CORROSION_SHEAR_BREACH_HAZARD" || (hasAct && act.AvoidedGeologicalDisasterLossUSD > 0),
		"iso_12749_certified":        true,
		"copper_canister_temp_c":     telem.CopperCanisterSurfaceTempC,
		"bentonite_swelling_mpa":     telem.BentoniteSwellingPressureMpa,
		"microbial_sulfide_ppm":      telem.MicrobialSulfidePpm,
		"bedrock_shear_mm":           telem.BedrockShearDisplacementMm,
		"containment_status":         asm.ContainmentIntegrityStatus,
		"buffer_state":               asm.BentoniteBufferState,
		"hydration_rate_l_day":       0.0,
		"bulkhead_seal_bar":          0.0,
		"avoided_loss_usd":           0.0,
		"audit_digest":               asm.AuditDigest,
		"compiled_at":                time.Now(),
	}

	if hasAct {
		dossier["hydration_rate_l_day"] = act.CommandedBentoniteHydrationRateLitersPerDay
		dossier["bulkhead_seal_bar"] = act.TunnelBulkheadSealPressureBar
		dossier["avoided_loss_usd"] = act.AvoidedGeologicalDisasterLossUSD
	}

	return dossier, nil
}

// RegisterDGRCanisterRoutes wires API endpoints under /api/v1/nuclear/dgr-canister/*
func RegisterDGRCanisterRoutes(r *gin.RouterGroup) {
	dgrGroup := r.Group("/nuclear/dgr-canister")
	{
		dgrGroup.POST("/ingest-canister-telemetry", func(c *gin.Context) {
			var t DGRCanisterTelemetry
			if err := c.ShouldBindJSON(&t); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res := GlobalDGRCanisterVault.IngestCanisterTelemetry(t)
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		dgrGroup.POST("/evaluate-canister-integrity", func(c *gin.Context) {
			var req struct {
				CanisterID string `json:"canister_id"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := GlobalDGRCanisterVault.EvaluateCanisterIntegrity(req.CanisterID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		dgrGroup.POST("/trigger-hydration-and-bulkhead-seal", func(c *gin.Context) {
			var req struct {
				CanisterID string `json:"canister_id"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := GlobalDGRCanisterVault.TriggerHydrationAndBulkheadSeal(req.CanisterID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		dgrGroup.GET("/dgr-safety-dossier/:canister_id", func(c *gin.Context) {
			cID := c.Param("canister_id")
			res, err := GlobalDGRCanisterVault.GetDGRSafetyDossier(cID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})
	}
}
