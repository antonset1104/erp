package customer_portal

import (
	"fmt"
	"time"

	"erp-golang/internal/modules/account"
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) AutoMigrate() error {
	return r.db.AutoMigrate(
		&CustomerPortalAccount{},
		&B2BOrderRequest{},
		&B2BOrderLine{},
		&RMARequest{},
		&CustomerPaymentNotification{},
	)
}

// Sequence generators
func (r *Repository) GenerateOrderRequestNumber() (string, error) {
	year := time.Now().Year()
	var count int64
	if err := r.db.Model(&B2BOrderRequest{}).Count(&count).Error; err != nil {
		return "", err
	}
	return fmt.Sprintf("CP-SO/%d/%05d", year, count+1), nil
}

func (r *Repository) GenerateRMANumber() (string, error) {
	year := time.Now().Year()
	var count int64
	if err := r.db.Model(&RMARequest{}).Count(&count).Error; err != nil {
		return "", err
	}
	return fmt.Sprintf("RMA/%d/%05d", year, count+1), nil
}

func (r *Repository) GeneratePaymentNotificationNumber() (string, error) {
	year := time.Now().Year()
	var count int64
	if err := r.db.Model(&CustomerPaymentNotification{}).Count(&count).Error; err != nil {
		return "", err
	}
	return fmt.Sprintf("CP-PAY/%d/%05d", year, count+1), nil
}

// Customer Accounts
func (r *Repository) CreateAccount(acc *CustomerPortalAccount) error {
	return r.db.Create(acc).Error
}

func (r *Repository) GetAccountByID(id uint) (*CustomerPortalAccount, error) {
	var acc CustomerPortalAccount
	if err := r.db.First(&acc, id).Error; err != nil {
		return nil, err
	}
	return &acc, nil
}

func (r *Repository) GetAccountByToken(token string) (*CustomerPortalAccount, error) {
	var acc CustomerPortalAccount
	if err := r.db.Where("access_token = ? AND is_active = ?", token, true).First(&acc).Error; err != nil {
		return nil, err
	}
	return &acc, nil
}

func (r *Repository) GetAccountByPartnerID(partnerID uint) (*CustomerPortalAccount, error) {
	var acc CustomerPortalAccount
	if err := r.db.Where("partner_id = ?", partnerID).First(&acc).Error; err != nil {
		return nil, err
	}
	return &acc, nil
}

func (r *Repository) ListAccounts() ([]CustomerPortalAccount, error) {
	var accounts []CustomerPortalAccount
	err := r.db.Order("id desc").Find(&accounts).Error
	return accounts, err
}

func (r *Repository) UpdateAccount(acc *CustomerPortalAccount) error {
	return r.db.Save(acc).Error
}

// B2B Order Requests
func (r *Repository) CreateOrderRequest(req *B2BOrderRequest) error {
	return r.db.Create(req).Error
}

func (r *Repository) GetOrderRequestByID(id uint) (*B2BOrderRequest, error) {
	var req B2BOrderRequest
	if err := r.db.Preload("Lines.Product").Preload("SaleOrder").Preload("PortalAccount").First(&req, id).Error; err != nil {
		return nil, err
	}
	return &req, nil
}

func (r *Repository) ListOrderRequests(portalAccountID uint) ([]B2BOrderRequest, error) {
	var list []B2BOrderRequest
	query := r.db.Preload("Lines.Product").Preload("SaleOrder")
	if portalAccountID > 0 {
		query = query.Where("portal_account_id = ?", portalAccountID)
	}
	err := query.Order("id desc").Find(&list).Error
	return list, err
}

func (r *Repository) ListAllOrderRequests() ([]B2BOrderRequest, error) {
	return r.ListOrderRequests(0)
}

func (r *Repository) UpdateOrderRequest(req *B2BOrderRequest) error {
	return r.db.Save(req).Error
}

// RMA Requests
func (r *Repository) CreateRMA(rma *RMARequest) error {
	return r.db.Create(rma).Error
}

func (r *Repository) GetRMAByID(id uint) (*RMARequest, error) {
	var rma RMARequest
	if err := r.db.Preload("Product").Preload("SaleOrder").Preload("PortalAccount").First(&rma, id).Error; err != nil {
		return nil, err
	}
	return &rma, nil
}

func (r *Repository) ListRMAs(portalAccountID uint) ([]RMARequest, error) {
	var list []RMARequest
	query := r.db.Preload("Product").Preload("SaleOrder")
	if portalAccountID > 0 {
		query = query.Where("portal_account_id = ?", portalAccountID)
	}
	err := query.Order("id desc").Find(&list).Error
	return list, err
}

func (r *Repository) ListAllRMAs() ([]RMARequest, error) {
	return r.ListRMAs(0)
}

func (r *Repository) UpdateRMA(rma *RMARequest) error {
	return r.db.Save(rma).Error
}

// Payment Notifications
func (r *Repository) CreatePaymentNotification(notif *CustomerPaymentNotification) error {
	return r.db.Create(notif).Error
}

func (r *Repository) GetPaymentNotificationByID(id uint) (*CustomerPaymentNotification, error) {
	var notif CustomerPaymentNotification
	if err := r.db.Preload("Invoice").Preload("PortalAccount").First(&notif, id).Error; err != nil {
		return nil, err
	}
	return &notif, nil
}

func (r *Repository) ListPaymentNotifications(portalAccountID uint) ([]CustomerPaymentNotification, error) {
	var list []CustomerPaymentNotification
	query := r.db.Preload("Invoice")
	if portalAccountID > 0 {
		query = query.Where("portal_account_id = ?", portalAccountID)
	}
	err := query.Order("id desc").Find(&list).Error
	return list, err
}

func (r *Repository) ListAllPaymentNotifications() ([]CustomerPaymentNotification, error) {
	return r.ListPaymentNotifications(0)
}

func (r *Repository) UpdatePaymentNotification(notif *CustomerPaymentNotification) error {
	return r.db.Save(notif).Error
}

// Stats & KPI
func (r *Repository) GetStats() (*CustomerPortalStats, error) {
	var stats CustomerPortalStats

	var totalAcc, activeAcc int64
	r.db.Model(&CustomerPortalAccount{}).Count(&totalAcc)
	r.db.Model(&CustomerPortalAccount{}).Where("is_active = ?", true).Count(&activeAcc)
	stats.TotalCustomerAccounts = int(totalAcc)
	stats.ActiveAccounts = int(activeAcc)

	var totalOrders, approvedOrders int64
	r.db.Model(&B2BOrderRequest{}).Count(&totalOrders)
	r.db.Model(&B2BOrderRequest{}).Where("status IN ?", []string{"approved", "converted_to_so"}).Count(&approvedOrders)
	stats.TotalB2BOrders = int(totalOrders)
	stats.ApprovedB2BOrders = int(approvedOrders)

	var totalOrderVal float64
	r.db.Model(&B2BOrderRequest{}).Select("COALESCE(SUM(total_amount), 0)").Scan(&totalOrderVal)
	stats.TotalB2BOrderValue = totalOrderVal

	var totalRMAs, pendingRMAs int64
	r.db.Model(&RMARequest{}).Count(&totalRMAs)
	r.db.Model(&RMARequest{}).Where("status IN ?", []string{"submitted", "under_review"}).Count(&pendingRMAs)
	stats.TotalRMAs = int(totalRMAs)
	stats.PendingRMAs = int(pendingRMAs)

	var pendingPay int64
	r.db.Model(&CustomerPaymentNotification{}).Where("status = ?", "pending_verification").Count(&pendingPay)
	stats.TotalPendingPayments = int(pendingPay)

	// Calculate outstanding AR balance from posted out_invoice account_moves
	var arBalance float64
	r.db.Model(&account.AccountMove{}).
		Where("move_type = ? AND state = ? AND payment_state IN ?", "out_invoice", "posted", []string{"not_paid", "partial"}).
		Select("COALESCE(SUM(amount_residual), 0)").
		Scan(&arBalance)
	stats.OutstandingARBalance = arBalance

	return &stats, nil
}
