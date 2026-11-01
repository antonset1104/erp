package payroll

import (
	"erp-golang/internal/core/models"
	"erp-golang/internal/modules/hr"
)

// SalaryRuleCategory represents Odoo hr.salary.rule.category
type SalaryRuleCategory struct {
	models.ModelBase
	Code        string `gorm:"size:32;not null;index" json:"code"` // BASIC, ALW, DED, TAX, NET
	Name        string `gorm:"size:128;not null" json:"name"`
	Description string `gorm:"size:255" json:"description,omitempty"`
}

// TableName specifies custom table name
func (SalaryRuleCategory) TableName() string {
	return "payroll_salary_rule_categories"
}

// PayrollStructure represents Odoo hr.payroll.structure
type PayrollStructure struct {
	models.ModelBase
	Name        string       `gorm:"size:128;not null;index" json:"name"`
	Code        string       `gorm:"size:32;not null;index" json:"code"` // e.g. IDN_STD, IDN_EXEC, IDN_HOURLY
	Description string       `gorm:"size:255" json:"description,omitempty"`
	Active      bool         `gorm:"default:true" json:"active"`
	Rules       []SalaryRule `gorm:"foreignKey:StructureID" json:"rules,omitempty"`
}

// TableName specifies custom table name
func (PayrollStructure) TableName() string {
	return "payroll_structures"
}

// SalaryRule represents Odoo hr.salary.rule
type SalaryRule struct {
	models.ModelBase
	StructureID       uint    `gorm:"not null;index" json:"structure_id"`
	Name              string  `gorm:"size:128;not null" json:"name"`
	Code              string  `gorm:"size:64;not null;index" json:"code"`     // e.g. BASIC, ALW_JOB, ALW_OT, DED_BPJS_TK, NET
	Category          string  `gorm:"size:32;not null;index" json:"category"` // BASIC, ALW, DED, TAX, NET
	Sequence          int     `gorm:"default:10;index" json:"sequence"`
	AmountType        string  `gorm:"size:32;default:'fixed'" json:"amount_type"` // fixed, percentage, dynamic
	FixedAmount       float64 `gorm:"default:0" json:"fixed_amount"`
	Percentage        float64 `gorm:"default:0" json:"percentage"`
	DebitAccountCode  string  `gorm:"size:32" json:"debit_account_code,omitempty"`
	CreditAccountCode string  `gorm:"size:32" json:"credit_account_code,omitempty"`
	Active            bool    `gorm:"default:true" json:"active"`
}

// TableName specifies custom table name
func (SalaryRule) TableName() string {
	return "payroll_salary_rules"
}

// PayslipRun represents Odoo hr.payslip.run (Batch Payroll Processing)
type PayslipRun struct {
	models.ModelBase
	Name            string    `gorm:"size:128;not null;uniqueIndex" json:"name"` // e.g. BATCH/PAY/2026/10
	Period          string    `gorm:"size:32;not null;index" json:"period"`      // e.g. 2026-10 / October 2026
	DateStart       string    `gorm:"size:32;not null" json:"date_start"`
	DateEnd         string    `gorm:"size:32;not null" json:"date_end"`
	State           string    `gorm:"size:32;default:'draft';index" json:"state"` // draft, verify, close, paid
	TotalEmployees  int       `gorm:"default:0" json:"total_employees"`
	TotalGross      float64   `gorm:"default:0" json:"total_gross"`
	TotalDeductions float64   `gorm:"default:0" json:"total_deductions"`
	TotalNet        float64   `gorm:"default:0" json:"total_net"`
	Notes           string    `gorm:"type:text" json:"notes,omitempty"`
	Payslips        []Payslip `gorm:"foreignKey:PayslipRunID" json:"payslips,omitempty"`
}

// TableName specifies custom table name
func (PayslipRun) TableName() string {
	return "payroll_payslip_runs"
}

// Payslip represents Odoo hr.payslip
type Payslip struct {
	models.ModelBase
	Number               string            `gorm:"size:64;not null;uniqueIndex" json:"number"` // e.g. SLIP/2026/10/00001
	EmployeeID           uint              `gorm:"not null;index" json:"employee_id"`
	Employee             *hr.Employee      `gorm:"foreignKey:EmployeeID" json:"employee,omitempty"`
	StructureID          uint              `gorm:"not null;index" json:"structure_id"`
	Structure            *PayrollStructure `gorm:"foreignKey:StructureID" json:"structure,omitempty"`
	PayslipRunID         *uint             `gorm:"index" json:"payslip_run_id,omitempty"`
	Period               string            `gorm:"size:32;not null;index" json:"period"` // e.g. October 2026
	DateFrom             string            `gorm:"size:32;not null" json:"date_from"`
	DateTo               string            `gorm:"size:32;not null" json:"date_to"`
	WorkingDays          float64           `gorm:"default:22" json:"working_days"`
	WorkedHours          float64           `gorm:"default:176" json:"worked_hours"`
	OvertimeHours        float64           `gorm:"default:0" json:"overtime_hours"`
	OvertimeRate         float64           `gorm:"default:0" json:"overtime_rate"`
	OvertimeAmount       float64           `gorm:"default:0" json:"overtime_amount"`
	UnpaidLeaveDays      float64           `gorm:"default:0" json:"unpaid_leave_days"`
	UnpaidLeaveDeduction float64           `gorm:"default:0" json:"unpaid_leave_deduction"`
	BasicSalary          float64           `gorm:"default:0" json:"basic_salary"`
	GrossSalary          float64           `gorm:"default:0" json:"gross_salary"`
	TotalDeductions      float64           `gorm:"default:0" json:"total_deductions"`
	NetSalary            float64           `gorm:"default:0" json:"net_salary"`
	State                string            `gorm:"size:32;default:'draft';index" json:"state"` // draft, verify, done, paid, cancel
	AccountingMoveID     *uint             `gorm:"index" json:"accounting_move_id,omitempty"`
	AccountingMoveRef    string            `gorm:"size:64" json:"accounting_move_ref,omitempty"`
	AICheckStatus        string            `gorm:"size:32;default:'pending'" json:"ai_check_status"` // pending, passed, warning, flagged
	AIRiskScore          float64           `gorm:"default:0" json:"ai_risk_score"`
	AINotes              string            `gorm:"type:text" json:"ai_notes,omitempty"`
	Notes                string            `gorm:"type:text" json:"notes,omitempty"`
	Lines                []PayslipLine     `gorm:"foreignKey:PayslipID" json:"lines,omitempty"`
}

// TableName specifies custom table name
func (Payslip) TableName() string {
	return "payroll_payslips"
}

// PayslipLine represents Odoo hr.payslip.line
type PayslipLine struct {
	models.ModelBase
	PayslipID         uint    `gorm:"not null;index" json:"payslip_id"`
	SalaryRuleID      *uint   `gorm:"index" json:"salary_rule_id,omitempty"`
	Code              string  `gorm:"size:64;not null;index" json:"code"`
	Name              string  `gorm:"size:128;not null" json:"name"`
	Category          string  `gorm:"size:32;not null;index" json:"category"`
	Sequence          int     `gorm:"default:10" json:"sequence"`
	Rate              float64 `gorm:"default:100" json:"rate"` // percentage or multiplier
	Amount            float64 `gorm:"default:0" json:"amount"`
	Total             float64 `gorm:"default:0" json:"total"`
	DebitAccountCode  string  `gorm:"size:32" json:"debit_account_code,omitempty"`
	CreditAccountCode string  `gorm:"size:32" json:"credit_account_code,omitempty"`
}

// TableName specifies custom table name
func (PayslipLine) TableName() string {
	return "payroll_payslip_lines"
}

// --- DTOs and Requests ---

type CreatePayrollStructureRequest struct {
	Name        string                    `json:"name" binding:"required"`
	Code        string                    `json:"code" binding:"required"`
	Description string                    `json:"description"`
	Rules       []CreateSalaryRuleRequest `json:"rules"`
}

type CreateSalaryRuleRequest struct {
	Name              string  `json:"name" binding:"required"`
	Code              string  `json:"code" binding:"required"`
	Category          string  `json:"category" binding:"required"`
	Sequence          int     `json:"sequence"`
	AmountType        string  `json:"amount_type"`
	FixedAmount       float64 `json:"fixed_amount"`
	Percentage        float64 `json:"percentage"`
	DebitAccountCode  string  `json:"debit_account_code"`
	CreditAccountCode string  `json:"credit_account_code"`
}

type CreatePayslipRunRequest struct {
	Name      string `json:"name" binding:"required"`
	Period    string `json:"period" binding:"required"`
	DateStart string `json:"date_start" binding:"required"`
	DateEnd   string `json:"date_end" binding:"required"`
	Notes     string `json:"notes"`
}

type CreatePayslipRequest struct {
	EmployeeID   uint    `json:"employee_id" binding:"required"`
	StructureID  uint    `json:"structure_id" binding:"required"`
	PayslipRunID *uint   `json:"payslip_run_id"`
	Period       string  `json:"period" binding:"required"`
	DateFrom     string  `json:"date_from" binding:"required"`
	DateTo       string  `json:"date_to" binding:"required"`
	WorkingDays  float64 `json:"working_days"`
	Notes        string  `json:"notes"`
}

type ComputePayslipRequest struct {
	SyncAttendance bool    `json:"sync_attendance"`
	SyncLeave      bool    `json:"sync_leave"`
	ManualOTHours  float64 `json:"manual_ot_hours"`
}

type AIPayrollAuditResult struct {
	PayslipID                uint     `json:"payslip_id"`
	EmployeeName             string   `json:"employee_name"`
	Period                   string   `json:"period"`
	RiskLevel                string   `json:"risk_level"` // low, medium, high
	ComplianceScore          float64  `json:"compliance_score"`
	TaxComplianceStatus      string   `json:"tax_compliance_status"`
	OvertimeRegulationStatus string   `json:"overtime_regulation_status"`
	AuditFlags               []string `json:"audit_flags"`
	ExecutiveSummary         string   `json:"executive_summary"`
	Recommendations          []string `json:"recommendations"`
	Source                   string   `json:"source"`
	AIConfidence             float64  `json:"ai_confidence"`
}

type PayrollStats struct {
	TotalRuns              int64   `json:"total_runs"`
	TotalPayslips          int64   `json:"total_payslips"`
	TotalPaidSlips         int64   `json:"total_paid_slips"`
	TotalDraftSlips        int64   `json:"total_draft_slips"`
	TotalConfirmedSlips    int64   `json:"total_confirmed_slips"`
	TotalGrossExpenditure  float64 `json:"total_gross_expenditure"`
	TotalDeductionsWithheld float64 `json:"total_deductions_withheld"`
	TotalNetPaid           float64 `json:"total_net_paid"`
	TotalOvertimeHours     float64 `json:"total_overtime_hours"`
	PendingAuditCount      int64   `json:"pending_audit_count"`
}
