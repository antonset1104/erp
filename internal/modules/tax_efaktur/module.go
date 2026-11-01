package tax_efaktur

import (
	"log"
	"time"

	"erp-golang/internal/core/ai"
	"erp-golang/internal/modules/account"
	"erp-golang/internal/modules/chatter"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Module struct {
	db       *gorm.DB
	service  *Service
	handler  *Handler
	aiClient *ai.Client
}

func NewModule(db *gorm.DB, accountSvc *account.Service, chatterSvc *chatter.Service, aiClient *ai.Client) *Module {
	repo := NewRepository(db)
	service := NewService(db, repo, accountSvc, chatterSvc, aiClient)
	handler := NewHandler(service)

	return &Module{
		db:       db,
		service:  service,
		handler:  handler,
		aiClient: aiClient,
	}
}

func (m *Module) Name() string {
	return "tax_efaktur"
}

func (m *Module) Service() *Service {
	return m.service
}

func (m *Module) RegisterRoutes(api *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	group := api.Group("/tax-efaktur")
	group.Use(authMiddleware)
	{
		// NSFP Quotas
		group.GET("/nsfp", m.handler.ListNSFP)
		group.POST("/nsfp", m.handler.CreateNSFP)

		// Invoices
		group.GET("/invoices", m.handler.ListTaxInvoices)
		group.GET("/invoices/:id", m.handler.GetTaxInvoiceByID)
		group.POST("/invoices/out", m.handler.GenerateOutTaxInvoice)
		group.POST("/invoices/in", m.handler.RecordInTaxInvoice)
		group.PUT("/invoices/:id/state", m.handler.UpdateState)

		// Analytics, SPT & Export
		group.GET("/stats", m.handler.Stats)
		group.GET("/spt-summary", m.handler.SPTPeriodSummary)
		group.GET("/export-csv", m.handler.ExportCSV)
		group.GET("/ai-audit", m.handler.TaxComplianceAIAudit)
	}
}

func (m *Module) AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&TaxNSFP{},
		&TaxInvoice{},
		&TaxInvoiceLine{},
	)
}

func (m *Module) Seed(db *gorm.DB) error {
	var count int64
	db.Model(&TaxNSFP{}).Count(&count)
	if count > 0 {
		return nil
	}

	year := time.Now().Year()

	// Seed official DJP NSFP Allocation Range (500 quota)
	initialNSFP := TaxNSFP{
		Prefix:         "010",
		Year:           year,
		StartNumber:    "00000001",
		EndNumber:      "00000500",
		CurrentNumber:  "00000001",
		TotalQuota:     500,
		UsedQuota:      0,
		RemainingQuota: 500,
		State:          "active",
		CompanyID:      1,
	}

	if err := db.Create(&initialNSFP).Error; err != nil {
		return err
	}

	log.Println("[Tax e-Faktur] Berhasil seeding jatah NSFP resmi DJP (500 kuota)")
	return nil
}
