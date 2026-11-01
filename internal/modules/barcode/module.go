package barcode

import (
	"log"

	"erp-golang/internal/modules/product"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Module struct {
	service *Service
	handler *Handler
}

func NewModule(db *gorm.DB) *Module {
	repo := NewRepository(db)
	svc := NewService(db, repo)
	h := NewHandler(svc)
	return &Module{
		service: svc,
		handler: h,
	}
}

func (m *Module) Name() string {
	return "barcode"
}

func (m *Module) Service() *Service {
	return m.service
}

func (m *Module) RegisterRoutes(api *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	group := api.Group("/barcode")
	group.Use(authMiddleware)
	{
		// Universal Scanner & Lookup
		group.POST("/scan", m.handler.Scan)
		group.GET("/lookup/:code", m.handler.Lookup)
		group.GET("/label/:type/:id", m.handler.GenerateLabel)

		// Lot & Serial Numbers
		group.GET("/lots", m.handler.ListLots)
		group.POST("/lots", m.handler.CreateLot)
		group.GET("/lots/:id", m.handler.GetLot)

		// Analytics
		group.GET("/stats", m.handler.GetStats)
	}
}

func (m *Module) AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&StockProductionLot{})
}

func (m *Module) Seed(db *gorm.DB) error {
	// 1. Ensure existing products have clean barcodes for demo scanning
	var prods []product.Product
	db.Find(&prods)
	for i, p := range prods {
		if p.Barcode == "" {
			p.Barcode = "89900010000" + string('1'+rune(i%9))
			db.Save(&p)
		}
	}

	// 2. Seed demo Lot/Serial Numbers if empty
	var countLots int64
	db.Model(&StockProductionLot{}).Count(&countLots)
	if countLots == 0 && len(prods) > 0 {
		firstProdID := prods[0].ID
		lots := []StockProductionLot{
			{
				Name:           "LOT-2026-ELC-001",
				ProductID:      firstProdID,
				Barcode:        "LOT2026ELC001",
				Ref:            "PO-BATCH-OCT-2026-A",
				ExpirationDate: "2028-12-31",
				AlertDate:      "2028-06-30",
				Qty:            50,
				Notes:          "Batch pengiriman pabrik perakitan pertama",
			},
			{
				Name:           "LOT-2026-ELC-002",
				ProductID:      firstProdID,
				Barcode:        "LOT2026ELC002",
				Ref:            "PO-BATCH-OCT-2026-B",
				ExpirationDate: "2029-03-31",
				AlertDate:      "2028-09-30",
				Qty:            100,
				Notes:          "Batch pengadaan komponen impor Q4",
			},
		}
		for i := range lots {
			lots[i].CompanyID = 1
			db.Create(&lots[i])
		}
		log.Println("[Barcode] Berhasil seeding 2 demo Lot/Serial Numbers")
	}

	return nil
}
