package stock_inventory

import (
	"erp-golang/internal/core/models"
	"erp-golang/internal/modules/product"
	"erp-golang/internal/modules/stock"
)

// StockInventory corresponds to Odoo's stock.inventory (Physical Stock Opname / Inventory Adjustment)
type StockInventory struct {
	models.ModelBase
	Name                string               `gorm:"size:64;not null;uniqueIndex" json:"name"` // e.g. OPNAME/2026/00001
	WarehouseID         *uint                `gorm:"index" json:"warehouse_id,omitempty"`
	Warehouse           *stock.StockWarehouse `gorm:"foreignKey:WarehouseID" json:"warehouse,omitempty"`
	LocationID          uint                 `gorm:"not null;index" json:"location_id"` // Location being counted, e.g. WH/Stock
	Location            *stock.StockLocation `gorm:"foreignKey:LocationID" json:"location,omitempty"`
	AccountingMoveID    *uint                `gorm:"index" json:"accounting_move_id,omitempty"` // Linked GL Journal Entry
	AccountingMoveName  string               `gorm:"size:64" json:"accounting_move_name,omitempty"`
	Date                string               `gorm:"size:32;not null" json:"date"`
	State               string               `gorm:"size:32;default:'draft';index" json:"state"` // "draft", "confirm", "done", "cancel"
	TotalTheoreticalQty float64              `gorm:"default:0" json:"total_theoretical_qty"`
	TotalCountedQty     float64              `gorm:"default:0" json:"total_counted_qty"`
	TotalDifferenceQty  float64              `gorm:"default:0" json:"total_difference_qty"`
	TotalLossValue      float64              `gorm:"default:0" json:"total_loss_value"` // Rp Kerugian Selisih Kurang
	TotalGainValue      float64              `gorm:"default:0" json:"total_gain_value"` // Rp Keuntungan Selisih Lebih
	AIRiskLevel         string               `gorm:"size:32;default:'none'" json:"ai_risk_level"` // "none", "low", "medium", "high"
	AIAuditSummary      string               `gorm:"type:text" json:"ai_audit_summary,omitempty"`
	Notes               string               `gorm:"type:text" json:"notes,omitempty"`
	Lines               []StockInventoryLine `gorm:"foreignKey:InventoryID" json:"lines"`
}

// StockInventoryLine corresponds to Odoo's stock.inventory.line
type StockInventoryLine struct {
	models.ModelBase
	InventoryID    uint             `gorm:"not null;index" json:"inventory_id"`
	ProductID      uint             `gorm:"not null;index" json:"product_id"`
	Product        *product.Product `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	TheoreticalQty float64          `gorm:"not null;default:0" json:"theoretical_qty"` // Book quantity in stock_quants
	CountedQty     float64          `gorm:"not null;default:0" json:"counted_qty"`     // Physical counted quantity
	DifferenceQty  float64          `gorm:"not null;default:0" json:"difference_qty"`  // Counted - Theoretical
	CostPrice      float64          `gorm:"not null;default:0" json:"cost_price"`      // Standard/Cost Price
	VarianceValue  float64          `gorm:"not null;default:0" json:"variance_value"`  // DifferenceQty * CostPrice
	Reason         string           `gorm:"size:255" json:"reason,omitempty"`          // e.g. "Kerusakan fisik", "Selisih hitung"
	State          string           `gorm:"size:32;default:'draft'" json:"state"`      // "draft", "done"
}

// Request and Filter DTOs
type CreateInventoryRequest struct {
	Name        string `json:"name"`
	WarehouseID *uint  `json:"warehouse_id"`
	LocationID  uint   `json:"location_id" binding:"required"`
	Date        string `json:"date"`
	Notes       string `json:"notes"`
}

type UpdateInventoryLineRequest struct {
	CountedQty float64 `json:"counted_qty"`
	Reason     string  `json:"reason"`
}

type AddInventoryLineRequest struct {
	ProductID  uint    `json:"product_id" binding:"required"`
	CountedQty float64 `json:"counted_qty"`
	Reason     string  `json:"reason"`
}

type AIAuditResult struct {
	InventoryID        uint     `json:"inventory_id"`
	InventoryName      string   `json:"inventory_name"`
	RiskLevel          string   `json:"risk_level"` // "low", "medium", "high"
	Summary            string   `json:"summary"`
	DiscrepancyCount   int      `json:"discrepancy_count"`
	TotalLossValue     float64  `json:"total_loss_value"`
	TotalGainValue     float64  `json:"total_gain_value"`
	SuspiciousItems    []string `json:"suspicious_items"`
	RecommendedActions []string `json:"recommended_actions"`
	EngineSource       string   `json:"engine_source"`
}

type InventoryStats struct {
	TotalInventories     int64   `json:"total_inventories"`
	DraftCount           int64   `json:"draft_count"`
	ConfirmedCount       int64   `json:"confirmed_count"`
	DoneCount            int64   `json:"done_count"`
	TotalDifferenceUnits float64 `json:"total_difference_units"`
	TotalLossValue       float64 `json:"total_loss_value"`
	TotalGainValue       float64 `json:"total_gain_value"`
	HighRiskAuditCount   int64   `json:"high_risk_audit_count"`
}
