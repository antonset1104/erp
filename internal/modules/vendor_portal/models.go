package vendor_portal

import (
	"time"

	"erp-golang/internal/modules/purchase"
	"gorm.io/gorm"
)

// VendorPortalAccount represents a secure self-service portal account for a supplier
type VendorPortalAccount struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	PartnerID   uint           `gorm:"not null;index" json:"partner_id"`
	VendorName  string         `gorm:"size:255;not null" json:"vendor_name"`
	ContactName string         `gorm:"size:255" json:"contact_name"`
	Email       string         `gorm:"size:255;not null;index" json:"email"`
	Phone       string         `gorm:"size:100" json:"phone"`
	AccessToken string         `gorm:"size:128;uniqueIndex;not null" json:"access_token"`
	IsActive    bool           `gorm:"default:true" json:"is_active"`
	LastLoginAt *time.Time     `json:"last_login_at"`
	TotalLogins int            `gorm:"default:0" json:"total_logins"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// VendorQuotationBid represents a price quotation bid submitted by a vendor for an RFQ
type VendorQuotationBid struct {
	ID                   uint                     `gorm:"primaryKey" json:"id"`
	BidNumber            string                   `gorm:"size:64;uniqueIndex;not null" json:"bid_number"` // BID/2026/00001
	RFQReference         string                   `gorm:"size:64;index" json:"rfq_reference"`             // PO00001 (RFQ) or PR/2026/00001
	PurchaseOrderID      *uint                    `gorm:"index" json:"purchase_order_id"`
	PartnerID            uint                     `gorm:"not null;index" json:"partner_id"`
	VendorName           string                   `gorm:"size:255" json:"vendor_name"`
	TotalBidAmount       float64                  `gorm:"type:decimal(16,2);not null" json:"total_bid_amount"`
	DeliveryLeadTimeDays int                      `gorm:"default:7" json:"delivery_lead_time_days"`
	PaymentTerms         string                   `gorm:"size:100;default:'NET 30'" json:"payment_terms"`
	Status               string                   `gorm:"size:32;default:'submitted'" json:"status"` // submitted, accepted, rejected
	Notes                string                   `gorm:"type:text" json:"notes"`
	SubmittedAt          time.Time                `json:"submitted_at"`
	Lines                []VendorQuotationBidLine `gorm:"foreignKey:BidID" json:"lines,omitempty"`
	CreatedAt            time.Time                `json:"created_at"`
	UpdatedAt            time.Time                `json:"updated_at"`
	DeletedAt            gorm.DeletedAt           `gorm:"index" json:"-"`
}

// VendorQuotationBidLine represents individual item bids
type VendorQuotationBidLine struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	BidID       uint           `gorm:"not null;index" json:"bid_id"`
	ProductID   uint           `gorm:"not null;index" json:"product_id"`
	ProductName string         `gorm:"size:255" json:"product_name"`
	ProductCode string         `gorm:"size:100" json:"product_code"`
	Quantity    float64        `gorm:"not null" json:"quantity"`
	UnitPrice   float64        `gorm:"type:decimal(16,2);not null" json:"unit_price"`
	Subtotal    float64        `gorm:"type:decimal(16,2);not null" json:"subtotal"`
	Notes       string         `gorm:"size:255" json:"notes"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// VendorASN represents Advance Shipping Notice / Surat Jalan Pemasok
type VendorASN struct {
	ID                   uint            `gorm:"primaryKey" json:"id"`
	ASNNumber            string          `gorm:"size:64;uniqueIndex;not null" json:"asn_number"` // ASN/2026/00001
	PurchaseOrderID      uint            `gorm:"not null;index" json:"purchase_order_id"`
	PurchaseOrderRef     string          `gorm:"size:64;index" json:"purchase_order_ref"`
	PartnerID            uint            `gorm:"not null;index" json:"partner_id"`
	VendorName           string          `gorm:"size:255" json:"vendor_name"`
	CarrierName          string          `gorm:"size:100" json:"carrier_name"` // e.g. JNE Trucking, Internal Armada
	TrackingRef          string          `gorm:"size:100" json:"tracking_ref"` // No. Resi / Surat Jalan Supplier
	DriverName           string          `gorm:"size:100" json:"driver_name"`
	VehicleNumber        string          `gorm:"size:50" json:"vehicle_number"` // e.g. B 9812 XYZ
	EstimatedArrivalDate time.Time       `json:"estimated_arrival_date"`
	ActualArrivalDate    *time.Time      `json:"actual_arrival_date"`
	Status               string          `gorm:"size:32;default:'in_transit'" json:"status"` // in_transit, received, cancelled
	Notes                string          `gorm:"type:text" json:"notes"`
	Lines                []VendorASNLine `gorm:"foreignKey:ASNID" json:"lines,omitempty"`
	CreatedAt            time.Time       `json:"created_at"`
	UpdatedAt            time.Time       `json:"updated_at"`
	DeletedAt            gorm.DeletedAt  `gorm:"index" json:"-"`
}

// VendorASNLine represents individual goods within the shipping notice
type VendorASNLine struct {
	ID               uint           `gorm:"primaryKey" json:"id"`
	ASNID            uint           `gorm:"not null;index" json:"asn_id"`
	ProductID        uint           `gorm:"not null;index" json:"product_id"`
	ProductName      string         `gorm:"size:255" json:"product_name"`
	QuantityShipped  float64        `gorm:"not null" json:"quantity_shipped"`
	QuantityReceived float64        `gorm:"default:0" json:"quantity_received"`
	LotNumber        string         `gorm:"size:100" json:"lot_number"`
	Notes            string         `gorm:"size:255" json:"notes"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
}

// VendorInvoiceSubmission represents electronic bill submitted by vendor for 3-way matching
type VendorInvoiceSubmission struct {
	ID                    uint           `gorm:"primaryKey" json:"id"`
	SubmissionNumber      string         `gorm:"size:64;uniqueIndex;not null" json:"submission_number"` // VINV/2026/00001
	PurchaseOrderID       uint           `gorm:"not null;index" json:"purchase_order_id"`
	PurchaseOrderRef      string         `gorm:"size:64;index" json:"purchase_order_ref"`
	PartnerID             uint           `gorm:"not null;index" json:"partner_id"`
	VendorName            string         `gorm:"size:255" json:"vendor_name"`
	InvoiceNumberSupplier string         `gorm:"size:100;not null" json:"invoice_number_supplier"`
	InvoiceDate           time.Time      `json:"invoice_date"`
	DueDate               time.Time      `json:"due_date"`
	Subtotal              float64        `gorm:"type:decimal(16,2);not null" json:"subtotal"`
	TaxAmount             float64        `gorm:"type:decimal(16,2);default:0" json:"tax_amount"`
	TotalAmount           float64        `gorm:"type:decimal(16,2);not null" json:"total_amount"`
	TaxInvoiceNumber      string         `gorm:"size:100" json:"tax_invoice_number"` // No Faktur Pajak e-Faktur
	FileAttachmentURL     string         `gorm:"size:512" json:"file_attachment_url"`
	MatchingStatus        string         `gorm:"size:32;default:'pending_match'" json:"matching_status"` // pending_match, matched_3way, discrepancy_price, discrepancy_qty, rejected
	DiscrepancyNotes      string         `gorm:"type:text" json:"discrepancy_notes"`
	AccountMoveID         *uint          `gorm:"index" json:"account_move_id"`              // Linked ERP Vendor Bill
	Status                string         `gorm:"size:32;default:'submitted'" json:"status"` // submitted, verified, approved, rejected
	Notes                 string         `gorm:"type:text" json:"notes"`
	CreatedAt             time.Time      `json:"created_at"`
	UpdatedAt             time.Time      `json:"updated_at"`
	DeletedAt             gorm.DeletedAt `gorm:"index" json:"-"`
}

// AIVendorPerformanceRating represents AI evaluation of vendor reliability
type AIVendorPerformanceRating struct {
	PartnerID                 uint      `json:"partner_id"`
	VendorName                string    `json:"vendor_name"`
	TotalOrders               int       `json:"total_orders"`
	OnTimeDeliveryRate        float64   `json:"on_time_delivery_rate"`       // e.g. 96.5%
	PriceCompetitivenessScore float64   `json:"price_competitiveness_score"` // 0 - 100
	QualityAcceptanceRate     float64   `json:"quality_acceptance_rate"`     // e.g. 99.1%
	OverallRatingScore        float64   `json:"overall_rating_score"`        // 0 - 100
	TierBadge                 string    `json:"tier_badge"`                  // platinum, gold, silver, bronze
	ExecutiveSummary          string    `json:"executive_summary"`
	ActionableRecommendations []string  `json:"actionable_recommendations"`
	AIProvider                string    `json:"ai_provider"`
	EvaluatedAt               time.Time `json:"evaluated_at"`
}

// Request and Response DTOs
type CreateVendorAccountRequest struct {
	PartnerID   uint   `json:"partner_id" binding:"required"`
	VendorName  string `json:"vendor_name" binding:"required"`
	ContactName string `json:"contact_name"`
	Email       string `json:"email" binding:"required"`
	Phone       string `json:"phone"`
}

type SubmitBidLineRequest struct {
	ProductID   uint    `json:"product_id" binding:"required"`
	ProductName string  `json:"product_name"`
	ProductCode string  `json:"product_code"`
	Quantity    float64 `json:"quantity" binding:"required"`
	UnitPrice   float64 `json:"unit_price" binding:"required"`
	Notes       string  `json:"notes"`
}

type SubmitBidRequest struct {
	RFQReference         string                 `json:"rfq_reference" binding:"required"`
	PurchaseOrderID      *uint                  `json:"purchase_order_id"`
	DeliveryLeadTimeDays int                    `json:"delivery_lead_time_days"`
	PaymentTerms         string                 `json:"payment_terms"`
	Notes                string                 `json:"notes"`
	Lines                []SubmitBidLineRequest `json:"lines" binding:"required,min=1"`
}

type SubmitASNLineRequest struct {
	ProductID       uint    `json:"product_id" binding:"required"`
	ProductName     string  `json:"product_name"`
	QuantityShipped float64 `json:"quantity_shipped" binding:"required"`
	LotNumber       string  `json:"lot_number"`
	Notes           string  `json:"notes"`
}

type SubmitASNRequest struct {
	PurchaseOrderID      uint                   `json:"purchase_order_id" binding:"required"`
	CarrierName          string                 `json:"carrier_name"`
	TrackingRef          string                 `json:"tracking_ref"`
	DriverName           string                 `json:"driver_name"`
	VehicleNumber        string                 `json:"vehicle_number"`
	EstimatedArrivalDate string                 `json:"estimated_arrival_date"` // YYYY-MM-DD
	Notes                string                 `json:"notes"`
	Lines                []SubmitASNLineRequest `json:"lines" binding:"required,min=1"`
}

type SubmitVendorInvoiceRequest struct {
	PurchaseOrderID       uint    `json:"purchase_order_id" binding:"required"`
	InvoiceNumberSupplier string  `json:"invoice_number_supplier" binding:"required"`
	InvoiceDate           string  `json:"invoice_date" binding:"required"` // YYYY-MM-DD
	DueDate               string  `json:"due_date"`
	Subtotal              float64 `json:"subtotal" binding:"required"`
	TaxAmount             float64 `json:"tax_amount"`
	TotalAmount           float64 `json:"total_amount" binding:"required"`
	TaxInvoiceNumber      string  `json:"tax_invoice_number"`
	FileAttachmentURL     string  `json:"file_attachment_url"`
	Notes                 string  `json:"notes"`
}

type ThreeWayMatchResult struct {
	SubmissionID      uint     `json:"submission_id"`
	MatchingStatus    string   `json:"matching_status"` // matched_3way, discrepancy_price, discrepancy_qty
	IsMatch           bool     `json:"is_match"`
	ExpectedAmount    float64  `json:"expected_amount"`
	BilledAmount      float64  `json:"billed_amount"`
	VarianceAmount    float64  `json:"variance_amount"`
	DiscrepancyDetail []string `json:"discrepancy_detail"`
	GeneratedBillID   *uint    `json:"generated_bill_id,omitempty"`
}

type VendorPortalStats struct {
	TotalVendorAccounts   int64   `json:"total_vendor_accounts"`
	ActiveVendorAccounts  int64   `json:"active_vendor_accounts"`
	TotalBidsSubmitted    int64   `json:"total_bids_submitted"`
	AcceptedBids          int64   `json:"accepted_bids"`
	TotalASNsInTransit    int64   `json:"total_asns_in_transit"`
	ReceivedASNs          int64   `json:"received_asns"`
	PendingInvoiceMatches int64   `json:"pending_invoice_matches"`
	Matched3WayInvoices   int64   `json:"matched_3way_invoices"`
	TotalBilledAmount     float64 `json:"total_billed_amount"`
}

// VendorPortalRFQView represents an open RFQ view for external vendor portal
type VendorPortalRFQView struct {
	ID             uint                         `json:"id"`
	Name           string                       `json:"name"`
	DateOrder      string                       `json:"date_order"`
	State          string                       `json:"state"`
	Notes          string                       `json:"notes"`
	Lines          []purchase.PurchaseOrderLine `json:"lines"`
	AlreadyBid     bool                         `json:"already_bid"`
	ExistingBidNum string                       `json:"existing_bid_num,omitempty"`
}
