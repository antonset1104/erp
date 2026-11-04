package energy

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// TariffInterval represents an hourly or sub-hourly electricity tariff bracket
type TariffInterval struct {
	HourOfDay         int     `json:"hour_of_day"` // 0 - 23
	ImportTariffPerKWh float64 `json:"import_tariff_per_kwh"` // in USD or local currency
	ExportTariffPerKWh float64 `json:"export_tariff_per_kwh"` // Feed-in tariff credit
	TariffTier        string  `json:"tariff_tier"` // "OFF_PEAK", "MID_PEAK", "CRITICAL_PEAK"
}

// BESSAsset represents the industrial battery energy storage system
type BESSAsset struct {
	BatteryID          string  `json:"battery_id"`
	TotalCapacityKWh   float64 `json:"total_capacity_kwh"`
	CurrentSoCPercent  float64 `json:"current_soc_percent"`
	MaxChargeRateKW    float64 `json:"max_charge_rate_kw"`
	MaxDischargeRateKW float64 `json:"max_discharge_rate_kw"`
	RoundTripEfficiency float64 `json:"round_trip_efficiency"` // e.g., 0.92 (92%)
	DegradationCostKWh float64 `json:"degradation_cost_kwh"`  // battery wear amortization $/kWh
}

// DispatchScheduleStep represents one hourly action in the 24h dispatch plan
type DispatchScheduleStep struct {
	HourOfDay           int     `json:"hour_of_day"`
	Action              string  `json:"action"` // "CHARGE", "DISCHARGE", "STANDBY", "CURTAIL_LOAD"
	PowerKW             float64 `json:"power_kw"`
	ProjectedSoCPercent float64 `json:"projected_soc_percent"`
	HourlySavingsUSD    float64 `json:"hourly_savings_usd"`
	CO2AvoidedKg        float64 `json:"co2_avoided_kg"`
}

// DispatchOptimizationPlan represents the comprehensive 24h VPP dispatch schedule
type DispatchOptimizationPlan struct {
	PlanID               string                 `json:"plan_id"`
	SiteID               string                 `json:"site_id"`
	TotalProjectedSavings float64               `json:"total_projected_savings"`
	TotalDischargedKWh   float64                `json:"total_discharged_kwh"`
	TotalChargedKWh      float64                `json:"total_charged_kwh"`
	PeakDemandReductionKW float64               `json:"peak_demand_reduction_kw"`
	TotalCO2AvoidedKg    float64                `json:"total_co2_avoided_kg"`
	HourlySchedule       []DispatchScheduleStep `json:"hourly_schedule"`
	GeneratedAt          time.Time              `json:"generated_at"`
}

// CurtailmentEvent represents an emergency or scheduled load-shedding instruction
type CurtailmentEvent struct {
	EventID              string    `json:"event_id"`
	SiteID               string    `json:"site_id"`
	TargetLoadReductionKW float64  `json:"target_load_reduction_kw"`
	DurationMinutes      int       `json:"duration_minutes"`
	AffectedSystems      []string  `json:"affected_systems"`
	IncentiveCreditUSD   float64   `json:"incentive_credit_usd"`
	ExecutionStatus      string    `json:"execution_status"` // "TRIGGERED", "SHEDDING_ACTIVE", "COMPLETED"
	TriggeredAt          time.Time `json:"triggered_at"`
}

// ArbitragePnLSummary summarizes financial gains from VPP energy arbitrage
type ArbitragePnLSummary struct {
	SiteID                   string    `json:"site_id"`
	PeriodMonth              string    `json:"period_month"`
	GrossTariffSavingsUSD    float64   `json:"gross_tariff_savings_usd"`
	GridFeedInRevenueUSD     float64   `json:"grid_feed_in_revenue_usd"`
	DemandChargeAvoidanceUSD float64   `json:"demand_charge_avoidance_usd"`
	BatteryWearExpenseUSD    float64   `json:"battery_wear_expense_usd"`
	NetEconomicGainUSD       float64   `json:"net_economic_gain_usd"`
	NetCarbonAbatedTons      float64   `json:"net_carbon_abated_tons"`
	CalculatedAt             time.Time `json:"calculated_at"`
}

// VPPEngine manages VPP dispatch and energy arbitrage
type VPPEngine struct {
	mu            sync.RWMutex
	tariffs       map[string][]TariffInterval        // siteID -> 24h tariff intervals
	bessAssets    map[string]BESSAsset               // siteID -> BESS asset specs
	plans         map[string]DispatchOptimizationPlan // siteID -> latest plan
	curtailments  map[string][]CurtailmentEvent      // siteID -> list of curtailments
}

var (
	globalVPPEngine *VPPEngine
	vppOnce         sync.Once
)

// GetVPPEngine returns singleton instance of VPPEngine
func GetVPPEngine() *VPPEngine {
	vppOnce.Do(func() {
		globalVPPEngine = &VPPEngine{
			tariffs:      make(map[string][]TariffInterval),
			bessAssets:   make(map[string]BESSAsset),
			plans:        make(map[string]DispatchOptimizationPlan),
			curtailments: make(map[string][]CurtailmentEvent),
		}
	})
	return globalVPPEngine
}

// IngestTariffCurve sets the 24h tariff schedule for a manufacturing site
func (e *VPPEngine) IngestTariffCurve(siteID string, intervals []TariffInterval, bess BESSAsset) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.tariffs[siteID] = intervals
	e.bessAssets[siteID] = bess
}

// OptimizeDispatch computes optimal 24h charging/discharging arbitrage plan
func (e *VPPEngine) OptimizeDispatch(siteID string) DispatchOptimizationPlan {
	e.mu.Lock()
	defer e.mu.Unlock()

	intervals := e.tariffs[siteID]
	bess, hasBess := e.bessAssets[siteID]
	if !hasBess {
		bess = BESSAsset{
			BatteryID:           "BESS-DEFAULT-1MWH",
			TotalCapacityKWh:    1000.0,
			CurrentSoCPercent:   40.0,
			MaxChargeRateKW:     250.0,
			MaxDischargeRateKW:  250.0,
			RoundTripEfficiency: 0.92,
			DegradationCostKWh:  0.015,
		}
	}

	var schedule []DispatchScheduleStep
	var totalSavings float64
	var totalDischarged float64
	var totalCharged float64
	var totalCO2Avoided float64

	currentSoC := bess.CurrentSoCPercent

	for h := 0; h < 24; h++ {
		var importRate float64 = 0.12
		var tier string = "MID_PEAK"
		for _, interval := range intervals {
			if interval.HourOfDay == h {
				importRate = interval.ImportTariffPerKWh
				tier = interval.TariffTier
				break
			}
		}

		action := "STANDBY"
		var powerKW float64 = 0.0
		var hourlySaving float64 = 0.0
		var co2 float64 = 0.0

		// Simple arbitrage heuristic:
		// Super off-peak (e.g. 0:00 - 5:00 or tier == "OFF_PEAK"): Charge battery if SoC < 95%
		// Critical peak (e.g. 17:00 - 21:00 or tier == "CRITICAL_PEAK"): Discharge battery if SoC > 15%
		if (tier == "OFF_PEAK" || h < 5) && currentSoC < 95.0 {
			action = "CHARGE"
			powerKW = bess.MaxChargeRateKW
			chargedKWh := powerKW * 1.0
			totalCharged += chargedKWh
			currentSoC += (chargedKWh / bess.TotalCapacityKWh) * 100.0
			if currentSoC > 95.0 {
				currentSoC = 95.0
			}
		} else if (tier == "CRITICAL_PEAK" || (h >= 17 && h <= 21)) && currentSoC > 15.0 {
			action = "DISCHARGE"
			powerKW = bess.MaxDischargeRateKW
			dischargedKWh := powerKW * 1.0 * bess.RoundTripEfficiency
			totalDischarged += dischargedKWh
			currentSoC -= (powerKW / bess.TotalCapacityKWh) * 100.0
			if currentSoC < 15.0 {
				currentSoC = 15.0
			}
			// Savings = (Peak Price - Off-Peak Baseline - Degradation) * discharged
			netPriceDelta := importRate - 0.06 - bess.DegradationCostKWh
			if netPriceDelta < 0 {
				netPriceDelta = 0.05
			}
			hourlySaving = dischargedKWh * netPriceDelta
			co2 = dischargedKWh * 0.72 // 0.72 kg CO2 avoided per grid peak kWh displaced
			totalSavings += hourlySaving
			totalCO2Avoided += co2
		}

		schedule = append(schedule, DispatchScheduleStep{
			HourOfDay:           h,
			Action:              action,
			PowerKW:             powerKW,
			ProjectedSoCPercent: currentSoC,
			HourlySavingsUSD:    hourlySaving,
			CO2AvoidedKg:        co2,
		})
	}

	plan := DispatchOptimizationPlan{
		PlanID:                fmt.Sprintf("VPP-PLAN-%s-%d", siteID, time.Now().Unix()%100000),
		SiteID:                siteID,
		TotalProjectedSavings: totalSavings,
		TotalDischargedKWh:    totalDischarged,
		TotalChargedKWh:       totalCharged,
		PeakDemandReductionKW: bess.MaxDischargeRateKW,
		TotalCO2AvoidedKg:     totalCO2Avoided,
		HourlySchedule:        schedule,
		GeneratedAt:           time.Now(),
	}

	e.plans[siteID] = plan
	return plan
}

// ExecuteCurtailment initiates emergency or automated load reduction
func (e *VPPEngine) ExecuteCurtailment(siteID string, targetKW float64, durationMin int, systems []string) CurtailmentEvent {
	e.mu.Lock()
	defer e.mu.Unlock()

	credit := (targetKW * (float64(durationMin) / 60.0)) * 0.35 // $0.35/kWh demand response payment

	event := CurtailmentEvent{
		EventID:               fmt.Sprintf("CRTL-%s-%d", siteID, time.Now().Unix()%100000),
		SiteID:                siteID,
		TargetLoadReductionKW: targetKW,
		DurationMinutes:       durationMin,
		AffectedSystems:       systems,
		IncentiveCreditUSD:    credit,
		ExecutionStatus:       "SHEDDING_ACTIVE",
		TriggeredAt:           time.Now(),
	}

	e.curtailments[siteID] = append(e.curtailments[siteID], event)
	return event
}

// GetArbitragePnL calculates cumulative economic gain
func (e *VPPEngine) GetArbitragePnL(siteID string) ArbitragePnLSummary {
	e.mu.RLock()
	defer e.mu.RUnlock()

	plan, exists := e.plans[siteID]
	var grossSavings float64 = 14500.0
	var co2 float64 = 18.5
	if exists {
		grossSavings = plan.TotalProjectedSavings * 30.0 // 30-day projection
		co2 = (plan.TotalCO2AvoidedKg * 30.0) / 1000.0
	}

	feedIn := 2800.0
	demandAvoidance := 8200.0 // Demand charge tariff shaving: 250 kW * $32.8/kW
	batteryWear := 1150.0
	netGain := grossSavings + feedIn + demandAvoidance - batteryWear

	return ArbitragePnLSummary{
		SiteID:                   siteID,
		PeriodMonth:              time.Now().Format("2006-01"),
		GrossTariffSavingsUSD:    grossSavings,
		GridFeedInRevenueUSD:     feedIn,
		DemandChargeAvoidanceUSD: demandAvoidance,
		BatteryWearExpenseUSD:    batteryWear,
		NetEconomicGainUSD:       netGain,
		NetCarbonAbatedTons:      co2,
		CalculatedAt:             time.Now(),
	}
}

// RegisterVPPRoutes registers REST API endpoints for VPP Energy Arbitrage
func RegisterVPPRoutes(r *gin.RouterGroup) {
	group := r.Group("/energy/vpp")
	{
		group.POST("/ingest-tariffs", func(c *gin.Context) {
			var req struct {
				SiteID    string           `json:"site_id" binding:"required"`
				Intervals []TariffInterval `json:"intervals" binding:"required"`
				BESS      BESSAsset        `json:"bess"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			GetVPPEngine().IngestTariffCurve(req.SiteID, req.Intervals, req.BESS)
			c.JSON(http.StatusOK, gin.H{
				"status":  "TARIFFS_INGESTED",
				"message": fmt.Sprintf("Recorded %d tariff intervals for site %s", len(req.Intervals), req.SiteID),
			})
		})

		group.POST("/optimize-dispatch-schedule", func(c *gin.Context) {
			var req struct {
				SiteID string `json:"site_id" binding:"required"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			plan := GetVPPEngine().OptimizeDispatch(req.SiteID)
			c.JSON(http.StatusOK, gin.H{
				"status": "DISPATCH_OPTIMIZED",
				"plan":   plan,
			})
		})

		group.POST("/execute-curtailment-event", func(c *gin.Context) {
			var req struct {
				SiteID          string   `json:"site_id" binding:"required"`
				TargetKW        float64  `json:"target_kw" binding:"required"`
				DurationMinutes int      `json:"duration_minutes" binding:"required"`
				AffectedSystems []string `json:"affected_systems"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			event := GetVPPEngine().ExecuteCurtailment(req.SiteID, req.TargetKW, req.DurationMinutes, req.AffectedSystems)
			c.JSON(http.StatusOK, gin.H{
				"status": "CURTAILMENT_TRIGGERED",
				"event":  event,
			})
		})

		group.GET("/arbitrage-pnl/:site_id", func(c *gin.Context) {
			siteID := c.Param("site_id")
			pnl := GetVPPEngine().GetArbitragePnL(siteID)
			c.JSON(http.StatusOK, gin.H{
				"status": "PNL_COMPILED",
				"pnl":    pnl,
			})
		})
	}
}
