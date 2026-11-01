package tax_ebupot

import (
	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"erp-golang/internal/core/ai"
	"erp-golang/internal/modules/account"
	"erp-golang/internal/modules/chatter"
	"erp-golang/internal/modules/payroll"
)

type Module struct {
	repo       Repository
	service    Service
	handler    *Handler
	db         *gorm.DB
	payrollSvc *payroll.Service
	accountSvc *account.Service
	chatterSvc *chatter.Service
	aiClient   *ai.Client
}

func NewModule(
	db *gorm.DB,
	payrollSvc *payroll.Service,
	accountSvc *account.Service,
	chatterSvc *chatter.Service,
	aiClient *ai.Client,
) *Module {
	repo := NewRepository(db)
	service := NewService(repo, db, payrollSvc, accountSvc, chatterSvc, aiClient)
	handler := NewHandler(service)

	return &Module{
		repo:       repo,
		service:    service,
		handler:    handler,
		db:         db,
		payrollSvc: payrollSvc,
		accountSvc: accountSvc,
		chatterSvc: chatterSvc,
		aiClient:   aiClient,
	}
}

func (m *Module) Name() string {
	return "tax_ebupot"
}

func (m *Module) Priority() int {
	return 41
}

func (m *Module) Service() Service {
	return m.service
}

func (m *Module) AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&EbupotPeriod{},
		&WithholdingSlip{},
		&EbupotExportLog{},
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
	db.Model(&EbupotPeriod{}).Count(&count)
	if count > 0 {
		return nil
	}

	log.Println("🌱 Seeding Indonesian e-Bupot 21/26 & SPT Masa PPh 21 Sample Data...")

	// Create sample period for October 2026
	period := EbupotPeriod{
		PeriodCode:       "SPT21/2026/10/0",
		TaxMonth:         10,
		TaxYear:          2026,
		RevisionNumber:   0,
		CompanyID:        1,
		State:            "draft",
		TotalGrossWages:  0,
		TotalTaxDeducted: 0,
		TotalBupotCount:  0,
		Notes:            "SPT Masa PPh Pasal 21/26 Masa Oktober 2026 (Skema TER PP 58/2023)",
	}
	if err := db.Create(&period).Error; err != nil {
		return err
	}

	// Create sample withholding slips
	slips := []WithholdingSlip{
		{
			PeriodID:             period.ID,
			SlipNumber:           "BP21/2026/10/00001",
			SlipType:             "bp_pegawai_tetap",
			TaxObjectCode:        "21-100-01",
			TaxObjectName:        "Pegawai Tetap (Gaji Pokok & Tunjangan Jabatan)",
			RecipientType:        "employee",
			NIK:                  "3171012304890001",
			NPWP:                 "012345678901000",
			RecipientName:        "Ahmad Fauzi, S.T.",
			RecipientAddress:     "Jl. Sudirman Kav 24, Jakarta Selatan",
			PTKPStatus:           "TK/0",
			TERCategory:          "TER_A",
			TaxCalculationMethod: "ter_monthly",
			GrossIncome:          12500000,
			TaxableBaseDPP:       12500000,
			EffectiveRatePct:     4.0, // TER A (11.6jt - 12.5jt)
			TaxWithheld:          500000,
			TaxFacilityType:      "none",
			Status:               "verified",
			IsNIKMatched:         true,
			CompanyID:            1,
			Notes:                "Potongan TER A (TK/0) 4.0% atas gaji Rp 12.500.000",
		},
		{
			PeriodID:             period.ID,
			SlipNumber:           "BP21/2026/10/00002",
			SlipType:             "bp_pegawai_tetap",
			TaxObjectCode:        "21-100-01",
			TaxObjectName:        "Pegawai Tetap (Gaji Pokok & Tunjangan Keluarga)",
			RecipientType:        "employee",
			NIK:                  "3276021508920003",
			NPWP:                 "098765432109000",
			RecipientName:        "Siti Nurhaliza, M.M.",
			RecipientAddress:     "Jl. Margonda Raya No. 45, Depok",
			PTKPStatus:           "K/1",
			TERCategory:          "TER_B",
			TaxCalculationMethod: "ter_monthly",
			GrossIncome:          18500000,
			TaxableBaseDPP:       18500000,
			EffectiveRatePct:     8.0, // TER B (18.45jt - 21.85jt)
			TaxWithheld:          1480000,
			TaxFacilityType:      "none",
			Status:               "verified",
			IsNIKMatched:         true,
			CompanyID:            1,
			Notes:                "Potongan TER B (K/1) 8.0% atas gaji Rp 18.500.000",
		},
		{
			PeriodID:             period.ID,
			SlipNumber:           "BP21/2026/10/00003",
			SlipType:             "bp_tidak_final",
			TaxObjectCode:        "21-100-07",
			TaxObjectName:        "Tenaga Ahli Konsultan ERP (Bukan Pegawai)",
			RecipientType:        "vendor_expert",
			NIK:                  "3174092011850002",
			NPWP:                 "081234567890000",
			RecipientName:        "Dr. Ir. Budi Santoso, M.Kom.",
			RecipientAddress:     "Kuningan Barat No. 12, Jakarta",
			PTKPStatus:           "TK/0",
			TERCategory:          "PASAL_17",
			TaxCalculationMethod: "pph21_regular",
			GrossIncome:          25000000,
			TaxableBaseDPP:       12500000, // 50% x 25.000.000
			EffectiveRatePct:     5.0,
			TaxWithheld:          625000,
			TaxFacilityType:      "none",
			Status:               "verified",
			IsNIKMatched:         true,
			CompanyID:            1,
			Notes:                "Honorarium Jasa Konsultasi Teknis - DPP 50% (Pasal 17)",
		},
	}

	var totalGross, totalTax float64
	for _, sl := range slips {
		db.Create(&sl)
		totalGross += sl.GrossIncome
		totalTax += sl.TaxWithheld
	}

	period.TotalGrossWages = totalGross
	period.TotalTaxDeducted = totalTax
	period.TotalBupotCount = len(slips)
	db.Save(&period)

	log.Printf("✅ e-Bupot 21/26 seeded successfully: Period %s, %d Slips, Total Pajak Rp %.2f\n",
		period.PeriodCode, len(slips), totalTax)

	return nil
}

func (m *Module) Init(r *gin.Engine, db *gorm.DB, authMiddleware gin.HandlerFunc) error {
	m.db = db
	apiGroup := r.Group("/api/v1")
	m.RegisterRoutes(apiGroup, authMiddleware)
	return m.Seed(db)
}
