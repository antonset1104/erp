package treasury

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"erp-golang/internal/core/ai"
	"erp-golang/internal/core/models"
	"erp-golang/internal/modules/account"
	"erp-golang/internal/modules/chatter"
)

type Module struct {
	db         *gorm.DB
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
		service:    svc,
		handler:    h,
		accountSvc: accountSvc,
		chatterSvc: chatterSvc,
		aiClient:   aiClient,
	}
}

func (m *Module) Name() string {
	return "treasury"
}

func (m *Module) Priority() int {
	return 39
}

func (m *Module) Service() *Service {
	return m.service
}

func (m *Module) AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&TreasuryBankAccount{},
		&CashFlowForecast{},
		&CashFlowForecastItem{},
		&InterBankTransfer{},
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
	db.Model(&TreasuryBankAccount{}).Count(&count)
	if count > 0 {
		return nil
	}

	log.Println("🌱 Seeding Enterprise Treasury Bank Accounts & Cash Flow Forecasting...")

	// 1. Locate Bank Journal or Accounts
	var bankJournal account.AccountJournal
	db.Where("type = 'bank'").First(&bankJournal)

	var bankGLAccount account.AccountAccount
	db.Where("code LIKE '102%' OR code LIKE '111%' OR type = 'asset_cash'").First(&bankGLAccount)
	glAccID := bankGLAccount.ID
	if glAccID == 0 {
		glAccID = 1
	}

	// 2. Seed 3 Active Bank Accounts
	accBCA := TreasuryBankAccount{
		ModelBase: models.ModelBase{
			CompanyID: 1,
			CreateUID: 1,
			Active:    true,
		},
		AccountCode:     "BA/2026/001",
		BankName:        "Bank Central Asia (BCA)",
		AccountNumber:   "800-1288-9900",
		AccountHolder:   "PT Odoo Golang Nusantara",
		AccountType:     BankTypeChecking,
		Currency:        "IDR",
		CurrentBalance:  750000000, // Rp 750 Juta
		MinimumBalance:  100000000, // Rp 100 Juta safety buffer
		OverdraftLimit:  50000000,
		InterestRatePct: 2.5,
		JournalID:       bankJournal.ID,
		GLAccountID:     glAccID,
		Status:          "active",
		IsPrimaryPool:   true,
		Notes:           "Rekening Giro Utama Likuiditas Holding & Operasional Sentral.",
	}
	db.Create(&accBCA)

	accMandiri := TreasuryBankAccount{
		ModelBase: models.ModelBase{
			CompanyID: 1,
			CreateUID: 1,
			Active:    true,
		},
		AccountCode:     "BA/2026/002",
		BankName:        "Bank Mandiri",
		AccountNumber:   "137-00-9821-445",
		AccountHolder:   "PT Odoo Golang Nusantara",
		AccountType:     BankTypeChecking,
		Currency:        "IDR",
		CurrentBalance:  320000000, // Rp 320 Juta
		MinimumBalance:  50000000,
		OverdraftLimit:  25000000,
		InterestRatePct: 2.2,
		JournalID:       bankJournal.ID,
		GLAccountID:     glAccID,
		Status:          "active",
		IsPrimaryPool:   false,
		Notes:           "Rekening Operasional Khusus Penggajian Payroll & BPJS.",
	}
	db.Create(&accMandiri)

	accBNI := TreasuryBankAccount{
		ModelBase: models.ModelBase{
			CompanyID: 1,
			CreateUID: 1,
			Active:    true,
		},
		AccountCode:     "BA/2026/003",
		BankName:        "Bank Negara Indonesia (BNI)",
		AccountNumber:   "089-2234-551",
		AccountHolder:   "PT Odoo Golang Nusantara",
		AccountType:     BankTypeDeposit,
		Currency:        "IDR",
		CurrentBalance:  200000000, // Rp 200 Juta
		MinimumBalance:  50000000,
		OverdraftLimit:  0,
		InterestRatePct: 4.75, // Deposito 1 bulan
		JournalID:       bankJournal.ID,
		GLAccountID:     glAccID,
		Status:          "active",
		IsPrimaryPool:   false,
		Notes:           "Deposito Berjangka Jangka Pendek untuk Optimalisasi Yield Kas.",
	}
	db.Create(&accBNI)

	// 3. Seed Sample Cash Flow Forecast (30 Days)
	now := time.Now()
	totalOpening := accBCA.CurrentBalance + accMandiri.CurrentBalance + accBNI.CurrentBalance // 1.27 Milyar

	forecast := CashFlowForecast{
		ModelBase: models.ModelBase{
			CompanyID: 1,
			CreateUID: 1,
			Active:    true,
		},
		ForecastCode:        "CFF/2026/00001",
		Name:                "Proyeksi Likuiditas & Arus Kas Konsolidasi Q4 2026",
		BaseDate:            now,
		HorizonDays:         30,
		OpeningCashBalance:  totalOpening,
		ProjectedInflow:     450000000, // 450 Juta
		ProjectedOutflow:    280000000, // 280 Juta
		ProjectedNetCash:    170000000, // Net +170 Juta
		ProjectedEndingCash: totalOpening + 170000000,
		MinimumSafetyBuffer: 200000000,
		LowestPointCash:     totalOpening + 45000000,
		DeficitRisk:         RiskNone,
		EstimatedRunwayDays: 136,
		Status:              "active",
		Notes:               "Proyeksi kas operasional 30 hari ke depan dengan asumsi penagihan invoice normal.",
	}
	db.Create(&forecast)

	// Seed Sample Forecast Items
	items := []CashFlowForecastItem{
		{
			ModelBase:       models.ModelBase{CompanyID: 1, CreateUID: 1, Active: true},
			ForecastID:      forecast.ID,
			Category:        CategoryARReceivables,
			Direction:       DirectionInflow,
			ExpectedDate:    now.AddDate(0, 0, 5),
			GrossAmount:     250000000,
			CertaintyFactor: 0.95,
			AdjustedAmount:  237500000,
			SourceDocType:   "account.move",
			SourceDocRef:    "INV/2026/0001",
			PartnerName:     "PT Mitra Global Mandiri",
			Notes:           "Faktur Penjualan Termin 1 jatuh tempo",
		},
		{
			ModelBase:       models.ModelBase{CompanyID: 1, CreateUID: 1, Active: true},
			ForecastID:      forecast.ID,
			Category:        CategoryRecurringMRR,
			Direction:       DirectionInflow,
			ExpectedDate:    now.AddDate(0, 0, 15),
			GrossAmount:     212500000,
			CertaintyFactor: 0.90,
			AdjustedAmount:  191250000,
			SourceDocType:   "subscription",
			SourceDocRef:    "SUB/2026/00001",
			PartnerName:     "Klien Langganan Enterprise Cloud",
			Notes:           "Perpanjangan langganan tahunan berulang",
		},
		{
			ModelBase:       models.ModelBase{CompanyID: 1, CreateUID: 1, Active: true},
			ForecastID:      forecast.ID,
			Category:        CategoryAPPayables,
			Direction:       DirectionOutflow,
			ExpectedDate:    now.AddDate(0, 0, 8),
			GrossAmount:     120000000,
			CertaintyFactor: 1.0,
			AdjustedAmount:  120000000,
			SourceDocType:   "account.move",
			SourceDocRef:    "BILL/2026/0001",
			PartnerName:     "PT Logistik Nusantara Sejahtera",
			Notes:           "Pelunasan tagihan freight logistik impor",
		},
		{
			ModelBase:       models.ModelBase{CompanyID: 1, CreateUID: 1, Active: true},
			ForecastID:      forecast.ID,
			Category:        CategoryPayrollSalaries,
			Direction:       DirectionOutflow,
			ExpectedDate:    now.AddDate(0, 0, 25),
			GrossAmount:     160000000,
			CertaintyFactor: 1.0,
			AdjustedAmount:  160000000,
			SourceDocType:   "hr.payroll",
			Notes:           "Kewajiban Penggajian Bulanan Karyawan & Iuran BPJS Ketenagakerjaan",
		},
	}
	db.Create(&items)

	// 4. Seed Sample Inter-Bank Fund Sweep
	transfer := InterBankTransfer{
		ModelBase: models.ModelBase{
			CompanyID: 1,
			CreateUID: 1,
			Active:    true,
		},
		TransferCode:    "IBT/2026/00001",
		FromAccountID:   accBCA.ID,
		FromAccountName: "Bank Central Asia (BCA) (800-1288-9900)",
		ToAccountID:     accMandiri.ID,
		ToAccountName:   "Bank Mandiri (137-00-9821-445)",
		TransferDate:    now.AddDate(0, 0, -2),
		Amount:          150000000, // Rp 150 Juta
		TransferFee:     2500,
		Reference:       "SWEEP-PAYROLL-OKT26",
		State:           "posted",
		ExecutedByUID:   1,
		Memo:            "Likuiditas fund sweep dari rekening operasional ke rekening payroll untuk gaji bulanan.",
	}
	db.Create(&transfer)

	log.Println("✅ Enterprise Treasury Bank Accounts & Cash Flow Forecasting seeded successfully.")
	return nil
}
