package purchase

import (
	"erp-golang/internal/core/models"
	"erp-golang/internal/modules/base"
	"erp-golang/internal/modules/product"
)

// PurchaseOrder corresponds to Odoo's purchase.order
type PurchaseOrder struct {
	models.ModelBase
	Name          string              `gorm:"size:64;not null;uniqueIndex" json:"name"` // e.g. PO0001
	PartnerID     uint                `gorm:"not null;index" json:"partner_id"`         // Vendor
	Partner       base.Partner        `gorm:"foreignKey:PartnerID" json:"partner,omitempty"`
	DateOrder     string              `gorm:"size:32;not null" json:"date_order"`
	State         string              `gorm:"size:32;default:'draft';index" json:"state"` // "draft", "sent", "to approve", "purchase", "done", "cancel"
	AmountUntaxed float64             `gorm:"default:0" json:"amount_untaxed"`
	AmountTax     float64             `gorm:"default:0" json:"amount_tax"`
	AmountTotal   float64             `gorm:"default:0" json:"amount_total"`
	TaxRate       float64             `gorm:"default:11.0" json:"tax_rate"` // PPN 11%
	OrderLines    []PurchaseOrderLine `gorm:"foreignKey:OrderID" json:"order_lines"`
	ReceiptID     *uint               `json:"receipt_id,omitempty"` // Related StockPicking ID (Incoming)
	BillID        *uint               `json:"bill_id,omitempty"`    // Related AccountMove ID (Vendor Bill)
	RequisitionID *uint               `gorm:"index" json:"requisition_id,omitempty"` // Related PurchaseRequisition ID (Tender / Blanket)
	Note          string              `gorm:"type:text" json:"note"`
}

// PurchaseOrderLine corresponds to Odoo's purchase.order.line
type PurchaseOrderLine struct {
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
