package bank_reconciliation

import (
	"erp-golang/internal/core/models"
	"erp-golang/internal/modules/account"
	"erp-golang/internal/modules/base"
)

// AccountBankStatement corresponds to standard enterprise account.bank.statement
type AccountBankStatement struct {
	models.ModelBase
	Name            string                     `gorm:"size:64;not null;uniqueIndex" json:"name"` // e.g. BNK/STAT/2026/0001
	JournalID       uint                       `gorm:"not null;index" json:"journal_id"`
	Journal         account.AccountJournal     `gorm:"foreignKey:JournalID" json:"journal,omitempty"`
	Date            string                     `gorm:"size:32;not null" json:"date"` // Statement Date YYYY-MM-DD
	BalanceStart    float64                    `gorm:"default:0" json:"balance_start"`
	BalanceEnd      float64                    `gorm:"default:0" json:"balance_end"`              // Computed = BalanceStart + sum(lines.Amount)
	BalanceEndReal  float64                    `gorm:"default:0" json:"balance_end_real"`         // From statement file/document
	Difference      float64                    `gorm:"default:0" json:"difference"`               // BalanceEndReal - BalanceEnd
	State           string                     `gorm:"size:32;default:'open';index" json:"state"` // "open", "confirm", "cancel"
	LineIDs         []AccountBankStatementLine `gorm:"foreignKey:StatementID" json:"line_ids,omitempty"`
	TotalLines      int                        `gorm:"default:0" json:"total_lines"`
	ReconciledLines int                        `gorm:"default:0" json:"reconciled_lines"`
	Note            string                     `gorm:"type:text" json:"note"`
}

// AccountBankStatementLine corresponds to standard enterprise account.bank.statement.line
type AccountBankStatementLine struct {
	models.ModelBase
	StatementID         uint                  `gorm:"not null;index" json:"statement_id"`
	Statement           *AccountBankStatement `gorm:"foreignKey:StatementID" json:"statement,omitempty"`
	Date                string                `gorm:"size:32;not null" json:"date"`
	Name                string                `gorm:"size:255;not null" json:"name"` // Transaction Memo / Description
	PartnerID           *uint                 `gorm:"index" json:"partner_id,omitempty"`
	Partner             *base.Partner         `gorm:"foreignKey:PartnerID" json:"partner,omitempty"`
	Amount              float64               `gorm:"not null" json:"amount"` // Positive = Inflow (CR), Negative = Outflow (DB)
	Ref                 string                `gorm:"size:128" json:"ref"`    // Bank Ref / Check No / Transfer ID
	IsReconciled        bool                  `gorm:"default:false;index" json:"is_reconciled"`
	MoveID              *uint                 `gorm:"index" json:"move_id,omitempty"`      // Reconciled AccountMove
	MoveLineID          *uint                 `gorm:"index" json:"move_line_id,omitempty"` // Specific MoveLine
	PaymentID           *uint                 `gorm:"index" json:"payment_id,omitempty"`   // Created Payment
	ReconcileRuleID     *uint                 `json:"reconcile_rule_id,omitempty"`
	AutoMatchConfidence float64               `gorm:"default:0" json:"auto_match_confidence"`
	AutoMatchReason     string                `gorm:"size:255" json:"auto_match_reason"`
}

// AccountReconcileModel corresponds to standard enterprise account.reconcile.model (Rules for auto-writeoffs, bank charges)
type AccountReconcileModel struct {
	models.ModelBase
	RuleName  string `gorm:"size:128;not null" json:"rule_name"`           // e.g. "Biaya Administrasi Bank", "Pajak Bunga"
	MatchType string `gorm:"size:32;not null" json:"match_type"`           // "contains", "regex", "exact", "prefix"
	MatchText string `gorm:"size:255;not null" json:"match_text"`          // Text to look for in statement line memo
	RuleType  string `gorm:"size:32;default:'write_off'" json:"rule_type"` // "write_off", "partner", "auto_reconcile"
	AccountID uint   `gorm:"not null" json:"account_id"`                   // Target Expense/Income Account
	Sequence  int    `gorm:"default:10" json:"sequence"`
}

// Request & Response DTOs

type CreateStatementRequest struct {
	Name           string  `json:"name"`
	JournalID      uint    `json:"journal_id" binding:"required"`
	Date           string  `json:"date"`
	BalanceStart   float64 `json:"balance_start"`
	BalanceEndReal float64 `json:"balance_end_real"`
	Note           string  `json:"note"`
}

type CreateStatementLineRequest struct {
	Date      string  `json:"date"`
	Name      string  `json:"name" binding:"required"`
	PartnerID *uint   `json:"partner_id"`
	Amount    float64 `json:"amount" binding:"required"`
	Ref       string  `json:"ref"`
}

type ImportStatementCSVRequest struct {
	JournalID      uint    `json:"journal_id" binding:"required"`
	StatementID    *uint   `json:"statement_id"` // Optional: append to existing, or create new
	Name           string  `json:"name"`
	Date           string  `json:"date"`
	Format         string  `json:"format"` // "bca", "mandiri", "bri", "generic"
	BalanceStart   float64 `json:"balance_start"`
	BalanceEndReal float64 `json:"balance_end_real"`
	CSVContent     string  `json:"csv_content" binding:"required"`
}

type ReconcileLineRequest struct {
	StatementLineID   uint    `json:"statement_line_id" binding:"required"`
	MoveID            *uint   `json:"move_id"`              // Invoice to reconcile
	MoveLineID        *uint   `json:"move_line_id"`         // Specific invoice line
	WriteOffAccountID *uint   `json:"write_off_account_id"` // E.g. Bank Fee GL account
	WriteOffAmount    float64 `json:"write_off_amount"`
	PartnerID         *uint   `json:"partner_id"`
	Notes             string  `json:"notes"`
}

type BatchReconcileRequest struct {
	StatementID uint                   `json:"statement_id" binding:"required"`
	Items       []ReconcileLineRequest `json:"items" binding:"required"`
}

type AutoMatchSuggestion struct {
	LineID             uint    `json:"line_id"`
	LineMemo           string  `json:"line_memo"`
	LineAmount         float64 `json:"line_amount"`
	LineDate           string  `json:"line_date"`
	SuggestedMoveID    *uint   `json:"suggested_move_id,omitempty"`
	SuggestedMoveName  string  `json:"suggested_move_name,omitempty"`
	SuggestedMoveType  string  `json:"suggested_move_type,omitempty"`
	SuggestedResidual  float64 `json:"suggested_residual,omitempty"`
	SuggestedPartnerID *uint   `json:"suggested_partner_id,omitempty"`
	SuggestedPartner   string  `json:"suggested_partner,omitempty"`
	SuggestedAccountID *uint   `json:"suggested_account_id,omitempty"`
	SuggestedAccount   string  `json:"suggested_account,omitempty"`
	MatchType          string  `json:"match_type"` // "exact_ref", "amount_partner", "bank_charge_rule", "fuzzy_amount"
	Confidence         float64 `json:"confidence"` // 0.0 - 1.0 (or 0-100)
	Reason             string  `json:"reason"`
}

type BankReconciliationStats struct {
	TotalStatements      int64   `json:"total_statements"`
	OpenStatements       int64   `json:"open_statements"`
	ConfirmedStatements  int64   `json:"confirmed_statements"`
	TotalLines           int64   `json:"total_lines"`
	ReconciledLines      int64   `json:"reconciled_lines"`
	UnreconciledLines    int64   `json:"unreconciled_lines"`
	ReconciledPercentage float64 `json:"reconciled_percentage"`
	TotalInflow          float64 `json:"total_inflow"`
	TotalOutflow         float64 `json:"total_outflow"`
	NetMovement          float64 `json:"net_movement"`
	DifferenceSum        float64 `json:"difference_sum"`
}

type AIBankMatchReport struct {
	StatementID           uint     `json:"statement_id"`
	StatementName         string   `json:"statement_name"`
	RiskAssessment        string   `json:"risk_assessment"` // "Low Risk", "Medium Risk", "High Risk"
	ExecutiveSummary      string   `json:"executive_summary"`
	TotalDiscrepancies    int      `json:"total_discrepancies"`
	DiscrepancyNotes      []string `json:"discrepancy_notes"`
	ActionRecommendations []string `json:"action_recommendations"`
	Engine                string   `json:"engine"`
}
