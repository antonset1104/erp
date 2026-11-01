package integration_hub

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"erp-golang/internal/core/models"
)

// APIKey represents a machine-to-machine authentication credential for 3rd-party applications
type APIKey struct {
	models.ModelBase
	CompanyID     uint       `json:"company_id" gorm:"index;not null"`
	Name          string     `json:"name" gorm:"size:128;not null"`          // e.g. "Shopify Store Connector", "Zapier Integration", "WMS Scanner App"
	KeyPrefix     string     `json:"key_prefix" gorm:"size:16;index"`        // e.g. "ak_live_a1b2" for identification
	KeyHash       string     `json:"-" gorm:"size:64;uniqueIndex;not null"`  // SHA-256 hash of the secret key
	Scopes        string     `json:"scopes" gorm:"size:512;default:'*'"`     // Comma-separated: "read:products,write:sales,read:inventory,*"
	AllowedIPs    string     `json:"allowed_ips" gorm:"size:256;default:'*'"` // Comma-separated IPs or "*"
	RateLimitRPM  int        `json:"rate_limit_rpm" gorm:"default:120"`      // Requests per minute limit
	TotalRequests int64      `json:"total_requests" gorm:"default:0"`
	LastUsedAt    *time.Time `json:"last_used_at"`
	ExpiresAt     *time.Time `json:"expires_at"`
	IsActive      bool       `json:"is_active" gorm:"default:true"`
	CreatedBy     uint       `json:"created_by"`
}

func (APIKey) TableName() string {
	return "integration_api_keys"
}

// GenerateRawKey creates a cryptographically secure random API key with prefix
func GenerateRawKey(isLive bool) (string, string, string) {
	bytes := make([]byte, 24)
	rand.Read(bytes)
	hexStr := hex.EncodeToString(bytes)

	prefix := "ak_test_"
	if isLive {
		prefix = "ak_live_"
	}
	prefix += hexStr[:6]
	fullKey := prefix + "_" + hexStr[6:]
	hash := HashAPIKey(fullKey)

	return fullKey, prefix, hash
}

// HashAPIKey computes the SHA-256 hex digest of a key
func HashAPIKey(rawKey string) string {
	h := sha256.Sum256([]byte(rawKey))
	return hex.EncodeToString(h[:])
}

// HasScope checks if the API key has a specific scope or wildcard "*"
func (k *APIKey) HasScope(requiredScope string) bool {
	if k.Scopes == "*" || strings.Contains(k.Scopes, "*") {
		return true
	}
	scopes := strings.Split(k.Scopes, ",")
	for _, s := range scopes {
		if strings.TrimSpace(s) == requiredScope {
			return true
		}
	}
	return false
}

// IntegrationConnector represents a pre-built adapter to an external platform
type IntegrationConnector struct {
	models.ModelBase
	CompanyID   uint       `json:"company_id" gorm:"index;not null"`
	Code        string     `json:"code" gorm:"size:64;index;not null"`    // e.g. "shopee_ecommerce", "midtrans_pg", "jne_logistics", "waba_whatsapp", "bca_h2h"
	Name        string     `json:"name" gorm:"size:128;not null"`
	Category    string     `json:"category" gorm:"size:32;not null"`      // "ecommerce", "payment_gateway", "logistics_3pl", "messaging", "banking", "custom"
	Status      string     `json:"status" gorm:"size:32;default:'idle'"`  // "connected", "disconnected", "error", "idle"
	AuthType    string     `json:"auth_type" gorm:"size:32;default:'api_key'"` // "api_key", "bearer_token", "oauth2", "webhook_secret"
	BaseURL     string     `json:"base_url" gorm:"size:256"`
	ConfigJSON  string     `json:"config_json" gorm:"type:text"`          // Stores JSON configuration (credentials, merchant_id, headers)
	WebhookURL  string     `json:"webhook_url" gorm:"size:256"`
	LastSyncAt  *time.Time `json:"last_sync_at"`
	SyncStatus  string     `json:"sync_status" gorm:"size:32;default:'idle'"` // "success", "failed", "in_progress", "idle"
	SyncMessage string     `json:"sync_message" gorm:"size:256"`
	IsActive    bool       `json:"is_active" gorm:"default:true"`
}

func (IntegrationConnector) TableName() string {
	return "integration_connectors"
}

// IntegrationLog records telemetry and audit trails for all external integrations
type IntegrationLog struct {
	models.ModelBase
	CompanyID      uint   `json:"company_id" gorm:"index;not null"`
	LogCode        string `json:"log_code" gorm:"size:32;uniqueIndex;not null"`
	RequestID      string `json:"request_id" gorm:"size:64;index"`
	APIKeyID       *uint  `json:"api_key_id" gorm:"index"`
	APIKeyName     string `json:"api_key_name" gorm:"size:128"`
	CallerIP       string `json:"caller_ip" gorm:"size:64"`
	AuthType       string `json:"auth_type" gorm:"size:32"` // "api_key", "jwt", "public_webhook", "connector"
	Method         string `json:"method" gorm:"size:16"`
	Endpoint       string `json:"endpoint" gorm:"size:256;index"`
	Model          string `json:"model" gorm:"size:64;index"`
	Action         string `json:"action" gorm:"size:64"`
	StatusCode     int    `json:"status_code" gorm:"index"`
	DurationMs     int64  `json:"duration_ms"`
	RequestSummary string `json:"request_summary" gorm:"type:text"`
	ErrorMessage   string `json:"error_message" gorm:"type:text"`
}

func (IntegrationLog) TableName() string {
	return "integration_logs"
}

// IntegrationDataJob manages batch ETL data synchronization (import/export)
type IntegrationDataJob struct {
	models.ModelBase
	CompanyID        uint   `json:"company_id" gorm:"index;not null"`
	JobCode          string `json:"job_code" gorm:"size:32;uniqueIndex;not null"`
	JobType          string `json:"job_type" gorm:"size:16;not null"` // "import", "export"
	TargetModel      string `json:"target_model" gorm:"size:64;not null"` // "res.partner", "product.product", "sale.order", etc.
	FileFormat       string `json:"file_format" gorm:"size:16;not null"` // "csv", "json", "xml"
	Status           string `json:"status" gorm:"size:32;default:'pending'"` // "pending", "processing", "completed", "failed"
	FieldMappingJSON string `json:"field_mapping_json" gorm:"type:text"`
	TotalRecords     int    `json:"total_records" gorm:"default:0"`
	ProcessedRecords int    `json:"processed_records" gorm:"default:0"`
	SuccessRecords   int    `json:"success_records" gorm:"default:0"`
	FailedRecords    int    `json:"failed_records" gorm:"default:0"`
	ErrorLog         string `json:"error_log" gorm:"type:text"`
}

func (IntegrationDataJob) TableName() string {
	return "integration_data_jobs"
}

// ---------------- DTOs ----------------

type CreateAPIKeyRequest struct {
	Name         string     `json:"name" binding:"required"`
	Scopes       string     `json:"scopes"`         // default: "*"
	AllowedIPs   string     `json:"allowed_ips"`    // default: "*"
	RateLimitRPM int        `json:"rate_limit_rpm"` // default: 120
	ExpiresAt    *time.Time `json:"expires_at"`
	IsLive       bool       `json:"is_live"`
}

type CreateAPIKeyResponse struct {
	ID        uint       `json:"id"`
	Name      string     `json:"name"`
	KeyPrefix string     `json:"key_prefix"`
	SecretKey string     `json:"secret_key"` // Shown only once upon creation
	Scopes    string     `json:"scopes"`
	ExpiresAt *time.Time `json:"expires_at"`
	Message   string     `json:"message"`
}

type RPCQueryRequest struct {
	Model  string          `json:"model" binding:"required"` // e.g. "res.partner", "product.product", "sale.order"
	Domain [][]interface{} `json:"domain"`                   // e.g. [["is_company", "=", true], ["customer_rank", ">", 0]]
	Fields []string        `json:"fields"`                   // e.g. ["id", "name", "email", "phone"]
	Limit  int             `json:"limit"`                    // default: 50, max: 200
	Offset int             `json:"offset"`                   // default: 0
	Order  string          `json:"order"`                    // e.g. "id desc"
}

type RPCCreateRequest struct {
	Model  string                 `json:"model" binding:"required"`
	Values map[string]interface{} `json:"values" binding:"required"`
}

type RPCWriteRequest struct {
	Model  string                 `json:"model" binding:"required"`
	ID     uint                   `json:"id" binding:"required"`
	Values map[string]interface{} `json:"values" binding:"required"`
}

type RPCUnlinkRequest struct {
	Model string `json:"model" binding:"required"`
	ID    uint   `json:"id" binding:"required"`
}

type RPCExecuteActionRequest struct {
	Model   string                 `json:"model" binding:"required"`
	ID      uint                   `json:"id" binding:"required"`
	Action  string                 `json:"action" binding:"required"` // e.g. "action_confirm", "action_post", "action_cancel"
	Context map[string]interface{} `json:"context"`
}

type RPCResponse struct {
	Success bool        `json:"success"`
	Model   string      `json:"model"`
	Action  string      `json:"action,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Count   int         `json:"count,omitempty"`
	Total   int64       `json:"total,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type ExecuteConnectorActionRequest struct {
	ConnectorCode string                 `json:"connector_code" binding:"required"`
	Action        string                 `json:"action" binding:"required"` // "sync_stock", "create_charge", "generate_awb", "send_whatsapp", "test_connection"
	Payload       map[string]interface{} `json:"payload"`
}

type ExecuteConnectorActionResponse struct {
	Success      bool                   `json:"success"`
	Connector    string                 `json:"connector"`
	Action       string                 `json:"action"`
	ExternalID   string                 `json:"external_id,omitempty"`
	ResponseData map[string]interface{} `json:"response_data,omitempty"`
	Message      string                 `json:"message"`
}

type RunBatchDataJobRequest struct {
	TargetModel      string                 `json:"target_model" binding:"required"`
	JobType          string                 `json:"job_type" binding:"required"` // "import", "export"
	FileFormat       string                 `json:"file_format" binding:"required"` // "json", "csv"
	FieldMapping     map[string]string      `json:"field_mapping"`
	DataRows         []map[string]interface{}`json:"data_rows,omitempty"` // For import
}

type AIIntegrationArchitectRequest struct {
	Requirement string `json:"requirement" binding:"required"` // e.g. "Sinkronisasi pesanan dari Tokopedia ke Odoo-Go", "Integrasi POS Kasir Cabang", "Automasi Zapier lead CRM"
	TargetApp   string `json:"target_app"`   // e.g. "Shopify", "Tokopedia", "Zapier", "Custom Mobile App"
	Language    string `json:"language"`     // "curl", "python", "javascript", "go", "all"
}

type AIIntegrationArchitectReport struct {
	UseCase             string   `json:"use_case"`
	TargetApp           string   `json:"target_app"`
	IntegrationPattern  string   `json:"integration_pattern"` // "REST RPC Inbound", "Outbound Webhooks", "Batch ETL", "Hybrid"
	RecommendedEndpoint string   `json:"recommended_endpoint"`
	RequiredScopes      []string `json:"required_scopes"`
	ArchitectureFlow    string   `json:"architecture_flow"`
	CurlSnippet         string   `json:"curl_snippet"`
	PythonSnippet       string   `json:"python_snippet"`
	NodeSnippet         string   `json:"node_snippet"`
	GoSnippet           string   `json:"go_snippet"`
	SecurityGuidelines  string   `json:"security_guidelines"`
	AIProvider          string   `json:"ai_provider"`
}

type IntegrationHubStats struct {
	TotalAPIKeys         int64   `json:"total_api_keys"`
	ActiveAPIKeys        int64   `json:"active_api_keys"`
	TotalConnectors      int64   `json:"total_connectors"`
	ConnectedConnectors  int64   `json:"connected_connectors"`
	TotalCalls24h        int64   `json:"total_calls_24h"`
	SuccessCalls24h      int64   `json:"success_calls_24h"`
	FailedCalls24h       int64   `json:"failed_calls_24h"`
	SuccessRatePercent   float64 `json:"success_rate_percent"`
	AvgLatencyMs         float64 `json:"avg_latency_ms"`
	TotalDataJobs        int64   `json:"total_data_jobs"`
}

type ModelMetadata struct {
	Model       string   `json:"model"`
	TableName   string   `json:"table_name"`
	Description string   `json:"description"`
	AllowedOps  []string `json:"allowed_ops"` // ["search_read", "create", "write", "unlink", "execute_action"]
	KeyFields   []string `json:"key_fields"`
}
