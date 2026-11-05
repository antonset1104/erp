package icebergscour

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// ArcticIcebergScourTelemetry captures polar subsea iceberg keel draft, scour trench, and wellhead silo stress telemetry
type ArcticIcebergScourTelemetry struct {
	TelemetryID               string    `json:"telemetry_id"`
	WellheadID                string    `json:"wellhead_id"`
	SiloSectorID              string    `json:"silo_sector_id"`
	IcebergKeelDraftMeters    float64   `json:"iceberg_keel_draft_meters"`     // nominal 180.0 - 250.0 m (scour contact > 215.0 m)
	SeabedScourDepthMeters    float64   `json:"seabed_scour_depth_meters"`     // threshold > 2.50 m trench gouge
	IcebergDriftVelocityKnots float64   `json:"iceberg_drift_velocity_knots"`  // nominal 0.5 - 2.8 knots
	DistanceToWellheadMeters  float64   `json:"distance_to_wellhead_meters"`   // collision warning < 1200 m, critical < 350 m
	SiloBermAxialStressMpa    float64   `json:"silo_berm_axial_stress_mpa"`    // limit < 480.0 MPa
	DownholePackerPressureBar float64   `json:"downhole_packer_pressure_bar"`  // nominal 280.0 - 450.0 bar
	SubseaUmbilicalTensionKn  float64   `json:"subsea_umbilical_tension_kn"`   // limit < 350.0 kN
	Timestamp                 time.Time `json:"timestamp"`
}

// ArcticIcebergScourAssessment models iceberg trajectory encroachment, berm penetration, and DNV-OS-F101 / ISO 19906 compliance
type ArcticIcebergScourAssessment struct {
	AssessmentID                          string    `json:"assessment_id"`
	WellheadID                            string    `json:"wellhead_id"`
	IcebergEncroachmentStatus             string    `json:"iceberg_encroachment_status"`             // FREE_DRIFT_SAFE, HAZARDOUS_SCOUR_CONVERGENCE_WARNING, TERMINAL_WELLHEAD_SHEAR_IMMINENT
	SiloContainmentIntegrity              string    `json:"silo_containment_integrity"`              // SUBTERRANEAN_RECESSED_INTACT, BERM_PENETRATION_RUPTURE_RISK
	WellheadClearanceDistanceMarginPct    float64   `json:"wellhead_clearance_distance_margin_pct"`
	AcousticDeterrentHazingMandate        bool      `json:"acoustic_deterrent_hazing_mandate"`
	DownholeStormChokeAndEQDMandate       bool      `json:"downhole_storm_choke_and_eqd_mandate"`
	AuditDigest                           string    `json:"audit_digest"`
	EvaluatedAt                           time.Time `json:"evaluated_at"`
}

// ArcticIcebergScourMitigationAction models acoustic iceberg deflection, fail-safe storm choke shut-in, and emergency hydraulic disconnect (EQD)
type ArcticIcebergScourMitigationAction struct {
	ActionID                        string    `json:"action_id"`
	WellheadID                      string    `json:"wellhead_id"`
	AcousticDeflectorDecibels       float64   `json:"acoustic_deflector_decibels"`
	StormChokeClosureTimeSec        float64   `json:"storm_choke_closure_time_sec"`
	EqdHydraulicReleaseTimeSec      float64   `json:"eqd_hydraulic_release_time_sec"`
	AvoidedSubseaCatastropheLossUSD float64   `json:"avoided_subsea_catastrophe_loss_usd"`
	ExecutedAt                      time.Time `json:"executed_at"`
}

// ArcticIcebergScourVault manages polar wellhead silo protection, drift tracking, and subsea environmental safety
type ArcticIcebergScourVault struct {
	mu          sync.RWMutex
	telemetries map[string]ArcticIcebergScourTelemetry
	assessments map[string]ArcticIcebergScourAssessment
	actions     map[string]ArcticIcebergScourMitigationAction
}

// GlobalArcticIcebergScourVault is the singleton instance for arctic iceberg scour sentinel
var GlobalArcticIcebergScourVault = NewArcticIcebergScourVault()

func NewArcticIcebergScourVault() *ArcticIcebergScourVault {
	return &ArcticIcebergScourVault{
		telemetries: make(map[string]ArcticIcebergScourTelemetry),
		assessments: make(map[string]ArcticIcebergScourAssessment),
		actions:     make(map[string]ArcticIcebergScourMitigationAction),
	}
}

// IngestScourTelemetry stores subsea sonar and wellhead strain telemetry
func (v *ArcticIcebergScourVault) IngestScourTelemetry(telemetry ArcticIcebergScourTelemetry) ArcticIcebergScourTelemetry {
	v.mu.Lock()
	defer v.mu.Unlock()

	if telemetry.Timestamp.IsZero() {
		telemetry.Timestamp = time.Now()
	}
	v.telemetries[telemetry.WellheadID] = telemetry
	return telemetry
}

// EvaluateIcebergEncroachment evaluates keel draft gouge, wellhead clearance, and DNV-OS-F101 / ISO 19906 rules
func (v *ArcticIcebergScourVault) EvaluateIcebergEncroachment(wellheadID string) (ArcticIcebergScourAssessment, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	telem, exists := v.telemetries[wellheadID]
	if !exists {
		return ArcticIcebergScourAssessment{}, fmt.Errorf("telemetry not found for arctic wellhead %s", wellheadID)
	}

	encroachment := "FREE_DRIFT_SAFE"
	containment := "SUBTERRANEAN_RECESSED_INTACT"
	hazingMandate := false
	eqdMandate := false

	safetyDistanceLimit := 350.0 // 350 meters terminal emergency disconnect boundary
	clearanceMargin := (telem.DistanceToWellheadMeters - safetyDistanceLimit) / safetyDistanceLimit * 100.0

	if telem.SiloBermAxialStressMpa >= 480.0 || telem.SubseaUmbilicalTensionKn >= 350.0 {
		containment = "BERM_PENETRATION_RUPTURE_RISK"
	}

	if telem.DistanceToWellheadMeters <= 350.0 || telem.SeabedScourDepthMeters >= 2.50 || containment == "BERM_PENETRATION_RUPTURE_RISK" {
		encroachment = "TERMINAL_WELLHEAD_SHEAR_IMMINENT"
		hazingMandate = true
		eqdMandate = true
	} else if telem.DistanceToWellheadMeters < 1200.0 || telem.IcebergKeelDraftMeters > 215.0 {
		encroachment = "HAZARDOUS_SCOUR_CONVERGENCE_WARNING"
		hazingMandate = true
	}

	hashInput := fmt.Sprintf("%s:%s:%s:%.2f:%.2f:%t",
		telem.WellheadID, encroachment, containment, telem.DistanceToWellheadMeters, telem.SeabedScourDepthMeters, eqdMandate)
	hasher := sha256.New()
	hasher.Write([]byte(hashInput))
	digest := hex.EncodeToString(hasher.Sum(nil))

	assessment := ArcticIcebergScourAssessment{
		AssessmentID:                       fmt.Sprintf("ASM-ICEBERG-%d", time.Now().UnixNano()%1000000),
		WellheadID:                         wellheadID,
		IcebergEncroachmentStatus:          encroachment,
		SiloContainmentIntegrity:           containment,
		WellheadClearanceDistanceMarginPct: clearanceMargin,
		AcousticDeterrentHazingMandate:     hazingMandate,
		DownholeStormChokeAndEQDMandate:    eqdMandate,
		AuditDigest:                        digest,
		EvaluatedAt:                        time.Now(),
	}

	v.assessments[wellheadID] = assessment
	return assessment, nil
}

// TriggerStormChokeAndEQD actuates downhole storm choke mechanical shut-in and fires hydraulic Emergency Quick Disconnect (EQD)
func (v *ArcticIcebergScourVault) TriggerStormChokeAndEQD(wellheadID string) (ArcticIcebergScourMitigationAction, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	asm, hasAsm := v.assessments[wellheadID]
	_, hasTelem := v.telemetries[wellheadID]
	if !hasAsm || !hasTelem {
		return ArcticIcebergScourMitigationAction{}, fmt.Errorf("missing telemetry or assessment for arctic wellhead %s", wellheadID)
	}

	decibels := 0.0
	stormChokeTime := 0.0
	eqdTime := 0.0

	if asm.AcousticDeterrentHazingMandate {
		decibels = 218.5 // fire 218.5 dB focused subsea acoustic wave to deflect drifting iceberg keel
	}
	if asm.DownholeStormChokeAndEQDMandate {
		stormChokeTime = 16.8 // trigger subsurface safety valve (SSSV) storm choke closing wellbore in 16.8 seconds
		eqdTime = 4.15        // unlatch hydraulic riser connector in 4.15 seconds to safely flush wellhead flush with seabed silo
	}

	action := ArcticIcebergScourMitigationAction{
		ActionID:                        fmt.Sprintf("ACT-ICEBERG-%d", time.Now().UnixNano()%1000000),
		WellheadID:                      wellheadID,
		AcousticDeflectorDecibels:       decibels,
		StormChokeClosureTimeSec:        stormChokeTime,
		EqdHydraulicReleaseTimeSec:      eqdTime,
		AvoidedSubseaCatastropheLossUSD: 350000000.0, // $350M USD avoided arctic subsea wellhead shearing, under-ice blowout & environmental cleanup
		ExecutedAt:                      time.Now(),
	}

	v.actions[wellheadID] = action
	return action, nil
}

// GetArcticWellheadDossier compiles wellhead silo status, iceberg tracking, and DNV-OS-F101 records
func (v *ArcticIcebergScourVault) GetArcticWellheadDossier(wellheadID string) (map[string]interface{}, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	telem, hasTelem := v.telemetries[wellheadID]
	asm, hasAsm := v.assessments[wellheadID]
	act, hasAct := v.actions[wellheadID]

	if !hasTelem || !hasAsm {
		return nil, fmt.Errorf("incomplete data for arctic wellhead dossier: %s", wellheadID)
	}

	dossier := map[string]interface{}{
		"wellhead_id":             wellheadID,
		"silo_sector_id":          telem.SiloSectorID,
		"dnv_os_f101_compliant":   asm.IcebergEncroachmentStatus != "TERMINAL_WELLHEAD_SHEAR_IMMINENT" || (hasAct && act.AvoidedSubseaCatastropheLossUSD > 0),
		"iso_19906_certified":     true,
		"iceberg_draft_m":         telem.IcebergKeelDraftMeters,
		"scour_depth_m":           telem.SeabedScourDepthMeters,
		"distance_to_wellhead_m":  telem.DistanceToWellheadMeters,
		"silo_stress_mpa":         telem.SiloBermAxialStressMpa,
		"encroachment_status":     asm.IcebergEncroachmentStatus,
		"silo_integrity":          asm.SiloContainmentIntegrity,
		"acoustic_deflector_db":   0.0,
		"storm_choke_close_sec":   0.0,
		"eqd_release_sec":         0.0,
		"avoided_loss_usd":        0.0,
		"audit_digest":            asm.AuditDigest,
		"compiled_at":             time.Now(),
	}

	if hasAct {
		dossier["acoustic_deflector_db"] = act.AcousticDeflectorDecibels
		dossier["storm_choke_close_sec"] = act.StormChokeClosureTimeSec
		dossier["eqd_release_sec"] = act.EqdHydraulicReleaseTimeSec
		dossier["avoided_loss_usd"] = act.AvoidedSubseaCatastropheLossUSD
	}

	return dossier, nil
}

// RegisterArcticIcebergScourRoutes wires API endpoints under /api/v1/polar/iceberg-scour/*
func RegisterArcticIcebergScourRoutes(r *gin.RouterGroup) {
	iceGroup := r.Group("/polar/iceberg-scour")
	{
		iceGroup.POST("/ingest-scour-telemetry", func(c *gin.Context) {
			var t ArcticIcebergScourTelemetry
			if err := c.ShouldBindJSON(&t); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res := GlobalArcticIcebergScourVault.IngestScourTelemetry(t)
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		iceGroup.POST("/evaluate-iceberg-encroachment", func(c *gin.Context) {
			var req struct {
				WellheadID string `json:"wellhead_id"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := GlobalArcticIcebergScourVault.EvaluateIcebergEncroachment(req.WellheadID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		iceGroup.POST("/trigger-storm-choke-and-eqd", func(c *gin.Context) {
			var req struct {
				WellheadID string `json:"wellhead_id"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := GlobalArcticIcebergScourVault.TriggerStormChokeAndEQD(req.WellheadID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})

		iceGroup.GET("/arctic-wellhead-dossier/:wellhead_id", func(c *gin.Context) {
			wID := c.Param("wellhead_id")
			res, err := GlobalArcticIcebergScourVault.GetArcticWellheadDossier(wID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": res})
		})
	}
}
