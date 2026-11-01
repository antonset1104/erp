package crm

import (
	"log"

	"erp-golang/internal/modules/sale"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Module struct {
	db      *gorm.DB
	saleSvc *sale.Service
	svc     *Service
	handler *Handler
}

func NewModule(db *gorm.DB, saleSvc *sale.Service) *Module {
	svc := NewService(db, saleSvc)
	handler := NewHandler(svc)
	return &Module{
		db:      db,
		saleSvc: saleSvc,
		svc:     svc,
		handler: handler,
	}
}

func (m *Module) Service() *Service {
	return m.svc
}

func (m *Module) Name() string {
	return "crm"
}

func (m *Module) AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&CrmLead{})
}

func (m *Module) RegisterRoutes(api *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	group := api.Group("/crm")
	group.Use(authMiddleware)
	{
		group.GET("/leads", m.handler.ListLeads)
		group.POST("/leads", m.handler.CreateLead)
		group.GET("/leads/:id", m.handler.GetLead)
		group.PUT("/leads/:id/stage", m.handler.UpdateStage)
		group.POST("/leads/:id/convert", m.handler.ConvertToQuotation)
	}
}

func (m *Module) Seed(db *gorm.DB) error {
	var count int64
	db.Model(&CrmLead{}).Count(&count)
	if count == 0 {
		partnerID1 := uint(1) // PT Maju Bersama Teknologi
		partnerID2 := uint(2) // CV Berkah Sejahtera

		leads := []CrmLead{
			{
				Name:            "Pengadaan 10 Unit Laptop Executive",
				PartnerID:       &partnerID1,
				PartnerName:     "PT Maju Bersama Teknologi",
				ContactName:     "Budi Santoso",
				Email:           "budi@majubersama.com",
				Phone:           "+62 812 3456 7890",
				ExpectedRevenue: 190000000,
				Probability:     70,
				Stage:           "proposition",
				Priority:        "2",
				Description:     "Kebutuhan laptop untuk tim engineering baru Q4",
			},
			{
				Name:            "Upgrade Perangkat Ergonomic & Aksesoris",
				PartnerID:       &partnerID2,
				PartnerName:     "CV Berkah Sejahtera",
				ContactName:     "Dewi Lestari",
				Email:           "dewi@berkahsejahtera.com",
				Phone:           "+62 813 9876 5432",
				ExpectedRevenue: 35000000,
				Probability:     35,
				Stage:           "qualified",
				Priority:        "1",
				Description:     "Peluang perlengkapan ruang kerja kantor cabang",
			},
			{
				Name:            "Tender Pengadaan Kertas dan ATK Tahunan",
				PartnerName:     "PT Global Indo Perkasa",
				ContactName:     "Agus Salim",
				Email:           "procurement@globalindo.com",
				Phone:           "+62 811 2233 4455",
				ExpectedRevenue: 85000000,
				Probability:     10,
				Stage:           "new",
				Priority:        "3",
				Description:     "Lead masuk via website portal contact form",
			},
		}

		for i := range leads {
			leads[i].CompanyID = 1
			db.Create(&leads[i])
		}
		log.Println("[CRM] Seeded sample CRM Leads & Opportunities Pipeline")
	}

	return nil
}
