package customs_bonded

import (
	"log"
	"time"

	"erp-golang/internal/core/ai"
	"erp-golang/internal/modules/chatter"
	"erp-golang/internal/modules/stock"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Module struct {
	db         *gorm.DB
	repo       Repository
	service    Service
	handler    *Handler
	stockSvc   *stock.Service
	chatterSvc *chatter.Service
	aiClient   *ai.Client
}

func NewModule(db *gorm.DB, stockSvc *stock.Service, chatterSvc *chatter.Service, aiClient *ai.Client) *Module {
	repo := NewRepository(db)
	service := NewService(repo, db, stockSvc, chatterSvc, aiClient)
	handler := NewHandler(service)

	return &Module{
		db:         db,
		repo:       repo,
		service:    service,
		handler:    handler,
		stockSvc:   stockSvc,
		chatterSvc: chatterSvc,
		aiClient:   aiClient,
	}
}

func (m *Module) Name() string {
	return "customs_bonded"
}

func (m *Module) Service() Service {
	return m.service
}

func (m *Module) Repository() Repository {
	return m.repo
}

func (m *Module) AutoMigrate(db *gorm.DB) error {
	m.db = db
	return m.repo.AutoMigrate()
}

func (m *Module) Seed(db *gorm.DB) error {
	var count int64
	db.Model(&CustomsDocument{}).Count(&count)
	if count > 0 {
		return nil
	}

	log.Println("🌱 Seeding Kawasan Berikat & IT Inventory DJBC CEISA 4.0 Sample Data...")

	now := time.Now()
	regDate := now.AddDate(0, 0, -10)
	sppbDate := now.AddDate(0, 0, -9)

	// 1. Dokumen BC 2.3 (Impor Bahan Baku Lembaran Baja dari Nippon Steel Jepang)
	bc23 := CustomsDocument{
		CompanyID:          1,
		DocumentType:       DocTypeBC23,
		RegistrationNumber: "001248",
		RegistrationDate:   &regDate,
		SubmissionNumber:   "000023-000123-20261002-000001",
		SubmissionDate:     regDate,
		CustomsOfficeCode:  "040300", // KPPBC TMP A Bekasi
		PartnerID:          1,
		PartnerName:        "Nippon Steel Corporation (Tokyo, Japan)",
		CountryOfOrigin:    "JP",
		CurrencyCode:       "USD",
		ExchangeRate:       16250,
		TotalCIFUSD:        12500,
		TotalCIFIDR:        12500 * 16250,
		DutyAmount:         10156250, // 5% ditangguhkan
		VATAmount:          23447188, // 11% tidak dipungut
		InvoiceNumber:      "INV-NIPPON-2026-9912",
		BLNumber:           "ONE-TYO-JKT-88912",
		ContainerNumber:    "ONEY8819201",
		SPPBNumber:         "SPPB-040300-2026-001248",
		SPPBDate:           &sppbDate,
		Status:             "approved_sppb",
		Notes:              "Impor bahan baku lembaran baja canai dingin untuk perakitan rak industri (Fasilitas Kawasan Berikat)",
		Lines: []CustomsDocumentLine{
			{
				LineNumber:        1,
				ProductID:         1,
				HSCode:            "7209.16.00",
				ItemDescription:   "Cold-Rolled Steel Sheet Coil 2.0mm",
				ItemCategory:      BookRawMaterial,
				Quantity:          5000,
				UoM:               "KGM",
				UnitPriceUSD:      2.50,
				TotalCIFUSD:       12500,
				DutyTariffPercent: 5.0,
				DutyAmount:        10156250,
				VATAmount:         23447188,
			},
		},
	}
	db.Create(&bc23)

	// Catat Mutasi Buku 1: Bahan Baku
	db.Create(&CustomsITInventoryMutation{
		CompanyID:         1,
		BookCategory:      BookRawMaterial,
		ProductID:         1,
		ItemCode:          "RAW-STEEL-01",
		ItemName:          "Cold-Rolled Steel Sheet Coil 2.0mm",
		UoM:               "KGM",
		MutationDate:      sppbDate,
		CustomsDocumentID: &bc23.ID,
		CustomsDocType:    DocTypeBC23,
		CustomsDocNumber:  "001248",
		ProofNumber:       "SPPB-040300-2026-001248",
		QtyBeginning:      1500,
		QtyIn:             5000,
		QtyOut:            0,
		QtyEnding:         6500,
		PhysicalStockQty:  6500,
		DifferenceQty:     0,
		Notes:             "Pemasukan BC 2.3 dari Pelabuhan Tanjung Priok ke Kawasan Berikat",
	})

	// 2. Dokumen BC 3.0 (Ekspor Barang Jadi Rak Baja Heavy-Duty ke Singapura)
	exportRegDate := now.AddDate(0, 0, -4)
	exportSPPBDate := now.AddDate(0, 0, -3)
	bc30 := CustomsDocument{
		CompanyID:          1,
		DocumentType:       DocTypeBC30,
		RegistrationNumber: "005910",
		RegistrationDate:   &exportRegDate,
		SubmissionNumber:   "000030-000123-20261008-000001",
		SubmissionDate:     exportRegDate,
		CustomsOfficeCode:  "040300",
		PartnerID:          2,
		PartnerName:        "Singa Logistics Hub Pte Ltd (Singapore)",
		CountryOfOrigin:    "ID",
		CurrencyCode:       "USD",
		ExchangeRate:       16250,
		TotalCIFUSD:        35000,
		TotalCIFIDR:        35000 * 16250,
		DutyAmount:         0,
		VATAmount:          0,
		InvoiceNumber:      "EXP-2026-0045",
		BLNumber:           "PIL-JKT-SIN-11928",
		ContainerNumber:    "PCIU9928172",
		SPPBNumber:         "PEB-SPPB-040300-2026-005910",
		SPPBDate:           &exportSPPBDate,
		Status:             "approved_sppb",
		Notes:              "Ekspor 120 unit Industrial Heavy-Duty Storage Rack ke Jurong Port Singapore",
		Lines: []CustomsDocumentLine{
			{
				LineNumber:        1,
				ProductID:         2,
				HSCode:            "9403.20.90",
				ItemDescription:   "Industrial Heavy-Duty Storage Rack S01",
				ItemCategory:      BookFinishedGoods,
				Quantity:          120,
				UoM:               "SET",
				UnitPriceUSD:      291.6667,
				TotalCIFUSD:       35000,
				DutyTariffPercent: 0,
				DutyAmount:        0,
				VATAmount:         0,
			},
		},
	}
	db.Create(&bc30)

	// Catat Mutasi Buku 2: Barang Jadi
	db.Create(&CustomsITInventoryMutation{
		CompanyID:         1,
		BookCategory:      BookFinishedGoods,
		ProductID:         2,
		ItemCode:          "FG-RACK-01",
		ItemName:          "Industrial Heavy-Duty Storage Rack S01",
		UoM:               "SET",
		MutationDate:      exportSPPBDate,
		CustomsDocumentID: &bc30.ID,
		CustomsDocType:    DocTypeBC30,
		CustomsDocNumber:  "005910",
		ProofNumber:       "PEB-SPPB-040300-2026-005910",
		QtyBeginning:      200,
		QtyIn:             0,
		QtyOut:            120,
		QtyEnding:         80,
		PhysicalStockQty:  80,
		DifferenceQty:     0,
		Notes:             "Pengeluaran Ekspor BC 3.0 via Tanjung Priok",
	})

	// 3. Subkontrak Pemotongan & Tekuk Laser (BC 2.61 / 2.62)
	db.Create(&CustomsSubcontract{
		CompanyID:         1,
		ContractNumber:    "SUBK/2026/00001",
		CustomsApprovalNo: "S-541/KBC.040302/2026",
		PartnerID:         3,
		PartnerName:       "PT Metalindo Presisi Utama (Bekasi)",
		ProductID:         1,
		QtySent:           1000,
		QtyReturned:       1000,
		UoM:               "KGM",
		StartDate:         now.AddDate(0, 0, -20),
		DueDate:           now.AddDate(0, 0, 40),
		BC261DocNo:        "BC261-2026-000412",
		BC262DocNo:        "BC262-2026-000488",
		Status:            "returned",
		Notes:             "Pekerjaan jasa bending dan punching lembaran baja plat",
	})

	log.Println("✅ Kawasan Berikat Initial Seeds Complete.")
	return nil
}

func (m *Module) RegisterRoutes(r *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	m.handler.RegisterRoutes(r, authMiddleware)
}
