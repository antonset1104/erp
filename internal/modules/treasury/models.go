package treasury

import (
	"time"

	"erp-golang/internal/core/models"
)

// ============================================================================
// 1. BANK ACCOUNT & LIQUIDITY POOL
// ============================================================================

type BankAccountType string

const (
	BankTypeChecking  BankAccountType = "checking"     // Giro / Operasional
	BankTypeSavings   BankAccountType = "savings"      // Tabungan
	BankTypeDeposit   BankAccountType = "time_deposit" // Deposito Berjangka
	BankTypePettyCash BankAccountType = "petty_cash"   // Kas Kecil
	BankTypeEscrow    BankAccountType = "escrow"       // Rekening Penampung
)

type TreasuryBankAccount struct {
	models.ModelBase
	AccountCode     string          `gorm:"size:32;not null;uniqueIndex" json:"account_code"` // e.g. BA/2026/001
	BankName        string          `gorm:"size:64;not null;index" json:"bank_name"`          // e.g. Bank Central Asia (BCA)
	AccountNumber   string          `gorm:"size:64;not null" json:"account_number"`
	AccountHolder   string          `gorm:"size:128;not null" json:"account_holder"`
	AccountType     BankAccountType `gorm:"size:32;default:'checking'" json:"account_type"`
	Currency        string          `gorm:"size:8;default:'IDR'" json:"currency"`
	CurrentBalance  float64         `gorm:"default:0" json:"current_balance"`
	MinimumBalance  float64         `gorm:"default:0" json:"minimum_balance"`       // Safety buffer
	OverdraftLimit  float64         `gorm:"default:0" json:"overdraft_limit"`       // Batas cerukan kredit
	InterestRatePct float64         `gorm:"default:0" json:"interest_rate_pct"`     // Bunga p.a.
	JournalID       uint            `gorm:"index" json:"journal_id"`                // Linked account.journal ID
	GLAccountID     uint            `gorm:"index" json:"gl_account_id"`             // Linked Kas/Bank asset account ID
	Status          string          `gorm:"size:32;default:'active'" json:"status"` // active, frozen, closed
	IsPrimaryPool   bool            `gorm:"default:false" json:"is_primary_pool"`   // Rekening utama kas holding
	Notes           string          `gorm:"type:text" json:"notes"`
}

// ============================================================================
// 2. CASH FLOW FORECAST (PROYEKSI ARUS KAS MULTI-HORIZON)
// ============================================================================

type DeficitRiskLevel string

const (
	RiskNone     DeficitRiskLevel = "none"     // Surplus nyaman
	RiskModerate DeficitRiskLevel = "moderate" // Kas di bawah safety buffer
	RiskCritical DeficitRiskLevel = "critical" // Potensi defisit likuiditas negatif
)

type CashFlowForecast struct {
	models.ModelBase
	ForecastCode        string                 `gorm:"size:64;not null;uniqueIndex" json:"forecast_code"` // e.g. CFF/2026/00001
	Name                string                 `gorm:"size:128;not null" json:"name"`
	BaseDate            time.Time              `gorm:"not null" json:"base_date"`
	HorizonDays         int                    `gorm:"default:30" json:"horizon_days"` // 30, 60, 90, 180, 365
	OpeningCashBalance  float64                `gorm:"default:0" json:"opening_cash_balance"`
	ProjectedInflow     float64                `gorm:"default:0" json:"projected_inflow"`      // Total tertimbang kas masuk
	ProjectedOutflow    float64                `gorm:"default:0" json:"projected_outflow"`     // Total tertimbang kas keluar
	ProjectedNetCash    float64                `gorm:"default:0" json:"projected_net_cash"`    // Inflow - Outflow
	ProjectedEndingCash float64                `gorm:"default:0" json:"projected_ending_cash"` // Opening + NetCash
	MinimumSafetyBuffer float64                `gorm:"default:0" json:"minimum_safety_buffer"`
	LowestPointCash     float64                `gorm:"default:0" json:"lowest_point_cash"` // Saldo kas terendah selama periode
	DeficitRisk         DeficitRiskLevel       `gorm:"size:32;default:'none'" json:"deficit_risk"`
	EstimatedRunwayDays int                    `gorm:"default:0" json:"estimated_runway_days"` // Berapa hari kas mencukupi
	Status              string                 `gorm:"size:32;default:'active'" json:"status"` // draft, active, archived
	Notes               string                 `gorm:"type:text" json:"notes"`
	Items               []CashFlowForecastItem `gorm:"foreignKey:ForecastID" json:"items,omitempty"`
}

// Item rincian pergerakan kas proyeksi
type CashFlowCategory string

const (
	CategoryARReceivables CashFlowCategory = "customer_ar"    // Piutang dagang jatuh tempo
	CategoryRecurringMRR  CashFlowCategory = "recurring_mrr"  // Langganan berulang SaaS
	CategorySalesOrder    CashFlowCategory = "sales_pipeline" // SO confirmed belum terbit invoice
	CategoryOtherInflow   CashFlowCategory = "other_inflow"   // Pendapatan bunga, restitusi pajak

	CategoryAPPayables      CashFlowCategory = "vendor_ap"        // Hutang vendor jatuh tempo
	CategoryPayrollSalaries CashFlowCategory = "payroll_salaries" // Gaji bulanan & BPJS
	CategoryTaxObligations  CashFlowCategory = "tax_obligations"  // SPT Masa PPN/PPh
	CategoryCapexPurchase   CashFlowCategory = "capex_po"         // PO belanja modal confirmed
	CategoryLoanRepayment   CashFlowCategory = "debt_service"     // Pokok + bunga pinjaman
	CategoryOperationalOpex CashFlowCategory = "operational_opex" // Sewa, utilitas, biaya kantor
)

type CashFlowDirection string

const (
	DirectionInflow  CashFlowDirection = "inflow"
	DirectionOutflow CashFlowDirection = "outflow"
)

type CashFlowForecastItem struct {
	models.ModelBase
	ForecastID      uint              `gorm:"not null;index" json:"forecast_id"`
	Category        CashFlowCategory  `gorm:"size:64;not null;index" json:"category"`
	Direction       CashFlowDirection `gorm:"size:16;not null;index" json:"direction"`
	ExpectedDate    time.Time         `gorm:"not null;index" json:"expected_date"`
	GrossAmount     float64           `gorm:"default:0" json:"gross_amount"`
	CertaintyFactor float64           `gorm:"default:1.0" json:"certainty_factor"` // 0.0 s.d 1.0 (probabilitas realisasi)
	AdjustedAmount  float64           `gorm:"default:0" json:"adjusted_amount"`    // GrossAmount * CertaintyFactor
	SourceDocType   string            `gorm:"size:64" json:"source_doc_type"`      // account.move, sale.order, purchase.order, hr.payslip
	SourceDocID     uint              `gorm:"default:0" json:"source_doc_id"`
	SourceDocRef    string            `gorm:"size:128" json:"source_doc_ref"`
	PartnerName     string            `gorm:"size:128" json:"partner_name"`
	Notes           string            `gorm:"type:text" json:"notes"`
}

// ============================================================================
// 3. INTER-BANK FUND SWEEP & TRANSFER (SWEEP LIQUIDITY ENGINE)
// ============================================================================

type InterBankTransfer struct {
	models.ModelBase
	TransferCode    string    `gorm:"size:64;not null;uniqueIndex" json:"transfer_code"` // e.g. IBT/2026/00001
	FromAccountID   uint      `gorm:"not null;index" json:"from_account_id"`
	FromAccountName string    `gorm:"size:128" json:"from_account_name"`
	ToAccountID     uint      `gorm:"not null;index" json:"to_account_id"`
	ToAccountName   string    `gorm:"size:128" json:"to_account_name"`
	TransferDate    time.Time `gorm:"not null" json:"transfer_date"`
	Amount          float64   `gorm:"not null" json:"amount"`
	TransferFee     float64   `gorm:"default:0" json:"transfer_fee"`
	Reference       string    `gorm:"size:128" json:"reference"`
	State           string    `gorm:"size:32;default:'draft';index" json:"state"` // draft, posted, cancelled
	JournalMoveID   uint      `gorm:"default:0" json:"journal_move_id"`           // ID account.move hasil auto posting GL
	JournalMoveName string    `gorm:"size:64" json:"journal_move_name"`
	ExecutedByUID   uint      `json:"executed_by_uid"`
	Memo            string    `gorm:"type:text" json:"memo"`
}

// ============================================================================
// 4. DTOs & ANALYTICS REQUEST/RESPONSE
// ============================================================================

type CreateBankAccountRequest struct {
	BankName        string          `json:"bank_name" binding:"required"`
	AccountNumber   string          `json:"account_number" binding:"required"`
	AccountHolder   string          `json:"account_holder" binding:"required"`
	AccountType     BankAccountType `json:"account_type"`
	Currency        string          `json:"currency"`
	CurrentBalance  float64         `json:"current_balance"`
	MinimumBalance  float64         `json:"minimum_balance"`
	OverdraftLimit  float64         `json:"overdraft_limit"`
	InterestRatePct float64         `json:"interest_rate_pct"`
	JournalID       uint            `json:"journal_id"`
	GLAccountID     uint            `json:"gl_account_id"`
	IsPrimaryPool   bool            `json:"is_primary_pool"`
	Notes           string          `json:"notes"`
}

type GenerateForecastRequest struct {
	Name                string    `json:"name" binding:"required"`
	HorizonDays         int       `json:"horizon_days"` // 30, 60, 90, 180, 365
	BaseDate            time.Time `json:"base_date"`
	MinimumSafetyBuffer float64   `json:"minimum_safety_buffer"`
	IncludePipelineSO   bool      `json:"include_pipeline_so"`
	IncludeApprovedPO   bool      `json:"include_approved_po"`
	IncludeMRRRenewal   bool      `json:"include_mrr_renewal"`
	Notes               string    `json:"notes"`
}

type CreateInterBankTransferRequest struct {
	FromAccountID uint      `json:"from_account_id" binding:"required"`
	ToAccountID   uint      `json:"to_account_id" binding:"required"`
	TransferDate  time.Time `json:"transfer_date"`
	Amount        float64   `json:"amount" binding:"required,gt=0"`
	TransferFee   float64   `json:"transfer_fee"`
	Reference     string    `json:"reference"`
	Memo          string    `json:"memo"`
}

type TreasuryStats struct {
	TotalLiquidCash       float64          `json:"total_liquid_cash"`
	TotalActiveAccounts   int64            `json:"total_active_accounts"`
	Projected30DInflow    float64          `json:"projected_30d_inflow"`
	Projected30DOutflow   float64          `json:"projected_30d_outflow"`
	Projected30DNetCash   float64          `json:"projected_30d_net_cash"`
	RunwayDays            int              `json:"runway_days"`
	PrimaryLiquidityRisk  DeficitRiskLevel `json:"primary_liquidity_risk"`
	TotalTransfersVolume  float64          `json:"total_transfers_volume"`
	PendingTransfersCount int64            `json:"pending_transfers_count"`
	SafetyBufferCoverage  float64          `json:"safety_buffer_coverage_pct"` // % kas terhadap minimum safety buffer
}

type WorkingCapitalMetric struct {
	MetricName  string  `json:"metric_name"`
	CurrentVal  float64 `json:"current_val"`
	Benchmark   float64 `json:"benchmark"`
	Status      string  `json:"status"` // OPTIMAL, WARNING, CRITICAL
	Description string  `json:"description"`
}

type AITreasuryAdvisoryReport struct {
	ForecastCode          string                 `json:"forecast_code"`
	LiquidityHealth       string                 `json:"liquidity_health"` // HEALTHY, CAUTION, DISTRESSED
	HealthScore           int                    `json:"health_score"`     // 0 - 100
	ExecutiveSummary      string                 `json:"executive_summary"`
	EstimatedRunwayMonths float64                `json:"estimated_runway_months"`
	LowestCashProjection  float64                `json:"lowest_cash_projection"`
	ProjectedDeficitDate  string                 `json:"projected_deficit_date,omitempty"`
	WorkingCapitalMetrics []WorkingCapitalMetric `json:"working_capital_metrics"`
	ActionableAdvice      []string               `json:"actionable_advice"`
	AIProvider            string                 `json:"ai_provider"`
}
