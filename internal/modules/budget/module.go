package budget

import (
	"log"

	"erp-golang/internal/core/ai"
	"erp-golang/internal/modules/account"
	"erp-golang/internal/modules/chatter"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Module struct {
	service *Service
	handler *Handler
}

func NewModule(db *gorm.DB, chatterSvc *chatter.Service, aiClient *ai.Client, accountSvc *account.Service) *Module {
	svc := NewService(db, chatterSvc, aiClient, accountSvc)
	h := NewHandler(svc)
	return &Module{
		service: svc,
		handler: h,
	}
}

func (m *Module) Name() string {
	return "budget"
}

func (m *Module) Service() *Service {
	return m.service
}

func (m *Module) AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&AnalyticAccount{},
		&BudgetaryPosition{},
		&Budget{},
		&BudgetLine{},
	)
}

func (m *Module) Seed(db *gorm.DB) error {
	// 1. Seed Analytic Accounts (Cost Centers / Projects / Departments)
	var countAnalytic int64
	db.Model(&AnalyticAccount{}).Count(&countAnalytic)
	if countAnalytic == 0 {
		analytics := []AnalyticAccount{
			{
				Code:        "AN/ENG",
				Name:        "Departemen IT & Software Engineering",
				Description: "Pusat biaya pengembangan sistem ERP, cloud server, dan arsitektur data",
				Active:      true,
			},
			{
				Code:        "AN/MKT",
				Name:        "Divisi Penjualan & Digital Marketing",
				Description: "Pusat biaya periklanan, campaign B2B, komisi sales, dan pameran",
				Active:      true,
			},
			{
				Code:        "AN/OPS",
				Name:        "Operasional Pabrik & Pergudangan",
				Description: "Pusat biaya manufaktur barang jadi, perawatan mesin, dan logistik armada",
				Active:      true,
			},
			{
				Code:        "AN/HR",
				Name:        "Departemen SDM & General Affairs",
				Description: "Pusat biaya rekrutmen karyawan, pelatihan, payroll, dan perlengkapan kantor",
				Active:      true,
			},
		}
		for _, a := range analytics {
			a.CompanyID = 1
			if err := db.Create(&a).Error; err != nil {
				log.Printf("[Budget] Gagal seeding akun analitik %s: %v", a.Code, err)
			}
		}
		log.Println("[Budget] Berhasil seeding 4 master Akun Analitik / Cost Center")
	}

	// 2. Seed Budgetary Positions (Pos Anggaran Akuntansi)
	var countPositions int64
	db.Model(&BudgetaryPosition{}).Count(&countPositions)
	if countPositions == 0 {
		// Fetch actual account IDs from DB
		var acctGaji, acctTunj, acctOps, acctMaint account.AccountAccount
		db.Where("code = ?", "610200").First(&acctGaji)
		db.Where("code = ?", "610300").First(&acctTunj)
		db.Where("code = ?", "610000").First(&acctOps)
		db.Where("code = ?", "620100").First(&acctMaint)

		gajiIDs := "20,21"
		if acctGaji.ID > 0 && acctTunj.ID > 0 {
			gajiIDs = string(rune(acctGaji.ID)) + "," + string(rune(acctTunj.ID))
		}

		positions := []BudgetaryPosition{
			{
				Name:        "Pos Beban Gaji & Ketenagakerjaan",
				Code:        "BP/PAYROLL",
				Description: "Plafon anggaran kompensasi pokok, tunjangan keahlian, lembur, dan BPJS karyawan",
				AccountIDs:  gajiIDs,
			},
			{
				Name:        "Pos Beban Operasional Kantor & Perjalanan",
				Code:        "BP/OPS",
				Description: "Plafon biaya perjalanan dinas, konsumsi tim, utilitas kantor, dan ATK",
				AccountIDs:  "20,21,22",
			},
			{
				Name:        "Pos Beban Pemeliharaan Mesin & Armada",
				Code:        "BP/MAINT",
				Description: "Plafon servis berkala mesin pabrik, suku cadang, dan pemeliharaan armada truk",
				AccountIDs:  "20,21",
			},
			{
				Name:        "Pos Beban Pemasaran & Campaign Promosi",
				Code:        "BP/MKT",
				Description: "Plafon belanja iklan digital, event kemitraan, dan promosi retail",
				AccountIDs:  "20,21,22",
			},
		}

		for _, p := range positions {
			p.CompanyID = 1
			if err := db.Create(&p).Error; err != nil {
				log.Printf("[Budget] Gagal seeding pos anggaran %s: %v", p.Code, err)
			}
		}
		log.Println("[Budget] Berhasil seeding 4 master Pos Anggaran (Budgetary Positions)")
	}

	// 3. Seed Demo Budget Plan (RKAB Q4 2026)
	var countBudgets int64
	db.Model(&Budget{}).Count(&countBudgets)
	if countBudgets == 0 {
		var posPayroll, posOps, posMaint, posMkt BudgetaryPosition
		db.Where("code = ?", "BP/PAYROLL").First(&posPayroll)
		db.Where("code = ?", "BP/OPS").First(&posOps)
		db.Where("code = ?", "BP/MAINT").First(&posMaint)
		db.Where("code = ?", "BP/MKT").First(&posMkt)

		var anEng, anMkt, anOps, anHR AnalyticAccount
		db.Where("code = ?", "AN/ENG").First(&anEng)
		db.Where("code = ?", "AN/MKT").First(&anMkt)
		db.Where("code = ?", "AN/OPS").First(&anOps)
		db.Where("code = ?", "AN/HR").First(&anHR)

		if posPayroll.ID > 0 && anEng.ID > 0 {
			demoBudget := Budget{
				Name:                   "Rencana Kerja & Anggaran Biaya (RKAB) Q4 2026",
				Code:                   "BDG/2026/Q4/00001",
				UserID:                 1,
				DateFrom:               "2026-10-01",
				DateTo:                 "2026-12-31",
				State:                  "validate",
				TotalPlannedAmount:     150000000,
				TotalPracticalAmount:   20750000,
				TotalTheoreticalAmount: 12500000,
				AchievementPercentage:  13.83,
				AICheckStatus:          "low",
				AIRiskScore:            15.0,
				AINotes:                "Audit Anggaran AI: Penyerapan kuartal berjalan sehat dan terkontrol (13.8% dari plafon Rp 150.000.000). Seluruh alokasi pos berada di bawah batas risiko overspend.",
				Notes:                  "Rencana Anggaran Operasional & Capex Perusahaan Q4 2026",
			}
			demoBudget.CompanyID = 1
			if err := db.Create(&demoBudget).Error; err == nil {
				lines := []BudgetLine{
					{
						BudgetID:            demoBudget.ID,
						BudgetaryPositionID: posPayroll.ID,
						AnalyticAccountID:   anEng.ID,
						DateFrom:            "2026-10-01",
						DateTo:              "2026-12-31",
						PlannedAmount:       75000000,
						PracticalAmount:     20750000,
						TheoriticalAmount:   6250000,
						Percentage:          27.67,
						VarianceAmount:      54250000,
						Notes:               "Beban Gaji & Tunjangan Tim Software Engineering",
					},
					{
						BudgetID:            demoBudget.ID,
						BudgetaryPositionID: posOps.ID,
						AnalyticAccountID:   anOps.ID,
						DateFrom:            "2026-10-01",
						DateTo:              "2026-12-31",
						PlannedAmount:       35000000,
						PracticalAmount:     0,
						TheoriticalAmount:   2916667,
						Percentage:          0,
						VarianceAmount:      35000000,
						Notes:               "Operasional Kantor & Utilitas Gudang",
					},
					{
						BudgetID:            demoBudget.ID,
						BudgetaryPositionID: posMkt.ID,
						AnalyticAccountID:   anMkt.ID,
						DateFrom:            "2026-10-01",
						DateTo:              "2026-12-31",
						PlannedAmount:       25000000,
						PracticalAmount:     0,
						TheoriticalAmount:   2083333,
						Percentage:          0,
						VarianceAmount:      25000000,
						Notes:               "Belanja Iklan & Lead Generation Q4",
					},
					{
						BudgetID:            demoBudget.ID,
						BudgetaryPositionID: posMaint.ID,
						AnalyticAccountID:   anOps.ID,
						DateFrom:            "2026-10-01",
						DateTo:              "2026-12-31",
						PlannedAmount:       15000000,
						PracticalAmount:     0,
						TheoriticalAmount:   1250000,
						Percentage:          0,
						VarianceAmount:      15000000,
						Notes:               "Servis Preventif Mesin & Armada Logistik",
					},
				}
				for _, l := range lines {
					l.CompanyID = 1
					_ = db.Create(&l)
				}
				log.Println("[Budget] Berhasil seeding Demo Rencana Anggaran RKAB Q4 2026 (4 lines)")
			}
		}
	}

	return nil
}

func (m *Module) RegisterRoutes(api *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	group := api.Group("/budget")
	group.Use(authMiddleware)
	{
		// Analytic Accounts
		group.GET("/analytic-accounts", m.handler.ListAnalyticAccounts)
		group.POST("/analytic-accounts", m.handler.CreateAnalyticAccount)
		group.GET("/analytic-accounts/:id", m.handler.GetAnalyticAccount)

		// Budgetary Positions
		group.GET("/positions", m.handler.ListBudgetaryPositions)
		group.POST("/positions", m.handler.CreateBudgetaryPosition)
		group.GET("/positions/:id", m.handler.GetBudgetaryPosition)

		// Budgets
		group.GET("/budgets", m.handler.ListBudgets)
		group.POST("/budgets", m.handler.CreateBudget)
		group.GET("/budgets/:id", m.handler.GetBudget)

		// Actions & Lifecycle
		group.POST("/budgets/:id/compute", m.handler.ComputeBudget)
		group.POST("/budgets/:id/confirm", m.handler.ConfirmBudget)
		group.POST("/budgets/:id/validate", m.handler.ValidateBudget)
		group.POST("/budgets/:id/done", m.handler.DoneBudget)
		group.POST("/budgets/:id/cancel", m.handler.CancelBudget)
		group.POST("/budgets/:id/draft", m.handler.ResetDraftBudget)

		// AI Variance & Burn-rate Audit
		group.POST("/budgets/:id/ai-audit", m.handler.AuditWithAI)

		// KPI Stats
		group.GET("/stats", m.handler.GetStats)
	}
}
