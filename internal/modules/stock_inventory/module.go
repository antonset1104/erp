package stock_inventory

import (
	"log"

	"erp-golang/internal/core/ai"
	"erp-golang/internal/modules/chatter"
	"erp-golang/internal/modules/product"
	"erp-golang/internal/modules/stock"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Module struct {
	db      *gorm.DB
	svc     *Service
	handler *Handler
}

func NewModule(db *gorm.DB, chatterSvc *chatter.Service, aiClient *ai.Client) *Module {
	repo := NewRepository(db)
	svc := NewService(db, repo, chatterSvc, aiClient)
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
	return "stock_inventory"
}

func (m *Module) AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&StockInventory{},
		&StockInventoryLine{},
	)
}

func (m *Module) RegisterRoutes(api *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	group := api.Group("/stock-inventory")
	group.Use(authMiddleware)
	{
		group.GET("/inventories", m.handler.ListInventories)
		group.POST("/inventories", m.handler.CreateInventory)
		group.GET("/inventories/:id", m.handler.GetInventory)
		group.POST("/inventories/:id/start", m.handler.StartInventory)
		group.PUT("/inventories/:id/lines/:line_id", m.handler.UpdateLine)
		group.POST("/inventories/:id/lines", m.handler.AddLine)
		group.POST("/inventories/:id/ai-audit", m.handler.AuditWithAI)
		group.POST("/inventories/:id/validate", m.handler.ValidateInventory)
		group.POST("/inventories/:id/cancel", m.handler.CancelInventory)
		group.GET("/stats", m.handler.GetStats)
	}
}

func (m *Module) Seed(db *gorm.DB) error {
	var count int64
	db.Model(&StockInventory{}).Count(&count)
	if count > 0 {
		return nil
	}

	// Find WH/Stock location and WH-JKT warehouse
	var loc stock.StockLocation
	if err := db.Where("name = ?", "WH/Stock").First(&loc).Error; err != nil {
		return nil
	}

	var wh stock.StockWarehouse
	db.Where("code = ?", "WH-JKT").First(&wh)

	// Fetch sample products
	var prods []product.Product
	db.Where("type = 'product'").Limit(3).Find(&prods)
	if len(prods) == 0 {
		return nil
	}

	inv := StockInventory{
		Name:                "OPNAME/2026/00001",
		WarehouseID:         &wh.ID,
		LocationID:          loc.ID,
		Date:                "2026-10-08",
		State:               "confirm",
		TotalTheoreticalQty: 0,
		TotalCountedQty:     0,
		TotalDifferenceQty:  0,
		TotalLossValue:      0,
		TotalGainValue:      0,
		AIRiskLevel:         "medium",
		AIAuditSummary:      "Audit awal mendeteksi 1 item selisih kurang pada produk komponen elektronik berharga tinggi. Disarankan investigasi rekaman CCTV dan verifikasi picking pengiriman.",
		Notes:               "Stock Opname Rutin Bulanan Q4 Gudang Utama Pulogadung",
	}
	inv.CompanyID = 1

	if err := db.Create(&inv).Error; err != nil {
		return err
	}

	// Create sample lines
	var totalTheo, totalCounted, totalDiff, totalLoss, totalGain float64
	for i, p := range prods {
		theo := float64(20 + i*10)
		counted := theo
		diff := float64(0)
		reason := "Sesuai fisik rak"
		if i == 0 {
			// Deficit 1 unit (Shrinkage)
			counted = theo - 1
			diff = -1
			reason = "1 unit retak fisik/kemasan rusak di rak B2"
		} else if i == 1 {
			// Surplus 2 units
			counted = theo + 2
			diff = 2
			reason = "2 unit bonus promo vendor belum tercatat di sistem"
		}

		cost := p.StandardPrice
		if cost == 0 {
			cost = 500000
		}
		varVal := diff * cost

		totalTheo += theo
		totalCounted += counted
		totalDiff += diff
		if varVal < 0 {
			totalLoss += -varVal
		} else {
			totalGain += varVal
		}

		line := StockInventoryLine{
			InventoryID:    inv.ID,
			ProductID:      p.ID,
			TheoreticalQty: theo,
			CountedQty:     counted,
			DifferenceQty:  diff,
			CostPrice:      cost,
			VarianceValue:  varVal,
			Reason:         reason,
			State:          "draft",
		}
		line.CompanyID = 1
		db.Create(&line)
	}

	inv.TotalTheoreticalQty = totalTheo
	inv.TotalCountedQty = totalCounted
	inv.TotalDifferenceQty = totalDiff
	inv.TotalLossValue = totalLoss
	inv.TotalGainValue = totalGain
	db.Save(&inv)

	log.Printf("[Stock Inventory] Seeded demo stock opname '%s' with %d item lines\n", inv.Name, len(prods))
	return nil
}
