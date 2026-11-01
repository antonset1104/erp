package stock_inventory

import (
	"erp-golang/internal/modules/account"
	"erp-golang/internal/modules/stock"

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

func (r *Repository) CreateInventory(inv *StockInventory) error {
	return r.db.Create(inv).Error
}

func (r *Repository) UpdateInventory(inv *StockInventory) error {
	// Clear associations to prevent GORM association overwrite bugs
	inv.Warehouse = nil
	inv.Location = nil
	return r.db.Save(inv).Error
}

func (r *Repository) GetInventoryByID(id uint, companyID uint) (*StockInventory, error) {
	var inv StockInventory
	err := r.db.Preload("Warehouse").
		Preload("Location").
		Preload("Lines.Product").
		Where("id = ? AND company_id = ?", id, companyID).
		First(&inv).Error
	if err != nil {
		return nil, err
	}
	return &inv, nil
}

func (r *Repository) ListInventories(companyID uint, state string, warehouseID uint, locationID uint, search string) ([]StockInventory, error) {
	var list []StockInventory
	query := r.db.Preload("Warehouse").
		Preload("Location").
		Preload("Lines.Product").
		Where("company_id = ?", companyID)

	if state != "" {
		query = query.Where("state = ?", state)
	}
	if warehouseID > 0 {
		query = query.Where("warehouse_id = ?", warehouseID)
	}
	if locationID > 0 {
		query = query.Where("location_id = ?", locationID)
	}
	if search != "" {
		s := "%" + search + "%"
		query = query.Where("name LIKE ? OR notes LIKE ?", s, s)
	}

	err := query.Order("id desc").Find(&list).Error
	return list, err
}

func (r *Repository) CreateLine(line *StockInventoryLine) error {
	return r.db.Create(line).Error
}

func (r *Repository) UpdateLine(line *StockInventoryLine) error {
	line.Product = nil
	return r.db.Save(line).Error
}

func (r *Repository) GetLineByID(id uint) (*StockInventoryLine, error) {
	var line StockInventoryLine
	err := r.db.Preload("Product").Where("id = ?", id).First(&line).Error
	if err != nil {
		return nil, err
	}
	return &line, nil
}

func (r *Repository) DeleteLine(id uint) error {
	return r.db.Delete(&StockInventoryLine{}, id).Error
}

func (r *Repository) GetQuantsByLocation(locationID uint, companyID uint) ([]stock.StockQuant, error) {
	var quants []stock.StockQuant
	err := r.db.Preload("Product").
		Where("location_id = ? AND company_id = ?", locationID, companyID).
		Find(&quants).Error
	return quants, err
}

func (r *Repository) GetLocationByName(name string, companyID uint) (*stock.StockLocation, error) {
	var loc stock.StockLocation
	err := r.db.Where("name = ? AND company_id = ?", name, companyID).First(&loc).Error
	if err != nil {
		return nil, err
	}
	return &loc, nil
}

func (r *Repository) GetLocationByID(id uint, companyID uint) (*stock.StockLocation, error) {
	var loc stock.StockLocation
	err := r.db.Where("id = ? AND company_id = ?", id, companyID).First(&loc).Error
	if err != nil {
		return nil, err
	}
	return &loc, nil
}

func (r *Repository) GetAccountByCode(code string, companyID uint) (*account.AccountAccount, error) {
	var acc account.AccountAccount
	err := r.db.Where("code = ? AND company_id = ?", code, companyID).First(&acc).Error
	if err != nil {
		return nil, err
	}
	return &acc, nil
}

func (r *Repository) GetJournalByCode(code string, companyID uint) (*account.AccountJournal, error) {
	var jrn account.AccountJournal
	err := r.db.Where("code = ? AND company_id = ?", code, companyID).First(&jrn).Error
	if err != nil {
		return nil, err
	}
	return &jrn, nil
}

func (r *Repository) GetStats(companyID uint) (*InventoryStats, error) {
	stats := &InventoryStats{}

	r.db.Model(&StockInventory{}).Where("company_id = ?", companyID).Count(&stats.TotalInventories)
	r.db.Model(&StockInventory{}).Where("company_id = ? AND state = 'draft'", companyID).Count(&stats.DraftCount)
	r.db.Model(&StockInventory{}).Where("company_id = ? AND state = 'confirm'", companyID).Count(&stats.ConfirmedCount)
	r.db.Model(&StockInventory{}).Where("company_id = ? AND state = 'done'", companyID).Count(&stats.DoneCount)
	r.db.Model(&StockInventory{}).Where("company_id = ? AND ai_risk_level = 'high'", companyID).Count(&stats.HighRiskAuditCount)

	type SumResult struct {
		TotalDiff float64
		TotalLoss float64
		TotalGain float64
	}
	var res SumResult
	r.db.Model(&StockInventory{}).
		Where("company_id = ? AND state = 'done'", companyID).
		Select("COALESCE(SUM(total_difference_qty), 0) as total_diff, COALESCE(SUM(total_loss_value), 0) as total_loss, COALESCE(SUM(total_gain_value), 0) as total_gain").
		Scan(&res)

	stats.TotalDifferenceUnits = res.TotalDiff
	stats.TotalLossValue = res.TotalLoss
	stats.TotalGainValue = res.TotalGain

	return stats, nil
}
