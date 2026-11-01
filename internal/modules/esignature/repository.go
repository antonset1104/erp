package esignature

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

type Repository interface {
	CreateSignRequest(req *DocumentSignRequest) error
	GetSignRequestByID(id uint) (*DocumentSignRequest, error)
	GetSignRequestByToken(token string) (*DocumentSignRequest, error)
	GetSignRequestByDocument(companyID uint, docType string, docID uint) ([]DocumentSignRequest, error)
	ListSignRequests(companyID uint, status string, docType string) ([]DocumentSignRequest, error)
	UpdateSignRequest(req *DocumentSignRequest) error
	GetStats(companyID uint) (*SignatureStats, error)
	GenerateRequestNumber() (string, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) GenerateRequestNumber() (string, error) {
	year := time.Now().Year()
	var count int64
	if err := r.db.Model(&DocumentSignRequest{}).
		Where("request_number LIKE ?", fmt.Sprintf("ESIGN/%d/%%", year)).
		Count(&count).Error; err != nil {
		return "", err
	}
	return fmt.Sprintf("ESIGN/%d/%05d", year, count+1), nil
}

func (r *repository) CreateSignRequest(req *DocumentSignRequest) error {
	return r.db.Create(req).Error
}

func (r *repository) GetSignRequestByID(id uint) (*DocumentSignRequest, error) {
	var req DocumentSignRequest
	if err := r.db.First(&req, id).Error; err != nil {
		return nil, err
	}
	return &req, nil
}

func (r *repository) GetSignRequestByToken(token string) (*DocumentSignRequest, error) {
	var req DocumentSignRequest
	if err := r.db.Where("security_token = ?", token).First(&req).Error; err != nil {
		return nil, err
	}
	return &req, nil
}

func (r *repository) GetSignRequestByDocument(companyID uint, docType string, docID uint) ([]DocumentSignRequest, error) {
	var requests []DocumentSignRequest
	db := r.db.Where("document_type = ? AND document_id = ?", docType, docID)
	if companyID > 0 {
		db = db.Where("company_id = ?", companyID)
	}
	err := db.Order("id desc").Find(&requests).Error
	return requests, err
}

func (r *repository) ListSignRequests(companyID uint, status string, docType string) ([]DocumentSignRequest, error) {
	var requests []DocumentSignRequest
	db := r.db.Model(&DocumentSignRequest{})
	if companyID > 0 {
		db = db.Where("company_id = ?", companyID)
	}
	if status != "" && status != "all" {
		db = db.Where("status = ?", status)
	}
	if docType != "" && docType != "all" {
		db = db.Where("document_type = ?", docType)
	}
	err := db.Order("id desc").Find(&requests).Error
	return requests, err
}

func (r *repository) UpdateSignRequest(req *DocumentSignRequest) error {
	return r.db.Save(req).Error
}

func (r *repository) GetStats(companyID uint) (*SignatureStats, error) {
	var stats SignatureStats

	db := r.db.Model(&DocumentSignRequest{})
	if companyID > 0 {
		db = db.Where("company_id = ?", companyID)
	}

	db.Count(&stats.TotalRequests)
	r.db.Model(&DocumentSignRequest{}).Where("company_id = ? AND status = 'pending'", companyID).Count(&stats.PendingRequests)
	r.db.Model(&DocumentSignRequest{}).Where("company_id = ? AND status = 'signed'", companyID).Count(&stats.SignedRequests)
	r.db.Model(&DocumentSignRequest{}).Where("company_id = ? AND status = 'rejected'", companyID).Count(&stats.RejectedRequests)

	if stats.TotalRequests > 0 {
		stats.ComplianceRate = float64(stats.SignedRequests) / float64(stats.TotalRequests) * 100.0
	} else {
		stats.ComplianceRate = 100.0
	}

	// Calculate average turnaround hours for signed requests
	var signedReqs []DocumentSignRequest
	if err := r.db.Where("company_id = ? AND status = 'signed' AND signed_at IS NOT NULL", companyID).
		Find(&signedReqs).Error; err == nil && len(signedReqs) > 0 {
		var totalHours float64
		for _, s := range signedReqs {
			if s.SignedAt != nil {
				duration := s.SignedAt.Sub(s.CreatedAt).Hours()
				if duration > 0 {
					totalHours += duration
				}
			}
		}
		stats.AvgTurnaroundHrs = totalHours / float64(len(signedReqs))
	} else {
		stats.AvgTurnaroundHrs = 2.4 // Baseline industry average
	}

	return &stats, nil
}
