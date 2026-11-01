package wms

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"erp-golang/internal/core/ai"
	"erp-golang/internal/modules/chatter"
	"erp-golang/internal/modules/stock"
)

type Module struct {
	db         *gorm.DB
	stockSvc   *stock.Service
	chatterSvc *chatter.Service
	aiClient   *ai.Client
	repo       Repository
	service    Service
	handler    *Handler
}

func NewModule(db *gorm.DB, stockSvc *stock.Service, chatterSvc *chatter.Service, aiClient *ai.Client) *Module {
	repo := NewRepository(db)
	svc := NewService(repo, db, stockSvc, chatterSvc, aiClient)
	handler := NewHandler(svc)

	return &Module{
		db:         db,
		stockSvc:   stockSvc,
		chatterSvc: chatterSvc,
		aiClient:   aiClient,
		repo:       repo,
		service:    svc,
		handler:    handler,
	}
}

func (m *Module) Name() string {
	return "wms"
}

func (m *Module) AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&WMSZone{},
		&WMSBinLocation{},
		&WMSPutawayRule{},
		&WMSInventorySlot{},
		&WMSWavePicking{},
		&WMSWavePickingLine{},
	)
}

func (m *Module) RegisterRoutes(api *gin.RouterGroup, authMW gin.HandlerFunc) {
	m.handler.RegisterRoutes(api, authMW)
}

func (m *Module) Service() Service {
	return m.service
}

func (m *Module) Seed(db *gorm.DB) error {
	var count int64
	db.Model(&WMSZone{}).Count(&count)
	if count > 0 {
		return nil
	}

	log.Println("🌱 Seeding Enterprise WMS (Zones, Bins, Putaway & Wave Picking) Sample Data...")

	// 1. Warehouse Zones
	zones := []WMSZone{
		{
			WarehouseID:       1,
			Code:              "ZONE-DRY-A",
			Name:              "Zona Penyimpanan Kering A (Fast-Moving)",
			ZoneType:          "dry",
			TemperatureMin:    20,
			TemperatureMax:    28,
			MaxWeightCapacity: 50000,
			Active:            true,
			Description:       "Zona penyimpanan utama untuk barang ritel dan komponen fast-moving dekat pintu loading dock",
		},
		{
			WarehouseID:       1,
			Code:              "ZONE-DRY-B",
			Name:              "Zona Penyimpanan Kering B (Bulk & Raw Material)",
			ZoneType:          "bulk",
			TemperatureMin:    20,
			TemperatureMax:    32,
			MaxWeightCapacity: 100000,
			Active:            true,
			Description:       "Zona lantai dan palet untuk bahan baku industri berat dan mesin",
		},
		{
			WarehouseID:       1,
			Code:              "ZONE-COLD-C",
			Name:              "Zona Rantai Dingin C (Cold Chain Storage)",
			ZoneType:          "cold",
			TemperatureMin:    2,
			TemperatureMax:    8,
			MaxWeightCapacity: 20000,
			Active:            true,
			Description:       "Zona pengatur suhu terkontrol untuk bahan mudah rusak dan farmasi",
		},
		{
			WarehouseID:       1,
			Code:              "ZONE-STAGE-OUT",
			Name:              "Zona Staging & Sortir Pengiriman Keluar",
			ZoneType:          "staging",
			TemperatureMin:    20,
			TemperatureMax:    30,
			MaxWeightCapacity: 30000,
			Active:            true,
			Description:       "Area buffer konsolidasi barang hasil wave picking sebelum muat ke truk ekspedisi",
		},
	}
	for i := range zones {
		db.Create(&zones[i])
	}

	// 2. 3D Bin Locations in Zone A (Fast Moving) and Zone B
	bins := []WMSBinLocation{
		// Aisle A01, Rack R01
		{WarehouseID: 1, ZoneID: zones[0].ID, Code: "WH1-ZONE-DRY-A-A01-R01-S01-B01", Aisle: "A01", Rack: "R01", Shelf: "S01", Bin: "B01", MaxWeightKg: 500, MaxVolumeCbm: 1.2, CurrentWeightKg: 120, CurrentVolumeCbm: 0.35, IsOccupied: true, Status: "occupied", VelocityCategory: "A", PickingSequence: 1},
		{WarehouseID: 1, ZoneID: zones[0].ID, Code: "WH1-ZONE-DRY-A-A01-R01-S01-B02", Aisle: "A01", Rack: "R01", Shelf: "S01", Bin: "B02", MaxWeightKg: 500, MaxVolumeCbm: 1.2, CurrentWeightKg: 85, CurrentVolumeCbm: 0.22, IsOccupied: true, Status: "occupied", VelocityCategory: "A", PickingSequence: 2},
		{WarehouseID: 1, ZoneID: zones[0].ID, Code: "WH1-ZONE-DRY-A-A01-R01-S02-B01", Aisle: "A01", Rack: "R01", Shelf: "S02", Bin: "B01", MaxWeightKg: 500, MaxVolumeCbm: 1.2, CurrentWeightKg: 0, CurrentVolumeCbm: 0, IsOccupied: false, Status: "available", VelocityCategory: "B", PickingSequence: 3},
		{WarehouseID: 1, ZoneID: zones[0].ID, Code: "WH1-ZONE-DRY-A-A01-R01-S02-B02", Aisle: "A01", Rack: "R01", Shelf: "S02", Bin: "B02", MaxWeightKg: 500, MaxVolumeCbm: 1.2, CurrentWeightKg: 0, CurrentVolumeCbm: 0, IsOccupied: false, Status: "available", VelocityCategory: "B", PickingSequence: 4},

		// Aisle A01, Rack R02
		{WarehouseID: 1, ZoneID: zones[0].ID, Code: "WH1-ZONE-DRY-A-A01-R02-S01-B01", Aisle: "A01", Rack: "R02", Shelf: "S01", Bin: "B01", MaxWeightKg: 600, MaxVolumeCbm: 1.5, CurrentWeightKg: 0, CurrentVolumeCbm: 0, IsOccupied: false, Status: "available", VelocityCategory: "B", PickingSequence: 5},
		{WarehouseID: 1, ZoneID: zones[0].ID, Code: "WH1-ZONE-DRY-A-A01-R02-S02-B01", Aisle: "A01", Rack: "R02", Shelf: "S02", Bin: "B01", MaxWeightKg: 600, MaxVolumeCbm: 1.5, CurrentWeightKg: 0, CurrentVolumeCbm: 0, IsOccupied: false, Status: "available", VelocityCategory: "C", PickingSequence: 6},

		// Aisle A02, Rack R01
		{WarehouseID: 1, ZoneID: zones[0].ID, Code: "WH1-ZONE-DRY-A-A02-R01-S01-B01", Aisle: "A02", Rack: "R01", Shelf: "S01", Bin: "B01", MaxWeightKg: 500, MaxVolumeCbm: 1.2, CurrentWeightKg: 200, CurrentVolumeCbm: 0.5, IsOccupied: true, Status: "occupied", VelocityCategory: "A", PickingSequence: 7},
		{WarehouseID: 1, ZoneID: zones[0].ID, Code: "WH1-ZONE-DRY-A-A02-R01-S02-B01", Aisle: "A02", Rack: "R01", Shelf: "S02", Bin: "B01", MaxWeightKg: 500, MaxVolumeCbm: 1.2, CurrentWeightKg: 0, CurrentVolumeCbm: 0, IsOccupied: false, Status: "available", VelocityCategory: "B", PickingSequence: 8},

		// Zone Cold C
		{WarehouseID: 1, ZoneID: zones[2].ID, Code: "WH1-ZONE-COLD-C-C01-R01-S01-B01", Aisle: "C01", Rack: "R01", Shelf: "S01", Bin: "B01", MaxWeightKg: 400, MaxVolumeCbm: 1.0, CurrentWeightKg: 50, CurrentVolumeCbm: 0.15, IsOccupied: true, Status: "occupied", VelocityCategory: "A", PickingSequence: 10},
		{WarehouseID: 1, ZoneID: zones[2].ID, Code: "WH1-ZONE-COLD-C-C01-R01-S02-B01", Aisle: "C01", Rack: "R01", Shelf: "S02", Bin: "B01", MaxWeightKg: 400, MaxVolumeCbm: 1.0, CurrentWeightKg: 0, CurrentVolumeCbm: 0, IsOccupied: false, Status: "available", VelocityCategory: "B", PickingSequence: 11},
	}
	for i := range bins {
		db.Create(&bins[i])
	}

	// 3. Putaway Rules
	prodID1 := uint(1)
	prodID2 := uint(2)
	rules := []WMSPutawayRule{
		{
			WarehouseID:  1,
			ProductID:    &prodID1,
			TargetZoneID: zones[0].ID,
			Strategy:     "fastest_access",
			Priority:     1,
			Active:       true,
			Description:  "Tempatkan produk fast-moving langsung di Zona Kering A dekat docking keluar",
		},
		{
			WarehouseID:  1,
			ProductID:    &prodID2,
			TargetZoneID: zones[1].ID,
			Strategy:     "least_loaded",
			Priority:     2,
			Active:       true,
			Description:  "Tempatkan bahan baku industri di Zona Kering B (Bulk)",
		},
		{
			WarehouseID:  1,
			TargetZoneID: zones[0].ID,
			Strategy:     "fastest_access",
			Priority:     10,
			Active:       true,
			Description:  "Aturan bawaan gudang: prioritaskan lorong utama zona A",
		},
	}
	for i := range rules {
		db.Create(&rules[i])
	}

	// 4. Initial Inventory Slots in Bins
	now := time.Now()
	slots := []WMSInventorySlot{
		{
			BinLocationID:    bins[0].ID,
			ProductID:        1,
			LotNumber:        "LOT-2026-A01",
			Quantity:         120,
			WeightKg:         120,
			VolumeCbm:        0.35,
			LastStockCountAt: &now,
		},
		{
			BinLocationID:    bins[1].ID,
			ProductID:        2,
			LotNumber:        "LOT-2026-B02",
			Quantity:         85,
			WeightKg:         85,
			VolumeCbm:        0.22,
			LastStockCountAt: &now,
		},
	}
	for i := range slots {
		db.Create(&slots[i])
	}

	// 5. Sample Wave Picking
	started := now.Add(-1 * time.Hour)
	wave1 := WMSWavePicking{
		WaveNumber:         "WAVE/2026/00001",
		Name:               "Wave Pengiriman Ekspedisi Siang (Batch 1)",
		WarehouseID:        1,
		Status:             "in_progress",
		AssignedPickerName: "Agus Pratama (Staf Gudang)",
		TotalPickings:      2,
		TotalLines:         2,
		PickedLines:        1,
		StartedAt:          &started,
		Notes:              "Prioritas pengiriman sebelum cut-off kurir jam 14:00",
	}
	db.Create(&wave1)

	waveLines := []WMSWavePickingLine{
		{
			WaveID:            wave1.ID,
			PickingID:         1,
			PickingName:       "WH/OUT/00001",
			StockMoveID:       1,
			ProductID:         1,
			ProductName:       "Kursi Ergonomis Kantor",
			ProductSku:        "FURN-CHR-001",
			BinLocationID:     bins[0].ID,
			BinLocationCode:   bins[0].Code,
			PickingSequence:   1,
			QuantityRequested: 10,
			QuantityPicked:    10,
			Status:            "picked",
			PickedAt:          &now,
		},
		{
			WaveID:            wave1.ID,
			PickingID:         2,
			PickingName:       "WH/OUT/00002",
			StockMoveID:       2,
			ProductID:         2,
			ProductName:       "Meja Kerja Kayu Solid",
			ProductSku:        "FURN-DSK-002",
			BinLocationID:     bins[1].ID,
			BinLocationCode:   bins[1].Code,
			PickingSequence:   2,
			QuantityRequested: 5,
			QuantityPicked:    0,
			Status:            "pending",
		},
	}
	for i := range waveLines {
		db.Create(&waveLines[i])
	}

	log.Println("✅ Enterprise WMS Initial Seeds Complete.")
	return nil
}
