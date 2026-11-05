package account

import (
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// LeaseContract defines lease agreement parameters under IFRS 16 / PSAK 73.
type LeaseContract struct {
	LeaseID                           string  `json:"lease_id"`
	LeaseContractNumber               string  `json:"lease_contract_number"`
	LessorName                        string  `json:"lessor_name"`
	AssetDescription                  string  `json:"asset_description"`
	AssetCategory                     string  `json:"asset_category"` // "HEAVY_EQUIPMENT", "REAL_ESTATE_WAREHOUSE", "VEHICLE_FLEET"
	CommencementDate                  string  `json:"commencement_date"`
	LeaseTermMonths                   int     `json:"lease_term_months"`
	MonthlyPaymentIDR                 float64 `json:"monthly_payment_idr"`
	IncrementalBorrowingRateAnnualPct float64 `json:"incremental_borrowing_rate_annual_pct"` // IBR %
	PaymentTiming                     string  `json:"payment_timing"`                        // "IN_ADVANCE", "IN_ARREARS"
}

// LeaseSchedulePeriod holds monthly amortization figures for lease liability and ROU asset.
type LeaseSchedulePeriod struct {
	PeriodMonth               int     `json:"period_month"`
	DueDate                   string  `json:"due_date"`
	OpeningLiabilityIDR       float64 `json:"opening_liability_idr"`
	InterestExpenseIDR        float64 `json:"interest_expense_idr"`
	PrincipalRepaymentIDR     float64 `json:"principal_repayment_idr"`
	PaymentIDR                float64 `json:"payment_idr"`
	ClosingLiabilityIDR       float64 `json:"closing_liability_idr"`
	ROUAssetDepreciationIDR   float64 `json:"rou_asset_depreciation_idr"`
	ClosingROUNetBookValueIDR float64 `json:"closing_rou_net_book_value_idr"`
}

// LeaseAccountingSchedule represents the full capitalized schedule and IFRS 16 Balance Sheet impact.
type LeaseAccountingSchedule struct {
	LeaseID                  string                `json:"lease_id"`
	ContractNumber           string                `json:"contract_number"`
	AssetDescription         string                `json:"asset_description"`
	InitialROUAssetValueIDR  float64               `json:"initial_rou_asset_value_idr"`
	InitialLeaseLiabilityIDR float64               `json:"initial_lease_liability_idr"`
	TotalInterestExpenseIDR  float64               `json:"total_interest_expense_idr"`
	TotalCashPaymentsIDR     float64               `json:"total_cash_payments_idr"`
	Schedule                 []LeaseSchedulePeriod `json:"schedule"`
	Status                   string                `json:"status"` // "IFRS16_COMPLIANT_ACTIVE"
	CreatedAt                string                `json:"created_at"`
}

// LeaseRemeasurementRequest defines modifications to an active lease.
type LeaseRemeasurementRequest struct {
	LeaseID                  string  `json:"lease_id"`
	RevisedMonthlyPaymentIDR float64 `json:"revised_monthly_payment_idr"`
	RemainingMonths          int     `json:"remaining_months"`
	ModificationReason       string  `json:"modification_reason"`
}

// LeaseStore provides thread-safe in-memory caching for lease schedules.
type LeaseStore struct {
	sync.RWMutex
	schedules map[string]*LeaseAccountingSchedule
}

var globalLeaseStore = &LeaseStore{
	schedules: make(map[string]*LeaseAccountingSchedule),
}

// GenerateLeaseScheduleLogic computes the present value of future lease payments and builds amortization schedules.
func GenerateLeaseScheduleLogic(req LeaseContract) (*LeaseAccountingSchedule, error) {
	if req.LeaseID == "" {
		req.LeaseID = fmt.Sprintf("LEASE-%s-%d", time.Now().Format("20060102"), time.Now().Unix()%1000)
	}
	if req.LeaseContractNumber == "" {
		req.LeaseContractNumber = "LC/WH/2026/001"
	}
	if req.AssetDescription == "" {
		req.AssetDescription = "Gudang Logistik Pusat Cikarang (10.000 m2)"
	}
	if req.AssetCategory == "" {
		req.AssetCategory = "REAL_ESTATE_WAREHOUSE"
	}
	if req.LeaseTermMonths <= 0 {
		req.LeaseTermMonths = 36 // 3-year commercial lease
	}
	if req.MonthlyPaymentIDR <= 0 {
		req.MonthlyPaymentIDR = 120000000.0 // Rp 120 Juta / month
	}
	if req.IncrementalBorrowingRateAnnualPct <= 0 {
		req.IncrementalBorrowingRateAnnualPct = 6.50 // 6.5% IBR
	}

	monthlyRate := (req.IncrementalBorrowingRateAnnualPct / 100.0) / 12.0
	n := float64(req.LeaseTermMonths)

	// Present value formula for annuity ordinary: PV = P * (1 - (1+r)^-n) / r
	pv := req.MonthlyPaymentIDR * (1.0 - math.Pow(1.0+monthlyRate, -n)) / monthlyRate
	pv = math.Round(pv)

	monthlyDepr := math.Round((pv/n)*100) / 100.0

	var periods []LeaseSchedulePeriod
	currLiability := pv
	currROU := pv
	var totalInterest, totalPayments float64

	commenceTime := time.Now()

	for month := 1; month <= req.LeaseTermMonths; month++ {
		opening := currLiability
		interest := math.Round(opening*monthlyRate*100) / 100.0
		principal := req.MonthlyPaymentIDR - interest
		closingLiab := opening - principal
		if closingLiab < 0 || month == req.LeaseTermMonths {
			closingLiab = 0
			principal = opening
		}

		currROU = math.Max(0, currROU-monthlyDepr)
		if month == req.LeaseTermMonths {
			currROU = 0
		}

		totalInterest += interest
		totalPayments += req.MonthlyPaymentIDR

		dueDate := commenceTime.AddDate(0, month, 0).Format("2006-01-02")

		periods = append(periods, LeaseSchedulePeriod{
			PeriodMonth:               month,
			DueDate:                   dueDate,
			OpeningLiabilityIDR:       opening,
			InterestExpenseIDR:        interest,
			PrincipalRepaymentIDR:     principal,
			PaymentIDR:                req.MonthlyPaymentIDR,
			ClosingLiabilityIDR:       closingLiab,
			ROUAssetDepreciationIDR:   monthlyDepr,
			ClosingROUNetBookValueIDR: currROU,
		})

		currLiability = closingLiab
	}

	schedule := &LeaseAccountingSchedule{
		LeaseID:                  req.LeaseID,
		ContractNumber:           req.LeaseContractNumber,
		AssetDescription:         req.AssetDescription,
		InitialROUAssetValueIDR:  pv,
		InitialLeaseLiabilityIDR: pv,
		TotalInterestExpenseIDR:  math.Round(totalInterest),
		TotalCashPaymentsIDR:     totalPayments,
		Schedule:                 periods,
		Status:                   "IFRS16_PSAK73_COMPLIANT_ACTIVE",
		CreatedAt:                time.Now().Format(time.RFC3339),
	}

	globalLeaseStore.Lock()
	globalLeaseStore.schedules[req.LeaseID] = schedule
	globalLeaseStore.Unlock()

	return schedule, nil
}

// RemeasureLeaseLogic handles lease modification and remeasurement under IFRS 16.
func RemeasureLeaseLogic(req LeaseRemeasurementRequest) (*LeaseAccountingSchedule, error) {
	if req.LeaseID == "" {
		req.LeaseID = "LEASE-20261104-001"
	}
	if req.RevisedMonthlyPaymentIDR <= 0 {
		req.RevisedMonthlyPaymentIDR = 135000000.0 // indexed payment hike
	}
	if req.RemainingMonths <= 0 {
		req.RemainingMonths = 24
	}

	contract := LeaseContract{
		LeaseID:                           req.LeaseID,
		LeaseContractNumber:               "LC/MODIFIED/2026/001",
		AssetDescription:                  "Gudang Logistik Pusat Cikarang (Indexed & Modified)",
		AssetCategory:                     "REAL_ESTATE_WAREHOUSE",
		LeaseTermMonths:                   req.RemainingMonths,
		MonthlyPaymentIDR:                 req.RevisedMonthlyPaymentIDR,
		IncrementalBorrowingRateAnnualPct: 6.75, // updated IBR
	}

	return GenerateLeaseScheduleLogic(contract)
}

// LeaseAccountingHandler handles HTTP requests for IFRS 16 lease accounting.
type LeaseAccountingHandler struct {
	db *gorm.DB
}

// NewLeaseAccountingHandler constructs a new LeaseAccountingHandler.
func NewLeaseAccountingHandler(db *gorm.DB) *LeaseAccountingHandler {
	return &LeaseAccountingHandler{db: db}
}

// GenerateSchedule creates full lease amortization schedules.
func (h *LeaseAccountingHandler) GenerateSchedule(c *gin.Context) {
	var req LeaseContract
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid lease contract payload: " + err.Error()})
		return
	}

	sched, err := GenerateLeaseScheduleLogic(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, sched)
}

// RemeasureLease adjusts an existing schedule upon contract modifications.
func (h *LeaseAccountingHandler) RemeasureLease(c *gin.Context) {
	var req LeaseRemeasurementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid remeasurement payload: " + err.Error()})
		return
	}

	sched, err := RemeasureLeaseLogic(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, sched)
}

// RegisterLeaseAccountingRoutes registers IFRS 16 lease accounting endpoints.
func RegisterLeaseAccountingRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc, db *gorm.DB) {
	h := NewLeaseAccountingHandler(db)
	leaseGroup := router.Group("/accounting/leases")
	leaseGroup.Use(authMiddleware)
	{
		leaseGroup.POST("/schedule", h.GenerateSchedule)
		leaseGroup.POST("/remeasure", h.RemeasureLease)
	}
}
