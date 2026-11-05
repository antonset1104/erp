package tax_efaktur

import (
	"erp-golang/internal/core/models"
	"erp-golang/internal/modules/account"
	"erp-golang/internal/modules/base"
)

// TaxNSFP represents DJP's Nomor Seri Faktur Pajak allocation range
type TaxNSFP struct {
	models.ModelBase
	Prefix         string `gorm:"size:16;not null;default:'010'" json:"prefix"` // e.g. "010" (Penyerahan BKP Normal), "020" (WAPU)
	Year           int    `gorm:"not null;index" json:"year"`                   // e.g. 2026
	StartNumber    string `gorm:"size:16;not null" json:"start_number"`         // e.g. "00000001"
	EndNumber      string `gorm:"size:16;not null" json:"end_number"`           // e.g. "00000500"
	CurrentNumber  string `gorm:"size:16;not null" json:"current_number"`       // Current sequence running
	TotalQuota     int    `gorm:"not null" json:"total_quota"`
	UsedQuota      int    `gorm:"default:0" json:"used_quota"`
	RemainingQuota int    `gorm:"not null" json:"remaining_quota"`
	State          string `gorm:"size:32;default:'active';index" json:"state"` // "active", "depleted", "expired"
	CompanyID      uint   `gorm:"default:1;index" json:"company_id"`
}

func (TaxNSFP) TableName() string {
	return "account_tax_nsfps"
}

// TaxInvoice represents e-Faktur Pajak Elektronik DJP
type TaxInvoice struct {
	models.ModelBase
	TaxInvoiceType string               `gorm:"size:16;not null;index" json:"tax_invoice_type"`  // "out_tax" (Keluaran/Sales) or "in_tax" (Masukan/Purchase)
	NSFPNumber     string               `gorm:"size:32;not null;uniqueIndex" json:"nsfp_number"` // Full 16-digit e.g. "010.001-26.00000001"
	InvoiceDate    string               `gorm:"size:32;not null;index" json:"invoice_date"`      // YYYY-MM-DD
	TaxPeriod      int                  `gorm:"not null;index" json:"tax_period"`                // 1 - 12 (Month)
	TaxYear        int                  `gorm:"not null;index" json:"tax_year"`                  // YYYY
	PartnerID      uint                 `gorm:"not null;index" json:"partner_id"`
	Partner        *base.Partner        `gorm:"foreignKey:PartnerID" json:"partner,omitempty"`
	PartnerName    string               `gorm:"size:128" json:"partner_name"`
	NPWP           string               `gorm:"size:32;not null;index" json:"npwp"` // 15 or 16 digit NIK/NPWP
	Address        string               `gorm:"size:255" json:"address"`
	MoveID         *uint                `gorm:"index" json:"move_id,omitempty"` // Link to Commercial Invoice (account.move)
	Move           *account.AccountMove `gorm:"foreignKey:MoveID" json:"move,omitempty"`
	DPP            float64              `gorm:"not null" json:"dpp"`                    // Dasar Pengenaan Pajak
	TaxRatePPN     float64              `gorm:"default:11.0" json:"tax_rate_ppn"`       // PPN 11% or 12%
	PPN            float64              `gorm:"not null" json:"ppn"`                    // Pajak Pertambahan Nilai
	PPNBM          float64              `gorm:"default:0" json:"ppnbm"`                 // Pajak Penjualan atas Barang Mewah
	PPhType        string               `gorm:"size:32;default:'none'" json:"pph_type"` // "none", "pph_23", "pph_4_2", "pph_22"
	PPhRate        float64              `gorm:"default:0" json:"pph_rate"`
	PPhAmount      float64              `gorm:"default:0" json:"pph_amount"`
	TotalAmount    float64              `gorm:"not null" json:"total_amount"`               // DPP + PPN + PPNBM - PPhAmount
	State          string               `gorm:"size:32;default:'draft';index" json:"state"` // "draft", "generated", "reported", "cancelled"
	IsReplacement  bool                 `gorm:"default:false" json:"is_replacement"`        // Faktur Pajak Pengganti (FG_PENGGANTI=1)
	OriginalNSFP   string               `gorm:"size:32" json:"original_nsfp,omitempty"`
	QRURL          string               `gorm:"size:255" json:"qr_url,omitempty"` // DJP Verification QR URL
	Notes          string               `gorm:"type:text" json:"notes,omitempty"`
	Lines          []TaxInvoiceLine     `gorm:"foreignKey:TaxInvoiceID" json:"lines,omitempty"`
	CompanyID      uint                 `gorm:"default:1;index" json:"company_id"`
}

func (TaxInvoice) TableName() string {
	return "account_tax_invoices"
}

// TaxInvoiceLine represents Detail Item Barang / Jasa Kena Pajak (BKP/JKP)
type TaxInvoiceLine struct {
	models.ModelBase
	TaxInvoiceID uint    `gorm:"not null;index" json:"tax_invoice_id"`
	ProductID    *uint   `gorm:"index" json:"product_id,omitempty"`
	ProductCode  string  `gorm:"size:64" json:"product_code"`
	ProductName  string  `gorm:"size:128;not null" json:"product_name"`
	PriceUnit    float64 `gorm:"not null" json:"price_unit"`
	Quantity     float64 `gorm:"not null;default:1" json:"quantity"`
	TotalPrice   float64 `gorm:"not null" json:"total_price"`
	Discount     float64 `gorm:"default:0" json:"discount"`
	DPP          float64 `gorm:"not null" json:"dpp"`
	PPN          float64 `gorm:"not null" json:"ppn"`
	PPNBM        float64 `gorm:"default:0" json:"ppnbm"`
}

func (TaxInvoiceLine) TableName() string {
	return "account_tax_invoice_lines"
}

// --- DTOs ---

type CreateNSFPRequest struct {
	Prefix      string `json:"prefix"`       // Default "010"
	Year        int    `json:"year"`         // Default 2026
	StartNumber string `json:"start_number"` // "00000001"
	EndNumber   string `json:"end_number"`   // "00000500"
}

type GenerateTaxInvoiceRequest struct {
	MoveID         uint    `json:"move_id"`                // ID from account.move (Customer Invoice or Vendor Bill)
	TaxInvoiceType string  `json:"tax_invoice_type"`       // "out_tax" or "in_tax"
	CustomNSFP     string  `json:"custom_nsfp,omitempty"`  // Optional manual NSFP (for vendor bills)
	TaxPeriod      int     `json:"tax_period,omitempty"`   // Month (1-12)
	TaxYear        int     `json:"tax_year,omitempty"`     // Year
	TaxRatePPN     float64 `json:"tax_rate_ppn,omitempty"` // e.g. 11.0
	PPhType        string  `json:"pph_type,omitempty"`     // "none", "pph_23", "pph_4_2"
	PPhRate        float64 `json:"pph_rate,omitempty"`
	Notes          string  `json:"notes,omitempty"`
}

type TaxStats struct {
	TotalNSFPAllocated  int     `json:"total_nsfp_allocated"`
	RemainingNSFPQuota  int     `json:"remaining_nsfp_quota"`
	TotalOutTaxInvoices int     `json:"total_out_tax_invoices"`
	TotalInTaxInvoices  int     `json:"total_in_tax_invoices"`
	TotalPPNKeluaran    float64 `json:"total_ppn_keluaran"`
	TotalPPNMasukan     float64 `json:"total_ppn_masukan"`
	NetPPNKurangBayar   float64 `json:"net_ppn_kurang_bayar"` // TotalPPNKeluaran - TotalPPNMasukan
	TotalDPPOutTax      float64 `json:"total_dpp_out_tax"`
	TotalDPPInTax       float64 `json:"total_dpp_in_tax"`
	CurrentTaxPeriod    int     `json:"current_tax_period"`
	CurrentTaxYear      int     `json:"current_tax_year"`
}

type SPTPeriodSummary struct {
	TaxPeriod             int     `json:"tax_period"`
	TaxYear               int     `json:"tax_year"`
	TotalOutTaxInvoices   int     `json:"total_out_tax_invoices"`
	TotalInTaxInvoices    int     `json:"total_in_tax_invoices"`
	DPPPenyerahanKeluaran float64 `json:"dpp_penyerahan_keluaran"`
	PPNKeluaranDipungut   float64 `json:"ppn_keluaran_dipungut"`
	DPPPerolehanMasukan   float64 `json:"dpp_perolehan_masukan"`
	PPNMasukanDikreditkan float64 `json:"ppn_masukan_dikreditkan"`
	PPNKurangLebihBayar   float64 `json:"ppn_kurang_lebih_bayar"` // PPN Keluaran - PPN Masukan
	StatusFiskal          string  `json:"status_fiskal"`          // "Kurang Bayar", "Lebih Bayar", "Nihil"
}

type TaxAuditReport struct {
	TaxPeriod             int      `json:"tax_period"`
	TaxYear               int      `json:"tax_year"`
	ComplianceScore       float64  `json:"compliance_score"` // 0.0 - 100.0%
	RiskLevel             string   `json:"risk_level"`       // "low", "medium", "high", "critical"
	TotalInvoicesAudited  int      `json:"total_invoices_audited"`
	InvalidNPWPCount      int      `json:"invalid_npwp_count"`
	DateDiscrepancies     int      `json:"date_discrepancies"`
	NetTaxPayable         float64  `json:"net_tax_payable"`
	ExecutiveSummary      string   `json:"executive_summary"`
	ActionRecommendations []string `json:"action_recommendations"`
	Engine                string   `json:"engine"`
}
