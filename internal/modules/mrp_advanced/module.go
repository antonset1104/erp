package mrp_advanced

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"erp-golang/internal/core/ai"
	"erp-golang/internal/modules/chatter"
	"erp-golang/internal/modules/mrp"
	"erp-golang/internal/modules/product"
)

type Module struct {
	repo       Repository
	service    Service
	handler    *Handler
	db         *gorm.DB
	mrpSvc     *mrp.Service
	chatterSvc *chatter.Service
	aiClient   *ai.Client
}

func NewModule(db *gorm.DB, mrpSvc *mrp.Service, chatterSvc *chatter.Service, aiClient *ai.Client) *Module {
	repo := NewRepository(db)
	service := NewService(repo, db, mrpSvc, chatterSvc, aiClient)
	handler := NewHandler(service)

	return &Module{
		repo:       repo,
		service:    service,
		handler:    handler,
		db:         db,
		mrpSvc:     mrpSvc,
		chatterSvc: chatterSvc,
		aiClient:   aiClient,
	}
}

func (m *Module) Name() string {
	return "mrp_advanced"
}

func (m *Module) AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&WorkCenter{},
		&Routing{},
		&RoutingOperation{},
		&WorkOrder{},
	)
}

func (m *Module) Priority() int {
	return 40
}

func (m *Module) RegisterRoutes(api *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	m.handler.RegisterRoutes(api, authMiddleware)
}

func (m *Module) Seed(db *gorm.DB) error {
	if err := m.AutoMigrate(db); err != nil {
		return err
	}

	m.seedDefaultWorkCenters(db)
	m.seedDefaultRoutingsAndWorkOrders(db)
	return nil
}

func (m *Module) Init(r *gin.Engine, db *gorm.DB, authMiddleware gin.HandlerFunc) error {
	m.db = db
	apiGroup := r.Group("/api/v1")
	m.RegisterRoutes(apiGroup, authMiddleware)

	// Seed Master Data
	return m.Seed(db)
}

func (m *Module) Service() Service {
	return m.service
}

func (m *Module) Handler() *Handler {
	return m.handler
}

func (m *Module) seedDefaultWorkCenters(db *gorm.DB) {
	var count int64
	db.Model(&WorkCenter{}).Count(&count)
	if count > 0 {
		return
	}

	log.Println("🌱 Seeding Advanced MRP II Work Centers & Machine Capacities...")

	workCenters := []WorkCenter{
		{
			Code:               "WC/2026/001",
			Name:               "Mesin CNC Milling 5-Axis (High Precision)",
			WorkCenterType:     "machining",
			CapacityPerHour:    12.0,
			WorkingHoursPerDay: 8.0,
			TotalOEEPercentage: 88.5,
			CostPerHour:        250000.0,
			MachinePowerKW:     22.0,
			Status:             "available",
			Notes:              "Pusat kerja permesinan presisi tinggi komponen baja & aluminium otomotif.",
		},
		{
			Code:               "WC/2026/002",
			Name:               "Stasiun Welding & Robot Fabrikasi Rangka",
			WorkCenterType:     "welding",
			CapacityPerHour:    15.0,
			WorkingHoursPerDay: 8.0,
			TotalOEEPercentage: 85.0,
			CostPerHour:        180000.0,
			MachinePowerKW:     18.5,
			Status:             "available",
			Notes:              "Robot pengelasan otomatis argon & CO2 dengan sensor ultrasonic seam tracker.",
		},
		{
			Code:               "WC/2026/003",
			Name:               "Lini Perakitan Utama (Final Assembly Line)",
			WorkCenterType:     "assembly",
			CapacityPerHour:    20.0,
			WorkingHoursPerDay: 8.0,
			TotalOEEPercentage: 92.0,
			CostPerHour:        120000.0,
			MachinePowerKW:     7.5,
			Status:             "available",
			Notes:              "Conveyor line perakitan akhir motor listrik, wiring harness & penutup bodi.",
		},
		{
			Code:               "WC/2026/004",
			Name:               "Stasiun Inspeksi & Quality Testing (End of Line)",
			WorkCenterType:     "qc",
			CapacityPerHour:    30.0,
			WorkingHoursPerDay: 8.0,
			TotalOEEPercentage: 95.0,
			CostPerHour:        100000.0,
			MachinePowerKW:     5.0,
			Status:             "available",
			Notes:              "Stasiun uji dyno kelistrikan, vibrasi, dan toleransi dimensi standar ISO 9001.",
		},
	}

	for _, wc := range workCenters {
		wc.CompanyID = 1
		wc.CreateUID = 1
		wc.Active = true
		db.Create(&wc)
	}

	log.Println("✅ Advanced MRP II Work Centers seeded successfully.")
}

func (m *Module) seedDefaultRoutingsAndWorkOrders(db *gorm.DB) {
	var count int64
	db.Model(&Routing{}).Count(&count)
	if count > 0 {
		return
	}

	// Find any finished product
	var prod product.Product
	db.Where("type = 'product'").First(&prod)
	prodID := prod.ID
	if prodID == 0 {
		prodID = 1
	}

	// Find Work Centers
	var wc1, wc2, wc3, wc4 WorkCenter
	db.Where("code = 'WC/2026/001'").First(&wc1)
	db.Where("code = 'WC/2026/002'").First(&wc2)
	db.Where("code = 'WC/2026/003'").First(&wc3)
	db.Where("code = 'WC/2026/004'").First(&wc4)

	if wc1.ID == 0 || wc2.ID == 0 || wc3.ID == 0 || wc4.ID == 0 {
		return
	}

	routing := Routing{
		RoutingCode: "ROUT/2026/00001",
		Name:        "Rute Standar Fabrikasi & Perakitan Mesin Industri",
		ProductID:   prodID,
		Notes:       "Standar alur proses 4 tahap dari pembentukan komponen hingga pengujian QC akhir.",
		Operations: []RoutingOperation{
			{
				Sequence:                10,
				Name:                    "Milling & Pembentukan Komponen Presisi",
				WorkCenterID:            wc1.ID,
				SetupTimeMinutes:        20.0,
				CycleTimePerUnitMinutes: 6.0,
				TeardownTimeMinutes:     10.0,
				BatchSize:               50.0,
				Description:             "Pembubutan dan milling CNC blok silinder.",
			},
			{
				Sequence:                20,
				Name:                    "Pengelasan & Fabrikasi Rangka",
				WorkCenterID:            wc2.ID,
				SetupTimeMinutes:        15.0,
				CycleTimePerUnitMinutes: 8.0,
				TeardownTimeMinutes:     10.0,
				BatchSize:               50.0,
				Description:             "Penyatuan rangka struktural dengan robot welding.",
			},
			{
				Sequence:                30,
				Name:                    "Perakitan Akhir & Integrasi Wiring",
				WorkCenterID:            wc3.ID,
				SetupTimeMinutes:        10.0,
				CycleTimePerUnitMinutes: 12.0,
				TeardownTimeMinutes:     5.0,
				BatchSize:               25.0,
				Description:             "Pemasangan kabel motor dan baut torsi sesuai spesifikasi.",
			},
			{
				Sequence:                40,
				Name:                    "Pengujian Dyno & Inspeksi Kualitas Akhir",
				WorkCenterID:            wc4.ID,
				SetupTimeMinutes:        5.0,
				CycleTimePerUnitMinutes: 4.0,
				TeardownTimeMinutes:     5.0,
				BatchSize:               50.0,
				Description:             "Uji performa dyno dan pemeriksaan toleransi.",
			},
		},
	}
	routing.CompanyID = 1
	routing.CreateUID = 1
	routing.Active = true

	for i := range routing.Operations {
		routing.Operations[i].CompanyID = 1
		routing.Operations[i].CreateUID = 1
		routing.Operations[i].Active = true
	}

	if err := db.Create(&routing).Error; err != nil {
		log.Printf("Failed to seed routing: %v", err)
		return
	}

	// Seed Sample Work Orders for an existing MO or mock MO
	var mo mrp.MrpProduction
	db.First(&mo)
	moID := mo.ID
	moName := mo.Name
	if moID == 0 {
		moID = 1
		moName = "MO/2026/00001"
	}

	now := time.Now()
	t1Start := now.Add(-2 * time.Hour)
	t1End := now.Add(-1 * time.Hour)
	t2Start := now.Add(-1 * time.Hour)
	t2End := now.Add(30 * time.Minute)
	t3Start := now.Add(30 * time.Minute)
	t3End := now.Add(3 * time.Hour)
	t4Start := now.Add(3 * time.Hour)
	t4End := now.Add(4 * time.Hour)

	sampleWOs := []WorkOrder{
		{
			WorkOrderCode:          "WO/2026/00001",
			ProductionOrderID:      moID,
			ProductionOrderName:    moName,
			RoutingOperationID:     routing.Operations[0].ID,
			WorkCenterID:           wc1.ID,
			Sequence:               10,
			OperationName:          routing.Operations[0].Name,
			PlannedStartTime:       &t1Start,
			PlannedEndTime:         &t1End,
			ActualStartTime:        &t1Start,
			ActualEndTime:          &t1End,
			PlannedDurationMinutes: 60.0,
			ActualDurationMinutes:  60.0,
			QtyPlanned:             10.0,
			QtyProduced:            10.0,
			QtyScrap:               0.0,
			MachineHours:           1.0,
			TotalMachineCost:       250000.0,
			State:                  "done",
			Notes:                  "Selesai sesuai target tanpa scrap.",
		},
		{
			WorkOrderCode:          "WO/2026/00002",
			ProductionOrderID:      moID,
			ProductionOrderName:    moName,
			RoutingOperationID:     routing.Operations[1].ID,
			WorkCenterID:           wc2.ID,
			Sequence:               20,
			OperationName:          routing.Operations[1].Name,
			PlannedStartTime:       &t2Start,
			PlannedEndTime:         &t2End,
			ActualStartTime:        &t2Start,
			PlannedDurationMinutes: 90.0,
			QtyPlanned:             10.0,
			QtyProduced:            0.0,
			State:                  "in_progress",
			Notes:                  "Sedang dalam proses pengelasan robotik tahap 2.",
		},
		{
			WorkOrderCode:          "WO/2026/00003",
			ProductionOrderID:      moID,
			ProductionOrderName:    moName,
			RoutingOperationID:     routing.Operations[2].ID,
			WorkCenterID:           wc3.ID,
			Sequence:               30,
			OperationName:          routing.Operations[2].Name,
			PlannedStartTime:       &t3Start,
			PlannedEndTime:         &t3End,
			PlannedDurationMinutes: 150.0,
			QtyPlanned:             10.0,
			QtyProduced:            0.0,
			State:                  "ready",
			Notes:                  "Komponen siap diserahterimakan dari stasiun welding.",
		},
		{
			WorkOrderCode:          "WO/2026/00004",
			ProductionOrderID:      moID,
			ProductionOrderName:    moName,
			RoutingOperationID:     routing.Operations[3].ID,
			WorkCenterID:           wc4.ID,
			Sequence:               40,
			OperationName:          routing.Operations[3].Name,
			PlannedStartTime:       &t4Start,
			PlannedEndTime:         &t4End,
			PlannedDurationMinutes: 60.0,
			QtyPlanned:             10.0,
			QtyProduced:            0.0,
			State:                  "pending",
			Notes:                  "Menunggu selesainya perakitan di Work Center 3.",
		},
	}

	for _, wo := range sampleWOs {
		wo.CompanyID = 1
		wo.CreateUID = 1
		wo.Active = true
		db.Create(&wo)
	}

	log.Println("✅ Advanced MRP II Routings and Sample Work Orders seeded successfully.")
}
