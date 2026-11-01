package customs_bonded

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

type Repository interface {
	AutoMigrate() error
	GenerateSubmissionNumber(docType string) string
	CreateDocument(doc *CustomsDocument) error
	GetDocumentByID(id uint) (*CustomsDocument, error)
	GetDocumentBySubmission(subNo string) (*CustomsDocument, error)
	UpdateDocument(doc *CustomsDocument) error
	ListDocuments(companyID uint, docType string, status string) ([]CustomsDocument, error)

	CreateMutation(m *CustomsITInventoryMutation) error
	GetMutationsByBook(companyID uint, bookCategory string, startDate, endDate *time.Time) ([]CustomsITInventoryMutation, error)
	GetLatestMutationEndingQty(productID uint, bookCategory string) float64

	CreateSubcontract(sub *CustomsSubcontract) error
	ListSubcontracts(companyID uint, status string) ([]CustomsSubcontract, error)
	GetSubcontractByID(id uint) (*CustomsSubcontract, error)
	UpdateSubcontract(sub *CustomsSubcontract) error

	GetStats(companyID uint) (*CustomsBondedStats, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) AutoMigrate() error {
	return r.db.AutoMigrate(
		&CustomsDocument{},
		&CustomsDocumentLine{},
		&CustomsITInventoryMutation{},
		&CustomsSubcontract{},
	)
}

func (r *repository) GenerateSubmissionNumber(docType string) string {
	now := time.Now()
	dateStr := now.Format("20060102")
	prefix := "000023"
	if docType == DocTypeBC25 {
		prefix = "000025"
	} else if docType == DocTypeBC30 {
		prefix = "000030"
	} else if docType == DocTypeBC27 {
		prefix = "000027"
	}

	var count int64
	r.db.Model(&CustomsDocument{}).Where("submission_number LIKE ?", fmt.Sprintf("%s-000123-%s-%%", prefix, dateStr)).Count(&count)
	return fmt.Sprintf("%s-000123-%s-%06d", prefix, dateStr, count+1)
}

func (r *repository) CreateDocument(doc *CustomsDocument) error {
	return r.db.Create(doc).Error
}

func (r *repository) GetDocumentByID(id uint) (*CustomsDocument, error) {
	var doc CustomsDocument
	err := r.db.Preload("Lines.Product").Preload("StockPicking").First(&doc, id).Error
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

func (r *repository) GetDocumentBySubmission(subNo string) (*CustomsDocument, error) {
	var doc CustomsDocument
	err := r.db.Preload("Lines.Product").Preload("StockPicking").Where("submission_number = ?", subNo).First(&doc).Error
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

func (r *repository) UpdateDocument(doc *CustomsDocument) error {
	return r.db.Save(doc).Error
}

func (r *repository) ListDocuments(companyID uint, docType string, status string) ([]CustomsDocument, error) {
	var docs []CustomsDocument
	query := r.db.Preload("Lines.Product").Where("company_id = ?", companyID)
	if docType != "" {
		query = query.Where("document_type = ?", docType)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	err := query.Order("id desc").Find(&docs).Error
	return docs, err
}

func (r *repository) CreateMutation(m *CustomsITInventoryMutation) error {
	return r.db.Create(m).Error
}

func (r *repository) GetMutationsByBook(companyID uint, bookCategory string, startDate, endDate *time.Time) ([]CustomsITInventoryMutation, error) {
	var list []CustomsITInventoryMutation
	query := r.db.Preload("Product").Where("company_id = ?", companyID)
	if bookCategory != "" {
		query = query.Where("book_category = ?", bookCategory)
	}
	if startDate != nil {
		query = query.Where("mutation_date >= ?", *startDate)
	}
	if endDate != nil {
		query = query.Where("mutation_date <= ?", *endDate)
	}
	err := query.Order("mutation_date asc, id asc").Find(&list).Error
	return list, err
}

func (r *repository) GetLatestMutationEndingQty(productID uint, bookCategory string) float64 {
	var m CustomsITInventoryMutation
	err := r.db.Where("product_id = ? AND book_category = ?", productID, bookCategory).
		Order("id desc").First(&m).Error
	if err != nil {
		return 0
	}
	return m.QtyEnding
}

func (r *repository) CreateSubcontract(sub *CustomsSubcontract) error {
	return r.db.Create(sub).Error
}

func (r *repository) ListSubcontracts(companyID uint, status string) ([]CustomsSubcontract, error) {
	var list []CustomsSubcontract
	query := r.db.Preload("Product").Where("company_id = ?", companyID)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	err := query.Order("id desc").Find(&list).Error
	return list, err
}

func (r *repository) GetSubcontractByID(id uint) (*CustomsSubcontract, error) {
	var sub CustomsSubcontract
	err := r.db.Preload("Product").First(&sub, id).Error
	if err != nil {
		return nil, err
	}
	return &sub, nil
}

func (r *repository) UpdateSubcontract(sub *CustomsSubcontract) error {
	return r.db.Save(sub).Error
}

func (r *repository) GetStats(companyID uint) (*CustomsBondedStats, error) {
	var stats CustomsBondedStats

	r.db.Model(&CustomsDocument{}).Where("company_id = ?", companyID).Count(&stats.TotalDocuments)
	r.db.Model(&CustomsDocument{}).Where("company_id = ? AND document_type = ?", companyID, DocTypeBC23).Count(&stats.TotalBC23Import)
	r.db.Model(&CustomsDocument{}).Where("company_id = ? AND document_type = ?", companyID, DocTypeBC25).Count(&stats.TotalBC25LocalSale)
	r.db.Model(&CustomsDocument{}).Where("company_id = ? AND document_type = ?", companyID, DocTypeBC30).Count(&stats.TotalBC30Export)
	r.db.Model(&CustomsDocument{}).Where("company_id = ? AND document_type = ?", companyID, DocTypeBC27).Count(&stats.TotalBC27Transfer)
	r.db.Model(&CustomsDocument{}).Where("company_id = ? AND status = 'submitted'", companyID).Count(&stats.PendingSPPBCount)

	// Subcontracts
	r.db.Model(&CustomsSubcontract{}).Where("company_id = ?", companyID).Count(&stats.TotalSubcontracts)
	r.db.Model(&CustomsSubcontract{}).Where("company_id = ? AND status = 'active'", companyID).Count(&stats.ActiveSubcontracts)

	// Financial Aggregations
	row := r.db.Model(&CustomsDocument{}).
		Where("company_id = ? AND status IN ('submitted', 'approved_sppb')", companyID).
		Select("COALESCE(SUM(duty_amount), 0), COALESCE(SUM(vat_amount), 0)").
		Row()
	if row != nil {
		_ = row.Scan(&stats.TotalDutySuspended, &stats.TotalVATExempted)
	}

	// Total Export Value
	var totalExport float64
	r.db.Model(&CustomsDocument{}).
		Where("company_id = ? AND document_type = ? AND status IN ('submitted', 'approved_sppb')", companyID, DocTypeBC30).
		Select("COALESCE(SUM(total_cif_usd), 0)").
		Scan(&totalExport)
	stats.TotalExportValueUSD = totalExport

	return &stats, nil
}
