package inbound_webhook

import (
	"log"
	"time"

	"erp-golang/internal/core/ai"
	"erp-golang/internal/core/models"
	"erp-golang/internal/modules/account"
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
	return "inbound_webhook"
}

func (m *Module) Service() *Service {
	return m.service
}

func (m *Module) AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&InboundWebhookEndpoint{},
		&InboundWebhookLog{},
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
	db.Model(&InboundWebhookEndpoint{}).Count(&count)
	if count > 0 {
		return nil
	}

	log.Println("🌱 Seeding Enterprise Inbound Webhook Endpoints & Sample Logs...")

	now := time.Now()
	oneHourAgo := now.Add(-1 * time.Hour)
	thirtyMinAgo := now.Add(-30 * time.Minute)

	// 1. Seed Default Endpoints
	endpoints := []InboundWebhookEndpoint{
		{
			ModelBase:    models.ModelBase{CompanyID: 1, Active: true},
			EndpointCode: "INWH/2026/00001",
			EndpointKey:  "midtrans_snap_payment",
			Name:         "Midtrans Payment Gateway Callback",
			ProviderType: ProviderPaymentMidtrans,
			AuthType:     AuthSHA512,
			SecretKey:    "SB-Mid-server-TEST-KEY-998811",
			TargetModel:  "account.move",
			AutoAction:   ActionMarkInvoicePaid,
			AllowedIPs:   "",
			IsActive:     true,
			TotalCalls:   1,
			LastCalledAt: &oneHourAgo,
			LastStatus:   StatusProcessed,
			Description:  "Menerima notifikasi settlement pembayaran QRIS, Virtual Account, dan Kartu Kredit dari Midtrans Snap untuk melunasi faktur otomatis.",
		},
		{
			ModelBase:    models.ModelBase{CompanyID: 1, Active: true},
			EndpointCode: "INWH/2026/00002",
			EndpointKey:  "jne_shipment_callback",
			Name:         "JNE Logistics Courier Tracking Updates",
			ProviderType: ProviderLogistics3PL,
			AuthType:     AuthHMACSHA256,
			SecretKey:    "whsec_in_jne_logistics_secret_2026",
			TargetModel:  "stock.picking",
			AutoAction:   ActionUpdatePickingTracking,
			AllowedIPs:   "",
			IsActive:     true,
			TotalCalls:   1,
			LastCalledAt: &thirtyMinAgo,
			LastStatus:   StatusProcessed,
			Description:  "Menerima pembaruan resi dan status serah terima barang dari ekspedisi pengiriman kurir rekanan.",
		},
		{
			ModelBase:    models.ModelBase{CompanyID: 1, Active: true},
			EndpointCode: "INWH/2026/00003",
			EndpointKey:  "web_leads_capture",
			Name:         "Website Contact Form & Lead Ingestion",
			ProviderType: ProviderCRMLeads,
			AuthType:     AuthBearer,
			SecretKey:    "tok_web_lead_capture_9921",
			TargetModel:  "crm.lead",
			AutoAction:   ActionCreateCRMLead,
			AllowedIPs:   "",
			IsActive:     true,
			TotalCalls:   0,
			LastCalledAt: nil,
			LastStatus:   "idle",
			Description:  "Menerima data pendaftaran calon prospek penjualan baru dari formulir kontak website dan landing page promosi.",
		},
	}

	for i := range endpoints {
		if err := db.Create(&endpoints[i]).Error; err != nil {
			log.Printf("⚠️ Gagal seeding inbound endpoint %s: %v", endpoints[i].Name, err)
		}
	}

	// 2. Seed Sample Inbound Logs
	if len(endpoints) > 0 {
		sampleLogs := []InboundWebhookLog{
			{
				ModelBase:         models.ModelBase{CompanyID: 1, Active: true},
				LogCode:           "INLOG/2026/00001",
				EndpointID:        endpoints[0].ID,
				EndpointKey:       endpoints[0].EndpointKey,
				SourceIP:          "103.208.22.45",
				HTTPMethod:        "POST",
				RequestHeaders:    `{"Content-Type":["application/json"],"User-Agent":["Midtrans-Webhook/2.0"],"X-Midtrans-Signature":["sha512=sample"]}`,
				PayloadJSON:       `{"order_id":"INV/2026/00001","status_code":"200","gross_amount":"5000000.00","transaction_status":"settlement","payment_type":"qris","transaction_time":"2026-10-09 18:30:00"}`,
				SignatureReceived: "sha512=sample_verified",
				SignatureValid:    true,
				ProcessStatus:     StatusProcessed,
				HTTPStatusCode:    200,
				ActionTaken:       "Faktur 'INV/2026/00001' berhasil dilunasi otomatis sebesar Rp 5000000.00 via Midtrans Payment Gateway Callback",
				TargetResID:       1,
				TargetResName:     "INV/2026/00001",
				DurationMs:        45,
			},
			{
				ModelBase:         models.ModelBase{CompanyID: 1, Active: true},
				LogCode:           "INLOG/2026/00002",
				EndpointID:        endpoints[1].ID,
				EndpointKey:       endpoints[1].EndpointKey,
				SourceIP:          "182.253.110.12",
				HTTPMethod:        "POST",
				RequestHeaders:    `{"Content-Type":["application/json"],"User-Agent":["JNE-Gateway/1.5"],"X-Signature":["sha256=sample"]}`,
				PayloadJSON:       `{"waybill":"JNE-EXP-992188","picking_name":"WH/OUT/00001","status":"delivered","recipient":"Security Gedung A","delivered_at":"2026-10-09 19:15:00"}`,
				SignatureReceived: "sha256=sample_verified",
				SignatureValid:    true,
				ProcessStatus:     StatusProcessed,
				HTTPStatusCode:    200,
				ActionTaken:       "Surat Jalan 'WH/OUT/00001' diperbarui dengan status kurir 'delivered'",
				TargetResID:       1,
				TargetResName:     "WH/OUT/00001",
				DurationMs:        32,
			},
		}

		for _, l := range sampleLogs {
			if err := db.Create(&l).Error; err != nil {
				log.Printf("⚠️ Gagal seeding inbound webhook log %s: %v", l.LogCode, err)
			}
		}
	}

	log.Println("✅ Inbound Webhook Endpoints & Sample Logs seeded successfully.")
	return nil
}
