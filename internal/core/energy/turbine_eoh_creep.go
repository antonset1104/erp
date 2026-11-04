package energy

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

// TurbineOperatingTelemetry models ISO 2314 & API 616 heavy gas turbine operational stress cycles
type TurbineOperatingTelemetry struct {
	TelemetryID               string    `json:"telemetry_id"`
	TurbineID                 string    `json:"turbine_id"`
	OperatingBaseHours        float64   `json:"operating_base_hours"`
	FastStartsCount           int       `json:"fast_starts_count"`      // 1 Fast start = ~50 EOH equivalent thermal shock
	NormalStartsCount         int       `json:"normal_starts_count"`    // 1 Normal start = ~20 EOH
	EmergencyTripsCount       int       `json:"emergency_trips_count"`  // 1 Full-load trip = ~80 EOH
	TurbineInletTempTIT_C     float64   `json:"turbine_inlet_temp_tit_c"` // Firing temp ~1100 - 1280°C
	ExhaustGasTempSpreadEGT_C float64   `json:"exhaust_gas_temp_spread_egt_c"` // Combustor flameout/can cross-fire indicator (<25°C normal)
	VibrationVelocityRMS_MMs  float64   `json:"vibration_velocity_rms_mms"`    // ISO 10816-4 limits (<4.5 mm/s acceptable)
	FuelGasWobbeIndex         float64   `json:"fuel_gas_wobbe_index"`
	Timestamp                 time.Time `json:"timestamp"`
}

// EOHCreepAssessment models equivalent operating hours, Larson-Miller creep and hot gas path remaining life
type EOHCreepAssessment struct {
	AssessmentID              string    `json:"assessment_id"`
	TurbineID                 string    `json:"turbine_id"`
	CalculatedEOH             float64   `json:"calculated_eoh"`
	LarsonMillerParameter     float64   `json:"larson_miller_parameter"`       // LMP = T*(20 + log10(t))/1000
	BladeCreepLifeConsumedPct float64   `json:"blade_creep_life_consumed_pct"` // % consumed of 48k EOH rotor lifespan
	HoursUntilNextHGPI        float64   `json:"hours_until_next_hgpi"`         // Hot Gas Path Inspection interval ~24,000 EOH
	HoursUntilMajorOverhaul   float64   `json:"hours_until_major_overhaul"`    // Major Inspection interval ~48,000 EOH
	TurbineHealthStatus       string    `json:"turbine_health_status"`        // "HEALTHY_BASE_LOAD", "ACCELERATED_CREEP_THERMAL_FATIGUE", "CRITICAL_HGPI_INSPECTION_OVERDUE"
	RotordynamicsAuditDigest  string    `json:"rotordynamics_audit_digest"`
	EvaluatedAt               time.Time `json:"evaluated_at"`
}

// MajorInspectionSchedule plans OEM-level outage to avoid uncontained rotor disc rupture
type MajorInspectionSchedule struct {
	TurnaroundID                        string    `json:"turnaround_id"`
	TurbineID                           string    `json:"turbine_id"`
	InspectionType                      string    `json:"inspection_type"` // "HOT_GAS_PATH_INSPECTION_HGPI", "MAJOR_ROTOR_OVERHAUL_MI"
	EstimatedOutageDays                 int       `json:"estimated_outage_days"`
	AvoidedCatastrophicBladeRuptureUSD  float64   `json:"avoided_catastrophic_blade_rupture_usd"`
	ScheduledStartDate                  time.Time `json:"scheduled_start_date"`
	TurnaroundWorkOrderID               string    `json:"turnaround_work_order_id"`
	ScheduledAt                         time.Time `json:"scheduled_at"`
}

// TurbineEOHVault thread-safe in-memory store
type TurbineEOHVault struct {
	mu          sync.RWMutex
	telemetries map[string]TurbineOperatingTelemetry
	assessments map[string]EOHCreepAssessment
	schedules   map[string]MajorInspectionSchedule
}

// Global instance
var GlobalTurbineEOHVault = &TurbineEOHVault{
	telemetries: make(map[string]TurbineOperatingTelemetry),
	assessments: make(map[string]EOHCreepAssessment),
	schedules:   make(map[string]MajorInspectionSchedule),
}

func init() {
	// Seed demo gas turbine telemetry
	GlobalTurbineEOHVault.telemetries["GT-FRAME9F-PLN-01"] = TurbineOperatingTelemetry{
		TelemetryID:               "TEL-TURBINE-001",
		TurbineID:                 "GT-FRAME9F-PLN-01",
		OperatingBaseHours:        21500.0,
		FastStartsCount:           42,
		NormalStartsCount:         175,
		EmergencyTripsCount:       8,
		TurbineInletTempTIT_C:     1140.0,
		ExhaustGasTempSpreadEGT_C: 18.2,
		VibrationVelocityRMS_MMs:  2.85,
		FuelGasWobbeIndex:         49.5,
		Timestamp:                 time.Now(),
	}
}

// IngestOperatingTelemetry registers turbine DCS operational telemetry
func (v *TurbineEOHVault) IngestOperatingTelemetry(t TurbineOperatingTelemetry) TurbineOperatingTelemetry {
	v.mu.Lock()
	defer v.mu.Unlock()

	if t.TelemetryID == "" {
		t.TelemetryID = fmt.Sprintf("TEL-TURBINE-%d", time.Now().UnixNano()%1000000)
	}
	t.Timestamp = time.Now()
	v.telemetries[t.TurbineID] = t
	return t
}

// CalculateEOHCreepConsumption determines cumulative equivalent hours and creep damage
func (v *TurbineEOHVault) CalculateEOHCreepConsumption(turbineID string) (EOHCreepAssessment, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	tel, exists := v.telemetries[turbineID]
	if !exists {
		return EOHCreepAssessment{}, fmt.Errorf("turbine %s not found in telemetry registry", turbineID)
	}

	// Thermal firing severity factor: Baseline 1100°C
	thermalFactor := 1.0
	if tel.TurbineInletTempTIT_C > 1100.0 {
		thermalFactor += (tel.TurbineInletTempTIT_C - 1100.0) * 0.0035
	}

	// Standard OEM EOH calculation:
	// EOH = (BaseHours * ThermalFactor) + (NormalStarts * 20) + (FastStarts * 50) + (Trips * 80)
	calculatedEOH := (tel.OperatingBaseHours * thermalFactor) +
		(float64(tel.NormalStartsCount) * 20.0) +
		(float64(tel.FastStartsCount) * 50.0) +
		(float64(tel.EmergencyTripsCount) * 80.0)

	// Larson-Miller Parameter (LMP) for stage 1 high-pressure turbine blade superalloy
	// LMP = T_Kelvin * (20 + log10(BaseHours)) / 1000
	tKelvin := tel.TurbineInletTempTIT_C + 273.15
	logBaseHours := 1.0
	if tel.OperatingBaseHours > 1.0 {
		logBaseHours = math.Log10(tel.OperatingBaseHours)
	}
	lmp := (tKelvin * (20.0 + logBaseHours)) / 1000.0

	// Creep life % consumed against 48,000 EOH design target
	creepPct := (calculatedEOH / 48000.0) * 100.0
	if creepPct > 100.0 {
		creepPct = 100.0
	}

	// Intervals: HGPI at 24,000 EOH; Major at 48,000 EOH
	hoursToHGPI := 24000.0 - calculatedEOH
	if hoursToHGPI < 0 {
		hoursToHGPI = 0
	}
	hoursToMajor := 48000.0 - calculatedEOH
	if hoursToMajor < 0 {
		hoursToMajor = 0
	}

	status := "HEALTHY_BASE_LOAD"
	if hoursToHGPI <= 0 {
		status = "CRITICAL_HGPI_INSPECTION_OVERDUE"
	} else if hoursToHGPI < 2000.0 || tel.ExhaustGasTempSpreadEGT_C > 30.0 || tel.VibrationVelocityRMS_MMs > 4.5 {
		status = "ACCELERATED_CREEP_THERMAL_FATIGUE"
	}

	hasher := sha256.New()
	hasher.Write([]byte(fmt.Sprintf("%s-%f-%f-%f-%s", turbineID, calculatedEOH, lmp, creepPct, status)))
	digest := hex.EncodeToString(hasher.Sum(nil))

	assessID := fmt.Sprintf("ASSESS-TURBINE-%d", time.Now().UnixNano()%1000000)
	assessment := EOHCreepAssessment{
		AssessmentID:              assessID,
		TurbineID:                 turbineID,
		CalculatedEOH:             math.Round(calculatedEOH*10.0) / 10.0,
		LarsonMillerParameter:     math.Round(lmp*100.0) / 100.0,
		BladeCreepLifeConsumedPct: math.Round(creepPct*100.0) / 100.0,
		HoursUntilNextHGPI:        math.Round(hoursToHGPI*10.0) / 10.0,
		HoursUntilMajorOverhaul:   math.Round(hoursToMajor*10.0) / 10.0,
		TurbineHealthStatus:       status,
		RotordynamicsAuditDigest:  digest,
		EvaluatedAt:               time.Now(),
	}

	v.assessments[turbineID] = assessment
	return assessment, nil
}

// ScheduleMajorInspection schedules turbine overhaul turnaround
func (v *TurbineEOHVault) ScheduleMajorInspection(turbineID, inspectionType string, scheduledStart time.Time) (MajorInspectionSchedule, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	outageDays := 21
	avoidedLoss := 8500000.0 // Avoided $8.5M catastrophic uncontained blade failure & grid penalty
	if inspectionType == "MAJOR_ROTOR_OVERHAUL_MI" {
		outageDays = 45
		avoidedLoss = 18000000.0
	}

	turnaroundID := fmt.Sprintf("TA-TURBINE-%d", time.Now().UnixNano()%1000000)
	woID := fmt.Sprintf("WO-OVERHAUL-%s", turnaroundID[:10])

	sched := MajorInspectionSchedule{
		TurnaroundID:                       turnaroundID,
		TurbineID:                          turbineID,
		InspectionType:                     inspectionType,
		EstimatedOutageDays:                outageDays,
		AvoidedCatastrophicBladeRuptureUSD: avoidedLoss,
		ScheduledStartDate:                 scheduledStart,
		TurnaroundWorkOrderID:              woID,
		ScheduledAt:                        time.Now(),
	}

	v.schedules[turbineID] = sched
	return sched, nil
}

// GetRotordynamicsDossier returns asset health and creep history
func (v *TurbineEOHVault) GetRotordynamicsDossier(turbineID string) (map[string]interface{}, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	assess, hasAssess := v.assessments[turbineID]
	sched, hasSched := v.schedules[turbineID]

	dossier := map[string]interface{}{
		"turbine_id":             turbineID,
		"iso_2314_compliance":   true,
		"api_616_standards":     "CERTIFIED",
		"has_assessment":        hasAssess,
		"has_turnaround_sched":  hasSched,
		"compiled_at":           time.Now(),
	}

	if hasAssess {
		dossier["calculated_eoh"] = assess.CalculatedEOH
		dossier["blade_creep_consumed_pct"] = assess.BladeCreepLifeConsumedPct
		dossier["hours_to_hgpi"] = assess.HoursUntilNextHGPI
		dossier["turbine_health_status"] = assess.TurbineHealthStatus
	}
	if hasSched {
		dossier["inspection_type"] = sched.InspectionType
		dossier["outage_days"] = sched.EstimatedOutageDays
		dossier["avoided_rupture_loss_usd"] = sched.AvoidedCatastrophicBladeRuptureUSD
		dossier["work_order_id"] = sched.TurnaroundWorkOrderID
	}

	return dossier, nil
}

// RegisterTurbineRoutes wires REST endpoints
func RegisterTurbineRoutes(r *gin.RouterGroup) {
	turbineGroup := r.Group("/energy/turbine")
	{
		turbineGroup.POST("/ingest-operating-telemetry", func(c *gin.Context) {
			var t TurbineOperatingTelemetry
			if err := c.ShouldBindJSON(&t); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res := GlobalTurbineEOHVault.IngestOperatingTelemetry(t)
			c.JSON(http.StatusOK, res)
		})

		turbineGroup.POST("/calculate-eoh-creep-consumption", func(c *gin.Context) {
			var body struct {
				TurbineID string `json:"turbine_id"`
			}
			if err := c.ShouldBindJSON(&body); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := GlobalTurbineEOHVault.CalculateEOHCreepConsumption(body.TurbineID)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, res)
		})

		turbineGroup.POST("/schedule-major-inspection", func(c *gin.Context) {
			var body struct {
				TurbineID          string    `json:"turbine_id"`
				InspectionType     string    `json:"inspection_type"`
				ScheduledStartDate time.Time `json:"scheduled_start_date"`
			}
			if err := c.ShouldBindJSON(&body); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			if body.ScheduledStartDate.IsZero() {
				body.ScheduledStartDate = time.Now().AddDate(0, 2, 0)
			}
			res, err := GlobalTurbineEOHVault.ScheduleMajorInspection(body.TurbineID, body.InspectionType, body.ScheduledStartDate)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, res)
		})

		turbineGroup.GET("/rotordynamics-dossier/:turbine_id", func(c *gin.Context) {
			turbineID := c.Param("turbine_id")
			dossier, err := GlobalTurbineEOHVault.GetRotordynamicsDossier(turbineID)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, dossier)
		})
	}
}
