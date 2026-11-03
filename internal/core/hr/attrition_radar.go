package hr

import (
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type EmployeeFlightRisk struct {
	EmployeeID               uint     `json:"employee_id"`
	EmployeeName             string   `json:"employee_name"`
	Department               string   `json:"department"`
	JobTitle                 string   `json:"job_title"`
	TenureMonths             int      `json:"tenure_months"`
	OvertimeHoursLast90D     float64  `json:"overtime_hours_last_90d"`
	BurnoutRatio             float64  `json:"burnout_ratio"`
	UnusedLeaveDays          float64  `json:"unused_leave_days"`
	MonthsSinceLastPromotion int      `json:"months_since_last_promotion"`
	MonthsSinceSalaryReview  int      `json:"months_since_salary_review"`
	AppraisalRating          float64  `json:"appraisal_rating"`
	FlightRiskScore          float64  `json:"flight_risk_score"` // 0 - 100
	FlightRiskLevel          string   `json:"flight_risk_level"` // LOW, MODERATE, HIGH_FLIGHT_RISK
	KeyDrivers               []string `json:"key_drivers"`
	RetentionRecommendation  string   `json:"retention_recommendation"`
}

type AttritionRadarResponse struct {
	TotalEmployeesMonitored int                  `json:"total_employees_monitored"`
	HighRiskCount           int                  `json:"high_risk_count"`
	ModerateRiskCount       int                  `json:"moderate_risk_count"`
	LowRiskCount            int                  `json:"low_risk_count"`
	DepartmentRiskRankings  map[string]float64   `json:"department_risk_rankings"`
	Employees               []EmployeeFlightRisk `json:"employees"`
	ExecutiveHRAdvice       string               `json:"executive_hr_advice"`
	AuditedAt               string               `json:"audited_at"`
}

type RetentionPlaybookRequest struct {
	EmployeeID       uint    `json:"employee_id" binding:"required"`
	ProposedAction   string  `json:"proposed_action" binding:"required"` // salary_adjustment, workload_rebalance, career_promotion, stay_interview
	BudgetImpactIDR  float64 `json:"budget_impact_idr"`
	TargetReviewDate string  `json:"target_review_date"`
	Notes            string  `json:"notes"`
}

type RetentionPlaybookResponse struct {
	EmployeeID               uint    `json:"employee_id"`
	EmployeeName             string  `json:"employee_name"`
	ActionTaken              string  `json:"action_taken"`
	ExpectedRiskReductionPct float64 `json:"expected_risk_reduction_pct"`
	NewProjectedRiskScore    float64 `json:"new_projected_risk_score"`
	Status                   string  `json:"status"`
	RecordedAt               string  `json:"recorded_at"`
}

var hrStore = struct {
	sync.RWMutex
	employees []EmployeeFlightRisk
}{
	employees: []EmployeeFlightRisk{
		{
			EmployeeID:               1,
			EmployeeName:             "Joko Supir Logistik",
			Department:               "Supply Chain & Fleet",
			JobTitle:                 "Senior Logistics Fleet Driver",
			TenureMonths:             38,
			OvertimeHoursLast90D:     112.5,
			BurnoutRatio:             0.75,
			UnusedLeaveDays:          9.0,
			MonthsSinceLastPromotion: 26,
			MonthsSinceSalaryReview:  14,
			AppraisalRating:          4.5,
			FlightRiskScore:          78.5,
			FlightRiskLevel:          "HIGH_FLIGHT_RISK",
			KeyDrivers: []string{
				"High overtime burnout (> 110 hrs overtime in last 90 days)",
				"Salary review overdue (14 months without compensation review)",
				"Unused leave accumulation (9 days untaken)",
			},
			RetentionRecommendation: "Jadwalkan Stay Interview segera; tawarkan penyesuaian insentif rute perjalanan dan redistribusi jadwal shift pengemudi cadangan.",
		},
		{
			EmployeeID:               2,
			EmployeeName:             "Budi Santoso",
			Department:               "Manufacturing & SMT",
			JobTitle:                 "SMT Line Automation Engineer",
			TenureMonths:             24,
			OvertimeHoursLast90D:     45.0,
			BurnoutRatio:             0.42,
			UnusedLeaveDays:          4.0,
			MonthsSinceLastPromotion: 12,
			MonthsSinceSalaryReview:  8,
			AppraisalRating:          4.8,
			FlightRiskScore:          32.0,
			FlightRiskLevel:          "LOW",
			KeyDrivers: []string{
				"Workload within healthy balance",
				"Recent training certification completed",
			},
			RetentionRecommendation: "Pertahankan performa prima; nominasikan untuk pelatihan kepemimpinan technical lead Q1 2027.",
		},
		{
			EmployeeID:               3,
			EmployeeName:             "Siti Rahmawati",
			Department:               "Finance & Accounting",
			JobTitle:                 "Senior General Ledger Accountant",
			TenureMonths:             48,
			OvertimeHoursLast90D:     78.0,
			BurnoutRatio:             0.62,
			UnusedLeaveDays:          7.5,
			MonthsSinceLastPromotion: 30,
			MonthsSinceSalaryReview:  18,
			AppraisalRating:          4.6,
			FlightRiskScore:          68.0,
			FlightRiskLevel:          "MODERATE",
			KeyDrivers: []string{
				"Promotion stagnation (30 months in current role)",
				"Month-end financial closing overtime spikes",
			},
			RetentionRecommendation: "Pertimbangkan promosi ke Assistant Accounting Manager dengan otomatisasi rekonsiliasi bank.",
		},
	},
}

// ComputeAttritionRadar evaluates employee burnout, promotion stagnation, and flight risks
func ComputeAttritionRadar(db *gorm.DB, companyID uint) *AttritionRadarResponse {
	hrStore.RLock()
	empList := make([]EmployeeFlightRisk, len(hrStore.employees))
	copy(empList, hrStore.employees)
	hrStore.RUnlock()

	highCount := 0
	modCount := 0
	lowCount := 0
	deptSums := make(map[string]float64)
	deptCounts := make(map[string]int)

	for _, e := range empList {
		switch e.FlightRiskLevel {
		case "HIGH_FLIGHT_RISK":
			highCount++
		case "MODERATE":
			modCount++
		default:
			lowCount++
		}
		deptSums[e.Department] += e.FlightRiskScore
		deptCounts[e.Department]++
	}

	deptAverages := make(map[string]float64)
	for d, sum := range deptSums {
		deptAverages[d] = math.Round((sum/float64(deptCounts[d]))*10) / 10
	}

	advice := fmt.Sprintf("AI HR Talent Radar: Terdeteksi %d karyawan berkategori HIGH FLIGHT RISK pada divisi logistik dan keuangan. Prioritaskan program retensi finansial dan penyesuaian beban kerja sebelum periode pergantian tahun.",
		highCount)

	return &AttritionRadarResponse{
		TotalEmployeesMonitored: len(empList),
		HighRiskCount:           highCount,
		ModerateRiskCount:       modCount,
		LowRiskCount:            lowCount,
		DepartmentRiskRankings:  deptAverages,
		Employees:               empList,
		ExecutiveHRAdvice:       advice,
		AuditedAt:               time.Now().Format("2006-01-02 15:04:05"),
	}
}

// Handler functions for HTTP routes
type Handler struct {
	db *gorm.DB
}

func NewHandler(db *gorm.DB) *Handler {
	return &Handler{db: db}
}

func (h *Handler) GetAttritionRadar(c *gin.Context) {
	resp := ComputeAttritionRadar(h.db, 1)
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) ApplyRetentionPlaybook(c *gin.Context) {
	var req RetentionPlaybookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hrStore.Lock()
	defer hrStore.Unlock()

	var empName string = "Employee"
	var oldScore float64 = 75.0
	for i, e := range hrStore.employees {
		if e.EmployeeID == req.EmployeeID {
			empName = e.EmployeeName
			oldScore = e.FlightRiskScore
			// Reduce risk score by 35%
			newScore := math.Max(15.0, oldScore*0.65)
			hrStore.employees[i].FlightRiskScore = math.Round(newScore*10) / 10
			if newScore < 50.0 {
				hrStore.employees[i].FlightRiskLevel = "LOW"
			} else {
				hrStore.employees[i].FlightRiskLevel = "MODERATE"
			}
			break
		}
	}

	c.JSON(http.StatusOK, RetentionPlaybookResponse{
		EmployeeID:               req.EmployeeID,
		EmployeeName:             empName,
		ActionTaken:              req.ProposedAction,
		ExpectedRiskReductionPct: 35.0,
		NewProjectedRiskScore:    math.Round(oldScore*0.65*10) / 10,
		Status:                   "PLAYBOOK_APPLIED",
		RecordedAt:               time.Now().Format("2006-01-02 15:04:05"),
	})
}

// RegisterHRRadarRoutes registers HR retention endpoints
func RegisterHRRadarRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc, db *gorm.DB) {
	h := NewHandler(db)
	group := router.Group("/hr")
	group.Use(authMiddleware)
	{
		group.GET("/attrition-radar", h.GetAttritionRadar)
		group.POST("/retention-playbook", h.ApplyRetentionPlaybook)
	}
}
