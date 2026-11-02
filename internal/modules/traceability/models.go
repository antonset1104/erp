package traceability

import (
	"erp-golang/internal/core/models"
	"erp-golang/internal/modules/barcode"
	"erp-golang/internal/modules/product"
)

// TraceEvent types represent physical movements and transformations of lots
const (
	EventReceipt    = "receipt"     // Incoming vendor receipt (WH/IN/...)
	EventMOConsume  = "mo_consume"  // Raw material consumed in manufacturing (MO/...)
	EventMOProduce  = "mo_produce"  // Finished goods produced in manufacturing (MO/...)
	EventDelivery   = "delivery"    // Outgoing customer delivery (WH/OUT/...)
	EventTransfer   = "transfer"    // Internal warehouse transfer (WH/INT/...)
	EventAdjustment = "adjustment"  // Stock opname / physical adjustment
	EventScrap      = "scrap"       // Damaged / discarded / quarantine scrap
)

// Recall states
const (
	RecallStateDraft       = "draft"
	RecallStateActive      = "active"
	RecallStateQuarantined = "quarantined"
	RecallStateCompleted   = "completed"
	RecallStateCancelled   = "cancelled"
)

// LotTraceRecord represents an immutable audit log of a lot's physical movement or transformation.
// Corresponds to standard enterprise stock.move.line & stock.traceability.report.
type LotTraceRecord struct {
	models.ModelBase
	LotID          uint                       `gorm:"not null;index" json:"lot_id"`
	Lot            *barcode.StockProductionLot `gorm:"foreignKey:LotID" json:"lot,omitempty"`
	LotName        string                     `gorm:"size:64;not null;index" json:"lot_name"`
	ProductID      uint                       `gorm:"not null;index" json:"product_id"`
	Product        *product.Product           `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	ProductName    string                     `gorm:"size:255;not null" json:"product_name"`
	EventType      string                     `gorm:"size:32;not null;index" json:"event_type"` // receipt, mo_consume, mo_produce, delivery, transfer, scrap
	Reference      string                     `gorm:"size:64;not null;index" json:"reference"`  // WH/IN/0001, MO/2026/0001, WH/OUT/0001
	SourceDocument string                     `gorm:"size:64" json:"source_document,omitempty"` // PO0001, SO0001
	PartnerID      *uint                      `gorm:"index" json:"partner_id,omitempty"`
	PartnerName    string                     `gorm:"size:255" json:"partner_name,omitempty"`    // Supplier or Customer
	PickingID      *uint                      `gorm:"index" json:"picking_id,omitempty"`
	ProductionID   *uint                      `gorm:"index" json:"production_id,omitempty"`

	// Genealogical links for bi-directional upstream/downstream tracing
	ParentLotID     *uint `gorm:"index" json:"parent_lot_id,omitempty"` // Raw lot that went into this finished lot
	ParentLotName   string `gorm:"size:64" json:"parent_lot_name,omitempty"`
	ProducedLotID   *uint `gorm:"index" json:"produced_lot_id,omitempty"` // Finished lot produced if this was consumed in MO
	ProducedLotName string `gorm:"size:64" json:"produced_lot_name,omitempty"`
	ConsumedLotID   *uint `gorm:"index" json:"consumed_lot_id,omitempty"` // Raw lot consumed if this was produced in MO
	ConsumedLotName string `gorm:"size:64" json:"consumed_lot_name,omitempty"`

	Quantity         float64 `gorm:"not null" json:"quantity"`
	UoM              string  `gorm:"size:32" json:"uom,omitempty"`
	LocationSrcID    *uint   `json:"location_src_id,omitempty"`
	LocationSrcName  string  `gorm:"size:128" json:"location_src_name,omitempty"`
	LocationDestID   *uint   `json:"location_dest_id,omitempty"`
	LocationDestName string  `gorm:"size:128" json:"location_dest_name,omitempty"`
	Date             string  `gorm:"size:32;not null" json:"date"`
	ExpirationDate   string  `gorm:"size:32" json:"expiration_date,omitempty"`
	Notes            string  `gorm:"type:text" json:"notes,omitempty"`
}

func (LotTraceRecord) TableName() string {
	return "stock_lot_traces"
}

// ProductRecall corresponds to standard enterprise stock.lot.recall.
// Captures official recall campaigns for contaminated or defective lots.
type ProductRecall struct {
	models.ModelBase
	RecallCode                string  `gorm:"size:64;not null;uniqueIndex" json:"recall_code"` // e.g. RCL/2026/00001
	TargetLotID               uint    `gorm:"not null;index" json:"target_lot_id"`
	TargetLotName             string  `gorm:"size:64;not null" json:"target_lot_name"`
	ProductID                 uint    `gorm:"not null;index" json:"product_id"`
	ProductName               string  `gorm:"size:255;not null" json:"product_name"`
	Reason                    string  `gorm:"type:text;not null" json:"reason"`
	Severity                  string  `gorm:"size:32;default:'critical'" json:"severity"` // critical, high, medium, low
	State                     string  `gorm:"size:32;default:'draft';index" json:"state"` // draft, active, quarantined, completed, cancelled
	AffectedStockQuarantined  float64 `gorm:"default:0" json:"affected_stock_quarantined"`
	AffectedDeliveriesCount   int     `gorm:"default:0" json:"affected_deliveries_count"`
	AffectedCustomersCount    int     `gorm:"default:0" json:"affected_customers_count"`
	FinancialRisk             float64 `gorm:"default:0" json:"financial_risk"`
	CustomerNoticeText        string  `gorm:"type:text" json:"customer_notice_text"`
	RegulatoryNoticeText      string  `gorm:"type:text" json:"regulatory_notice_text"`
	ResolutionNotes           string  `gorm:"type:text" json:"resolution_notes"`
	AIProvider                string  `gorm:"size:64" json:"ai_provider,omitempty"`
}

func (ProductRecall) TableName() string {
	return "stock_lot_recalls"
}

// ---------------------------------------------------------------------------
// DTOs & Reports
// ---------------------------------------------------------------------------

type RecordLotMovementRequest struct {
	LotID          uint    `json:"lot_id" binding:"required"`
	ProductID      uint    `json:"product_id" binding:"required"`
	EventType      string  `json:"event_type" binding:"required"` // receipt, mo_consume, mo_produce, delivery, transfer, scrap
	Reference      string  `json:"reference" binding:"required"`
	SourceDocument string  `json:"source_document"`
	PartnerID      *uint   `json:"partner_id"`
	PartnerName    string  `json:"partner_name"`
	PickingID      *uint   `json:"picking_id"`
	ProductionID   *uint   `json:"production_id"`
	ParentLotID    *uint   `json:"parent_lot_id"`
	ProducedLotID  *uint   `json:"produced_lot_id"`
	ConsumedLotID  *uint   `json:"consumed_lot_id"`
	Quantity       float64 `json:"quantity" binding:"required"`
	UoM            string  `json:"uom"`
	LocationSrcID  *uint   `json:"location_src_id"`
	LocationDestID *uint   `json:"location_dest_id"`
	Date           string  `json:"date"`
	Notes          string  `json:"notes"`
}

// FEFO Picking Engine Request & Response
type FEFOAllocationRequest struct {
	ProductID        uint    `json:"product_id" binding:"required"`
	RequestedQty     float64 `json:"requested_qty" binding:"required"`
	LocationID       *uint   `json:"location_id"`
	AllowNearExpiry  bool    `json:"allow_near_expiry"` // Allow allocation of lots expiring within warning window
	ExpiryCutoffDays int     `json:"expiry_cutoff_days"` // Default 30 days
}

type FEFOAllocationItem struct {
	LotID            uint    `json:"lot_id"`
	LotName          string  `json:"lot_name"`
	Barcode          string  `json:"barcode"`
	ExpirationDate   string  `json:"expiration_date"`
	RemovalDate      string  `json:"removal_date"`
	AvailableQty     float64 `json:"available_qty"`
	AllocatedQty     float64 `json:"allocated_qty"`
	DaysUntilExpiry  int     `json:"days_until_expiry"`
	Status           string  `json:"status"` // fresh, warning_near_expiry, expired
	LocationID       *uint   `json:"location_id,omitempty"`
	LocationName     string  `json:"location_name,omitempty"`
}

type FEFOAllocationResult struct {
	ProductID         uint                 `json:"product_id"`
	ProductName       string               `json:"product_name"`
	RequestedQty      float64              `json:"requested_qty"`
	TotalAllocatedQty float64              `json:"total_allocated_qty"`
	ShortageQty       float64              `json:"shortage_qty"`
	IsFullyAllocated  bool                 `json:"is_fully_allocated"`
	StrategyApplied   string               `json:"strategy_applied"`
	Allocations       []FEFOAllocationItem `json:"allocations"`
}

// Bidirectional Traceability Tree
type TraceabilityNode struct {
	ID             string              `json:"id"`
	NodeType       string              `json:"node_type"` // vendor, receipt, raw_lot, production, finished_lot, delivery, customer
	Title          string              `json:"title"`
	Reference      string              `json:"reference"`
	LotID          *uint               `json:"lot_id,omitempty"`
	LotName        string              `json:"lot_name,omitempty"`
	ProductID      uint                `json:"product_id,omitempty"`
	ProductName    string              `json:"product_name,omitempty"`
	Quantity       float64             `json:"quantity"`
	PartnerName    string              `json:"partner_name,omitempty"`
	Date           string              `json:"date,omitempty"`
	ExpirationDate string              `json:"expiration_date,omitempty"`
	Status         string              `json:"status,omitempty"`
	Children       []*TraceabilityNode `json:"children,omitempty"`
}

type TraceabilityTreeReport struct {
	TargetLotID                      uint              `json:"target_lot_id"`
	TargetLotName                    string            `json:"target_lot_name"`
	TargetProductID                  uint              `json:"target_product_id"`
	TargetProductName                string            `json:"target_product_name"`
	Direction                        string            `json:"direction"` // upstream, downstream, full
	GeneratedAt                      string            `json:"generated_at"`
	CurrentLotQty                    float64           `json:"current_lot_qty"`
	ExpirationDate                   string            `json:"expiration_date"`
	UpstreamTree                     *TraceabilityNode `json:"upstream_tree,omitempty"`
	DownstreamTree                   *TraceabilityNode `json:"downstream_tree,omitempty"`
	TotalSuppliersInvolved           int               `json:"total_suppliers_involved"`
	TotalManufacturingOrdersInvolved int               `json:"total_manufacturing_orders_involved"`
	TotalCustomerDeliveriesInvolved  int               `json:"total_customer_deliveries_involved"`
	ImpactedCustomers                []string          `json:"impacted_customers"`
	ImpactedLocations                []string          `json:"impacted_locations"`
	ImpactedFinishedLots             []string          `json:"impacted_finished_lots"`
}

// Product Recall Request
type ProductRecallRequest struct {
	LotID    uint   `json:"lot_id" binding:"required"`
	Reason   string `json:"reason" binding:"required"`
	Severity string `json:"severity"` // critical, high, medium, low
}

// Expiry Risk Audit
type ExpiryLotDetail struct {
	LotID          uint    `json:"lot_id"`
	LotName        string  `json:"lot_name"`
	ProductID      uint    `json:"product_id"`
	ProductName    string  `json:"product_name"`
	Qty            float64 `json:"qty"`
	StandardPrice  float64 `json:"standard_price"`
	TotalLossRisk  float64 `json:"total_loss_risk"`
	ExpirationDate string  `json:"expiration_date"`
	DaysRemaining  int     `json:"days_remaining"`
	Status         string  `json:"status"` // EXPIRED, CRITICAL_7_DAYS, WARNING_30_DAYS, FRESH
	LocationName   string  `json:"location_name"`
}

type ExpiryRiskReport struct {
	TotalLotsTracked    int               `json:"total_lots_tracked"`
	ExpiredLotsCount    int               `json:"expired_lots_count"`
	ExpiredLotsValue    float64           `json:"expired_lots_value"`
	NearExpiryLotsCount int               `json:"near_expiry_lots_count"`
	NearExpiryLotsValue float64           `json:"near_expiry_lots_value"`
	FreshLotsCount      int               `json:"fresh_lots_count"`
	OverallRiskLevel    string            `json:"overall_risk_level"` // CRITICAL, ELEVATED, MODERATE, HEALTHY
	CriticalLots        []ExpiryLotDetail `json:"critical_lots"`
	AIInsights          string            `json:"ai_insights"`
	AIProvider          string            `json:"ai_provider"`
}

// Traceability Dashboard Stats
type TraceabilityStats struct {
	TotalLots            int64   `json:"total_lots"`
	ActiveLotsWithStock  int64   `json:"active_lots_with_stock"`
	ExpiredLots          int64   `json:"expired_lots"`
	NearExpiryLots       int64   `json:"near_expiry_lots"`
	TotalTraceRecords    int64   `json:"total_trace_records"`
	ActiveRecalls        int64   `json:"active_recalls"`
	TotalQuarantinedStock float64 `json:"total_quarantined_stock"`
}
