package agriculture

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

// SiloTelemetry represents multi-point sensors embedded across vertical grain bulk silos
type SiloTelemetry struct {
	TelemetryID                     string    `json:"telemetry_id"`
	SiloID                          string    `json:"silo_id"`
	GrainCommodity                  string    `json:"grain_commodity"` // "HARD_RED_WINTER_WHEAT", "YELLOW_CORN_GRADE_2", "SOYBEANS"
	GrainTotalTons                  float64   `json:"grain_total_tons"`
	AverageGrainTempC               float64   `json:"average_grain_temp_c"`
	TopHeadspaceTempC               float64   `json:"top_headspace_temp_c"`
	InterstitialRelativeHumidityPct float64   `json:"interstitial_relative_humidity_pct"` // %
	GrainMoistureContentPct         float64   `json:"grain_moisture_content_pct"`         // % (Target <= 13.5%)
	HeadspaceCO2PPM                 float64   `json:"headspace_co2_ppm"`                  // Early metabolic indicator of insect/fungi (>1000 ppm warning)
	AmbientAirTempC                 float64   `json:"ambient_air_temp_c"`
	AmbientAirRHPct                 float64   `json:"ambient_air_rh_pct"`
	Timestamp                       time.Time `json:"timestamp"`
}

// AerationDecision models thermodynamic aeration fan control and mycotoxin risk assessment
type AerationDecision struct {
	DecisionID                    string    `json:"decision_id"`
	SiloID                        string    `json:"silo_id"`
	FanOperatingCommand           string    `json:"fan_operating_command"` // "AERATION_FAN_ACTIVE_COOLING", "AERATION_FAN_ACTIVE_DRYING", "FANS_OFF_STANDBY", "RECIRCULATION_FUMIGATION_REQUIRED"
	EquilibriumMoistureContentEMC float64   `json:"equilibrium_moisture_content_emc"`
	AerationPsychrometricWindowOK bool      `json:"aeration_psychrometric_window_ok"`
	HotspotDetected               bool      `json:"hotspot_detected"`
	CalculatedSafeStorageDays     float64   `json:"calculated_safe_storage_days"` // Days before mould germination (Safe Storage Life)
	MycotoxinRiskIndex            float64   `json:"mycotoxin_risk_index"`         // 0 - 100 (Aflatoxin / Ochratoxin A hazard)
	RiskClassification            string    `json:"risk_classification"`          // "OPTIMAL_PRESERVATION", "MODERATE_SPOILAGE_RISK", "CRITICAL_MYCOTOXIN_HAZARD"
	Recommendation                string    `json:"recommendation"`
	EvaluatedAt                   time.Time `json:"evaluated_at"`
}

// GrainQuarantineRecord models quarantine of compromised grain batches to prevent downstream food contamination
type GrainQuarantineRecord struct {
	QuarantineID                   string    `json:"quarantine_id"`
	SiloID                         string    `json:"silo_id"`
	BatchLotNumber                 string    `json:"batch_lot_number"`
	QuarantineReason               string    `json:"quarantine_reason"`
	DownstreamDischargeGateLocked  bool      `json:"downstream_discharge_gate_locked"`
	LabSamplingWorkOrderID         string    `json:"lab_sampling_work_order_id"`
	AuditRegulatoryDigest          string    `json:"audit_regulatory_digest"`
	QuarantinedAt                  time.Time `json:"quarantined_at"`
}

// SiloAerationVault thread-safe store
type SiloAerationVault struct {
	mu           sync.RWMutex
	telemetries  map[string]SiloTelemetry
	decisions    map[string]AerationDecision
	quarantines  map[string]GrainQuarantineRecord
}

// Global Silo Aeration instance
var GlobalSiloAerationVault = &SiloAerationVault{
	telemetries: make(map[string]SiloTelemetry),
	decisions:   make(map[string]AerationDecision),
	quarantines: make(map[string]GrainQuarantineRecord),
}

func init() {
	// Seed demo grain terminal silo
	GlobalSiloAerationVault.telemetries["SILO-WHEAT-PORT-01"] = SiloTelemetry{
		TelemetryID:                     "TEL-SILO-001",
		SiloID:                          "SILO-WHEAT-PORT-01",
		GrainCommodity:                  "HARD_RED_WINTER_WHEAT",
		GrainTotalTons:                  15000.0,
		AverageGrainTempC:               28.5,
		TopHeadspaceTempC:               34.0,
		InterstitialRelativeHumidityPct: 78.5,
		GrainMoistureContentPct:         15.2,
		HeadspaceCO2PPM:                 1450.0, // Hotspot respiratory spike
		AmbientAirTempC:                 21.0,
		AmbientAirRHPct:                 55.0,
		Timestamp:                       time.Now(),
	}
}

// IngestTelemetry stores real-time sensor measurements
func (v *SiloAerationVault) IngestTelemetry(t SiloTelemetry) SiloTelemetry {
	v.mu.Lock()
	defer v.mu.Unlock()

	t.TelemetryID = fmt.Sprintf("TEL-SILO-%d", time.Now().UnixNano()%1000000)
	t.Timestamp = time.Now()
	v.telemetries[t.SiloID] = t
	return t
}

// EvaluateAerationTrigger computes Henderson psychrometric equilibrium, SSL days, and fan command
func (v *SiloAerationVault) EvaluateAerationTrigger(siloID string) (*AerationDecision, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	t, ok := v.telemetries[siloID]
	if !ok {
		return nil, fmt.Errorf("silo telemetry not found for id: %s", siloID)
	}

	// Approximate Equilibrium Moisture Content (EMC %) using Henderson-Thompson equation
	// For wheat/cereal: EMC = 100 * [ -ln(1 - RH/100) / (0.00011 * (T + 40)) ]^(1 / 2.1)
	rhFrac := t.AmbientAirRHPct / 100.0
	if rhFrac >= 0.99 {
		rhFrac = 0.99
	}
	if rhFrac <= 0.05 {
		rhFrac = 0.05
	}
	emc := math.Pow(-math.Log(1.0-rhFrac)/(0.00011*(t.AmbientAirTempC+40.0)), 1.0/2.1)
	emc = math.Round(emc*10) / 10

	// Hotspot Detection: High CO2 (>1100 ppm) or Significant Headspace Temp elevation
	hotspot := false
	if t.HeadspaceCO2PPM > 1100.0 || (t.TopHeadspaceTempC-t.AverageGrainTempC) > 4.5 {
		hotspot = true
	}

	// Psychrometric window: Ambient temp must be at least 4°C cooler than grain, and ambient EMC < grain moisture
	psychrometricOK := false
	if (t.AverageGrainTempC-t.AmbientAirTempC) >= 4.0 && emc <= t.GrainMoistureContentPct {
		psychrometricOK = true
	}

	// Safe Storage Life (SSL) calculation based on USDA Grain Storage equations
	// SSL (days) decreases exponentially with temperature and moisture
	tempPenalty := math.Max(0, t.AverageGrainTempC-15.0) * 0.08
	moisturePenalty := math.Max(0, t.GrainMoistureContentPct-13.0) * 0.35
	sslDays := 120.0 * math.Exp(-tempPenalty-moisturePenalty)
	if hotspot {
		sslDays *= 0.35 // Severe reduction under active fungal respiration
	}
	sslDays = math.Max(3.0, math.Round(sslDays*10)/10)

	// Mycotoxin Risk Index (0 - 100)
	var mycotoxinScore float64
	if t.GrainMoistureContentPct > 14.5 && t.AverageGrainTempC > 24.0 {
		mycotoxinScore = 40.0 + (t.GrainMoistureContentPct-14.5)*15.0 + (t.AverageGrainTempC-24.0)*3.5
		if hotspot {
			mycotoxinScore += 25.0
		}
	} else {
		mycotoxinScore = 15.0 + (t.GrainMoistureContentPct-12.0)*4.0
	}
	if mycotoxinScore > 98.0 {
		mycotoxinScore = 98.0
	}
	mycotoxinScore = math.Round(mycotoxinScore*10) / 10

	fanCmd := "FANS_OFF_STANDBY"
	riskClass := "OPTIMAL_PRESERVATION"
	recommendation := "Grain bulk is in stable storage condition. Maintain routine monitoring."

	if mycotoxinScore >= 65.0 || hotspot {
		riskClass = "CRITICAL_MYCOTOXIN_HAZARD"
		if psychrometricOK {
			fanCmd = "AERATION_FAN_ACTIVE_COOLING"
			recommendation = "Emergency cooling fans engaged: Cold ambient window open to suppress fungal hot-spot."
		} else {
			fanCmd = "RECIRCULATION_FUMIGATION_REQUIRED"
			recommendation = "Ambient air too humid/warm for fans. Lock silo outfall and initiate phosphine fumigation."
		}
	} else if t.GrainMoistureContentPct > 14.0 && psychrometricOK {
		fanCmd = "AERATION_FAN_ACTIVE_DRYING"
		riskClass = "MODERATE_SPOILAGE_RISK"
		recommendation = "Optimal psychrometric drying window detected. Continuous aeration active."
	}

	decID := fmt.Sprintf("DEC-AER-%d", time.Now().UnixNano()%1000000)
	decision := AerationDecision{
		DecisionID:                    decID,
		SiloID:                        siloID,
		FanOperatingCommand:           fanCmd,
		EquilibriumMoistureContentEMC: emc,
		AerationPsychrometricWindowOK: psychrometricOK,
		HotspotDetected:               hotspot,
		CalculatedSafeStorageDays:     sslDays,
		MycotoxinRiskIndex:            mycotoxinScore,
		RiskClassification:            riskClass,
		Recommendation:                recommendation,
		EvaluatedAt:                   time.Now(),
	}

	v.decisions[siloID] = decision
	return &decision, nil
}

// QuarantineSpoilageRisk locks discharge slide gates and orders lab mycotoxin ELISA assay
func (v *SiloAerationVault) QuarantineSpoilageRisk(
	siloID, lotNo, reason string,
) (*GrainQuarantineRecord, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	certPayload := fmt.Sprintf("QUARANTINE_GRAIN:%s:%s:%s:%d", siloID, lotNo, reason, time.Now().Unix())
	hashBytes := sha256.Sum256([]byte(certPayload))
	digest := hex.EncodeToString(hashBytes[:])

	quarID := fmt.Sprintf("QRN-GRAIN-%d", time.Now().UnixNano()%1000000)
	labWO := fmt.Sprintf("WO-LAB-ELISA-%d", time.Now().Unix()%10000)

	record := GrainQuarantineRecord{
		QuarantineID:                  quarID,
		SiloID:                        siloID,
		BatchLotNumber:                lotNo,
		QuarantineReason:              reason,
		DownstreamDischargeGateLocked: true,
		LabSamplingWorkOrderID:        labWO,
		AuditRegulatoryDigest:         digest,
		QuarantinedAt:                 time.Now(),
	}

	v.quarantines[siloID] = record
	return &record, nil
}

// GetGrainHealthDossier compiles comprehensive preservation report
func (v *SiloAerationVault) GetGrainHealthDossier(siloID string) (gin.H, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	tel, hasTel := v.telemetries[siloID]
	dec, hasDec := v.decisions[siloID]
	quar, hasQuar := v.quarantines[siloID]

	if !hasTel {
		return nil, fmt.Errorf("no grain silo records found for: %s", siloID)
	}

	return gin.H{
		"silo_id":                 siloID,
		"latest_telemetry":        tel,
		"has_telemetry":           hasTel,
		"aeration_decision":       dec,
		"has_decision":            hasDec,
		"quarantine_record":       quar,
		"has_quarantine":          hasQuar,
		"statutory_standards":     "ISO 6322 Storage of Cereals & Pulses / FAO Post-Harvest / USDA GIPSA",
		"food_safety_cert":        "HACCP Codex Alimentarius Cereals Critical Control Point #3",
	}, nil
}

// RegisterAgricultureRoutes registers HTTP endpoints
func RegisterAgricultureRoutes(r *gin.RouterGroup) {
	agr := r.Group("/agriculture/silo")
	{
		agr.POST("/ingest-telemetry", func(c *gin.Context) {
			var req SiloTelemetry
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			t := GlobalSiloAerationVault.IngestTelemetry(req)
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": t})
		})

		agr.POST("/evaluate-aeration-trigger", func(c *gin.Context) {
			var req struct {
				SiloID string `json:"silo_id" binding:"required"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			dec, err := GlobalSiloAerationVault.EvaluateAerationTrigger(req.SiloID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": dec})
		})

		agr.POST("/quarantine-spoilage-risk", func(c *gin.Context) {
			var req struct {
				SiloID         string `json:"silo_id" binding:"required"`
				BatchLotNumber string `json:"batch_lot_number" binding:"required"`
				Reason         string `json:"reason" binding:"required"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			qr, err := GlobalSiloAerationVault.QuarantineSpoilageRisk(req.SiloID, req.BatchLotNumber, req.Reason)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": qr})
		})

		agr.GET("/grain-health-dossier/:silo_id", func(c *gin.Context) {
			siloID := c.Param("silo_id")
			dossier, err := GlobalSiloAerationVault.GetGrainHealthDossier(siloID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": dossier})
		})
	}
}
