package hr

import (
	"erp-golang/internal/core/models"
)

// Department corresponds to standard enterprise hr.department
type Department struct {
	models.ModelBase
	Name        string `gorm:"size:128;not null;index" json:"name"`
	ManagerID   *uint  `json:"manager_id,omitempty"`
	Description string `gorm:"type:text" json:"description"`
}

// JobPosition corresponds to standard enterprise hr.job
type JobPosition struct {
	models.ModelBase
	Name         string     `gorm:"size:128;not null;index" json:"name"` // e.g. "Software Engineer"
	DepartmentID uint       `gorm:"not null;index" json:"department_id"`
	Department   Department `gorm:"foreignKey:DepartmentID" json:"department,omitempty"`
}

// Employee corresponds to standard enterprise hr.employee
type Employee struct {
	models.ModelBase
	Name         string      `gorm:"size:128;not null;index" json:"name"`
	WorkEmail    string      `gorm:"size:128;uniqueIndex" json:"work_email"`
	WorkPhone    string      `gorm:"size:64" json:"work_phone"`
	DepartmentID uint        `gorm:"not null;index" json:"department_id"`
	Department   Department  `gorm:"foreignKey:DepartmentID" json:"department,omitempty"`
	JobID        uint        `gorm:"not null;index" json:"job_id"`
	Job          JobPosition `gorm:"foreignKey:JobID" json:"job,omitempty"`
	BasicSalary  float64     `gorm:"default:0" json:"basic_salary"`
	BankName     string      `gorm:"size:64" json:"bank_name"`
	BankAccount  string      `gorm:"size:64" json:"bank_account"`
	Status       string      `gorm:"size:32;default:'active'" json:"status"` // "active", "resigned"
}

// Payslip corresponds to standard enterprise hr.payslip (Employee Payroll)
type Payslip struct {
	models.ModelBase
	Name        string   `gorm:"size:64;not null;uniqueIndex" json:"name"` // e.g. SLIP/2026/10/00001
	EmployeeID  uint     `gorm:"not null;index" json:"employee_id"`
	Employee    Employee `gorm:"foreignKey:EmployeeID" json:"employee,omitempty"`
	Period      string   `gorm:"size:32;not null" json:"period"` // e.g. "October 2026"
	Date        string   `gorm:"size:32;not null" json:"date"`
	BasicSalary float64  `gorm:"not null" json:"basic_salary"`
	Allowances  float64  `gorm:"default:0" json:"allowances"` // Tunjangan transport, makan, bonus
	Deductions  float64  `gorm:"default:0" json:"deductions"` // BPJS, PPh 21, potongan
	NetSalary   float64  `gorm:"not null" json:"net_salary"`   // Basic + Allowances - Deductions
	State       string   `gorm:"size:32;default:'draft';index" json:"state"` // "draft", "confirmed", "paid"
	Note        string   `gorm:"type:text" json:"note"`
}
