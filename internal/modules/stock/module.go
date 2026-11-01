package stock

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
	return "stock"
}

func (m *Module) AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&StockWarehouse{},
		&StockLocation{},
		&StockPicking{},
		&StockMove{},
		&StockQuant{},
		&StockOrderpoint{},
	)
}

func (m *Module) RegisterRoutes(api *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	group := api.Group("/stock")
	group.Use(authMiddleware)
	{
		group.GET("/pickings", m.handler.ListPickings)
		group.POST("/pickings", m.handler.CreatePicking)
		group.GET("/pickings/:id", m.handler.GetPicking)
		group.POST("/pickings/:id/validate", m.handler.ValidatePicking)

		group.GET("/warehouses", m.handler.ListWarehouses)
		group.POST("/warehouses", m.handler.CreateWarehouse)
		group.GET("/warehouses/:id", m.handler.GetWarehouse)
		group.POST("/transfers", m.handler.CreateInternalTransfer)

		group.GET("/quants", m.handler.ListQuants)
		group.GET("/locations", m.handler.ListLocations)

		group.GET("/orderpoints", m.handler.ListOrderpoints)
		group.POST("/orderpoints", m.handler.CreateOrderpoint)
	}

	// Printable delivery slip / receipt document view
	api.GET("/stock/pickings/:id/print", m.handler.PrintPicking)
}

func (m *Module) Seed(db *gorm.DB) error {
	locations := []StockLocation{
		{
			Name:         "WH/Stock",
			Usage:        "internal",
			CompleteName: "WH/Stock (Gudang Utama Jakarta)",
		},
		{
			Name:         "WH-SBY/Stock",
			Usage:        "internal",
			CompleteName: "WH-SBY/Stock (Hub Regional Surabaya)",
		},
		{
			Name:         "WH-BDG/Stock",
			Usage:        "internal",
			CompleteName: "WH-BDG/Stock (Distribution Center Bandung)",
		},
		{
			Name:         "Virtual Locations/Vendors",
			Usage:        "supplier",
			CompleteName: "Virtual Locations/Vendors",
		},
		{
			Name:         "Virtual Locations/Customers",
			Usage:        "customer",
			CompleteName: "Virtual Locations/Customers",
		},
		{
			Name:         "Virtual Locations/Inventory Adjustment",
			Usage:        "inventory",
			CompleteName: "Virtual Locations/Inventory Adjustment",
		},
		{
			Name:         "Virtual Locations/Inter-Warehouse Transit",
			Usage:        "transit",
			CompleteName: "Virtual Locations/Inter-Warehouse Transit",
		},
	}

	for i := range locations {
		var existing StockLocation
		if err := db.Where("name = ?", locations[i].Name).First(&existing).Error; err != nil {
			locations[i].CompanyID = 1
			db.Create(&locations[i])
		}
	}

	var whJktLoc, whSbyLoc, whBdgLoc StockLocation
	db.Where("name = ?", "WH/Stock").First(&whJktLoc)
	db.Where("name = ?", "WH-SBY/Stock").First(&whSbyLoc)
	db.Where("name = ?", "WH-BDG/Stock").First(&whBdgLoc)

	warehouses := []StockWarehouse{
		{
			Code:       "WH-JKT",
			Name:       "Gudang Utama Jakarta",
			LotStockID: whJktLoc.ID,
			Address:    "Kawasan Industri Pulogadung Blok B No. 12, Jakarta Timur",
			Active:     true,
		},
		{
			Code:       "WH-SBY",
			Name:       "Hub Regional Jawa Timur (Surabaya)",
			LotStockID: whSbyLoc.ID,
			Address:    "Kawasan Industri SIER Rungkut Industri Raya No. 45, Surabaya",
			Active:     true,
		},
		{
			Code:       "WH-BDG",
			Name:       "Gudang Distribusi Jawa Barat (Bandung)",
			LotStockID: whBdgLoc.ID,
			Address:    "Jalan Soekarno Hatta No. 420, Buahbatu, Bandung",
			Active:     true,
		},
	}

	for i := range warehouses {
		var existing StockWarehouse
		if err := db.Where("code = ?", warehouses[i].Code).First(&existing).Error; err != nil {
			warehouses[i].CompanyID = 1
			db.Create(&warehouses[i])
			db.Model(&StockLocation{}).Where("id = ?", warehouses[i].LotStockID).Update("warehouse_id", warehouses[i].ID)
		}
	}
	log.Println("[Stock] Seeded multi-warehouse infrastructure (WH-JKT, WH-SBY, WH-BDG)")

	// Check starting inventory balance in WH-JKT
	var quantCount int64
	db.Model(&StockQuant{}).Where("location_id = ?", whJktLoc.ID).Count(&quantCount)
	if quantCount == 0 && whJktLoc.ID > 0 {
		initQuants := []StockQuant{
			{ProductID: 1, LocationID: whJktLoc.ID, Quantity: 10},
			{ProductID: 2, LocationID: whJktLoc.ID, Quantity: 25},
			{ProductID: 3, LocationID: whJktLoc.ID, Quantity: 50},
		}
		for i := range initQuants {
			initQuants[i].CompanyID = 1
			db.Create(&initQuants[i])
		}
		db.Table("products").Where("id = ?", 1).Update("qty_available", 10)
		db.Table("products").Where("id = ?", 2).Update("qty_available", 25)
		db.Table("products").Where("id = ?", 3).Update("qty_available", 50)
	}

	// Check starting inventory balance in WH-SBY
	var sbyCount int64
	db.Model(&StockQuant{}).Where("location_id = ?", whSbyLoc.ID).Count(&sbyCount)
	if sbyCount == 0 && whSbyLoc.ID > 0 {
		sbyQuant := StockQuant{
			ProductID:  1,
			LocationID: whSbyLoc.ID,
			Quantity:   5,
		}
		sbyQuant.CompanyID = 1
		db.Create(&sbyQuant)
	}

	var opCount int64
	db.Model(&StockOrderpoint{}).Count(&opCount)
	if opCount == 0 {
		op := StockOrderpoint{
			Name:          "OP/00001",
			ProductID:     1, // Laptop Dell XPS 15
			LocationID:    whJktLoc.ID,
			ProductMinQty: 5,
			ProductMaxQty: 20,
			QtyMultiple:   1,
			Trigger:       "auto",
		}
		op.CompanyID = 1
		db.Create(&op)
		log.Println("[Stock] Seeded default Reordering Rule (OP/00001: Min 5, Max 20)")
	}

	return nil
}
