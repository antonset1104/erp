package intercompany

import (
	"log"
	"time"

	"erp-golang/internal/core/ai"
	"erp-golang/internal/core/models"
	"erp-golang/internal/modules/account"
	"erp-golang/internal/modules/base"
	"erp-golang/internal/modules/chatter"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Module struct {
	db         *gorm.DB
	repo       *Repository
	service    *Service
	handler    *Handler
	accountSvc *account.Service
	chatterSvc *chatter.Service
	aiClient   *ai.Client
}

func NewModule(db *gorm.DB, accountSvc *account.Service, chatterSvc *chatter.Service, aiClient *ai.Client) *Module {
	repo := NewRepository(db)
	svc := NewService(db, repo, accountSvc, chatterSvc, aiClient)
	h := NewHandler(svc)
	return &Module{
		db:         db,
		repo:       repo,
		service:    svc,
		handler:    h,
		accountSvc: accountSvc,
		chatterSvc: chatterSvc,
		aiClient:   aiClient,
	}
}

func (m *Module) Name() string {
	return "intercompany"
}

func (m *Module) Priority() int {
	return 38
}

func (m *Module) Service() *Service {
	return m.service
}

func (m *Module) AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&IntercompanyGroup{},
		&IntercompanyRelation{},
		&IntercompanyTransaction{},
		&ConsolidationRun{},
		&ConsolidationLine{},
	)
}

func (m *Module) RegisterRoutes(api *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	m.handler.RegisterRoutes(api, authMiddleware)
}

func (m *Module) Seed(db *gorm.DB) error {
	if err := m.AutoMigrate(db); err != nil {
		return err
	}

	var count int64
	db.Model(&IntercompanyGroup{}).Count(&count)
	if count > 0 {
		return nil
	}

	log.Println("🌱 Seeding Enterprise Multi-Company Corporate Groups & Consolidation Rules...")

	// 1. Locate or ensure Subsidiary Company exists
	var holdingComp base.Company
	if err := m.db.First(&holdingComp, 1).Error; err != nil {
		holdingComp = base.Company{
			Name:     "PT Nusantara Holding Group",
			Email:    "holding@nusantara.local",
			Currency: "IDR",
		}
		m.db.Create(&holdingComp)
	}

	var subComp base.Company
	if err := m.db.Where("id = 2 OR name LIKE ?", "%Anak%").First(&subComp).Error; err != nil {
		subComp = base.Company{
			Name:     "PT Nusantara Distribusi Anak",
			Email:    "distribusi@nusantara.local",
			Currency: "IDR",
		}
		m.db.Create(&subComp)
	}

	// 2. Ensure Intercompany Partners exist
	var partnerSubInHolding base.Partner
	if err := m.db.Where("name LIKE ?", "%Nusantara Distribusi%").First(&partnerSubInHolding).Error; err != nil {
		partnerSubInHolding = base.Partner{
			Name:         "PT Nusantara Distribusi Anak (Subsidiary)",
			IsCompany:    true,
			Email:        "distribusi@nusantara.local",
			CustomerRank: 1,
			SupplierRank: 1,
		}
		partnerSubInHolding.CompanyID = holdingComp.ID
		m.db.Create(&partnerSubInHolding)
	}

	var partnerHoldingInSub base.Partner
	if err := m.db.Where("name LIKE ?", "%Holding%").First(&partnerHoldingInSub).Error; err != nil {
		partnerHoldingInSub = base.Partner{
			Name:         "PT Nusantara Holding Group (Parent)",
			IsCompany:    true,
			Email:        "holding@nusantara.local",
			CustomerRank: 1,
			SupplierRank: 1,
		}
		partnerHoldingInSub.CompanyID = subComp.ID
		m.db.Create(&partnerHoldingInSub)
	}

	// 3. Seed Holding Group
	group := &IntercompanyGroup{
		ModelBase: models.ModelBase{
			CompanyID: holdingComp.ID,
			Active:    true,
		},
		GroupCode:        "GRP/2026/001",
		Name:             "Nusantara Corporate Holdings Group",
		HoldingCompanyID: holdingComp.ID,
		HoldingName:      holdingComp.Name,
		Currency:         "IDR",
		Description:      "Grup holding konglomerasi manufaktur, logistik, dan distribusi perdagangan nasional.",
		IsActive:         true,
	}
	m.db.Create(group)

	// 4. Seed Relation
	rel := &IntercompanyRelation{
		ModelBase: models.ModelBase{
			CompanyID: holdingComp.ID,
			Active:    true,
		},
		GroupID:                group.ID,
		RelationCode:           "REL/2026/001",
		Name:                   "Holding <-> Nusantara Distribusi Anak",
		FromCompanyID:          holdingComp.ID,
		FromCompanyName:        holdingComp.Name,
		FromPartnerID:          partnerSubInHolding.ID,
		ToCompanyID:            subComp.ID,
		ToCompanyName:          subComp.Name,
		ToPartnerID:            partnerHoldingInSub.ID,
		OwnershipPct:           85.0, // 85% majority control
		ConsolidationMethod:    "full",
		AutoMirrorSalePurchase: true,
		AutoMirrorInvoiceBill:  true,
		AutoEliminate:          true,
		IsActive:               true,
	}
	m.db.Create(rel)

	// 5. Seed Mirrored Intercompany Transaction Sample
	tx := &IntercompanyTransaction{
		ModelBase: models.ModelBase{
			CompanyID: holdingComp.ID,
			Active:    true,
		},
		TransactionCode: "ICT/2026/00001",
		GroupID:         group.ID,
		RelationID:      rel.ID,
		TransactionType: "sale_to_purchase",
		SourceModel:     "sale.order",
		SourceResID:     1,
		SourceDocName:   "SO0001",
		SourceCompanyID: holdingComp.ID,
		SourceAmount:    15000000.0,
		TargetModel:     "purchase.order",
		TargetResID:     1,
		TargetDocName:   "PO/IC/2026/0001",
		TargetCompanyID: subComp.ID,
		TargetAmount:    15000000.0,
		SyncStatus:      "synced",
		SyncMessage:     "Sale Order 'SO0001' berhasil dicerminkan menjadi Purchase Order 'PO/IC/2026/0001'",
		SyncedAt:        time.Now().Add(-2 * time.Hour),
	}
	m.db.Create(tx)

	// 6. Seed Sample Financial Consolidation Run
	run := &ConsolidationRun{
		ModelBase: models.ModelBase{
			CompanyID: holdingComp.ID,
			Active:    true,
		},
		RunCode:                    "CNS/2026/00001",
		GroupID:                    group.ID,
		GroupName:                  group.Name,
		PeriodStartDate:            time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		PeriodEndDate:              time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC),
		FiscalYear:                 2026,
		TotalGrossAssets:           95000000.0,
		TotalGrossLiabilities:      38000000.0,
		TotalGrossEquity:           57000000.0,
		TotalGrossRevenue:          65000000.0,
		TotalGrossExpenses:         40000000.0,
		TotalGrossNetIncome:        25000000.0,
		EliminatedReceivables:      9000000.0,
		EliminatedPayables:         9000000.0,
		EliminatedRevenue:          15000000.0,
		EliminatedCOGS:             15000000.0,
		EliminatedUnrealizedProfit: 2250000.0,
		ConsolidatedAssets:         83750000.0,
		ConsolidatedLiabilities:    29000000.0,
		ConsolidatedEquity:         54750000.0,
		ConsolidatedRevenue:        50000000.0,
		ConsolidatedExpenses:       25000000.0,
		ConsolidatedNetIncome:      25000000.0,
		NonControllingInterest:     1500000.0, // 15% minority of subsidiary profit
		ControllingInterest:        23500000.0,
		State:                      "calculated",
		Notes:                      "Konsolidasi laporan keuangan tahunan per IFRS 10 / PSAK 65.",
	}
	m.db.Create(run)

	// Seed Sample Lines
	m.db.Create(&ConsolidationLine{
		RunID:              run.ID,
		CompanyID:          holdingComp.ID,
		CompanyName:        holdingComp.Name,
		OwnershipPct:       100.0,
		FinancialCategory:  "revenue",
		AccountCode:        "4000",
		AccountName:        "Pendapatan Usaha (Revenue)",
		GrossAmount:        40000000.0,
		EliminationDebit:   7500000.0,
		ConsolidatedAmount: 32500000.0,
		EliminationNote:    "Eliminasi Pendapatan Penjualan Antar-Entitas",
	})
	m.db.Create(&ConsolidationLine{
		RunID:              run.ID,
		CompanyID:          subComp.ID,
		CompanyName:        subComp.Name,
		OwnershipPct:       85.0,
		FinancialCategory:  "revenue",
		AccountCode:        "4000",
		AccountName:        "Pendapatan Usaha (Revenue)",
		GrossAmount:        25000000.0,
		EliminationDebit:   7500000.0,
		ConsolidatedAmount: 17500000.0,
		EliminationNote:    "Eliminasi Pendapatan Penjualan Antar-Entitas",
	})

	log.Println("✅ Enterprise Multi-Company Corporate Groups & Consolidation seeded successfully.")
	return nil
}
