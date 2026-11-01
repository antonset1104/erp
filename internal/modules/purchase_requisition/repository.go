package purchase_requisition

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

// Create persists a new PurchaseRequisition with lines
func (r *Repository) Create(req *PurchaseRequisition) error {
	return r.db.Create(req).Error
}

// GetByID retrieves a PurchaseRequisition with lines, products, vendor, and generated purchase orders
func (r *Repository) GetByID(id uint, companyID uint) (*PurchaseRequisition, error) {
	var record PurchaseRequisition
	q := r.db.Preload("Lines.Product").
		Preload("Vendor").
		Preload("User").
		Preload("PurchaseOrders.Partner").
		Preload("PurchaseOrders.OrderLines.Product")

	if companyID > 0 {
		q = q.Where("company_id = ?", companyID)
	}

	err := q.First(&record, id).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// Update updates a PurchaseRequisition
func (r *Repository) Update(req *PurchaseRequisition) error {
	return r.db.Save(req).Error
}

// List lists purchase requisitions with optional filters
func (r *Repository) List(companyID uint, state string, reqType string, search string) ([]PurchaseRequisition, error) {
	var records []PurchaseRequisition
	q := r.db.Preload("Lines.Product").Preload("Vendor").Preload("User").Order("id desc")

	if companyID > 0 {
		q = q.Where("company_id = ?", companyID)
	}
	if state != "" && state != "all" {
		q = q.Where("state = ?", state)
	}
	if reqType != "" && reqType != "all" {
		q = q.Where("type = ?", reqType)
	}
	if search != "" {
		s := "%" + search + "%"
		q = q.Where("name LIKE ? OR origin LIKE ? OR description LIKE ?", s, s, s)
	}

	err := q.Find(&records).Error
	return records, err
}

// GetLineByID fetches a single requisition line
func (r *Repository) GetLineByID(lineID uint) (*PurchaseRequisitionLine, error) {
	var line PurchaseRequisitionLine
	err := r.db.Preload("Product").First(&line, lineID).Error
	if err != nil {
		return nil, err
	}
	return &line, nil
}

// UpdateLine updates a requisition line (e.g. qty_ordered increment)
func (r *Repository) UpdateLine(line *PurchaseRequisitionLine) error {
	return r.db.Save(line).Error
}

// GetNextSequence generates PR/YYYY/XXXXX or BO/YYYY/XXXXX
func (r *Repository) GetNextSequence(reqType RequisitionType, companyID uint) (string, error) {
	year := time.Now().Year()
	prefix := "PR"
	if reqType == TypeBlanketOrder {
		prefix = "BO"
	}

	var count int64
	pattern := fmt.Sprintf("%s/%d/%%", prefix, year)
	err := r.db.Model(&PurchaseRequisition{}).
		Where("company_id = ? AND name LIKE ?", companyID, pattern).
		Count(&count).Error
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/%d/%05d", prefix, year, count+1), nil
}

// GetStats returns KPI statistics for purchase requisitions
func (r *Repository) GetStats(companyID uint) (*RequisitionStats, error) {
	var stats RequisitionStats

	q := r.db.Model(&PurchaseRequisition{})
	if companyID > 0 {
		q = q.Where("company_id = ?", companyID)
	}

	q.Count(&stats.TotalRequisitions)

	qDraft := r.db.Model(&PurchaseRequisition{}).Where("state = ?", StateDraft)
	if companyID > 0 {
		qDraft = qDraft.Where("company_id = ?", companyID)
	}
	qDraft.Count(&stats.DraftCount)

	qProg := r.db.Model(&PurchaseRequisition{}).Where("state IN ?", []string{string(StateConfirmed), string(StateInProgress)})
	if companyID > 0 {
		qProg = qProg.Where("company_id = ?", companyID)
	}
	qProg.Count(&stats.InProgressCount)

	qDone := r.db.Model(&PurchaseRequisition{}).Where("state = ?", StateDone)
	if companyID > 0 {
		qDone = qDone.Where("company_id = ?", companyID)
	}
	qDone.Count(&stats.DoneCount)

	qBlanket := r.db.Model(&PurchaseRequisition{}).Where("type = ?", TypeBlanketOrder)
	if companyID > 0 {
		qBlanket = qBlanket.Where("company_id = ?", companyID)
	}
	qBlanket.Count(&stats.BlanketOrdersCount)

	qTender := r.db.Model(&PurchaseRequisition{}).Where("type = ? AND state IN ?", TypePurchaseTender, []string{string(StateConfirmed), string(StateInProgress)})
	if companyID > 0 {
		qTender = qTender.Where("company_id = ?", companyID)
	}
	qTender.Count(&stats.ActiveTendersCount)

	// Sum total estimated value
	var sumVal struct {
		Total float64
	}
	r.db.Model(&PurchaseRequisition{}).
		Where("company_id = ? AND state != ?", companyID, StateCancelled).
		Select("COALESCE(SUM(total_amount), 0) as total").
		Scan(&sumVal)
	stats.TotalEstimatedVal = sumVal.Total

	return &stats, nil
}
