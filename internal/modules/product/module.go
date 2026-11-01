package product

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

func (m *Module) Name() string {
	return "product"
}

func (m *Module) AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&ProductCategory{},
		&UoM{},
		&Product{},
	)
}

func (m *Module) RegisterRoutes(api *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	group := api.Group("/products")
	group.Use(authMiddleware)
	{
		group.GET("", m.handler.ListProducts)
		group.GET("/export", m.handler.ExportProductsCSV)
		group.POST("", m.handler.CreateProduct)
		group.GET("/:id", m.handler.GetProduct)
		group.GET("/categories", m.handler.ListCategories)
		group.GET("/uoms", m.handler.ListUoM)
	}
}

func (m *Module) Seed(db *gorm.DB) error {
	// Seed UoM
	var uomCount int64
	db.Model(&UoM{}).Count(&uomCount)
	if uomCount == 0 {
		uoms := []UoM{
			{Name: "Unit(s)", Category: "Unit", Factor: 1.0},
			{Name: "Pcs", Category: "Unit", Factor: 1.0},
			{Name: "Dozen", Category: "Unit", Factor: 12.0},
			{Name: "kg", Category: "Weight", Factor: 1.0},
			{Name: "Gram", Category: "Weight", Factor: 0.001},
			{Name: "Hour(s)", Category: "Working Time", Factor: 1.0},
		}
		for i := range uoms {
			uoms[i].CompanyID = 1
			db.Create(&uoms[i])
		}
		log.Println("[Product] Seeded default Units of Measure (UoM)")
	}

	// Seed Categories
	var catCount int64
	db.Model(&ProductCategory{}).Count(&catCount)
	if catCount == 0 {
		categories := []ProductCategory{
			{Name: "All Products"},
			{Name: "Computers & Electronics"},
			{Name: "Office Supplies"},
			{Name: "Services"},
		}
		for i := range categories {
			categories[i].CompanyID = 1
			db.Create(&categories[i])
		}
		log.Println("[Product] Seeded default product categories")
	}

	// Seed Sample Products
	var prodCount int64
	db.Model(&Product{}).Count(&prodCount)
	if prodCount == 0 {
		products := []Product{
			{
				Name:          "Laptop Dell XPS 15",
				DefaultCode:   "DELL-XPS-15",
				Barcode:       "899123456001",
				Type:          "product",
				CategoryID:    2,
				UoMID:         1,
				ListPrice:     24500000,
				StandardPrice: 19000000,
				QtyAvailable:  0,
				Description:   "High-performance business laptop, Core i7, 32GB RAM, 1TB SSD",
			},
			{
				Name:          "Wireless Ergonomic Mouse",
				DefaultCode:   "LOGI-MX-3",
				Barcode:       "899123456002",
				Type:          "product",
				CategoryID:    2,
				UoMID:         1,
				ListPrice:     1450000,
				StandardPrice: 950000,
				QtyAvailable:  0,
				Description:   "Logitech MX Master 3 wireless rechargeable mouse",
			},
			{
				Name:          "Kertas HVS A4 80gr Rim",
				DefaultCode:   "PAP-A4-80",
				Barcode:       "899123456003",
				Type:          "product",
				CategoryID:    3,
				UoMID:         2,
				ListPrice:     55000,
				StandardPrice: 42000,
				QtyAvailable:  0,
				Description:   "Kertas print kualitas tinggi 500 lembar per rim",
			},
			{
				Name:          "IT Consulting & Implementation Service",
				DefaultCode:   "SRV-CONSULT-01",
				Barcode:       "",
				Type:          "service",
				CategoryID:    4,
				UoMID:         6,
				ListPrice:     750000,
				StandardPrice: 350000,
				QtyAvailable:  0,
				Description:   "Professional ERP setup & customization services per hour",
			},
		}

		for i := range products {
			products[i].CompanyID = 1
			db.Create(&products[i])
		}
		log.Println("[Product] Seeded sample products")
	}

	return nil
}
