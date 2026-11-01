package treasury

import (
	"fmt"
	"math"

	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// ============================================================================
// SEQUENCES
// ============================================================================

func (r *Repository) GenerateBankAccountCode(year int) (string, error) {
	var count int64
	if err := r.db.Model(&TreasuryBankAccount{}).Count(&count).Error; err != nil {
		return "", err
	}
	return fmt.Sprintf("BA/%d/%03d", year, count+1), nil
}

func (r *Repository) GenerateForecastCode(year int) (string, error) {
	var count int64
	if err := r.db.Model(&CashFlowForecast{}).Count(&count).Error; err != nil {
		return "", err
	}
	return fmt.Sprintf("CFF/%d/%05d", year, count+1), nil
}

func (r *Repository) GenerateTransferCode(year int) (string, error) {
	var count int64
	if err := r.db.Model(&InterBankTransfer{}).Count(&count).Error; err != nil {
		return "", err
	}
	return fmt.Sprintf("IBT/%d/%05d", year, count+1), nil
}

// ============================================================================
// BANK ACCOUNTS
// ============================================================================

func (r *Repository) CreateBankAccount(acc *TreasuryBankAccount) error {
	return r.db.Create(acc).Error
}

func (r *Repository) GetBankAccountByID(id uint, companyID uint) (*TreasuryBankAccount, error) {
	var acc TreasuryBankAccount
	query := r.db.Where("id = ?", id)
	if companyID > 0 {
		query = query.Where("company_id = ?", companyID)
	}
	if err := query.First(&acc).Error; err != nil {
		return nil, err
	}
	return &acc, nil
}

func (r *Repository) ListBankAccounts(companyID uint) ([]TreasuryBankAccount, error) {
	var accs []TreasuryBankAccount
	query := r.db.Order("is_primary_pool desc, bank_name asc")
	if companyID > 0 {
		query = query.Where("company_id = ?", companyID)
	}
	if err := query.Find(&accs).Error; err != nil {
		return nil, err
	}
	return accs, nil
}

func (r *Repository) UpdateBankAccountBalance(id uint, delta float64) error {
	return r.db.Model(&TreasuryBankAccount{}).
		Where("id = ?", id).
		Update("current_balance", gorm.Expr("current_balance + ?", delta)).Error
}

// ============================================================================
// CASH FLOW FORECASTS
// ============================================================================

func (r *Repository) CreateForecast(f *CashFlowForecast) error {
	return r.db.Create(f).Error
}

func (r *Repository) GetForecastByID(id uint, companyID uint) (*CashFlowForecast, error) {
	var f CashFlowForecast
	query := r.db.Preload("Items", func(db *gorm.DB) *gorm.DB {
		return db.Order("expected_date asc")
	}).Where("id = ?", id)
	if companyID > 0 {
		query = query.Where("company_id = ?", companyID)
	}
	if err := query.First(&f).Error; err != nil {
		return nil, err
	}
	return &f, nil
}

func (r *Repository) ListForecasts(companyID uint) ([]CashFlowForecast, error) {
	var forecasts []CashFlowForecast
	query := r.db.Order("created_at desc")
	if companyID > 0 {
		query = query.Where("company_id = ?", companyID)
	}
	if err := query.Find(&forecasts).Error; err != nil {
		return nil, err
	}
	return forecasts, nil
}

func (r *Repository) CreateForecastItems(items []CashFlowForecastItem) error {
	if len(items) == 0 {
		return nil
	}
	return r.db.Create(&items).Error
}

func (r *Repository) GetForecastItems(forecastID uint) ([]CashFlowForecastItem, error) {
	var items []CashFlowForecastItem
	if err := r.db.Where("forecast_id = ?", forecastID).Order("expected_date asc").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// ============================================================================
// INTER-BANK TRANSFERS
// ============================================================================

func (r *Repository) CreateTransfer(t *InterBankTransfer) error {
	return r.db.Create(t).Error
}

func (r *Repository) GetTransferByID(id uint, companyID uint) (*InterBankTransfer, error) {
	var t InterBankTransfer
	query := r.db.Where("id = ?", id)
	if companyID > 0 {
		query = query.Where("company_id = ?", companyID)
	}
	if err := query.First(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *Repository) ListTransfers(companyID uint) ([]InterBankTransfer, error) {
	var transfers []InterBankTransfer
	query := r.db.Order("transfer_date desc")
	if companyID > 0 {
		query = query.Where("company_id = ?", companyID)
	}
	if err := query.Find(&transfers).Error; err != nil {
		return nil, err
	}
	return transfers, nil
}

func (r *Repository) UpdateTransferState(id uint, state string, moveID uint, moveName string) error {
	updates := map[string]interface{}{
		"state": state,
	}
	if moveID > 0 {
		updates["journal_move_id"] = moveID
		updates["journal_move_name"] = moveName
	}
	return r.db.Model(&InterBankTransfer{}).Where("id = ?", id).Updates(updates).Error
}

// ============================================================================
// AGGREGATE STATS & DASHBOARD
// ============================================================================

func (r *Repository) GetStats(companyID uint) (*TreasuryStats, error) {
	stats := &TreasuryStats{
		PrimaryLiquidityRisk: RiskNone,
	}

	// 1. Total Liquid Cash across active bank accounts
	var accs []TreasuryBankAccount
	accQuery := r.db.Where("status = ?", "active")
	if companyID > 0 {
		accQuery = accQuery.Where("company_id = ?", companyID)
	}
	if err := accQuery.Find(&accs).Error; err != nil {
		return nil, err
	}

	var totalMinBalance float64
	for _, a := range accs {
		stats.TotalLiquidCash += a.CurrentBalance
		totalMinBalance += a.MinimumBalance
	}
	stats.TotalActiveAccounts = int64(len(accs))

	// 2. Latest active 30-day forecast
	var latestForecast CashFlowForecast
	fcQuery := r.db.Where("status = ?", "active").Order("created_at desc")
	if companyID > 0 {
		fcQuery = fcQuery.Where("company_id = ?", companyID)
	}
	if err := fcQuery.First(&latestForecast).Error; err == nil {
		stats.Projected30DInflow = latestForecast.ProjectedInflow
		stats.Projected30DOutflow = latestForecast.ProjectedOutflow
		stats.Projected30DNetCash = latestForecast.ProjectedNetCash
		stats.PrimaryLiquidityRisk = latestForecast.DeficitRisk
		stats.RunwayDays = latestForecast.EstimatedRunwayDays
	} else {
		// Default estimation if no forecast exists yet
		dailyBurn := 500000.0 // Default burn assumption if none
		if stats.TotalLiquidCash > 0 {
			stats.RunwayDays = int(stats.TotalLiquidCash / dailyBurn)
		}
	}

	// Safety buffer coverage percentage
	if totalMinBalance > 0 {
		stats.SafetyBufferCoverage = math.Round((stats.TotalLiquidCash/totalMinBalance)*1000) / 10
	} else {
		stats.SafetyBufferCoverage = 100.0
	}

	// 3. Inter-Bank Transfers volume
	var txs []InterBankTransfer
	txQuery := r.db.Model(&InterBankTransfer{})
	if companyID > 0 {
		txQuery = txQuery.Where("company_id = ?", companyID)
	}
	txQuery.Find(&txs)
	for _, t := range txs {
		if t.State == "posted" {
			stats.TotalTransfersVolume += t.Amount
		} else if t.State == "draft" {
			stats.PendingTransfersCount++
		}
	}

	return stats, nil
}
