package attachment

import (
	"encoding/json"
	"log"
	"time"

	"erp-golang/internal/core/ai"
	"erp-golang/internal/core/storage"
	"erp-golang/internal/modules/chatter"
	"erp-golang/internal/modules/expense"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Module struct {
	db         *gorm.DB
	svc        *Service
	handler    *Handler
	storage    storage.Provider
	aiClient   *ai.Client
	chatterSvc *chatter.Service
	expenseSvc *expense.Service
}

func NewModule(
	db *gorm.DB,
	storageProvider storage.Provider,
	aiClient *ai.Client,
	chatterSvc *chatter.Service,
	expenseSvc *expense.Service,
) *Module {
	repo := NewRepository(db)
	svc := NewService(db, repo, storageProvider, aiClient, chatterSvc, expenseSvc)
	handler := NewHandler(svc)

	return &Module{
		db:         db,
		svc:        svc,
		handler:    handler,
		storage:    storageProvider,
		aiClient:   aiClient,
		chatterSvc: chatterSvc,
		expenseSvc: expenseSvc,
	}
}

func (m *Module) Service() *Service {
	return m.svc
}

func (m *Module) Name() string {
	return "attachment"
}

func (m *Module) AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&Attachment{})
}

func (m *Module) RegisterRoutes(api *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	group := api.Group("/attachments")
	group.Use(authMiddleware)
	{
		group.POST("/upload", m.handler.Upload)
		group.GET("", m.handler.List)
		group.GET("/stats", m.handler.GetStats)
		group.GET("/:id", m.handler.Get)
		group.GET("/:id/download", m.handler.Download)
		group.DELETE("/:id", m.handler.Delete)
		group.POST("/:id/ocr", m.handler.RunOCR)
		group.POST("/:id/create-expense", m.handler.CreateExpenseFromReceipt)
	}
}

func (m *Module) Seed(db *gorm.DB) error {
	var count int64
	db.Model(&Attachment{}).Count(&count)
	if count > 0 {
		return nil
	}

	sampleOCR := ai.ReceiptOCRResult{
		VendorName:         "SPBU Pertamina Pasti Pas 34-12345",
		InvoiceDate:        time.Now().Format("2006-01-02"),
		TotalAmount:        385000.0,
		TaxAmount:          38500.0,
		Currency:           "IDR",
		CategorySuggestion: "Bahan Bakar & Transportasi",
		LineItems: []ai.ReceiptLineItem{
			{Description: "BBM Dexlite (Solar)", Qty: 25.0, UnitPrice: 13860.0, Subtotal: 346500.0},
			{Description: "PPN Masukan (11%)", Qty: 1.0, UnitPrice: 38500.0, Subtotal: 38500.0},
		},
		ConfidenceScore:  0.96,
		RawExtractedText: "SPBU Pertamina 34-12345. Dexlite Solar 25L. PPN 11%. Total: Rp 385.000",
		Source:           "local_heuristic_ocr",
	}
	ocrJSON, _ := json.Marshal(sampleOCR)

	sampleDocs := []Attachment{
		{
			Name:          "SOP_Pengadaan_dan_Otorisasi_2026.pdf",
			ResModel:      "document",
			ResID:         0,
			ResName:       "SOP/CORP/2026/01",
			MimeType:      "application/pdf",
			FileSize:      245600,
			StorageType:   m.storage.Type(),
			StoragePath:   "seed/SOP_Pengadaan_2026.pdf",
			FileURL:       "/api/v1/attachments/1/download",
			Checksum:      "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			Description:   "Pedoman Standar Operasional Prosedur Pengadaan Barang, Capex, dan Delegasi Wewenang Direksi 2026",
			OCRStatus:     "none",
			CompanyID:     1,
			CreatedByID:   1,
			CreatedByName: "Administrator ERP",
		},
		{
			Name:          "Kwitansi_BBM_Solar_Dexlite_Armada.jpg",
			ResModel:      "hr.expense",
			ResID:         1,
			ResName:       "EXP/2026/00001",
			MimeType:      "image/jpeg",
			FileSize:      154200,
			StorageType:   m.storage.Type(),
			StoragePath:   "seed/Kwitansi_BBM_Solar.jpg",
			FileURL:       "/api/v1/attachments/2/download",
			Checksum:      "ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb",
			Description:   "Struk pembelian bahan bakar solar Dexlite pengiriman ekspedisi logistik Canter",
			OCRStatus:     "processed",
			OCRData:       string(ocrJSON),
			CompanyID:     1,
			CreatedByID:   1,
			CreatedByName: "Administrator ERP",
		},
	}

	for _, doc := range sampleDocs {
		if err := db.Create(&doc).Error; err != nil {
			log.Printf("[Attachment] Warning seeding document %s: %v", doc.Name, err)
		}
	}

	log.Printf("[Attachment] Initialized Attachment Hub & R2 Document Storage (Seeded %d sample enterprise documents)", len(sampleDocs))
	return nil
}
