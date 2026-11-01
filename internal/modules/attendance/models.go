package attendance

import (
	"time"

	"erp-golang/internal/core/models"
	"erp-golang/internal/modules/hr"
)

// Shift corresponds to Odoo's resource.calendar / work shift schedule
type Shift struct {
	models.ModelBase
	Name               string  `gorm:"size:128;not null;index" json:"name"` // e.g. "Shift Reguler (08:30 - 17:30)"
	Code               string  `gorm:"size:32;not null;index" json:"code"`  // e.g. "REGULAR", "NIGHT", "WFH"
	StartTime          string  `gorm:"size:16;not null" json:"start_time"`  // "08:30"
	EndTime            string  `gorm:"size:16;not null" json:"end_time"`    // "17:30"
	StandardHours      float64 `gorm:"default:8.0" json:"standard_hours"`   // Jam kerja standar di luar istirahat
	GracePeriodMinutes int     `gorm:"default:15" json:"grace_period_minutes"` // Toleransi keterlambatan menit
	IsFlexible         bool    `gorm:"default:false" json:"is_flexible"`
	Color              string  `gorm:"size:32;default:'#3b82f6'" json:"color"`
	Description        string  `gorm:"size:255" json:"description,omitempty"`
}

func (Shift) TableName() string {
	return "hr_attendance_shifts"
}

// Attendance corresponds to Odoo's hr.attendance
type Attendance struct {
	models.ModelBase
	EmployeeID        uint         `gorm:"not null;index" json:"employee_id"`
	Employee          *hr.Employee `gorm:"foreignKey:EmployeeID" json:"employee,omitempty"`
	ShiftID           *uint        `gorm:"index" json:"shift_id,omitempty"`
	Shift             *Shift       `gorm:"foreignKey:ShiftID" json:"shift,omitempty"`
	Date              string       `gorm:"size:32;not null;index" json:"date"` // YYYY-MM-DD
	CheckIn           time.Time    `gorm:"not null;index" json:"check_in"`
	CheckOut          *time.Time   `json:"check_out,omitempty"`
	WorkedHours       float64      `gorm:"default:0" json:"worked_hours"`
	OvertimeHours     float64      `gorm:"default:0" json:"overtime_hours"`
	LateMinutes       int          `gorm:"default:0" json:"late_minutes"`
	EarlyLeaveMinutes int          `gorm:"default:0" json:"early_leave_minutes"`
	Status            string       `gorm:"size:32;default:'checked_in';index" json:"status"` // "checked_in", "checked_out", "late", "overtime", "normal"

	// GPS & Cloudflare Edge Telemetry
	InLatitude   float64 `json:"in_latitude,omitempty"`
	InLongitude  float64 `json:"in_longitude,omitempty"`
	InAddress    string  `gorm:"size:255" json:"in_address,omitempty"`
	OutLatitude  float64 `json:"out_latitude,omitempty"`
	OutLongitude float64 `json:"out_longitude,omitempty"`
	OutAddress   string  `gorm:"size:255" json:"out_address,omitempty"`
	Device       string  `gorm:"size:128" json:"device,omitempty"` // User Agent / Kiosk terminal
	IPAddress    string  `gorm:"size:64" json:"ip_address,omitempty"` // Cloudflare CF-Connecting-IP
	Country      string  `gorm:"size:32" json:"country,omitempty"` // Cloudflare CF-IPCountry
	CFRay        string  `gorm:"size:64" json:"cf_ray,omitempty"` // Cloudflare Ray ID

	Note             string `gorm:"type:text" json:"note,omitempty"`
	AnomalyFlag      bool   `gorm:"default:false;index" json:"anomaly_flag"`
	AnomalyReason    string `gorm:"size:255" json:"anomaly_reason,omitempty"`
	SyncedToPayroll  bool   `gorm:"default:false;index" json:"synced_to_payroll"`
}

func (Attendance) TableName() string {
	return "hr_attendances"
}

// CheckInRequest represents payload to clock-in
type CheckInRequest struct {
	EmployeeID uint    `json:"employee_id" binding:"required"`
	ShiftID    *uint   `json:"shift_id"`
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
	Address    string  `json:"address"`
	Device     string  `json:"device"`
	Note       string  `json:"note"`
}

// CheckOutRequest represents payload to clock-out
type CheckOutRequest struct {
	AttendanceID uint    `json:"attendance_id"`
	EmployeeID   uint    `json:"employee_id" binding:"required"`
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
	Address      string  `json:"address"`
	Note         string  `json:"note"`
}

// AttendanceFilter provides filtering options for attendance query
type AttendanceFilter struct {
	EmployeeID   uint
	DepartmentID uint
	DateFrom     string
	DateTo       string
	Status       string
	AnomalyOnly  bool
	Search       string
	Limit        int
	Offset       int
}

// AttendanceStats provides executive analytics for attendance
type AttendanceStats struct {
	TotalToday          int64   `json:"total_today"`
	CurrentlyCheckedIn  int64   `json:"currently_checked_in"`
	LateToday           int64   `json:"late_today"`
	TotalOvertimeHours  float64 `json:"total_overtime_hours"`
	AverageWorkedHours  float64 `json:"average_worked_hours"`
	AnomalyCount        int64   `json:"anomaly_count"`
}

// SyncPayrollRequest provides input for syncing attendance overtime & late penalties to employee payslip
type SyncPayrollRequest struct {
	EmployeeID           uint    `json:"employee_id" binding:"required"`
	Period               string  `json:"period" binding:"required"` // e.g. "October 2026"
	OvertimeRatePerHour  float64 `json:"overtime_rate_per_hour"`    // e.g. 50000 IDR/hr
	LatePenaltyPerMinute float64 `json:"late_penalty_per_minute"`   // e.g. 1000 IDR/min
}

// AttendanceAnomalyResult holds analysis output from Cloudflare Workers AI or local anomaly heuristic
type AttendanceAnomalyResult struct {
	AnalyzedAttendances  int      `json:"analyzed_attendances"`
	AnomaliesDetected    int      `json:"anomalies_detected"`
	BurnoutRiskEmployees []string `json:"burnout_risk_employees"`
	ExcessiveTardiness   []string `json:"excessive_tardiness"`
	SuspiciousLocations  []string `json:"suspicious_locations"`
	Summary              string   `json:"summary"`
	Recommendations      []string `json:"recommendations"`
	AIConfidence         float64  `json:"ai_confidence"`
	Source               string   `json:"source"`
}
