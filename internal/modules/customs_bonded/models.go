package customs_bonded

import (
	"time"

	"erp-golang/internal/modules/product"
	"erp-golang/internal/modules/stock"
	"gorm.io/gorm"
)

// Jenis Dokumen Pabean Standar DJBC Indonesia (CEISA 4.0)
const (
	DocTypeBC20  = "BC_20"  // Impor Umum / Non-Fasilitas
	DocTypeBC23  = "BC_23"  // Pemasukan Barang Impor ke Kawasan Berikat (Penangguhan Bea Masuk & PDRI)
	DocTypeBC25  = "BC_25"  // Pengeluaran Barang Jadi dari KB ke Tempat Lain Dalam Daerah Pabean (TLDDP)
	DocTypeBC261 = "BC_261" // Pengeluaran Sementara ke TLDDP (Subkontrak / Reparasi)
	DocTypeBC262 = "BC_262" // Pemasukan Kembali dari TLDDP setelah Subkontrak
	DocTypeBC27  = "BC_27"  // Pemasukan / Pengeluaran Antar Kawasan Berikat (Kite / KB ke KB lain)
	DocTypeBC30  = "BC_30"  // Ekspor Barang Jadi ke Luar Negeri dari Kawasan Berikat
	DocTypeBC40  = "BC_40"  // Pemasukan Barang dari TLDDP ke Kawasan Berikat
	DocTypeBC41  = "BC_41"  // Pengeluaran Barang ke TLDDP (Retur / Kemasan / Rusak)
)

// Kategori Buku IT Inventory Wajib DJBC
const (
	BookRawMaterial   = "raw_material"   // Buku 1: Bahan Baku & Bahan Penolong
	BookFinishedGoods = "finished_goods" // Buku 2: Barang Jadi (Finished Goods)
	BookCapitalGoods  = "capital_goods"  // Buku 3: Barang Modal, Mesin & Peralatan Pabrik
	BookScrapWaste    = "scrap_waste"    // Buku 4: Sisa / Scrap / Limbah Produksi
)

// CustomsDocument merepresentasikan Dokumen Pemberitahuan Pabean Bea Cukai
type CustomsDocument struct {
	ID                 uint                  `gorm:"primaryKey" json:"id"`
	CompanyID          uint                  `gorm:"index;default:1" json:"company_id"`
	DocumentType       string                `gorm:"size:32;not null;index" json:"document_type"`           // BC_23, BC_25, BC_27, BC_30, dll
	RegistrationNumber string                `gorm:"size:64;index" json:"registration_number"`              // No Pendaftaran dari KPPBC (contoh: 001290)
	RegistrationDate   *time.Time            `json:"registration_date"`                                     // Tanggal Pendaftaran Resmi
	SubmissionNumber   string                `gorm:"size:64;uniqueIndex;not null" json:"submission_number"` // Nomor Pengajuan Aju (26 digit: 000023-000123-20261012-000001)
	SubmissionDate     time.Time             `json:"submission_date"`
	CustomsOfficeCode  string                `gorm:"size:32;default:'040300'" json:"customs_office_code"` // Kantor Pabean (contoh: 040300 - KPPBC TMP A Bekasi)
	PartnerID          uint                  `gorm:"index;not null" json:"partner_id"`                    // Supplier / Buyer / Vendor Subkontrak
	PartnerName        string                `gorm:"size:255" json:"partner_name"`
	CountryOfOrigin    string                `gorm:"size:64;default:'ID'" json:"country_of_origin"`     // ID, CN, JP, DE, US
	CurrencyCode       string                `gorm:"size:8;default:'USD'" json:"currency_code"`         // USD, EUR, IDR, JPY
	ExchangeRate       float64               `gorm:"type:decimal(12,4);default:1" json:"exchange_rate"` // Kurs Keputusan Menkeu (KMK)
	TotalCIFUSD        float64               `gorm:"column:total_cif_usd;type:decimal(16,2);default:0" json:"total_cif_usd"`
	TotalCIFIDR        float64               `gorm:"column:total_cif_idr;type:decimal(16,2);default:0" json:"total_cif_idr"`
	DutyAmount         float64               `gorm:"type:decimal(16,2);default:0" json:"duty_amount"`       // Bea Masuk (Ditangguhkan / Dibayar)
	VATAmount          float64               `gorm:"type:decimal(16,2);default:0" json:"vat_amount"`        // PPN Impor (Tidak Dipungut)
	IncomeTaxAmount    float64               `gorm:"type:decimal(16,2);default:0" json:"income_tax_amount"` // PPh Pasal 22
	StockPickingID     *uint                 `gorm:"index" json:"stock_picking_id"`                         // Tautan ke Surat Jalan / Penerimaan Barang
	StockPicking       *stock.StockPicking   `gorm:"foreignKey:StockPickingID" json:"stock_picking,omitempty"`
	InvoiceNumber      string                `gorm:"size:64" json:"invoice_number"`
	BLNumber           string                `gorm:"size:64" json:"bl_number"` // Bill of Lading / Airway Bill
	ContainerNumber    string                `gorm:"size:64" json:"container_number"`
	SPPBNumber         string                `gorm:"size:64" json:"sppb_number"` // Surat Persetujuan Pengeluaran Barang
	SPPBDate           *time.Time            `json:"sppb_date"`
	Status             string                `gorm:"size:32;default:'draft';index" json:"status"` // draft, submitted, approved_sppb, rejected, closed
	Notes              string                `gorm:"type:text" json:"notes"`
	Lines              []CustomsDocumentLine `gorm:"foreignKey:CustomsDocumentID" json:"lines"`
	CreatedAt          time.Time             `json:"created_at"`
	UpdatedAt          time.Time             `json:"updated_at"`
	DeletedAt          gorm.DeletedAt        `gorm:"index" json:"-"`
}

func (CustomsDocument) TableName() string {
	return "customs_documents"
}

// CustomsDocumentLine merepresentasikan detail item barang pabean
type CustomsDocumentLine struct {
	ID                uint             `gorm:"primaryKey" json:"id"`
	CustomsDocumentID uint             `gorm:"index;not null" json:"customs_document_id"`
	LineNumber        int              `gorm:"default:1" json:"line_number"`
	ProductID         uint             `gorm:"index;not null" json:"product_id"`
	Product           *product.Product `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	HSCode            string           `gorm:"size:32;not null" json:"hs_code"` // Harmonized System Code (8-10 digit)
	ItemDescription   string           `gorm:"size:255;not null" json:"item_description"`
	ItemCategory      string           `gorm:"size:32;default:'raw_material'" json:"item_category"` // raw_material, finished_goods, capital_goods, scrap_waste
	Quantity          float64          `gorm:"type:decimal(12,2);not null" json:"quantity"`
	UoM               string           `gorm:"size:32;default:'KGM'" json:"uom"` // KGM, PCS, MTR, SET
	UnitPriceUSD      float64          `gorm:"type:decimal(16,4);default:0" json:"unit_price_usd"`
	TotalCIFUSD       float64          `gorm:"type:decimal(16,2);default:0" json:"total_cif_usd"`
	DutyTariffPercent float64          `gorm:"type:decimal(6,2);default:5" json:"duty_tariff_percent"` // Bea Masuk %
	DutyAmount        float64          `gorm:"type:decimal(16,2);default:0" json:"duty_amount"`
	VATAmount         float64          `gorm:"type:decimal(16,2);default:0" json:"vat_amount"`
	CreatedAt         time.Time        `json:"created_at"`
	UpdatedAt         time.Time        `json:"updated_at"`
}

func (CustomsDocumentLine) TableName() string {
	return "customs_document_lines"
}

// CustomsITInventoryMutation merepresentasikan Buku Mutasi Wajib DJBC
type CustomsITInventoryMutation struct {
	ID                uint             `gorm:"primaryKey" json:"id"`
	CompanyID         uint             `gorm:"index;default:1" json:"company_id"`
	BookCategory      string           `gorm:"size:32;not null;index" json:"book_category"` // raw_material, finished_goods, capital_goods, scrap_waste
	ProductID         uint             `gorm:"index;not null" json:"product_id"`
	Product           *product.Product `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	ItemCode          string           `gorm:"size:64;not null" json:"item_code"`
	ItemName          string           `gorm:"size:255;not null" json:"item_name"`
	UoM               string           `gorm:"size:32;default:'KGM'" json:"uom"`
	MutationDate      time.Time        `gorm:"index;not null" json:"mutation_date"`
	CustomsDocumentID *uint            `gorm:"index" json:"customs_document_id"`
	CustomsDocType    string           `gorm:"size:32" json:"customs_doc_type"`   // BC_23, BC_25, BC_30, dll
	CustomsDocNumber  string           `gorm:"size:64" json:"customs_doc_number"` // No Pendaftaran / Aju
	ProofNumber       string           `gorm:"size:64" json:"proof_number"`       // Nomor Surat Jalan / Bukti Terima (SPPB / WH-IN)
	QtyBeginning      float64          `gorm:"type:decimal(12,2);default:0" json:"qty_beginning"`
	QtyIn             float64          `gorm:"type:decimal(12,2);default:0" json:"qty_in"`
	QtyOut            float64          `gorm:"type:decimal(12,2);default:0" json:"qty_out"`
	QtyAdjustment     float64          `gorm:"type:decimal(12,2);default:0" json:"qty_adjustment"`
	QtyEnding         float64          `gorm:"type:decimal(12,2);default:0" json:"qty_ending"`
	PhysicalStockQty  float64          `gorm:"type:decimal(12,2);default:0" json:"physical_stock_qty"` // Hasil Stock Opname fisik
	DifferenceQty     float64          `gorm:"type:decimal(12,2);default:0" json:"difference_qty"`     // QtyEnding - PhysicalStockQty
	Notes             string           `gorm:"size:255" json:"notes"`
	CreatedAt         time.Time        `json:"created_at"`
	UpdatedAt         time.Time        `json:"updated_at"`
}

func (CustomsITInventoryMutation) TableName() string {
	return "customs_it_inventory_mutations"
}

// CustomsSubcontract merepresentasikan Pemantauan Kontrak Pekerjaan Subkontrak (BC 2.61 / 2.62)
type CustomsSubcontract struct {
	ID                uint             `gorm:"primaryKey" json:"id"`
	CompanyID         uint             `gorm:"index;default:1" json:"company_id"`
	ContractNumber    string           `gorm:"size:64;uniqueIndex;not null" json:"contract_number"` // SUBK/2026/00001
	CustomsApprovalNo string           `gorm:"size:64;not null" json:"customs_approval_no"`         // No Persetujuan KPPBC
	PartnerID         uint             `gorm:"index;not null" json:"partner_id"`                    // Mitra Kerja Subkontrak
	PartnerName       string           `gorm:"size:255" json:"partner_name"`
	ProductID         uint             `gorm:"index;not null" json:"product_id"`
	Product           *product.Product `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	QtySent           float64          `gorm:"type:decimal(12,2);not null" json:"qty_sent"`
	QtyReturned       float64          `gorm:"type:decimal(12,2);default:0" json:"qty_returned"`
	UoM               string           `gorm:"size:32;default:'PCS'" json:"uom"`
	StartDate         time.Time        `json:"start_date"`
	DueDate           time.Time        `json:"due_date"`                               // Maksimal 60 hari menurut regulasi PMK 131/2018
	BC261DocNo        string           `gorm:"size:64" json:"bc261_doc_no"`            // Dokumen Pengeluaran
	BC262DocNo        string           `gorm:"size:64" json:"bc262_doc_no"`            // Dokumen Pemasukan Kembali
	Status            string           `gorm:"size:32;default:'active'" json:"status"` // active, returned, overdue, closed
	Notes             string           `gorm:"type:text" json:"notes"`
	CreatedAt         time.Time        `json:"created_at"`
	UpdatedAt         time.Time        `json:"updated_at"`
}

func (CustomsSubcontract) TableName() string {
	return "customs_subcontracts"
}

// -------------------------------------------------------------
// DTOs & Analytical Reports
// -------------------------------------------------------------

type CreateCustomsDocumentRequest struct {
	DocumentType       string                     `json:"document_type" binding:"required"`
	RegistrationNumber string                     `json:"registration_number"`
	RegistrationDate   string                     `json:"registration_date"`
	CustomsOfficeCode  string                     `json:"customs_office_code"`
	PartnerID          uint                       `json:"partner_id" binding:"required"`
	PartnerName        string                     `json:"partner_name"`
	CountryOfOrigin    string                     `json:"country_of_origin"`
	CurrencyCode       string                     `json:"currency_code"`
	ExchangeRate       float64                    `json:"exchange_rate"`
	StockPickingID     *uint                      `json:"stock_picking_id"`
	InvoiceNumber      string                     `json:"invoice_number"`
	BLNumber           string                     `json:"bl_number"`
	ContainerNumber    string                     `json:"container_number"`
	Notes              string                     `json:"notes"`
	Lines              []CreateCustomsLineRequest `json:"lines" binding:"required,dive"`
}

type CreateCustomsLineRequest struct {
	ProductID         uint    `json:"product_id" binding:"required"`
	HSCode            string  `json:"hs_code" binding:"required"`
	ItemDescription   string  `json:"item_description" binding:"required"`
	ItemCategory      string  `json:"item_category"` // raw_material, finished_goods, capital_goods, scrap_waste
	Quantity          float64 `json:"quantity" binding:"required,gt=0"`
	UoM               string  `json:"uom"`
	UnitPriceUSD      float64 `json:"unit_price_usd" binding:"required,gt=0"`
	DutyTariffPercent float64 `json:"duty_tariff_percent"`
}

type ApproveSPPBRequest struct {
	SPPBNumber string `json:"sppb_number" binding:"required"`
	SPPBDate   string `json:"sppb_date" binding:"required"`
	Notes      string `json:"notes"`
}

type CustomsBondedStats struct {
	TotalDocuments      int64   `json:"total_documents"`
	TotalBC23Import     int64   `json:"total_bc23_import"`
	TotalBC25LocalSale  int64   `json:"total_bc25_local_sale"`
	TotalBC30Export     int64   `json:"total_bc30_export"`
	TotalBC27Transfer   int64   `json:"total_bc27_transfer"`
	TotalSubcontracts   int64   `json:"total_subcontracts"`
	ActiveSubcontracts  int64   `json:"active_subcontracts"`
	TotalDutySuspended  float64 `json:"total_duty_suspended_idr"` // Total Bea Masuk Ditangguhkan
	TotalVATExempted    float64 `json:"total_vat_exempted_idr"`   // Total PPN Tidak Dipungut
	TotalExportValueUSD float64 `json:"total_export_value_usd"`
	PendingSPPBCount    int64   `json:"pending_sppb_count"`
}

type ITInventoryBookReportItem struct {
	ItemCode      string  `json:"item_code"`
	ItemName      string  `json:"item_name"`
	UoM           string  `json:"uom"`
	QtyBeginning  float64 `json:"qty_beginning"`
	QtyIn         float64 `json:"qty_in"`
	QtyOut        float64 `json:"qty_out"`
	QtyEnding     float64 `json:"qty_ending"`
	PhysicalStock float64 `json:"physical_stock"`
	Difference    float64 `json:"difference"`
	Status        string  `json:"status"` // balanced, surplus, deficit
}

type AICustomsComplianceReport struct {
	ComplianceScore            float64  `json:"compliance_score"`  // 0 - 100
	ComplianceStatus           string   `json:"compliance_status"` // COMPLIANT, WARNING, AUDIT_RISK
	TotalDiscrepancyItems      int      `json:"total_discrepancy_items"`
	OverdueSubcontractsCount   int      `json:"overdue_subcontracts_count"`
	DutySuspendedRiskAmountIDR float64  `json:"duty_suspended_risk_amount_idr"`
	BOMConversionIntegrityRate float64  `json:"bom_conversion_integrity_rate"`
	ExecutiveSummary           string   `json:"executive_summary"`
	AuditFindings              []string `json:"audit_findings"`
	ActionableRecommendations  []string `json:"actionable_recommendations"`
	AIProvider                 string   `json:"ai_provider"`
	GeneratedAt                string   `json:"generated_at"`
}
