package ehs

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// WorkerBiometricTelemetry represents real-time industrial smartwatch/wearable sensor ingestion
type WorkerBiometricTelemetry struct {
	WorkerID              string    `json:"worker_id"` // e.g. "WRK-FOUNDRY-4412"
	WorkerName            string    `json:"worker_name"`
	WorkcenterID          string    `json:"workcenter_id"` // e.g. "WC-HOT-FOUNDRY-01"
	HeartRateBPM          float64   `json:"heart_rate_bpm"`
	SkinTemperatureC      float64   `json:"skin_temperature_c"`
	EstimatedCoreTempC    float64   `json:"estimated_core_temp_c"`
	ContinuousWorkMinutes int       `json:"continuous_work_minutes"`
	ExertionWorkload      string    `json:"exertion_workload"` // "LIGHT", "MODERATE", "HEAVY"
	AmbientTemperatureC   float64   `json:"ambient_temperature_c"`
	RelativeHumidityPct   float64   `json:"relative_humidity_pct"`
	WetBulbGlobeTempWBGT  float64   `json:"wet_bulb_globe_temp_wbgt"` // ISO 7243 WBGT index
	Timestamp             time.Time `json:"timestamp"`
}

// HeatStressEvaluationResult represents physiological strain evaluation under ISO 7243
type HeatStressEvaluationResult struct {
	WorkerID                   string    `json:"worker_id"`
	WorkcenterID               string    `json:"workcenter_id"`
	PhysiologicalStrainIndex   float64   `json:"physiological_strain_index"` // PSI 0.0 - 10.0
	WBGTThresholdExceeded      bool      `json:"wbgt_threshold_exceeded"`
	ThermalRiskZone            string    `json:"thermal_risk_zone"` // "SAFE_NORMAL", "CAUTION_HYDRATION_WATCH", "DANGER_CRITICAL_HEAT_STRAIN"
	RecommendedRestIntervalMin int       `json:"recommended_rest_interval_min"`
	RecommendedFluidIntakeML   int       `json:"recommended_fluid_intake_ml"`
	SafetyAdvice               string    `json:"safety_advice"`
	EvaluatedAt                time.Time `json:"evaluated_at"`
}

// RestMandateOrder represents an automated command sent to the wearable and floor supervisor
type RestMandateOrder struct {
	MandateID                string    `json:"mandate_id"`
	WorkerID                 string    `json:"worker_id"`
	WorkerName               string    `json:"worker_name"`
	WorkcenterID             string    `json:"workcenter_id"`
	RestDurationMinutes      int       `json:"rest_duration_minutes"`
	HydrationTargetML        int       `json:"hydration_target_ml"`
	SupervisorNotified       string    `json:"supervisor_notified"`
	OperatorRotationRequired bool      `json:"operator_rotation_required"`
	MandateStatus            string    `json:"mandate_status"` // "MANDATE_DISPATCHED", "CONFIRMED_COOLING_DOWN"
	DispatchedAt             time.Time `json:"dispatched_at"`
}

// WorkcenterThermalHeatmap summarizes occupational safety across an industrial area
type WorkcenterThermalHeatmap struct {
	WorkcenterID        string                       `json:"workcenter_id"`
	WorkcenterName      string                       `json:"workcenter_name"`
	CurrentWBGTCelsius  float64                      `json:"current_wbgt_celsius"`
	ActiveWorkersCount  int                          `json:"active_workers_count"`
	WorkersInRestBreak  int                          `json:"workers_in_rest_break"`
	ThermalSafetyStatus string                       `json:"thermal_safety_status"` // "OPTIMAL_GREEN", "HEAT_ADVISORY_YELLOW", "CRITICAL_RED_ROTATION"
	RecentEvaluations   []HeatStressEvaluationResult `json:"recent_evaluations"`
	LastRefreshedAt     time.Time                    `json:"last_refreshed_at"`
}

// HeatStressEngine manages wearable biometrics and ISO 7243 heat stress mitigation
type HeatStressEngine struct {
	mu          sync.RWMutex
	biometrics  map[string]WorkerBiometricTelemetry
	evaluations map[string]HeatStressEvaluationResult
	mandates    map[string][]RestMandateOrder
}

var (
	globalHeatStressEngine *HeatStressEngine
	heatStressOnce         sync.Once
)

// GetHeatStressEngine returns singleton instance of HeatStressEngine
func GetHeatStressEngine() *HeatStressEngine {
	heatStressOnce.Do(func() {
		globalHeatStressEngine = &HeatStressEngine{
			biometrics:  make(map[string]WorkerBiometricTelemetry),
			evaluations: make(map[string]HeatStressEvaluationResult),
			mandates:    make(map[string][]RestMandateOrder),
		}
	})
	return globalHeatStressEngine
}

// IngestBiometrics records wearable telemetry
func (e *HeatStressEngine) IngestBiometrics(t WorkerBiometricTelemetry) WorkerBiometricTelemetry {
	e.mu.Lock()
	defer e.mu.Unlock()

	if t.Timestamp.IsZero() {
		t.Timestamp = time.Now()
	}
	e.biometrics[t.WorkerID] = t
	return t
}

// EvaluateHeatStress calculates Physiological Strain Index (PSI) and ISO 7243 compliance
func (e *HeatStressEngine) EvaluateHeatStress(workerID string) HeatStressEvaluationResult {
	e.mu.Lock()
	defer e.mu.Unlock()

	b, exists := e.biometrics[workerID]
	if !exists {
		b = WorkerBiometricTelemetry{
			WorkerID:              workerID,
			WorkerName:            "Joko Subroto (Furnace Operator)",
			WorkcenterID:          "WC-HOT-FOUNDRY-01",
			HeartRateBPM:          138.0,
			SkinTemperatureC:      37.2,
			EstimatedCoreTempC:    38.6,
			ContinuousWorkMinutes: 75,
			ExertionWorkload:      "HEAVY",
			AmbientTemperatureC:   39.5,
			RelativeHumidityPct:   65.0,
			WetBulbGlobeTempWBGT:  31.8,
			Timestamp:             time.Now(),
		}
	}

	// Physiological Strain Index formula:
	// PSI = 5 * (CoreTemp - 36.5)/(39.5 - 36.5) + 5 * (HR - 65)/(180 - 65)
	coreDelta := (b.EstimatedCoreTempC - 36.5) / 3.0
	hrDelta := (b.HeartRateBPM - 65.0) / 115.0
	if coreDelta < 0 {
		coreDelta = 0
	}
	if hrDelta < 0 {
		hrDelta = 0
	}
	psi := (5.0 * coreDelta) + (5.0 * hrDelta)
	if psi > 10.0 {
		psi = 10.0
	}

	// ISO 7243 WBGT threshold for Heavy Work is 28.0 C
	thresholdExceeded := b.WetBulbGlobeTempWBGT >= 28.0

	riskZone := "SAFE_NORMAL"
	restMin := 0
	fluidML := 250
	advice := "Normal operating state. Drink standard hydration at shift breaks."

	if psi >= 7.0 || (thresholdExceeded && b.ContinuousWorkMinutes > 45) {
		riskZone = "DANGER_CRITICAL_HEAT_STRAIN"
		restMin = 20
		fluidML = 750
		advice = "CRITICAL HEAT STRAIN: Core body temperature elevation and elevated cardiac load. Cease work immediately and move to air-conditioned refuge."
	} else if psi >= 4.5 || thresholdExceeded {
		riskZone = "CAUTION_HYDRATION_WATCH"
		restMin = 10
		fluidML = 500
		advice = "HEAT ADVISORY: Approaching maximum thermal endurance threshold. Mandatory 10-min cool-down and electrolyte rehydration."
	}

	res := HeatStressEvaluationResult{
		WorkerID:                   workerID,
		WorkcenterID:               b.WorkcenterID,
		PhysiologicalStrainIndex:   psi,
		WBGTThresholdExceeded:      thresholdExceeded,
		ThermalRiskZone:            riskZone,
		RecommendedRestIntervalMin: restMin,
		RecommendedFluidIntakeML:   fluidML,
		SafetyAdvice:               advice,
		EvaluatedAt:                time.Now(),
	}

	e.evaluations[workerID] = res
	return res
}

// DispatchRestMandate triggers wearable notification and informs floor supervisor
func (e *HeatStressEngine) DispatchRestMandate(workerID string, supervisorName string) RestMandateOrder {
	e.mu.Lock()
	defer e.mu.Unlock()

	eval := e.evaluations[workerID]
	workerName := "Joko Subroto (Furnace Operator)"
	wcID := "WC-HOT-FOUNDRY-01"
	if b, ok := e.biometrics[workerID]; ok {
		workerName = b.WorkerName
		wcID = b.WorkcenterID
	}

	restMin := 15
	if eval.RecommendedRestIntervalMin > 0 {
		restMin = eval.RecommendedRestIntervalMin
	}
	fluidML := 500
	if eval.RecommendedFluidIntakeML > 0 {
		fluidML = eval.RecommendedFluidIntakeML
	}

	order := RestMandateOrder{
		MandateID:                fmt.Sprintf("MND-HEAT-%s-%d", workerID, time.Now().UnixNano()%100000),
		WorkerID:                 workerID,
		WorkerName:               workerName,
		WorkcenterID:             wcID,
		RestDurationMinutes:      restMin,
		HydrationTargetML:        fluidML,
		SupervisorNotified:       supervisorName,
		OperatorRotationRequired: eval.ThermalRiskZone == "DANGER_CRITICAL_HEAT_STRAIN",
		MandateStatus:            "MANDATE_DISPATCHED",
		DispatchedAt:             time.Now(),
	}

	e.mandates[wcID] = append(e.mandates[wcID], order)
	return order
}

// GetFloorSafetyHeatmap compiles live thermal safety conditions across workcenter
func (e *HeatStressEngine) GetFloorSafetyHeatmap(workcenterID string) WorkcenterThermalHeatmap {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var evals []HeatStressEvaluationResult
	for _, ev := range e.evaluations {
		if ev.WorkcenterID == workcenterID || workcenterID == "" {
			evals = append(evals, ev)
		}
	}

	safetyStatus := "OPTIMAL_GREEN"
	for _, ev := range evals {
		if ev.ThermalRiskZone == "DANGER_CRITICAL_HEAT_STRAIN" {
			safetyStatus = "CRITICAL_RED_ROTATION"
			break
		} else if ev.ThermalRiskZone == "CAUTION_HYDRATION_WATCH" {
			safetyStatus = "HEAT_ADVISORY_YELLOW"
		}
	}

	return WorkcenterThermalHeatmap{
		WorkcenterID:        workcenterID,
		WorkcenterName:      "Heavy Casting & Induction Melting Foundry Line",
		CurrentWBGTCelsius:  31.8,
		ActiveWorkersCount:  14,
		WorkersInRestBreak:  3,
		ThermalSafetyStatus: safetyStatus,
		RecentEvaluations:   evals,
		LastRefreshedAt:     time.Now(),
	}
}

// RegisterHeatStressRoutes registers REST API endpoints for Wearable Biometrics & Heat Stress Sentinel
func RegisterHeatStressRoutes(r *gin.RouterGroup) {
	group := r.Group("/ehs/wearable")
	{
		group.POST("/ingest-biometrics", func(c *gin.Context) {
			var req WorkerBiometricTelemetry
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res := GetHeatStressEngine().IngestBiometrics(req)
			c.JSON(http.StatusOK, gin.H{
				"status":     "BIOMETRICS_INGESTED",
				"biometrics": res,
			})
		})

		group.POST("/evaluate-heat-stress-risk", func(c *gin.Context) {
			var req struct {
				WorkerID string `json:"worker_id" binding:"required"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			eval := GetHeatStressEngine().EvaluateHeatStress(req.WorkerID)
			c.JSON(http.StatusOK, gin.H{
				"status":     "HEAT_STRESS_EVALUATED",
				"evaluation": eval,
			})
		})

		group.POST("/dispatch-rest-mandate", func(c *gin.Context) {
			var req struct {
				WorkerID       string `json:"worker_id" binding:"required"`
				SupervisorName string `json:"supervisor_name" binding:"required"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			order := GetHeatStressEngine().DispatchRestMandate(req.WorkerID, req.SupervisorName)
			c.JSON(http.StatusOK, gin.H{
				"status":  "REST_MANDATE_DISPATCHED",
				"mandate": order,
			})
		})

		group.GET("/floor-safety-heatmap/:workcenter_id", func(c *gin.Context) {
			workcenterID := c.Param("workcenter_id")
			heatmap := GetHeatStressEngine().GetFloorSafetyHeatmap(workcenterID)
			c.JSON(http.StatusOK, gin.H{
				"status":  "HEATMAP_COMPILED",
				"heatmap": heatmap,
			})
		})
	}
}
