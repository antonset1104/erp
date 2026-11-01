package intercompany

import (
	"time"

	"erp-golang/internal/core/models"
)

// IntercompanyGroup defines a multi-company holding or business group
type IntercompanyGroup struct {
	models.ModelBase
	GroupCode        string  `gorm:"size:64;not null;uniqueIndex" json:"group_code"`
	Name             string  `gorm:"size:255;not null" json:"name"`
	HoldingCompanyID uint    `gorm:"not null;index" json:"holding_company_id"`
	HoldingName      string  `gorm:"size:255" json:"holding_name"`
	Currency         string  `gorm:"size:10;default:'IDR'" json:"currency"`
	Description      string  `gorm:"size:500" json:"description"`
	IsActive         bool    `gorm:"default:true" json:"is_active"`
	TotalSubsidiaries int    `gorm:"default:0" json:"total_subsidiaries"`
}

// IntercompanyRelation defines bilateral trade & accounting relation between two entities
type IntercompanyRelation struct {
	models.ModelBase
	GroupID             uint    `gorm:"not null;index" json:"group_id"`
	RelationCode        string  `gorm:"size:64;not null;uniqueIndex" json:"relation_code"`
	Name                string  `gorm:"size:255;not null" json:"name"`
	
	// Source / Parent Company
	FromCompanyID       uint    `gorm:"not null;index" json:"from_company_id"`
	FromCompanyName     string  `gorm:"size:255" json:"from_company_name"`
	FromPartnerID       uint    `gorm:"index" json:"from_partner_id"` // Partner ID representing ToCompany in FromCompany

	// Target / Subsidiary Company
	ToCompanyID         uint    `gorm:"not null;index" json:"to_company_id"`
	ToCompanyName       string  `gorm:"size:255" json:"to_company_name"`
	ToPartnerID         uint    `gorm:"index" json:"to_partner_id"` // Partner ID representing FromCompany in ToCompany

	// Shareholding & Ownership
	OwnershipPct        float64 `gorm:"default:100.0" json:"ownership_pct"` // 100% Wholly Owned, 75% Majority, etc.
	ConsolidationMethod string  `gorm:"size:64;default:'full'" json:"consolidation_method"` // 'full' (IFRS 10), 'equity' (IAS 28)

	// Automation Toggles
	AutoMirrorSalePurchase bool `gorm:"default:true" json:"auto_mirror_sale_purchase"` // Auto generate PO from SO
	AutoMirrorInvoiceBill  bool `gorm:"default:true" json:"auto_mirror_invoice_bill"`   // Auto generate Bill from Invoice
	AutoEliminate          bool `gorm:"default:true" json:"auto_eliminate"`             // Include in consolidation elimination

	// Accounting Elimination Accounts
	IntercompanyARAccountID uint `json:"intercompany_ar_account_id"`
	IntercompanyAPAccountID uint `json:"intercompany_ap_account_id"`

	IsActive            bool    `gorm:"default:true" json:"is_active"`
}

// IntercompanyTransaction records dual-sided mirrored documents and sync states
type IntercompanyTransaction struct {
	models.ModelBase
	TransactionCode string  `gorm:"size:64;not null;uniqueIndex" json:"transaction_code"`
	GroupID         uint    `gorm:"not null;index" json:"group_id"`
	RelationID      uint    `gorm:"not null;index" json:"relation_id"`
	
	TransactionType string  `gorm:"size:64;not null" json:"transaction_type"` // sale_to_purchase, invoice_to_bill, elimination
	
	// Source Document
	SourceModel     string  `gorm:"size:64;not null" json:"source_model"` // sale.order, account.move
	SourceResID     uint    `gorm:"not null" json:"source_res_id"`
	SourceDocName   string  `gorm:"size:128;not null" json:"source_doc_name"`
	SourceCompanyID uint    `gorm:"not null" json:"source_company_id"`
	SourceAmount    float64 `gorm:"default:0" json:"source_amount"`

	// Mirrored / Target Document
	TargetModel     string  `gorm:"size:64;not null" json:"target_model"` // purchase.order, account.move
	TargetResID     uint    `gorm:"default:0" json:"target_res_id"`
	TargetDocName   string  `gorm:"size:128" json:"target_doc_name"`
	TargetCompanyID uint    `gorm:"not null" json:"target_company_id"`
	TargetAmount    float64 `gorm:"default:0" json:"target_amount"`

	SyncStatus      string  `gorm:"size:32;default:'synced'" json:"sync_status"` // synced, failed, pending, eliminated
	SyncMessage     string  `gorm:"size:500" json:"sync_message"`
	SyncedAt        time.Time `json:"synced_at"`
}

// ConsolidationRun stores multi-company financial consolidation executions
type ConsolidationRun struct {
	models.ModelBase
	RunCode                    string    `gorm:"size:64;not null;uniqueIndex" json:"run_code"`
	GroupID                    uint      `gorm:"not null;index" json:"group_id"`
	GroupName                  string    `gorm:"size:255" json:"group_name"`
	PeriodStartDate            time.Time `json:"period_start_date"`
	PeriodEndDate              time.Time `json:"period_end_date"`
	FiscalYear                 int       `gorm:"default:2026" json:"fiscal_year"`
	
	// Gross Financials (Aggregated across subsidiaries)
	TotalGrossAssets           float64   `gorm:"default:0" json:"total_gross_assets"`
	TotalGrossLiabilities      float64   `gorm:"default:0" json:"total_gross_liabilities"`
	TotalGrossEquity           float64   `gorm:"default:0" json:"total_gross_equity"`
	TotalGrossRevenue          float64   `gorm:"default:0" json:"total_gross_revenue"`
	TotalGrossExpenses         float64   `gorm:"default:0" json:"total_gross_expenses"`
	TotalGrossNetIncome        float64   `gorm:"default:0" json:"total_gross_net_income"`

	// Eliminations (Reciprocal transactions removed)
	EliminatedReceivables      float64   `gorm:"default:0" json:"eliminated_receivables"`
	EliminatedPayables         float64   `gorm:"default:0" json:"eliminated_payables"`
	EliminatedRevenue          float64   `gorm:"default:0" json:"eliminated_revenue"`
	EliminatedCOGS             float64   `gorm:"default:0" json:"eliminated_cogs"`
	EliminatedUnrealizedProfit float64   `gorm:"default:0" json:"eliminated_unrealized_profit"`

	// Net Consolidated Financials (IFRS 10 / PSAK 65)
	ConsolidatedAssets         float64   `gorm:"default:0" json:"consolidated_assets"`
	ConsolidatedLiabilities    float64   `gorm:"default:0" json:"consolidated_liabilities"`
	ConsolidatedEquity         float64   `gorm:"default:0" json:"consolidated_equity"`
	ConsolidatedRevenue        float64   `gorm:"default:0" json:"consolidated_revenue"`
	ConsolidatedExpenses       float64   `gorm:"default:0" json:"consolidated_expenses"`
	ConsolidatedNetIncome      float64   `gorm:"default:0" json:"consolidated_net_income"`

	// Minority / Non-Controlling Interest (NCI)
	NonControllingInterest     float64   `gorm:"default:0" json:"non_controlling_interest"` // Hak Minoritas
	ControllingInterest        float64   `gorm:"default:0" json:"controlling_interest"`     // Entitas Induk

	State                      string    `gorm:"size:32;default:'calculated'" json:"state"` // draft, calculated, posted, archived
	EliminationJournalID       uint      `json:"elimination_journal_id"`
	Notes                      string    `gorm:"size:1000" json:"notes"`
	CreatedByUID               uint      `json:"created_by_uid"`

	Lines                      []ConsolidationLine `gorm:"foreignKey:RunID" json:"lines,omitempty"`
}

// ConsolidationLine detailed line breakdown per subsidiary and financial category
type ConsolidationLine struct {
	models.ModelBase
	RunID               uint    `gorm:"not null;index" json:"run_id"`
	CompanyID           uint    `gorm:"not null;index" json:"company_id"`
	CompanyName         string  `gorm:"size:255;not null" json:"company_name"`
	OwnershipPct        float64 `gorm:"default:100.0" json:"ownership_pct"`
	FinancialCategory   string  `gorm:"size:64;not null" json:"financial_category"` // asset, liability, equity, revenue, expense
	AccountCode         string  `gorm:"size:64" json:"account_code"`
	AccountName         string  `gorm:"size:255" json:"account_name"`
	
	GrossAmount         float64 `gorm:"default:0" json:"gross_amount"`
	EliminationDebit    float64 `gorm:"default:0" json:"elimination_debit"`
	EliminationCredit   float64 `gorm:"default:0" json:"elimination_credit"`
	ConsolidatedAmount  float64 `gorm:"default:0" json:"consolidated_amount"`
	EliminationNote     string  `gorm:"size:255" json:"elimination_note"`
}

// ---------------------------------------------------------------------------
// DTOs & Request/Response Objects
// ---------------------------------------------------------------------------

type CreateGroupRequest struct {
	GroupCode        string `json:"group_code" binding:"required"`
	Name             string `json:"name" binding:"required"`
	HoldingCompanyID uint   `json:"holding_company_id" binding:"required"`
	Currency         string `json:"currency"`
	Description      string `json:"description"`
}

type CreateRelationRequest struct {
	GroupID             uint    `json:"group_id" binding:"required"`
	Name                string  `json:"name" binding:"required"`
	FromCompanyID       uint    `json:"from_company_id" binding:"required"`
	FromPartnerID       uint    `json:"from_partner_id"`
	ToCompanyID         uint    `json:"to_company_id" binding:"required"`
	ToPartnerID         uint    `json:"to_partner_id"`
	OwnershipPct        float64 `json:"ownership_pct"`
	ConsolidationMethod string  `json:"consolidation_method"`
	AutoMirrorSalePurchase *bool `json:"auto_mirror_sale_purchase"`
	AutoMirrorInvoiceBill  *bool `json:"auto_mirror_invoice_bill"`
	AutoEliminate          *bool `json:"auto_eliminate"`
}

type TriggerMirrorSaleRequest struct {
	SaleOrderID uint `json:"sale_order_id" binding:"required"`
}

type TriggerMirrorInvoiceRequest struct {
	InvoiceID uint `json:"invoice_id" binding:"required"`
}

type RunConsolidationRequest struct {
	GroupID         uint      `json:"group_id" binding:"required"`
	PeriodStartDate time.Time `json:"period_start_date" binding:"required"`
	PeriodEndDate   time.Time `json:"period_end_date" binding:"required"`
	FiscalYear      int       `json:"fiscal_year"`
	Notes           string    `json:"notes"`
}

type IntercompanyStats struct {
	TotalGroups                 int64   `json:"total_groups"`
	TotalRelations              int64   `json:"total_relations"`
	TotalMirroredTransactions   int64   `json:"total_mirrored_transactions"`
	TotalConsolidationRuns      int64   `json:"total_consolidation_runs"`
	TotalEliminatedVolume       float64 `json:"total_eliminated_volume"`
	LastConsolidatedRevenue     float64 `json:"last_consolidated_revenue"`
	LastConsolidatedNetIncome   float64 `json:"last_consolidated_net_income"`
}

type AIConsolidationAuditReport struct {
	GroupCode                 string            `json:"group_code"`
	GroupName                 string            `json:"group_name"`
	ComplianceStatus          string            `json:"compliance_status"` // COMPLIANT, WARNING, NON_COMPLIANT
	AuditScore                int               `json:"audit_score"`        // 0-100
	ExecutiveSummary          string            `json:"executive_summary"`
	ReciprocalVariances       []string          `json:"reciprocal_variances"`
	RecommendedEliminations   map[string]float64 `json:"recommended_eliminations"`
	NonControllingRisk        string            `json:"non_controlling_risk"`
	IFRS10Checklist           map[string]bool   `json:"ifrs10_checklist"`
	AIProvider                string            `json:"ai_provider"`
}
