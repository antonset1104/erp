package asset

import (
	"log"

	"erp-golang/internal/modules/account"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Module struct {
	db         *gorm.DB
	accountSvc *account.Service
	svc        *Service
	handler    *Handler
}

func NewModule(db *gorm.DB, accountSvc *account.Service) *Module {
	svc := NewService(db, accountSvc)
	handler := NewHandler(svc)
	return &Module{
		db:         db,
		accountSvc: accountSvc,
		svc:        svc,
		handler:    handler,
	}
}

func (m *Module) Service() *Service {
	return m.svc
}

func (m *Module) Name() string {
	return "asset"
}

func (m *Module) AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&Asset{}, &AssetDepreciationLine{})
}

func (m *Module) RegisterRoutes(api *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	group := api.Group("/assets")
	group.Use(authMiddleware)
	{
		group.GET("", m.handler.ListAssets)
		group.POST("", m.handler.CreateAsset)
		group.GET("/:id", m.handler.GetAsset)
		group.POST("/:id/compute", m.handler.ComputeDepreciation)
		group.POST("/:id/confirm", m.handler.ActionConfirm)
		group.POST("/lines/:line_id/post", m.handler.PostDepreciationLine)
		group.POST("/auto-depreciate", m.handler.AutoPostDueDepreciations)
	}
}

func (m *Module) Seed(db *gorm.DB) error {
	var count int64
	db.Model(&Asset{}).Count(&count)
	if count == 0 {
		// Find Accounts
		var assetAcc, accumAcc, deprAcc account.AccountAccount
		db.Where("code = ?", "150100").First(&assetAcc)
		db.Where("code = ?", "150900").First(&accumAcc)
		db.Where("code = ?", "610100").First(&deprAcc)

		// Fallback if not found yet
		if assetAcc.ID == 0 {
			assetAcc = account.AccountAccount{Code: "150100", Name: "Aset Tetap Peralatan & Mesin", Type: "asset_cash"}
			assetAcc.CompanyID = 1
			db.Create(&assetAcc)
		}
		if accumAcc.ID == 0 {
			accumAcc = account.AccountAccount{Code: "150900", Name: "Akumulasi Penyusutan Peralatan", Type: "asset_cash"}
			accumAcc.CompanyID = 1
			db.Create(&accumAcc)
		}
		if deprAcc.ID == 0 {
			deprAcc = account.AccountAccount{Code: "610100", Name: "Beban Penyusutan Aset Tetap", Type: "expense"}
			deprAcc.CompanyID = 1
			db.Create(&deprAcc)
		}

		var miscJrn account.AccountJournal
		db.Where("code = ?", "MISC").First(&miscJrn)
		if miscJrn.ID == 0 {
			miscJrn = account.AccountJournal{Code: "MISC", Name: "Miscellaneous Operations", Type: "general", DefaultAccountID: 1}
			miscJrn.CompanyID = 1
			db.Create(&miscJrn)
		}

		partnerID := uint(1) // PT Maju Bersama Teknologi
		asset1 := Asset{
			Name:                  "Server Rack Enterprise Datacenter Dell PowerEdge",
			Code:                  "AST/2026/00001",
			PartnerID:             &partnerID,
			PurchaseDate:          "2026-01-15",
			OriginalValue:         72000000,
			SalvageValue:          7200000,
			Method:                "linear",
			MethodNumber:          36, // 36 months / 3 years
			MethodPeriod:          "months",
			AssetAccountID:        assetAcc.ID,
			DepreciationAccountID: deprAcc.ID,
			AccumulatedAccountID:  accumAcc.ID,
			JournalID:             miscJrn.ID,
			State:                 "running",
			Note:                  "Server infrastruktur cloud ERP multi-tenant dengan redundant power supply",
		}
		asset1.CompanyID = 1
		asset1.CreateUID = 1
		asset1.WriteUID = 1

		if err := m.svc.CreateAsset(&asset1); err == nil {
			m.svc.ActionConfirm(asset1.ID, 1)

			// Automatically post first 2 depreciation periods for demo data
			var firstLines []AssetDepreciationLine
			db.Where("asset_id = ?", asset1.ID).Order("sequence asc").Limit(2).Find(&firstLines)
			for _, line := range firstLines {
				m.svc.PostDepreciationLine(line.ID, 1, 1)
			}
			log.Println("[Asset] Seeded default Fixed Asset & posted initial GL depreciations")
		}
	}
	return nil
}
