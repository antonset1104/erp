package quality

import (
	"log"

	"erp-golang/internal/modules/chatter"
	"erp-golang/internal/modules/mrp"
	"erp-golang/internal/modules/stock"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Module struct {
	db         *gorm.DB
	svc        *Service
	handler    *Handler
	stockSvc   *stock.Service
	mrpSvc     *mrp.Service
	chatterSvc *chatter.Service
}

func NewModule(db *gorm.DB, stockSvc *stock.Service, mrpSvc *mrp.Service, chatterSvc *chatter.Service) *Module {
	svc := NewService(db, stockSvc, mrpSvc, chatterSvc)
	handler := NewHandler(svc)
	return &Module{
		db:         db,
		svc:        svc,
		handler:    handler,
		stockSvc:   stockSvc,
		mrpSvc:     mrpSvc,
		chatterSvc: chatterSvc,
	}
}

func (m *Module) Service() *Service {
	return m.svc
}

func (m *Module) Name() string {
	return "quality"
}

func (m *Module) AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&QualityPoint{},
		&QualityCheck{},
		&QualityAlert{},
	)
}

func (m *Module) RegisterRoutes(api *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	group := api.Group("/quality")
	group.Use(authMiddleware)
	{
		// Points (Rules)
		group.GET("/points", m.handler.ListPoints)
		group.POST("/points", m.handler.CreatePoint)
		group.GET("/points/:id", m.handler.GetPoint)
		group.PUT("/points/:id", m.handler.UpdatePoint)
		group.DELETE("/points/:id", m.handler.DeletePoint)

		// Checks (Inspections)
		group.GET("/checks", m.handler.ListChecks)
		group.POST("/checks", m.handler.CreateCheck)
		group.GET("/checks/:id", m.handler.GetCheck)
		group.POST("/checks/:id/pass", m.handler.ActionPassCheck)
		group.POST("/checks/:id/fail", m.handler.ActionFailCheck)
		group.POST("/checks/generate/picking/:picking_id", m.handler.GenerateChecksForPicking)
		group.POST("/checks/generate/production/:production_id", m.handler.GenerateChecksForProduction)

		// Alerts (Non-Conformance / RCA / CAPA)
		group.GET("/alerts", m.handler.ListAlerts)
		group.POST("/alerts", m.handler.CreateAlert)
		group.GET("/alerts/:id", m.handler.GetAlert)
		group.PUT("/alerts/:id/stage", m.handler.UpdateAlertStage)

		// Analytics
		group.GET("/stats", m.handler.GetQualityStats)
	}
}

func (m *Module) Seed(db *gorm.DB) error {
	var count int64
	db.Model(&QualityPoint{}).Count(&count)
	if count > 0 {
		return nil
	}

	points := []QualityPoint{
		{
			Title:         "Inspeksi Visual & Segel Kemasan Komponen Masuk",
			Code:          "QP/WH-IN/01",
			OperationType: "incoming",
			TestType:      "passfail",
			Instructions:  "Periksa keutuhan segel antistatis, label barcode vendor, dan pastikan tidak ada cacat fisik atau penyok pada kemasan boks.",
			Active:        true,
		},
		{
			Title:         "Pengukuran Voltase Adaptor Daya 12V",
			Code:          "QP/WH-IN/02",
			OperationType: "incoming",
			TestType:      "measure",
			Norm:          12.0,
			ToleranceMin:  11.5,
			ToleranceMax:  12.5,
			NormUnit:      "V",
			Instructions:  "Ukur tegangan output DC menggunakan multimeter digital terkalibrasi. Batas aman: 11.50V - 12.50V.",
			Active:        true,
		},
		{
			Title:         "Quality Assurance Akhir Perakitan PC Desktop",
			Code:          "QP/MO/01",
			OperationType: "mrp_operation",
			TestType:      "passfail",
			Instructions:  "Nyalakan sistem, verifikasi BIOS POST, pastikan semua fan berputar tanpa getaran abnormal, dan kabel terikat rapi (cable management).",
			Active:        true,
		},
		{
			Title:         "Pengujian Temperatur CPU Full Load Stress Test",
			Code:          "QP/MO/02",
			OperationType: "mrp_operation",
			TestType:      "measure",
			Norm:          65.0,
			ToleranceMin:  30.0,
			ToleranceMax:  75.0,
			NormUnit:      "°C",
			Instructions:  "Jalankan stress test selama 5 menit. Suhu puncak tidak boleh melebihi 75.0°C.",
			Active:        true,
		},
	}

	for i := range points {
		points[i].CompanyID = 1
		points[i].CreateUID = 1
		points[i].WriteUID = 1
		if err := db.Create(&points[i]).Error; err != nil {
			log.Printf("[Quality Seed] Error seeding point %s: %v", points[i].Code, err)
		}
	}

	log.Println("[Quality] Successfully seeded default Quality Control Points.")
	return nil
}
