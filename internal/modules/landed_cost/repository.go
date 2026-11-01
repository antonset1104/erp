package landed_cost

import (
	"erp-golang/internal/modules/stock"
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(cost *StockLandedCost) error {
	return r.db.Create(cost).Error
}

func (r *Repository) GetByID(id uint, companyID uint) (*StockLandedCost, error) {
	var cost StockLandedCost
	err := r.db.Preload("CostLines").
		Preload("ValuationLines").
		Preload("Picking").
		Preload("Picking.MoveLines").
		Preload("Picking.MoveLines.Product").
		Preload("AccountMove").
		Preload("AccountMove.LineIDs").
		Where("id = ? AND company_id = ?", id, companyID).
		First(&cost).Error
	if err != nil {
		return nil, err
	}
	return &cost, nil
}

func (r *Repository) List(companyID uint, state string) ([]StockLandedCost, error) {
	var costs []StockLandedCost
	query := r.db.Preload("CostLines").
		Preload("ValuationLines").
		Preload("Picking").
		Where("company_id = ?", companyID)
	if state != "" {
		query = query.Where("state = ?", state)
	}
	err := query.Order("id DESC").Find(&costs).Error
	return costs, err
}

func (r *Repository) Update(cost *StockLandedCost) error {
	return r.db.Save(cost).Error
}

func (r *Repository) Delete(id uint, companyID uint) error {
	return r.db.Where("id = ? AND company_id = ? AND state = ?", id, companyID, StateDraft).
		Delete(&StockLandedCost{}).Error
}

func (r *Repository) GetIncomingPicking(pickingID uint, companyID uint) (*stock.StockPicking, error) {
	var picking stock.StockPicking
	err := r.db.Preload("MoveLines").
		Preload("MoveLines.Product").
		Where("id = ? AND company_id = ?", pickingID, companyID).
		First(&picking).Error
	if err != nil {
		return nil, err
	}
	return &picking, nil
}

func (r *Repository) SaveValuationLines(costID uint, lines []StockValuationAdjustmentLine) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Clear existing valuation lines for this landed cost
		if err := tx.Where("cost_id = ?", costID).Delete(&StockValuationAdjustmentLine{}).Error; err != nil {
			return err
		}
		if len(lines) > 0 {
			if err := tx.Create(&lines).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *Repository) GetStats(companyID uint) (*LandedCostStats, error) {
	var stats LandedCostStats
	r.db.Model(&StockLandedCost{}).Where("company_id = ?", companyID).Count(&stats.TotalLandedCosts)
	r.db.Model(&StockLandedCost{}).Where("company_id = ? AND state = ?", companyID, StateDraft).Count(&stats.DraftCount)
	r.db.Model(&StockLandedCost{}).Where("company_id = ? AND state = ?", companyID, StateDone).Count(&stats.DoneCount)

	type SumResult struct {
		Total float64
	}
	var sumRes SumResult
	r.db.Model(&StockLandedCost{}).Where("company_id = ? AND state = ?", companyID, StateDone).
		Select("COALESCE(SUM(total_amount), 0) as total").Scan(&sumRes)
	stats.TotalCostAllocated = sumRes.Total

	// Calculate average overhead ratio
	var valSum struct {
		TotalFormer float64
		TotalAdd    float64
	}
	r.db.Model(&StockValuationAdjustmentLine{}).
		Joins("JOIN stock_landed_costs ON stock_landed_costs.id = stock_valuation_adjustment_lines.cost_id").
		Where("stock_landed_costs.company_id = ? AND stock_landed_costs.state = ?", companyID, StateDone).
		Select("COALESCE(SUM(former_cost), 0) as total_former, COALESCE(SUM(additional_landed_cost), 0) as total_add").
		Scan(&valSum)

	if valSum.TotalFormer > 0 {
		stats.AverageCostRatio = (valSum.TotalAdd / valSum.TotalFormer) * 100.0
	}

	return &stats, nil
}
