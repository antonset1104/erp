package integration_hub

import (
	"log"
	"time"

	"erp-golang/internal/core/ai"
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
	return "integration_hub"
}

func (m *Module) Service() *Service {
	return m.service
}

func (m *Module) RegisterRoutes(api *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	m.handler.RegisterRoutes(api, authMiddleware)
}

func (m *Module) AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&APIKey{},
		&IntegrationConnector{},
		&IntegrationLog{},
		&IntegrationDataJob{},
	)
}

func (m *Module) Seed(db *gorm.DB) error {
	return m.InitMasterData(db)
}

func (m *Module) InitMasterData(db *gorm.DB) error {
	var count int64
	db.Model(&IntegrationConnector{}).Count(&count)
	if count > 0 {
		return nil
	}

	log.Println("Seeding Universal 3rd-Party Integration Connectors and Master Data...")

	now := time.Now()
	connectors := []IntegrationConnector{
		{
			CompanyID:   1,
			Code:        "shopee_ecommerce",
			Name:        "Shopee Open Platform (E-Commerce)",
			Category:    "ecommerce",
			Status:      "connected",
			AuthType:    "oauth2",
			BaseURL:     "https://partner.shopeemobile.com/api/v2",
			ConfigJSON:  `{"partner_id": "2004819", "shop_id": "84729103", "refresh_token": "shopee_tok_demo_982"}`,
			WebhookURL:  "https://erp.local/api/v1/inbound-webhooks/gateway/shopee-orders",
			LastSyncAt:  &now,
			SyncStatus:  "success",
			SyncMessage: "Synchronized 142 SKUs stock levels",
			IsActive:    true,
		},
		{
			CompanyID:   1,
			Code:        "tokopedia_ecommerce",
			Name:        "Tokopedia / TikTok Shop Seller API",
			Category:    "ecommerce",
			Status:      "connected",
			AuthType:    "oauth2",
			BaseURL:     "https://fs.tokopedia.net/v1",
			ConfigJSON:  `{"fs_id": "18492", "client_id": "tokped_client_live", "shop_id": "938102"}`,
			WebhookURL:  "https://erp.local/api/v1/inbound-webhooks/gateway/tokped-orders",
			LastSyncAt:  &now,
			SyncStatus:  "success",
			SyncMessage: "Orders synced in realtime",
			IsActive:    true,
		},
		{
			CompanyID:   1,
			Code:        "midtrans_pg",
			Name:        "Midtrans Snap & Core Payment Gateway",
			Category:    "payment_gateway",
			Status:      "connected",
			AuthType:    "api_key",
			BaseURL:     "https://api.midtrans.com/v2",
			ConfigJSON:  `{"merchant_id": "G84910284", "server_key_masked": "Mid-server-J18***9a"}`,
			WebhookURL:  "https://erp.local/api/v1/inbound-webhooks/gateway/midtrans-prod",
			LastSyncAt:  &now,
			SyncStatus:  "success",
			SyncMessage: "QRIS and Virtual Account notifications active",
			IsActive:    true,
		},
		{
			CompanyID:   1,
			Code:        "xendit_pg",
			Name:        "Xendit XenPlatform Payments & Invoicing",
			Category:    "payment_gateway",
			Status:      "connected",
			AuthType:    "api_key",
			BaseURL:     "https://api.xendit.co",
			ConfigJSON:  `{"public_key": "xnd_public_demo", "secret_key_masked": "xnd_development_***"}`,
			WebhookURL:  "https://erp.local/api/v1/inbound-webhooks/gateway/xendit-invoices",
			LastSyncAt:  &now,
			SyncStatus:  "success",
			SyncMessage: "Payouts & disbursement webhooks verified",
			IsActive:    true,
		},
		{
			CompanyID:   1,
			Code:        "jne_logistics",
			Name:        "JNE Express Logistics & Courier 3PL",
			Category:    "logistics_3pl",
			Status:      "connected",
			AuthType:    "api_key",
			BaseURL:     "https://api.jne.co.id/tracing/api/list",
			ConfigJSON:  `{"username": "PT_ENTERPRISE_JNE", "api_key_masked": "jne_live_key_***"}`,
			WebhookURL:  "https://erp.local/api/v1/inbound-webhooks/gateway/jne-logistics",
			LastSyncAt:  &now,
			SyncStatus:  "success",
			SyncMessage: "Auto AWB generation and live tracking active",
			IsActive:    true,
		},
		{
			CompanyID:   1,
			Code:        "waba_whatsapp",
			Name:        "WhatsApp Business Cloud API (Meta)",
			Category:    "messaging",
			Status:      "connected",
			AuthType:    "bearer_token",
			BaseURL:     "https://graph.facebook.com/v19.0",
			ConfigJSON:  `{"phone_number_id": "1049281048", "waba_account_id": "29481029"}`,
			WebhookURL:  "https://erp.local/api/v1/inbound-webhooks/gateway/whatsapp-inbound",
			LastSyncAt:  &now,
			SyncStatus:  "success",
			SyncMessage: "Transactional invoice & delivery notifications active",
			IsActive:    true,
		},
		{
			CompanyID:   1,
			Code:        "bca_h2h",
			Name:        "Bank Central Asia (BCA) KlikBCA Host-to-Host",
			Category:    "banking",
			Status:      "connected",
			AuthType:    "oauth2",
			BaseURL:     "https://api.klikbca.com",
			ConfigJSON:  `{"corporate_id": "BCAKORP001", "client_id": "bca_api_live"}`,
			WebhookURL:  "https://erp.local/api/v1/inbound-webhooks/gateway/bca-h2h-statement",
			LastSyncAt:  &now,
			SyncStatus:  "success",
			SyncMessage: "MT940 auto bank statement reconciliation active",
			IsActive:    true,
		},
		{
			CompanyID:   1,
			Code:        "slack_alerts",
			Name:        "Slack Enterprise Webhook Alerts",
			Category:    "messaging",
			Status:      "connected",
			AuthType:    "webhook_secret",
			BaseURL:     "https://hooks.slack.com/services",
			ConfigJSON:  `{"channel": "#erp-finance-alerts", "bot_name": "GoERP Bot"}`,
			WebhookURL:  "https://hooks.slack.com/services/T00/B00/X00",
			LastSyncAt:  &now,
			SyncStatus:  "success",
			SyncMessage: "System anomaly & high-value approval alerts active",
			IsActive:    true,
		},
	}

	for _, conn := range connectors {
		_ = db.Create(&conn)
	}

	// Seed Demo API Key
	demoRawKey := "ak_live_enterprise_demo_key_9876543210abcdef"
	demoHash := HashAPIKey(demoRawKey)
	apiKey := APIKey{
		CompanyID:     1,
		Name:          "Enterprise M2M Master API Key (Production)",
		KeyPrefix:     "ak_live_enterp",
		KeyHash:       demoHash,
		Scopes:        "*",
		AllowedIPs:    "*",
		RateLimitRPM:  240,
		TotalRequests: 1840,
		LastUsedAt:    &now,
		IsActive:      true,
		CreatedBy:     1,
	}
	_ = db.Create(&apiKey)

	// Seed Sample Integration Telemetry Log
	sampleLog := IntegrationLog{
		CompanyID:      1,
		LogCode:        "INTG/2026/00001",
		RequestID:      "REQ-1728391-200",
		APIKeyID:       &apiKey.ID,
		APIKeyName:     apiKey.Name,
		CallerIP:       "127.0.0.1",
		AuthType:       "api_key",
		Method:         "POST",
		Endpoint:       "/api/v1/integration/rpc/search_read",
		Model:          "product.product",
		Action:         "search_read",
		StatusCode:     200,
		DurationMs:     18,
		RequestSummary: "Filtered 50 sellable products for marketplace sync",
		ErrorMessage:   "",
	}
	_ = db.Create(&sampleLog)

	log.Println("Seeded 8 Pre-Built Connectors, 1 Master API Key, and Telemetry Logs successfully!")
	return nil
}
