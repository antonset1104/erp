package expense

import (
	"erp-golang/internal/core/models"
	"erp-golang/internal/modules/account"
	"erp-golang/internal/modules/base"
	"erp-golang/internal/modules/fleet"
	"erp-golang/internal/modules/hr"
	"erp-golang/internal/modules/project"
)

// ExpenseCategory corresponds to Odoo's hr.expense.category (Expense Products & Types)
type ExpenseCategory struct {
	models.ModelBase
	Name          string                  `gorm:"size:128;not null;index" json:"name"`         // e.g. "Perjalanan Dinas & Akomodasi"
	Code          string                  `gorm:"size:32;not null;uniqueIndex" json:"code"`    // e.g. "TRAVEL", "MEAL", "FUEL", "OFFICE"
	AccountCode   string                  `gorm:"size:32;default:'620100'" json:"account_code"` // Chart of accounts code
	AccountID     *uint                   `gorm:"index" json:"account_id,omitempty"`
	Account       *account.AccountAccount `gorm:"foreignKey:AccountID" json:"account,omitempty"`
	StandardPrice float64                 `gorm:"default:0" json:"standard_price"`             // Standard unit cost if fixed (e.g. per KM)
	Color         string                  `gorm:"size:32;default:'#3b82f6'" json:"color"`      // Hex badge color
	Description   string                  `gorm:"size:255" json:"description,omitempty"`
}

// Expense corresponds to Odoo's hr.expense (Individual Expense Item / Claim Line)
type Expense struct {
	models.ModelBase
	ExpenseNumber string              `gorm:"size:64;not null;uniqueIndex" json:"expense_number"` // e.g. EXP-ITM/2026/00001
	Name          string              `gorm:"size:255;not null" json:"name"`                      // e.g. "BBM Solar Truk Ekspedisi B 1024 TKO"
	Date          string              `gorm:"size:32;not null;index" json:"date"`                 // YYYY-MM-DD
	EmployeeID    uint                `gorm:"not null;index" json:"employee_id"`
	Employee      *hr.Employee        `gorm:"foreignKey:EmployeeID" json:"employee,omitempty"`
	CategoryID    uint                `gorm:"not null;index" json:"category_id"`
	Category      *ExpenseCategory    `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	SheetID       *uint               `gorm:"index" json:"sheet_id,omitempty"`                    // Optional link to bundled ExpenseSheet
	PaymentMode   string              `gorm:"size:32;default:'own_account';index" json:"payment_mode"` // "own_account" (reimburse employee), "company_account" (company card/cash)
	UnitPrice     float64             `gorm:"not null" json:"unit_price"`
	Quantity      float64             `gorm:"default:1" json:"quantity"`
	TotalAmount   float64             `gorm:"not null" json:"total_amount"`                       // UnitPrice * Quantity
	ReceiptRef    string              `gorm:"size:128" json:"receipt_ref,omitempty"`              // Invoice / Struk SPBU number
	ProjectID     *uint               `gorm:"index" json:"project_id,omitempty"`                  // Project costing / client billable
	Project       *project.Project    `gorm:"foreignKey:ProjectID" json:"project,omitempty"`
	VehicleID     *uint               `gorm:"index" json:"vehicle_id,omitempty"`                  // Fleet vehicle operational cost
	Vehicle       *fleet.FleetVehicle `gorm:"foreignKey:VehicleID" json:"vehicle,omitempty"`
	State         string              `gorm:"size:32;default:'draft';index" json:"state"`         // "draft", "reported", "approved", "done", "refused"
	Notes         string              `gorm:"type:text" json:"notes,omitempty"`
}

// ExpenseSheet corresponds to Odoo's hr.expense.sheet (Expense Report / Rekapitulasi Pengajuan)
type ExpenseSheet struct {
	models.ModelBase
	SheetNumber    string               `gorm:"size:64;not null;uniqueIndex" json:"sheet_number"` // e.g. EXP/2026/00001
	Name           string               `gorm:"size:255;not null" json:"name"`                   // e.g. "Klaim Perjalanan Dinas Surabaya - Budi Santoso"
	EmployeeID     uint                 `gorm:"not null;index" json:"employee_id"`
	Employee       *hr.Employee         `gorm:"foreignKey:EmployeeID" json:"employee,omitempty"`
	AccountingDate string               `gorm:"size:32" json:"accounting_date,omitempty"`        // Posting date for GL
	PaymentMode    string               `gorm:"size:32;default:'own_account'" json:"payment_mode"`
	TotalAmount    float64              `gorm:"default:0" json:"total_amount"`
	State          string               `gorm:"size:32;default:'draft';index" json:"state"`      // "draft", "submit", "approve", "post", "done", "refuse"
	ApproverID     *uint                `gorm:"index" json:"approver_id,omitempty"`
	Approver       *base.User           `gorm:"foreignKey:ApproverID" json:"approver,omitempty"`
	ApprovalDate   string               `gorm:"size:32" json:"approval_date,omitempty"`
	AccountMoveID  *uint                `gorm:"index" json:"account_move_id,omitempty"`          // Link to posted General Ledger entry
	AccountMove    *account.AccountMove `gorm:"foreignKey:AccountMoveID" json:"account_move,omitempty"`
	RefusalReason  string               `gorm:"type:text" json:"refusal_reason,omitempty"`
	Notes          string               `gorm:"type:text" json:"notes,omitempty"`
	Expenses       []Expense            `gorm:"foreignKey:SheetID" json:"expenses,omitempty"`
}
