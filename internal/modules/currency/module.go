package currency

import (
	"log"

	"erp-golang/internal/core/ai"
	"erp-golang/internal/modules/account"
	"erp-golang/internal/modules/chatter"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Module struct {
	repo       *Repository
	service    *Service
	handler    *Handler
	accountSvc *account.Service
	chatterSvc *chatter.Service
	aiClient   *ai.Client
}

func NewModule(db *gorm.DB, accountSvc *account.Service, chatterSvc *chatter.Service, aiClient *ai.Client) *Module {
	repo := NewRepository(db)
	service := NewService(db, repo, accountSvc, chatterSvc, aiClient)
	handler := NewHandler(service)

	return &Module{
		repo:       repo,
		service:    service,
		handler:    handler,
		accountSvc: accountSvc,
		chatterSvc: chatterSvc,
		aiClient:   aiClient,
	}
}

func (m *Module) Name() string {
	return "currency"
}

func (m *Module) Service() *Service {
	return m.service
}

func (m *Module) RegisterRoutes(api *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	group := api.Group("/currencies")
	group.Use(authMiddleware)
	{
		group.GET("", m.handler.ListCurrencies)
		group.POST("", m.handler.CreateCurrency)
		group.GET("/stats", m.handler.Stats)
		group.POST("/convert", m.handler.Convert)
		group.POST("/realized-gain-loss", m.handler.CalculateRealizedGainLoss)

		// Revaluations (PSAK 10)
		group.GET("/revaluations", m.handler.ListRevaluations)
		group.POST("/revaluations", m.handler.ComputeRevaluation)
		group.POST("/revaluations/compute", m.handler.ComputeRevaluation)
		group.GET("/revaluations/:id", m.handler.GetRevaluationByID)
		group.POST("/revaluations/:id/post", m.handler.PostRevaluation)
		group.PUT("/revaluations/:id/post", m.handler.PostRevaluation)
		group.POST("/revaluations/:id/cancel", m.handler.CancelRevaluation)
		group.PUT("/revaluations/:id/cancel", m.handler.CancelRevaluation)

		// Currency details, rates & AI audit
		group.GET("/:id", m.handler.GetCurrencyByID)
		group.GET("/:id/rates", m.handler.ListRates)
		group.POST("/:id/rates", m.handler.CreateRate)
		group.GET("/:id/ai-risk-audit", m.handler.ForexRiskAIAudit)
	}
}

func (m *Module) AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&Currency{},
		&CurrencyRate{},
		&ForexRevaluation{},
		&ForexRevaluationLine{},
	)
}

func (m *Module) Seed(db *gorm.DB) error {
	// 1. Ensure Forex Accounts exist in Chart of Accounts
	forexAccounts := []account.AccountAccount{
		{Code: "710100", Name: "Pendapatan Laba Selisih Kurs Terealisasi (Realized Forex Gain)", Type: "income"},
		{Code: "620100", Name: "Beban Kerugian Selisih Kurs Terealisasi (Realized Forex Loss)", Type: "expense"},
		{Code: "710110", Name: "Laba Selisih Kurs Belum Terealisasi (Unrealized Forex Gain)", Type: "income"},
		{Code: "620110", Name: "Rugi Selisih Kurs Belum Terealisasi (Unrealized Forex Loss)", Type: "expense"},
	}
	for i := range forexAccounts {
		var existing account.AccountAccount
		if err := db.Where("code = ?", forexAccounts[i].Code).First(&existing).Error; err != nil {
			forexAccounts[i].CompanyID = 1
			db.Create(&forexAccounts[i])
		}
	}

	// 2. Check if Currencies already exist
	var count int64
	db.Model(&Currency{}).Count(&count)
	if count > 0 {
		return nil
	}

	currencies := []Currency{
		{Name: "IDR", Symbol: "Rp", FullName: "Indonesian Rupiah", DecimalPlaces: 2, Rounding: 1.0, Active: true, IsBase: true},
		{Name: "USD", Symbol: "$", FullName: "US Dollar", DecimalPlaces: 2, Rounding: 0.01, Active: true, IsBase: false},
		{Name: "EUR", Symbol: "€", FullName: "Euro", DecimalPlaces: 2, Rounding: 0.01, Active: true, IsBase: false},
		{Name: "SGD", Symbol: "S$", FullName: "Singapore Dollar", DecimalPlaces: 2, Rounding: 0.01, Active: true, IsBase: false},
		{Name: "JPY", Symbol: "¥", FullName: "Japanese Yen", DecimalPlaces: 0, Rounding: 1.0, Active: true, IsBase: false},
		{Name: "GBP", Symbol: "£", FullName: "British Pound Sterling", DecimalPlaces: 2, Rounding: 0.01, Active: true, IsBase: false},
	}

	for i := range currencies {
		currencies[i].CompanyID = 1
		db.Create(&currencies[i])
	}

	// 3. Seed Realistic Currency Rates
	type rateSeed struct {
		CurrName   string
		Date       string
		DirectRate float64
	}

	seeds := []rateSeed{
		// Historical Rates (1 Oct 2026)
		{CurrName: "USD", Date: "2026-10-01", DirectRate: 15950.0},
		{CurrName: "EUR", Date: "2026-10-01", DirectRate: 17300.0},
		{CurrName: "SGD", Date: "2026-10-01", DirectRate: 12150.0},
		{CurrName: "JPY", Date: "2026-10-01", DirectRate: 107.8},
		{CurrName: "GBP", Date: "2026-10-01", DirectRate: 20600.0},

		// Recent Rates (8 Oct 2026)
		{CurrName: "USD", Date: "2026-10-08", DirectRate: 16120.0},
		{CurrName: "EUR", Date: "2026-10-08", DirectRate: 17520.0},
		{CurrName: "SGD", Date: "2026-10-08", DirectRate: 12340.0},
		{CurrName: "JPY", Date: "2026-10-08", DirectRate: 109.2},
		{CurrName: "GBP", Date: "2026-10-08", DirectRate: 20950.0},
	}

	for _, s := range seeds {
		var c Currency
		if err := db.Where("name = ?", s.CurrName).First(&c).Error; err == nil {
			r := CurrencyRate{
				CurrencyID: c.ID,
				Name:       s.Date,
				DirectRate: s.DirectRate,
				Rate:       1.0 / s.DirectRate,
				Source:     "Bank Indonesia JISDOR",
				CompanyID:  1,
			}
			db.Create(&r)
		}
	}

	log.Println("[Currency] Berhasil seeding 6 mata uang master & histori kurs Bank Indonesia JISDOR")
	return nil
}
