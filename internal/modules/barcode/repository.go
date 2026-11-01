package barcode

import (
	"strings"
	"time"

	"erp-golang/internal/modules/product"
	"erp-golang/internal/modules/stock"

	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// ---------------------------------------------------------------------------
// Barcode Lookups
// ---------------------------------------------------------------------------

// FindProductByBarcode searches product by barcode or default_code (SKU)
func (r *Repository) FindProductByBarcode(code string, companyID uint) (*product.Product, error) {
	cleanCode := strings.TrimSpace(code)
	if cleanCode == "" {
		return nil, gorm.ErrRecordNotFound
	}

	var prod product.Product
	q := r.db.Preload("Category").Preload("UoM")
	if companyID > 0 {
		q = q.Where("company_id = ?", companyID)
	}

	// Try matching barcode first, then default_code (SKU)
	err := q.Where("barcode = ? OR default_code = ?", cleanCode, cleanCode).First(&prod).Error
	if err != nil {
		// Case-insensitive fallback
		err = q.Where("LOWER(barcode) = LOWER(?) OR LOWER(default_code) = LOWER(?)", cleanCode, cleanCode).First(&prod).Error
	}
	if err != nil {
		return nil, err
	}
	return &prod, nil
}

// FindLocationByBarcode searches stock location by complete_name or name
func (r *Repository) FindLocationByBarcode(code string, companyID uint) (*stock.StockLocation, error) {
	cleanCode := strings.TrimSpace(code)
	if cleanCode == "" {
		return nil, gorm.ErrRecordNotFound
	}

	var loc stock.StockLocation
	q := r.db.Preload("Warehouse")
	if companyID > 0 {
		q = q.Where("company_id = ?", companyID)
	}

	// Support LOC- prefix or raw complete name
	searchTerm := cleanCode
	if strings.HasPrefix(strings.ToUpper(cleanCode), "LOC-") {
		searchTerm = strings.TrimPrefix(cleanCode, "LOC-")
		searchTerm = strings.TrimPrefix(searchTerm, "loc-")
	}

	err := q.Where("complete_name = ? OR name = ?", cleanCode, cleanCode).First(&loc).Error
	if err != nil {
		err = q.Where("LOWER(complete_name) LIKE ? OR LOWER(name) LIKE ?",
			"%"+strings.ToLower(searchTerm)+"%", "%"+strings.ToLower(searchTerm)+"%").First(&loc).Error
	}
	if err != nil {
		return nil, err
	}
	return &loc, nil
}

// FindLotByBarcode searches lot/serial number by barcode or lot name
func (r *Repository) FindLotByBarcode(code string, companyID uint) (*StockProductionLot, error) {
	cleanCode := strings.TrimSpace(code)
	if cleanCode == "" {
		return nil, gorm.ErrRecordNotFound
	}

	var lot StockProductionLot
	q := r.db.Preload("Product").Preload("Location")
	if companyID > 0 {
		q = q.Where("company_id = ?", companyID)
	}

	err := q.Where("barcode = ? OR name = ?", cleanCode, cleanCode).First(&lot).Error
	if err != nil {
		err = q.Where("LOWER(barcode) = LOWER(?) OR LOWER(name) = LOWER(?)", cleanCode, cleanCode).First(&lot).Error
	}
	if err != nil {
		return nil, err
	}
	return &lot, nil
}

// FindPickingByBarcode searches stock picking (transfer/receipt/delivery) by name
func (r *Repository) FindPickingByBarcode(code string, companyID uint) (*stock.StockPicking, error) {
	cleanCode := strings.TrimSpace(code)
	if cleanCode == "" {
		return nil, gorm.ErrRecordNotFound
	}

	var pick stock.StockPicking
	q := r.db.Preload("Location").Preload("LocationDest").Preload("MoveLines.Product")
	if companyID > 0 {
		q = q.Where("company_id = ?", companyID)
	}

	err := q.Where("name = ? OR origin = ?", cleanCode, cleanCode).First(&pick).Error
	if err != nil {
		return nil, err
	}
	return &pick, nil
}

// ---------------------------------------------------------------------------
// Lot & Serial Number CRUD
// ---------------------------------------------------------------------------

func (r *Repository) ListLots(productID uint, companyID uint, search string) ([]StockProductionLot, error) {
	var list []StockProductionLot
	q := r.db.Preload("Product").Preload("Location").Order("id desc")

	if companyID > 0 {
		q = q.Where("company_id = ?", companyID)
	}
	if productID > 0 {
		q = q.Where("product_id = ?", productID)
	}
	if search != "" {
		pattern := "%" + strings.ToLower(search) + "%"
		q = q.Where("LOWER(name) LIKE ? OR LOWER(barcode) LIKE ? OR LOWER(ref) LIKE ?", pattern, pattern, pattern)
	}

	if err := q.Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *Repository) CreateLot(lot *StockProductionLot) error {
	return r.db.Create(lot).Error
}

func (r *Repository) GetLotByID(id uint, companyID uint) (*StockProductionLot, error) {
	var lot StockProductionLot
	q := r.db.Preload("Product").Preload("Location")
	if companyID > 0 {
		q = q.Where("company_id = ?", companyID)
	}
	if err := q.First(&lot, id).Error; err != nil {
		return nil, err
	}
	return &lot, nil
}

// ---------------------------------------------------------------------------
// Statistics
// ---------------------------------------------------------------------------

func (r *Repository) GetStats(companyID uint) (*BarcodeStats, error) {
	var stats BarcodeStats

	// Count products with barcode
	pq := r.db.Model(&product.Product{}).Where("barcode != '' AND barcode IS NOT NULL")
	if companyID > 0 {
		pq = pq.Where("company_id = ?", companyID)
	}
	pq.Count(&stats.TotalProductsBarcoded)

	// Count lots
	lq := r.db.Model(&StockProductionLot{})
	if companyID > 0 {
		lq = lq.Where("company_id = ?", companyID)
	}
	lq.Count(&stats.TotalLots)

	// Active lots with stock > 0
	lsq := r.db.Model(&StockProductionLot{}).Where("qty > 0")
	if companyID > 0 {
		lsq = lsq.Where("company_id = ?", companyID)
	}
	lsq.Count(&stats.ActiveLotsWithStock)

	// Expired lots
	todayStr := time.Now().Format("2006-01-02")
	exq := r.db.Model(&StockProductionLot{}).Where("expiration_date != '' AND expiration_date < ?", todayStr)
	if companyID > 0 {
		exq = exq.Where("company_id = ?", companyID)
	}
	exq.Count(&stats.ExpiredLots)

	// Total locations
	locQ := r.db.Model(&stock.StockLocation{})
	if companyID > 0 {
		locQ = locQ.Where("company_id = ?", companyID)
	}
	locQ.Count(&stats.TotalLocationsBarcoded)

	return &stats, nil
}
