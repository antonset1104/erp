package wms

import (
	"time"

	"erp-golang/internal/core/models"
	"erp-golang/internal/modules/product"
)

// WMSZone defines storage zones inside a warehouse (e.g. Dry, Cold Chain, Hazardous, Staging, Bulk)
type WMSZone struct {
	models.ModelBase
	WarehouseID       uint    `gorm:"index;not null" json:"warehouse_id"`
	Code              string  `gorm:"size:32;not null;uniqueIndex" json:"code"` // e.g. ZONE-DRY-A, ZONE-COLD-B
	Name              string  `gorm:"size:128;not null" json:"name"`
	ZoneType          string  `gorm:"size:32;not null;default:'dry'" json:"zone_type"` // dry, cold, hazardous, staging, bulk
	TemperatureMin    float64 `gorm:"default:0" json:"temperature_min"`
	TemperatureMax    float64 `gorm:"default:35" json:"temperature_max"`
	MaxWeightCapacity float64 `gorm:"default:10000" json:"max_weight_capacity"` // in KG
	CurrentWeight     float64 `gorm:"default:0" json:"current_weight"`
	Active            bool    `gorm:"default:true" json:"active"`
	Description       string  `gorm:"type:text" json:"description"`
}

// WMSBinLocation defines a 3D coordinate bin within warehouse (Aisle, Rack, Shelf, Bin)
type WMSBinLocation struct {
	models.ModelBase
	WarehouseID      uint     `gorm:"index;not null" json:"warehouse_id"`
	ZoneID           uint     `gorm:"index;not null" json:"zone_id"`
	Zone             *WMSZone `gorm:"foreignKey:ZoneID" json:"zone,omitempty"`
	Code             string   `gorm:"size:64;not null;uniqueIndex" json:"code"` // e.g. WH1-DRY-A01-R02-S03-B01
	Aisle            string   `gorm:"size:16;not null" json:"aisle"`            // e.g. A01
	Rack             string   `gorm:"size:16;not null" json:"rack"`             // e.g. R02
	Shelf            string   `gorm:"size:16;not null" json:"shelf"`            // e.g. S03 (Tingkat rak)
	Bin              string   `gorm:"size:16;not null" json:"bin"`              // e.g. B01 (Nomor kotak)
	MaxWeightKg      float64  `gorm:"default:500" json:"max_weight_kg"`
	MaxVolumeCbm     float64  `gorm:"default:1.5" json:"max_volume_cbm"`
	CurrentWeightKg  float64  `gorm:"default:0" json:"current_weight_kg"`
	CurrentVolumeCbm float64  `gorm:"default:0" json:"current_volume_cbm"`
	IsOccupied       bool     `gorm:"default:false" json:"is_occupied"`
	IsLocked         bool     `gorm:"default:false" json:"is_locked"`
	Status           string   `gorm:"size:32;default:'available'" json:"status"`   // available, occupied, maintenance, reserved
	VelocityCategory string   `gorm:"size:8;default:'B'" json:"velocity_category"` // A (fast), B (medium), C (slow)
	PickingSequence  int      `gorm:"default:10" json:"picking_sequence"`          // traversal routing sort order
}

// WMSInventorySlot represents products currently stored inside a specific Bin Location
type WMSInventorySlot struct {
	models.ModelBase
	BinLocationID    uint             `gorm:"index;not null" json:"bin_location_id"`
	BinLocation      *WMSBinLocation  `gorm:"foreignKey:BinLocationID" json:"bin_location,omitempty"`
	ProductID        uint             `gorm:"index;not null" json:"product_id"`
	Product          *product.Product `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	LotID            *uint            `gorm:"index" json:"lot_id,omitempty"`
	LotNumber        string           `gorm:"size:64" json:"lot_number,omitempty"`
	Quantity         float64          `gorm:"not null;default:0" json:"quantity"`
	WeightKg         float64          `gorm:"default:0" json:"weight_kg"`
	VolumeCbm        float64          `gorm:"default:0" json:"volume_cbm"`
	LastStockCountAt *time.Time       `json:"last_stock_count_at,omitempty"`
}

// WMSPutawayRule dictates automated bin placement for received inventory
type WMSPutawayRule struct {
	models.ModelBase
	WarehouseID  uint             `gorm:"index;not null" json:"warehouse_id"`
	ProductID    *uint            `gorm:"index" json:"product_id,omitempty"`
	Product      *product.Product `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	CategoryID   *uint            `gorm:"index" json:"category_id,omitempty"`
	TargetZoneID uint             `gorm:"index;not null" json:"target_zone_id"`
	TargetZone   *WMSZone         `gorm:"foreignKey:TargetZoneID" json:"target_zone,omitempty"`
	Strategy     string           `gorm:"size:32;default:'fastest_access'" json:"strategy"` // fastest_access, least_loaded, dedicated_bin, cold_storage
	Priority     int              `gorm:"default:10" json:"priority"`                       // lower number = higher precedence
	Active       bool             `gorm:"default:true" json:"active"`
	Description  string           `gorm:"type:text" json:"description"`
}

// WMSWavePicking groups multiple delivery orders / stock pickings into a single optimized wave
type WMSWavePicking struct {
	models.ModelBase
	WaveNumber         string               `gorm:"size:64;not null;uniqueIndex" json:"wave_number"` // WAVE/2026/00001
	Name               string               `gorm:"size:128;not null" json:"name"`
	WarehouseID        uint                 `gorm:"index;not null" json:"warehouse_id"`
	Status             string               `gorm:"size:32;default:'draft'" json:"status"` // draft, in_progress, completed, cancelled
	AssignedPickerID   *uint                `gorm:"index" json:"assigned_picker_id,omitempty"`
	AssignedPickerName string               `gorm:"size:128" json:"assigned_picker_name"`
	TotalPickings      int                  `gorm:"default:0" json:"total_pickings"`
	TotalLines         int                  `gorm:"default:0" json:"total_lines"`
	PickedLines        int                  `gorm:"default:0" json:"picked_lines"`
	StartedAt          *time.Time           `json:"started_at,omitempty"`
	CompletedAt        *time.Time           `json:"completed_at,omitempty"`
	Notes              string               `gorm:"type:text" json:"notes"`
	Lines              []WMSWavePickingLine `gorm:"foreignKey:WaveID" json:"lines,omitempty"`
}

// WMSWavePickingLine represents an individual item line in wave picking route
type WMSWavePickingLine struct {
	models.ModelBase
	WaveID            uint       `gorm:"index;not null" json:"wave_id"`
	PickingID         uint       `gorm:"index;not null" json:"picking_id"`
	PickingName       string     `gorm:"size:64" json:"picking_name"`
	StockMoveID       uint       `gorm:"index;not null" json:"stock_move_id"`
	ProductID         uint       `gorm:"index;not null" json:"product_id"`
	ProductName       string     `gorm:"size:255" json:"product_name"`
	ProductSku        string     `gorm:"size:64" json:"product_sku"`
	BinLocationID     uint       `gorm:"index;not null" json:"bin_location_id"`
	BinLocationCode   string     `gorm:"size:64" json:"bin_location_code"`
	PickingSequence   int        `gorm:"default:10" json:"picking_sequence"`
	QuantityRequested float64    `gorm:"not null" json:"quantity_requested"`
	QuantityPicked    float64    `gorm:"default:0" json:"quantity_picked"`
	Status            string     `gorm:"size:32;default:'pending'" json:"status"` // pending, picked, shortage
	PickedAt          *time.Time `json:"picked_at,omitempty"`
}

// DTOs for WMS operations
type CreateZoneRequest struct {
	WarehouseID       uint    `json:"warehouse_id" binding:"required"`
	Code              string  `json:"code" binding:"required"`
	Name              string  `json:"name" binding:"required"`
	ZoneType          string  `json:"zone_type"`
	TemperatureMin    float64 `json:"temperature_min"`
	TemperatureMax    float64 `json:"temperature_max"`
	MaxWeightCapacity float64 `json:"max_weight_capacity"`
	Description       string  `json:"description"`
}

type CreateBinRequest struct {
	WarehouseID      uint    `json:"warehouse_id" binding:"required"`
	ZoneID           uint    `json:"zone_id" binding:"required"`
	Aisle            string  `json:"aisle" binding:"required"`
	Rack             string  `json:"rack" binding:"required"`
	Shelf            string  `json:"shelf" binding:"required"`
	Bin              string  `json:"bin" binding:"required"`
	MaxWeightKg      float64 `json:"max_weight_kg"`
	MaxVolumeCbm     float64 `json:"max_volume_cbm"`
	VelocityCategory string  `json:"velocity_category"`
	PickingSequence  int     `json:"picking_sequence"`
}

type CreatePutawayRuleRequest struct {
	WarehouseID  uint   `json:"warehouse_id" binding:"required"`
	ProductID    *uint  `json:"product_id"`
	CategoryID   *uint  `json:"category_id"`
	TargetZoneID uint   `json:"target_zone_id" binding:"required"`
	Strategy     string `json:"strategy"`
	Priority     int    `json:"priority"`
	Description  string `json:"description"`
}

type SuggestPutawayRequest struct {
	WarehouseID uint    `json:"warehouse_id" binding:"required"`
	ProductID   uint    `json:"product_id" binding:"required"`
	Quantity    float64 `json:"quantity" binding:"required"`
	WeightKg    float64 `json:"weight_kg"`
	VolumeCbm   float64 `json:"volume_cbm"`
}

type PutawaySuggestion struct {
	BinLocationID   uint    `json:"bin_location_id"`
	BinLocationCode string  `json:"bin_location_code"`
	ZoneName        string  `json:"zone_name"`
	ZoneType        string  `json:"zone_type"`
	RemainingWeight float64 `json:"remaining_weight_kg"`
	RemainingVolume float64 `json:"remaining_volume_cbm"`
	StrategyApplied string  `json:"strategy_applied"`
	ConfidenceScore int     `json:"confidence_score"`
}

type CreateWavePickingRequest struct {
	Name               string `json:"name" binding:"required"`
	WarehouseID        uint   `json:"warehouse_id" binding:"required"`
	PickingIDs         []uint `json:"picking_ids" binding:"required"`
	AssignedPickerID   *uint  `json:"assigned_picker_id"`
	AssignedPickerName string `json:"assigned_picker_name"`
	Notes              string `json:"notes"`
}

type PickLineRequest struct {
	QuantityPicked float64 `json:"quantity_picked" binding:"required"`
}

type WMSStats struct {
	TotalZones        int     `json:"total_zones"`
	TotalBins         int     `json:"total_bins"`
	OccupiedBins      int     `json:"occupied_bins"`
	AvailableBins     int     `json:"available_bins"`
	BinUtilizationPct float64 `json:"bin_utilization_pct"`
	ActiveWaves       int     `json:"active_waves"`
	CompletedWaves    int     `json:"completed_waves"`
	TotalPutawayRules int     `json:"total_putaway_rules"`
	TotalStoredItems  int     `json:"total_stored_items"`
}

type AIWarehouseSlottingReport struct {
	OverallStatus          string   `json:"overall_status"` // OPTIMIZED, SUB_OPTIMAL, HIGH_CONGESTION
	EfficiencyScore        int      `json:"efficiency_score"`
	AverageTravelDistanceM float64  `json:"average_travel_distance_m"`
	FastMovingInColdBins   int      `json:"fast_moving_in_cold_bins"`
	RecommendedRelocations []string `json:"recommended_relocations"`
	ExecutiveSummary       string   `json:"executive_summary"`
	ActionableAdvice       []string `json:"actionable_advice"`
	AIProvider             string   `json:"ai_provider"`
}
