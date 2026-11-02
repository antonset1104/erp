package subscription

import (
	"erp-golang/internal/core/models"
	"erp-golang/internal/modules/base"
	"erp-golang/internal/modules/product"
)

// State constants for Subscription Lifecycle
const (
	StateDraft      = "draft"
	StateInProgress = "in_progress"
	StatePaused     = "paused"
	StateChurned    = "churned"
	StateClosed     = "closed"
)

// Recurring Interval constants
const (
	IntervalDaily     = "daily"
	IntervalWeekly    = "weekly"
	IntervalMonthly   = "monthly"
	IntervalQuarterly = "quarterly"
	IntervalYearly    = "yearly"
)

// SubscriptionTemplate represents contract templates (Enterprise ERP sale.subscription.template)
type SubscriptionTemplate struct {
	models.ModelBase
	Code               string  `gorm:"size:64;not null;uniqueIndex" json:"code"` // e.g. "TMPL-MONTHLY-PRO"
	Name               string  `gorm:"size:128;not null" json:"name"`            // e.g. "Cloud SaaS Pro (Monthly)"
	RecurringInterval  string  `gorm:"size:32;not null;default:'monthly'" json:"recurring_interval"` // daily, weekly, monthly, quarterly, yearly
	RecurringRuleCount int     `gorm:"default:1" json:"recurring_rule_count"`   // e.g. 1 month, 3 months
	InvoiceJournalID   uint    `gorm:"default:1" json:"invoice_journal_id"`     // Default Customer Invoices journal
	AutoPostInvoice    bool    `gorm:"default:true" json:"auto_post_invoice"`   // Auto post invoice upon creation
	Description        string  `gorm:"type:text" json:"description"`
	Active             bool    `gorm:"default:true" json:"active"`
}

// Subscription represents recurring subscription contracts (Enterprise ERP sale.subscription)
type Subscription struct {
	models.ModelBase
	Code                   string                    `gorm:"size:64;not null;uniqueIndex" json:"code"` // e.g. "SUB/2026/00001"
	Name                   string                    `gorm:"size:255;not null" json:"name"`            // e.g. "Cloud ERP Enterprise - PT Maju Bersama"
	PartnerID              uint                      `gorm:"not null;index" json:"partner_id"`
	Partner                base.Partner              `gorm:"foreignKey:PartnerID" json:"partner,omitempty"`
	TemplateID             uint                      `gorm:"not null;index" json:"template_id"`
	Template               SubscriptionTemplate      `gorm:"foreignKey:TemplateID" json:"template,omitempty"`
	State                  string                    `gorm:"size:32;default:'draft';index" json:"state"` // draft, in_progress, paused, churned, closed
	StartDate              string                    `gorm:"size:32;not null" json:"start_date"`         // "YYYY-MM-DD"
	EndDate                string                    `gorm:"size:32" json:"end_date"`                    // "YYYY-MM-DD" (optional for indefinite)
	NextInvoiceDate        string                    `gorm:"size:32;index" json:"next_invoice_date"`     // "YYYY-MM-DD"
	RecurringInterval      string                    `gorm:"size:32;default:'monthly'" json:"recurring_interval"`
	RecurringIntervalCount int                       `gorm:"default:1" json:"recurring_interval_count"`
	Currency               string                    `gorm:"size:16;default:'IDR'" json:"currency"`
	RecurringAmount        float64                   `gorm:"default:0" json:"recurring_amount"`          // Sum of lines per billing cycle
	TotalMRR               float64                   `gorm:"default:0;index" json:"total_mrr"`           // Monthly Recurring Revenue normalized
	TotalARR               float64                   `gorm:"default:0" json:"total_arr"`                 // Annual Recurring Revenue (MRR * 12)
	InvoicedCount          int                       `gorm:"default:0" json:"invoiced_count"`            // Total billing cycles invoiced
	TotalInvoicedAmount    float64                   `gorm:"default:0" json:"total_invoiced_amount"`     // Cumulative revenue billed
	ChurnReason            string                    `gorm:"size:255" json:"churn_reason"`
	ChurnDate              string                    `gorm:"size:32" json:"churn_date"`
	AutoRenew              bool                      `gorm:"default:true" json:"auto_renew"`
	PaymentMethod          string                    `gorm:"size:64;default:'bank_transfer'" json:"payment_method"` // bank_transfer, credit_card, auto_debit
	Notes                  string                    `gorm:"type:text" json:"notes"`
	Lines                  []SubscriptionLine        `gorm:"foreignKey:SubscriptionID" json:"lines"`
	Invoices               []SubscriptionInvoiceLog  `gorm:"foreignKey:SubscriptionID" json:"invoices"`
}

// SubscriptionLine represents recurring items/licenses in a subscription
type SubscriptionLine struct {
	models.ModelBase
	SubscriptionID uint            `gorm:"not null;index" json:"subscription_id"`
	ProductID      uint            `gorm:"not null;index" json:"product_id"`
	Product        product.Product `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	Name           string          `gorm:"size:255;not null" json:"name"` // Line label
	Quantity       float64         `gorm:"default:1" json:"quantity"`
	UnitPrice      float64         `gorm:"default:0" json:"price_unit"`
	Discount       float64         `gorm:"default:0" json:"discount"` // Discount percentage
	Subtotal       float64         `gorm:"default:0" json:"subtotal"`
	UoM            string          `gorm:"size:32;default:'Units'" json:"uom"`
}

// SubscriptionInvoiceLog records history of invoices created by recurring billing
type SubscriptionInvoiceLog struct {
	models.ModelBase
	SubscriptionID     uint    `gorm:"not null;index" json:"subscription_id"`
	InvoiceID          uint    `gorm:"not null;index" json:"invoice_id"`           // AccountMove ID
	InvoiceNumber      string  `gorm:"size:64;not null" json:"invoice_number"`    // INV/2026/00001
	InvoiceDate        string  `gorm:"size:32;not null" json:"invoice_date"`
	Amount             float64 `gorm:"default:0" json:"amount"`
	Status             string  `gorm:"size:32;default:'draft'" json:"status"`      // draft, posted, paid
	NextInvoiceDateSet string  `gorm:"size:32" json:"next_invoice_date_set"`
}

// SubscriptionStageLog tracks changes in subscription state & MRR for SaaS cohort auditing
type SubscriptionStageLog struct {
	models.ModelBase
	SubscriptionID uint    `gorm:"not null;index" json:"subscription_id"`
	OldState       string  `gorm:"size:32" json:"old_state"`
	NewState       string  `gorm:"size:32;not null" json:"new_state"`
	Reason         string  `gorm:"size:255" json:"reason"`
	MRRChange      float64 `gorm:"default:0" json:"mrr_change"` // Difference in active MRR (+ or -)
	ChangedAt      string  `gorm:"size:32;not null" json:"changed_at"`
	UserID         uint    `gorm:"default:1" json:"user_id"`
}

// ---------------------------------------------------------------------------
// DTOs & Request/Response Structs
// ---------------------------------------------------------------------------

// CreateSubscriptionRequest payload
type CreateSubscriptionRequest struct {
	Name                   string                          `json:"name" binding:"required"`
	PartnerID              uint                            `json:"partner_id" binding:"required"`
	TemplateID             uint                            `json:"template_id" binding:"required"`
	StartDate              string                          `json:"start_date" binding:"required"`
	EndDate                string                          `json:"end_date"`
	NextInvoiceDate        string                          `json:"next_invoice_date"`
	RecurringInterval      string                          `json:"recurring_interval"`
	RecurringIntervalCount int                             `json:"recurring_interval_count"`
	Currency               string                          `json:"currency"`
	PaymentMethod          string                          `json:"payment_method"`
	AutoRenew              *bool                           `json:"auto_renew"`
	Notes                  string                          `json:"notes"`
	Lines                  []CreateSubscriptionLineRequest `json:"lines" binding:"required,min=1"`
}

// CreateSubscriptionLineRequest payload
type CreateSubscriptionLineRequest struct {
	ProductID uint    `json:"product_id" binding:"required"`
	Name      string  `json:"name"`
	Quantity  float64 `json:"quantity" binding:"required,gt=0"`
	UnitPrice float64 `json:"price_unit" binding:"required,gte=0"`
	Discount  float64 `json:"discount"`
	UoM       string  `json:"uom"`
}

// UpdateSubscriptionRequest payload
type UpdateSubscriptionRequest struct {
	Name            string  `json:"name"`
	NextInvoiceDate string  `json:"next_invoice_date"`
	EndDate         string  `json:"end_date"`
	PaymentMethod   string  `json:"payment_method"`
	AutoRenew       *bool   `json:"auto_renew"`
	Notes           string  `json:"notes"`
}

// CreateTemplateRequest payload
type CreateTemplateRequest struct {
	Code               string `json:"code" binding:"required"`
	Name               string `json:"name" binding:"required"`
	RecurringInterval  string `json:"recurring_interval" binding:"required"`
	RecurringRuleCount int    `json:"recurring_rule_count"`
	InvoiceJournalID   uint   `json:"invoice_journal_id"`
	AutoPostInvoice    *bool  `json:"auto_post_invoice"`
	Description        string `json:"description"`
}

// BillingBatchRequest triggers recurring invoice generation for due contracts
type BillingBatchRequest struct {
	TargetDate string `json:"target_date"` // YYYY-MM-DD (defaults to today)
}

// BillingBatchResult provides summary of batch execution
type BillingBatchResult struct {
	TargetDate              string                     `json:"target_date"`
	ProcessedSubscriptions  int                        `json:"processed_subscriptions"`
	InvoicesGenerated       int                        `json:"invoices_generated"`
	TotalBilledRevenue      float64                    `json:"total_billed_revenue"`
	GeneratedInvoices       []SubscriptionInvoiceLog   `json:"generated_invoices"`
	Errors                  []string                   `json:"errors"`
}

// SaaSMetrics gives executive recurring revenue & cohort analytics
type SaaSMetrics struct {
	TotalSubscriptions        int64              `json:"total_subscriptions"`
	ActiveSubscriptions       int64              `json:"active_subscriptions"`
	PausedSubscriptions       int64              `json:"paused_subscriptions"`
	ChurnedSubscriptions      int64              `json:"churned_subscriptions"`
	DraftSubscriptions        int64              `json:"draft_subscriptions"`
	TotalLiveMRR              float64            `json:"total_live_mrr"`
	TotalLiveARR              float64            `json:"total_live_arr"`
	TotalInvoicedLifetime     float64            `json:"total_invoiced_lifetime"`
	TotalInvoicesCount        int64              `json:"total_invoices_count"`
	LogoChurnRatePct          float64            `json:"logo_churn_rate_pct"`
	RevenueChurnRatePct       float64            `json:"revenue_churn_rate_pct"`
	ARPU                      float64            `json:"arpu"` // Average Revenue Per User
	EstimatedLTV              float64            `json:"estimated_ltv"` // Customer Lifetime Value
	SubscriptionsByInterval   map[string]int64   `json:"subscriptions_by_interval"`
	IntervalMRRBreakdown      map[string]float64 `json:"interval_mrr_breakdown"`
}

// AIChurnRiskReport represents output from Cloudflare Workers AI Retention Copilot
type AIChurnRiskReport struct {
	SubscriptionID        uint     `json:"subscription_id"`
	SubscriptionCode      string   `json:"subscription_code"`
	PartnerName           string   `json:"partner_name"`
	ContractAgeDays       int      `json:"contract_age_days"`
	RiskLevel             string   `json:"risk_level"`       // "LOW", "MEDIUM", "HIGH", "CRITICAL"
	ChurnProbabilityPct   float64  `json:"churn_probability_pct"`
	KeyRiskDrivers        []string `json:"key_risk_drivers"`
	RetentionActionPlan   []string `json:"retention_action_plan"`
	ExecutiveSummary      string   `json:"executive_summary"`
	RecommendedDiscount   float64  `json:"recommended_discount_pct"`
	AIProvider            string   `json:"ai_provider"`
}
