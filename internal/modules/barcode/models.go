package barcode

import (
	"erp-golang/internal/core/models"
	"erp-golang/internal/modules/product"
	"erp-golang/internal/modules/stock"
)

// StockProductionLot corresponds to standard enterprise stock.production.lot (Lot & Serial Number Tracking)
type StockProductionLot struct {
	models.ModelBase
	Name           string               `gorm:"size:64;not null;index" json:"name"` // e.g. LOT/2026/001, SN-88291039
	ProductID      uint                 `gorm:"not null;index" json:"product_id"`
	Product        *product.Product     `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	Barcode        string               `gorm:"size:64;index" json:"barcode,omitempty"`
	Ref            string               `gorm:"size:128" json:"ref,omitempty"` // Internal reference / batch vendor
	ExpirationDate string               `gorm:"size:32" json:"expiration_date,omitempty"` // Best Before / Kadaluwarsa
	AlertDate      string               `gorm:"size:32" json:"alert_date,omitempty"`
	RemovalDate    string               `gorm:"size:32" json:"removal_date,omitempty"`
	Qty            float64              `gorm:"default:0" json:"qty"` // Current lot balance
	LocationID     *uint                `gorm:"index" json:"location_id,omitempty"`
	Location       *stock.StockLocation `gorm:"foreignKey:LocationID" json:"location,omitempty"`
	Notes          string               `gorm:"type:text" json:"notes,omitempty"`
}

func (StockProductionLot) TableName() string {
	return "stock_production_lots"
}

// DTOs
type CreateLotRequest struct {
	Name           string  `json:"name" binding:"required"`
	ProductID      uint    `json:"product_id" binding:"required"`
	Barcode        string  `json:"barcode"`
	Ref            string  `json:"ref"`
	ExpirationDate string  `json:"expiration_date"`
	AlertDate      string  `json:"alert_date"`
	RemovalDate    string  `json:"removal_date"`
	Qty            float64 `json:"qty"`
	LocationID     *uint   `json:"location_id"`
	Notes          string  `json:"notes"`
}

type BarcodeScanRequest struct {
	Barcode  string  `json:"barcode" binding:"required"`
	Context  string  `json:"context"`   // "pos", "stock_inventory", "picking", "general"
	TargetID uint    `json:"target_id"` // inventory_id, picking_id, or order_id
	Qty      float64 `json:"qty"`       // Default 1.0
	Action   string  `json:"action"`    // "lookup", "increment", "add"
}

type BarcodeScanResult struct {
	Found          bool                `json:"found"`
	Barcode        string              `json:"barcode"`
	Type           string              `json:"type"` // "product", "location", "lot", "picking", "unknown"
	ActionTaken    string              `json:"action_taken,omitempty"`
	Message        string              `json:"message"`
	Product        *product.Product    `json:"product,omitempty"`
	Location       *stock.StockLocation `json:"location,omitempty"`
	Lot            *StockProductionLot `json:"lot,omitempty"`
	Picking        *stock.StockPicking `json:"picking,omitempty"`
	Data           interface{}         `json:"data,omitempty"`
}

type BarcodeStats struct {
	TotalProductsBarcoded  int64 `json:"total_products_barcoded"`
	TotalLots              int64 `json:"total_lots"`
	TotalLocationsBarcoded int64 `json:"total_locations_barcoded"`
	ActiveLotsWithStock    int64 `json:"active_lots_with_stock"`
	ExpiredLots            int64 `json:"expired_lots"`
}
