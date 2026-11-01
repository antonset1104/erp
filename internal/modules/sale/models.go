package sale

import (
	"erp-golang/internal/core/models"
	"erp-golang/internal/modules/base"
	"erp-golang/internal/modules/product"
)

// SaleOrder corresponds to Odoo's sale.order
type SaleOrder struct {
	models.ModelBase
	Name          string          `gorm:"size:64;not null;uniqueIndex" json:"name"` // e.g. SO0001
	PartnerID     uint            `gorm:"not null;index" json:"partner_id"`
	Partner       base.Partner    `gorm:"foreignKey:PartnerID" json:"partner,omitempty"`
	DateOrder     string          `gorm:"size:32;not null" json:"date_order"`
	State         string          `gorm:"size:32;default:'draft';index" json:"state"` // "draft", "sent", "sale", "done", "cancel"
	AmountUntaxed float64         `gorm:"default:0" json:"amount_untaxed"`
	AmountTax     float64         `gorm:"default:0" json:"amount_tax"`
	AmountTotal   float64         `gorm:"default:0" json:"amount_total"`
	TaxRate       float64         `gorm:"default:11.0" json:"tax_rate"` // PPN 11% default in Indonesia
	OrderLines    []SaleOrderLine `gorm:"foreignKey:OrderID" json:"order_lines"`
	DeliveryID    *uint           `json:"delivery_id,omitempty"` // Related StockPicking ID
	InvoiceID     *uint           `json:"invoice_id,omitempty"`  // Related AccountMove ID
	Note          string          `gorm:"type:text" json:"note"`
}

// SaleOrderLine corresponds to Odoo's sale.order.line
type SaleOrderLine struct {
	models.ModelBase
	OrderID       uint            `gorm:"index" json:"order_id"`
	ProductID     uint            `gorm:"not null" json:"product_id"`
	Product       product.Product `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	Name          string          `gorm:"size:255" json:"name"` // Description
	ProductUomQty float64         `gorm:"not null" json:"product_uom_qty"`
	PriceUnit     float64         `gorm:"not null" json:"price_unit"`
	PriceSubtotal float64         `gorm:"not null" json:"price_subtotal"`
	PriceTotal    float64         `gorm:"not null" json:"price_total"`
}
