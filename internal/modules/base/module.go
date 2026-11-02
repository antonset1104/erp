package base

import (
	"log"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Module struct {
	db        *gorm.DB
	jwtSecret string
	svc       *Service
	handler   *Handler
}

func NewModule(db *gorm.DB, jwtSecret string) *Module {
	svc := NewService(db, jwtSecret)
	handler := NewHandler(svc)
	return &Module{
		db:        db,
		jwtSecret: jwtSecret,
		svc:       svc,
		handler:   handler,
	}
}

func (m *Module) Name() string {
	return "base"
}

func (m *Module) AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&Company{},
		&Currency{},
		&User{},
		&Partner{},
	)
}

func (m *Module) RegisterRoutes(api *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	authGroup := api.Group("/auth")
	{
		authGroup.POST("/login", m.handler.Login)
		authGroup.GET("/me", authMiddleware, m.handler.Me)
	}

	partnerGroup := api.Group("/partners")
	partnerGroup.Use(authMiddleware)
	{
		partnerGroup.GET("", m.handler.ListPartners)
		partnerGroup.POST("", m.handler.CreatePartner)
		partnerGroup.GET("/:id", m.handler.GetPartner)
	}
}

func (m *Module) Seed(db *gorm.DB) error {
	// Seed Base Currency
	var curCount int64
	db.Model(&Currency{}).Count(&curCount)
	if curCount == 0 {
		currencies := []Currency{
			{Name: "IDR", Symbol: "Rp", Rate: 1.0, Digits: 0},
			{Name: "USD", Symbol: "$", Rate: 0.000063, Digits: 2},
		}
		for i := range currencies {
			currencies[i].CompanyID = 1
			db.Create(&currencies[i])
		}
		log.Println("[Base] Seeded default currencies (IDR, USD)")
	}

	// Seed Default Company
	var compCount int64
	db.Model(&Company{}).Count(&compCount)
	if compCount == 0 {
		company := Company{
			Name:       "PT Golang ERP Nusantara",
			Email:      "info@goerplang.co.id",
			Phone:      "+62 21 555 1234",
			Street:     "Jl. Jendral Sudirman Kav. 52",
			City:       "Jakarta",
			Country:    "Indonesia",
			CurrencyID: 1,
			Currency:   "IDR",
			TaxID:      "01.234.567.8-012.000",
		}
		company.CompanyID = 1
		db.Create(&company)
		log.Println("[Base] Seeded default company")
	}

	// Seed Admin User
	var userCount int64
	db.Model(&User{}).Count(&userCount)
	if userCount == 0 {
		hash, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)

		adminPartner := Partner{
			Name:      "System Administrator",
			Email:     "admin@erp.local",
			IsCompany: false,
		}
		adminPartner.CompanyID = 1
		db.Create(&adminPartner)

		admin := User{
			Name:         "Administrator",
			Email:        "admin@erp.local",
			PasswordHash: string(hash),
			Role:         "admin",
			PartnerID:    adminPartner.ID,
		}
		admin.CompanyID = 1
		db.Create(&admin)
		log.Println("[Base] Seeded admin user: admin@erp.local / admin123")

		// Seed sample Customers and Suppliers
		samplePartners := []Partner{
			{
				Name:         "PT Maju Bersama Makmur",
				IsCompany:    true,
				Email:        "purchasing@majubersama.com",
				Phone:        "+62 21 888 9999",
				City:         "Jakarta",
				CustomerRank: 1,
				CreditLimit:  100000000,
			},
			{
				Name:         "CV Sumber Jaya Teknik",
				IsCompany:    true,
				Email:        "sales@sumberjaya.com",
				Phone:        "+62 31 777 5555",
				City:         "Surabaya",
				SupplierRank: 1,
			},
			{
				Name:         "Budi Santoso",
				IsCompany:    false,
				Email:        "budi.santoso@gmail.com",
				Phone:        "+62 812 3456 7890",
				City:         "Bandung",
				CustomerRank: 1,
				CreditLimit:  20000000,
			},
		}

		for i := range samplePartners {
			samplePartners[i].CompanyID = 1
			db.Create(&samplePartners[i])
		}
		log.Println("[Base] Seeded sample partners (customers & vendors)")
	}

	return nil
}
