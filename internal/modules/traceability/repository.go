package traceability

import (
	"strings"
	"time"

	"erp-golang/internal/modules/barcode"

	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// ---------------------------------------------------------------------------
// Lot Trace Records
// ---------------------------------------------------------------------------

func (r *Repository) CreateTraceRecord(record *LotTraceRecord) error {
	return r.db.Create(record).Error
}

func (r *Repository) ListTraceRecords(lotID uint, companyID uint, eventType string, search string) ([]LotTraceRecord, error) {
	var records []LotTraceRecord
	q := r.db.Preload("Lot").Preload("Product").Where("company_id = ?", companyID)

	if lotID > 0 {
		q = q.Where("lot_id = ?", lotID)
	}
	if eventType != "" {
		q = q.Where("event_type = ?", eventType)
	}
	if search != "" {
		searchLike := "%" + strings.TrimSpace(search) + "%"
		q = q.Where("lot_name LIKE ? OR product_name LIKE ? OR reference LIKE ? OR partner_name LIKE ?",
			searchLike, searchLike, searchLike, searchLike)
	}

	if err := q.Order("id desc").Find(&records).Error; err != nil {
		return nil, err
	}
	return records, nil
}

func (r *Repository) GetTracesByLotID(lotID uint, companyID uint) ([]LotTraceRecord, error) {
	var records []LotTraceRecord
	err := r.db.Where("lot_id = ? AND company_id = ?", lotID, companyID).
		Order("id asc").Find(&records).Error
	return records, err
}

func (r *Repository) GetTracesByParentLotID(parentLotID uint, companyID uint) ([]LotTraceRecord, error) {
	var records []LotTraceRecord
	err := r.db.Where("parent_lot_id = ? AND company_id = ?", parentLotID, companyID).
		Order("id asc").Find(&records).Error
	return records, err
}

func (r *Repository) GetTracesByConsumedLotID(consumedLotID uint, companyID uint) ([]LotTraceRecord, error) {
	var records []LotTraceRecord
	err := r.db.Where("consumed_lot_id = ? AND company_id = ?", consumedLotID, companyID).
		Order("id asc").Find(&records).Error
	return records, err
}

func (r *Repository) GetTracesByProducedLotID(producedLotID uint, companyID uint) ([]LotTraceRecord, error) {
	var records []LotTraceRecord
	err := r.db.Where("produced_lot_id = ? AND company_id = ?", producedLotID, companyID).
		Order("id asc").Find(&records).Error
	return records, err
}

// ---------------------------------------------------------------------------
// Lot & Product Lookups
// ---------------------------------------------------------------------------

func (r *Repository) GetLotByID(lotID uint, companyID uint) (*barcode.StockProductionLot, error) {
	var lot barcode.StockProductionLot
	err := r.db.Preload("Product").Preload("Location").
		Where("id = ? AND company_id = ?", lotID, companyID).First(&lot).Error
	if err != nil {
		return nil, err
	}
	return &lot, nil
}

func (r *Repository) FindLotsForProduct(productID uint, companyID uint, locationID *uint) ([]barcode.StockProductionLot, error) {
	var lots []barcode.StockProductionLot
	q := r.db.Preload("Product").Preload("Location").
		Where("product_id = ? AND company_id = ? AND qty > 0", productID, companyID)

	if locationID != nil && *locationID > 0 {
		q = q.Where("location_id = ?", *locationID)
	}

	// Order by expiration date ascending for FEFO (empty/null dates at the end)
	err := q.Order("CASE WHEN expiration_date = '' OR expiration_date IS NULL THEN 1 ELSE 0 END, expiration_date ASC, id ASC").
		Find(&lots).Error
	return lots, err
}

func (r *Repository) GetAllActiveLots(companyID uint) ([]barcode.StockProductionLot, error) {
	var lots []barcode.StockProductionLot
	err := r.db.Preload("Product").Preload("Location").
		Where("company_id = ? AND qty > 0", companyID).
		Order("CASE WHEN expiration_date = '' OR expiration_date IS NULL THEN 1 ELSE 0 END, expiration_date ASC, id ASC").
		Find(&lots).Error
	return lots, err
}

// ---------------------------------------------------------------------------
// Product Recalls
// ---------------------------------------------------------------------------

func (r *Repository) CreateRecall(recall *ProductRecall) error {
	return r.db.Create(recall).Error
}

func (r *Repository) GetRecallByID(id uint, companyID uint) (*ProductRecall, error) {
	var recall ProductRecall
	err := r.db.Where("id = ? AND company_id = ?", id, companyID).First(&recall).Error
	if err != nil {
		return nil, err
	}
	return &recall, nil
}

func (r *Repository) ListRecalls(companyID uint, state string) ([]ProductRecall, error) {
	var recalls []ProductRecall
	q := r.db.Where("company_id = ?", companyID)
	if state != "" {
		q = q.Where("state = ?", state)
	}
	err := q.Order("id desc").Find(&recalls).Error
	return recalls, err
}

func (r *Repository) UpdateRecall(recall *ProductRecall) error {
	return r.db.Save(recall).Error
}

// ---------------------------------------------------------------------------
// Executive Statistics
// ---------------------------------------------------------------------------

func (r *Repository) GetStats(companyID uint) (*TraceabilityStats, error) {
	stats := &TraceabilityStats{}
	todayStr := time.Now().Format("2006-01-02")
	nearExpiryStr := time.Now().AddDate(0, 0, 30).Format("2006-01-02")

	// Total Lots
	r.db.Model(&barcode.StockProductionLot{}).Where("company_id = ?", companyID).Count(&stats.TotalLots)

	// Lots with Stock > 0
	r.db.Model(&barcode.StockProductionLot{}).Where("company_id = ? AND qty > 0", companyID).Count(&stats.ActiveLotsWithStock)

	// Expired Lots
	r.db.Model(&barcode.StockProductionLot{}).
		Where("company_id = ? AND qty > 0 AND expiration_date != '' AND expiration_date < ?", companyID, todayStr).
		Count(&stats.ExpiredLots)

	// Near Expiry Lots (next 30 days)
	r.db.Model(&barcode.StockProductionLot{}).
		Where("company_id = ? AND qty > 0 AND expiration_date != '' AND expiration_date >= ? AND expiration_date <= ?",
			companyID, todayStr, nearExpiryStr).
		Count(&stats.NearExpiryLots)

	// Total Trace Records
	r.db.Model(&LotTraceRecord{}).Where("company_id = ?", companyID).Count(&stats.TotalTraceRecords)

	// Active Recalls
	r.db.Model(&ProductRecall{}).Where("company_id = ? AND state IN ('active', 'quarantined')", companyID).
		Count(&stats.ActiveRecalls)

	// Total Quarantined Stock from recalls
	r.db.Model(&ProductRecall{}).Where("company_id = ? AND state IN ('active', 'quarantined')", companyID).
		Select("COALESCE(SUM(affected_stock_quarantined), 0)").Scan(&stats.TotalQuarantinedStock)

	return stats, nil
}
