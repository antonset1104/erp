package pos

import (
	"errors"
	"fmt"
	"time"

	"erp-golang/internal/modules/product"
	"erp-golang/internal/modules/stock"

	"gorm.io/gorm"
)

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) ListConfigs(companyID uint) ([]PosConfig, error) {
	var configs []PosConfig
	if err := s.db.Where("company_id = ?", companyID).Find(&configs).Error; err != nil {
		return nil, err
	}
	return configs, nil
}

func (s *Service) ListSessions(companyID uint) ([]PosSession, error) {
	var sessions []PosSession
	if err := s.db.Preload("Config").Preload("User").
		Where("company_id = ?", companyID).Order("id desc").Find(&sessions).Error; err != nil {
		return nil, err
	}
	return sessions, nil
}

func (s *Service) OpenSession(configID uint, userID uint, openingCash float64, companyID uint) (*PosSession, error) {
	var count int64
	s.db.Model(&PosSession{}).Count(&count)
	name := fmt.Sprintf("POS/SESS/%05d", count+1)

	session := &PosSession{
		Name:                     name,
		ConfigID:                 configID,
		UserID:                   userID,
		State:                    "opened",
		StartAt:                  time.Now().Format("2006-01-02 15:04:05"),
		CashRegisterBalanceStart: openingCash,
	}
	session.CompanyID = companyID
	if err := s.db.Create(session).Error; err != nil {
		return nil, err
	}
	return session, nil
}

func (s *Service) CloseSession(sessionID uint, closingCash float64, companyID uint) (*PosSession, error) {
	var session PosSession
	if err := s.db.Where("id = ? AND company_id = ?", sessionID, companyID).First(&session).Error; err != nil {
		return nil, errors.New("pos session not found")
	}
	if session.State == "closed" {
		return nil, errors.New("session already closed")
	}

	session.State = "closed"
	session.StopAt = time.Now().Format("2006-01-02 15:04:05")
	session.CashRegisterBalanceEnd = closingCash

	if err := s.db.Save(&session).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

func (s *Service) ListOrders(companyID uint) ([]PosOrder, error) {
	var orders []PosOrder
	if err := s.db.Preload("Partner").Preload("Lines.Product").
		Where("company_id = ?", companyID).Order("id desc").Find(&orders).Error; err != nil {
		return nil, err
	}
	return orders, nil
}

// CreateOrder processes an instant POS checkout transaction and reduces physical warehouse inventory immediately
func (s *Service) CreateOrder(order *PosOrder, companyID uint) (*PosOrder, error) {
	var session PosSession
	if err := s.db.Where("id = ? AND state = 'opened'", order.SessionID).First(&session).Error; err != nil {
		return nil, errors.New("active opened POS session is required to checkout")
	}

	if order.Name == "" {
		var count int64
		s.db.Model(&PosOrder{}).Count(&count)
		order.Name = fmt.Sprintf("POS/%s/%05d", time.Now().Format("2006"), count+1)
	}
	if order.DateOrder == "" {
		order.DateOrder = time.Now().Format("2006-01-02 15:04:05")
	}
	order.State = "paid"
	order.CompanyID = companyID

	// Calculate totals and lines
	var total float64
	for i := range order.Lines {
		line := &order.Lines[i]
		line.PriceSubtotal = line.Qty * line.PriceUnit
		total += line.PriceSubtotal
		line.CompanyID = companyID
	}
	order.AmountTotal = total
	order.AmountChange = order.AmountPaid - order.AmountTotal
	if order.AmountChange < 0 {
		return nil, errors.New("amount paid is less than order total")
	}

	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Create(order).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// Instant Stock Deduction from Location 1 (WH/Stock)
	for _, line := range order.Lines {
		var quant stock.StockQuant
		err := tx.Where("product_id = ? AND location_id = 1", line.ProductID).First(&quant).Error
		if err == nil {
			quant.Quantity -= line.Qty
			tx.Save(&quant)
		}

		// Update product on-hand
		var totalQty float64
		tx.Model(&stock.StockQuant{}).
			Joins("JOIN stock_locations ON stock_locations.id = stock_quants.location_id").
			Where("stock_quants.product_id = ? AND stock_locations.usage = 'internal'", line.ProductID).
			Select("COALESCE(SUM(quantity), 0)").
			Scan(&totalQty)
		tx.Model(&product.Product{}).Where("id = ?", line.ProductID).Update("qty_available", totalQty)
	}

	// Update Session aggregates
	session.TotalOrders += 1
	session.TotalRevenue += order.AmountTotal
	tx.Save(&session)

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return order, nil
}
