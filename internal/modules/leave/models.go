package leave

import (
	"erp-golang/internal/core/models"
	"erp-golang/internal/modules/base"
	"erp-golang/internal/modules/hr"
)

// LeaveType corresponds to standard enterprise hr.leave.type
type LeaveType struct {
	models.ModelBase
	Name               string  `gorm:"size:128;not null;index" json:"name"` // e.g. "Cuti Tahunan (Annual Leave)"
	Code               string  `gorm:"size:32;not null;index" json:"code"`  // e.g. "ANNUAL", "SICK", "UNPAID"
	IsUnpaid           bool    `gorm:"default:false" json:"is_unpaid"`      // If true, deducts from payroll payslip
	RequiresAllocation bool    `json:"requires_allocation"`
	DefaultDays        float64 `gorm:"default:12" json:"default_days"`
	Color              string  `gorm:"size:32;default:'#10b981'" json:"color"`
	Description        string  `gorm:"size:255" json:"description,omitempty"`
}

// LeaveAllocation corresponds to standard enterprise hr.leave.allocation
type LeaveAllocation struct {
	models.ModelBase
	Name          string       `gorm:"size:128;not null" json:"name"`
	EmployeeID    uint         `gorm:"not null;index" json:"employee_id"`
	Employee      *hr.Employee `gorm:"foreignKey:EmployeeID" json:"employee,omitempty"`
	LeaveTypeID   uint         `gorm:"not null;index" json:"leave_type_id"`
	LeaveType     *LeaveType   `gorm:"foreignKey:LeaveTypeID" json:"leave_type,omitempty"`
	Year          int          `gorm:"not null;index" json:"year"` // e.g. 2026
	AllocatedDays float64      `gorm:"not null" json:"allocated_days"`
	UsedDays      float64      `gorm:"default:0" json:"used_days"`
	RemainingDays float64      `gorm:"not null" json:"remaining_days"`
	State         string       `gorm:"size:32;default:'approved';index" json:"state"` // "draft", "approved", "refused"
	Notes         string       `gorm:"type:text" json:"notes,omitempty"`
}

// LeaveRequest corresponds to standard enterprise hr.leave (Time Off / Holiday)
type LeaveRequest struct {
	models.ModelBase
	RequestNumber string       `gorm:"size:64;not null;uniqueIndex" json:"request_number"` // e.g. LV/2026/00001
	EmployeeID    uint         `gorm:"not null;index" json:"employee_id"`
	Employee      *hr.Employee `gorm:"foreignKey:EmployeeID" json:"employee,omitempty"`
	LeaveTypeID   uint         `gorm:"not null;index" json:"leave_type_id"`
	LeaveType     *LeaveType   `gorm:"foreignKey:LeaveTypeID" json:"leave_type,omitempty"`
	DateFrom      string       `gorm:"size:32;not null" json:"date_from"` // YYYY-MM-DD
	DateTo        string       `gorm:"size:32;not null" json:"date_to"`   // YYYY-MM-DD
	NumberOfDays  float64      `gorm:"not null" json:"number_of_days"`    // e.g. 1.0, 2.5, 3.0
	Description   string       `gorm:"size:255;not null" json:"description"`
	State         string       `gorm:"size:32;default:'confirm';index" json:"state"` // "draft", "confirm", "validate", "refuse"
	ApproverID    *uint        `gorm:"index" json:"approver_id,omitempty"`
	Approver      *base.User   `gorm:"foreignKey:ApproverID" json:"approver,omitempty"`
	ApprovalDate  string       `gorm:"size:32" json:"approval_date,omitempty"`
	RefusalReason string       `gorm:"type:text" json:"refusal_reason,omitempty"`
}
