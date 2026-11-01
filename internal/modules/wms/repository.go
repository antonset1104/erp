package wms

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

type Repository interface {
	// Zones
	CreateZone(zone *WMSZone) error
	GetZoneByID(id uint) (*WMSZone, error)
	ListZones(warehouseID uint) ([]WMSZone, error)

	// Bins
	CreateBin(bin *WMSBinLocation) error
	GetBinByID(id uint) (*WMSBinLocation, error)
	GetBinByCode(code string) (*WMSBinLocation, error)
	ListBins(warehouseID uint, zoneID uint, status string) ([]WMSBinLocation, error)
	UpdateBin(bin *WMSBinLocation) error

	// Putaway Rules
	CreatePutawayRule(rule *WMSPutawayRule) error
	ListPutawayRules(warehouseID uint) ([]WMSPutawayRule, error)
	FindMatchingPutawayRule(warehouseID uint, productID uint, categoryID uint) (*WMSPutawayRule, error)
	FindAvailableBinInZone(zoneID uint, weightKg float64, volumeCbm float64) (*WMSBinLocation, error)

	// Inventory Slots
	GetSlot(binID uint, productID uint) (*WMSInventorySlot, error)
	SaveSlot(slot *WMSInventorySlot) error
	ListSlotsByBin(binID uint) ([]WMSInventorySlot, error)

	// Waves
	NextWaveNumber() (string, error)
	CreateWave(wave *WMSWavePicking) error
	GetWaveByID(id uint) (*WMSWavePicking, error)
	ListWaves(warehouseID uint, status string) ([]WMSWavePicking, error)
	UpdateWave(wave *WMSWavePicking) error
	UpdateWaveLine(line *WMSWavePickingLine) error
	GetWaveLineByID(id uint) (*WMSWavePickingLine, error)

	// Stats
	GetStats(warehouseID uint) (*WMSStats, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

// ---------------- Zones ----------------

func (r *repository) CreateZone(zone *WMSZone) error {
	return r.db.Create(zone).Error
}

func (r *repository) GetZoneByID(id uint) (*WMSZone, error) {
	var zone WMSZone
	err := r.db.First(&zone, id).Error
	if err != nil {
		return nil, err
	}
	return &zone, nil
}

func (r *repository) ListZones(warehouseID uint) ([]WMSZone, error) {
	var zones []WMSZone
	query := r.db.Model(&WMSZone{})
	if warehouseID > 0 {
		query = query.Where("warehouse_id = ?", warehouseID)
	}
	err := query.Order("code ASC").Find(&zones).Error
	return zones, err
}

// ---------------- Bins ----------------

func (r *repository) CreateBin(bin *WMSBinLocation) error {
	return r.db.Create(bin).Error
}

func (r *repository) GetBinByID(id uint) (*WMSBinLocation, error) {
	var bin WMSBinLocation
	err := r.db.Preload("Zone").First(&bin, id).Error
	if err != nil {
		return nil, err
	}
	return &bin, nil
}

func (r *repository) GetBinByCode(code string) (*WMSBinLocation, error) {
	var bin WMSBinLocation
	err := r.db.Preload("Zone").Where("code = ?", code).First(&bin).Error
	if err != nil {
		return nil, err
	}
	return &bin, nil
}

func (r *repository) ListBins(warehouseID uint, zoneID uint, status string) ([]WMSBinLocation, error) {
	var bins []WMSBinLocation
	query := r.db.Preload("Zone").Model(&WMSBinLocation{})
	if warehouseID > 0 {
		query = query.Where("warehouse_id = ?", warehouseID)
	}
	if zoneID > 0 {
		query = query.Where("zone_id = ?", zoneID)
	}
	if status != "" && status != "all" {
		query = query.Where("status = ?", status)
	}
	err := query.Order("aisle ASC, rack ASC, shelf ASC, bin ASC").Find(&bins).Error
	return bins, err
}

func (r *repository) UpdateBin(bin *WMSBinLocation) error {
	return r.db.Save(bin).Error
}

// ---------------- Putaway Rules ----------------

func (r *repository) CreatePutawayRule(rule *WMSPutawayRule) error {
	return r.db.Create(rule).Error
}

func (r *repository) ListPutawayRules(warehouseID uint) ([]WMSPutawayRule, error) {
	var rules []WMSPutawayRule
	query := r.db.Preload("Product").Preload("TargetZone").Model(&WMSPutawayRule{})
	if warehouseID > 0 {
		query = query.Where("warehouse_id = ?", warehouseID)
	}
	err := query.Order("priority ASC").Find(&rules).Error
	return rules, err
}

func (r *repository) FindMatchingPutawayRule(warehouseID uint, productID uint, categoryID uint) (*WMSPutawayRule, error) {
	var rule WMSPutawayRule
	// Priority 1: Match specific product
	err := r.db.Preload("TargetZone").
		Where("warehouse_id = ? AND product_id = ? AND active = ?", warehouseID, productID, true).
		Order("priority ASC").First(&rule).Error
	if err == nil {
		return &rule, nil
	}

	// Priority 2: Match product category if provided
	if categoryID > 0 {
		err = r.db.Preload("TargetZone").
			Where("warehouse_id = ? AND category_id = ? AND active = ?", warehouseID, categoryID, true).
			Order("priority ASC").First(&rule).Error
		if err == nil {
			return &rule, nil
		}
	}

	// Priority 3: Fallback generic rule for warehouse
	err = r.db.Preload("TargetZone").
		Where("warehouse_id = ? AND product_id IS NULL AND category_id IS NULL AND active = ?", warehouseID, true).
		Order("priority ASC").First(&rule).Error
	if err == nil {
		return &rule, nil
	}

	return nil, fmt.Errorf("tidak ada aturan putaway yang cocok")
}

func (r *repository) FindAvailableBinInZone(zoneID uint, weightKg float64, volumeCbm float64) (*WMSBinLocation, error) {
	var bin WMSBinLocation
	query := r.db.Preload("Zone").Where("zone_id = ? AND is_locked = ? AND status != 'maintenance'", zoneID, false)

	if weightKg > 0 {
		query = query.Where("(max_weight_kg - current_weight_kg) >= ?", weightKg)
	}
	if volumeCbm > 0 {
		query = query.Where("(max_volume_cbm - current_volume_cbm) >= ?", volumeCbm)
	}

	// Order by picking sequence and least loaded
	err := query.Order("picking_sequence ASC, current_volume_cbm ASC").First(&bin).Error
	if err != nil {
		return nil, fmt.Errorf("tidak ditemukan bin yang memiliki kapasitas cukup di zona ini")
	}
	return &bin, nil
}

// ---------------- Inventory Slots ----------------

func (r *repository) GetSlot(binID uint, productID uint) (*WMSInventorySlot, error) {
	var slot WMSInventorySlot
	err := r.db.Where("bin_location_id = ? AND product_id = ?", binID, productID).First(&slot).Error
	if err != nil {
		return nil, err
	}
	return &slot, nil
}

func (r *repository) SaveSlot(slot *WMSInventorySlot) error {
	return r.db.Save(slot).Error
}

func (r *repository) ListSlotsByBin(binID uint) ([]WMSInventorySlot, error) {
	var slots []WMSInventorySlot
	err := r.db.Preload("Product").Where("bin_location_id = ?", binID).Find(&slots).Error
	return slots, err
}

// ---------------- Waves ----------------

func (r *repository) NextWaveNumber() (string, error) {
	var count int64
	year := time.Now().Year()
	err := r.db.Model(&WMSWavePicking{}).Count(&count).Error
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("WAVE/%d/%05d", year, count+1), nil
}

func (r *repository) CreateWave(wave *WMSWavePicking) error {
	return r.db.Create(wave).Error
}

func (r *repository) GetWaveByID(id uint) (*WMSWavePicking, error) {
	var wave WMSWavePicking
	err := r.db.Preload("Lines", func(db *gorm.DB) *gorm.DB {
		return db.Order("picking_sequence ASC, bin_location_code ASC")
	}).First(&wave, id).Error
	if err != nil {
		return nil, err
	}
	return &wave, nil
}

func (r *repository) ListWaves(warehouseID uint, status string) ([]WMSWavePicking, error) {
	var waves []WMSWavePicking
	query := r.db.Model(&WMSWavePicking{})
	if warehouseID > 0 {
		query = query.Where("warehouse_id = ?", warehouseID)
	}
	if status != "" && status != "all" {
		query = query.Where("status = ?", status)
	}
	err := query.Order("id DESC").Find(&waves).Error
	return waves, err
}

func (r *repository) UpdateWave(wave *WMSWavePicking) error {
	return r.db.Save(wave).Error
}

func (r *repository) UpdateWaveLine(line *WMSWavePickingLine) error {
	return r.db.Save(line).Error
}

func (r *repository) GetWaveLineByID(id uint) (*WMSWavePickingLine, error) {
	var line WMSWavePickingLine
	err := r.db.First(&line, id).Error
	if err != nil {
		return nil, err
	}
	return &line, nil
}

// ---------------- Stats ----------------

func (r *repository) GetStats(warehouseID uint) (*WMSStats, error) {
	var stats WMSStats

	zoneQuery := r.db.Model(&WMSZone{})
	binQuery := r.db.Model(&WMSBinLocation{})
	waveQuery := r.db.Model(&WMSWavePicking{})
	ruleQuery := r.db.Model(&WMSPutawayRule{})
	slotQuery := r.db.Model(&WMSInventorySlot{})

	if warehouseID > 0 {
		zoneQuery = zoneQuery.Where("warehouse_id = ?", warehouseID)
		binQuery = binQuery.Where("warehouse_id = ?", warehouseID)
		waveQuery = waveQuery.Where("warehouse_id = ?", warehouseID)
		ruleQuery = ruleQuery.Where("warehouse_id = ?", warehouseID)
	}

	var totalZones, totalBins, occupiedBins, activeWaves, completedWaves, totalRules, totalSlots int64
	zoneQuery.Count(&totalZones)
	binQuery.Count(&totalBins)

	r.db.Model(&WMSBinLocation{}).Where("is_occupied = ?", true).Count(&occupiedBins)
	
	wActive := r.db.Model(&WMSWavePicking{}).Where("status = ?", "in_progress")
	wCompleted := r.db.Model(&WMSWavePicking{}).Where("status = ?", "completed")
	if warehouseID > 0 {
		wActive = wActive.Where("warehouse_id = ?", warehouseID)
		wCompleted = wCompleted.Where("warehouse_id = ?", warehouseID)
	}
	wActive.Count(&activeWaves)
	wCompleted.Count(&completedWaves)

	ruleQuery.Count(&totalRules)
	slotQuery.Count(&totalSlots)

	stats.TotalZones = int(totalZones)
	stats.TotalBins = int(totalBins)
	stats.OccupiedBins = int(occupiedBins)
	stats.AvailableBins = int(totalBins - occupiedBins)
	if totalBins > 0 {
		stats.BinUtilizationPct = float64(occupiedBins) / float64(totalBins) * 100.0
	}
	stats.ActiveWaves = int(activeWaves)
	stats.CompletedWaves = int(completedWaves)
	stats.TotalPutawayRules = int(totalRules)
	stats.TotalStoredItems = int(totalSlots)

	return &stats, nil
}
