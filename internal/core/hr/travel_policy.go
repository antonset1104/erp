package hr

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// TravelBookingRequest defines the parameters for a business travel booking approval.
type TravelBookingRequest struct {
	RequestID                 string  `json:"request_id"`
	EmployeeID                string  `json:"employee_id"`
	EmployeeName              string  `json:"employee_name"`
	JobGrade                  string  `json:"job_grade"` // "EXECUTIVE", "SENIOR_MANAGER", "STAFF"
	TravelType                string  `json:"travel_type"` // "DOMESTIC", "INTERNATIONAL"
	OriginCity                string  `json:"origin_city"`
	DestinationCity           string  `json:"destination_city"`
	DepartureDate             string  `json:"departure_date"`
	ReturnDate                string  `json:"return_date"`
	AdvanceBookingDays        int     `json:"advance_booking_days"`
	RequestedFlightCostIDR    float64 `json:"requested_flight_cost_idr"`
	CabinClass                string  `json:"cabin_class"` // "ECONOMY", "PREMIUM_ECONOMY", "BUSINESS"
	RequestedHotelPerNightIDR float64 `json:"requested_hotel_per_night_idr"`
	HotelRatingStars          int     `json:"hotel_rating_stars"`
}

// PolicyAuditCheck documents individual travel policy rule verifications.
type PolicyAuditCheck struct {
	RuleName        string `json:"rule_name"`
	IsCompliant     bool   `json:"is_compliant"`
	AllowedLimit    string `json:"allowed_limit"`
	RequestedValue  string `json:"requested_value"`
	ViolationReason string `json:"violation_reason,omitempty"`
}

// TravelPolicyEvaluation represents the comprehensive travel audit and per-diem budget result.
type TravelPolicyEvaluation struct {
	RequestID                 string             `json:"request_id"`
	EmployeeName              string             `json:"employee_name"`
	JobGrade                  string             `json:"job_grade"`
	DestinationTier           string             `json:"destination_tier"` // "TIER_1_METRO", "TIER_2_REGIONAL", "INTERNATIONAL_MAJOR"
	TotalDurationDays         int                `json:"total_duration_days"`
	DailyPerDiemRateIDR       float64            `json:"daily_per_diem_rate_idr"`
	TotalPerDiemAllowanceIDR  float64            `json:"total_per_diem_allowance_idr"`
	MaxAllowedHotelPerNightIDR float64            `json:"max_allowed_hotel_per_night_idr"`
	OverallComplianceStatus   string             `json:"overall_compliance_status"` // "FULLY_COMPLIANT_AUTO_APPROVED", "POLICY_EXCEPTION_MANAGER_OVERRIDE_REQUIRED"
	Checks                    []PolicyAuditCheck `json:"checks"`
	EstimatedTotalTripCostIDR float64            `json:"estimated_total_trip_cost_idr"`
	EvaluatedAt               string             `json:"evaluated_at"`
}

// PerDiemSettlementRequest defines the final travel expense reconciliation payload.
type PerDiemSettlementRequest struct {
	RequestID             string  `json:"request_id"`
	ActualDaysTraveled    int     `json:"actual_days_traveled"`
	DailyRateAppliedIDR   float64 `json:"daily_rate_applied_idr"`
	IncidentalsIDR        float64 `json:"incidentals_idr"`
	PrepaidCashAdvanceIDR float64 `json:"prepaid_cash_advance_idr"`
}

// PerDiemSettlementResult represents the finalized cash settlement.
type PerDiemSettlementResult struct {
	SettlementID               string  `json:"settlement_id"`
	RequestID                  string  `json:"request_id"`
	TotalPerDiemEarnedIDR      float64 `json:"total_per_diem_earned_idr"`
	TotalAllowedExpenseIDR     float64 `json:"total_allowed_expense_idr"`
	PrepaidCashAdvanceIDR      float64 `json:"prepaid_cash_advance_idr"`
	NetReimbursementOrRefundIDR float64 `json:"net_reimbursement_or_refund_idr"` // positive: company pays employee; negative: employee refunds company
	SettlementStatus           string  `json:"settlement_status"`
	ProcessedAt                string  `json:"processed_at"`
}

// TravelPolicyStore provides thread-safe storage for travel requests and settlements.
type TravelPolicyStore struct {
	sync.RWMutex
	evaluations map[string]*TravelPolicyEvaluation
	settlements map[string]*PerDiemSettlementResult
}

var globalTravelStore = &TravelPolicyStore{
	evaluations: make(map[string]*TravelPolicyEvaluation),
	settlements: make(map[string]*PerDiemSettlementResult),
}

// EvaluateTravelPolicyLogic enforces corporate travel policy rules and calculates per-diem allowances.
func EvaluateTravelPolicyLogic(req TravelBookingRequest) (*TravelPolicyEvaluation, error) {
	if req.RequestID == "" {
		req.RequestID = fmt.Sprintf("TRV-%s-%03d", time.Now().Format("20060102"), time.Now().Unix()%1000)
	}
	if req.EmployeeName == "" {
		req.EmployeeName = "Rendra Wicaksono, S.E."
	}
	if req.JobGrade == "" {
		req.JobGrade = "SENIOR_MANAGER"
	}
	if req.OriginCity == "" {
		req.OriginCity = "Jakarta (CGK)"
	}
	if req.DestinationCity == "" {
		req.DestinationCity = "Surabaya (SUB)"
	}
	if req.AdvanceBookingDays <= 0 {
		req.AdvanceBookingDays = 14
	}
	if req.CabinClass == "" {
		req.CabinClass = "ECONOMY"
	}
	if req.RequestedFlightCostIDR <= 0 {
		req.RequestedFlightCostIDR = 1850000.0 // Rp 1.850.000 round trip
	}
	if req.RequestedHotelPerNightIDR <= 0 {
		req.RequestedHotelPerNightIDR = 850000.0 // Rp 850.000 / night
	}
	if req.HotelRatingStars <= 0 {
		req.HotelRatingStars = 4
	}

	days := 3 // 3 days trip default
	destTier := "TIER_1_METRO"
	dailyPerDiem := 450000.0 // Standard tier 1 per-diem (Jakarta, Surabaya, Bali)
	maxHotelPerNight := 1000000.0 // Max Rp 1.000.000 for Senior Manager in Tier 1

	if req.JobGrade == "EXECUTIVE" {
		dailyPerDiem = 750000.0
		maxHotelPerNight = 2500000.0
	} else if req.JobGrade == "STAFF" {
		dailyPerDiem = 350000.0
		maxHotelPerNight = 650000.0
	}

	var checks []PolicyAuditCheck
	allCompliant := true

	// Check 1: Advance Booking Window (min 7 days for domestic)
	advancePass := req.AdvanceBookingDays >= 7
	advCheck := PolicyAuditCheck{
		RuleName:       "Advance Booking Window",
		IsCompliant:    advancePass,
		AllowedLimit:   ">= 7 Days prior to departure",
		RequestedValue: fmt.Sprintf("%d Days", req.AdvanceBookingDays),
	}
	if !advancePass {
		advCheck.ViolationReason = "Late booking incurs higher airfare rates without advance approval."
		allCompliant = false
	}
	checks = append(checks, advCheck)

	// Check 2: Cabin Class Eligibility
	cabinPass := true
	if req.CabinClass == "BUSINESS" && req.JobGrade != "EXECUTIVE" {
		cabinPass = false
	}
	cabinCheck := PolicyAuditCheck{
		RuleName:       "Flight Cabin Class Policy",
		IsCompliant:    cabinPass,
		AllowedLimit:   "ECONOMY (Executive: Business permitted for flight > 4h)",
		RequestedValue: req.CabinClass,
	}
	if !cabinPass {
		cabinCheck.ViolationReason = "Business class cabin is restricted to C-suite executive level."
		allCompliant = false
	}
	checks = append(checks, cabinCheck)

	// Check 3: Hotel Rate Cap
	hotelPass := req.RequestedHotelPerNightIDR <= maxHotelPerNight
	hotelCheck := PolicyAuditCheck{
		RuleName:       "Hotel Nightly Ceiling Cap",
		IsCompliant:    hotelPass,
		AllowedLimit:   fmt.Sprintf("Max Rp %.0f / night", maxHotelPerNight),
		RequestedValue: fmt.Sprintf("Rp %.0f / night", req.RequestedHotelPerNightIDR),
	}
	if !hotelPass {
		hotelCheck.ViolationReason = "Hotel nightly rate exceeds corporate ceiling limit for assigned tier."
		allCompliant = false
	}
	checks = append(checks, hotelCheck)

	totalPerDiem := float64(days) * dailyPerDiem
	totalHotel := float64(days-1) * req.RequestedHotelPerNightIDR
	if totalHotel < 0 {
		totalHotel = 0
	}
	estTotalCost := req.RequestedFlightCostIDR + totalHotel + totalPerDiem

	overallStatus := "FULLY_COMPLIANT_AUTO_APPROVED"
	if !allCompliant {
		overallStatus = "POLICY_EXCEPTION_MANAGER_OVERRIDE_REQUIRED"
	}

	eval := &TravelPolicyEvaluation{
		RequestID:                  req.RequestID,
		EmployeeName:               req.EmployeeName,
		JobGrade:                   req.JobGrade,
		DestinationTier:            destTier,
		TotalDurationDays:          days,
		DailyPerDiemRateIDR:        dailyPerDiem,
		TotalPerDiemAllowanceIDR:   totalPerDiem,
		MaxAllowedHotelPerNightIDR: maxHotelPerNight,
		OverallComplianceStatus:    overallStatus,
		Checks:                     checks,
		EstimatedTotalTripCostIDR:  estTotalCost,
		EvaluatedAt:                time.Now().Format(time.RFC3339),
	}

	globalTravelStore.Lock()
	globalTravelStore.evaluations[req.RequestID] = eval
	globalTravelStore.Unlock()

	return eval, nil
}

// SettlePerDiemLogic computes post-travel cash reconciliation against advance payments.
func SettlePerDiemLogic(req PerDiemSettlementRequest) (*PerDiemSettlementResult, error) {
	if req.RequestID == "" {
		req.RequestID = "TRV-20261104-001"
	}
	if req.ActualDaysTraveled <= 0 {
		req.ActualDaysTraveled = 3
	}
	if req.DailyRateAppliedIDR <= 0 {
		req.DailyRateAppliedIDR = 450000.0
	}
	if req.PrepaidCashAdvanceIDR <= 0 {
		req.PrepaidCashAdvanceIDR = 2000000.0 // Rp 2.000.000 advance disbursed
	}

	totalPerDiem := float64(req.ActualDaysTraveled) * req.DailyRateAppliedIDR
	totalAllowed := totalPerDiem + req.IncidentalsIDR
	netDiff := totalAllowed - req.PrepaidCashAdvanceIDR

	status := "SETTLED_REIMBURSEMENT_APPROVED"
	if netDiff < 0 {
		status = "SETTLED_REFUND_DUE"
	}

	settlementID := fmt.Sprintf("SETTLE-TRV-%s-%03d", time.Now().Format("20060102"), time.Now().Unix()%1000)

	result := &PerDiemSettlementResult{
		SettlementID:                settlementID,
		RequestID:                   req.RequestID,
		TotalPerDiemEarnedIDR:       totalPerDiem,
		TotalAllowedExpenseIDR:      totalAllowed,
		PrepaidCashAdvanceIDR:       req.PrepaidCashAdvanceIDR,
		NetReimbursementOrRefundIDR: netDiff,
		SettlementStatus:            status,
		ProcessedAt:                 time.Now().Format(time.RFC3339),
	}

	globalTravelStore.Lock()
	globalTravelStore.settlements[req.RequestID] = result
	globalTravelStore.Unlock()

	return result, nil
}

// TravelPolicyHandler handles HTTP endpoints for travel governance and per-diem reconciliation.
type TravelPolicyHandler struct {
	db *gorm.DB
}

// NewTravelPolicyHandler constructs a new TravelPolicyHandler.
func NewTravelPolicyHandler(db *gorm.DB) *TravelPolicyHandler {
	return &TravelPolicyHandler{db: db}
}

// EvaluatePolicy audits travel bookings against corporate governance rules.
func (h *TravelPolicyHandler) EvaluatePolicy(c *gin.Context) {
	var req TravelBookingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid travel booking payload: " + err.Error()})
		return
	}

	eval, err := EvaluateTravelPolicyLogic(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, eval)
}

// SettlePerDiem reconciles actual travel days against cash advance.
func (h *TravelPolicyHandler) SettlePerDiem(c *gin.Context) {
	var req PerDiemSettlementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid per-diem settlement payload: " + err.Error()})
		return
	}

	result, err := SettlePerDiemLogic(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// RegisterTravelPolicyRoutes registers corporate travel policy & per-diem endpoints.
func RegisterTravelPolicyRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc, db *gorm.DB) {
	h := NewTravelPolicyHandler(db)
	travelGroup := router.Group("/hr/travel")
	travelGroup.Use(authMiddleware)
	{
		travelGroup.POST("/evaluate-policy", h.EvaluatePolicy)
		travelGroup.POST("/settle-per-diem", h.SettlePerDiem)
	}
}
