package approval

import (
	"log"

	"erp-golang/internal/modules/chatter"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Module struct {
	db         *gorm.DB
	svc        *Service
	handler    *Handler
	chatterSvc *chatter.Service
}

func NewModule(db *gorm.DB, chatterSvc *chatter.Service) *Module {
	svc := NewService(db, chatterSvc)
	handler := NewHandler(svc)
	return &Module{
		db:         db,
		svc:        svc,
		handler:    handler,
		chatterSvc: chatterSvc,
	}
}

func (m *Module) Service() *Service {
	return m.svc
}

func (m *Module) Name() string {
	return "approval"
}

func (m *Module) AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&ApprovalCategory{},
		&ApprovalRequest{},
		&ApprovalApprover{},
	)
}

func (m *Module) RegisterRoutes(api *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	group := api.Group("/approvals")
	group.Use(authMiddleware)
	{
		// Categories
		group.GET("/categories", m.handler.ListCategories)
		group.POST("/categories", m.handler.CreateCategory)

		// Requests
		group.GET("/requests", m.handler.ListRequests)
		group.POST("/requests", m.handler.CreateRequest)
		group.GET("/requests/:id", m.handler.GetRequest)
		group.POST("/requests/:id/submit", m.handler.SubmitRequest)
		group.POST("/requests/:id/approve", m.handler.ApproveRequest)
		group.POST("/requests/:id/refuse", m.handler.RefuseRequest)
		group.POST("/requests/:id/cancel", m.handler.CancelRequest)

		// KPI Stats
		group.GET("/stats", m.handler.GetStats)
	}
}

func (m *Module) Seed(db *gorm.DB) error {
	var count int64
	db.Model(&ApprovalCategory{}).Count(&count)
	if count > 0 {
		return nil
	}

	categories := []ApprovalCategory{
		{
			Name:                 "Pengadaan Capex & Aset Tetap",
			Description:          "Alur otorisasi multi-tier untuk pembelian server IT, mesin produksi, dan armada logistik baru dengan nilai di atas Rp 10.000.000.",
			SequenceCode:         "CAPEX",
			ApprovalType:         "capex",
			HasAmount:            true,
			MinAmount:            10000000,
			HasDate:              true,
			MinApprovalsRequired: 2,
			IsManagerApprover:    true,
			Active:               true,
		},
		{
			Name:                 "Diskon Penjualan Khusus (> 10%)",
			Description:          "Otorisasi persetujuan potongan harga komersial untuk penawaran quotation Sales Order yang melebihi batas diskon standar.",
			SequenceCode:         "DISC",
			ApprovalType:         "discount",
			HasAmount:            true,
			MinAmount:            5000000,
			HasDate:              false,
			MinApprovalsRequired: 1,
			IsManagerApprover:    true,
			Active:               true,
		},
		{
			Name:                 "Perjalanan Dinas VIP & Luar Negeri",
			Description:          "Persetujuan perjalanan dinas eksekutif dan akomodasi internasional yang memerlukan review Kepala Departemen dan Direktur Keuangan.",
			SequenceCode:         "TRIP",
			ApprovalType:         "payment",
			HasAmount:            true,
			MinAmount:            2500000,
			HasDate:              true,
			MinApprovalsRequired: 2,
			IsManagerApprover:    true,
			Active:               true,
		},
		{
			Name:                 "Persetujuan Kontrak & Perjanjian Legal",
			Description:          "Tinjauan kepatuhan hukum dan klausul kontrak vendor/klien skala besar sebelum penandatanganan resmi.",
			SequenceCode:         "LEGAL",
			ApprovalType:         "general",
			HasAmount:            false,
			MinApprovalsRequired: 2,
			IsManagerApprover:    false,
			Active:               true,
		},
	}

	for i := range categories {
		categories[i].CompanyID = 1
		categories[i].CreateUID = 1
		categories[i].WriteUID = 1
		if err := db.Create(&categories[i]).Error; err != nil {
			log.Printf("[Approval Seed] Error seeding category %s: %v", categories[i].Name, err)
		}
	}

	log.Println("[Approval] Successfully seeded default corporate Approval Categories.")
	return nil
}
