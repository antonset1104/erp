package landed_cost

import (
	"log"

	"erp-golang/internal/core/ai"
	"erp-golang/internal/modules/account"
	"erp-golang/internal/modules/chatter"
	"erp-golang/internal/modules/stock"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Module struct {
	db       *gorm.DB
	service  *Service
	handler  *Handler
	aiClient *ai.Client
}

func NewModule(db *gorm.DB, accountSvc *account.Service, stockSvc *stock.Service, chatterSvc *chatter.Service, aiClient *ai.Client) *Module {
	repo := NewRepository(db)
	service := NewService(db, repo, accountSvc, chatterSvc, aiClient)
	handler := NewHandler(service, stockSvc)

	return &Module{
		db:       db,
		service:  service,
		handler:  handler,
		aiClient: aiClient,
	}
}

func (m *Module) Name() string {
	return "landed_cost"
}

func (m *Module) Service() *Service {
	return m.service
}

func (m *Module) AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&StockLandedCost{},
		&StockLandedCostLine{},
		&StockValuationAdjustmentLine{},
	)
}

func (m *Module) RegisterRoutes(api *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	group := api.Group("/landed-costs")
	group.Use(authMiddleware)
	{
		group.GET("", m.handler.ListLandedCosts)
		group.POST("", m.handler.CreateLandedCost)
		group.GET("/stats", m.handler.GetStats)
		group.GET("/pickings", m.handler.ListEligiblePickings)
		group.GET("/:id", m.handler.GetLandedCost)
		group.POST("/:id/compute", m.handler.ComputeValuationLines)
		group.POST("/:id/validate", m.handler.ActionValidate)
		group.POST("/:id/cancel", m.handler.ActionCancel)
		group.GET("/:id/ai-audit", m.handler.AILandedCostCopilot)
	}
}

func (m *Module) Seed(db *gorm.DB) error {
	if err := m.AutoMigrate(db); err != nil {
		return err
	}

	// Seed Landed Cost Interim Clearing Account (130090) if not exists
	var existingAcc account.AccountAccount
	if err := db.Where("code = ?", "130090").First(&existingAcc).Error; err != nil {
		clearingAcc := account.AccountAccount{
			Code: "130090",
			Name: "Akun Antara Biaya Impor & Logistik (Landed Cost Interim)",
			Type: "liability_payable",
		}
		clearingAcc.CompanyID = 1
		if err := db.Create(&clearingAcc).Error; err == nil {
			log.Println("[LandedCost] Seeded Chart of Accounts: 130090 Landed Cost Interim Clearing")
		}
	}

	// Optional: Check if incoming pickings exist to seed a demo landed cost
	var costCount int64
	db.Model(&StockLandedCost{}).Count(&costCount)
	if costCount == 0 {
		var incomingPicking stock.StockPicking
		if err := db.Where("picking_type = ?", "incoming").First(&incomingPicking).Error; err == nil {
			demoCost := StockLandedCost{
				Name:             "LC/2026/0001",
				Date:             "2026-10-09",
				PickingID:        incomingPicking.ID,
				State:            StateDraft,
				CompanyID:        1,
				Description:      "Alokasi Biaya Pengiriman Laut (Sea Freight) & Bea Masuk Impor Kontainer",
				TotalAmount:      7500000.0,
				AccountJournalID: 5,
				CostLines: []StockLandedCostLine{
					{
						Name:        "Ocean Freight & Container Shipping",
						SplitMethod: SplitByValue,
						PriceUnit:   5000000.0,
						AccountID:   m.service.GetOrCreateClearingAccountID(1),
					},
					{
						Name:        "Bea Masuk & Pajak Impor DJBC",
						SplitMethod: SplitByQuantity,
						PriceUnit:   2000000.0,
						AccountID:   m.service.GetOrCreateClearingAccountID(1),
					},
					{
						Name:        "Terminal Handling Charges (THC Tanjung Priok)",
						SplitMethod: SplitEqual,
						PriceUnit:   500000.0,
						AccountID:   m.service.GetOrCreateClearingAccountID(1),
					},
				},
			}
			if err := db.Create(&demoCost).Error; err == nil {
				log.Println("[LandedCost] Seeded demo landed cost document LC/2026/0001")
				// Try auto-compute for demo
				_, _ = m.service.ComputeValuationLines(demoCost.ID, 1, 1)
			}
		}
	}

	return nil
}
