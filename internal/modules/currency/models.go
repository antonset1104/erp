package currency

import (
	"erp-golang/internal/core/models"
	"erp-golang/internal/modules/account"
	"erp-golang/internal/modules/base"
)

// Currency represents standard enterprise res.currency
type Currency struct {
	models.ModelBase
	Name          string         `gorm:"size:16;not null;uniqueIndex" json:"name"` // e.g. "USD", "EUR", "IDR"
	Symbol        string         `gorm:"size:8;not null" json:"symbol"`            // e.g. "$", "€", "Rp"
	FullName      string         `gorm:"size:64;not null" json:"full_name"`        // e.g. "US Dollar", "Indonesian Rupiah"
	DecimalPlaces int            `gorm:"default:2" json:"decimal_places"`
	Rounding      float64        `gorm:"default:0.01" json:"rounding"`
	Active        bool           `gorm:"default:true;index" json:"active"`
	IsBase         bool           `gorm:"default:false;index" json:"is_base"` // True for IDR (Company base currency)
	Rates          []CurrencyRate `gorm:"foreignKey:CurrencyID" json:"rates,omitempty"`
	LatestSpotRate float64        `gorm:"-" json:"latest_spot_rate"`
	LatestRateDate string         `gorm:"-" json:"latest_rate_date"`
}

func (Currency) TableName() string {
	return "res_currencies"
}

// CurrencyRate represents standard enterprise res.currency.rate
type CurrencyRate struct {
	models.ModelBase
	CurrencyID uint      `gorm:"not null;index" json:"currency_id"`
	Currency   *Currency `gorm:"foreignKey:CurrencyID" json:"currency,omitempty"`
	Name       string    `gorm:"size:32;not null;index" json:"name"` // Date "YYYY-MM-DD"
	Rate       float64   `gorm:"not null" json:"rate"`               // Standard Enterprise ERP rate (Foreign per 1 Base, e.g. 1/16000)
	DirectRate float64   `gorm:"not null" json:"direct_rate"`        // Indonesian market rate (Base per 1 Foreign, e.g. 16,000 IDR/USD)
	Source     string    `gorm:"size:64;default:'Bank Indonesia JISDOR'" json:"source"`
	CompanyID  uint      `gorm:"default:1;index" json:"company_id"`
}

func (CurrencyRate) TableName() string {
	return "res_currency_rates"
}

// ForexRevaluation represents standard enterprise account.multicurrency.revaluation
type ForexRevaluation struct {
	models.ModelBase
	Name            string                 `gorm:"size:64;not null;uniqueIndex" json:"name"` // e.g. "REV/2026/10/00001"
	RevaluationDate string                 `gorm:"size:32;not null;index" json:"revaluation_date"`
	CurrencyID      uint                   `gorm:"not null;index" json:"currency_id"`
	Currency        *Currency              `gorm:"foreignKey:CurrencyID" json:"currency,omitempty"`
	ClosingRate     float64                `gorm:"not null" json:"closing_rate"`             // Spot rate at closing date (e.g. 16,300 IDR/USD)
	State           string                 `gorm:"size:32;default:'draft';index" json:"state"` // "draft", "posted", "cancel"
	TotalGain       float64                `gorm:"default:0" json:"total_gain"`
	TotalLoss       float64                `gorm:"default:0" json:"total_loss"`
	NetGainLoss     float64                `gorm:"default:0" json:"net_gain_loss"` // TotalGain - TotalLoss
	JournalEntryID  *uint                  `gorm:"index" json:"journal_entry_id,omitempty"`
	JournalEntry    *account.AccountMove   `gorm:"foreignKey:JournalEntryID" json:"journal_entry,omitempty"`
	Notes           string                 `gorm:"type:text" json:"notes"`
	Lines           []ForexRevaluationLine `gorm:"foreignKey:RevaluationID" json:"lines,omitempty"`
	CompanyID       uint                   `gorm:"default:1;index" json:"company_id"`
	UserID          uint                   `gorm:"default:1" json:"user_id"`
}

func (ForexRevaluation) TableName() string {
	return "account_multicurrency_revaluations"
}

// ForexRevaluationLine represents each open receivable/payable being revalued
type ForexRevaluationLine struct {
	models.ModelBase
	RevaluationID    uint          `gorm:"not null;index" json:"revaluation_id"`
	MoveID           uint          `gorm:"not null;index" json:"move_id"`
	MoveName         string        `gorm:"size:64;not null" json:"move_name"`
	MoveType         string        `gorm:"size:32;not null" json:"move_type"` // "out_invoice", "in_invoice"
	MoveLineID       uint          `gorm:"not null;index" json:"move_line_id"`
	PartnerID        uint          `gorm:"not null;index" json:"partner_id"`
	PartnerName      string        `gorm:"size:128" json:"partner_name"`
	Partner          *base.Partner `gorm:"foreignKey:PartnerID" json:"partner,omitempty"`
	AccountID        uint          `gorm:"not null" json:"account_id"`
	AccountCode      string        `gorm:"size:32" json:"account_code"`
	AccountName      string        `gorm:"size:128" json:"account_name"`
	AccountType      string        `gorm:"size:64;not null" json:"account_type"` // "asset_receivable", "liability_payable"
	AmountCurrency   float64       `gorm:"not null" json:"amount_currency"`      // Open amount in foreign currency (e.g. $1,000)
	HistoricalRate   float64       `gorm:"not null" json:"historical_rate"`      // Rate at invoice date (e.g. 16,000)
	HistoricalAmount float64       `gorm:"not null" json:"historical_amount"`    // IDR historical book balance (e.g. 16,000,000)
	ClosingRate      float64       `gorm:"not null" json:"closing_rate"`         // Balance sheet revaluation rate (e.g. 16,300)
	RevaluedAmount   float64       `gorm:"not null" json:"revalued_amount"`      // IDR revalued balance (e.g. 16,300,000)
	GainLoss         float64       `gorm:"not null" json:"gain_loss"`            // Difference in IDR
	GainLossType     string        `gorm:"size:16;not null" json:"gain_loss_type"` // "gain", "loss"
}

func (ForexRevaluationLine) TableName() string {
	return "account_multicurrency_revaluation_lines"
}

// Request and Response DTOs

type CreateCurrencyRequest struct {
	Name          string  `json:"name" binding:"required"`
	Symbol        string  `json:"symbol" binding:"required"`
	FullName      string  `json:"full_name" binding:"required"`
	DecimalPlaces int     `json:"decimal_places"`
	Rounding      float64 `json:"rounding"`
	IsBase        bool    `json:"is_base"`
}

type CreateRateRequest struct {
	CurrencyID uint    `json:"currency_id" binding:"required"`
	Date       string  `json:"date" binding:"required"` // YYYY-MM-DD
	DirectRate float64 `json:"direct_rate" binding:"required,gt=0"`
	Source     string  `json:"source"`
}

type CurrencyConversionRequest struct {
	Amount         float64 `json:"amount" binding:"required"`
	FromCurrencyID uint    `json:"from_currency_id" binding:"required"`
	ToCurrencyID   uint    `json:"to_currency_id" binding:"required"`
	Date           string  `json:"date"`
}

type CurrencyConversionResponse struct {
	OriginalAmount float64 `json:"original_amount"`
	FromCurrency   string  `json:"from_currency"`
	ToCurrency     string  `json:"to_currency"`
	ConvertedAmount float64 `json:"converted_amount"`
	AppliedRate    float64 `json:"applied_rate"`
	RateDate       string  `json:"rate_date"`
}

type ComputeRevaluationRequest struct {
	CurrencyID      uint    `json:"currency_id" binding:"required"`
	RevaluationDate string  `json:"revaluation_date" binding:"required"` // YYYY-MM-DD
	ClosingRate     float64 `json:"closing_rate" binding:"required,gt=0"`
	Notes           string  `json:"notes"`
}

type RealizedGainLossCalculationRequest struct {
	MoveID        uint    `json:"move_id" binding:"required"`
	PaymentDate   string  `json:"payment_date" binding:"required"`
	PaymentRate   float64 `json:"payment_rate" binding:"required,gt=0"`
	PaymentAmount float64 `json:"payment_amount" binding:"required,gt=0"`
}

type RealizedGainLossResult struct {
	MoveID             uint    `json:"move_id"`
	MoveName           string  `json:"move_name"`
	CurrencyName       string  `json:"currency_name"`
	ForeignAmountPaid  float64 `json:"foreign_amount_paid"`
	HistoricalRate     float64 `json:"historical_rate"`
	PaymentRate        float64 `json:"payment_rate"`
	HistoricalIDR      float64 `json:"historical_idr"`
	ActualPaidIDR      float64 `json:"actual_paid_idr"`
	RealizedGainLoss   float64 `json:"realized_gain_loss"`
	GainLossType       string  `json:"gain_loss_type"` // "gain", "loss"
	JournalAccountCode string  `json:"journal_account_code"`
	JournalAccountName string  `json:"journal_account_name"`
}

type CurrencyStats struct {
	TotalCurrencies       int64   `json:"total_currencies"`
	TotalRatesRecorded    int64   `json:"total_rates_recorded"`
	OpenUSDReceivables    float64 `json:"open_usd_receivables"`
	OpenUSDPayables       float64 `json:"open_usd_payables"`
	NetUSDExposure        float64 `json:"net_usd_exposure"`
	TotalRevaluationsDone int64   `json:"total_revaluations_done"`
	YTDUnrealizedGainLoss float64 `json:"ytd_unrealized_gain_loss"`
	LatestUSDRate         float64 `json:"latest_usd_rate"`
	LatestEURRate         float64 `json:"latest_eur_rate"`
}

type ForexRiskReport struct {
	BaseCurrency          string   `json:"base_currency"`
	ForeignCurrency       string   `json:"foreign_currency"`
	SpotRate              float64  `json:"spot_rate"`
	OpenReceivablesFC     float64  `json:"open_receivables_fc"`
	OpenPayablesFC        float64  `json:"open_payables_fc"`
	NetExposureFC         float64  `json:"net_exposure_fc"`
	NetExposureIDR        float64  `json:"net_exposure_idr"`
	SensitivityShock2Pct  float64  `json:"sensitivity_shock_2pct"`
	SensitivityShock5Pct  float64  `json:"sensitivity_shock_5pct"`
	SensitivityShock10Pct float64  `json:"sensitivity_shock_10pct"`
	RiskLevel             string   `json:"risk_level"` // "low", "medium", "high"
	ExecutiveSummary      string   `json:"executive_summary"`
	ActionRecommendations []string `json:"action_recommendations"`
	Engine                string   `json:"engine"`
}
