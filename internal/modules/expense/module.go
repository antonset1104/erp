package expense

import (
	"log"

	"erp-golang/internal/modules/account"
	"erp-golang/internal/modules/chatter"
	"erp-golang/internal/modules/fleet"
	"erp-golang/internal/modules/hr"
	"erp-golang/internal/modules/project"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Module struct {
	db         *gorm.DB
	svc        *Service
	handler    *Handler
	accountSvc *account.Service
	chatterSvc *chatter.Service
}

func NewModule(db *gorm.DB, accountSvc *account.Service, chatterSvc *chatter.Service) *Module {
	svc := NewService(db, accountSvc, chatterSvc)
	handler := NewHandler(svc)
	return &Module{
		db:         db,
		svc:        svc,
		handler:    handler,
		accountSvc: accountSvc,
		chatterSvc: chatterSvc,
	}
}

func (m *Module) Service() *Service {
	return m.svc
}

func (m *Module) Name() string {
	return "expense"
}

func (m *Module) AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&ExpenseCategory{},
		&Expense{},
		&ExpenseSheet{},
	)
}

func (m *Module) RegisterRoutes(api *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	group := api.Group("/expenses")
	group.Use(authMiddleware)
	{
		// Categories
		group.GET("/categories", m.handler.ListCategories)
		group.POST("/categories", m.handler.CreateCategory)

		// Items (hr.expense)
		group.GET("/items", m.handler.ListExpenses)
		group.POST("/items", m.handler.CreateExpense)
		group.GET("/items/:id", m.handler.GetExpense)
		group.PUT("/items/:id", m.handler.UpdateExpense)
		group.DELETE("/items/:id", m.handler.DeleteExpense)

		// Sheets (hr.expense.sheet)
		group.GET("/sheets", m.handler.ListExpenseSheets)
		group.POST("/sheets", m.handler.CreateExpenseSheet)
		group.GET("/sheets/:id", m.handler.GetExpenseSheet)
		group.POST("/sheets/:id/attach", m.handler.AttachExpenses)
		group.DELETE("/sheets/:id/items/:expense_id", m.handler.RemoveExpense)
		group.PUT("/sheets/:id/submit", m.handler.ActionSubmitSheet)
		group.PUT("/sheets/:id/approve", m.handler.ActionApproveSheet)
		group.POST("/sheets/:id/post", m.handler.ActionPostSheet)
		group.PUT("/sheets/:id/pay", m.handler.ActionPaySheet)
		group.PUT("/sheets/:id/refuse", m.handler.ActionRefuseSheet)

		// Analytics
		group.GET("/stats", m.handler.GetExpenseStats)
	}
}

func (m *Module) Seed(db *gorm.DB) error {
	// 1. Ensure Standard Operational Expense Accounts exist in General Ledger Chart of Accounts
	expenseAccounts := []account.AccountAccount{
		{Code: "620100", Name: "Beban Perjalanan Dinas & Akomodasi", Type: "expense"},
		{Code: "620200", Name: "Beban Konsumsi & Representasi Tamu", Type: "expense"},
		{Code: "620300", Name: "Beban Perlengkapan Kantor & ATK", Type: "expense"},
		{Code: "620400", Name: "Beban Operasional Armada, BBM & Tol", Type: "expense"},
		{Code: "620500", Name: "Beban Pulsa, Paket Data & Internet", Type: "expense"},
		{Code: "620600", Name: "Beban Pelatihan & Sertifikasi Karyawan", Type: "expense"},
	}

	for i := range expenseAccounts {
		var existing account.AccountAccount
		if err := db.Where("code = ?", expenseAccounts[i].Code).First(&existing).Error; err != nil {
			expenseAccounts[i].CompanyID = 1
			db.Create(&expenseAccounts[i])
		}
	}

	// 2. Seed Expense Categories
	var catCount int64
	db.Model(&ExpenseCategory{}).Count(&catCount)
	if catCount == 0 {
		categories := []ExpenseCategory{
			{Name: "Perjalanan Dinas & Tiket", Code: "TRAVEL", AccountCode: "620100", Color: "#3b82f6", Description: "Biaya tiket kereta/pesawat, penginapan hotel dinas luar kota"},
			{Name: "Konsumsi & Representasi", Code: "MEAL", AccountCode: "620200", Color: "#f59e0b", Description: "Makan siang dinas luar kantor dan jamuan meeting bersama klien"},
			{Name: "Bahan Bakar & Tol Armada", Code: "FUEL", AccountCode: "620400", Color: "#ef4444", Description: "Pengisian BBM solar/bensin dan tarif tol kendaraan operasional"},
			{Name: "Perlengkapan Kantor & ATK", Code: "OFFICE", AccountCode: "620300", Color: "#10b981", Description: "Alat tulis kantor, kertas rim, materai dan printer toner"},
			{Name: "Pulsa & Komunikasi", Code: "COMM", AccountCode: "620500", Color: "#8b5cf6", Description: "Paket kuota data internet dan pulsa telepon komunikasi bisnis"},
			{Name: "Pelatihan & Kursus", Code: "TRAINING", AccountCode: "620600", Color: "#06b6d4", Description: "Biaya sertifikasi teknis, seminar, workshop dan training SDM"},
		}

		for i := range categories {
			categories[i].CompanyID = 1
			db.Create(&categories[i])
		}
		log.Println("[Expense] Seeded 6 default expense categories")
	}

	// 3. Seed Sample Expenses & Sheet if empty
	var expCount int64
	db.Model(&Expense{}).Count(&expCount)
	if expCount == 0 {
		var emp hr.Employee
		if err := db.First(&emp).Error; err != nil {
			// No employees yet
			return nil
		}

		var catTravel, catMeal, catFuel ExpenseCategory
		db.Where("code = ?", "TRAVEL").First(&catTravel)
		db.Where("code = ?", "MEAL").First(&catMeal)
		db.Where("code = ?", "FUEL").First(&catFuel)

		var prj project.Project
		hasProject := db.First(&prj).Error == nil

		var vhc fleet.FleetVehicle
		hasVehicle := db.First(&vhc).Error == nil

		var pProjectID *uint
		if hasProject {
			pProjectID = &prj.ID
		}
		var pVehicleID *uint
		if hasVehicle {
			pVehicleID = &vhc.ID
		}

		// Create Sample Expenses
		exp1 := Expense{
			ExpenseNumber: "EXP-ITM/2026/00001",
			Name:          "Hotel & Tiket Kereta Dinas Surabaya",
			Date:          "2026-10-02",
			EmployeeID:    emp.ID,
			CategoryID:    catTravel.ID,
			PaymentMode:   "own_account",
			UnitPrice:     1750000,
			Quantity:      1,
			TotalAmount:   1750000,
			ReceiptRef:    "INV-KA/99214-ASTON",
			ProjectID:     pProjectID,
			State:         "approved",
			Notes:         "Perjalanan dinas 2 hari presentasi implementasi sistem",
		}
		exp1.CompanyID = 1
		db.Create(&exp1)

		exp2 := Expense{
			ExpenseNumber: "EXP-ITM/2026/00002",
			Name:          "Jamuan Makan Siang Meeting Klien PT Maju",
			Date:          "2026-10-03",
			EmployeeID:    emp.ID,
			CategoryID:    catMeal.ID,
			PaymentMode:   "own_account",
			UnitPrice:     450000,
			Quantity:      1,
			TotalAmount:   450000,
			ReceiptRef:    "STRUK-RESTO-8812",
			ProjectID:     pProjectID,
			State:         "approved",
			Notes:         "Makan siang penandatanganan addendum kontrak",
		}
		exp2.CompanyID = 1
		db.Create(&exp2)

		exp3 := Expense{
			ExpenseNumber: "EXP-ITM/2026/00003",
			Name:          "BBM Solar Truk Ekspedisi B 1024 TKO",
			Date:          "2026-10-04",
			EmployeeID:    emp.ID,
			CategoryID:    catFuel.ID,
			PaymentMode:   "own_account",
			UnitPrice:     350000,
			Quantity:      1,
			TotalAmount:   350000,
			ReceiptRef:    "SPBU-SHELL-77192",
			VehicleID:     pVehicleID,
			State:         "draft",
			Notes:         "Pengisian solar perjalanan rute Jakarta - Karawang",
		}
		exp3.CompanyID = 1
		db.Create(&exp3)

		// Create Sample Approved & Posted Expense Sheet for exp1 and exp2
		sheet := ExpenseSheet{
			SheetNumber:    "EXP/2026/00001",
			Name:           "Klaim Reimbursement Dinas Surabaya & Client Meeting",
			EmployeeID:     emp.ID,
			AccountingDate: "2026-10-04",
			PaymentMode:    "own_account",
			TotalAmount:    2200000,
			State:          "approve",
			Notes:          "Lengkap dengan bukti boarding pass tiket dan struk restoran",
		}
		sheet.CompanyID = 1
		db.Create(&sheet)

		// Attach exp1 and exp2 to sheet
		db.Model(&Expense{}).Where("id IN ?", []uint{exp1.ID, exp2.ID}).
			Updates(map[string]interface{}{
				"sheet_id": sheet.ID,
				"state":    "approved",
			})

		// Post sheet to GL automatically if accountSvc is available
		if m.accountSvc != nil {
			_, _, _ = m.svc.ActionPostSheet(sheet.ID, 1, 1)
		}

		log.Println("[Expense] Seeded initial sample expenses and posted expense report EXP/2026/00001")
	}

	return nil
}
