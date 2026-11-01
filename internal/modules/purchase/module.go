package purchase

import (
	"log"

	"erp-golang/internal/modules/stock"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Module struct {
	db       *gorm.DB
	stockSvc *stock.Service
	svc      *Service
	handler  *Handler
}

func NewModule(db *gorm.DB, stockSvc *stock.Service) *Module {
	svc := NewService(db, stockSvc)
	handler := NewHandler(svc)
	return &Module{
		db:       db,
		stockSvc: stockSvc,
		svc:      svc,
		handler:  handler,
	}
}

func (m *Module) Service() *Service {
	return m.svc
}

func (m *Module) Name() string {
	return "purchase"
}

func (m *Module) AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&PurchaseOrder{},
		&PurchaseOrderLine{},
	)
}

func (m *Module) RegisterRoutes(api *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	group := api.Group("/purchases")
	group.Use(authMiddleware)
	{
		group.GET("/orders", m.handler.ListOrders)
		group.POST("/orders", m.handler.CreateOrder)
		group.GET("/orders/:id", m.handler.GetOrder)
		group.POST("/orders/:id/confirm", m.handler.ActionConfirm)
		group.POST("/reordering/run", m.handler.RunReorderingRules)
	}
}

func (m *Module) Seed(db *gorm.DB) error {
	var count int64
	db.Model(&PurchaseOrder{}).Count(&count)
	if count == 0 {
		po := PurchaseOrder{
			Name:          "PO00001",
			PartnerID:     3, // CV Sumber Jaya Teknik
			DateOrder:     "2026-10-04 09:30:00",
			State:         "draft",
			TaxRate:       11.0,
			AmountUntaxed: 19000000,
			AmountTax:     2090000,
			AmountTotal:   21090000,
			OrderLines: []PurchaseOrderLine{
				{
					ProductID:     1, // Laptop Dell XPS 15
					Name:          "Laptop Dell XPS 15 - Vendor Purchase",
					ProductUomQty: 1,
					PriceUnit:     19000000,
					PriceSubtotal: 19000000,
					PriceTotal:    21090000,
				},
			},
			Note: "Standard PO for restock",
		}
		po.CompanyID = 1
		for i := range po.OrderLines {
			po.OrderLines[i].CompanyID = 1
		}
		db.Create(&po)
		log.Println("[Purchase] Seeded sample RFQ PO00001")
	}
	return nil
}
