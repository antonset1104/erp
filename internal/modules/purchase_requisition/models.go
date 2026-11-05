package purchase_requisition

import (
	"time"

	"erp-golang/internal/modules/base"
	"erp-golang/internal/modules/product"
	"erp-golang/internal/modules/purchase"

	"gorm.io/gorm"
)

// RequisitionType defines the operational tender mode
type RequisitionType string

const (
	TypePurchaseTender RequisitionType = "purchase_tender" // Tender pengadaan kompetitif (Call for Bids)
	TypeBlanketOrder   RequisitionType = "blanket_order"   // Blanket Purchase Order (Kontrak harga & volume jangka panjang)
)

// RequisitionState defines the lifecycle of a purchase requisition
type RequisitionState string

const (
	StateDraft      RequisitionState = "draft"       // Draf pengajuan internal
	StateConfirmed  RequisitionState = "confirmed"   // Disetujui atasan / siap tender
	StateInProgress RequisitionState = "in_progress" // Tender dibuka / RFQ disebar ke vendor
	StateDone       RequisitionState = "done"        // Selesai / PO terpilih dikonfirmasi
	StateCancelled  RequisitionState = "cancel"      // Dibatalkan
)

// PurchaseRequisition represents internal purchase agreement / tender (purchase.requisition)
type PurchaseRequisition struct {
	ID           uint             `gorm:"primaryKey" json:"id"`
	CreatedAt    time.Time        `json:"created_at"`
	UpdatedAt    time.Time        `json:"updated_at"`
	DeletedAt    gorm.DeletedAt   `gorm:"index" json:"-"`
	Name         string           `gorm:"type:varchar(64);uniqueIndex;not null" json:"name"` // e.g. PR/2026/00001
	Origin       string           `gorm:"type:varchar(128)" json:"origin"`                   // Dokumen pemicu / nomor memo
	Type         RequisitionType  `gorm:"type:varchar(32);default:'purchase_tender'" json:"type"`
	State        RequisitionState `gorm:"type:varchar(32);default:'draft';index" json:"state"`
	UserID       uint             `json:"user_id"` // Pembuat permintaan / Buyer
	User         *base.User       `gorm:"foreignKey:UserID" json:"user,omitempty"`
	VendorID     *uint            `json:"vendor_id,omitempty"` // Rekanan terpilih / Blanket vendor
	Vendor       *base.Partner    `gorm:"foreignKey:VendorID" json:"vendor,omitempty"`
	OrderingDate string           `gorm:"type:varchar(32)" json:"ordering_date"` // Tanggal pengajuan
	DateEnd      string           `gorm:"type:varchar(32)" json:"date_end"`      // Batas akhir tender (Deadline)
	ScheduleDate string           `gorm:"type:varchar(32)" json:"schedule_date"` // Jadwal pengiriman diharapkan
	CompanyID    uint             `gorm:"default:1" json:"company_id"`
	Description  string           `gorm:"type:text" json:"description"`
	TotalAmount  float64          `gorm:"type:decimal(15,2);default:0" json:"total_amount"` // Total estimasi plafon PR
	SelectedPOID *uint            `json:"selected_po_id,omitempty"`                         // PO pemenang tender

	Lines          []PurchaseRequisitionLine `gorm:"foreignKey:RequisitionID" json:"lines"`
	PurchaseOrders []purchase.PurchaseOrder  `gorm:"foreignKey:RequisitionID" json:"purchase_orders,omitempty"`
}

// PurchaseRequisitionLine represents item lines in requisition (purchase.requisition.line)
type PurchaseRequisitionLine struct {
	ID                uint            `gorm:"primaryKey" json:"id"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
	DeletedAt         gorm.DeletedAt  `gorm:"index" json:"-"`
	RequisitionID     uint            `gorm:"index;not null" json:"requisition_id"`
	ProductID         uint            `gorm:"not null" json:"product_id"`
	Product           product.Product `gorm:"foreignKey:ProductID" json:"product"`
	Description       string          `gorm:"type:varchar(255)" json:"description"`
	ProductQty        float64         `gorm:"type:decimal(12,2);not null" json:"product_qty"`  // Kuantitas diminta
	QtyOrdered        float64         `gorm:"type:decimal(12,2);default:0" json:"qty_ordered"` // Kuantitas terbit ke PO
	PriceUnit         float64         `gorm:"type:decimal(15,2);default:0" json:"price_unit"`  // Estimasi harga per unit
	Subtotal          float64         `gorm:"type:decimal(15,2);default:0" json:"subtotal"`
	AccountAnalyticID *uint           `json:"account_analytic_id,omitempty"` // Link ke cost center budget
}

// TableName overrides GORM default table name
func (PurchaseRequisition) TableName() string {
	return "purchase_requisitions"
}

// TableName overrides GORM default table name
func (PurchaseRequisitionLine) TableName() string {
	return "purchase_requisition_lines"
}

// DTOs for API requests & responses

type CreateRequisitionLineRequest struct {
	ProductID         uint    `json:"product_id" binding:"required"`
	Description       string  `json:"description"`
	ProductQty        float64 `json:"product_qty" binding:"required,gt=0"`
	PriceUnit         float64 `json:"price_unit"`
	AccountAnalyticID *uint   `json:"account_analytic_id"`
}

type CreateRequisitionRequest struct {
	Origin       string                         `json:"origin"`
	Type         RequisitionType                `json:"type"` // purchase_tender / blanket_order
	VendorID     *uint                          `json:"vendor_id"`
	OrderingDate string                         `json:"ordering_date"`
	DateEnd      string                         `json:"date_end"`
	ScheduleDate string                         `json:"schedule_date"`
	Description  string                         `json:"description"`
	Lines        []CreateRequisitionLineRequest `json:"lines" binding:"required,min=1"`
}

type GenerateRFQRequest struct {
	VendorID uint   `json:"vendor_id" binding:"required"`
	Notes    string `json:"notes"`
}

type ReleaseBlanketPORequest struct {
	VendorID uint               `json:"vendor_id"`
	Lines    []ReleasePOLineReq `json:"lines" binding:"required,min=1"`
	Notes    string             `json:"notes"`
}

type ReleasePOLineReq struct {
	RequisitionLineID uint     `json:"requisition_line_id"`
	ProductID         uint     `json:"product_id"`
	Qty               float64  `json:"qty" binding:"required,gt=0"`
	PriceUnit         *float64 `json:"price_unit"`
}

type RequisitionStats struct {
	TotalRequisitions  int64   `json:"total_requisitions"`
	DraftCount         int64   `json:"draft_count"`
	InProgressCount    int64   `json:"in_progress_count"`
	DoneCount          int64   `json:"done_count"`
	TotalEstimatedVal  float64 `json:"total_estimated_value"`
	BlanketOrdersCount int64   `json:"blanket_orders_count"`
	ActiveTendersCount int64   `json:"active_tenders_count"`
}
