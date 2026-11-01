package bank_reconciliation

import (
	"log"

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
	return "bank_reconciliation"
}

func (m *Module) Service() *Service {
	return m.service
}

func (m *Module) AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&AccountBankStatement{},
		&AccountBankStatementLine{},
		&AccountReconcileModel{},
	)
}

func (m *Module) RegisterRoutes(api *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	group := api.Group("/bank-reconcile")
	group.Use(authMiddleware)
	{
		// Statements
		group.GET("/statements", m.handler.ListStatements)
		group.POST("/statements", m.handler.CreateStatement)
		group.GET("/statements/:id", m.handler.GetStatement)
		group.DELETE("/statements/:id", m.handler.DeleteStatement)
		group.POST("/statements/:id/confirm", m.handler.ConfirmStatement)

		// Statement Lines
		group.POST("/statements/:id/lines", m.handler.AddStatementLine)
		group.DELETE("/lines/:id", m.handler.DeleteStatementLine)

		// CSV Mutasi Import
		group.POST("/import-csv", m.handler.ParseAndImportCSV)

		// Reconcile Engine & Auto-Matching
		group.GET("/statements/:id/suggestions", m.handler.GetAutoMatchSuggestions)
		group.POST("/reconcile", m.handler.ReconcileLine)
		group.POST("/batch-reconcile", m.handler.BatchReconcile)
		group.POST("/statements/:id/auto-reconcile", m.handler.AutoReconcileHighConfidence)

		// AI Copilot & Export
		group.GET("/statements/:id/ai-audit", m.handler.AIBankCopilot)
		group.GET("/statements/:id/export", m.handler.ExportStatementCSV)

		// Rules & Analytics
		group.GET("/rules", m.handler.ListRules)
		group.POST("/rules", m.handler.CreateRule)
		group.GET("/stats", m.handler.Stats)
	}
}

func (m *Module) Seed(db *gorm.DB) error {
	if err := m.AutoMigrate(db); err != nil {
		return err
	}

	// Seed default rules
	m.service.SeedDefaultRules(1)

	// Check if demo statement exists
	var count int64
	db.Model(&AccountBankStatement{}).Count(&count)
	if count == 0 {
		// Find Bank BCA journal
		var bnkJournal account.AccountJournal
		if err := db.Where("code = ?", "BNK").First(&bnkJournal).Error; err != nil {
			// Find any bank journal
			if err := db.Where("type = 'bank'").First(&bnkJournal).Error; err != nil {
				return nil
			}
		}

		demoStmt := AccountBankStatement{
			Name:           "BNK/STAT/2026/0001",
			JournalID:      bnkJournal.ID,
			Date:           "2026-10-09",
			BalanceStart:   50000000.0, // 50 Juta
			BalanceEnd:     50000000.0,
			BalanceEndReal: 74975000.0, // Target after lines
			Difference:     24975000.0,
			State:          "open",
			Note:           "Rekening Koran BCA Giro Operasional Oktober 2026",
		}
		demoStmt.CompanyID = 1
		if err := db.Create(&demoStmt).Error; err == nil {
			demoLines := []AccountBankStatementLine{
				{
					StatementID:  demoStmt.ID,
					Date:         "2026-10-09",
					Name:         "TRSF CR PT MITRA ABADI INV/2026/00001",
					Amount:       15000000.0,
					Ref:          "BCA-TRSF-88129",
					IsReconciled: false,
				},
				{
					StatementID:  demoStmt.ID,
					Date:         "2026-10-09",
					Name:         "TRSF CR CV KARYA MAKMUR INV/2026/00002",
					Amount:       10000000.0,
					Ref:          "BCA-TRSF-88130",
					IsReconciled: false,
				},
				{
					StatementID:  demoStmt.ID,
					Date:         "2026-10-09",
					Name:         "BIAYA ADM BULANAN GIRO BCA",
					Amount:       -25000.0,
					Ref:          "ADM-BCA-1026",
					IsReconciled: false,
				},
			}
			for i := range demoLines {
				demoLines[i].CompanyID = 1
			}
			_ = db.Create(&demoLines)

			// Update statement totals
			m.service.recalculateStatementBalances(&demoStmt)
			_ = db.Save(&demoStmt)
			log.Println("[BankReconciliation] Seeded initial demo bank statement with 3 lines")
		}
	}

	return nil
}
