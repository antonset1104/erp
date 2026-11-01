package maintenance

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Module struct {
	db      *gorm.DB
	svc     *Service
	handler *Handler
}

func NewModule(db *gorm.DB) *Module {
	svc := NewService(db)
	handler := NewHandler(svc)
	return &Module{
		db:      db,
		svc:     svc,
		handler: handler,
	}
}

func (m *Module) Service() *Service {
	return m.svc
}

func (m *Module) Name() string {
	return "maintenance"
}

func (m *Module) AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&MaintenanceEquipmentCategory{},
		&MaintenanceEquipment{},
		&MaintenanceRequest{},
	)
}

func (m *Module) RegisterRoutes(api *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	group := api.Group("/maintenance")
	group.Use(authMiddleware)
	{
		// Categories
		group.GET("/categories", m.handler.ListCategories)
		group.POST("/categories", m.handler.CreateCategory)

		// Equipments
		group.GET("/equipments", m.handler.ListEquipments)
		group.GET("/equipments/:id", m.handler.GetEquipment)
		group.POST("/equipments", m.handler.CreateEquipment)

		// Requests
		group.GET("/requests", m.handler.ListRequests)
		group.GET("/requests/:id", m.handler.GetRequest)
		group.POST("/requests", m.handler.CreateRequest)
		group.PUT("/requests/:id/stage", m.handler.UpdateRequestStage)
		group.PUT("/requests/:id/assign", m.handler.AssignRequest)
		group.PUT("/requests/:id/complete", m.handler.CompleteRequest)
		group.POST("/requests/auto-preventive", m.handler.AutoSchedulePreventive)
	}
}

func (m *Module) Seed(db *gorm.DB) error {
	var catCount int64
	db.Model(&MaintenanceEquipmentCategory{}).Count(&catCount)
	if catCount == 0 {
		userID1 := uint(1) // Admin user

		cat1 := MaintenanceEquipmentCategory{
			Name:             "Mesin Pabrik & Lini Produksi",
			TechnicianUserID: &userID1,
			Color:            "#ef4444",
			Note:             "Mesin manufaktur perakitan dan perkakas kerja workshop",
		}
		cat1.CompanyID = 1
		cat1.CreateUID = 1
		cat1.WriteUID = 1
		db.Create(&cat1)

		cat2 := MaintenanceEquipmentCategory{
			Name:             "Infrastruktur IT & Server",
			TechnicianUserID: &userID1,
			Color:            "#3b82f6",
			Note:             "Perangkat keras server, switch jaringan, dan workstation kantor",
		}
		cat2.CompanyID = 1
		cat2.CreateUID = 1
		cat2.WriteUID = 1
		db.Create(&cat2)

		cat3 := MaintenanceEquipmentCategory{
			Name:             "Armada Kendaraan & Logistik",
			TechnicianUserID: &userID1,
			Color:            "#10b981",
			Note:             "Forklift gudang, mobil box operasional, dan truk angkut",
		}
		cat3.CompanyID = 1
		cat3.CreateUID = 1
		cat3.WriteUID = 1
		db.Create(&cat3)

		var eqCount int64
		db.Model(&MaintenanceEquipment{}).Count(&eqCount)
		if eqCount == 0 {
			partnerID1 := uint(1) // PT Maju Bersama
			deptID1 := uint(1)

			eq1 := MaintenanceEquipment{
				Name:                  "Mesin Solder & Perakitan SMT Otomatis",
				EquipmentNumber:       "EQP/2026/00001",
				SerialNumber:          "SN-SMT-8820",
				Model:                 "Yamaha YSM20R Modular",
				CategoryID:            &cat1.ID,
				PartnerID:             &partnerID1,
				DepartmentID:          &deptID1,
				TechnicianUserID:      &userID1,
				Location:              "Gudang Utama - Lantai 2 Workshop Produksi",
				Cost:                  250000000,
				WarrantyDate:          "2028-06-30",
				MaintenancePeriodDays: 30,
				NextActionDate:        time.Now().AddDate(0, 0, 30).Format("2006-01-02"),
				Status:                "operational",
				Note:                  "Mesin utama perakitan komponen elektronika",
			}
			eq1.CompanyID = 1
			eq1.CreateUID = 1
			eq1.WriteUID = 1
			db.Create(&eq1)

			assetID1 := uint(1) // Dell PowerEdge Fixed Asset
			eq2 := MaintenanceEquipment{
				Name:                  "Dell PowerEdge R750 Enterprise Server",
				EquipmentNumber:       "EQP/2026/00002",
				SerialNumber:          "SN-DELL-R750-99",
				Model:                 "Dell PowerEdge R750 2U",
				CategoryID:            &cat2.ID,
				PartnerID:             &partnerID1,
				AssetID:               &assetID1,
				DepartmentID:          &deptID1,
				TechnicianUserID:      &userID1,
				Location:              "Data Center Server Room Rack 01",
				Cost:                  48000000,
				WarrantyDate:          "2029-10-01",
				MaintenancePeriodDays: 60,
				NextActionDate:        time.Now().AddDate(0, 0, 60).Format("2006-01-02"),
				Status:                "operational",
				Note:                  "Server database pusat dan host ERP cloud",
			}
			eq2.CompanyID = 1
			eq2.CreateUID = 1
			eq2.WriteUID = 1
			db.Create(&eq2)

			// Overdue equipment for testing auto preventive scheduler
			overdueDate := time.Now().AddDate(0, 0, -2).Format("2006-01-02")
			eq3 := MaintenanceEquipment{
				Name:                  "Forklift Elektrik Toyota 3 Ton",
				EquipmentNumber:       "EQP/2026/00003",
				SerialNumber:          "SN-FL-TYT-30",
				Model:                 "Toyota 8FBN30",
				CategoryID:            &cat3.ID,
				PartnerID:             &partnerID1,
				DepartmentID:          &deptID1,
				TechnicianUserID:      &userID1,
				Location:              "Area Muat Logistik Gudang Jakarta",
				Cost:                  185000000,
				WarrantyDate:          "2027-12-31",
				MaintenancePeriodDays: 45,
				NextActionDate:        overdueDate,
				Status:                "operational",
				Note:                  "Forklift pemindahan pallet barang masuk dan keluar",
			}
			eq3.CompanyID = 1
			eq3.CreateUID = 1
			eq3.WriteUID = 1
			db.Create(&eq3)

			// Sample requests
			today := time.Now().Format("2006-01-02")
			req1 := MaintenanceRequest{
				RequestNumber:   "REQ/MAINT/2026/00001",
				Name:            "Pembersihan Nozzle SMT & Kalibrasi Sensor Feeder",
				EquipmentID:     eq1.ID,
				MaintenanceType: "preventive",
				Priority:        "1",
				Stage:           "repaired",
				RequestDate:     today,
				ScheduleDate:    today,
				CloseDate:       today + " 11:30:00",
				DurationHours:   2.0,
				UserID:          &userID1,
				Description:     "Pembersihan berkala lensa optik dan sensor laser alignment.",
				Resolution:      "Nozzle dibersihkan dengan cairan khusus dan uji presisi penempatan komponen lolos 100%.",
			}
			req1.CompanyID = 1
			req1.CreateUID = 1
			req1.WriteUID = 1
			db.Create(&req1)

			req2 := MaintenanceRequest{
				RequestNumber:   "REQ/MAINT/2026/00002",
				Name:            "Pemeriksaan Kebocoran Seal Silinder Mast Forklift",
				EquipmentID:     eq3.ID,
				MaintenanceType: "corrective",
				Priority:        "3",
				Stage:           "in_progress",
				RequestDate:     today,
				ScheduleDate:    today,
				DurationHours:   1.5,
				UserID:          &userID1,
				Description:     "Tekanan hidrolik garpu angkat sedikit menurun saat mengangkat beban palet di atas 2 ton.",
			}
			req2.CompanyID = 1
			req2.CreateUID = 1
			req2.WriteUID = 1
			db.Create(&req2)
		}

		log.Println("[Maintenance] Seeded default Equipment Categories, Assets & Maintenance Requests")
	}
	return nil
}
