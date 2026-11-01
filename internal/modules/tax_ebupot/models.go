package tax_ebupot

import (
	"time"

	"erp-golang/internal/core/models"
	"erp-golang/internal/modules/base"
	"erp-golang/internal/modules/hr"
	"erp-golang/internal/modules/payroll"
)

// EbupotPeriod represents a monthly tax period for SPT Masa PPh 21/26
type EbupotPeriod struct {
	models.ModelBase
	PeriodCode       string     `gorm:"size:32;not null;uniqueIndex" json:"period_code"` // e.g. SPT21/2026/10/0
	TaxMonth         int        `gorm:"not null;index" json:"tax_month"`                 // 1 - 12
	TaxYear          int        `gorm:"not null;index" json:"tax_year"`                  // e.g. 2026
	RevisionNumber   int        `gorm:"default:0;index" json:"revision_number"`          // 0 = Normal, 1 = Pembetulan 1
	CompanyID        uint       `gorm:"default:1;index" json:"company_id"`
	TotalGrossWages  float64    `gorm:"default:0" json:"total_gross_wages"`
	TotalTaxDeducted float64    `gorm:"default:0" json:"total_tax_deducted"`
	TotalBupotCount  int        `gorm:"default:0" json:"total_bupot_count"`
	State            string     `gorm:"size:32;default:'draft';index" json:"state"` // draft, locked, reported, archived
	SubmissionNTPN   string     `gorm:"size:64" json:"submission_ntpn,omitempty"`   // Kode NTPN Pembayaran Pajak
	DJPReceiptNumber string     `gorm:"size:64" json:"djp_receipt_number,omitempty"`// BPE (Bukti Penerimaan Elektronik) DJP
	ReportedAt       *time.Time `json:"reported_at,omitempty"`
	Notes            string     `gorm:"type:text" json:"notes,omitempty"`

	// Relationships
	WithholdingSlips []WithholdingSlip `gorm:"foreignKey:PeriodID" json:"withholding_slips,omitempty"`
}

func (EbupotPeriod) TableName() string {
	return "tax_ebupot_periods"
}

// WithholdingSlip represents Bukti Pemotongan PPh 21 / PPh 26 (e-Bupot)
type WithholdingSlip struct {
	models.ModelBase
	PeriodID         uint              `gorm:"not null;index" json:"period_id"`
	SlipNumber       string            `gorm:"size:64;not null;uniqueIndex" json:"slip_number"` // e.g. BP21/2026/10/00001
	SlipType         string            `gorm:"size:32;not null;index" json:"slip_type"`         // bp_pegawai_tetap, bp_tidak_final, bp_final, bp_pasal_26, bp_a1_tahunan
	TaxObjectCode    string            `gorm:"size:32;not null;index" json:"tax_object_code"`   // e.g. 21-100-01 (Pegawai Tetap), 21-100-07 (Tenaga Ahli), 21-401-01 (Pesangon)
	TaxObjectName    string            `gorm:"size:128;not null" json:"tax_object_name"`

	// Taxpayer / Recipient Identity
	RecipientType    string            `gorm:"size:32;default:'employee'" json:"recipient_type"` // employee, vendor_expert, foreigner, other
	EmployeeID       *uint             `gorm:"index" json:"employee_id,omitempty"`
	Employee         *hr.Employee      `gorm:"foreignKey:EmployeeID" json:"employee,omitempty"`
	PartnerID        *uint             `gorm:"index" json:"partner_id,omitempty"`
	Partner          *base.Partner     `gorm:"foreignKey:PartnerID" json:"partner,omitempty"`
	PayslipID        *uint             `gorm:"index" json:"payslip_id,omitempty"`
	Payslip          *payroll.Payslip  `gorm:"foreignKey:PayslipID" json:"payslip,omitempty"`

	NIK              string            `gorm:"size:20;index" json:"nik"`               // 16-digit NIK
	NPWP             string            `gorm:"size:20;index" json:"npwp"`              // 15/16-digit NPWP
	RecipientName    string            `gorm:"size:128;not null" json:"recipient_name"`
	RecipientAddress string            `gorm:"size:255" json:"recipient_address"`
	PostalCode       string            `gorm:"size:10" json:"postal_code,omitempty"`
	Email            string            `gorm:"size:128" json:"email,omitempty"`

	// Tax PTKP & TER Scheme (PP 58/2023)
	PTKPStatus       string            `gorm:"size:16;default:'TK/0'" json:"ptkp_status"` // TK/0, TK/1, TK/2, TK/3, K/0, K/1, K/2, K/3
	TERCategory      string            `gorm:"size:16;default:'TER_A'" json:"ter_category"`// TER_A, TER_B, TER_C, PASAL_17, FINAL
	TaxCalculationMethod string        `gorm:"size:32;default:'ter_monthly'" json:"tax_calculation_method"` // ter_monthly, ter_daily, pph21_regular, pph26_foreign

	// Financials & Tax Amounts
	GrossIncome      float64           `gorm:"not null" json:"gross_income"`
	NonTaxableIncome float64           `gorm:"default:0" json:"non_taxable_income"` // Pengurang biaya jabatan/iuran
	TaxableBaseDPP   float64           `gorm:"not null" json:"taxable_base_dpp"`     // Dasar Pengenaan Pajak
	EffectiveRatePct float64           `gorm:"not null" json:"effective_rate_pct"`   // % TER or Pasal 17
	TaxWithheld      float64           `gorm:"not null" json:"tax_withheld"`         // PPh 21 Dipotong

	// Tax Facility
	TaxFacilityType  string            `gorm:"size:32;default:'none'" json:"tax_facility_type"` // none, dtp (Ditanggung Pemerintah), skb (Surat Keterangan Bebas), p3b (Tax Treaty DGT)
	TaxFacilityDocNo string            `gorm:"size:64" json:"tax_facility_doc_no,omitempty"`

	// Status & Verification
	Status           string            `gorm:"size:32;default:'verified';index" json:"status"` // draft, verified, submitted, cancelled
	IsNIKMatched     bool              `gorm:"default:true" json:"is_nik_matched"`             // Validasi 16 digit NIK/NPWP
	CompanyID        uint              `gorm:"default:1;index" json:"company_id"`
	Notes            string            `gorm:"type:text" json:"notes,omitempty"`
}

func (WithholdingSlip) TableName() string {
	return "tax_ebupot_withholding_slips"
}

// EbupotExportLog records each DJP file export event
type EbupotExportLog struct {
	models.ModelBase
	PeriodID      uint      `gorm:"not null;index" json:"period_id"`
	Format        string    `gorm:"size:16;not null" json:"format"` // csv, xml, json
	FileName      string    `gorm:"size:128;not null" json:"file_name"`
	RecordCount   int       `gorm:"not null" json:"record_count"`
	TotalTax      float64   `gorm:"not null" json:"total_tax"`
	ChecksumSHA256 string   `gorm:"size:64" json:"checksum_sha256"`
	ExportedBy    string    `gorm:"size:64;default:'Administrator'" json:"exported_by"`
	CompanyID     uint      `gorm:"default:1;index" json:"company_id"`
}

func (EbupotExportLog) TableName() string {
	return "tax_ebupot_export_logs"
}

// DTOs & Reports
type CreatePeriodRequest struct {
	TaxMonth       int    `json:"tax_month" binding:"required"`
	TaxYear        int    `json:"tax_year" binding:"required"`
	RevisionNumber int    `json:"revision_number"`
	Notes          string `json:"notes"`
}

type CreateWithholdingSlipRequest struct {
	PeriodID         uint    `json:"period_id" binding:"required"`
	SlipType         string  `json:"slip_type"`
	TaxObjectCode    string  `json:"tax_object_code"`
	TaxObjectName    string  `json:"tax_object_name"`
	RecipientType    string  `json:"recipient_type"`
	EmployeeID       *uint   `json:"employee_id"`
	PartnerID        *uint   `json:"partner_id"`
	PayslipID        *uint   `json:"payslip_id"`
	NIK              string  `json:"nik"`
	NPWP             string  `json:"npwp"`
	RecipientName    string  `json:"recipient_name" binding:"required"`
	RecipientAddress string  `json:"recipient_address"`
	PTKPStatus       string  `json:"ptkp_status"`
	GrossIncome      float64 `json:"gross_income" binding:"required"`
	TaxFacilityType  string  `json:"tax_facility_type"`
	TaxFacilityDocNo string  `json:"tax_facility_doc_no"`
	Notes            string  `json:"notes"`
}

type ImportFromPayrollRequest struct {
	PeriodID     uint  `json:"period_id" binding:"required"`
	PayslipRunID *uint `json:"payslip_run_id"` // Optional specific batch
}

type EbupotStats struct {
	TotalPeriods          int     `json:"total_periods"`
	ActivePeriodCode      string  `json:"active_period_code"`
	TotalWithholdingSlips int     `json:"total_withholding_slips"`
	TotalGrossWages       float64 `json:"total_gross_wages"`
	TotalTaxWithheld      float64 `json:"total_tax_withheld"`
	ValidNIKCount         int     `json:"valid_nik_count"`
	PendingReconciliation int     `json:"pending_reconciliation"`
}

type TaxReconciliationReport struct {
	PeriodCode            string  `json:"period_code"`
	TaxMonth              int     `json:"tax_month"`
	TaxYear               int     `json:"tax_year"`
	TotalPayrollGross     float64 `json:"total_payroll_gross"`
	TotalEbupotGross      float64 `json:"total_ebupot_gross"`
	GrossVariance         float64 `json:"gross_variance"`
	TotalPayrollTaxDeducted float64 `json:"total_payroll_tax_deducted"`
	TotalEbupotTaxWithheld  float64 `json:"total_ebupot_tax_withheld"`
	TaxVariance           float64 `json:"tax_variance"`
	GLTaxPayableBalance   float64 `json:"gl_tax_payable_balance"`
	IsReconciled          bool    `json:"is_reconciled"`
	StatusSummary         string  `json:"status_summary"`
	UnmatchedEmployees    []string `json:"unmatched_employees,omitempty"`
}

type AITaxComplianceReport struct {
	ComplianceStatus        string   `json:"compliance_status"` // COMPLIANT, WARNING, AUDIT_REQUIRED
	ComplianceScore         int      `json:"compliance_score"`  // 0 - 100
	PeriodCode              string   `json:"period_code"`
	TotalSlipsAnalyzed      int      `json:"total_slips_analyzed"`
	Matched16DigitNIKCount  int      `json:"matched_16digit_nik_count"`
	TERVerificationStatus   string   `json:"ter_verification_status"`
	EstimatedPenaltyRisk    string   `json:"estimated_penalty_risk"`
	ExecutiveSummary        string   `json:"executive_summary"`
	ComplianceChecklist     []string `json:"compliance_checklist"`
	ActionableAdvice        []string `json:"actionable_advice"`
	AIProvider              string   `json:"ai_provider"`
}
