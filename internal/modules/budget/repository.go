package budget

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) DB() *gorm.DB {
	return r.db
}

// --- Analytic Accounts ---

func (r *Repository) CreateAnalyticAccount(acc *AnalyticAccount) error {
	return r.db.Create(acc).Error
}

func (r *Repository) GetAnalyticAccountByID(id uint) (*AnalyticAccount, error) {
	var acc AnalyticAccount
	if err := r.db.Preload("Partner").First(&acc, id).Error; err != nil {
		return nil, err
	}
	return &acc, nil
}

func (r *Repository) ListAnalyticAccounts(companyID uint) ([]AnalyticAccount, error) {
	var list []AnalyticAccount
	q := r.db.Preload("Partner").Order("code asc")
	if companyID > 0 {
		q = q.Where("company_id = ?", companyID)
	}
	if err := q.Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// --- Budgetary Positions ---

func (r *Repository) CreateBudgetaryPosition(pos *BudgetaryPosition) error {
	return r.db.Create(pos).Error
}

func (r *Repository) GetBudgetaryPositionByID(id uint) (*BudgetaryPosition, error) {
	var pos BudgetaryPosition
	if err := r.db.Preload("Accounts").First(&pos, id).Error; err != nil {
		return nil, err
	}
	return &pos, nil
}

func (r *Repository) ListBudgetaryPositions(companyID uint) ([]BudgetaryPosition, error) {
	var list []BudgetaryPosition
	q := r.db.Preload("Accounts").Order("code asc")
	if companyID > 0 {
		q = q.Where("company_id = ?", companyID)
	}
	if err := q.Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// --- Budgets ---

func (r *Repository) CreateBudget(b *Budget) error {
	return r.db.Create(b).Error
}

func (r *Repository) GetBudgetByID(id uint) (*Budget, error) {
	var b Budget
	err := r.db.Preload("User").
		Preload("Lines.BudgetaryPosition").
		Preload("Lines.AnalyticAccount").
		First(&b, id).Error
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *Repository) ListBudgets(companyID uint, state string, search string) ([]Budget, error) {
	var list []Budget
	q := r.db.Preload("User").
		Preload("Lines.BudgetaryPosition").
		Preload("Lines.AnalyticAccount").
		Order("id desc")

	if companyID > 0 {
		q = q.Where("company_id = ?", companyID)
	}
	if state != "" {
		q = q.Where("state = ?", state)
	}
	if search != "" {
		pattern := "%" + strings.ToLower(search) + "%"
		q = q.Where("LOWER(name) LIKE ? OR LOWER(code) LIKE ?", pattern, pattern)
	}

	if err := q.Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *Repository) UpdateBudget(b *Budget) error {
	return r.db.Save(b).Error
}

func (r *Repository) UpdateBudgetLine(line *BudgetLine) error {
	return r.db.Save(line).Error
}

func (r *Repository) ReplaceBudgetLines(budgetID uint, lines []BudgetLine) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("budget_id = ?", budgetID).Delete(&BudgetLine{}).Error; err != nil {
			return err
		}
		if len(lines) > 0 {
			for i := range lines {
				lines[i].BudgetID = budgetID
			}
			if err := tx.Create(&lines).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// --- Realized Expenditure Calculation ---

// QueryPracticalExpenditure calculates sum(debit - credit) from posted account_move_lines
// for specified account IDs and date interval
func (r *Repository) QueryPracticalExpenditure(accountIDs []uint, dateFrom string, dateTo string, companyID uint) (float64, error) {
	if len(accountIDs) == 0 {
		return 0, nil
	}

	type Result struct {
		TotalDebit  float64
		TotalCredit float64
	}
	var res Result

	q := r.db.Table("account_move_lines").
		Select("COALESCE(SUM(account_move_lines.debit), 0) as total_debit, COALESCE(SUM(account_move_lines.credit), 0) as total_credit").
		Joins("JOIN account_moves ON account_moves.id = account_move_lines.move_id").
		Where("account_moves.state = ?", "posted").
		Where("account_move_lines.account_id IN (?)", accountIDs)

	if companyID > 0 {
		q = q.Where("account_move_lines.company_id = ?", companyID)
	}
	if dateFrom != "" {
		q = q.Where("account_moves.invoice_date >= ?", dateFrom)
	}
	if dateTo != "" {
		q = q.Where("account_moves.invoice_date <= ?", dateTo)
	}

	if err := q.Scan(&res).Error; err != nil {
		return 0, err
	}

	// For expense accounts: net cost is Debit - Credit
	practical := res.TotalDebit - res.TotalCredit
	if practical < 0 {
		practical = 0
	}
	return practical, nil
}

// GetStats compiles executive KPI analytics
func (r *Repository) GetStats(companyID uint) (*BudgetStats, error) {
	stats := &BudgetStats{}

	q := r.db.Model(&Budget{})
	if companyID > 0 {
		q = q.Where("company_id = ?", companyID)
	}

	q.Count(&stats.TotalBudgets)

	var draftCount, confirmCount, validateCount, doneCount int64
	r.db.Model(&Budget{}).Where("state = ?", "draft").Count(&draftCount)
	r.db.Model(&Budget{}).Where("state = ?", "confirm").Count(&confirmCount)
	r.db.Model(&Budget{}).Where("state = ?", "validate").Count(&validateCount)
	r.db.Model(&Budget{}).Where("state = ?", "done").Count(&doneCount)

	stats.DraftBudgets = draftCount
	stats.ConfirmedBudgets = confirmCount
	stats.ValidatedBudgets = validateCount
	stats.DoneBudgets = doneCount

	type Totals struct {
		Planned     float64
		Practical   float64
		Theoretical float64
	}
	var totals Totals
	qTotals := r.db.Model(&Budget{})
	if companyID > 0 {
		qTotals = qTotals.Where("company_id = ?", companyID)
	}
	qTotals.Select("COALESCE(SUM(total_planned_amount), 0) as planned, COALESCE(SUM(total_practical_amount), 0) as practical, COALESCE(SUM(total_theoretical_amount), 0) as theoretical").Scan(&totals)

	stats.TotalPlannedAmount = totals.Planned
	stats.TotalPracticalAmount = totals.Practical
	stats.TotalTheoreticalAmount = totals.Theoretical

	if stats.TotalPlannedAmount > 0 {
		stats.AverageAchievementRate = (stats.TotalPracticalAmount / stats.TotalPlannedAmount) * 100
	}

	// Overspent lines count: practical > planned AND planned > 0
	r.db.Model(&BudgetLine{}).Where("practical_amount > planned_amount AND planned_amount > 0").Count(&stats.OverspentLinesCount)

	// High risk budgets count
	r.db.Model(&Budget{}).Where("ai_risk_score >= 60 OR ai_check_status IN ('high', 'critical')").Count(&stats.HighRiskBudgetsCount)

	return stats, nil
}

// NextBudgetCode generates sequence like BDG/YYYY/Qx/00001
func (r *Repository) NextBudgetCode(year string, quarter string) string {
	prefix := fmt.Sprintf("BDG/%s/%s", year, quarter)
	var count int64
	r.db.Model(&Budget{}).Where("code LIKE ?", prefix+"%").Count(&count)

	candidate := fmt.Sprintf("%s/%05d", prefix, count+1)
	var exists int64
	for {
		r.db.Model(&Budget{}).Where("code = ?", candidate).Count(&exists)
		if exists == 0 {
			break
		}
		count++
		candidate = fmt.Sprintf("%s/%05d", prefix, count+1)
	}
	return candidate
}
