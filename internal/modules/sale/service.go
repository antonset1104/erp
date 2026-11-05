package sale

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

func (s *Service) ListOrders(companyID uint, state string) ([]SaleOrder, error) {
	var orders []SaleOrder
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

func (s *Service) GetOrder(id uint, companyID uint) (*SaleOrder, error) {
	var order SaleOrder
	if err := s.db.Preload("Partner").Preload("OrderLines.Product").
		Where("id = ? AND company_id = ?", id, companyID).First(&order).Error; err != nil {
		return nil, err
	}
	return &order, nil
}

func (s *Service) CreateOrder(order *SaleOrder) error {
	// Auto generate order sequence SO0001
	if order.Name == "" {
		var count int64
		s.db.Model(&SaleOrder{}).Count(&count)
		order.Name = fmt.Sprintf("SO%05d", count+1)
		for {
			var exists int64
			s.db.Model(&SaleOrder{}).Where("name = ?", order.Name).Count(&exists)
			if exists == 0 {
				break
			}
			count++
			order.Name = fmt.Sprintf("SO%05d", count+1)
		}
	}

	if order.DateOrder == "" {
		order.DateOrder = time.Now().Format("2006-01-02 15:04:05")
	}
	if order.State == "" {
		order.State = "draft"
	}

	// Calculate totals
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

// ActionConfirm executes standard enterprise action_confirm: confirm quotation -> Sales Order + auto-generate Delivery Order
func (s *Service) ActionConfirm(id uint, companyID uint) (*SaleOrder, *stock.StockPicking, error) {
	order, err := s.GetOrder(id, companyID)
	if err != nil {
		return nil, nil, errors.New("sales order not found")
	}

	if order.State == "sale" {
		return nil, nil, errors.New("sales order is already confirmed")
	}
	if order.State == "cancel" {
		return nil, nil, errors.New("cannot confirm a cancelled order")
	}

	// Generate Stock Delivery Order (WH/Stock -> Customers) for storable products
	var moveLines []stock.StockMove
	for _, line := range order.OrderLines {
		var prod product.Product
		if err := s.db.First(&prod, line.ProductID).Error; err == nil {
			if prod.Type == "product" { // Storable
				move := stock.StockMove{
					ProductID:      line.ProductID,
					ProductName:    line.Name,
					ProductUomQty:  line.ProductUomQty,
					LocationID:     1, // WH/Stock
					LocationDestID: 3, // Virtual Locations/Customers
					State:          "confirmed",
				}
				move.CompanyID = companyID
				moveLines = append(moveLines, move)
			}
		}
	}

	var delivery *stock.StockPicking
	if len(moveLines) > 0 {
		delivery = &stock.StockPicking{
			Origin:         order.Name,
			PickingType:    "outgoing",
			PartnerID:      &order.PartnerID,
			PartnerName:    order.Partner.Name,
			LocationID:     1, // WH/Stock
			LocationDestID: 3, // Customers
			State:          "assigned",
			ScheduledDate:  time.Now().Format("2006-01-02 15:04:05"),
			MoveLines:      moveLines,
			Note:           fmt.Sprintf("Delivery Order for %s", order.Name),
		}
		delivery.CompanyID = companyID

		if err := s.stockSvc.CreatePicking(delivery); err != nil {
			return nil, nil, fmt.Errorf("failed to create stock delivery order: %w", err)
		}
		order.DeliveryID = &delivery.ID
	}

	order.State = "sale"
	if err := s.db.Save(order).Error; err != nil {
		return nil, nil, err
	}

	return order, delivery, nil
}
