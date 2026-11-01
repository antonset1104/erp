package mrp

import (
	"errors"
	"fmt"
	"time"

	"erp-golang/internal/modules/product"
	"erp-golang/internal/modules/stock"

	"gorm.io/gorm"
)

type Service struct {
	db       *gorm.DB
	stockSvc *stock.Service
}

func NewService(db *gorm.DB, stockSvc *stock.Service) *Service {
	return &Service{db: db, stockSvc: stockSvc}
}

func (s *Service) ListBoms(companyID uint) ([]MrpBom, error) {
	var boms []MrpBom
	if err := s.db.Preload("Product").Preload("Lines.Product").
		Where("company_id = ?", companyID).Find(&boms).Error; err != nil {
		return nil, err
	}
	return boms, nil
}

func (s *Service) CreateBom(bom *MrpBom) error {
	return s.db.Create(bom).Error
}

func (s *Service) ListProductions(companyID uint, state string) ([]MrpProduction, error) {
	var mos []MrpProduction
	query := s.db.Preload("Product").Preload("Bom.Lines.Product").Where("company_id = ?", companyID)
	if state != "" {
		query = query.Where("state = ?", state)
	}
	if err := query.Order("id desc").Find(&mos).Error; err != nil {
		return nil, err
	}
	return mos, nil
}

func (s *Service) CreateProduction(mo *MrpProduction) error {
	if mo.Name == "" {
		var count int64
		s.db.Model(&MrpProduction{}).Count(&count)
		mo.Name = fmt.Sprintf("MO/%s/%05d", time.Now().Format("2006"), count+1)
	}
	if mo.DatePlanned == "" {
		mo.DatePlanned = time.Now().Format("2006-01-02 15:04:05")
	}
	if mo.State == "" {
		mo.State = "confirmed"
	}
	return s.db.Create(mo).Error
}

// ActionProduce executes Odoo's manufacturing order production: consumes raw components and produces finished goods
func (s *Service) ActionProduce(id uint, companyID uint) (*MrpProduction, error) {
	var mo MrpProduction
	if err := s.db.Preload("Bom.Lines").Where("id = ? AND company_id = ?", id, companyID).First(&mo).Error; err != nil {
		return nil, errors.New("manufacturing order not found")
	}

	if mo.State == "done" {
		return nil, errors.New("manufacturing order is already completed")
	}
	if mo.State == "cancel" {
		return nil, errors.New("cannot produce a cancelled manufacturing order")
	}

	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 1. Consume raw components from WH/Stock (Location 1)
	for _, line := range mo.Bom.Lines {
		consumedQty := line.ProductQty * mo.ProductQty

		var quant stock.StockQuant
		err := tx.Where("product_id = ? AND location_id = 1", line.ProductID).First(&quant).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				quant = stock.StockQuant{
					ProductID:  line.ProductID,
					LocationID: 1,
					Quantity:   -consumedQty,
				}
				quant.CompanyID = companyID
				if err := tx.Create(&quant).Error; err != nil {
					tx.Rollback()
					return nil, err
				}
			} else {
				tx.Rollback()
				return nil, err
			}
		} else {
			quant.Quantity -= consumedQty
			if err := tx.Save(&quant).Error; err != nil {
				tx.Rollback()
				return nil, err
			}
		}

		// Update component product on hand
		var totalCompQty float64
		tx.Model(&stock.StockQuant{}).
			Joins("JOIN stock_locations ON stock_locations.id = stock_quants.location_id").
			Where("stock_quants.product_id = ? AND stock_locations.usage = 'internal'", line.ProductID).
			Select("COALESCE(SUM(quantity), 0)").
			Scan(&totalCompQty)

		tx.Model(&product.Product{}).Where("id = ?", line.ProductID).Update("qty_available", totalCompQty)
	}

	// 2. Increase finished product in WH/Stock (Location 1)
	var finishedQuant stock.StockQuant
	err := tx.Where("product_id = ? AND location_id = 1", mo.ProductID).First(&finishedQuant).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			finishedQuant = stock.StockQuant{
				ProductID:  mo.ProductID,
				LocationID: 1,
				Quantity:   mo.ProductQty,
			}
			finishedQuant.CompanyID = companyID
			if err := tx.Create(&finishedQuant).Error; err != nil {
				tx.Rollback()
				return nil, err
			}
		} else {
			tx.Rollback()
			return nil, err
		}
	} else {
		finishedQuant.Quantity += mo.ProductQty
		if err := tx.Save(&finishedQuant).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	// Update finished product on hand
	var totalFinQty float64
	tx.Model(&stock.StockQuant{}).
		Joins("JOIN stock_locations ON stock_locations.id = stock_quants.location_id").
		Where("stock_quants.product_id = ? AND stock_locations.usage = 'internal'", mo.ProductID).
		Select("COALESCE(SUM(quantity), 0)").
		Scan(&totalFinQty)

	tx.Model(&product.Product{}).Where("id = ?", mo.ProductID).Update("qty_available", totalFinQty)

	// 3. Mark MO as done
	mo.State = "done"
	mo.DateDone = time.Now().Format("2006-01-02 15:04:05")
	if err := tx.Save(&mo).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return &mo, nil
}
