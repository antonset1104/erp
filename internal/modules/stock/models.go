package stock

import (
	"erp-golang/internal/core/models"
	"erp-golang/internal/modules/product"
)

// StockWarehouse corresponds to standard enterprise stock.warehouse (Multi-Warehouse Management)
type StockWarehouse struct {
	models.ModelBase
	Name       string         `gorm:"size:128;not null" json:"name"`
	Code       string         `gorm:"size:16;not null;uniqueIndex" json:"code"` // e.g. WH-JKT, WH-SBY
	LotStockID uint           `gorm:"not null" json:"lot_stock_id"`            // Main internal stock location
	LotStock   *StockLocation `gorm:"foreignKey:LotStockID" json:"lot_stock,omitempty"`
	PartnerID  *uint          `json:"partner_id,omitempty"`
	Address    string         `gorm:"size:255" json:"address,omitempty"`
	Active     bool           `gorm:"default:true" json:"active"`
}

// StockLocation corresponds to standard enterprise stock.location
type StockLocation struct {
	models.ModelBase
	Name         string          `gorm:"size:128;not null" json:"name"`
	Usage        string          `gorm:"size:32;not null" json:"usage"` // "internal", "transit", "supplier", "customer", "inventory", "production"
	ParentID     *uint           `json:"parent_id,omitempty"`
	WarehouseID  *uint           `gorm:"index" json:"warehouse_id,omitempty"`
	Warehouse    *StockWarehouse `gorm:"foreignKey:WarehouseID" json:"warehouse,omitempty"`
	CompleteName string          `gorm:"size:255" json:"complete_name"`
}

// StockPicking corresponds to standard enterprise stock.picking (Receipts, Deliveries, Transfers)
type StockPicking struct {
	models.ModelBase
	Name           string      `gorm:"size:64;not null;uniqueIndex" json:"name"` // e.g. WH/IN/0001, WH/OUT/0001
	Origin         string      `gorm:"size:64" json:"origin"`                    // e.g. SO0001, PO0001
	PickingType    string      `gorm:"size:32;not null" json:"picking_type"`     // "incoming", "outgoing", "internal"
	PartnerID      *uint       `json:"partner_id,omitempty"`
	PartnerName    string      `gorm:"size:255" json:"partner_name,omitempty"`
	LocationID     uint        `gorm:"not null" json:"location_id"`              // Source location
	Location       StockLocation `gorm:"foreignKey:LocationID" json:"location,omitempty"`
	LocationDestID uint        `gorm:"not null" json:"location_dest_id"`         // Destination location
	LocationDest   StockLocation `gorm:"foreignKey:LocationDestID" json:"location_dest,omitempty"`
	State          string      `gorm:"size:32;default:'draft';index" json:"state"` // "draft", "confirmed", "assigned", "done", "cancel"
	ScheduledDate  string      `gorm:"size:32" json:"scheduled_date"`
	DateDone       string      `gorm:"size:32" json:"date_done,omitempty"`
	MoveLines      []StockMove `gorm:"foreignKey:PickingID" json:"move_lines"`
	Note           string      `gorm:"type:text" json:"note"`
}

// StockMove corresponds to standard enterprise stock.move
type StockMove struct {
	models.ModelBase
	PickingID      uint            `gorm:"index" json:"picking_id"`
	ProductID      uint            `gorm:"not null;index" json:"product_id"`
	Product        product.Product `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	ProductName    string          `gorm:"size:255" json:"product_name"`
	ProductUomQty  float64         `gorm:"not null" json:"product_uom_qty"` // Demand qty
	QuantityDone   float64         `gorm:"default:0" json:"quantity_done"`  // Actual moved qty
	LocationID     uint            `gorm:"not null" json:"location_id"`     // Source
	LocationDestID uint            `gorm:"not null" json:"location_dest_id"`// Destination
	State          string          `gorm:"size:32;default:'draft'" json:"state"` // "draft", "confirmed", "done", "cancel"
}

// StockQuant corresponds to standard enterprise stock.quant (On-Hand Balance per Product & Location)
type StockQuant struct {
	models.ModelBase
	ProductID  uint            `gorm:"not null;index:idx_prod_loc,unique" json:"product_id"`
	Product    product.Product `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	LocationID uint            `gorm:"not null;index:idx_prod_loc,unique" json:"location_id"`
	Location   StockLocation   `gorm:"foreignKey:LocationID" json:"location,omitempty"`
	Quantity   float64         `gorm:"default:0;not null" json:"quantity"` // Real on-hand quantity
	Reserved   float64         `gorm:"default:0" json:"reserved"`
}

// StockOrderpoint corresponds to standard enterprise stock.warehouse.orderpoint (Reordering Rules)
type StockOrderpoint struct {
	models.ModelBase
	Name          string          `gorm:"size:64;not null;uniqueIndex" json:"name"` // e.g. OP/00001
	ProductID     uint            `gorm:"not null;index" json:"product_id"`
	Product       product.Product `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	LocationID    uint            `gorm:"not null;index" json:"location_id"`
	Location      StockLocation   `gorm:"foreignKey:LocationID" json:"location,omitempty"`
	ProductMinQty float64         `gorm:"not null;default:0" json:"product_min_qty"` // Safety Stock
	ProductMaxQty float64         `gorm:"not null;default:0" json:"product_max_qty"` // Target Stock
	QtyMultiple   float64         `gorm:"not null;default:1" json:"qty_multiple"`
	Trigger       string          `gorm:"size:32;default:'auto'" json:"trigger"`     // auto, manual
}

