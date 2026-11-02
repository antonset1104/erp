package account

import (
	"erp-golang/internal/core/models"
	"erp-golang/internal/modules/base"
)

// AccountAccount corresponds to standard enterprise account.account (Chart of Accounts)
type AccountAccount struct {
	models.ModelBase
	Code    string `gorm:"size:64;not null;uniqueIndex" json:"code"` // e.g. "1200", "2100", "4100"
	Name    string `gorm:"size:128;not null" json:"name"`
	Type    string `gorm:"size:64;not null" json:"type"` // "asset_receivable", "liability_payable", "income", "expense", "asset_cash"
}

// AccountJournal corresponds to standard enterprise account.journal
type AccountJournal struct {
	models.ModelBase
	Name             string `gorm:"size:128;not null" json:"name"`
	Code             string `gorm:"size:16;not null" json:"code"` // INV, BILL, BNK, CSH, MISC
	Type             string `gorm:"size:32;not null" json:"type"` // "sale", "purchase", "cash", "bank", "general"
	DefaultAccountID uint   `json:"default_account_id"`
}

// AccountMove corresponds to standard enterprise account.move (Customer Invoices, Vendor Bills, Journal Entries)
type AccountMove struct {
	models.ModelBase
	Name           string            `gorm:"size:64;not null;uniqueIndex" json:"name"` // e.g. INV/2026/00001
	MoveType       string            `gorm:"size:32;not null;index" json:"move_type"`  // "out_invoice", "in_invoice", "entry"
	JournalID      uint              `gorm:"not null" json:"journal_id"`
	Journal        AccountJournal    `gorm:"foreignKey:JournalID" json:"journal,omitempty"`
	PartnerID      uint              `gorm:"not null;index" json:"partner_id"`
	Partner        base.Partner      `gorm:"foreignKey:PartnerID" json:"partner,omitempty"`
	InvoiceDate    string            `gorm:"size:32;not null" json:"invoice_date"`
	InvoiceDueDate string            `gorm:"size:32" json:"invoice_due_date"`
	State          string            `gorm:"size:32;default:'draft';index" json:"state"` // "draft", "posted", "cancel"
	PaymentState   string            `gorm:"size:32;default:'not_paid'" json:"payment_state"` // "not_paid", "paid"
	AmountUntaxed  float64           `gorm:"default:0" json:"amount_untaxed"`
	AmountTax      float64           `gorm:"default:0" json:"amount_tax"`
	AmountTotal    float64           `gorm:"default:0" json:"amount_total"`
	AmountResidual float64           `gorm:"default:0" json:"amount_residual"` // Remaining amount to pay
	LineIDs        []AccountMoveLine `gorm:"foreignKey:MoveID" json:"line_ids"`
	Ref            string            `gorm:"size:128" json:"ref"` // Reference / SO / PO number
	Note           string            `gorm:"type:text" json:"note"`
	CurrencyID     *uint             `gorm:"index" json:"currency_id,omitempty"`
	CurrencyRate   float64           `gorm:"default:1" json:"currency_rate"`   // Direct spot rate at invoice creation (e.g. 16000 IDR/USD)
	AmountCurrency float64           `gorm:"default:0" json:"amount_currency"` // Amount in foreign currency
}

// AccountMoveLine corresponds to standard enterprise account.move.line (Balanced Debit and Credit entries)
type AccountMoveLine struct {
	models.ModelBase
	MoveID         uint           `gorm:"index" json:"move_id"`
	AccountID      uint           `gorm:"not null" json:"account_id"`
	Account        AccountAccount `gorm:"foreignKey:AccountID" json:"account,omitempty"`
	PartnerID      *uint          `json:"partner_id,omitempty"`
	Name           string         `gorm:"size:255;not null" json:"name"` // Label
	Debit          float64        `gorm:"default:0" json:"debit"`
	Credit         float64        `gorm:"default:0" json:"credit"`
	Balance        float64        `gorm:"default:0" json:"balance"` // debit - credit
	ProductID      *uint          `json:"product_id,omitempty"`
	Quantity       float64        `gorm:"default:0" json:"quantity"`
	PriceUnit      float64        `gorm:"default:0" json:"price_unit"`
	CurrencyID     *uint          `gorm:"index" json:"currency_id,omitempty"`
	AmountCurrency float64        `gorm:"default:0" json:"amount_currency"` // Foreign currency balance
}

// AccountPayment corresponds to standard enterprise account.payment (Payment Registration)
type AccountPayment struct {
	models.ModelBase
	Name        string       `gorm:"size:64;not null;uniqueIndex" json:"name"` // e.g. PAY/2026/00001
	MoveID      uint         `gorm:"index;not null" json:"move_id"`           // Invoice being paid
	PartnerID   uint         `gorm:"not null" json:"partner_id"`
	Partner     base.Partner `gorm:"foreignKey:PartnerID" json:"partner,omitempty"`
	JournalID   uint         `gorm:"not null" json:"journal_id"`              // Bank or Cash Journal
	Amount      float64      `gorm:"not null" json:"amount"`
	PaymentDate string       `gorm:"size:32;not null" json:"payment_date"`
	PaymentType string       `gorm:"size:32;not null" json:"payment_type"`    // "inbound" (customer), "outbound" (supplier)
	Memo        string       `gorm:"size:255" json:"memo"`
}
