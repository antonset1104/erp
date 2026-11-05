package payroll

import (
	"log"

	"erp-golang/internal/core/ai"
	"erp-golang/internal/modules/account"
	"erp-golang/internal/modules/chatter"
	"erp-golang/internal/modules/hr"

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
	return "payroll"
}

func (m *Module) Service() *Service {
	return m.service
}

func (m *Module) RegisterRoutes(api *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	group := api.Group("/payroll")
	group.Use(authMiddleware)
	{
		// Structure & Rules
		group.GET("/structures", m.handler.ListStructures)
		group.POST("/structures", m.handler.CreateStructure)
		group.GET("/structures/:id", m.handler.GetStructure)
		group.POST("/structures/:id/rules", m.handler.CreateSalaryRule)

		// Payslip Batches (Runs)
		group.GET("/runs", m.handler.ListPayslipRuns)
		group.POST("/runs", m.handler.CreatePayslipRun)
		group.GET("/runs/:id", m.handler.GetPayslipRun)
		group.POST("/runs/:id/generate", m.handler.GenerateRunPayslips)

		// Payslips
		group.GET("/payslips", m.handler.ListPayslips)
		group.POST("/payslips", m.handler.CreatePayslip)
		group.GET("/payslips/:id", m.handler.GetPayslip)
		group.POST("/payslips/:id/compute", m.handler.ComputePayslip)
		group.POST("/payslips/:id/ai-audit", m.handler.AuditPayslipWithAI)
		group.POST("/payslips/:id/confirm", m.handler.ConfirmPayslip)
		group.POST("/payslips/:id/pay", m.handler.PayPayslip)
		group.POST("/payslips/:id/cancel", m.handler.CancelPayslip)

		// Analytics
		group.GET("/stats", m.handler.GetStats)
	}
}

func (m *Module) AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&SalaryRuleCategory{},
		&PayrollStructure{},
		&SalaryRule{},
		&PayslipRun{},
		&Payslip{},
		&PayslipLine{},
	)
}

func (m *Module) Seed(db *gorm.DB) error {
	var count int64
	db.Model(&PayrollStructure{}).Count(&count)
	if count > 0 {
		return nil
	}

	// 1. Seed Categories
	categories := []SalaryRuleCategory{
		{Code: "BASIC", Name: "Gaji Pokok", Description: "Komponen imbalan dasar pekerja"},
		{Code: "ALW", Name: "Tunjangan Karyawan", Description: "Tunjangan jabatan, makan, transport & lembur"},
		{Code: "DED", Name: "Potongan Gaji", Description: "Potongan ketidakhadiran, cuti unpaid & jaminan sosial"},
		{Code: "TAX", Name: "Pajak Penghasilan (PPh 21)", Description: "Pajak penghasilan pasal 21 per UU HPP"},
		{Code: "NET", Name: "Gaji Bersih Take-Home Pay", Description: "Jumlah bersih yang ditransfer ke rekening karyawan"},
	}
	for i := range categories {
		categories[i].CompanyID = 1
		db.Create(&categories[i])
	}

	// 2. Seed Default Standard Salary Structure
	structure := PayrollStructure{
		Name:        "Struktur Standar Karyawan Tetap Indonesia",
		Code:        "IDN_STD",
		Description: "Struktur penggajian standar mengacu regulasi Kemenaker, BPJS Ketenagakerjaan/Kesehatan, dan PPh 21 TER.",
		Active:      true,
	}
	structure.CompanyID = 1
	if err := db.Create(&structure).Error; err != nil {
		return err
	}

	// 3. Seed Default Salary Rules
	rules := []SalaryRule{
		{
			StructureID:       structure.ID,
			Name:              "Gaji Pokok",
			Code:              "BASIC",
			Category:          "BASIC",
			Sequence:          10,
			AmountType:        "fixed",
			DebitAccountCode:  "610200",
			CreditAccountCode: "210200",
			Active:            true,
		},
		{
			StructureID:       structure.ID,
			Name:              "Tunjangan Jabatan & Keahlian",
			Code:              "ALW_JOB",
			Category:          "ALW",
			Sequence:          20,
			AmountType:        "fixed",
			FixedAmount:       1500000,
			DebitAccountCode:  "610300",
			CreditAccountCode: "210200",
			Active:            true,
		},
		{
			StructureID:       structure.ID,
			Name:              "Uang Makan & Transportasi",
			Code:              "ALW_MEAL_TRANS",
			Category:          "ALW",
			Sequence:          30,
			AmountType:        "fixed",
			FixedAmount:       750000,
			DebitAccountCode:  "610300",
			CreditAccountCode: "210200",
			Active:            true,
		},
		{
			StructureID:       structure.ID,
			Name:              "Upah Lembur Presensi (Attendance)",
			Code:              "ALW_OT",
			Category:          "ALW",
			Sequence:          40,
			AmountType:        "dynamic",
			DebitAccountCode:  "610300",
			CreditAccountCode: "210200",
			Active:            true,
		},
		{
			StructureID: structure.ID,
			Name:        "Potongan Cuti Tanpa Gaji (Unpaid Leave)",
			Code:        "DED_UNPAID",
			Category:    "DED",
			Sequence:    50,
			AmountType:  "dynamic",
			Active:      true,
		},
		{
			StructureID:       structure.ID,
			Name:              "Iuran BPJS Ketenagakerjaan (JHT 2% + JP 1%)",
			Code:              "DED_BPJS_TK",
			Category:          "DED",
			Sequence:          60,
			AmountType:        "percentage",
			Percentage:        3.0,
			DebitAccountCode:  "610200",
			CreditAccountCode: "210400",
			Active:            true,
		},
		{
			StructureID:       structure.ID,
			Name:              "Iuran BPJS Kesehatan (1%)",
			Code:              "DED_BPJS_KES",
			Category:          "DED",
			Sequence:          70,
			AmountType:        "percentage",
			Percentage:        1.0,
			DebitAccountCode:  "610200",
			CreditAccountCode: "210400",
			Active:            true,
		},
		{
			StructureID:       structure.ID,
			Name:              "Pajak PPh 21 Tarif Efektif Rata-Rata (TER)",
			Code:              "DED_PPH21",
			Category:          "TAX",
			Sequence:          80,
			AmountType:        "percentage",
			Percentage:        5.0,
			DebitAccountCode:  "610200",
			CreditAccountCode: "210300",
			Active:            true,
		},
		{
			StructureID:       structure.ID,
			Name:              "Gaji Bersih (Take-Home Pay)",
			Code:              "NET",
			Category:          "NET",
			Sequence:          100,
			AmountType:        "dynamic",
			CreditAccountCode: "210200",
			Active:            true,
		},
	}
	for i := range rules {
		rules[i].CompanyID = 1
		db.Create(&rules[i])
	}

	// 4. Seed Demo Payslip Run
	run := PayslipRun{
		Name:            "BATCH/PAY/2026/10",
		Period:          "October 2026",
		DateStart:       "2026-10-01",
		DateEnd:         "2026-10-31",
		State:           "verify",
		TotalEmployees:  1,
		TotalGross:      22250000,
		TotalDeductions: 1912500,
		TotalNet:        20337500,
		Notes:           "Batch Penggajian Reguler Periode Oktober 2026",
	}
	run.CompanyID = 1
	db.Create(&run)

	// 5. Seed Demo Payslip for Employee #1 (Bambang Wijaya)
	var emp hr.Employee
	if err := db.First(&emp).Error; err == nil {
		demoSlip := Payslip{
			Number:          "SLIP/2026/10/00001",
			EmployeeID:      emp.ID,
			StructureID:     structure.ID,
			PayslipRunID:    &run.ID,
			Period:          "October 2026",
			DateFrom:        "2026-10-01",
			DateTo:          "2026-10-31",
			WorkingDays:     22,
			WorkedHours:     176,
			BasicSalary:     emp.BasicSalary, // 20,000,000
			GrossSalary:     22250000,        // Basic 20M + Job 1.5M + Meal 750k
			TotalDeductions: 1912500,         // BPJS TK 600k + BPJS Kes 200k + PPh21 1.112.5k
			NetSalary:       20337500,
			State:           "verify",
			AICheckStatus:   "low",
			AIRiskScore:     10.0,
			AINotes:         "Audit Cloudflare Workers AI: Seluruh komponen gaji pokok, lembur presensi, potongan BPJS, dan PPh 21 berada dalam koridor kepatuhan regulasi ketenagakerjaan (96.5% Compliance).",
			Notes:           "Slip gaji demo sistem siap posting ke GL Buku Besar",
		}
		demoSlip.CompanyID = 1
		db.Create(&demoSlip)

		// Seed Lines
		slipLines := []PayslipLine{
			{PayslipID: demoSlip.ID, Code: "BASIC", Name: "Gaji Pokok", Category: "BASIC", Sequence: 10, Amount: 20000000, Total: 20000000, DebitAccountCode: "610200"},
			{PayslipID: demoSlip.ID, Code: "ALW_JOB", Name: "Tunjangan Jabatan & Keahlian", Category: "ALW", Sequence: 20, Amount: 1500000, Total: 1500000, DebitAccountCode: "610300"},
			{PayslipID: demoSlip.ID, Code: "ALW_MEAL_TRANS", Name: "Uang Makan & Transportasi", Category: "ALW", Sequence: 30, Amount: 750000, Total: 750000, DebitAccountCode: "610300"},
			{PayslipID: demoSlip.ID, Code: "DED_BPJS_TK", Name: "Iuran BPJS Ketenagakerjaan (3%)", Category: "DED", Sequence: 60, Amount: 600000, Total: 600000, CreditAccountCode: "210400"},
			{PayslipID: demoSlip.ID, Code: "DED_BPJS_KES", Name: "Iuran BPJS Kesehatan (1%)", Category: "DED", Sequence: 70, Amount: 200000, Total: 200000, CreditAccountCode: "210400"},
			{PayslipID: demoSlip.ID, Code: "DED_PPH21", Name: "Pajak PPh 21 TER (5%)", Category: "TAX", Sequence: 80, Amount: 1112500, Total: 1112500, CreditAccountCode: "210300"},
			{PayslipID: demoSlip.ID, Code: "NET", Name: "Gaji Bersih (Take-Home Pay)", Category: "NET", Sequence: 100, Amount: 20337500, Total: 20337500, CreditAccountCode: "210200"},
		}
		for i := range slipLines {
			slipLines[i].CompanyID = 1
			db.Create(&slipLines[i])
		}
	}

	log.Println("[Payroll] Seeded default payroll structures, salary rules, and demo October 2026 payslip")
	return nil
}
