package stock

import (
	"errors"
	"fmt"
	"time"

	"erp-golang/internal/modules/product"

	"gorm.io/gorm"
)

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) ListPickings(companyID uint, pickingType string) ([]StockPicking, error) {
	var pickings []StockPicking
	query := s.db.Preload("MoveLines").Preload("Location").Preload("LocationDest").
		Where("company_id = ?", companyID)

	if pickingType != "" {
		query = query.Where("picking_type = ?", pickingType)
	}

	if err := query.Order("id desc").Find(&pickings).Error; err != nil {
		return nil, err
	}
	return pickings, nil
}

func (s *Service) GetPicking(id uint, companyID uint) (*StockPicking, error) {
	var p StockPicking
	if err := s.db.Preload("MoveLines.Product").Preload("Location").Preload("LocationDest").
		Where("id = ? AND company_id = ?", id, companyID).First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *Service) CreatePicking(p *StockPicking) error {
	// Auto generate sequence if empty
	if p.Name == "" {
		var count int64
		s.db.Model(&StockPicking{}).Where("picking_type = ?", p.PickingType).Count(&count)
		prefix := "WH/IN"
		if p.PickingType == "outgoing" {
			prefix = "WH/OUT"
		} else if p.PickingType == "internal" {
			prefix = "WH/INT"
		}
		p.Name = fmt.Sprintf("%s/%05d", prefix, count+1)
		for {
			var exists int64
			s.db.Model(&StockPicking{}).Where("name = ?", p.Name).Count(&exists)
			if exists == 0 {
				break
			}
			count++
			p.Name = fmt.Sprintf("%s/%05d", prefix, count+1)
		}
	}

	if p.ScheduledDate == "" {
		p.ScheduledDate = time.Now().Format("2006-01-02 15:04:05")
	}
	if p.State == "" {
		p.State = "assigned" // Ready
	}

	return s.db.Create(p).Error
}

// ValidatePicking executes Odoo's double-entry stock movement atomically
func (s *Service) ValidatePicking(id uint, companyID uint) (*StockPicking, error) {
	var picking StockPicking
	if err := s.db.Preload("MoveLines").Preload("Location").Preload("LocationDest").
		Where("id = ? AND company_id = ?", id, companyID).First(&picking).Error; err != nil {
		return nil, errors.New("picking not found")
	}

	if picking.State == "done" {
		return nil, errors.New("picking is already validated and done")
	}
	if picking.State == "cancel" {
		return nil, errors.New("cannot validate a cancelled picking")
	}

	// Begin atomic transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	nowStr := time.Now().Format("2006-01-02 15:04:05")

	for i := range picking.MoveLines {
		move := &picking.MoveLines[i]
		qty := move.ProductUomQty
		if move.QuantityDone > 0 {
			qty = move.QuantityDone
		} else {
			move.QuantityDone = qty
		}

		// 1. If Source Location is Internal: Deduct from Stock Quant
		if picking.Location.Usage == "internal" {
			var srcQuant StockQuant
			err := tx.Where("product_id = ? AND location_id = ?", move.ProductID, picking.LocationID).
				First(&srcQuant).Error
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					srcQuant = StockQuant{
						ProductID:  move.ProductID,
						LocationID: picking.LocationID,
						Quantity:   -qty,
					}
					srcQuant.CompanyID = companyID
					if err := tx.Create(&srcQuant).Error; err != nil {
						tx.Rollback()
						return nil, err
					}
				} else {
					tx.Rollback()
					return nil, err
				}
			} else {
				srcQuant.Quantity -= qty
				if err := tx.Save(&srcQuant).Error; err != nil {
					tx.Rollback()
					return nil, err
				}
			}
		}

		// 2. If Destination Location is Internal: Add to Stock Quant
		if picking.LocationDest.Usage == "internal" {
			var destQuant StockQuant
			err := tx.Where("product_id = ? AND location_id = ?", move.ProductID, picking.LocationDestID).
				First(&destQuant).Error
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					destQuant = StockQuant{
						ProductID:  move.ProductID,
						LocationID: picking.LocationDestID,
						Quantity:   qty,
					}
					destQuant.CompanyID = companyID
					if err := tx.Create(&destQuant).Error; err != nil {
						tx.Rollback()
						return nil, err
					}
				} else {
					tx.Rollback()
					return nil, err
				}
			} else {
				destQuant.Quantity += qty
				if err := tx.Save(&destQuant).Error; err != nil {
					tx.Rollback()
					return nil, err
				}
			}
		}

		// 3. Update move status to done
		move.State = "done"
		if err := tx.Save(move).Error; err != nil {
			tx.Rollback()
			return nil, err
		}

		// 4. Update Product total On Hand Quantity
		var totalQty float64
		// Sum all quants where location usage is internal
		tx.Model(&StockQuant{}).
			Joins("JOIN stock_locations ON stock_locations.id = stock_quants.location_id").
			Where("stock_quants.product_id = ? AND stock_locations.usage = 'internal'", move.ProductID).
			Select("COALESCE(SUM(quantity), 0)").
			Scan(&totalQty)

		if err := tx.Model(&product.Product{}).Where("id = ?", move.ProductID).
			Update("qty_available", totalQty).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	// Mark picking done
	picking.State = "done"
	picking.DateDone = nowStr
	if err := tx.Save(&picking).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return &picking, nil
}

func (s *Service) ListQuants(companyID uint) ([]StockQuant, error) {
	var quants []StockQuant
	if err := s.db.Preload("Product").Preload("Location.Warehouse").
		Where("company_id = ?", companyID).
		Order("product_id asc, location_id asc").
		Find(&quants).Error; err != nil {
		return nil, err
	}
	return quants, nil
}

func (s *Service) ListLocations(companyID uint) ([]StockLocation, error) {
	var locs []StockLocation
	if err := s.db.Preload("Warehouse").Where("company_id = ?", companyID).Order("id asc").Find(&locs).Error; err != nil {
		return nil, err
	}
	return locs, nil
}

func (s *Service) ListWarehouses(companyID uint) ([]StockWarehouse, error) {
	var warehouses []StockWarehouse
	if err := s.db.Where("company_id = ?", companyID).Preload("LotStock").Order("id asc").Find(&warehouses).Error; err != nil {
		return nil, err
	}
	return warehouses, nil
}

func (s *Service) GetWarehouse(id, companyID uint) (*StockWarehouse, error) {
	var w StockWarehouse
	if err := s.db.Where("id = ? AND company_id = ?", id, companyID).Preload("LotStock").First(&w).Error; err != nil {
		return nil, err
	}
	return &w, nil
}

func (s *Service) CreateWarehouse(w *StockWarehouse) error {
	if w.Name == "" || w.Code == "" {
		return errors.New("warehouse name and code are required")
	}

	// If LotStockID is not provided, create default internal location
	if w.LotStockID == 0 {
		loc := StockLocation{
			Name:         fmt.Sprintf("%s/Stock", w.Code),
			Usage:        "internal",
			CompleteName: fmt.Sprintf("%s/Stock (%s)", w.Code, w.Name),
		}
		loc.CompanyID = w.CompanyID
		if err := s.db.Create(&loc).Error; err != nil {
			return err
		}
		w.LotStockID = loc.ID
	}

	if err := s.db.Create(w).Error; err != nil {
		return err
	}

	// Link location to this warehouse
	s.db.Model(&StockLocation{}).Where("id = ?", w.LotStockID).Update("warehouse_id", w.ID)
	return nil
}

func (s *Service) CreateInternalTransfer(companyID, userID uint, srcLocID, destLocID uint, moves []StockMove, origin, note string) (*StockPicking, error) {
	if srcLocID == 0 || destLocID == 0 {
		return nil, errors.New("source and destination locations are required")
	}
	if srcLocID == destLocID {
		return nil, errors.New("source and destination locations must be different")
	}
	if len(moves) == 0 {
		return nil, errors.New("at least one product line is required")
	}

	picking := StockPicking{
		PickingType:    "internal",
		LocationID:     srcLocID,
		LocationDestID: destLocID,
		Origin:         origin,
		Note:           note,
		State:          "assigned",
		ScheduledDate:  time.Now().Format("2006-01-02 15:04:05"),
	}
	picking.CompanyID = companyID
	picking.CreateUID = userID
	picking.WriteUID = userID

	for i := range moves {
		moves[i].CompanyID = companyID
		moves[i].CreateUID = userID
		moves[i].WriteUID = userID
		moves[i].LocationID = srcLocID
		moves[i].LocationDestID = destLocID
		moves[i].State = "draft"
	}
	picking.MoveLines = moves

	if err := s.CreatePicking(&picking); err != nil {
		return nil, err
	}

	return s.GetPicking(picking.ID, companyID)
}

func (s *Service) ListOrderpoints(companyID uint) ([]StockOrderpoint, error) {
	var ops []StockOrderpoint
	if err := s.db.Preload("Product").Preload("Location").
		Where("company_id = ?", companyID).Find(&ops).Error; err != nil {
		return nil, err
	}
	return ops, nil
}

func (s *Service) CreateOrderpoint(op *StockOrderpoint) error {
	if op.Name == "" {
		var count int64
		s.db.Model(&StockOrderpoint{}).Count(&count)
		op.Name = fmt.Sprintf("OP/%05d", count+1)
	}
	if op.QtyMultiple <= 0 {
		op.QtyMultiple = 1
	}
	if op.Trigger == "" {
		op.Trigger = "auto"
	}
	return s.db.Create(op).Error
}

