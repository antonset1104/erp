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
		// Google Cloud Platform (GCP) Ecosystem Connectors
		{
			CompanyID:   1,
			Code:        "gcp_bigquery",
			Name:        "Google Cloud BigQuery Data Warehouse",
			Category:    "google_cloud",
			Status:      "connected",
			AuthType:    "oauth2",
			BaseURL:     "https://bigquery.googleapis.com/bigquery/v2",
			ConfigJSON:  `{"project_id": "goerp-enterprise-cloud", "dataset": "goerp_analytics", "location": "asia-southeast2"}`,
			WebhookURL:  "https://erp.local/api/v1/ai/gcp/sync",
			LastSyncAt:  &now,
			SyncStatus:  "success",
			SyncMessage: "Streaming GL, Sales, and Inventory Quants to BigQuery OLAP",
			IsActive:    true,
		},
		{
			CompanyID:   1,
			Code:        "gcp_pubsub",
			Name:        "Google Cloud Pub/Sub Event Bus",
			Category:    "google_cloud",
			Status:      "connected",
			AuthType:    "oauth2",
			BaseURL:     "https://pubsub.googleapis.com/v1",
			ConfigJSON:  `{"project_id": "goerp-enterprise-cloud", "topic": "goerp-domain-events", "region": "asia-southeast2"}`,
			WebhookURL:  "https://erp.local/api/v1/inbound-webhooks/gateway/gcp-pubsub-push",
			LastSyncAt:  &now,
			SyncStatus:  "success",
			SyncMessage: "Real-time domain event streaming active",
			IsActive:    true,
		},
		{
			CompanyID:   1,
			Code:        "gcp_gcs_archive",
			Name:        "Google Cloud Storage (GCS) Archive Vault",
			Category:    "google_cloud",
			Status:      "connected",
			AuthType:    "api_key",
			BaseURL:     "https://storage.googleapis.com/storage/v1",
			ConfigJSON:  `{"bucket": "goerp-enterprise-vault", "storage_class": "STANDARD", "location": "ASIA-SOUTHEAST2"}`,
			WebhookURL:  "https://erp.local/api/v1/ai/gcp/sync",
			LastSyncAt:  &now,
			SyncStatus:  "success",
			SyncMessage: "S3-interoperable attachment & financial audit archive ready",
			IsActive:    true,
		},
		// Popular AI Agent & LLM API Connectors
		{
			CompanyID:   1,
			Code:        "gcp_vertex_ai",
			Name:        "Google Cloud Vertex AI & Gemini 2.5 Agent",
			Category:    "ai_agent",
			Status:      "connected",
			AuthType:    "api_key",
			BaseURL:     "https://generativelanguage.googleapis.com/v1beta",
			ConfigJSON:  `{"model": "gemini-2.5-flash", "vertex_region": "asia-southeast2", "function_calling": true}`,
			WebhookURL:  "https://erp.local/api/v1/ai/agent/execute",
			LastSyncAt:  &now,
			SyncStatus:  "success",
			SyncMessage: "Gemini 2.5 Flash/Pro multimodal & tool-calling agent active",
			IsActive:    true,
		},
		{
			CompanyID:   1,
			Code:        "openai_agents",
			Name:        "OpenAI GPT-4o & Agents SDK API",
			Category:    "ai_agent",
			Status:      "connected",
			AuthType:    "bearer_token",
			BaseURL:     "https://api.openai.com/v1",
			ConfigJSON:  `{"model": "gpt-4o", "reasoning_model": "o3-mini", "structured_outputs": true}`,
			WebhookURL:  "https://erp.local/api/v1/ai/agent/execute",
			LastSyncAt:  &now,
			SyncStatus:  "success",
			SyncMessage: "OpenAI Function Calling & Responses API ready",
			IsActive:    true,
		},
		{
			CompanyID:   1,
			Code:        "anthropic_claude",
			Name:        "Anthropic Claude 3.7 Sonnet & MCP Server",
			Category:    "ai_agent",
			Status:      "connected",
			AuthType:    "api_key",
			BaseURL:     "https://api.anthropic.com/v1",
			ConfigJSON:  `{"model": "claude-3-7-sonnet-latest", "mcp_version": "2025-03-26", "extended_thinking": true}`,
			WebhookURL:  "https://erp.local/api/v1/ai/tools",
			LastSyncAt:  &now,
			SyncStatus:  "success",
			SyncMessage: "Model Context Protocol (MCP) & Claude tool-use ready",
			IsActive:    true,
		},
		{
			CompanyID:   1,
			Code:        "deepseek_ai",
			Name:        "DeepSeek R1 / V3 Reasoning Agent API",
			Category:    "ai_agent",
			Status:      "connected",
			AuthType:    "bearer_token",
			BaseURL:     "https://api.deepseek.com/v1",
			ConfigJSON:  `{"model": "deepseek-chat", "reasoner_model": "deepseek-reasoner", " financial_audit": true}`,
			WebhookURL:  "https://erp.local/api/v1/ai/agent/execute",
			LastSyncAt:  &now,
			SyncStatus:  "success",
			SyncMessage: "Chain-of-thought financial & supply chain reasoning ready",
			IsActive:    true,
		},
	}

	var existingCount int64
	db.Model(&IntegrationConnector{}).Count(&existingCount)
	if existingCount == 0 {
		log.Println("Seeding Universal 3rd-Party, Google Cloud Platform & AI Agent Integration Connectors...")
	}

	for _, conn := range connectors {
		var existing IntegrationConnector
		if err := db.Where("code = ? AND company_id = ?", conn.Code, conn.CompanyID).First(&existing).Error; err != nil {
			_ = db.Create(&conn)
		}
	}

	if existingCount == 0 {
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

		log.Printf("Seeded %d Pre-Built Connectors (including Google Cloud & AI Agents), 1 Master API Key, and Telemetry Logs successfully!\n", len(connectors))
	}
	return nil
}
