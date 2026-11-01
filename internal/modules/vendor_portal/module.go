package vendor_portal

import (
	"log"
	"time"

	"erp-golang/internal/core/ai"
	"erp-golang/internal/modules/account"
	"erp-golang/internal/modules/chatter"
	"erp-golang/internal/modules/purchase"
	"erp-golang/internal/modules/stock"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Module struct {
	repo        *Repository
	service     *Service
	handler     *Handler
	purchaseSvc *purchase.Service
	stockSvc    *stock.Service
	accountSvc  *account.Service
	chatterSvc  *chatter.Service
	aiClient    *ai.Client
}

func NewModule(
	db *gorm.DB,
	purchaseSvc *purchase.Service,
	stockSvc *stock.Service,
	accountSvc *account.Service,
	chatterSvc *chatter.Service,
	aiClient *ai.Client,
) *Module {
	repo := NewRepository(db)
	service := NewService(repo, db, purchaseSvc, stockSvc, accountSvc, chatterSvc, aiClient)
	handler := NewHandler(service)
	return &Module{
		repo:        repo,
		service:     service,
		handler:     handler,
		purchaseSvc: purchaseSvc,
		stockSvc:    stockSvc,
		accountSvc:  accountSvc,
		chatterSvc:  chatterSvc,
		aiClient:    aiClient,
	}
}

func (m *Module) Name() string {
	return "vendor_portal"
}

func (m *Module) Init(db *gorm.DB) error {
	return nil
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
	db.Model(&VendorPortalAccount{}).Count(&count)
	if count > 0 {
		return nil
	}

	log.Println("🌱 Seeding Vendor Portal & Supplier Self-Service Sample Data...")

	// 1. Seed Vendor Portal Accounts
	megaVendor := &VendorPortalAccount{
		PartnerID:   1, // Linked to main partner/supplier
		VendorName:  "PT Mega Steel & Components Indonesia",
		ContactName: "Ir. Hendro Wijaya",
		Email:       "supplier@megasteel.co.id",
		Phone:       "+62-21-88997711",
		AccessToken: "vendor_portal_mega_token_2026",
		IsActive:    true,
		TotalLogins: 14,
	}
	db.Create(megaVendor)

	techVendor := &VendorPortalAccount{
		PartnerID:   2,
		VendorName:  "PT Surya Elektronika Mandiri",
		ContactName: "Lestari Anggraini",
		Email:       "sales@suryaelektronik.id",
		Phone:       "+62-21-55443322",
		AccessToken: "vendor_portal_surya_token_2026",
		IsActive:    true,
		TotalLogins: 8,
	}
	db.Create(techVendor)

	// 2. Seed Sample Quotation Bid
	sampleBid := &VendorQuotationBid{
		BidNumber:            "BID/2026/00001",
		RFQReference:         "PO00001",
		PartnerID:            1,
		VendorName:           "PT Mega Steel & Components Indonesia",
		TotalBidAmount:       18500000.0,
		DeliveryLeadTimeDays: 5,
		PaymentTerms:         "NET 30",
		Status:               "accepted",
		Notes:                "Harga penawaran khusus diskon volume B2B 5% dengan jaminan garansi 1 tahun.",
		SubmittedAt:          time.Now().AddDate(0, 0, -5),
		Lines: []VendorQuotationBidLine{
			{
				ProductID:   1,
				ProductName: "Plat Besi Galvanis 2mm",
				ProductCode: "STEEL-2MM",
				Quantity:    50,
				UnitPrice:   250000.0,
				Subtotal:    12500000.0,
				Notes:       "Standar SNI Grade A",
			},
			{
				ProductID:   2,
				ProductName: "Baut & Mur Baja M10 (Box 100 pcs)",
				ProductCode: "BOLT-M10",
				Quantity:    20,
				UnitPrice:   300000.0,
				Subtotal:    6000000.0,
				Notes:       "High tensile strength",
			},
		},
	}
	db.Create(sampleBid)

	// 3. Seed Sample ASN (Advance Shipping Notice)
	actualTime := time.Now().AddDate(0, 0, -2)
	sampleASN := &VendorASN{
		ASNNumber:            "ASN/2026/00001",
		PurchaseOrderID:      1,
		PurchaseOrderRef:     "PO00001",
		PartnerID:            1,
		VendorName:           "PT Mega Steel & Components Indonesia",
		CarrierName:          "JNE Trucking (JTR)",
		TrackingRef:          "JTR98273618293",
		DriverName:           "Bambang Sutrisno",
		VehicleNumber:        "B 9182 KXA",
		EstimatedArrivalDate: time.Now().AddDate(0, 0, -2),
		ActualArrivalDate:    &actualTime,
		Status:               "received",
		Notes:                "Pengiriman batch 1 pallet, kemasan aman berpalet kayu.",
		Lines: []VendorASNLine{
			{
				ProductID:        1,
				ProductName:      "Plat Besi Galvanis 2mm",
				QuantityShipped:  50,
				QuantityReceived: 50,
				LotNumber:        "LOT-STEEL-20261001",
			},
			{
				ProductID:        2,
				ProductName:      "Baut & Mur Baja M10 (Box 100 pcs)",
				QuantityShipped:  20,
				QuantityReceived: 20,
				LotNumber:        "LOT-BOLT-20261001",
			},
		},
	}
	db.Create(sampleASN)

	// 4. Seed Sample Vendor Invoice Submission
	sampleInvoice := &VendorInvoiceSubmission{
		SubmissionNumber:      "VINV/2026/00001",
		PurchaseOrderID:       1,
		PurchaseOrderRef:      "PO00001",
		PartnerID:             1,
		VendorName:            "PT Mega Steel & Components Indonesia",
		InvoiceNumberSupplier: "INV-MEGA-2026-8819",
		InvoiceDate:           time.Now().AddDate(0, 0, -2),
		DueDate:               time.Now().AddDate(0, 0, 28),
		Subtotal:              18500000.0,
		TaxAmount:             2035000.0, // PPN 11%
		TotalAmount:           20535000.0,
		TaxInvoiceNumber:      "010.002-26.98172635",
		FileAttachmentURL:     "/uploads/attachments/inv_mega_8819.pdf",
		MatchingStatus:        "matched_3way",
		Status:                "approved",
		Notes:                 "Tagihan telah diverifikasi 3-Way Matching dengan PO00001 dan ASN/2026/00001.",
	}
	db.Create(sampleInvoice)

	log.Println("✅ Vendor Portal Initial Seeds Complete.")
	return nil
}
