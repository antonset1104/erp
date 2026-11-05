package asset

import (
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// SubMeterTelemetry represents high-frequency IoT power measurement from industrial machinery.
type SubMeterTelemetry struct {
	MeterID           string  `json:"meter_id"`
	WorkCenter        string  `json:"work_center"`
	ActivePowerKW     float64 `json:"active_power_kw"`
	ReactivePowerKVAR float64 `json:"reactive_power_kvar"`
	PowerFactor       float64 `json:"power_factor"`
	Timestamp         string  `json:"timestamp"`
}

// PeakShavingRequest holds parameters to evaluate grid peak demand and execute automated load-shedding.
type PeakShavingRequest struct {
	FacilityID            string  `json:"facility_id"`
	ContractDemandLimitKW float64 `json:"contract_demand_limit_kw"`
	CurrentTotalDemandKW  float64 `json:"current_total_demand_kw"`
	GridTariffPeriod      string  `json:"grid_tariff_period"`     // "ON_PEAK_WBP", "OFF_PEAK_LWBP"
	BatteryBESSStatePct   float64 `json:"battery_bess_state_pct"` // 0-100%
}

// PeakShavingDispatchOrder captures automated energy dispatch decisions to avoid penalty demand charges.
type PeakShavingDispatchOrder struct {
	DispatchID               string   `json:"dispatch_id"`
	FacilityID               string   `json:"facility_id"`
	GridTariffPeriod         string   `json:"grid_tariff_period"`
	CurrentDemandKW          float64  `json:"current_demand_kw"`
	ContractDemandLimitKW    float64  `json:"contract_demand_limit_kw"`
	OverLimitExcessKW        float64  `json:"over_limit_excess_kw"`
	ShavingTargetKW          float64  `json:"shaving_target_kw"`
	ActionsExecuted          []string `json:"actions_executed"`
	BESSDischargeKW          float64  `json:"bess_discharge_kw"`
	DeferredBatchProcesses   []string `json:"deferred_batch_processes"`
	AvoidedPeakDemandCostIDR float64  `json:"avoided_peak_demand_cost_idr"`
	NetEnergySavedKWh        float64  `json:"net_energy_saved_kwh"`
	Status                   string   `json:"status"` // "EXECUTED_LOAD_SHED", "NORMAL_BASELINE"
	ExecutedAt               string   `json:"executed_at"`
}

// ISO50001AuditReport delivers statutory energy management system compliance metrics.
type ISO50001AuditReport struct {
	FacilityName                  string  `json:"facility_name"`
	AuditYear                     int     `json:"audit_year"`
	TotalConsumptionMWh           float64 `json:"total_consumption_mwh"`
	PeakDemandRecordedKW          float64 `json:"peak_demand_recorded_kw"`
	ContractDemandLimitKW         float64 `json:"contract_demand_limit_kw"`
	PeakShavingEventsCount        int     `json:"peak_shaving_events_count"`
	TotalCostAvoidedIDR           float64 `json:"total_cost_avoided_idr"`
	CarbonReductionTonsCO2        float64 `json:"carbon_reduction_tons_co2"`
	EnergyPerformanceIndexEnPI    float64 `json:"energy_performance_index_enpi"` // kWh per ton finished product
	ComplianceCertificationStatus string  `json:"compliance_certification_status"`
	GeneratedAt                   string  `json:"generated_at"`
}

// EnergyOptimizerStore manages thread-safe energy dispatch state.
type EnergyOptimizerStore struct {
	mu           sync.RWMutex
	readings     []SubMeterTelemetry
	orders       []PeakShavingDispatchOrder
	auditSummary *ISO50001AuditReport
}

var globalEnergyStore = &EnergyOptimizerStore{
	readings: make([]SubMeterTelemetry, 0),
	orders:   make([]PeakShavingDispatchOrder, 0),
}

func init() {
	globalEnergyStore.auditSummary = &ISO50001AuditReport{
		FacilityName:                  "GoERP Cikarang Heavy Industrial & Precision Casting Complex",
		AuditYear:                     2026,
		TotalConsumptionMWh:           28450.0,
		PeakDemandRecordedKW:          8200.0,
		ContractDemandLimitKW:         8500.0,
		PeakShavingEventsCount:        14,
		TotalCostAvoidedIDR:           485000000.0,
		CarbonReductionTonsCO2:        182.5,
		EnergyPerformanceIndexEnPI:    142.8,
		ComplianceCertificationStatus: "CERTIFIED_ISO50001_COMPLIANT",
		GeneratedAt:                   time.Now().Format(time.RFC3339),
	}
}

// IngestTelemetryLogic records active machine sub-meter readings.
func IngestTelemetryLogic(reading SubMeterTelemetry) (*SubMeterTelemetry, error) {
	globalEnergyStore.mu.Lock()
	defer globalEnergyStore.mu.Unlock()

	if reading.MeterID == "" {
		reading.MeterID = fmt.Sprintf("MTR-%d", time.Now().UnixNano()%1000000)
	}
	if reading.Timestamp == "" {
		reading.Timestamp = time.Now().Format(time.RFC3339)
	}
	if reading.PowerFactor <= 0 {
		reading.PowerFactor = 0.95
	}

	globalEnergyStore.readings = append(globalEnergyStore.readings, reading)
	return &reading, nil
}

// OrchestratePeakShavingLogic evaluates demand threshold and triggers automated battery BESS / load shifting.
func OrchestratePeakShavingLogic(req PeakShavingRequest) (*PeakShavingDispatchOrder, error) {
	globalEnergyStore.mu.Lock()
	defer globalEnergyStore.mu.Unlock()

	if req.FacilityID == "" {
		req.FacilityID = "PLANT-CIKARANG-01"
	}
	if req.ContractDemandLimitKW <= 0 {
		req.ContractDemandLimitKW = 8000.0
	}
	if req.CurrentTotalDemandKW <= 0 {
		req.CurrentTotalDemandKW = 8650.0
	}
	if req.GridTariffPeriod == "" {
		req.GridTariffPeriod = "ON_PEAK_WBP"
	}

	excess := req.CurrentTotalDemandKW - req.ContractDemandLimitKW
	dispatchID := fmt.Sprintf("DSP-ENERGY-%d", time.Now().UnixNano()%1000000)

	var actions []string
	var deferred []string
	var bessKW float64
	var avoidedCostIDR float64
	var energySavedKWh float64
	status := "NORMAL_BASELINE"

	if excess > 0 || req.GridTariffPeriod == "ON_PEAK_WBP" {
		status = "EXECUTED_LOAD_SHED"
		targetShave := math.Max(excess+200.0, 500.0) // 200kW safety margin

		if req.BatteryBESSStatePct >= 30.0 {
			bessKW = math.Min(targetShave*0.60, 600.0)
			actions = append(actions, fmt.Sprintf("Discharged %.0f kW from 1.2MWh BESS Battery Storage (State: %.1f%%).", bessKW, req.BatteryBESSStatePct))
		}

		remainingToShave := targetShave - bessKW
		if remainingToShave > 0 {
			deferred = append(deferred, "Secondary Scrap Metal Crusher Line 2 (Deferred 3 Hours)")
			deferred = append(deferred, "WMS Automated Battery Charger Bank C (Throttled to 30%)")
			actions = append(actions, fmt.Sprintf("Deferred non-critical batch processes (shedding ~%.0f kW).", remainingToShave))
		}

		// Avoided peak penalty charge: Rp 88,000 per excess kW per month + peak tariff delta
		avoidedCostIDR = (excess * 88000.0) + (targetShave * 2.5 * 1444.0) // 2.5 hours peak delta
		energySavedKWh = targetShave * 2.5
	} else {
		actions = append(actions, "Load is comfortably within contract limit; standard grid baseload operating.")
	}

	order := PeakShavingDispatchOrder{
		DispatchID:               dispatchID,
		FacilityID:               req.FacilityID,
		GridTariffPeriod:         req.GridTariffPeriod,
		CurrentDemandKW:          req.CurrentTotalDemandKW,
		ContractDemandLimitKW:    req.ContractDemandLimitKW,
		OverLimitExcessKW:        math.Max(0, excess),
		ShavingTargetKW:          math.Max(0, excess+200.0),
		ActionsExecuted:          actions,
		BESSDischargeKW:          bessKW,
		DeferredBatchProcesses:   deferred,
		AvoidedPeakDemandCostIDR: math.Round(avoidedCostIDR),
		NetEnergySavedKWh:        math.Round(energySavedKWh),
		Status:                   status,
		ExecutedAt:               time.Now().Format(time.RFC3339),
	}

	globalEnergyStore.orders = append(globalEnergyStore.orders, order)
	return &order, nil
}

// GetISO50001AuditLogic delivers energy performance indicators and certification audit status.
func GetISO50001AuditLogic() (*ISO50001AuditReport, error) {
	globalEnergyStore.mu.RLock()
	defer globalEnergyStore.mu.RUnlock()
	return globalEnergyStore.auditSummary, nil
}

// RegisterEnergyOptimizerRoutes mounts the industrial energy optimization routes.
func RegisterEnergyOptimizerRoutes(r *gin.RouterGroup) {
	group := r.Group("/asset/energy")
	{
		group.POST("/telemetry-reading", func(c *gin.Context) {
			var reading SubMeterTelemetry
			if err := c.ShouldBindJSON(&reading); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid sub-meter telemetry payload: " + err.Error()})
				return
			}
			res, err := IngestTelemetryLogic(reading)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusCreated, res)
		})

		group.POST("/orchestrate-peak-shaving", func(c *gin.Context) {
			var req PeakShavingRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid peak shaving request payload: " + err.Error()})
				return
			}
			res, err := OrchestratePeakShavingLogic(req)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, res)
		})

		group.GET("/iso50001-audit", func(c *gin.Context) {
			res, err := GetISO50001AuditLogic()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, res)
		})
	}
}
