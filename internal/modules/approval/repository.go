package approval

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// Categories
func (r *Repository) ListCategories(companyID uint) ([]ApprovalCategory, error) {
	var cats []ApprovalCategory
	err := r.db.Where("company_id = ? AND active = ?", companyID, true).Order("id asc").Find(&cats).Error
	return cats, err
}

func (r *Repository) GetCategoryByID(id, companyID uint) (*ApprovalCategory, error) {
	var cat ApprovalCategory
	err := r.db.Where("id = ? AND company_id = ?", id, companyID).First(&cat).Error
	if err != nil {
		return nil, err
	}
	return &cat, nil
}

func (r *Repository) CreateCategory(cat *ApprovalCategory) error {
	return r.db.Create(cat).Error
}

func (r *Repository) UpdateCategory(cat *ApprovalCategory) error {
	return r.db.Save(cat).Error
}

// Requests
func (r *Repository) NextSequence(companyID uint, prefix string) (string, error) {
	year := time.Now().Year()
	var count int64
	r.db.Model(&ApprovalRequest{}).Where("company_id = ?", companyID).Count(&count)
	return fmt.Sprintf("%s/%d/%05d", prefix, year, count+1), nil
}

func (r *Repository) CreateRequest(req *ApprovalRequest) error {
	return r.db.Create(req).Error
}

func (r *Repository) GetRequestByID(id, companyID uint) (*ApprovalRequest, error) {
	var req ApprovalRequest
	err := r.db.Preload("Category").
		Preload("RequestOwner").
		Preload("Approvers", func(db *gorm.DB) *gorm.DB {
			return db.Order("sequence asc")
		}).
		Preload("Approvers.User").
		Where("id = ? AND company_id = ?", id, companyID).
		First(&req).Error
	if err != nil {
		return nil, err
	}
	return &req, nil
}

func (r *Repository) ListRequests(companyID uint, categoryID uint, stage string, userID uint) ([]ApprovalRequest, error) {
	query := r.db.Preload("Category").
		Preload("RequestOwner").
		Preload("Approvers", func(db *gorm.DB) *gorm.DB {
			return db.Order("sequence asc")
		}).
		Preload("Approvers.User").
		Where("company_id = ?", companyID)

	if categoryID > 0 {
		query = query.Where("category_id = ?", categoryID)
	}
	if stage != "" {
		query = query.Where("stage = ?", stage)
	}
	if userID > 0 {
		// Filter requests owned by user or where user is an approver
		query = query.Where("request_owner_id = ? OR id IN (SELECT request_id FROM approval_approvers WHERE user_id = ?)", userID, userID)
	}

	var reqs []ApprovalRequest
	err := query.Order("id desc").Find(&reqs).Error
	return reqs, err
}

func (r *Repository) UpdateRequest(req *ApprovalRequest) error {
	return r.db.Save(req).Error
}

func (r *Repository) UpdateApprover(approver *ApprovalApprover) error {
	return r.db.Save(approver).Error
}

func (r *Repository) CreateApprovers(approvers []ApprovalApprover) error {
	return r.db.Create(&approvers).Error
}

func (r *Repository) DeleteApproversByRequest(requestID uint) error {
	return r.db.Where("request_id = ?", requestID).Delete(&ApprovalApprover{}).Error
}

// KPI Stats
func (r *Repository) GetStats(companyID uint) (map[string]int64, error) {
	stats := make(map[string]int64)
	var total, pending, approved, refused int64

	r.db.Model(&ApprovalRequest{}).Where("company_id = ?", companyID).Count(&total)
	r.db.Model(&ApprovalRequest{}).Where("company_id = ? AND stage = 'pending'", companyID).Count(&pending)
	r.db.Model(&ApprovalRequest{}).Where("company_id = ? AND stage = 'approved'", companyID).Count(&approved)
	r.db.Model(&ApprovalRequest{}).Where("company_id = ? AND stage = 'refused'", companyID).Count(&refused)

	stats["total"] = total
	stats["pending"] = pending
	stats["approved"] = approved
	stats["refused"] = refused

	return stats, nil
}
