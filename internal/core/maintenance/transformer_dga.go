package maintenance

import (
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// TransformerDGATelemetry represents in-line dissolved gas chromatography in transformer dielectric oil
type TransformerDGATelemetry struct {
	TelemetryID             string    `json:"telemetry_id"`
	AssetID                 string    `json:"asset_id"` // e.g. "TRF-MAIN-150KV-02"
	SubstationID            string    `json:"substation_id"`
	HydrogenPPM_H2          float64   `json:"hydrogen_ppm_h2"`          // Partial discharge
	MethanePPM_CH4          float64   `json:"methane_ppm_ch4"`          // Low/medium thermal
	EthanePPM_C2H6          float64   `json:"ethane_ppm_c2h6"`          // High thermal
	EthylenePPM_C2H4        float64   `json:"ethylene_ppm_c2h4"`        // Very high thermal
	AcetylenePPM_C2H2       float64   `json:"acetylene_ppm_c2h2"`       // High-energy electrical arcing (>35 ppm danger)
	CarbonMonoxidePPM_CO    float64   `json:"carbon_monoxide_ppm_co"`   // Paper insulation breakdown
	CarbonDioxidePPM_CO2    float64   `json:"carbon_dioxide_ppm_co2"`
	TotalCombustibleGasTDCG float64   `json:"total_combustible_gas_tdcg"`
	OilTemperatureC         float64   `json:"oil_temperature_c"`
	LoadCurrentAmperes      float64   `json:"load_current_amperes"`
	Timestamp               time.Time `json:"timestamp"`
}

// DuvalFaultDiagnosis models Duval Triangle 1 and IEC 60599 gas ratio fault classifications
type DuvalFaultDiagnosis struct {
	DiagnosisID                 string    `json:"diagnosis_id"`
	AssetID                     string    `json:"asset_id"`
	CH4_RatioPct                float64   `json:"ch4_ratio_pct"`  // % CH4 / (CH4 + C2H4 + C2H2)
	C2H4_RatioPct               float64   `json:"c2h4_ratio_pct"` // % C2H4 / (CH4 + C2H4 + C2H2)
	C2H2_RatioPct               float64   `json:"c2h2_ratio_pct"` // % C2H2 / (CH4 + C2H4 + C2H2)
	IdentifiedFaultType         string    `json:"identified_fault_type"` // "D2_HIGH_ENERGY_ARCING_CRITICAL", "T3_THERMAL_OVER_700C", "PD_PARTIAL_DISCHARGE", "NORMAL_NO_FAULT"
	CellulosePaperCO2toCORatio  float64   `json:"cellulose_paper_co2_to_co_ratio"`
	ActionSeverity              string    `json:"action_severity"` // "CRITICAL_EXPLOSION_HAZARD", "WARNING_INCREASE_SAMPLING", "STABLE_NORMAL"
	RecommendedAction           string    `json:"recommended_action"`
	DiagnosedAt                 time.Time `json:"diagnosed_at"`
}

// SafetyDeratingWorkOrder models autonomous transformer electrical load throttling
type SafetyDeratingWorkOrder struct {
	WorkOrderID                    string    `json:"work_order_id"`
	AssetID                        string    `json:"asset_id"`
	MaxAllowedLoadCapacityPct      float64   `json:"max_allowed_load_capacity_pct"` // e.g. 60.0%
	AvoidedTransformerLossUSD      float64   `json:"avoided_transformer_loss_usd"`
	EmergencyOilDegasification     bool      `json:"emergency_oil_degasification"`
	ScheduledOutageDate            time.Time `json:"scheduled_outage_date"`
	AuthorizedAt                   time.Time `json:"authorized_at"`
}

// TransformerDGAVault thread-safe store
type TransformerDGAVault struct {
	mu           sync.RWMutex
	telemetries  map[string]TransformerDGATelemetry
	diagnoses    map[string]DuvalFaultDiagnosis
	deratings    map[string]SafetyDeratingWorkOrder
}

// Global Transformer DGA instance
var GlobalTransformerDGAVault = &TransformerDGAVault{
	telemetries: make(map[string]TransformerDGATelemetry),
	diagnoses:   make(map[string]DuvalFaultDiagnosis),
	deratings:   make(map[string]SafetyDeratingWorkOrder),
}

func init() {
	// Seed demo substation transformer with incipient arcing telemetry
	GlobalTransformerDGAVault.telemetries["TRF-MAIN-150KV-02"] = TransformerDGATelemetry{
		TelemetryID:             "TEL-DGA-001",
		AssetID:                 "TRF-MAIN-150KV-02",
		SubstationID:            "SUBSTATION-CILEGON-WEST",
		HydrogenPPM_H2:          240.0,
		MethanePPM_CH4:          180.0,
		EthanePPM_C2H6:          75.0,
		EthylenePPM_C2H4:        310.0,
		AcetylenePPM_C2H2:       68.0, // Critical Acetylene trigger (>35 ppm)
		CarbonMonoxidePPM_CO:    520.0,
		CarbonDioxidePPM_CO2:    4200.0,
		TotalCombustibleGasTDCG: 1393.0,
		OilTemperatureC:         78.5,
		LoadCurrentAmperes:      1150.0,
		Timestamp:               time.Now(),
	}
}

// IngestDGATelemetry captures multi-gas chromatography readings
func (v *TransformerDGAVault) IngestDGATelemetry(t TransformerDGATelemetry) TransformerDGATelemetry {
	v.mu.Lock()
	defer v.mu.Unlock()

	t.TotalCombustibleGasTDCG = t.HydrogenPPM_H2 + t.MethanePPM_CH4 + t.EthanePPM_C2H6 + t.EthylenePPM_C2H4 + t.AcetylenePPM_C2H2 + t.CarbonMonoxidePPM_CO
	t.TelemetryID = fmt.Sprintf("TEL-DGA-%d", time.Now().UnixNano()%1000000)
	t.Timestamp = time.Now()
	v.telemetries[t.AssetID] = t
	return t
}

// DiagnoseDuvalFault classifies electrical and thermal defects using Duval Triangle coordinates
func (v *TransformerDGAVault) DiagnoseDuvalFault(assetID string) (*DuvalFaultDiagnosis, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	t, ok := v.telemetries[assetID]
	if !ok {
		return nil, fmt.Errorf("transformer dga telemetry not found for asset: %s", assetID)
	}

	sumGases := t.MethanePPM_CH4 + t.EthylenePPM_C2H4 + t.AcetylenePPM_C2H2
	if sumGases <= 0 {
		sumGases = 1.0
	}

	ch4Pct := (t.MethanePPM_CH4 / sumGases) * 100.0
	c2h4Pct := (t.EthylenePPM_C2H4 / sumGases) * 100.0
	c2h2Pct := (t.AcetylenePPM_C2H2 / sumGases) * 100.0

	ch4Pct = math.Round(ch4Pct*10) / 10
	c2h4Pct = math.Round(c2h4Pct*10) / 10
	c2h2Pct = math.Round(c2h2Pct*10) / 10

	// Ratio of CO2 to CO indicates paper insulation condition
	var co2toCORatio float64 = 8.0
	if t.CarbonMonoxidePPM_CO > 0 {
		co2toCORatio = t.CarbonDioxidePPM_CO2 / t.CarbonMonoxidePPM_CO
		co2toCORatio = math.Round(co2toCORatio*10) / 10
	}

	// Duval Triangle 1 Fault Zones
	fault := "NORMAL_NO_FAULT"
	severity := "STABLE_NORMAL"
	rec := "Dielectric oil gas concentrations within nominal IEEE C57.104 limits."

	if c2h2Pct >= 13.0 {
		// High energy arcing discharge (D2)
		fault = "D2_HIGH_ENERGY_ARCING_CRITICAL"
		severity = "CRITICAL_EXPLOSION_HAZARD"
		rec = "Active electrical arcing detected. Immediately de-rate transformer and schedule emergency outage."
	} else if c2h2Pct >= 4.0 && c2h2Pct < 13.0 {
		// Low energy electrical discharge / sparking (D1)
		fault = "D1_LOW_ENERGY_ARCING"
		severity = "WARNING_INCREASE_SAMPLING"
		rec = "Sparking or track breakdown across insulation barriers. Daily DGA sampling required."
	} else if c2h4Pct >= 50.0 {
		// Thermal fault > 700 C (T3)
		fault = "T3_THERMAL_OVER_700C"
		severity = "CRITICAL_EXPLOSION_HAZARD"
		rec = "Heavy localized core/tank overheating (>700C). Reduce load and inspect oil circulation."
	} else if c2h4Pct >= 20.0 && c2h4Pct < 50.0 && ch4Pct >= 50.0 {
		// Thermal fault 300 - 700 C (T2)
		fault = "T2_THERMAL_300_700C"
		severity = "WARNING_INCREASE_SAMPLING"
		rec = "Moderate thermal fault. Check tap-changer contacts and cooling fan operation."
	} else if ch4Pct >= 98.0 {
		// Partial discharge (PD)
		fault = "PD_PARTIAL_DISCHARGE"
		severity = "WARNING_INCREASE_SAMPLING"
		rec = "Micro-voids gas breakdown. Perform acoustic and electrical PD survey."
	}

	diagID := fmt.Sprintf("DIAG-DUVAL-%d", time.Now().UnixNano()%1000000)
	diagnosis := DuvalFaultDiagnosis{
		DiagnosisID:                diagID,
		AssetID:                    assetID,
		CH4_RatioPct:               ch4Pct,
		C2H4_RatioPct:              c2h4Pct,
		C2H2_RatioPct:              c2h2Pct,
		IdentifiedFaultType:        fault,
		CellulosePaperCO2toCORatio: co2toCORatio,
		ActionSeverity:             severity,
		RecommendedAction:          rec,
		DiagnosedAt:                time.Now(),
	}

	v.diagnoses[assetID] = diagnosis
	return &diagnosis, nil
}

// TriggerSafetyDerating issues load reduction work order to prevent catastrophic tank rupture
func (v *TransformerDGAVault) TriggerSafetyDerating(
	assetID string, deRatePct float64,
) (*SafetyDeratingWorkOrder, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	woID := fmt.Sprintf("WO-DERATE-%d", time.Now().UnixNano()%1000000)
	wo := SafetyDeratingWorkOrder{
		WorkOrderID:                woID,
		AssetID:                    assetID,
		MaxAllowedLoadCapacityPct:  deRatePct,
		AvoidedTransformerLossUSD:  2400000.0, // Replacement cost + business interruption
		EmergencyOilDegasification: true,
		ScheduledOutageDate:        time.Now().Add(48 * time.Hour),
		AuthorizedAt:               time.Now(),
	}

	v.deratings[assetID] = wo
	return &wo, nil
}

// GetInsulationHealthDossier compiles comprehensive substation asset safety report
func (v *TransformerDGAVault) GetInsulationHealthDossier(assetID string) (gin.H, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	tel, hasTel := v.telemetries[assetID]
	diag, hasDiag := v.diagnoses[assetID]
	wo, hasWO := v.deratings[assetID]

	if !hasTel {
		return nil, fmt.Errorf("no records found for transformer asset: %s", assetID)
	}

	return gin.H{
		"asset_id":               assetID,
		"latest_telemetry":       tel,
		"has_telemetry":          hasTel,
		"duval_diagnosis":        diag,
		"has_diagnosis":          hasDiag,
		"safety_derating":        wo,
		"has_derating":           hasWO,
		"diagnostic_standards":   "IEEE C57.104-2019 / IEC 60599 Duval Triangle Method",
		"asset_protection_state": "EXPLOSION_RUPTURE_MITIGATED_BY_DERATE",
	}, nil
}

// RegisterTransformerRoutes registers HTTP endpoints
func RegisterTransformerRoutes(r *gin.RouterGroup) {
	trf := r.Group("/maintenance/transformer")
	{
		trf.POST("/ingest-dga-telemetry", func(c *gin.Context) {
			var req TransformerDGATelemetry
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			t := GlobalTransformerDGAVault.IngestDGATelemetry(req)
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": t})
		})

		trf.POST("/diagnose-duval-fault", func(c *gin.Context) {
			var req struct {
				AssetID string `json:"asset_id" binding:"required"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			diag, err := GlobalTransformerDGAVault.DiagnoseDuvalFault(req.AssetID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": diag})
		})

		trf.POST("/trigger-safety-derating", func(c *gin.Context) {
			var req struct {
				AssetID   string  `json:"asset_id" binding:"required"`
				DeRatePct float64 `json:"derate_pct" binding:"required"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			wo, err := GlobalTransformerDGAVault.TriggerSafetyDerating(req.AssetID, req.DeRatePct)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": wo})
		})

		trf.GET("/insulation-health-dossier/:asset_id", func(c *gin.Context) {
			assetID := c.Param("asset_id")
			dossier, err := GlobalTransformerDGAVault.GetInsulationHealthDossier(assetID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": dossier})
		})
	}
}
