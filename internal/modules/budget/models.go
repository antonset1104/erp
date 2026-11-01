package budget

import (
	"erp-golang/internal/core/models"
	"erp-golang/internal/modules/account"
	"erp-golang/internal/modules/base"
)

// AnalyticAccount represents Odoo's account.analytic.account (Cost Center / Project / Department)
type AnalyticAccount struct {
	models.ModelBase
	Code         string        `gorm:"size:64;not null;index" json:"code"` // e.g. "AN/IT", "AN/SALES", "AN/OPS"
	Name         string        `gorm:"size:128;not null" json:"name"`
	DepartmentID *uint         `gorm:"index" json:"department_id,omitempty"`
	PartnerID    *uint         `gorm:"index" json:"partner_id,omitempty"`
	Partner      *base.Partner `gorm:"foreignKey:PartnerID" json:"partner,omitempty"`
	Description  string        `gorm:"size:255" json:"description,omitempty"`
	Balance      float64       `gorm:"default:0" json:"balance"` // Net balance of expenditures
	Active       bool          `gorm:"default:true" json:"active"`
}

func (AnalyticAccount) TableName() string {
	return "account_analytic_accounts"
}

// BudgetaryPosition represents Odoo's account.budget.post (Budgetary Positions / Pos Anggaran)
type BudgetaryPosition struct {
	models.ModelBase
	Name        string                   `gorm:"size:128;not null" json:"name"`
	Code        string                   `gorm:"size:64;not null;index" json:"code"` // e.g. "BP/PAYROLL", "BP/OPS"
	Description string                   `gorm:"size:255" json:"description,omitempty"`
	AccountIDs  string                   `gorm:"type:text" json:"account_ids,omitempty"` // Comma-separated account IDs e.g. "20,21,22"
	Accounts    []account.AccountAccount `gorm:"many2many:budget_position_accounts;" json:"accounts,omitempty"`
}

func (BudgetaryPosition) TableName() string {
	return "account_budget_posts"
}

// Budget represents Odoo's crossovered.budget (account.budget)
type Budget struct {
	models.ModelBase
	Name                   string       `gorm:"size:128;not null" json:"name"`
	Code                   string       `gorm:"size:64;not null;uniqueIndex" json:"code"` // e.g. "BDG/2026/Q4/0001"
	UserID                 uint         `gorm:"not null;index" json:"user_id"`           // Responsible Manager / PIC
	User                   *base.User   `gorm:"foreignKey:UserID" json:"user,omitempty"`
	DateFrom               string       `gorm:"size:32;not null;index" json:"date_from"`
	DateTo                 string       `gorm:"size:32;not null;index" json:"date_to"`
	State                  string       `gorm:"size:32;default:'draft';index" json:"state"` // "draft", "confirm", "validate", "done", "cancel"
	TotalPlannedAmount     float64      `gorm:"default:0" json:"total_planned_amount"`
	TotalPracticalAmount   float64      `gorm:"default:0" json:"total_practical_amount"`
	TotalTheoreticalAmount float64      `gorm:"default:0" json:"total_theoretical_amount"`
	AchievementPercentage  float64      `gorm:"default:0" json:"achievement_percentage"`
	AICheckStatus          string       `gorm:"size:32;default:'pending'" json:"ai_check_status"` // "pending", "low", "medium", "high", "critical"
	AIRiskScore            float64      `gorm:"default:0" json:"ai_risk_score"`
	AINotes                string       `gorm:"type:text" json:"ai_notes,omitempty"`
	Notes                  string       `gorm:"type:text" json:"notes,omitempty"`
	Lines                  []BudgetLine `gorm:"foreignKey:BudgetID" json:"lines,omitempty"`
}

func (Budget) TableName() string {
	return "account_budgets"
}

// BudgetLine represents Odoo's crossovered.budget.lines
type BudgetLine struct {
	models.ModelBase
	BudgetID            uint               `gorm:"not null;index" json:"budget_id"`
	BudgetaryPositionID uint               `gorm:"not null;index" json:"budgetary_position_id"`
	BudgetaryPosition   *BudgetaryPosition `gorm:"foreignKey:BudgetaryPositionID" json:"budgetary_position,omitempty"`
	AnalyticAccountID   uint               `gorm:"not null;index" json:"analytic_account_id"`
	AnalyticAccount     *AnalyticAccount   `gorm:"foreignKey:AnalyticAccountID" json:"analytic_account,omitempty"`
	DateFrom            string             `gorm:"size:32;not null" json:"date_from"`
	DateTo              string             `gorm:"size:32;not null" json:"date_to"`
	PlannedAmount       float64            `gorm:"default:0" json:"planned_amount"`
	PracticalAmount     float64            `gorm:"default:0" json:"practical_amount"`
	TheoriticalAmount   float64            `gorm:"default:0" json:"theoritical_amount"`
	Percentage          float64            `gorm:"default:0" json:"percentage"`
	VarianceAmount      float64            `gorm:"default:0" json:"variance_amount"` // Planned - Practical (favorable if >= 0)
	Notes               string             `gorm:"size:255" json:"notes,omitempty"`
}

func (BudgetLine) TableName() string {
	return "account_budget_lines"
}

// --- DTOs and Requests ---

type CreateAnalyticAccountRequest struct {
	Code         string `json:"code" binding:"required"`
	Name         string `json:"name" binding:"required"`
	DepartmentID *uint  `json:"department_id"`
	PartnerID    *uint  `json:"partner_id"`
	Description  string `json:"description"`
}

type CreateBudgetaryPositionRequest struct {
	Name        string `json:"name" binding:"required"`
	Code        string `json:"code" binding:"required"`
	Description string `json:"description"`
	AccountIDs  []uint `json:"account_ids"`
}

type CreateBudgetLineRequest struct {
	BudgetaryPositionID uint    `json:"budgetary_position_id" binding:"required"`
	AnalyticAccountID   uint    `json:"analytic_account_id" binding:"required"`
	DateFrom            string  `json:"date_from"`
	DateTo              string  `json:"date_to"`
	PlannedAmount       float64 `json:"planned_amount" binding:"required"`
	Notes               string  `json:"notes"`
}

type CreateBudgetRequest struct {
	Name     string                    `json:"name" binding:"required"`
	Code     string                    `json:"code"`
	DateFrom string                    `json:"date_from" binding:"required"`
	DateTo   string                    `json:"date_to" binding:"required"`
	Notes    string                    `json:"notes"`
	Lines    []CreateBudgetLineRequest `json:"lines"`
}

type AIBudgetAuditResult struct {
	BudgetID           uint     `json:"budget_id"`
	BudgetName         string   `json:"budget_name"`
	RiskLevel          string   `json:"risk_level"` // low, medium, high, critical
	RiskScore          float64  `json:"risk_score"` // 0-100
	OverallBurnRate    string   `json:"overall_burn_rate"` // normal, elevated, critical
	FlaggedOverspends  []string `json:"flagged_overspends"`
	ExecutiveSummary   string   `json:"executive_summary"`
	RecommendedActions []string `json:"recommended_actions"`
	Source             string   `json:"source"`
	AIConfidence       float64  `json:"ai_confidence"`
}

type BudgetStats struct {
	TotalBudgets           int64   `json:"total_budgets"`
	DraftBudgets           int64   `json:"draft_budgets"`
	ConfirmedBudgets       int64   `json:"confirmed_budgets"`
	ValidatedBudgets       int64   `json:"validated_budgets"`
	DoneBudgets            int64   `json:"done_budgets"`
	TotalPlannedAmount     float64 `json:"total_planned_amount"`
	TotalPracticalAmount   float64 `json:"total_practical_amount"`
	TotalTheoreticalAmount float64 `json:"total_theoretical_amount"`
	AverageAchievementRate float64 `json:"average_achievement_rate"`
	OverspentLinesCount    int64   `json:"overspent_lines_count"`
	HighRiskBudgetsCount   int64   `json:"high_risk_budgets_count"`
}
