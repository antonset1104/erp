package subscription

import (
	"log"
	"time"

	"erp-golang/internal/core/ai"
	"erp-golang/internal/modules/account"
	"erp-golang/internal/modules/base"
	"erp-golang/internal/modules/chatter"
	"erp-golang/internal/modules/product"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Module struct {
	db       *gorm.DB
	service  *Service
	handler  *Handler
	aiClient *ai.Client
}

func NewModule(db *gorm.DB, accountSvc *account.Service, chatterSvc *chatter.Service, aiClient *ai.Client) *Module {
	repo := NewRepository(db)
	service := NewService(db, repo, accountSvc, chatterSvc, aiClient)
	handler := NewHandler(service)

	return &Module{
		db:       db,
		service:  service,
		handler:  handler,
		aiClient: aiClient,
	}
}

func (m *Module) Name() string {
	return "subscription"
}

func (m *Module) Service() *Service {
	return m.service
}

func (m *Module) AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&SubscriptionTemplate{},
		&Subscription{},
		&SubscriptionLine{},
		&SubscriptionInvoiceLog{},
		&SubscriptionStageLog{},
	)
}

func (m *Module) RegisterRoutes(api *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	m.handler.RegisterRoutes(api, authMiddleware)
}

func (m *Module) Seed(db *gorm.DB) error {
	if err := m.AutoMigrate(db); err != nil {
		return err
	}

	// 1. Seed Default Subscription Templates
	var tmplCount int64
	db.Model(&SubscriptionTemplate{}).Count(&tmplCount)
	if tmplCount == 0 {
		templates := []SubscriptionTemplate{
			{
				Code:               "TMPL-SAAS-MONTHLY-PRO",
				Name:               "Cloud SaaS Pro (Bulanan)",
				RecurringInterval:  IntervalMonthly,
				RecurringRuleCount: 1,
				InvoiceJournalID:   1,
				AutoPostInvoice:    true,
				Description:        "Akses Cloud ERP Pro mencakup Modul Keuangan, Penjualan, & Stok untuk hingga 25 pengguna.",
				Active:             true,
			},
			{
				Code:               "TMPL-SAAS-ANNUAL-ENT",
				Name:               "Enterprise Cloud Unlimited (Tahunan)",
				RecurringInterval:  IntervalYearly,
				RecurringRuleCount: 1,
				InvoiceJournalID:   1,
				AutoPostInvoice:    true,
				Description:        "Paket Korporat Tahunan Unlimited Lisensi dengan Dedicated Account Manager dan SLA 99.99%.",
				Active:             true,
			},
			{
				Code:               "TMPL-MAINT-QUARTERLY",
				Name:               "Pemeliharaan & SLA Server (Triwulanan)",
				RecurringInterval:  IntervalQuarterly,
				RecurringRuleCount: 1,
				InvoiceJournalID:   1,
				AutoPostInvoice:    true,
				Description:        "Layanan audit berkala, backup basis data mingguan, dan patching keamanan sistem berulang.",
				Active:             true,
			},
		}

		for i := range templates {
			templates[i].CompanyID = 1
			db.Create(&templates[i])
		}
		log.Println("[Subscription] Seeded default subscription templates (Monthly Pro, Annual Enterprise, Quarterly SLA)")
	}

	// 2. Seed Demo Active Subscriptions if empty
	var subCount int64
	db.Model(&Subscription{}).Count(&subCount)
	if subCount == 0 {
		// Find demo customer
		var partner base.Partner
		if err := db.Where("customer_rank > 0").First(&partner).Error; err != nil {
			partner.ID = 2
			partner.Name = "PT Maju Bersama Makmur"
		}

		// Find demo template
		var tmplMonthly, tmplAnnual SubscriptionTemplate
		db.Where("code = ?", "TMPL-SAAS-MONTHLY-PRO").First(&tmplMonthly)
		db.Where("code = ?", "TMPL-SAAS-ANNUAL-ENT").First(&tmplAnnual)

		// Find demo product
		var prod product.Product
		if err := db.First(&prod).Error; err != nil {
			prod.ID = 1
			prod.Name = "Lisensi Cloud ERP Pro"
		}

		now := time.Now()
		startDateMonthly := now.AddDate(0, -1, 0).Format("2006-01-02")
		nextInvoiceMonthly := now.AddDate(0, 0, 2).Format("2006-01-02") // In 2 days

		// Demo Subscription 1: Monthly Pro
		sub1 := Subscription{
			Code:                   "SUB/2026/00001",
			Name:                   "Cloud ERP Pro Subscription - " + partner.Name,
			PartnerID:              partner.ID,
			TemplateID:             tmplMonthly.ID,
			State:                  StateInProgress,
			StartDate:              startDateMonthly,
			NextInvoiceDate:        nextInvoiceMonthly,
			RecurringInterval:      IntervalMonthly,
			RecurringIntervalCount: 1,
			Currency:               "IDR",
			RecurringAmount:        3500000,
			TotalMRR:               3500000,
			TotalARR:               42000000,
			InvoicedCount:          1,
			TotalInvoicedAmount:    3500000,
			AutoRenew:              true,
			PaymentMethod:          "bank_transfer",
			Notes:                  "Kontrak berlangganan bulanan software ERP multi-perusahaan.",
			Lines: []SubscriptionLine{
				{
					ProductID: prod.ID,
					Name:      "Paket Lisensi Cloud ERP Pro (25 User)",
					Quantity:  1,
					UnitPrice: 3000000,
					Discount:  0,
					Subtotal:  3000000,
					UoM:       "Bulan",
				},
				{
					ProductID: prod.ID,
					Name:      "Dedicated Cloud Storage 100GB Addon",
					Quantity:  1,
					UnitPrice: 500000,
					Discount:  0,
					Subtotal:  500000,
					UoM:       "Bulan",
				},
			},
		}
		sub1.CompanyID = 1
		for i := range sub1.Lines {
			sub1.Lines[i].CompanyID = 1
		}
		db.Create(&sub1)

		// Demo Subscription 2: Annual Enterprise
		if tmplAnnual.ID > 0 {
			startDateAnnual := now.AddDate(0, -3, 0).Format("2006-01-02")
			nextInvoiceAnnual := now.AddDate(0, 9, 0).Format("2006-01-02")

			sub2 := Subscription{
				Code:                   "SUB/2026/00002",
				Name:                   "Enterprise Cloud Unlimited - " + partner.Name,
				PartnerID:              partner.ID,
				TemplateID:             tmplAnnual.ID,
				State:                  StateInProgress,
				StartDate:              startDateAnnual,
				EndDate:                now.AddDate(1, 0, 0).Format("2006-01-02"),
				NextInvoiceDate:        nextInvoiceAnnual,
				RecurringInterval:      IntervalYearly,
				RecurringIntervalCount: 1,
				Currency:               "IDR",
				RecurringAmount:        36000000,
				TotalMRR:               3000000,  // 36jt / 12
				TotalARR:               36000000, // 3jt * 12
				InvoicedCount:          1,
				TotalInvoicedAmount:    36000000,
				AutoRenew:              true,
				PaymentMethod:          "auto_debit",
				Notes:                  "Kontrak Enterprise korporasi tahunan dengan SLA 99.99%.",
				Lines: []SubscriptionLine{
					{
						ProductID: prod.ID,
						Name:      "Enterprise Cloud ERP License Unlimited",
						Quantity:  1,
						UnitPrice: 36000000,
						Discount:  0,
						Subtotal:  36000000,
						UoM:       "Tahun",
					},
				},
			}
			sub2.CompanyID = 1
			for i := range sub2.Lines {
				sub2.Lines[i].CompanyID = 1
			}
			db.Create(&sub2)
		}

		log.Println("[Subscription] Seeded demo subscriptions SUB/2026/00001 & SUB/2026/00002")
	}

	return nil
}
