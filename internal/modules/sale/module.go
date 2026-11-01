package sale

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
	return "sale"
}

func (m *Module) AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&SaleOrder{},
		&SaleOrderLine{},
	)
}

func (m *Module) RegisterRoutes(api *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	group := api.Group("/sales")
	group.Use(authMiddleware)
	{
		group.GET("/orders", m.handler.ListOrders)
		group.GET("/orders/export", m.handler.ExportSalesCSV)
		group.POST("/orders", m.handler.CreateOrder)
		group.GET("/orders/:id", m.handler.GetOrder)
		group.POST("/orders/:id/confirm", m.handler.ActionConfirm)
	}
}

func (m *Module) Seed(db *gorm.DB) error {
	var count int64
	db.Model(&SaleOrder{}).Count(&count)
	if count == 0 {
		// Sample quotation
		so := SaleOrder{
			Name:          "SO00001",
			PartnerID:     2, // PT Maju Bersama Makmur
			DateOrder:     "2026-10-04 10:00:00",
			State:         "draft",
			TaxRate:       11.0,
			AmountUntaxed: 49000000,
			AmountTax:     5390000,
			AmountTotal:   54390000,
			OrderLines: []SaleOrderLine{
				{
					ProductID:     1, // Laptop Dell XPS 15
					Name:          "Laptop Dell XPS 15",
					ProductUomQty: 2,
					PriceUnit:     24500000,
					PriceSubtotal: 49000000,
					PriceTotal:    54390000,
				},
			},
			Note: "Deliver to Sudirman branch within 3 business days",
		}
		so.CompanyID = 1
		for i := range so.OrderLines {
			so.OrderLines[i].CompanyID = 1
		}
		db.Create(&so)
		log.Println("[Sale] Seeded sample quotation SO00001")
	}
	return nil
}
