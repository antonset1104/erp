package attachment

import (
	"strings"

	"gorm.io/gorm"
)

type Repository interface {
	Create(att *Attachment) error
	FindByID(id uint) (*Attachment, error)
	Update(att *Attachment) error
	Delete(id uint) error
	List(companyID uint, filter AttachmentFilter) ([]Attachment, int64, error)
	FindByModelAndID(companyID uint, resModel string, resID uint) ([]Attachment, error)
	GetStats(companyID uint) (map[string]interface{}, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(att *Attachment) error {
	return r.db.Create(att).Error
}

func (r *repository) FindByID(id uint) (*Attachment, error) {
	var att Attachment
	if err := r.db.First(&att, id).Error; err != nil {
		return nil, err
	}
	return &att, nil
}

func (r *repository) Update(att *Attachment) error {
	return r.db.Save(att).Error
}

func (r *repository) Delete(id uint) error {
	return r.db.Delete(&Attachment{}, id).Error
}

func (r *repository) List(companyID uint, filter AttachmentFilter) ([]Attachment, int64, error) {
	var attachments []Attachment
	var total int64

	q := r.db.Model(&Attachment{}).Where("company_id = ?", companyID)

	if filter.ResModel != "" {
		q = q.Where("res_model = ?", filter.ResModel)
	}
	if filter.ResID > 0 {
		q = q.Where("res_id = ?", filter.ResID)
	}
	if filter.OCRStatus != "" {
		q = q.Where("ocr_status = ?", filter.OCRStatus)
	}
	if filter.Search != "" {
		s := "%" + strings.ToLower(filter.Search) + "%"
		q = q.Where("LOWER(name) LIKE ? OR LOWER(description) LIKE ? OR LOWER(res_name) LIKE ?", s, s, s)
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	if err := q.Order("id desc").Limit(limit).Offset(offset).Find(&attachments).Error; err != nil {
		return nil, 0, err
	}

	return attachments, total, nil
}

func (r *repository) FindByModelAndID(companyID uint, resModel string, resID uint) ([]Attachment, error) {
	var attachments []Attachment
	err := r.db.Where("company_id = ? AND res_model = ? AND res_id = ?", companyID, resModel, resID).
		Order("id desc").
		Find(&attachments).Error
	return attachments, err
}

func (r *repository) GetStats(companyID uint) (map[string]interface{}, error) {
	var totalDocs int64
	var totalBytes int64
	var ocrProcessed int64
	var expenseReceipts int64

	r.db.Model(&Attachment{}).Where("company_id = ?", companyID).Count(&totalDocs)
	r.db.Model(&Attachment{}).Where("company_id = ?", companyID).Select("COALESCE(SUM(file_size), 0)").Scan(&totalBytes)
	r.db.Model(&Attachment{}).Where("company_id = ? AND ocr_status = 'processed'", companyID).Count(&ocrProcessed)
	r.db.Model(&Attachment{}).Where("company_id = ? AND res_model = 'hr.expense'", companyID).Count(&expenseReceipts)

	totalMB := float64(totalBytes) / (1024 * 1024)

	return map[string]interface{}{
		"total_documents":  totalDocs,
		"total_bytes":      totalBytes,
		"total_megabytes":  totalMB,
		"ocr_processed":    ocrProcessed,
		"expense_receipts": expenseReceipts,
	}, nil
}
