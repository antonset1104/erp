package mrp

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
	return "mrp"
}

func (m *Module) AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&MrpBom{},
		&MrpBomLine{},
		&MrpProduction{},
	)
}

func (m *Module) RegisterRoutes(api *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	group := api.Group("/mrp")
	group.Use(authMiddleware)
	{
		group.GET("/boms", m.handler.ListBoms)
		group.POST("/boms", m.handler.CreateBom)
		group.GET("/productions", m.handler.ListProductions)
		group.POST("/productions", m.handler.CreateProduction)
		group.POST("/productions/:id/produce", m.handler.ActionProduce)
	}
}

func (m *Module) Seed(db *gorm.DB) error {
	var count int64
	db.Model(&MrpBom{}).Count(&count)
	if count == 0 {
		// Sample BOM: "Custom Executive Workstation PC"
		// Made of: 1 Laptop/Core Unit (Product 1) + 1 Mouse (Product 2)
		bom := MrpBom{
			Code:        "BOM/WORKSTATION/01",
			ProductID:   1, // Output Product
			ProductQty:  1.0,
			Description: "Executive Workstation Bundle Assembly",
			Lines: []MrpBomLine{
				{
					ProductID:   2, // Mouse
					ProductName: "Wireless Ergonomic Mouse",
					ProductQty:  1.0,
				},
				{
					ProductID:   3, // Paper Rim
					ProductName: "Kertas HVS A4 80gr Rim",
					ProductQty:  2.0,
				},
			},
		}
		bom.CompanyID = 1
		for i := range bom.Lines {
			bom.Lines[i].CompanyID = 1
		}
		db.Create(&bom)

		// Seed sample MO
		mo := MrpProduction{
			Name:        "MO/2026/00001",
			ProductID:   1,
			BomID:       bom.ID,
			ProductQty:  2.0,
			State:       "confirmed",
			DatePlanned: "2026-10-04 14:00:00",
			Note:        "Standard assembly for showroom display",
		}
		mo.CompanyID = 1
		db.Create(&mo)

		log.Println("[MRP] Seeded sample Bill of Materials (BOM) & Manufacturing Order (MO)")
	}
	return nil
}
