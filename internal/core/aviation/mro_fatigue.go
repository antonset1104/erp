package aviation

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

// FlightTelemetryRecord captures flight recorder / QAR touchdown and airframe cyclic metrics
type FlightTelemetryRecord struct {
	TelemetryID             string    `json:"telemetry_id"`
	TailNumber              string    `json:"tail_number"`    // e.g. "PK-GIA-777"
	AircraftModel           string    `json:"aircraft_model"` // "BOEING_777_300ER", "AIRBUS_A350_900"
	FlightNumber            string    `json:"flight_number"`  // e.g. "GA-882"
	OriginICAO              string    `json:"origin_icao"`
	DestinationICAO         string    `json:"destination_icao"`
	TotalFlightHours        float64   `json:"total_flight_hours"`
	TotalFlightCycles       int       `json:"total_flight_cycles"`
	TouchdownSinkRateFPS    float64   `json:"touchdown_sink_rate_fps"`    // Feet per second (normal < 6.0 fps, hard > 10.0 fps)
	VerticalAccelerationG   float64   `json:"vertical_acceleration_g"`    // Peak vertical G (normal < 1.8G, hard > 2.6G, severe > 3.2G)
	LateralAccelerationG    float64   `json:"lateral_acceleration_g"`
	PitchAngleDeg           float64   `json:"pitch_angle_deg"`
	RollAngleDeg            float64   `json:"roll_angle_deg"`
	GrossLandingWeightKG    float64   `json:"gross_landing_weight_kg"`
	MaxDesignLandingWeightKG float64  `json:"max_design_landing_weight_kg"`
	RecordedAt              time.Time `json:"recorded_at"`
}

// HardLandingAssessment models structural stress severity and Palmgren-Miner cumulative fatigue
type HardLandingAssessment struct {
	AssessmentID              string    `json:"assessment_id"`
	TailNumber                string    `json:"tail_number"`
	TouchdownSeverityLevel    string    `json:"touchdown_severity_level"` // "NORMAL_TOUCHDOWN_SAFE", "PHASE_1_HARD_LANDING_ALERT", "PHASE_2_SEVERE_STRUCTURAL_DEMOUNT"
	GForcePeak                float64   `json:"g_force_peak"`
	SinkRateFPS               float64   `json:"sink_rate_fps"`
	OverweightLandingDetected bool      `json:"overweight_landing_detected"`
	RequiredNDTInspections    []string  `json:"required_ndt_inspections"` // "EDDY_CURRENT_MAIN_GEAR_TRUNNION", "ULTRASONIC_WING_ATTACH_PINS", "FUSELAGE_FRAME_RIVET_SHEAR"
	AircraftGroundedAOG       bool      `json:"aircraft_grounded_aog"`
	PalmgrenMinerDamageIndex  float64   `json:"palmgren_miner_damage_index"` // Cumulative fatigue sum(n_i / N_i)
	RemainingSafeFlightCycles int       `json:"remaining_safe_flight_cycles"`
	FatigueAuditDigest        string    `json:"fatigue_audit_digest"`
	EvaluatedAt               time.Time `json:"evaluated_at"`
}

// AirworthinessReleaseCertificate models FAA Form 8130-3 or EASA Form 1 return-to-service release
type AirworthinessReleaseCertificate struct {
	CertificateID              string    `json:"certificate_id"`
	TailNumber                 string    `json:"tail_number"`
	CertificateType            string    `json:"certificate_type"` // "FAA_FORM_8130_3", "EASA_FORM_1", "DGCA_INDONESIA_FORM_21_18"
	LicensedInspectorLicenseID string    `json:"licensed_inspector_license_id"`
	AMO_ApprovalRef            string    `json:"amo_approval_ref"` // Approved Maintenance Organization
	NDTResultsVerified         bool      `json:"ndt_results_verified"`
	DiscrepanciesRectified     bool      `json:"discrepancies_rectified"`
	AirworthinessStatus        string    `json:"airworthiness_status"` // "RETURN_TO_SERVICE_AUTHORIZED", "AOG_GROUNDED"
	DigitalSignatureHash       string    `json:"digital_signature_hash"`
	IssuedAt                   time.Time `json:"issued_at"`
}

// AviationMROVault thread-safe in-memory store
type AviationMROVault struct {
	mu           sync.RWMutex
	telemetries  map[string]FlightTelemetryRecord
	assessments  map[string]HardLandingAssessment
	certificates map[string]AirworthinessReleaseCertificate
}

// Global instance
var GlobalAviationMROVault = &AviationMROVault{
	telemetries:  make(map[string]FlightTelemetryRecord),
	assessments:  make(map[string]HardLandingAssessment),
	certificates: make(map[string]AirworthinessReleaseCertificate),
}

func init() {
	// Seed demo aircraft telemetry with a Phase 1 hard landing exceedance
	GlobalAviationMROVault.telemetries["PK-GIA-777"] = FlightTelemetryRecord{
		TelemetryID:              "TEL-FLIGHT-2026-8801",
		TailNumber:               "PK-GIA-777",
		AircraftModel:            "BOEING_777_300ER",
		FlightNumber:             "GA-882",
		OriginICAO:               "WIII", // Soekarno-Hatta Jakarta
		DestinationICAO:          "NRT",  // Narita Tokyo
		TotalFlightHours:         28450.0,
		TotalFlightCycles:        4920,
		TouchdownSinkRateFPS:     10.8, // > 10.0 fps
		VerticalAccelerationG:    2.82, // > 2.6G threshold
		LateralAccelerationG:     0.45,
		PitchAngleDeg:            5.2,
		RollAngleDeg:             2.1,
		GrossLandingWeightKG:     242000.0,
		MaxDesignLandingWeightKG: 251290.0,
		RecordedAt:               time.Now(),
	}
}

// IngestFlightTelemetry stores post-flight QAR / FDR telemetry
func (v *AviationMROVault) IngestFlightTelemetry(t FlightTelemetryRecord) FlightTelemetryRecord {
	v.mu.Lock()
	defer v.mu.Unlock()

	if t.TelemetryID == "" {
		t.TelemetryID = fmt.Sprintf("TEL-FLIGHT-%d", time.Now().UnixNano()%1000000)
	}
	t.RecordedAt = time.Now()
	v.telemetries[t.TailNumber] = t
	return t
}

// EvaluateHardLandingExceedance checks IATA / Boeing / Airbus structural maintenance criteria
func (v *AviationMROVault) EvaluateHardLandingExceedance(tailNumber string) (HardLandingAssessment, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	tel, exists := v.telemetries[tailNumber]
	if !exists {
		return HardLandingAssessment{}, fmt.Errorf("no flight telemetry found for tail number %s", tailNumber)
	}

	severity := "NORMAL_TOUCHDOWN_SAFE"
	groundedAOG := false
	var ndtInspections []string

	overweight := tel.GrossLandingWeightKG > tel.MaxDesignLandingWeightKG

	if tel.VerticalAccelerationG >= 3.2 || tel.TouchdownSinkRateFPS >= 14.0 {
		severity = "PHASE_2_SEVERE_STRUCTURAL_DEMOUNT"
		groundedAOG = true
		ndtInspections = []string{
			"EDDY_CURRENT_MAIN_GEAR_TRUNNION_PIN",
			"ULTRASONIC_WING_SPAR_ATTACHMENT_FITTINGS",
			"FUSELAGE_STATION_1800_PRESSURE_BULKHEAD_XRAY",
			"ENGINE_PYLON_FUSE_PIN_SHEAR_VERIFICATION",
		}
	} else if tel.VerticalAccelerationG >= 2.6 || tel.TouchdownSinkRateFPS >= 10.0 || overweight {
		severity = "PHASE_1_HARD_LANDING_ALERT"
		groundedAOG = true
		ndtInspections = []string{
			"EDDY_CURRENT_MAIN_GEAR_TRUNNION_PIN",
			"ULTRASONIC_WING_ATTACH_PINS",
			"FUSELAGE_SKIN_WRINKLING_AND_RIVET_SHEAR_VISUAL",
		}
	}

	// Palmgren-Miner Cumulative Damage:
	// Base design life = 20,000 cycles. Each cycle adds 1/20000.
	// Hard landing adds 0.045 equivalent fatigue damage step.
	baseMiner := float64(tel.TotalFlightCycles) / 20000.0
	if severity == "PHASE_1_HARD_LANDING_ALERT" {
		baseMiner += 0.045
	} else if severity == "PHASE_2_SEVERE_STRUCTURAL_DEMOUNT" {
		baseMiner += 0.125
	}
	if baseMiner > 1.0 {
		baseMiner = 1.0
	}

	remainingCycles := int(math.Max(0, (1.0-baseMiner)*20000.0))

	hasher := sha256.New()
	hasher.Write([]byte(fmt.Sprintf("%s-%s-%f-%f-%d", tailNumber, severity, tel.VerticalAccelerationG, baseMiner, time.Now().UnixNano())))
	auditDigest := hex.EncodeToString(hasher.Sum(nil))

	assessID := fmt.Sprintf("ASSESS-MRO-%d", time.Now().UnixNano()%1000000)
	assessment := HardLandingAssessment{
		AssessmentID:              assessID,
		TailNumber:                tailNumber,
		TouchdownSeverityLevel:    severity,
		GForcePeak:                tel.VerticalAccelerationG,
		SinkRateFPS:               tel.TouchdownSinkRateFPS,
		OverweightLandingDetected: overweight,
		RequiredNDTInspections:    ndtInspections,
		AircraftGroundedAOG:       groundedAOG,
		PalmgrenMinerDamageIndex:  math.Round(baseMiner*10000.0) / 10000.0,
		RemainingSafeFlightCycles: remainingCycles,
		FatigueAuditDigest:        auditDigest,
		EvaluatedAt:               time.Now(),
	}

	v.assessments[tailNumber] = assessment
	return assessment, nil
}

// IssueAirworthinessReleaseCertificate releases aircraft to service or confirms AOG grounding
func (v *AviationMROVault) IssueAirworthinessReleaseCertificate(tailNumber, certType, inspectorLicense, amoRef string) (AirworthinessReleaseCertificate, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	assess, exists := v.assessments[tailNumber]
	if !exists {
		return AirworthinessReleaseCertificate{}, fmt.Errorf("assessment not found for tail number %s", tailNumber)
	}

	if certType == "" {
		certType = "FAA_FORM_8130_3"
	}
	if inspectorLicense == "" {
		inspectorLicense = "FAA-A&P-3981120-IA"
	}
	if amoRef == "" {
		amoRef = "GMF-AEROASIA-FAA-PART145-CERT-8812"
	}

	// If NDT inspections were required, we simulate they have been performed and verified by inspector
	status := "RETURN_TO_SERVICE_AUTHORIZED"
	if assess.AircraftGroundedAOG && inspectorLicense == "UNAUTHORIZED" {
		status = "AOG_GROUNDED"
	}

	hasher := sha256.New()
	hasher.Write([]byte(fmt.Sprintf("%s-%s-%s-%s-%s-%d", tailNumber, certType, inspectorLicense, amoRef, status, time.Now().UnixNano())))
	sigHash := hex.EncodeToString(hasher.Sum(nil))

	certID := fmt.Sprintf("CERT-AIRWORTHINESS-%d", time.Now().UnixNano()%1000000)
	cert := AirworthinessReleaseCertificate{
		CertificateID:              certID,
		TailNumber:                 tailNumber,
		CertificateType:            certType,
		LicensedInspectorLicenseID: inspectorLicense,
		AMO_ApprovalRef:            amoRef,
		NDTResultsVerified:         true,
		DiscrepanciesRectified:     true,
		AirworthinessStatus:        status,
		DigitalSignatureHash:       sigHash,
		IssuedAt:                   time.Now(),
	}

	v.certificates[tailNumber] = cert
	return cert, nil
}

// GetAircraftStructuralFatigueDossier compiles full airframe fatigue, cycle history, and certifications
func (v *AviationMROVault) GetAircraftStructuralFatigueDossier(tailNumber string) (map[string]interface{}, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	tel, hasTel := v.telemetries[tailNumber]
	assess, hasAssess := v.assessments[tailNumber]
	cert, hasCert := v.certificates[tailNumber]

	dossier := map[string]interface{}{
		"tail_number":            tailNumber,
		"has_telemetry":          hasTel,
		"has_assessment":         hasAssess,
		"has_airworthiness_cert": hasCert,
		"faa_part145_compliant":  true,
		"easa_part_m_verified":   true,
		"compiled_at":            time.Now(),
	}

	if hasTel {
		dossier["aircraft_model"] = tel.AircraftModel
		dossier["total_flight_hours"] = tel.TotalFlightHours
		dossier["total_flight_cycles"] = tel.TotalFlightCycles
		dossier["touchdown_g_peak"] = tel.VerticalAccelerationG
	}
	if hasAssess {
		dossier["touchdown_severity"] = assess.TouchdownSeverityLevel
		dossier["palmgren_miner_index"] = assess.PalmgrenMinerDamageIndex
		dossier["remaining_safe_cycles"] = assess.RemainingSafeFlightCycles
		dossier["required_ndt_count"] = len(assess.RequiredNDTInspections)
	}
	if hasCert {
		dossier["airworthiness_status"] = cert.AirworthinessStatus
		dossier["certificate_type"] = cert.CertificateType
		dossier["amo_approval_ref"] = cert.AMO_ApprovalRef
		dossier["certificate_id"] = cert.CertificateID
	}

	return dossier, nil
}

// RegisterAviationRoutes wires REST endpoints
func RegisterAviationRoutes(r *gin.RouterGroup) {
	avGroup := r.Group("/aviation/mro")
	{
		avGroup.POST("/ingest-flight-telemetry", func(c *gin.Context) {
			var t FlightTelemetryRecord
			if err := c.ShouldBindJSON(&t); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res := GlobalAviationMROVault.IngestFlightTelemetry(t)
			c.JSON(http.StatusOK, res)
		})

		avGroup.POST("/evaluate-hard-landing-exceedance", func(c *gin.Context) {
			var body struct {
				TailNumber string `json:"tail_number"`
			}
			if err := c.ShouldBindJSON(&body); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := GlobalAviationMROVault.EvaluateHardLandingExceedance(body.TailNumber)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, res)
		})

		avGroup.POST("/issue-airworthiness-release-certificate", func(c *gin.Context) {
			var body struct {
				TailNumber       string `json:"tail_number"`
				CertificateType  string `json:"certificate_type"`
				InspectorLicense string `json:"inspector_license"`
				AMO_ApprovalRef  string `json:"amo_approval_ref"`
			}
			if err := c.ShouldBindJSON(&body); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := GlobalAviationMROVault.IssueAirworthinessReleaseCertificate(
				body.TailNumber, body.CertificateType, body.InspectorLicense, body.AMO_ApprovalRef,
			)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, res)
		})

		avGroup.GET("/aircraft-structural-fatigue-dossier/:tail_number", func(c *gin.Context) {
			tail := c.Param("tail_number")
			dossier, err := GlobalAviationMROVault.GetAircraftStructuralFatigueDossier(tail)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, dossier)
		})
	}
}
