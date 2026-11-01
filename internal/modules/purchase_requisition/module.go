package purchase_requisition

import (
	"log"

	"erp-golang/internal/modules/chatter"
	"erp-golang/internal/modules/product"
	"erp-golang/internal/modules/purchase"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Module struct {
	service *Service
	handler *Handler
}

func NewModule(db *gorm.DB, purchaseSvc *purchase.Service, chatterSvc *chatter.Service) *Module {
	repo := NewRepository(db)
	svc := NewService(db, repo, purchaseSvc, chatterSvc)
	h := NewHandler(svc)
	return &Module{
		service: svc,
		handler: h,
	}
}

func (m *Module) Name() string {
	return "purchase_requisition"
}

func (m *Module) Service() *Service {
	return m.service
}

func (m *Module) RegisterRoutes(api *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	group := api.Group("/purchase-requisitions")
	group.Use(authMiddleware)
	{
		group.GET("", m.handler.List)
		group.POST("", m.handler.Create)
		group.GET("/stats", m.handler.Stats)
		group.GET("/:id", m.handler.GetByID)
		group.POST("/:id/confirm", m.handler.Confirm)
		group.PUT("/:id/confirm", m.handler.Confirm)
		group.POST("/:id/open-tender", m.handler.OpenTender)
		group.PUT("/:id/open-tender", m.handler.OpenTender)
		group.POST("/:id/generate-rfq", m.handler.GenerateRFQ)
		group.POST("/:id/select-bid", m.handler.SelectBid)
		group.POST("/:id/release-blanket", m.handler.ReleaseBlanket)
		group.POST("/:id/cancel", m.handler.Cancel)
		group.PUT("/:id/cancel", m.handler.Cancel)
		group.POST("/:id/reset-draft", m.handler.ResetDraft)
		group.PUT("/:id/reset-draft", m.handler.ResetDraft)
		group.PUT("/:id/draft", m.handler.ResetDraft)
	}
}

func (m *Module) AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&PurchaseRequisition{},
		&PurchaseRequisitionLine{},
	)
}

func (m *Module) Seed(db *gorm.DB) error {
	var count int64
	db.Model(&PurchaseRequisition{}).Count(&count)
	if count > 0 {
		return nil
	}

	var prods []product.Product
	db.Find(&prods)
	if len(prods) == 0 {
		return nil
	}

	// 1. Seed Sample Competitive Tender (Call for Bids)
	tenderReq := PurchaseRequisition{
		Name:         "PR/2026/00001",
		Origin:       "MEMO-IT-PROC-Q4",
		Type:         TypePurchaseTender,
		State:        StateConfirmed,
		UserID:       1,
		OrderingDate: "2026-10-01",
		DateEnd:      "2026-10-25",
		ScheduleDate: "2026-11-01",
		CompanyID:    1,
		Description:  "Pengadaan massal unit laptop dan display kerja workstation untuk tim engineering",
		TotalAmount:  prods[0].StandardPrice * 10,
		Lines: []PurchaseRequisitionLine{
			{
				ProductID:   prods[0].ID,
				Description: prods[0].Name,
				ProductQty:  10,
				PriceUnit:   prods[0].StandardPrice,
				Subtotal:    prods[0].StandardPrice * 10,
			},
		},
	}
	db.Create(&tenderReq)

	// 2. Seed Sample Blanket Purchase Order Contract
	if len(prods) > 1 {
		vendorID := uint(2)
		blanketReq := PurchaseRequisition{
			Name:         "BO/2026/00001",
			Origin:       "AGREEMENT-SUPPLY-2026",
			Type:         TypeBlanketOrder,
			State:        StateConfirmed,
			UserID:       1,
			VendorID:     &vendorID,
			OrderingDate: "2026-10-01",
			DateEnd:      "2027-03-31",
			ScheduleDate: "2026-10-15",
			CompanyID:    1,
			Description:  "Kontrak suplai komponen inventaris periode Q4 2026 - Q1 2027",
			TotalAmount:  prods[1].StandardPrice * 50,
			Lines: []PurchaseRequisitionLine{
				{
					ProductID:   prods[1].ID,
					Description: prods[1].Name,
					ProductQty:  50,
					QtyOrdered:  10, // 10 already released
					PriceUnit:   prods[1].StandardPrice,
					Subtotal:    prods[1].StandardPrice * 50,
				},
			},
		}
		db.Create(&blanketReq)
	}

	log.Println("[PurchaseRequisition] Berhasil seeding master data Demo Tender & Blanket Purchase Order")
	return nil
}
