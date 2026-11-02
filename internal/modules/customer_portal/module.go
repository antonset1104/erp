package customer_portal

import (
	"log"
	"time"

	"erp-golang/internal/core/ai"
	"erp-golang/internal/modules/account"
	"erp-golang/internal/modules/chatter"
	"erp-golang/internal/modules/sale"
	"erp-golang/internal/modules/stock"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Module struct {
	repo       *Repository
	service    *Service
	handler    *Handler
	saleSvc    *sale.Service
	stockSvc   *stock.Service
	accountSvc *account.Service
	chatterSvc *chatter.Service
	aiClient   *ai.Client
}

func NewModule(
	db *gorm.DB,
	saleSvc *sale.Service,
	stockSvc *stock.Service,
	accountSvc *account.Service,
	chatterSvc *chatter.Service,
	aiClient *ai.Client,
) *Module {
	repo := NewRepository(db)
	service := NewService(repo, db, saleSvc, stockSvc, accountSvc, chatterSvc, aiClient)
	handler := NewHandler(service)
	return &Module{
		repo:       repo,
		service:    service,
		handler:    handler,
		saleSvc:    saleSvc,
		stockSvc:   stockSvc,
		accountSvc: accountSvc,
		chatterSvc: chatterSvc,
		aiClient:   aiClient,
	}
}

func (m *Module) Name() string {
	return "customer_portal"
}

func (m *Module) AutoMigrate(db *gorm.DB) error {
	return m.repo.AutoMigrate()
}

func (m *Module) RegisterRoutes(api *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	m.handler.RegisterRoutes(api, authMiddleware)
}

func (m *Module) Service() *Service {
	return m.service
}

func (m *Module) Seed(db *gorm.DB) error {
	var count int64
	db.Model(&CustomerPortalAccount{}).Count(&count)
	if count > 0 {
		return nil
	}

	log.Println("🌱 Seeding Customer Portal & B2B Self-Service Commerce Sample Data...")

	// 1. Seed Customer Portal Accounts
	now := time.Now()
	accIndah := &CustomerPortalAccount{
		PartnerID:      1,
		CustomerName:   "PT Indah Makmur Retail",
		ContactName:    "Dewi Kartika",
		Email:          "purchasing@indahmakmur.co.id",
		Phone:          "+62-21-77889900",
		AccessToken:    "customer_portal_indah_token_2026",
		IsActive:       true,
		CreditLimit:    150000000,
		TotalLogins:    12,
		LastLoginAt:    &now,
		AllowedReorder: true,
	}
	_ = db.Create(accIndah)

	accSinar := &CustomerPortalAccount{
		PartnerID:      2,
		CustomerName:   "PT Sinar Jaya Perkasa",
		ContactName:    "Rian Hidayat",
		Email:          "procurement@sinarjaya.id",
		Phone:          "+62-21-44556677",
		AccessToken:    "customer_portal_sinar_token_2026",
		IsActive:       true,
		CreditLimit:    200000000,
		TotalLogins:    9,
		LastLoginAt:    &now,
		AllowedReorder: true,
	}
	_ = db.Create(accSinar)

	// 2. Seed Sample B2B Order Request
	sampleOrder := &B2BOrderRequest{
		RequestNumber:   "CP-SO/2026/00001",
		PortalAccountID: accIndah.ID,
		PartnerID:       accIndah.PartnerID,
		CompanyID:       1,
		SubtotalAmount:  12500000,
		TaxAmount:       1375000,
		TotalAmount:     13875000,
		PaymentMethod:   "credit_terms",
		DeliveryAddress: "Kawasan Industri MM2100 Blok C-4, Cikarang Barat",
		CustomerNotes:   "Repeat order bulanan - mohon dikirim sebelum akhir pekan",
		Status:          "approved",
		Lines: []B2BOrderLine{
			{
				ProductID: 1,
				Quantity:  10,
				UnitPrice: 1250000,
				Subtotal:  12500000,
				Notes:     "Kemasan karton standar",
			},
		},
	}
	_ = db.Create(sampleOrder)

	// 3. Seed Sample RMA Request
	sampleRMA := &RMARequest{
		RMANumber:           "RMA/2026/00001",
		PortalAccountID:     accIndah.ID,
		PartnerID:           accIndah.PartnerID,
		ProductID:           1,
		Quantity:            1,
		ReasonCode:          "damaged_in_transit",
		Description:         "Kardus basah dan unit mengalami penyok pada sisi samping",
		EvidenceURL:         "https://storage.erp.local/rma/indah_rma_photo_01.jpg",
		ResolutionRequested: "replace_item",
		ResolutionAction:    "Kirim unit pengganti dari batch WH/STOCK",
		Status:              "approved",
		AdminNotes:          "Lolos verifikasi QC. Unit pengganti dijadwalkan kirim besok.",
		ResolvedAt:          &now,
	}
	_ = db.Create(sampleRMA)

	// 4. Seed Sample Payment Notification
	samplePay := &CustomerPaymentNotification{
		NotificationNumber: "CP-PAY/2026/00001",
		PortalAccountID:    accIndah.ID,
		PartnerID:          accIndah.PartnerID,
		InvoiceID:          1,
		AmountPaid:         13875000,
		PaymentDate:        now,
		SenderBank:         "BCA",
		SenderAccountName:  "PT Indah Makmur Retail",
		SenderAccountNo:    "8820192831",
		TargetJournalID:    1,
		ReferenceNumber:    "TRF-BCA-2026-981245",
		ProofURL:           "https://storage.erp.local/payments/slip_transfer_bca_981245.pdf",
		Status:             "confirmed",
		Notes:              "Pelunasan invoice tempo 30 hari via transfer bank",
		VerifiedAt:         &now,
	}
	_ = db.Create(samplePay)

	log.Println("✅ Customer Portal Initial Seeds Complete.")
	return nil
}
