package reports

import "time"

// ReportTemplateItem represents a catalog entry for available printable reports
type ReportTemplateItem struct {
	ModuleCode      string `json:"module_code"`   // e.g. "sale", "purchase", "account", "customs_bonded"
	ModuleName      string `json:"module_name"`   // e.g. "Sales & Penjualan"
	DocumentType    string `json:"document_type"` // e.g. "Sales Order / Quotation"
	TitleIndonesian string `json:"title_id"`      // e.g. "Surat Pesanan Penjualan"
	Icon            string `json:"icon"`          // FontAwesome icon class
	PaperFormat     string `json:"paper_format"`  // e.g. "A4 Portrait", "A4 Landscape", "Thermal 80mm"
	Endpoint        string `json:"endpoint"`      // e.g. "/api/v1/reports/print/sale/:id"
	Description     string `json:"description"`
}

// ReportCatalogResponse provides the full catalog of printable documents
type ReportCatalogResponse struct {
	TotalReports int                  `json:"total_reports"`
	GeneratedAt  time.Time            `json:"generated_at"`
	Catalog      []ReportTemplateItem `json:"catalog"`
}
