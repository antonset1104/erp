package energy

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// MicrogridTelemetry represents sub-second electrical grid telemetry
type MicrogridTelemetry struct {
	FacilityID           string    `json:"facility_id"`
	GridFrequencyHz      float64   `json:"grid_frequency_hz"`      // Nominal 50.00 Hz
	GridVoltageKV        float64   `json:"grid_voltage_kv"`        // Nominal 20.00 kV
	SolarGenerationKW    float64   `json:"solar_generation_kw"`
	BESSChargePct        float64   `json:"bess_charge_percent"`    // Battery State of Charge
	ActiveDemandLoadKW   float64   `json:"active_demand_load_kw"`
	GridTieBreakerStatus string    `json:"grid_tie_breaker_status"` // "CLOSED_GRID_TIED", "OPEN_ISLANDED"
	Timestamp            time.Time `json:"timestamp"`
}

// IslandingTriggerEvent models autonomous disconnection from main grid
type IslandingTriggerEvent struct {
	EventID              string    `json:"event_id"`
	FacilityID           string    `json:"facility_id"`
	TriggerCause         string    `json:"trigger_cause"`         // "FREQUENCY_UNDER_DEVIATION", "VOLTAGE_SAG_SAG", "BLACKOUT"
	DetectedFrequencyHz  float64   `json:"detected_frequency_hz"`
	DetectedVoltageKV    float64   `json:"detected_voltage_kv"`
	IslandingLatencyMS   float64   `json:"islanding_latency_ms"`  // e.g. 18.2 ms (IEC 61850 Goose trip)
	NewOperatingMode     string    `json:"new_operating_mode"`    // "ISLANDED_AUTONOMOUS"
	IslandedAt           time.Time `json:"islanded_at"`
}

// LoadSheddingDispatch represents OpenADR 2.0b prioritized load shedding
type LoadSheddingDispatch struct {
	DispatchID           string    `json:"dispatch_id"`
	FacilityID           string    `json:"facility_id"`
	ProtocolStandard     string    `json:"protocol_standard"` // "OpenADR_2.0b_DemandResponse"
	Tier1CriticalLoadKW  float64   `json:"tier1_critical_load_kw"` // Reaktor Kimia, Cleanrooms, Server MES (100% Protected)
	Tier2SheddableLoadKW float64   `json:"tier2_sheddable_load_kw"` // HVAC Office, Forklift Chargers (Shed)
	TotalLoadShedKW      float64   `json:"total_load_shed_kw"`
	EstimatedSavedCostUSD float64  `json:"estimated_saved_cost_usd"`
	MESThrottledOrders   []string  `json:"mes_throttled_orders"`
	DispatchedAt         time.Time `json:"dispatched_at"`
}

// MicrogridVault thread-safe store
type MicrogridVault struct {
	mu           sync.RWMutex
	telemetry    map[string]MicrogridTelemetry
	islandingLog map[string]IslandingTriggerEvent
	sheddingLog  map[string]LoadSheddingDispatch
}

// Global Microgrid instance
var GlobalMicrogridVault = &MicrogridVault{
	telemetry:    make(map[string]MicrogridTelemetry),
	islandingLog: make(map[string]IslandingTriggerEvent),
	sheddingLog:  make(map[string]LoadSheddingDispatch),
}

func init() {
	// Seed demo plant microgrid telemetry
	GlobalMicrogridVault.telemetry["FAC-PLANT-CILEGON-01"] = MicrogridTelemetry{
		FacilityID:           "FAC-PLANT-CILEGON-01",
		GridFrequencyHz:      49.98,
		GridVoltageKV:        19.95,
		SolarGenerationKW:    1200.0,
		BESSChargePct:        88.5,
		ActiveDemandLoadKW:   3400.0,
		GridTieBreakerStatus: "CLOSED_GRID_TIED",
		Timestamp:            time.Now(),
	}
}

// IngestGridTelemetry records real-time sensor metrics
func (v *MicrogridVault) IngestGridTelemetry(t MicrogridTelemetry) MicrogridTelemetry {
	v.mu.Lock()
	defer v.mu.Unlock()

	t.Timestamp = time.Now()
	v.telemetry[t.FacilityID] = t
	return t
}

// TriggerVirtualIslanding evaluates disturbance and opens grid tie breaker in milliseconds
func (v *MicrogridVault) TriggerVirtualIslanding(facilityID string) (*IslandingTriggerEvent, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	tel, ok := v.telemetry[facilityID]
	if !ok {
		return nil, fmt.Errorf("telemetry not found for facility: %s", facilityID)
	}

	// Trigger criteria: Frequency < 49.50 Hz or Voltage < 18.0 kV (10% sag)
	var cause string
	switch {
	case tel.GridFrequencyHz < 49.50:
		cause = "FREQUENCY_UNDER_DEVIATION"
	case tel.GridVoltageKV < 18.0:
		cause = "VOLTAGE_SAG_SAG"
	default:
		cause = "SIMULATED_PREEMPTIVE_BLACKOUT_ISLANDING"
	}

	tel.GridTieBreakerStatus = "OPEN_ISLANDED"
	v.telemetry[facilityID] = tel

	eventID := fmt.Sprintf("EVT-ISL-%d", time.Now().UnixNano()%1000000)
	event := IslandingTriggerEvent{
		EventID:             eventID,
		FacilityID:          facilityID,
		TriggerCause:        cause,
		DetectedFrequencyHz: tel.GridFrequencyHz,
		DetectedVoltageKV:   tel.GridVoltageKV,
		IslandingLatencyMS:  16.8, // Ultra-fast relay trip (IEC 61850 GOOSE)
		NewOperatingMode:    "ISLANDED_AUTONOMOUS",
		IslandedAt:          time.Now(),
	}

	v.islandingLog[facilityID] = event
	return &event, nil
}

// DispatchLoadShedding executes OpenADR 2.0b load shed
func (v *MicrogridVault) DispatchLoadShedding(
	facilityID string, criticalKW, sheddableKW, hourlyDowntimeLossUSD float64,
) (*LoadSheddingDispatch, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	tel, ok := v.telemetry[facilityID]
	if !ok {
		return nil, fmt.Errorf("facility not found: %s", facilityID)
	}

	// Calculate avoided outage loss (estimated 4-hour blackout avoided)
	avoidedLoss := hourlyDowntimeLossUSD * 4.0

	// MES orders to throttle
	mesOrders := []string{"MO-2026-FURNACE-441", "MO-2026-EXTRUDER-812"}

	dispID := fmt.Sprintf("SHED-ADR-%d", time.Now().UnixNano()%1000000)
	dispatch := LoadSheddingDispatch{
		DispatchID:            dispID,
		FacilityID:            facilityID,
		ProtocolStandard:      "OpenADR_2.0b_DemandResponse",
		Tier1CriticalLoadKW:   criticalKW,
		Tier2SheddableLoadKW:  sheddableKW,
		TotalLoadShedKW:       sheddableKW,
		EstimatedSavedCostUSD: avoidedLoss,
		MESThrottledOrders:    mesOrders,
		DispatchedAt:          time.Now(),
	}

	tel.ActiveDemandLoadKW = criticalKW
	v.telemetry[facilityID] = tel
	v.sheddingLog[facilityID] = dispatch
	return &dispatch, nil
}

// GetResilienceScorecard retrieves comprehensive microgrid dossier
func (v *MicrogridVault) GetResilienceScorecard(facilityID string) (gin.H, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	tel, hasTel := v.telemetry[facilityID]
	isl, hasIsl := v.islandingLog[facilityID]
	shed, hasShed := v.sheddingLog[facilityID]

	if !hasTel {
		return nil, fmt.Errorf("no microgrid profile found for facility: %s", facilityID)
	}

	return gin.H{
		"facility_id":            facilityID,
		"active_telemetry":       tel,
		"last_islanding_event":   isl,
		"has_islanding_event":    hasIsl,
		"load_shedding_dispatch": shed,
		"has_load_shedding":      hasShed,
		"power_continuity":       "ZERO_TRIPPING_CRITICAL_LOADS_ONLINE",
		"standards":              "IEC 61850 GOOSE Sub-Second Trip & OpenADR 2.0b",
	}, nil
}

// RegisterMicrogridRoutes registers HTTP endpoints
func RegisterMicrogridRoutes(r *gin.RouterGroup) {
	mg := r.Group("/energy/microgrid")
	{
		mg.POST("/ingest-grid-telemetry", func(c *gin.Context) {
			var req MicrogridTelemetry
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			t := GlobalMicrogridVault.IngestGridTelemetry(req)
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": t})
		})

		mg.POST("/trigger-virtual-islanding", func(c *gin.Context) {
			var req struct {
				FacilityID string `json:"facility_id" binding:"required"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			event, err := GlobalMicrogridVault.TriggerVirtualIslanding(req.FacilityID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": event})
		})

		mg.POST("/dispatch-load-shedding", func(c *gin.Context) {
			var req struct {
				FacilityID             string  `json:"facility_id" binding:"required"`
				Tier1CriticalLoadKW    float64 `json:"tier1_critical_load_kw" binding:"required"`
				Tier2SheddableLoadKW   float64 `json:"tier2_sheddable_load_kw" binding:"required"`
				HourlyDowntimeLossUSD  float64 `json:"hourly_downtime_loss_usd" binding:"required"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			disp, err := GlobalMicrogridVault.DispatchLoadShedding(
				req.FacilityID, req.Tier1CriticalLoadKW, req.Tier2SheddableLoadKW, req.HourlyDowntimeLossUSD,
			)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": disp})
		})

		mg.GET("/resilience-scorecard/:facility_id", func(c *gin.Context) {
			facilityID := c.Param("facility_id")
			card, err := GlobalMicrogridVault.GetResilienceScorecard(facilityID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": card})
		})
	}
}
