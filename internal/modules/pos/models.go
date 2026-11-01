package pos

import (
	"erp-golang/internal/core/models"
	"erp-golang/internal/modules/base"
	"erp-golang/internal/modules/product"
)

// PosConfig corresponds to Odoo's pos.config (POS Terminal/Register)
type PosConfig struct {
	models.ModelBase
	Name         string `gorm:"size:128;not null" json:"name"` // e.g. "Cashier Desk 1"
	ReceiptHeader string `gorm:"type:text" json:"receipt_header"`
	ReceiptFooter string `gorm:"type:text" json:"receipt_footer"`
}

// PosSession corresponds to Odoo's pos.session (Cashier Work Shift)
type PosSession struct {
	models.ModelBase
	Name                     string     `gorm:"size:64;not null;uniqueIndex" json:"name"` // e.g. POS/SESSION/0001
	ConfigID                 uint       `gorm:"not null;index" json:"config_id"`
	Config                   PosConfig  `gorm:"foreignKey:ConfigID" json:"config,omitempty"`
	UserID                   uint       `gorm:"not null" json:"user_id"`
	User                     base.User  `gorm:"foreignKey:UserID" json:"user,omitempty"`
	State                    string     `gorm:"size:32;default:'opened';index" json:"state"` // "opened", "closed"
	StartAt                  string     `gorm:"size:32;not null" json:"start_at"`
	StopAt                   string     `gorm:"size:32" json:"stop_at,omitempty"`
	CashRegisterBalanceStart float64    `gorm:"default:0" json:"cash_register_balance_start"`
	CashRegisterBalanceEnd   float64    `gorm:"default:0" json:"cash_register_balance_end"`
	TotalOrders              int        `gorm:"default:0" json:"total_orders"`
	TotalRevenue             float64    `gorm:"default:0" json:"total_revenue"`
}

// PosOrder corresponds to Odoo's pos.order (Point of Sale Transaction)
type PosOrder struct {
	models.ModelBase
	Name          string         `gorm:"size:64;not null;uniqueIndex" json:"name"` // e.g. POS/2026/00001
	SessionID     uint           `gorm:"not null;index" json:"session_id"`
	PartnerID     *uint          `json:"partner_id,omitempty"`
	Partner       *base.Partner  `gorm:"foreignKey:PartnerID" json:"partner,omitempty"`
	DateOrder     string         `gorm:"size:32;not null" json:"date_order"`
	AmountTotal   float64        `gorm:"not null" json:"amount_total"`
	AmountPaid    float64        `gorm:"not null" json:"amount_paid"`
	AmountChange  float64        `gorm:"default:0" json:"amount_change"`
	PaymentMethod string         `gorm:"size:32;default:'cash'" json:"payment_method"` // "cash", "bank", "qris"
	State         string         `gorm:"size:32;default:'paid'" json:"state"`          // "draft", "paid", "done", "invoiced"
	Lines         []PosOrderLine `gorm:"foreignKey:OrderID" json:"lines"`
	Note          string         `gorm:"type:text" json:"note"`
}

// PosOrderLine corresponds to Odoo's pos.order.line
type PosOrderLine struct {
	models.ModelBase
	OrderID       uint            `gorm:"index;not null" json:"order_id"`
	ProductID     uint            `gorm:"not null" json:"product_id"`
	Product       product.Product `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	ProductName   string          `gorm:"size:255" json:"product_name"`
	Qty           float64         `gorm:"not null" json:"qty"`
	PriceUnit     float64         `gorm:"not null" json:"price_unit"`
	PriceSubtotal float64         `gorm:"not null" json:"price_subtotal"`
}
