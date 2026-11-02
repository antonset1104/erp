package purchase

import (
	"errors"
	"fmt"
	"math"
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

func (s *Service) ListOrders(companyID uint, state string) ([]PurchaseOrder, error) {
	var orders []PurchaseOrder
	query := s.db.Preload("Partner").Preload("OrderLines.Product").
		Where("company_id = ?", companyID)

	if state != "" {
		query = query.Where("state = ?", state)
	}

	if err := query.Order("id desc").Find(&orders).Error; err != nil {
		return nil, err
	}
	return orders, nil
}

func (s *Service) GetOrder(id uint, companyID uint) (*PurchaseOrder, error) {
	var order PurchaseOrder
	if err := s.db.Preload("Partner").Preload("OrderLines.Product").
		Where("id = ? AND company_id = ?", id, companyID).First(&order).Error; err != nil {
		return nil, err
	}
	return &order, nil
}

func (s *Service) CreateOrder(order *PurchaseOrder) error {
	if order.Name == "" {
		var count int64
		s.db.Model(&PurchaseOrder{}).Count(&count)
		order.Name = fmt.Sprintf("PO%05d", count+1)
		for {
			var exists int64
			s.db.Model(&PurchaseOrder{}).Where("name = ?", order.Name).Count(&exists)
			if exists == 0 {
				break
			}
			count++
			order.Name = fmt.Sprintf("PO%05d", count+1)
		}
	}

	if order.DateOrder == "" {
		order.DateOrder = time.Now().Format("2006-01-02 15:04:05")
	}
	if order.State == "" {
		order.State = "draft"
	}

	var untaxed float64
	for i := range order.OrderLines {
		line := &order.OrderLines[i]
		line.PriceSubtotal = line.ProductUomQty * line.PriceUnit
		taxAmount := line.PriceSubtotal * (order.TaxRate / 100.0)
		line.PriceTotal = line.PriceSubtotal + taxAmount
		untaxed += line.PriceSubtotal
	}

	order.AmountUntaxed = untaxed
	order.AmountTax = untaxed * (order.TaxRate / 100.0)
	order.AmountTotal = order.AmountUntaxed + order.AmountTax

	return s.db.Create(order).Error
}

// ActionConfirm executes Odoo's button_confirm: confirm RFQ -> Purchase Order + auto-generate Stock Receipt
func (s *Service) ActionConfirm(id uint, companyID uint) (*PurchaseOrder, *stock.StockPicking, error) {
	order, err := s.GetOrder(id, companyID)
	if err != nil {
		return nil, nil, errors.New("purchase order not found")
	}

	if order.State == "purchase" {
		return nil, nil, errors.New("purchase order is already confirmed")
	}
	if order.State == "cancel" {
		return nil, nil, errors.New("cannot confirm a cancelled purchase order")
	}

	// Generate Stock Receipt (Vendors -> WH/Stock) for storable products
	var moveLines []stock.StockMove
	for _, line := range order.OrderLines {
		var prod product.Product
		if err := s.db.First(&prod, line.ProductID).Error; err == nil {
			if prod.Type == "product" { // Storable
				move := stock.StockMove{
					ProductID:      line.ProductID,
					ProductName:    line.Name,
					ProductUomQty:  line.ProductUomQty,
					LocationID:     2, // Virtual Locations/Vendors
					LocationDestID: 1, // WH/Stock
					State:          "confirmed",
				}
				move.CompanyID = companyID
				moveLines = append(moveLines, move)
			}
		}
	}

	var receipt *stock.StockPicking
	if len(moveLines) > 0 {
		receipt = &stock.StockPicking{
			Origin:         order.Name,
			PickingType:    "incoming",
			PartnerID:      &order.PartnerID,
			PartnerName:    order.Partner.Name,
			LocationID:     2, // Vendors
			LocationDestID: 1, // WH/Stock
			State:          "assigned",
			ScheduledDate:  time.Now().Format("2006-01-02 15:04:05"),
			MoveLines:      moveLines,
			Note:           fmt.Sprintf("Receipt for %s", order.Name),
		}
		receipt.CompanyID = companyID

		if err := s.stockSvc.CreatePicking(receipt); err != nil {
			return nil, nil, fmt.Errorf("failed to create stock receipt: %w", err)
		}
		order.ReceiptID = &receipt.ID
	}

	order.State = "purchase"
	if err := s.db.Save(order).Error; err != nil {
		return nil, nil, err
	}

	return order, receipt, nil
}

// RunReorderingRules scans stock levels against min/max orderpoints and generates draft RFQs
func (s *Service) RunReorderingRules(companyID uint) ([]PurchaseOrder, error) {
	var orderpoints []stock.StockOrderpoint
	if err := s.db.Preload("Product").Where("company_id = ? AND trigger = 'auto'", companyID).Find(&orderpoints).Error; err != nil {
		return nil, err
	}

	var createdPOs []PurchaseOrder
	year := time.Now().Format("2006")

	for _, op := range orderpoints {
		var quant stock.StockQuant
		err := s.db.Where("company_id = ? AND product_id = ? AND location_id = ?", companyID, op.ProductID, op.LocationID).First(&quant).Error
		currentQty := 0.0
		if err == nil {
			currentQty = quant.Quantity
		}

		if currentQty < op.ProductMinQty {
			replenishQty := op.ProductMaxQty - currentQty
			if op.QtyMultiple > 1 {
				replenishQty = math.Ceil(replenishQty/op.QtyMultiple) * op.QtyMultiple
			}
			if replenishQty <= 0 {
				continue
			}

			// Avoid duplicate draft orders for this orderpoint
			var existingDrafts int64
			ruleRef := fmt.Sprintf("Rule: %s", op.Name)
			s.db.Model(&PurchaseOrder{}).
				Where("company_id = ? AND state = 'draft' AND note LIKE ?", companyID, "%"+ruleRef+"%").
				Count(&existingDrafts)
			if existingDrafts > 0 {
				continue
			}

			var count int64
			s.db.Model(&PurchaseOrder{}).Count(&count)
			poName := fmt.Sprintf("PO/%s/%05d", year, count+1)

			vendorID := uint(3) // PT Sumber Elektronik Maju (vendor)
			priceUnit := op.Product.StandardPrice
			if priceUnit <= 0 {
				priceUnit = 100000
			}
			subtotal := replenishQty * priceUnit
			tax := subtotal * 0.11
			total := subtotal + tax

			po := PurchaseOrder{
				Name:          poName,
				PartnerID:     vendorID,
				DateOrder:     time.Now().Format("2006-01-02 15:04:05"),
				State:         "draft",
				AmountUntaxed: subtotal,
				AmountTax:     tax,
				AmountTotal:   total,
				TaxRate:       11.0,
				Note:          fmt.Sprintf("Auto-procured by Reordering Rule: %s (Stock: %.0f < Min: %.0f, Replenish: %.0f)", op.Name, currentQty, op.ProductMinQty, replenishQty),
				OrderLines: []PurchaseOrderLine{
					{
						ProductID:     op.ProductID,
						Name:          fmt.Sprintf("[Auto-Procure] %s", op.Product.Name),
						ProductUomQty: replenishQty,
						PriceUnit:     priceUnit,
						PriceSubtotal: subtotal,
						PriceTotal:    total,
					},
				},
			}
			po.CompanyID = companyID

			if err := s.CreateOrder(&po); err == nil {
				createdPOs = append(createdPOs, po)
			}
		}
	}

	return createdPOs, nil
}

