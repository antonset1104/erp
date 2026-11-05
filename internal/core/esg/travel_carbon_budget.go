package esg

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// TripSpecification represents the details of an employee business travel booking
type TripSpecification struct {
	TripID              string  `json:"trip_id"` // e.g. "TRIP-2026-LON-089"
	EmployeeID          string  `json:"employee_id"`
	DepartmentID        string  `json:"department_id"`    // e.g. "DEPT-SALES-APAC"
	OriginCity          string  `json:"origin_city"`      // e.g. "CGK" (Jakarta)
	DestinationCity     string  `json:"destination_city"` // e.g. "SIN" (Singapore)
	DistanceKM          float64 `json:"distance_km"`
	TravelMode          string  `json:"travel_mode"` // "FLIGHT", "HIGH_SPEED_RAIL", "ELECTRIC_VEHICLE"
	CabinClass          string  `json:"cabin_class"` // "ECONOMY", "PREMIUM_ECONOMY", "BUSINESS"
	HotelNights         int     `json:"hotel_nights"`
	HotelGreenCertified bool    `json:"hotel_green_certified"`
}

// TripEmissionsResult represents calculated Scope 3 Category 6 travel emissions
type TripEmissionsResult struct {
	TripID                   string    `json:"trip_id"`
	FlightEmissionsKgCO2e    float64   `json:"flight_emissions_kg_co2e"`
	HotelEmissionsKgCO2e     float64   `json:"hotel_emissions_kg_co2e"`
	TotalTripEmissionsKg     float64   `json:"total_trip_emissions_kg"`
	InternalCarbonFeeUSD     float64   `json:"internal_carbon_fee_usd"` // Shadow carbon price ($50/ton)
	HighSpeedRailFeasible    bool      `json:"high_speed_rail_feasible"`
	RailAlternativeSavingsKg float64   `json:"rail_alternative_savings_kg"`
	CalculatedAt             time.Time `json:"calculated_at"`
}

// PolicyAuditVerdict represents green corporate travel policy compliance
type PolicyAuditVerdict struct {
	TripID             string    `json:"trip_id"`
	PolicyCompliance   string    `json:"policy_compliance"` // "APPROVED_GREEN_TIER", "POLICY_EXCEPTION_REQUIRED", "VIOLATION_REJECTED"
	ViolationsFound    []string  `json:"violations_found"`
	RequiresVPApproval bool      `json:"requires_vp_approval"`
	RecommendedAction  string    `json:"recommended_action"`
	AuditedAt          time.Time `json:"audited_at"`
}

// DepartmentCarbonBudget represents quarterly Scope 3 carbon quota for a division
type DepartmentCarbonBudget struct {
	DepartmentID          string    `json:"department_id"`
	QuarterPeriod         string    `json:"quarter_period"` // "2026-Q1"
	AllocatedBudgetTons   float64   `json:"allocated_budget_tons"`
	ConsumedEmissionsTons float64   `json:"consumed_emissions_tons"`
	RemainingQuotaTons    float64   `json:"remaining_quota_tons"`
	BudgetBurnRatePct     float64   `json:"budget_burn_rate_pct"`
	InternalCarbonLevyUSD float64   `json:"internal_carbon_levy_usd"`
	DepartmentStanding    string    `json:"department_standing"` // "ON_TRACK_GREEN", "NEAR_CAP_WARNING", "BUDGET_EXCEEDED"
	LastUpdated           time.Time `json:"last_updated"`
}

// TravelCarbonEngine manages corporate travel emissions auditing and budget tracking
type TravelCarbonEngine struct {
	mu       sync.RWMutex
	trips    map[string]TripEmissionsResult
	budgets  map[string]DepartmentCarbonBudget
	verdicts map[string]PolicyAuditVerdict
}

var (
	globalTravelCarbonEngine *TravelCarbonEngine
	travelCarbonOnce         sync.Once
)

// GetTravelCarbonEngine returns singleton instance of TravelCarbonEngine
func GetTravelCarbonEngine() *TravelCarbonEngine {
	travelCarbonOnce.Do(func() {
		globalTravelCarbonEngine = &TravelCarbonEngine{
			trips:    make(map[string]TripEmissionsResult),
			budgets:  make(map[string]DepartmentCarbonBudget),
			verdicts: make(map[string]PolicyAuditVerdict),
		}
	})
	return globalTravelCarbonEngine
}

// CalculateTripEmissions computes GHG footprint using ICAO emission factors
func (e *TravelCarbonEngine) CalculateTripEmissions(spec TripSpecification) TripEmissionsResult {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Emission factor kg CO2e per km
	var factor float64 = 0.150 // Standard economy flight
	switch spec.TravelMode {
	case "HIGH_SPEED_RAIL":
		factor = 0.028 // 80% cleaner
	case "ELECTRIC_VEHICLE":
		factor = 0.045
	case "FLIGHT":
		if spec.CabinClass == "BUSINESS" {
			factor = 0.380 // 2.5x spatial multiplier for business class
		} else if spec.CabinClass == "PREMIUM_ECONOMY" {
			factor = 0.220
		}
	}

	flightCO2 := spec.DistanceKM * factor

	// Hotel emissions (standard ~24.5 kg CO2e/night vs green-certified ~12.0 kg)
	hotelFactor := 24.5
	if spec.HotelGreenCertified {
		hotelFactor = 12.0
	}
	hotelCO2 := float64(spec.HotelNights) * hotelFactor

	totalCO2 := flightCO2 + hotelCO2
	carbonPriceUSD := (totalCO2 / 1000.0) * 50.0 // $50/ton internal carbon price

	railFeasible := spec.DistanceKM <= 600.0 && spec.TravelMode == "FLIGHT"
	var railSavings float64
	if railFeasible {
		railSavings = (spec.DistanceKM * 0.150) - (spec.DistanceKM * 0.028)
	}

	res := TripEmissionsResult{
		TripID:                   spec.TripID,
		FlightEmissionsKgCO2e:    flightCO2,
		HotelEmissionsKgCO2e:     hotelCO2,
		TotalTripEmissionsKg:     totalCO2,
		InternalCarbonFeeUSD:     carbonPriceUSD,
		HighSpeedRailFeasible:    railFeasible,
		RailAlternativeSavingsKg: railSavings,
		CalculatedAt:             time.Now(),
	}

	e.trips[spec.TripID] = res
	return res
}

// AuditBookingPolicy enforces travel carbon rules and rail mandates
func (e *TravelCarbonEngine) AuditBookingPolicy(spec TripSpecification) PolicyAuditVerdict {
	e.mu.Lock()
	defer e.mu.Unlock()

	var violations []string
	requiresVP := false

	// Rule 1: High speed rail mandate under 500 km
	if spec.DistanceKM < 500.0 && spec.TravelMode == "FLIGHT" {
		violations = append(violations, "Short-haul flight (<500 km) violates green policy. High-speed rail or electric shuttle mandatory.")
		requiresVP = true
	}

	// Rule 2: Business class prohibition under 6 hours / 3000 km
	if spec.CabinClass == "BUSINESS" && spec.DistanceKM < 3000.0 {
		violations = append(violations, "Business class cabin prohibited for regional journeys under 3,000 km.")
		requiresVP = true
	}

	compliance := "APPROVED_GREEN_TIER"
	action := "Booking complies with corporate Scope 3 net-zero travel guidelines."
	if len(violations) > 0 {
		compliance = "POLICY_EXCEPTION_REQUIRED"
		action = "Booking flagged. Requires explicit ESG Director or Business Unit VP approval before ticketing."
	}

	verdict := PolicyAuditVerdict{
		TripID:             spec.TripID,
		PolicyCompliance:   compliance,
		ViolationsFound:    violations,
		RequiresVPApproval: requiresVP,
		RecommendedAction:  action,
		AuditedAt:          time.Now(),
	}

	e.verdicts[spec.TripID] = verdict
	return verdict
}

// DeductCarbonBudget charges trip footprint against department quota
func (e *TravelCarbonEngine) DeductCarbonBudget(deptID string, emissionsKg float64) DepartmentCarbonBudget {
	e.mu.Lock()
	defer e.mu.Unlock()

	budget, exists := e.budgets[deptID]
	if !exists {
		budget = DepartmentCarbonBudget{
			DepartmentID:          deptID,
			QuarterPeriod:         "2026-Q1",
			AllocatedBudgetTons:   45.0, // 45 tons CO2e quarterly allocation
			ConsumedEmissionsTons: 18.2,
		}
	}

	emissionsTons := emissionsKg / 1000.0
	budget.ConsumedEmissionsTons += emissionsTons
	budget.RemainingQuotaTons = budget.AllocatedBudgetTons - budget.ConsumedEmissionsTons
	budget.BudgetBurnRatePct = (budget.ConsumedEmissionsTons / budget.AllocatedBudgetTons) * 100.0
	budget.InternalCarbonLevyUSD = budget.ConsumedEmissionsTons * 50.0

	standing := "ON_TRACK_GREEN"
	if budget.BudgetBurnRatePct > 100.0 {
		standing = "BUDGET_EXCEEDED"
	} else if budget.BudgetBurnRatePct > 80.0 {
		standing = "NEAR_CAP_WARNING"
	}

	budget.DepartmentStanding = standing
	budget.LastUpdated = time.Now()

	e.budgets[deptID] = budget
	return budget
}

// GetDepartmentStatus returns quota tracking for department
func (e *TravelCarbonEngine) GetDepartmentStatus(deptID string) DepartmentCarbonBudget {
	e.mu.RLock()
	defer e.mu.RUnlock()

	budget, exists := e.budgets[deptID]
	if !exists {
		budget = DepartmentCarbonBudget{
			DepartmentID:          deptID,
			QuarterPeriod:         "2026-Q1",
			AllocatedBudgetTons:   45.0,
			ConsumedEmissionsTons: 19.5,
			RemainingQuotaTons:    25.5,
			BudgetBurnRatePct:     43.3,
			InternalCarbonLevyUSD: 975.0,
			DepartmentStanding:    "ON_TRACK_GREEN",
			LastUpdated:           time.Now(),
		}
	}
	return budget
}

// RegisterTravelCarbonRoutes registers REST API endpoints for Green Travel Carbon Budget
func RegisterTravelCarbonRoutes(r *gin.RouterGroup) {
	group := r.Group("/esg/travel")
	{
		group.POST("/calculate-trip-emissions", func(c *gin.Context) {
			var req TripSpecification
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res := GetTravelCarbonEngine().CalculateTripEmissions(req)
			c.JSON(http.StatusOK, gin.H{
				"status":    "EMISSIONS_CALCULATED",
				"emissions": res,
			})
		})

		group.POST("/audit-booking-policy", func(c *gin.Context) {
			var req TripSpecification
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res := GetTravelCarbonEngine().AuditBookingPolicy(req)
			c.JSON(http.StatusOK, gin.H{
				"status":  "AUDIT_COMPLETED",
				"verdict": res,
			})
		})

		group.POST("/deduct-carbon-budget", func(c *gin.Context) {
			var req struct {
				DepartmentID string  `json:"department_id" binding:"required"`
				EmissionsKg  float64 `json:"emissions_kg" binding:"required"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res := GetTravelCarbonEngine().DeductCarbonBudget(req.DepartmentID, req.EmissionsKg)
			c.JSON(http.StatusOK, gin.H{
				"status": "BUDGET_DEDUCTED",
				"budget": res,
			})
		})

		group.GET("/department-budget-status/:department_id", func(c *gin.Context) {
			deptID := c.Param("department_id")
			res := GetTravelCarbonEngine().GetDepartmentStatus(deptID)
			c.JSON(http.StatusOK, gin.H{
				"status": "STATUS_RESOLVED",
				"budget": res,
			})
		})
	}
}
