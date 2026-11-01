package account

import (
	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Module struct {
	db      *gorm.DB
	svc     *Service
	handler *Handler
}

func NewModule(db *gorm.DB) *Module {
	svc := NewService(db)
	handler := NewHandler(svc)
	return &Module{
		db:      db,
		svc:     svc,
		handler: handler,
	}
}

func (m *Module) Service() *Service {
	return m.svc
}

func (m *Module) Name() string {
	return "account"
}

func (m *Module) AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&AccountAccount{},
		&AccountJournal{},
		&AccountMove{},
		&AccountMoveLine{},
		&AccountPayment{},
	)
}

func (m *Module) RegisterRoutes(api *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	group := api.Group("/accounting")
	group.Use(authMiddleware)
	{
		group.GET("/moves", m.handler.ListMoves)
		group.POST("/moves", m.handler.CreateMove)
		group.GET("/moves/:id", m.handler.GetMove)
		group.POST("/moves/:id/post", m.handler.ActionPost)
		group.POST("/moves/:id/pay", m.handler.RegisterPayment)

		group.GET("/accounts", m.handler.ListAccounts)
		group.GET("/journals", m.handler.ListJournals)

		group.GET("/reports/profit-loss", m.handler.GetProfitAndLoss)
		group.GET("/reports/balance-sheet", m.handler.GetBalanceSheet)
		group.GET("/reports/trial-balance", m.handler.GetTrialBalance)
		group.GET("/reports/export/trial-balance", m.handler.ExportTrialBalanceCSV)
	}

	// Printable invoice document view
	api.GET("/accounting/moves/:id/print", m.handler.PrintMove)
}

func (m *Module) Seed(db *gorm.DB) error {
	// Seed Standard Chart of Accounts
	// Seed Standard Chart of Accounts
	accounts := []AccountAccount{
		{Code: "110100", Name: "Kas Utama (Cash)", Type: "asset_cash"},
		{Code: "110200", Name: "Bank BCA Operasional", Type: "asset_cash"},
		{Code: "120000", Name: "Piutang Usaha (Accounts Receivable)", Type: "asset_receivable"},
		{Code: "130000", Name: "Persediaan Barang Dagang (Stock Valuation)", Type: "asset_cash"},
		{Code: "210000", Name: "Hutang Usaha (Accounts Payable)", Type: "liability_payable"},
		{Code: "210100", Name: "Hutang PPN Keluaran (Tax Payable)", Type: "liability_payable"},
		{Code: "410000", Name: "Pendapatan Penjualan (Product Sales)", Type: "income"},
		{Code: "510000", Name: "Harga Pokok Penjualan (COGS)", Type: "expense"},
		{Code: "150100", Name: "Aset Tetap Peralatan & Mesin", Type: "asset_cash"},
		{Code: "150900", Name: "Akumulasi Penyusutan Peralatan", Type: "asset_cash"},
		{Code: "610100", Name: "Beban Penyusutan Aset Tetap", Type: "expense"},
		{Code: "610200", Name: "Beban Gaji & Upah Pokok Karyawan", Type: "expense"},
		{Code: "610300", Name: "Beban Tunjangan & Lembur Karyawan", Type: "expense"},
		{Code: "210200", Name: "Hutang Gaji & Upah Karyawan", Type: "liability_payable"},
		{Code: "210300", Name: "Hutang Pajak PPh Pasal 21", Type: "liability_payable"},
		{Code: "210400", Name: "Hutang Iuran BPJS Ketenagakerjaan & Kesehatan", Type: "liability_payable"},
		{Code: "620700", Name: "Beban Selisih Kurang Stok (Inventory Shrinkage/Loss)", Type: "expense"},
		{Code: "710200", Name: "Pendapatan Selisih Lebih Stok (Inventory Gain)", Type: "income"},
	}

	for i := range accounts {
		var existing AccountAccount
		if err := db.Where("code = ?", accounts[i].Code).First(&existing).Error; err != nil {
			accounts[i].CompanyID = 1
			db.Create(&accounts[i])
		}
	}

	// Seed Standard Journals
	var jrnCount int64
	db.Model(&AccountJournal{}).Count(&jrnCount)
	if jrnCount == 0 {
		journals := []AccountJournal{
			{Code: "INV", Name: "Customer Invoices", Type: "sale", DefaultAccountID: 7}, // Sales
			{Code: "BILL", Name: "Vendor Bills", Type: "purchase", DefaultAccountID: 8},  // COGS/Expense
			{Code: "BNK", Name: "Bank BCA", Type: "bank", DefaultAccountID: 2},
			{Code: "CSH", Name: "Cash", Type: "cash", DefaultAccountID: 1},
			{Code: "MISC", Name: "Miscellaneous Operations", Type: "general", DefaultAccountID: 1},
		}

		for i := range journals {
			journals[i].CompanyID = 1
			db.Create(&journals[i])
		}
		log.Println("[Account] Seeded default financial journals (INV, BILL, BNK, CSH, MISC)")

		// Seed a sample posted invoice
		sampleMove := AccountMove{
			Name:           "INV/2026/00001",
			MoveType:       "out_invoice",
			JournalID:      1, // INV
			PartnerID:      2, // PT Maju Bersama Makmur
			InvoiceDate:    "2026-10-04",
			InvoiceDueDate: "2026-11-04",
			State:          "posted",
			PaymentState:   "not_paid",
			AmountUntaxed:  24500000,
			AmountTax:      2695000,
			AmountTotal:    27195000,
			AmountResidual: 27195000,
			Ref:            "SO00001",
			Note:           "Tagihan atas pemesanan Laptop Dell XPS 15",
			LineIDs: []AccountMoveLine{
				{
					AccountID: 3, // Piutang Usaha
					Name:      "Piutang Penjualan - PT Maju Bersama Makmur",
					Debit:     27195000,
					Credit:    0,
					Balance:   27195000,
				},
				{
					AccountID: 7, // Pendapatan Penjualan
					Name:      "Laptop Dell XPS 15",
					Debit:     0,
					Credit:    24500000,
					Balance:   -24500000,
				},
				{
					AccountID: 6, // PPN Keluaran
					Name:      "PPN 11%",
					Debit:     0,
					Credit:    2695000,
					Balance:   -2695000,
				},
			},
		}
		sampleMove.CompanyID = 1
		for i := range sampleMove.LineIDs {
			sampleMove.LineIDs[i].CompanyID = 1
		}
		db.Create(&sampleMove)
		log.Println("[Account] Seeded sample posted customer invoice INV/2026/00001")
	}

	return nil
}
