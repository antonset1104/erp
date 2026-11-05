package product

import (
	"erp-golang/internal/core/models"
)

// ProductCategory corresponds to standard enterprise product.category
type ProductCategory struct {
	models.ModelBase
	Name     string `gorm:"size:128;not null" json:"name"`
	ParentID *uint  `json:"parent_id,omitempty"`
}

// UoM corresponds to standard enterprise uom.uom (Unit of Measure)
type UoM struct {
	models.ModelBase
	Name     string  `gorm:"size:64;not null" json:"name"` // Unit, Pcs, Kg, Liter
	Category string  `gorm:"size:64;default:'Unit'" json:"category"`
	Factor   float64 `gorm:"default:1.0" json:"factor"` // Ratio to reference unit
	Rounding float64 `gorm:"default:0.01" json:"rounding"`
	Active   bool    `gorm:"default:true" json:"active"`
}

// Product corresponds to standard enterprise product.template / product.product
type Product struct {
	models.ModelBase
	Name          string          `gorm:"size:255;not null;index" json:"name"`
	DefaultCode   string          `gorm:"size:64;index" json:"default_code"` // SKU / Internal Reference
	Barcode       string          `gorm:"size:64;index" json:"barcode"`
	Type          string          `gorm:"size:32;default:'product'" json:"type"` // "product" (storable), "consu" (consumable), "service"
	CategoryID    uint            `gorm:"default:1" json:"category_id"`
	Category      ProductCategory `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	UoMID         uint            `gorm:"default:1" json:"uom_id"`
	UoM           UoM             `gorm:"foreignKey:UoMID" json:"uom,omitempty"`
	ListPrice     float64         `gorm:"default:0;not null" json:"list_price"`     // Sales Price
	StandardPrice float64         `gorm:"default:0;not null" json:"standard_price"` // Cost Price
	Weight        float64         `gorm:"default:0" json:"weight"`                  // Gross weight in kg
	Volume        float64         `gorm:"default:0" json:"volume"`                  // Volume in m3
	QtyAvailable  float64         `gorm:"default:0" json:"qty_available"`           // Computed On Hand Qty
	Description   string          `gorm:"type:text" json:"description"`
}
