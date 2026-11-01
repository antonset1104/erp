package fleet

import (
	"log"

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
	return "fleet"
}

func (m *Module) AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&FleetVehicleModel{},
		&FleetVehicle{},
		&FleetOdometerLog{},
		&FleetServiceLog{},
	)
}

func (m *Module) RegisterRoutes(api *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	group := api.Group("/fleet")
	group.Use(authMiddleware)
	{
		// Models
		group.GET("/models", m.handler.ListModels)
		group.POST("/models", m.handler.CreateModel)

		// Vehicles
		group.GET("/vehicles", m.handler.ListVehicles)
		group.GET("/vehicles/:id", m.handler.GetVehicle)
		group.POST("/vehicles", m.handler.CreateVehicle)

		// Odometer
		group.GET("/odometer", m.handler.ListOdometerLogs)
		group.POST("/odometer", m.handler.LogOdometer)

		// Service Logs
		group.GET("/services", m.handler.ListServiceLogs)
		group.POST("/services", m.handler.CreateServiceLog)
		group.PUT("/services/:id/complete", m.handler.CompleteServiceLog)
	}
}

func (m *Module) Seed(db *gorm.DB) error {
	var modelCount int64
	db.Model(&FleetVehicleModel{}).Count(&modelCount)
	if modelCount == 0 {
		m1 := FleetVehicleModel{
			Brand:       "Mitsubishi",
			ModelName:   "Canter FE 74 HD 125 PS (Box Besi)",
			VehicleType: "truck",
		}
		m1.CompanyID = 1
		m1.CreateUID = 1
		m1.WriteUID = 1
		db.Create(&m1)

		m2 := FleetVehicleModel{
			Brand:       "Daihatsu",
			ModelName:   "Gran Max Blind Van 1.3 D",
			VehicleType: "van",
		}
		m2.CompanyID = 1
		m2.CreateUID = 1
		m2.WriteUID = 1
		db.Create(&m2)

		m3 := FleetVehicleModel{
			Brand:       "Toyota",
			ModelName:   "Hilux 2.4 D-Cab 4x4 MT",
			VehicleType: "truck",
		}
		m3.CompanyID = 1
		m3.CreateUID = 1
		m3.WriteUID = 1
		db.Create(&m3)

		m4 := FleetVehicleModel{
			Brand:       "Honda",
			ModelName:   "Vario 160 CBS (Kurir Dokumen)",
			VehicleType: "motorcycle",
		}
		m4.CompanyID = 1
		m4.CreateUID = 1
		m4.WriteUID = 1
		db.Create(&m4)

		var vehCount int64
		db.Model(&FleetVehicle{}).Count(&vehCount)
		if vehCount == 0 {
			driverID1 := uint(1)
			driverID2 := uint(2)
			driverID3 := uint(3)

			v1 := FleetVehicle{
				LicensePlate:     "B 1024 TKO",
				ModelID:          m1.ID,
				DriverEmployeeID: &driverID1,
				PlanPermitNumber: "STNK-JKT-2029-918",
				AcquisitionDate:  "2024-03-15",
				VinSN:            "MHFE74HDMK00123",
				EngineNumber:     "4D34-99120",
				Location:         "Gudang Utama Jakarta",
				Status:           "active",
				LastOdometer:     42500,
				FuelType:         "diesel",
				Color:            "Kuning / Silver",
				Seats:            3,
				Note:             "Truk pengiriman muatan besar & distribusi antar gudang regional",
			}
			v1.CompanyID = 1
			v1.CreateUID = 1
			v1.WriteUID = 1
			db.Create(&v1)

			v2 := FleetVehicle{
				LicensePlate:     "D 8891 BDG",
				ModelID:          m2.ID,
				DriverEmployeeID: &driverID2,
				PlanPermitNumber: "STNK-BDG-2028-442",
				AcquisitionDate:  "2024-08-20",
				VinSN:            "MHKGV13DDK00889",
				EngineNumber:     "K3-DE-11029",
				Location:         "Gudang Cabang Bandung",
				Status:           "active",
				LastOdometer:     18200,
				FuelType:         "gasoline",
				Color:            "Putih",
				Seats:            2,
				Note:             "Blind van delivery order pesanan pelanggan dalam kota",
			}
			v2.CompanyID = 1
			v2.CreateUID = 1
			v2.WriteUID = 1
			db.Create(&v2)

			v3 := FleetVehicle{
				LicensePlate:     "L 9140 SBY",
				ModelID:          m3.ID,
				DriverEmployeeID: &driverID3,
				PlanPermitNumber: "STNK-SBY-2028-701",
				AcquisitionDate:  "2025-01-10",
				VinSN:            "MROHA24DK99140",
				EngineNumber:     "2GD-FTV-8821",
				Location:         "Hub Regional Surabaya",
				Status:           "active",
				LastOdometer:     31000,
				FuelType:         "diesel",
				Color:            "Hitam Metalik",
				Seats:            5,
				Note:             "Kendaraan operasional inspeksi proyek dan survei logistik Jawa Timur",
			}
			v3.CompanyID = 1
			v3.CreateUID = 1
			v3.WriteUID = 1
			db.Create(&v3)

			// Sample service logs
			partnerID1 := uint(1) // PT Maju Bersama / Rekanan
			s1 := FleetServiceLog{
				ServiceNumber:   "SRV/FLEET/2026/00001",
				VehicleID:       v1.ID,
				Date:            "2026-09-20",
				ServiceType:     "oil_change",
				Cost:            1250000,
				VendorPartnerID: &partnerID1,
				Odometer:        40000,
				Stage:           "done",
				Description:     "Penggantian oli mesin diesel Shell Rimula 15W-40 (8 Liter) & filter oli orisinil",
			}
			s1.CompanyID = 1
			s1.CreateUID = 1
			s1.WriteUID = 1
			db.Create(&s1)

			s2 := FleetServiceLog{
				ServiceNumber:   "SRV/FLEET/2026/00002",
				VehicleID:       v2.ID,
				Date:            "2026-09-28",
				ServiceType:     "periodic_inspection_kir",
				Cost:            650000,
				VendorPartnerID: &partnerID1,
				Odometer:        17500,
				Stage:           "done",
				Description:     "Uji emisi & pengujian berkala buku KIR Dishub Kota Bandung lolos uji",
			}
			s2.CompanyID = 1
			s2.CreateUID = 1
			s2.WriteUID = 1
			db.Create(&s2)
		}

		log.Println("[Fleet] Seeded default Vehicle Models, Delivery Fleets & Service Logs")
	}
	return nil
}
