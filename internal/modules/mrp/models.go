package mrp

import (
	"erp-golang/internal/core/models"
	"erp-golang/internal/modules/product"
)

// MrpBom corresponds to standard enterprise mrp.bom (Bill of Materials)
type MrpBom struct {
	models.ModelBase
	Code        string       `gorm:"size:64;not null;uniqueIndex" json:"code"` // e.g. BOM/PC-DESK/01
	ProductID   uint         `gorm:"not null;index" json:"product_id"`         // Finished Product
	Product     product.Product `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	ProductQty  float64      `gorm:"default:1.0;not null" json:"product_qty"` // Quantity produced per BOM
	Lines       []MrpBomLine `gorm:"foreignKey:BomID" json:"lines"`
	Description string       `gorm:"type:text" json:"description"`
}

// MrpBomLine corresponds to standard enterprise mrp.bom.line (BOM Component / Raw Material)
type MrpBomLine struct {
	models.ModelBase
	BomID       uint            `gorm:"index;not null" json:"bom_id"`
	ProductID   uint            `gorm:"not null" json:"product_id"` // Component Product
	Product     product.Product `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	ProductName string          `gorm:"size:255" json:"product_name"`
	ProductQty  float64         `gorm:"not null" json:"product_qty"` // Required quantity
}

// MrpProduction corresponds to standard enterprise mrp.production (Manufacturing Order)
type MrpProduction struct {
	models.ModelBase
	Name        string          `gorm:"size:64;not null;uniqueIndex" json:"name"` // e.g. MO/2026/00001
	ProductID   uint            `gorm:"not null;index" json:"product_id"`
	Product     product.Product `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	BomID       uint            `gorm:"not null;index" json:"bom_id"`
	Bom         MrpBom          `gorm:"foreignKey:BomID" json:"bom,omitempty"`
	ProductQty  float64         `gorm:"not null" json:"product_qty"` // Target production quantity
	State       string          `gorm:"size:32;default:'draft';index" json:"state"` // "draft", "confirmed", "done", "cancel"
	DatePlanned string          `gorm:"size:32" json:"date_planned"`
	DateDone    string          `gorm:"size:32" json:"date_done,omitempty"`
	Note        string          `gorm:"type:text" json:"note"`
}
