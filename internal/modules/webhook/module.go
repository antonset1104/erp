package webhook

import (
	"log"
	"time"

	"erp-golang/internal/core/ai"
	"erp-golang/internal/core/models"
	"erp-golang/internal/modules/chatter"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Module struct {
	db       *gorm.DB
	service  *Service
	handler  *Handler
	aiClient *ai.Client
}

func NewModule(db *gorm.DB, chatterSvc *chatter.Service, aiClient *ai.Client) *Module {
	repo := NewRepository(db)
	service := NewService(db, repo, chatterSvc, aiClient)
	handler := NewHandler(service)

	return &Module{
		db:       db,
		service:  service,
		handler:  handler,
		aiClient: aiClient,
	}
}

func (m *Module) Name() string {
	return "webhook"
}

func (m *Module) Service() *Service {
	return m.service
}

func (m *Module) AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&WebhookSubscription{},
		&WebhookDelivery{},
	)
}

func (m *Module) RegisterRoutes(api *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	m.handler.RegisterRoutes(api, authMiddleware)
}

func (m *Module) Seed(db *gorm.DB) error {
	if err := m.AutoMigrate(db); err != nil {
		return err
	}

	var count int64
	db.Model(&WebhookSubscription{}).Count(&count)
	if count > 0 {
		return nil
	}

	log.Println("🌱 Seeding Enterprise Webhook Subscriptions & Sample Outbound Deliveries...")

	now := time.Now()
	twoHoursAgo := now.Add(-2 * time.Hour)
	tenMinAgo := now.Add(-10 * time.Minute)
	retryTime := now.Add(5 * time.Minute)

	// 1. Seed Subscriptions
	subs := []WebhookSubscription{
		{
			ModelBase:     models.ModelBase{CompanyID: 1, Active: true},
			Name:          "Integrasi Logistik & Ekspedisi 3PL",
			TargetURL:     "https://api.logistik3pl.id/v1/shipments/notify",
			SecretToken:   "whsec_logistik_8f3a9e217c4b0d5e89a",
			EventTopics:   EventPickingValidated,
			CustomHeaders: `{"Authorization":"Bearer tok_live_logistik_9921","X-Partner-ID":"PT-LOGISTIK-01"}`,
			MaxRetries:    5,
			RetryDelaySec: 30,
			TimeoutSec:    10,
			IsActive:      true,
			LastFiredAt:   &twoHoursAgo,
			LastStatus:    StatusSuccess,
			Description:   "Push otomatis notifikasi pengiriman ke platform partner ekspedisi & kurir saat delivery order divalidasi.",
		},
		{
			ModelBase:     models.ModelBase{CompanyID: 1, Active: true},
			Name:          "Sinkronisasi CRM & Lead Gateway",
			TargetURL:     "https://hooks.crmgateway.cloud/erp/leads",
			SecretToken:   "whsec_crm_9941a0e8821bcfae3427",
			EventTopics:   EventPartnerCreated + "," + EventSaleConfirmed,
			CustomHeaders: `{"X-CRM-Source":"GoERPlang-ERP"}`,
			MaxRetries:    3,
			RetryDelaySec: 60,
			TimeoutSec:    8,
			IsActive:      true,
			LastFiredAt:   &tenMinAgo,
			LastStatus:    StatusRetrying,
			Description:   "Sinkronisasi data pelanggan baru dan pesanan penjualan yang telah dikonfirmasi ke hub CRM eksternal.",
		},
		{
			ModelBase:     models.ModelBase{CompanyID: 1, Active: true},
			Name:          "Pemberitahuan Finansial & Faktur",
			TargetURL:     "https://finance.enterprise-hub.com/api/webhooks/invoices",
			SecretToken:   "whsec_fin_6619ebcca7821903e1a8",
			EventTopics:   EventInvoicePaid,
			CustomHeaders: `{"X-Environment":"production"}`,
			MaxRetries:    3,
			RetryDelaySec: 120,
			TimeoutSec:    15,
			IsActive:      true,
			LastFiredAt:   nil,
			LastStatus:    "idle",
			Description:   "Event pemicu saat invoice telah lunas untuk rekonsiliasi ERP pusat dan pembukuan eksternal.",
		},
	}

	for i := range subs {
		if err := db.Create(&subs[i]).Error; err != nil {
			log.Printf("⚠️ Gagal seeding webhook subscription %s: %v", subs[i].Name, err)
		}
	}

	// 2. Seed Sample Deliveries
	if len(subs) > 0 {
		sampleDeliveries := []WebhookDelivery{
			{
				ModelBase:      models.ModelBase{CompanyID: 1, Active: true},
				DeliveryCode:   "DLV/2026/00001",
				SubscriptionID: subs[0].ID,
				EventName:      EventPickingValidated,
				ModelName:      "stock.picking",
				ResID:          1,
				TargetURL:      subs[0].TargetURL,
				RequestHeaders: `{"Content-Type":["application/json"],"User-Agent":["GoERP-WebhookEngine/1.27.0"],"X-Webhook-Event":["stock.picking.done"],"X-Webhook-Signature":["sha256=a1b2c3d4e5f67890abcdef1234567890abcdef1234567890abcdef1234567890"]}`,
				PayloadJSON:    `{"id":"evt_seed_001","event":"stock.picking.done","timestamp":"2026-10-09T17:20:00Z","model":"stock.picking","resource_id":1,"company_id":1,"data":{"picking_id":1,"name":"WH/OUT/00001","state":"done","carrier":"JNE Cargo Express"}}`,
				HMACSignature:  "a1b2c3d4e5f67890abcdef1234567890abcdef1234567890abcdef1234567890",
				StatusCode:     200,
				ResponseBody:   `{"status":"acknowledged","tracking_code":"JNE-EXP-99821","queued_at":"2026-10-09T17:20:01Z"}`,
				DurationMs:     142,
				Status:         StatusSuccess,
				AttemptCount:   1,
			},
			{
				ModelBase:      models.ModelBase{CompanyID: 1, Active: true},
				DeliveryCode:   "DLV/2026/00002",
				SubscriptionID: subs[1].ID,
				EventName:      EventSaleConfirmed,
				ModelName:      "sale.order",
				ResID:          3,
				TargetURL:      subs[1].TargetURL,
				RequestHeaders: `{"Content-Type":["application/json"],"User-Agent":["GoERP-WebhookEngine/1.27.0"],"X-Webhook-Event":["sale.order.confirmed"],"X-Webhook-Signature":["sha256=88f9e0b1c2d3e4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9"]}`,
				PayloadJSON:    `{"id":"evt_seed_002","event":"sale.order.confirmed","timestamp":"2026-10-09T19:10:00Z","model":"sale.order","resource_id":3,"company_id":1,"data":{"order_id":3,"name":"SO0003","partner_id":2,"amount_total":18500000.00,"state":"sale"}}`,
				HMACSignature:  "88f9e0b1c2d3e4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9",
				StatusCode:     503,
				ResponseBody:   `{"error":"Service Temporarily Unavailable - Rate Limit Exceeded on Ingestion Gateway"}`,
				DurationMs:     820,
				Status:         StatusRetrying,
				AttemptCount:   1,
				NextRetryAt:    &retryTime,
				ErrorMessage:   "HTTP 503: Service Temporarily Unavailable - Gateway overloaded",
			},
		}

		for _, d := range sampleDeliveries {
			if err := db.Create(&d).Error; err != nil {
				log.Printf("⚠️ Gagal seeding webhook delivery %s: %v", d.DeliveryCode, err)
			}
		}
	}

	log.Println("✅ Webhook Subscriptions & Sample Deliveries seeded successfully.")
	return nil
}
