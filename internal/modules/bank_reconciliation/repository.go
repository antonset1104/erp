package bank_reconciliation

import (
	"erp-golang/internal/modules/account"
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// Statement CRUD

func (r *Repository) CreateStatement(stmt *AccountBankStatement) error {
	return r.db.Create(stmt).Error
}

func (r *Repository) GetStatementByID(id uint, companyID uint) (*AccountBankStatement, error) {
	var stmt AccountBankStatement
	err := r.db.Preload("Journal").
		Preload("LineIDs", func(db *gorm.DB) *gorm.DB {
			return db.Order("id asc")
		}).
		Preload("LineIDs.Partner").
		Where("id = ? AND company_id = ?", id, companyID).
		First(&stmt).Error
	if err != nil {
		return nil, err
	}
	return &stmt, nil
}

func (r *Repository) UpdateStatement(stmt *AccountBankStatement) error {
	return r.db.Save(stmt).Error
}

func (r *Repository) ListStatements(companyID uint, state string, journalID uint) ([]AccountBankStatement, error) {
	var stmts []AccountBankStatement
	query := r.db.Preload("Journal").Where("company_id = ?", companyID)
	if state != "" {
		query = query.Where("state = ?", state)
	}
	if journalID > 0 {
		query = query.Where("journal_id = ?", journalID)
	}
	err := query.Order("date desc, id desc").Find(&stmts).Error
	return stmts, err
}

func (r *Repository) DeleteStatement(id uint, companyID uint) error {
	return r.db.Where("id = ? AND company_id = ?", id, companyID).Delete(&AccountBankStatement{}).Error
}

// Statement Line CRUD

func (r *Repository) CreateStatementLines(lines []AccountBankStatementLine) error {
	if len(lines) == 0 {
		return nil
	}
	return r.db.Create(&lines).Error
}

func (r *Repository) CreateStatementLine(line *AccountBankStatementLine) error {
	return r.db.Create(line).Error
}

func (r *Repository) GetStatementLineByID(id uint, companyID uint) (*AccountBankStatementLine, error) {
	var line AccountBankStatementLine
	err := r.db.Preload("Partner").
		Preload("Statement.Journal").
		Where("id = ? AND company_id = ?", id, companyID).
		First(&line).Error
	if err != nil {
		return nil, err
	}
	return &line, nil
}

func (r *Repository) UpdateStatementLine(line *AccountBankStatementLine) error {
	return r.db.Save(line).Error
}

func (r *Repository) DeleteStatementLine(id uint, companyID uint) error {
	return r.db.Where("id = ? AND company_id = ?", id, companyID).Delete(&AccountBankStatementLine{}).Error
}

func (r *Repository) ListStatementLines(statementID uint, companyID uint) ([]AccountBankStatementLine, error) {
	var lines []AccountBankStatementLine
	err := r.db.Preload("Partner").
		Where("statement_id = ? AND company_id = ?", statementID, companyID).
		Order("id asc").
		Find(&lines).Error
	return lines, err
}

// Reconcile Models (Rules)

func (r *Repository) ListReconcileModels(companyID uint) ([]AccountReconcileModel, error) {
	var models []AccountReconcileModel
	err := r.db.Where("company_id = ?", companyID).Order("sequence asc, id asc").Find(&models).Error
	return models, err
}

func (r *Repository) CreateReconcileModel(model *AccountReconcileModel) error {
	return r.db.Create(model).Error
}

func (r *Repository) GetReconcileModelByID(id uint, companyID uint) (*AccountReconcileModel, error) {
	var model AccountReconcileModel
	err := r.db.Where("id = ? AND company_id = ?", id, companyID).First(&model).Error
	if err != nil {
		return nil, err
	}
	return &model, nil
}

// Open Invoices and Move Lines Search for Matching

func (r *Repository) FindOpenInvoices(companyID uint) ([]account.AccountMove, error) {
	var moves []account.AccountMove
	err := r.db.Preload("Partner").Preload("Journal").Preload("LineIDs.Account").
		Where("company_id = ? AND state = 'posted' AND payment_state != 'paid' AND amount_residual > 0", companyID).
		Order("invoice_date asc, id asc").
		Find(&moves).Error
	return moves, err
}

func (r *Repository) FindMoveByName(name string, companyID uint) (*account.AccountMove, error) {
	var move account.AccountMove
	err := r.db.Preload("Partner").Preload("Journal").Preload("LineIDs.Account").
		Where("name = ? AND company_id = ?", name, companyID).
		First(&move).Error
	if err != nil {
		return nil, err
	}
	return &move, nil
}

// Statistics

func (r *Repository) GetStats(companyID uint) (*BankReconciliationStats, error) {
	stats := &BankReconciliationStats{}

	// Statement counts
	r.db.Model(&AccountBankStatement{}).Where("company_id = ?", companyID).Count(&stats.TotalStatements)
	r.db.Model(&AccountBankStatement{}).Where("company_id = ? AND state = 'open'", companyID).Count(&stats.OpenStatements)
	r.db.Model(&AccountBankStatement{}).Where("company_id = ? AND state = 'confirm'", companyID).Count(&stats.ConfirmedStatements)

	// Line counts
	r.db.Model(&AccountBankStatementLine{}).Where("company_id = ?", companyID).Count(&stats.TotalLines)
	r.db.Model(&AccountBankStatementLine{}).Where("company_id = ? AND is_reconciled = true", companyID).Count(&stats.ReconciledLines)
	stats.UnreconciledLines = stats.TotalLines - stats.ReconciledLines
	if stats.TotalLines > 0 {
		stats.ReconciledPercentage = float64(stats.ReconciledLines) / float64(stats.TotalLines) * 100.0
	}

	// Inflows and Outflows
	type SumResult struct {
		Inflow     float64
		Outflow    float64
		Difference float64
	}
	var sumRes SumResult
	r.db.Model(&AccountBankStatementLine{}).
		Select("COALESCE(SUM(CASE WHEN amount > 0 THEN amount ELSE 0 END), 0) as inflow, COALESCE(SUM(CASE WHEN amount < 0 THEN ABS(amount) ELSE 0 END), 0) as outflow").
		Where("company_id = ?", companyID).
		Scan(&sumRes)

	stats.TotalInflow = sumRes.Inflow
	stats.TotalOutflow = sumRes.Outflow
	stats.NetMovement = sumRes.Inflow - sumRes.Outflow

	var diffSum float64
	r.db.Model(&AccountBankStatement{}).
		Select("COALESCE(SUM(difference), 0)").
		Where("company_id = ?", companyID).
		Scan(&diffSum)
	stats.DifferenceSum = diffSum

	return stats, nil
}
