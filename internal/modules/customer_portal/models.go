package customer_portal

import (
	"time"

	"erp-golang/internal/modules/account"
	"erp-golang/internal/modules/product"
	"erp-golang/internal/modules/sale"
	"gorm.io/gorm"
)

// CustomerPortalAccount represents a customer credential and self-service profile
type CustomerPortalAccount struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	PartnerID      uint           `gorm:"index;not null" json:"partner_id"`
	CustomerName   string         `gorm:"size:255;not null" json:"customer_name"`
	ContactName    string         `gorm:"size:255" json:"contact_name"`
	Email          string         `gorm:"size:255;not null" json:"email"`
	Phone          string         `gorm:"size:64" json:"phone"`
	AccessToken    string         `gorm:"size:128;uniqueIndex;not null" json:"access_token"`
	IsActive       bool           `gorm:"default:true" json:"is_active"`
	CreditLimit    float64        `gorm:"default:0" json:"credit_limit"`
	LastLoginAt    *time.Time     `json:"last_login_at"`
	TotalLogins    int            `gorm:"default:0" json:"total_logins"`
	AllowedReorder bool           `gorm:"default:true" json:"allowed_reorder"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

func (CustomerPortalAccount) TableName() string {
	return "customer_portal_accounts"
}

// B2BOrderRequest represents a customer self-service order / repeat purchase request
type B2BOrderRequest struct {
	ID                    uint                 `gorm:"primaryKey" json:"id"`
	RequestNumber         string               `gorm:"size:64;uniqueIndex;not null" json:"request_number"`
	PortalAccountID       uint                 `gorm:"index;not null" json:"portal_account_id"`
	PortalAccount         *CustomerPortalAccount `gorm:"foreignKey:PortalAccountID" json:"portal_account,omitempty"`
	PartnerID             uint                 `gorm:"index;not null" json:"partner_id"`
	CompanyID             uint                 `gorm:"index;default:1" json:"company_id"`
	SaleOrderID           *uint                `gorm:"index" json:"sale_order_id"`
	SaleOrder             *sale.SaleOrder      `gorm:"foreignKey:SaleOrderID" json:"sale_order,omitempty"`
	SubtotalAmount        float64              `gorm:"type:decimal(16,2);default:0" json:"subtotal_amount"`
	TaxAmount             float64              `gorm:"type:decimal(16,2);default:0" json:"tax_amount"`
	TotalAmount           float64              `gorm:"type:decimal(16,2);default:0" json:"total_amount"`
	PaymentMethod         string               `gorm:"size:64;default:'credit_terms'" json:"payment_method"` // credit_terms, bank_transfer, payment_gateway
	DeliveryAddress       string               `gorm:"type:text" json:"delivery_address"`
	RequestedDeliveryDate *time.Time           `json:"requested_delivery_date"`
	CustomerNotes         string               `gorm:"type:text" json:"customer_notes"`
	Status                string               `gorm:"size:32;default:'submitted'" json:"status"` // submitted, approved, converted_to_so, rejected, cancelled
	Lines                 []B2BOrderLine       `gorm:"foreignKey:OrderRequestID" json:"lines"`
	CreatedAt             time.Time            `json:"created_at"`
	UpdatedAt             time.Time            `json:"updated_at"`
	DeletedAt             gorm.DeletedAt       `gorm:"index" json:"-"`
}

func (B2BOrderRequest) TableName() string {
	return "customer_portal_order_requests"
}

// B2BOrderLine represents an individual line item in a customer order request
type B2BOrderLine struct {
	ID             uint             `gorm:"primaryKey" json:"id"`
	OrderRequestID uint             `gorm:"index;not null" json:"order_request_id"`
	ProductID      uint             `gorm:"index;not null" json:"product_id"`
	Product        *product.Product `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	Quantity       float64          `gorm:"type:decimal(12,2);not null" json:"quantity"`
	UnitPrice      float64          `gorm:"type:decimal(16,2);not null" json:"unit_price"`
	Subtotal       float64          `gorm:"type:decimal(16,2);not null" json:"subtotal"`
	Notes          string           `gorm:"size:255" json:"notes"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
}

func (B2BOrderLine) TableName() string {
	return "customer_portal_order_lines"
}

// RMARequest represents Return Merchandise Authorization requested by customer
type RMARequest struct {
	ID                  uint                 `gorm:"primaryKey" json:"id"`
	RMANumber           string               `gorm:"size:64;uniqueIndex;not null" json:"rma_number"`
	PortalAccountID     uint                 `gorm:"index;not null" json:"portal_account_id"`
	PortalAccount       *CustomerPortalAccount `gorm:"foreignKey:PortalAccountID" json:"portal_account,omitempty"`
	PartnerID           uint                 `gorm:"index;not null" json:"partner_id"`
	SaleOrderID         *uint                `gorm:"index" json:"sale_order_id"`
	SaleOrder           *sale.SaleOrder      `gorm:"foreignKey:SaleOrderID" json:"sale_order,omitempty"`
	InvoiceID           *uint                `gorm:"index" json:"invoice_id"`
	ProductID           uint                 `gorm:"index;not null" json:"product_id"`
	Product             *product.Product     `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	Quantity            float64              `gorm:"type:decimal(12,2);not null" json:"quantity"`
	ReasonCode          string               `gorm:"size:64;not null" json:"reason_code"` // defective, damaged_in_transit, wrong_item, expired, mismatch_spec
	Description         string               `gorm:"type:text;not null" json:"description"`
	EvidenceURL         string               `gorm:"size:512" json:"evidence_url"`
	ResolutionRequested string               `gorm:"size:64;default:'replace_item'" json:"resolution_requested"` // replace_item, refund_credit_note, repair
	ResolutionAction    string               `gorm:"size:64" json:"resolution_action"`
	Status              string               `gorm:"size:32;default:'submitted'" json:"status"` // submitted, under_review, approved, rejected, resolved
	CreditNoteMoveID    *uint                `gorm:"index" json:"credit_note_move_id"`
	AdminNotes          string               `gorm:"type:text" json:"admin_notes"`
	ResolvedAt          *time.Time           `json:"resolved_at"`
	CreatedAt           time.Time            `json:"created_at"`
	UpdatedAt           time.Time            `json:"updated_at"`
	DeletedAt           gorm.DeletedAt       `gorm:"index" json:"-"`
}

func (RMARequest) TableName() string {
	return "customer_portal_rma_requests"
}

// CustomerPaymentNotification represents customer proof of bank transfer / payment submission
type CustomerPaymentNotification struct {
	ID                 uint                 `gorm:"primaryKey" json:"id"`
	NotificationNumber string               `gorm:"size:64;uniqueIndex;not null" json:"notification_number"`
	PortalAccountID    uint                 `gorm:"index;not null" json:"portal_account_id"`
	PortalAccount      *CustomerPortalAccount `gorm:"foreignKey:PortalAccountID" json:"portal_account,omitempty"`
	PartnerID          uint                 `gorm:"index;not null" json:"partner_id"`
	InvoiceID          uint                 `gorm:"index;not null" json:"invoice_id"`
	Invoice            *account.AccountMove `gorm:"foreignKey:InvoiceID" json:"invoice,omitempty"`
	AmountPaid         float64              `gorm:"type:decimal(16,2);not null" json:"amount_paid"`
	PaymentDate        time.Time            `json:"payment_date"`
	SenderBank         string               `gorm:"size:128" json:"sender_bank"`
	SenderAccountName  string               `gorm:"size:255" json:"sender_account_name"`
	SenderAccountNo    string               `gorm:"size:64" json:"sender_account_no"`
	TargetJournalID    uint                 `gorm:"index;default:1" json:"target_journal_id"`
	ReferenceNumber    string               `gorm:"size:128" json:"reference_number"`
	ProofURL           string               `gorm:"size:512" json:"proof_url"`
	Status             string               `gorm:"size:32;default:'pending_verification'" json:"status"` // pending_verification, confirmed, rejected
	AccountPaymentID   *uint                `gorm:"index" json:"account_payment_id"`
	VerifiedByUserID   *uint                `gorm:"index" json:"verified_by_user_id"`
	VerifiedAt         *time.Time           `json:"verified_at"`
	Notes              string               `gorm:"type:text" json:"notes"`
	CreatedAt          time.Time            `json:"created_at"`
	UpdatedAt          time.Time            `json:"updated_at"`
	DeletedAt          gorm.DeletedAt       `gorm:"index" json:"-"`
}

func (CustomerPaymentNotification) TableName() string {
	return "customer_portal_payment_notifications"
}

// --- DTO & Analytics Structs ---

type CreateCustomerAccountRequest struct {
	PartnerID      uint    `json:"partner_id" binding:"required"`
	CustomerName   string  `json:"customer_name" binding:"required"`
	ContactName    string  `json:"contact_name"`
	Email          string  `json:"email" binding:"required,email"`
	Phone          string  `json:"phone"`
	CreditLimit    float64 `json:"credit_limit"`
	AllowedReorder bool    `json:"allowed_reorder"`
}

type CreateB2BOrderRequestPayload struct {
	DeliveryAddress       string            `json:"delivery_address"`
	RequestedDeliveryDate string            `json:"requested_delivery_date"`
	PaymentMethod         string            `json:"payment_method"`
	CustomerNotes         string            `json:"customer_notes"`
	Lines                 []B2BOrderLineReq `json:"lines" binding:"required,min=1"`
}

type B2BOrderLineReq struct {
	ProductID uint    `json:"product_id" binding:"required"`
	Quantity  float64 `json:"quantity" binding:"required,gt=0"`
	UnitPrice float64 `json:"unit_price"`
	Notes     string  `json:"notes"`
}

type SubmitRMARequestPayload struct {
	SaleOrderID         *uint   `json:"sale_order_id"`
	InvoiceID           *uint   `json:"invoice_id"`
	ProductID           uint    `json:"product_id" binding:"required"`
	Quantity            float64 `json:"quantity" binding:"required,gt=0"`
	ReasonCode          string  `json:"reason_code" binding:"required"`
	Description         string  `json:"description" binding:"required"`
	EvidenceURL         string  `json:"evidence_url"`
	ResolutionRequested string  `json:"resolution_requested"`
}

type SubmitPaymentNotificationPayload struct {
	InvoiceID         uint    `json:"invoice_id" binding:"required"`
	AmountPaid        float64 `json:"amount_paid" binding:"required,gt=0"`
	PaymentDate       string  `json:"payment_date"`
	SenderBank        string  `json:"sender_bank"`
	SenderAccountName string  `json:"sender_account_name"`
	SenderAccountNo   string  `json:"sender_account_no"`
	ReferenceNumber   string  `json:"reference_number"`
	ProofURL          string  `json:"proof_url"`
	Notes             string  `json:"notes"`
}

type CustomerPortalStats struct {
	TotalCustomerAccounts int     `json:"total_customer_accounts"`
	ActiveAccounts        int     `json:"active_accounts"`
	TotalB2BOrders        int     `json:"total_b2b_orders"`
	ApprovedB2BOrders     int     `json:"approved_b2b_orders"`
	TotalRMAs             int     `json:"total_rmas"`
	PendingRMAs           int     `json:"pending_rmas"`
	TotalPendingPayments  int     `json:"total_pending_payments"`
	TotalB2BOrderValue    float64 `json:"total_b2b_order_value"`
	OutstandingARBalance  float64 `json:"outstanding_ar_balance"`
}

type CustomerStatementSummary struct {
	PartnerID            uint                   `json:"partner_id"`
	CustomerName         string                 `json:"customer_name"`
	CreditLimit          float64                `json:"credit_limit"`
	TotalOutstanding     float64                `json:"total_outstanding"`
	AgingCurrent         float64                `json:"aging_current"` // not overdue
	Aging1To30           float64                `json:"aging_1_to_30"` // 1-30 days overdue
	Aging31To60          float64                `json:"aging_31_to_60"`
	AgingOver60          float64                `json:"aging_over_60"`
	Invoices             []account.AccountMove  `json:"invoices"`
}

type AICustomerHealthScorecard struct {
	PartnerID                 uint      `json:"partner_id"`
	CustomerName              string    `json:"customer_name"`
	TotalOrdersCount          int       `json:"total_orders_count"`
	TotalOrderLifetimeValue   float64   `json:"total_order_lifetime_value"`
	AverageOrderValue         float64   `json:"average_order_value"`
	PaymentPunctualityScore   float64   `json:"payment_punctuality_score"` // 0-100
	ReturnRatePercentage      float64   `json:"return_rate_percentage"`    // RMA / Orders
	OverallHealthScore        float64   `json:"overall_health_score"`      // 0-100
	ChurnRiskLevel            string    `json:"churn_risk_level"`          // LOW, MEDIUM, HIGH
	TierBadge                 string    `json:"tier_badge"`                // VIP PLATINUM, GOLD, REGULAR, AT_RISK
	ExecutiveSummary          string    `json:"executive_summary"`
	RecommendedActions        []string  `json:"recommended_actions"`
	AIProvider                string    `json:"ai_provider"`
	EvaluatedAt               time.Time `json:"evaluated_at"`
}
