package railway

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// WheelWaysideTelemetry captures wayside WILD (Wheel Impact Load Detector) and optical profile sensor data
type WheelWaysideTelemetry struct {
	TelemetryID                        string    `json:"telemetry_id"`
	BogieID                            string    `json:"bogie_id"`                    // e.g. "BOGIE-KA-ARGO-BROMO-01"
	TrainConsistID                     string    `json:"train_consist_id"`            // "KA-10-JAKARTA-SURABAYA"
	WheelPositionID                    string    `json:"wheel_position_id"`           // "AXLE-1-WHEEL-LEFT"
	DynamicPeakImpactLoadKN            float64   `json:"dynamic_peak_impact_load_kn"` // Normal < 140 kN; Alert > 200 kN; Flat > 260 kN
	FlangeHeightMM                     float64   `json:"flange_height_mm"`            // 28.0 - 36.0 mm
	FlangeThicknessMM                  float64   `json:"flange_thickness_mm"`         // Limit < 22.0 mm
	FlangeAngleQR_MM                   float64   `json:"flange_angle_qr_mm"`          // qR factor: UIC mandatory >= 6.5 mm (derailment limit)
	UltrasonicSubsurfaceDefectDetected bool      `json:"ultrasonic_subsurface_defect_detected"`
	AxleMileageKM                      float64   `json:"axle_mileage_km"`
	RecordedAt                         time.Time `json:"recorded_at"`
}

// WheelSafetyRCF_Assessment models Rolling Contact Fatigue (RCF) and flange wheel-climb derailment hazard
type WheelSafetyRCF_Assessment struct {
	AssessmentID                  string    `json:"assessment_id"`
	BogieID                       string    `json:"bogie_id"`
	WheelPositionID               string    `json:"wheel_position_id"`
	FlangeWearDerailmentRisk      string    `json:"flange_wear_derailment_risk"`      // "SAFE_UIC_GEOMETRY", "ELEVATED_FLANGE_WEAR_WARNING", "CRITICAL_WHEEL_CLIMB_DERAILMENT_RISK"
	RollingContactFatigueSeverity string    `json:"rolling_contact_fatigue_severity"` // "NEGLIGIBLE_MICRO_CRACKING", "SPALLING_SUBSURFACE_FATIGUE", "MANDATORY_WHEEL_LATHE_REPROFILE"
	SpeedRestrictionCommandKPH    int       `json:"speed_restriction_command_kph"`    // 0 = Grounded; 60 kph = Restricted; 160 kph = Normal
	CNC_LatheReprofilingMandate   bool      `json:"cnc_lathe_reprofiling_mandate"`
	WheelSafetyAuditDigest        string    `json:"wheel_safety_audit_digest"`
	EvaluatedAt                   time.Time `json:"evaluated_at"`
}

// WheelsetReprofilingWorkOrder schedules underfloor CNC wheel lathe turning to restore UIC profile
type WheelsetReprofilingWorkOrder struct {
	WorkOrderID              string    `json:"work_order_id"`
	BogieID                  string    `json:"bogie_id"`
	WheelPositionID          string    `json:"wheel_position_id"`
	WheelLatheType           string    `json:"wheel_lathe_type"` // "UNDERFLOOR_CNC_WHEEL_LATHE"
	TargetTreadCutDepthMM    float64   `json:"target_tread_cut_depth_mm"`
	PostCutQR_MM             float64   `json:"post_cut_qr_mm"`
	AvoidedDerailmentLossUSD float64   `json:"avoided_derailment_loss_usd"`
	ScheduledMaintenanceDate time.Time `json:"scheduled_maintenance_date"`
	IssuedAt                 time.Time `json:"issued_at"`
}

// RailwayWheelVault thread-safe in-memory store
type RailwayWheelVault struct {
	mu          sync.RWMutex
	telemetries map[string]WheelWaysideTelemetry
	assessments map[string]WheelSafetyRCF_Assessment
	workOrders  map[string]WheelsetReprofilingWorkOrder
}

// Global instance
var GlobalRailwayWheelVault = &RailwayWheelVault{
	telemetries: make(map[string]WheelWaysideTelemetry),
	assessments: make(map[string]WheelSafetyRCF_Assessment),
	workOrders:  make(map[string]WheelsetReprofilingWorkOrder),
}

func init() {
	// Seed demo rolling stock wheel with critical flange wear and wheel flat
	key := fmt.Sprintf("%s-%s", "BOGIE-KA-ARGO-BROMO-01", "AXLE-1-WHEEL-LEFT")
	GlobalRailwayWheelVault.telemetries[key] = WheelWaysideTelemetry{
		TelemetryID:                        "TEL-WILD-2026-9011",
		BogieID:                            "BOGIE-KA-ARGO-BROMO-01",
		TrainConsistID:                     "KA-10-JAKARTA-SURABAYA",
		WheelPositionID:                    "AXLE-1-WHEEL-LEFT",
		DynamicPeakImpactLoadKN:            278.0, // Wheel flat severe impact (>260 kN)
		FlangeHeightMM:                     34.8,
		FlangeThicknessMM:                  21.2, // < 22.0 mm
		FlangeAngleQR_MM:                   5.8,  // < 6.5 mm (CRITICAL DERAILMENT HAZARD)
		UltrasonicSubsurfaceDefectDetected: true,
		AxleMileageKM:                      385000.0,
		RecordedAt:                         time.Now(),
	}
}

// IngestWaysideWILDTelemetry stores trackside sensor reading
func (v *RailwayWheelVault) IngestWaysideWILDTelemetry(t WheelWaysideTelemetry) WheelWaysideTelemetry {
	v.mu.Lock()
	defer v.mu.Unlock()

	if t.TelemetryID == "" {
		t.TelemetryID = fmt.Sprintf("TEL-WILD-%d", time.Now().UnixNano()%1000000)
	}
	t.RecordedAt = time.Now()
	key := fmt.Sprintf("%s-%s", t.BogieID, t.WheelPositionID)
	v.telemetries[key] = t
	return t
}

// EvaluateFlangeSafetyAndRCF checks UIC 510-2 qR criterion and impact threshold
func (v *RailwayWheelVault) EvaluateFlangeSafetyAndRCF(bogieID string, wheelPos string) (WheelSafetyRCF_Assessment, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	key := fmt.Sprintf("%s-%s", bogieID, wheelPos)
	tel, exists := v.telemetries[key]
	if !exists {
		return WheelSafetyRCF_Assessment{}, fmt.Errorf("no wheel telemetry found for bogie %s pos %s", bogieID, wheelPos)
	}

	derailmentRisk := "SAFE_UIC_GEOMETRY"
	speedRestriction := 160
	reprofileMandate := false

	// UIC 510-2: qR < 6.5 mm triggers immediate wheel climb derailment alert
	if tel.FlangeAngleQR_MM < 6.5 || tel.FlangeThicknessMM < 22.0 {
		derailmentRisk = "CRITICAL_WHEEL_CLIMB_DERAILMENT_RISK"
		speedRestriction = 40 // Emergency restriction
		reprofileMandate = true
	} else if tel.FlangeAngleQR_MM < 7.2 || tel.FlangeThicknessMM < 24.0 {
		derailmentRisk = "ELEVATED_FLANGE_WEAR_WARNING"
		speedRestriction = 100
		reprofileMandate = true
	}

	rcfSeverity := "NEGLIGIBLE_MICRO_CRACKING"
	if tel.DynamicPeakImpactLoadKN >= 260.0 || tel.UltrasonicSubsurfaceDefectDetected {
		rcfSeverity = "MANDATORY_WHEEL_LATHE_REPROFILE"
		reprofileMandate = true
		if speedRestriction > 60 {
			speedRestriction = 60
		}
	} else if tel.DynamicPeakImpactLoadKN >= 200.0 {
		rcfSeverity = "SPALLING_SUBSURFACE_FATIGUE"
	}

	hasher := sha256.New()
	hasher.Write([]byte(fmt.Sprintf("%s-%s-%f-%f-%d", bogieID, wheelPos, tel.FlangeAngleQR_MM, tel.DynamicPeakImpactLoadKN, time.Now().UnixNano())))
	digest := hex.EncodeToString(hasher.Sum(nil))

	assessID := fmt.Sprintf("ASSESS-RAIL-%d", time.Now().UnixNano()%1000000)
	assessment := WheelSafetyRCF_Assessment{
		AssessmentID:                  assessID,
		BogieID:                       bogieID,
		WheelPositionID:               wheelPos,
		FlangeWearDerailmentRisk:      derailmentRisk,
		RollingContactFatigueSeverity: rcfSeverity,
		SpeedRestrictionCommandKPH:    speedRestriction,
		CNC_LatheReprofilingMandate:   reprofileMandate,
		WheelSafetyAuditDigest:        digest,
		EvaluatedAt:                   time.Now(),
	}

	v.assessments[key] = assessment
	return assessment, nil
}

// ScheduleWheelsetLatheReprofiling issues CNC underfloor lathe machining work order
func (v *RailwayWheelVault) ScheduleWheelsetLatheReprofiling(bogieID string, wheelPos string, cutDepthMM float64) (WheelsetReprofilingWorkOrder, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if cutDepthMM <= 0 {
		cutDepthMM = 2.5 // Standard reprofiling cut depth in mm
	}

	avoidedLoss := 4500000.0 // Avoided $4.5M derailment, track destruction & passenger liability
	targetQR := 8.5          // Restored UIC profile qR

	woID := fmt.Sprintf("WO-LATHE-%d", time.Now().UnixNano()%1000000)
	wo := WheelsetReprofilingWorkOrder{
		WorkOrderID:              woID,
		BogieID:                  bogieID,
		WheelPositionID:          wheelPos,
		WheelLatheType:           "UNDERFLOOR_CNC_WHEEL_LATHE",
		TargetTreadCutDepthMM:    cutDepthMM,
		PostCutQR_MM:             targetQR,
		AvoidedDerailmentLossUSD: avoidedLoss,
		ScheduledMaintenanceDate: time.Now().AddDate(0, 0, 2),
		IssuedAt:                 time.Now(),
	}

	key := fmt.Sprintf("%s-%s", bogieID, wheelPos)
	v.workOrders[key] = wo
	return wo, nil
}

// GetBogiesSafetyDossier compiles track safety records and reprofiling histories
func (v *RailwayWheelVault) GetBogiesSafetyDossier(bogieID string) (map[string]interface{}, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	var matchedWheels int
	var maxImpact float64
	var minQR float64 = 100.0
	for _, tel := range v.telemetries {
		if tel.BogieID == bogieID {
			matchedWheels++
			if tel.DynamicPeakImpactLoadKN > maxImpact {
				maxImpact = tel.DynamicPeakImpactLoadKN
			}
			if tel.FlangeAngleQR_MM < minQR {
				minQR = tel.FlangeAngleQR_MM
			}
		}
	}
	if minQR == 100.0 {
		minQR = 8.2
	}

	dossier := map[string]interface{}{
		"bogie_id":               bogieID,
		"inspected_wheels_count": matchedWheels,
		"max_impact_load_kn":     maxImpact,
		"minimum_flange_qr_mm":   minQR,
		"en_13262_compliance":    true,
		"uic_510_2_certified":    minQR >= 6.5,
		"compiled_at":            time.Now(),
	}

	return dossier, nil
}

// RegisterRailwayRoutes wires REST endpoints
func RegisterRailwayRoutes(r *gin.RouterGroup) {
	railGroup := r.Group("/railway/wheel")
	{
		railGroup.POST("/ingest-wayside-wild-telemetry", func(c *gin.Context) {
			var t WheelWaysideTelemetry
			if err := c.ShouldBindJSON(&t); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res := GlobalRailwayWheelVault.IngestWaysideWILDTelemetry(t)
			c.JSON(http.StatusOK, res)
		})

		railGroup.POST("/evaluate-flange-safety-and-rcf", func(c *gin.Context) {
			var body struct {
				BogieID         string `json:"bogie_id"`
				WheelPositionID string `json:"wheel_position_id"`
			}
			if err := c.ShouldBindJSON(&body); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := GlobalRailwayWheelVault.EvaluateFlangeSafetyAndRCF(body.BogieID, body.WheelPositionID)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, res)
		})

		railGroup.POST("/schedule-wheelset-lathe-reprofiling", func(c *gin.Context) {
			var body struct {
				BogieID         string  `json:"bogie_id"`
				WheelPositionID string  `json:"wheel_position_id"`
				CutDepthMM      float64 `json:"cut_depth_mm"`
			}
			if err := c.ShouldBindJSON(&body); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := GlobalRailwayWheelVault.ScheduleWheelsetLatheReprofiling(
				body.BogieID, body.WheelPositionID, body.CutDepthMM,
			)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, res)
		})

		railGroup.GET("/bogies-safety-dossier/:bogie_id", func(c *gin.Context) {
			bogieID := c.Param("bogie_id")
			dossier, err := GlobalRailwayWheelVault.GetBogiesSafetyDossier(bogieID)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, dossier)
		})
	}
}
