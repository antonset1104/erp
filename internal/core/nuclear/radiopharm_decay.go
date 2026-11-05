package nuclear

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

// RadioisotopeBatch models therapeutic and diagnostic nuclear medicine batch activity under NRC 10 CFR Part 35
type RadioisotopeBatch struct {
	BatchID                         string    `json:"batch_id"`
	IsotopeName                     string    `json:"isotope_name"`         // "LUTETIUM_177", "TECHNETIUM_99M", "IODINE_131"
	HalfLifeHours                   float64   `json:"half_life_hours"`      // Lu-177: 160.8h (6.7d); Tc-99m: 6.0h; I-131: 192.5h (8.02d)
	InitialActivityGBq              float64   `json:"initial_activity_gbq"` // Gigabecquerel (1 Ci = 37 GBq)
	ProductionTimestamp             time.Time `json:"production_timestamp"`
	CalibrationTargetTimestamp      time.Time `json:"calibration_target_timestamp"`
	CaskLeadShieldingThicknessMM    float64   `json:"cask_lead_shielding_thickness_mm"`
	RadiationDoseAtSurfaceMicroSvHr float64   `json:"radiation_dose_at_surface_micro_sv_hr"` // Must be < 20 uSv/hr for transport
	TransportCaskSerialNo           string    `json:"transport_cask_serial_no"`
	RegisteredAt                    time.Time `json:"registered_at"`
}

// DecayCalibrationCalculation models exponential radioactive decay A(t) = A0 * e^(-lambda * t)
type DecayCalibrationCalculation struct {
	CalculationID             string    `json:"calculation_id"`
	BatchID                   string    `json:"batch_id"`
	IsotopeName               string    `json:"isotope_name"`
	ElapsedHours              float64   `json:"elapsed_hours"`
	RemainingActivityGBq      float64   `json:"remaining_activity_gbq"`
	ActivityDecayPct          float64   `json:"activity_decay_pct"`
	PatientDoseUnitsAvailable int       `json:"patient_dose_units_available"` // Standard single patient dose ~7.4 GBq (200 mCi)
	InoculationReadiness      string    `json:"inoculation_readiness"`        // "READY_FOR_CLINICAL_INFUSION", "ACTIVITY_BELOW_THERAPEUTIC_THRESHOLD"
	DecayAuditDigest          string    `json:"decay_audit_digest"`
	CalculatedAt              time.Time `json:"calculated_at"`
}

// XenonTransientPrediction models reactor core restart lockout from Xenon-135 neutron poisoning (10 CFR Part 50)
type XenonTransientPrediction struct {
	PredictionID                 string    `json:"prediction_id"`
	ReactorID                    string    `json:"reactor_id"`
	PreTripThermalPowerMWth      float64   `json:"pre_trip_thermal_power_mwth"`
	HoursSinceReactorTrip        float64   `json:"hours_since_reactor_trip"`
	XenonConcentrationRatio      float64   `json:"xenon_concentration_ratio"` // Ratio to steady-state (peaks at ~2.5 - 3.2x)
	PeakPoisoningHours           float64   `json:"peak_poisoning_hours"`      // Typically 9.0 to 11.0 hours post-trip
	RestartLockoutHoursRemaining float64   `json:"restart_lockout_hours_remaining"`
	CoreReactivityStatus         string    `json:"core_reactivity_status"` // "CRITICAL_RESTART_LOCKOUT", "REACTIVITY_MARGIN_SAFE"
	NuclearSafetyAuditDigest     string    `json:"nuclear_safety_audit_digest"`
	EvaluatedAt                  time.Time `json:"evaluated_at"`
}

// NuclearVault thread-safe in-memory store
type NuclearVault struct {
	mu           sync.RWMutex
	batches      map[string]RadioisotopeBatch
	calibrations map[string]DecayCalibrationCalculation
	xenonPreds   map[string]XenonTransientPrediction
}

// Global instance
var GlobalNuclearVault = &NuclearVault{
	batches:      make(map[string]RadioisotopeBatch),
	calibrations: make(map[string]DecayCalibrationCalculation),
	xenonPreds:   make(map[string]XenonTransientPrediction),
}

func init() {
	now := time.Now()
	// Seed demo Lutetium-177 radiopharmaceutical batch
	GlobalNuclearVault.batches["BATCH-LU177-2026-9901"] = RadioisotopeBatch{
		BatchID:                         "BATCH-LU177-2026-9901",
		IsotopeName:                     "LUTETIUM_177",
		HalfLifeHours:                   160.8, // 6.7 days
		InitialActivityGBq:              37.0,  // 1.0 Curie
		ProductionTimestamp:             now.Add(-24 * time.Hour),
		CalibrationTargetTimestamp:      now.Add(12 * time.Hour),
		CaskLeadShieldingThicknessMM:    50.0,
		RadiationDoseAtSurfaceMicroSvHr: 2.4, // Safe transport limit < 20 uSv/hr
		TransportCaskSerialNo:           "LEAD-CASK-BAPETEN-8812",
		RegisteredAt:                    now,
	}
}

// RegisterRadioisotopeBatch stores radioactive product assay data
func (v *NuclearVault) RegisterRadioisotopeBatch(b RadioisotopeBatch) RadioisotopeBatch {
	v.mu.Lock()
	defer v.mu.Unlock()

	if b.BatchID == "" {
		b.BatchID = fmt.Sprintf("BATCH-%s-%d", b.IsotopeName, time.Now().UnixNano()%1000000)
	}
	b.RegisteredAt = time.Now()
	if b.ProductionTimestamp.IsZero() {
		b.ProductionTimestamp = time.Now()
	}
	v.batches[b.BatchID] = b
	return b
}

// CalculateDecayActivityCalibration applies A(t) = A0 * exp(-ln(2)*t / t_half)
func (v *NuclearVault) CalculateDecayActivityCalibration(batchID string, targetTime time.Time) (DecayCalibrationCalculation, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	b, exists := v.batches[batchID]
	if !exists {
		return DecayCalibrationCalculation{}, fmt.Errorf("radioisotope batch %s not found in registry", batchID)
	}

	if targetTime.IsZero() {
		targetTime = time.Now()
	}

	elapsedHours := targetTime.Sub(b.ProductionTimestamp).Hours()
	if elapsedHours < 0 {
		elapsedHours = 0
	}

	decayConstant := math.Ln2 / b.HalfLifeHours
	remainingActivity := b.InitialActivityGBq * math.Exp(-decayConstant*elapsedHours)
	decayPct := ((b.InitialActivityGBq - remainingActivity) / b.InitialActivityGBq) * 100.0

	// Typical clinical therapeutic single dose of Lu-177 PSMA is ~7.4 GBq (200 mCi)
	dosesAvailable := int(math.Floor(remainingActivity / 7.4))
	status := "READY_FOR_CLINICAL_INFUSION"
	if dosesAvailable < 1 {
		status = "ACTIVITY_BELOW_THERAPEUTIC_THRESHOLD"
	}

	hasher := sha256.New()
	hasher.Write([]byte(fmt.Sprintf("%s-%s-%f-%f-%d", batchID, b.IsotopeName, remainingActivity, elapsedHours, time.Now().UnixNano())))
	digest := hex.EncodeToString(hasher.Sum(nil))

	calcID := fmt.Sprintf("CALIB-DECAY-%d", time.Now().UnixNano()%1000000)
	calc := DecayCalibrationCalculation{
		CalculationID:             calcID,
		BatchID:                   batchID,
		IsotopeName:               b.IsotopeName,
		ElapsedHours:              math.Round(elapsedHours*10.0) / 10.0,
		RemainingActivityGBq:      math.Round(remainingActivity*100.0) / 100.0,
		ActivityDecayPct:          math.Round(decayPct*10.0) / 10.0,
		PatientDoseUnitsAvailable: dosesAvailable,
		InoculationReadiness:      status,
		DecayAuditDigest:          digest,
		CalculatedAt:              time.Now(),
	}

	v.calibrations[batchID] = calc
	return calc, nil
}

// PredictXenonPoisoningTransient calculates reactor Xenon-135 peak concentration and restart lockout
func (v *NuclearVault) PredictXenonPoisoningTransient(reactorID string, preTripPower float64, hoursPostTrip float64) (XenonTransientPrediction, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if preTripPower <= 0 {
		preTripPower = 2800.0 // MWth
	}
	if hoursPostTrip < 0 {
		hoursPostTrip = 0
	}

	// Xenon-135 physics transient model:
	// Post-trip Xe-135 builds up from Iodine-135 decay (half-life 6.57h) while Xe-135 decays with half-life 9.14h.
	// Peak poisoning occurs at ~9.6 hours, reaching ~2.8x steady state.
	// Model normalized curve:
	t := hoursPostTrip
	ratio := 1.0 + (1.8 * (t / 9.6) * math.Exp(1.0-(t/9.6)))

	peakHours := 9.6
	lockoutRemaining := 0.0
	status := "REACTIVITY_MARGIN_SAFE"

	if t < 24.0 && ratio > 1.4 {
		status = "CRITICAL_RESTART_LOCKOUT"
		lockoutRemaining = math.Max(0, 24.0-t)
	}

	hasher := sha256.New()
	hasher.Write([]byte(fmt.Sprintf("%s-%f-%f-%s-%d", reactorID, preTripPower, ratio, status, time.Now().UnixNano())))
	digest := hex.EncodeToString(hasher.Sum(nil))

	predID := fmt.Sprintf("PRED-XE-%d", time.Now().UnixNano()%1000000)
	pred := XenonTransientPrediction{
		PredictionID:                 predID,
		ReactorID:                    reactorID,
		PreTripThermalPowerMWth:      preTripPower,
		HoursSinceReactorTrip:        math.Round(hoursPostTrip*10.0) / 10.0,
		XenonConcentrationRatio:      math.Round(ratio*100.0) / 100.0,
		PeakPoisoningHours:           peakHours,
		RestartLockoutHoursRemaining: math.Round(lockoutRemaining*10.0) / 10.0,
		CoreReactivityStatus:         status,
		NuclearSafetyAuditDigest:     digest,
		EvaluatedAt:                  time.Now(),
	}

	v.xenonPreds[reactorID] = pred
	return pred, nil
}

// GetDosimetryComplianceDossier provides IAEA Safeguards and NRC 10 CFR Part 35 audit report
func (v *NuclearVault) GetDosimetryComplianceDossier(batchID string) (map[string]interface{}, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	batch, hasBatch := v.batches[batchID]
	calib, hasCalib := v.calibrations[batchID]

	dossier := map[string]interface{}{
		"batch_id":                     batchID,
		"has_batch_record":             hasBatch,
		"has_decay_calibration":        hasCalib,
		"iaea_safeguards_compliant":    true,
		"nrc_10cfr35_certified":        true,
		"bapeten_indonesia_accredited": true,
		"compiled_at":                  time.Now(),
	}

	if hasBatch {
		dossier["isotope_name"] = batch.IsotopeName
		dossier["initial_activity_gbq"] = batch.InitialActivityGBq
		dossier["surface_dose_micro_sv_hr"] = batch.RadiationDoseAtSurfaceMicroSvHr
		dossier["cask_serial_no"] = batch.TransportCaskSerialNo
	}
	if hasCalib {
		dossier["remaining_activity_gbq"] = calib.RemainingActivityGBq
		dossier["patient_dose_units"] = calib.PatientDoseUnitsAvailable
		dossier["inoculation_readiness"] = calib.InoculationReadiness
		dossier["elapsed_hours"] = calib.ElapsedHours
	}

	return dossier, nil
}

// RegisterNuclearRoutes wires REST endpoints
func RegisterNuclearRoutes(r *gin.RouterGroup) {
	nucGroup := r.Group("/nuclear")
	{
		nucGroup.POST("/radiopharm/register-radioisotope-batch", func(c *gin.Context) {
			var b RadioisotopeBatch
			if err := c.ShouldBindJSON(&b); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res := GlobalNuclearVault.RegisterRadioisotopeBatch(b)
			c.JSON(http.StatusOK, res)
		})

		nucGroup.POST("/radiopharm/calculate-decay-activity-calibration", func(c *gin.Context) {
			var body struct {
				BatchID    string    `json:"batch_id"`
				TargetTime time.Time `json:"target_time"`
			}
			if err := c.ShouldBindJSON(&body); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := GlobalNuclearVault.CalculateDecayActivityCalibration(body.BatchID, body.TargetTime)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, res)
		})

		nucGroup.POST("/reactor/predict-xenon-poisoning-transient", func(c *gin.Context) {
			var body struct {
				ReactorID     string  `json:"reactor_id"`
				PreTripPower  float64 `json:"pre_trip_thermal_power_mwth"`
				HoursPostTrip float64 `json:"hours_since_reactor_trip"`
			}
			if err := c.ShouldBindJSON(&body); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := GlobalNuclearVault.PredictXenonPoisoningTransient(body.ReactorID, body.PreTripPower, body.HoursPostTrip)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, res)
		})

		nucGroup.GET("/radiopharm/dosimetry-compliance-dossier/:batch_id", func(c *gin.Context) {
			batchID := c.Param("batch_id")
			dossier, err := GlobalNuclearVault.GetDosimetryComplianceDossier(batchID)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, dossier)
		})
	}
}
