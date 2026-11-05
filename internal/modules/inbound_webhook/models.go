package inbound_webhook

import (
	"time"

	"erp-golang/internal/core/models"
)

// Provider types
const (
	ProviderPaymentMidtrans = "midtrans"
	ProviderPaymentXendit   = "xendit"
	ProviderLogistics3PL    = "logistics_3pl"
	ProviderCRMLeads        = "crm_leads"
	ProviderGeneric         = "generic"
)

// Auth / Signature verification types
const (
	AuthHMACSHA256 = "hmac_sha256"
	AuthSHA512     = "sha512"
	AuthBearer     = "bearer_token"
	AuthBasic      = "basic"
	AuthNone       = "none"
)

// Automated Ingestion Actions
const (
	ActionMarkInvoicePaid       = "mark_invoice_paid"
	ActionUpdatePickingTracking = "update_tracking_status"
	ActionCreateCRMLead         = "create_lead"
	ActionLogOnly               = "log_only"
)

// Processing Statuses
const (
	StatusProcessed = "processed"
	StatusIgnored   = "ignored"
	StatusRejected  = "rejected"
	StatusError     = "error"
)

// InboundWebhookEndpoint defines a secure target URL for external partners to push webhooks into the ERP
type InboundWebhookEndpoint struct {
	models.ModelBase
	EndpointCode string     `gorm:"size:64;not null;uniqueIndex" json:"endpoint_code"`       // e.g. "INWH/2026/00001"
	EndpointKey  string     `gorm:"size:64;not null;uniqueIndex" json:"endpoint_key"`        // URL slug: /api/v1/inbound-webhooks/gateway/:key
	Name         string     `gorm:"size:128;not null" json:"name"`                           // e.g. "Midtrans Payment Gateway Ingestion"
	ProviderType string     `gorm:"size:64;not null" json:"provider_type"`                   // midtrans, xendit, logistics_3pl, crm_leads, generic
	AuthType     string     `gorm:"size:32;not null;default:'hmac_sha256'" json:"auth_type"` // hmac_sha256, sha512, bearer_token, none
	SecretKey    string     `gorm:"size:256" json:"secret_key"`                              // Shared secret for signature verification
	TargetModel  string     `gorm:"size:64" json:"target_model"`                             // account.move, stock.picking, crm.lead
	AutoAction   string     `gorm:"size:64;not null;default:'log_only'" json:"auto_action"`  // mark_invoice_paid, update_tracking_status, create_lead, log_only
	AllowedIPs   string     `gorm:"size:512" json:"allowed_ips"`                             // Optional comma-separated IP whitelist
	IsActive     bool       `gorm:"default:true;index" json:"is_active"`
	TotalCalls   int64      `gorm:"default:0" json:"total_calls"`
	LastCalledAt *time.Time `json:"last_called_at,omitempty"`
	LastStatus   string     `gorm:"size:32;default:'idle'" json:"last_status"` // idle, processed, error
	Description  string     `gorm:"type:text" json:"description"`
}

// InboundWebhookLog represents an immutable record of every inbound HTTP POST webhook received
type InboundWebhookLog struct {
	models.ModelBase
	LogCode           string                  `gorm:"size:64;not null;uniqueIndex" json:"log_code"` // e.g. "INLOG/2026/00001"
	EndpointID        uint                    `gorm:"not null;index" json:"endpoint_id"`
	Endpoint          *InboundWebhookEndpoint `gorm:"foreignKey:EndpointID" json:"endpoint,omitempty"`
	EndpointKey       string                  `gorm:"size:64;not null;index" json:"endpoint_key"`
	SourceIP          string                  `gorm:"size:64" json:"source_ip"`
	HTTPMethod        string                  `gorm:"size:16;default:'POST'" json:"http_method"`
	RequestHeaders    string                  `gorm:"type:text" json:"request_headers"`
	PayloadJSON       string                  `gorm:"type:text;not null" json:"payload_json"`
	SignatureReceived string                  `gorm:"size:256" json:"signature_received"`
	SignatureValid    bool                    `gorm:"default:false;index" json:"signature_valid"`
	ProcessStatus     string                  `gorm:"size:32;not null;index" json:"process_status"` // processed, ignored, rejected, error
	HTTPStatusCode    int                     `gorm:"default:200;index" json:"http_status_code"`
	ActionTaken       string                  `gorm:"type:text" json:"action_taken"`
	ErrorMessage      string                  `gorm:"type:text" json:"error_message"`
	TargetResID       uint                    `gorm:"index" json:"target_res_id"`
	TargetResName     string                  `gorm:"size:128" json:"target_res_name"`
	DurationMs        int64                   `gorm:"default:0" json:"duration_ms"`
}

// ---------------------------------------------------------------------------
// DTOs & Request / Response Structures
// ---------------------------------------------------------------------------

type CreateInboundEndpointRequest struct {
	Name         string `json:"name" binding:"required"`
	EndpointKey  string `json:"endpoint_key"` // Auto-generated if empty
	ProviderType string `json:"provider_type" binding:"required"`
	AuthType     string `json:"auth_type"`
	SecretKey    string `json:"secret_key"`
	TargetModel  string `json:"target_model"`
	AutoAction   string `json:"auto_action" binding:"required"`
	AllowedIPs   string `json:"allowed_ips"`
	Description  string `json:"description"`
}

type UpdateInboundEndpointRequest struct {
	Name         string `json:"name"`
	ProviderType string `json:"provider_type"`
	AuthType     string `json:"auth_type"`
	SecretKey    string `json:"secret_key"`
	TargetModel  string `json:"target_model"`
	AutoAction   string `json:"auto_action"`
	AllowedIPs   string `json:"allowed_ips"`
	IsActive     *bool  `json:"is_active"`
	Description  string `json:"description"`
}

type SimulateInboundRequest struct {
	EndpointKey string            `json:"endpoint_key" binding:"required"`
	Headers     map[string]string `json:"headers"`
	Payload     interface{}       `json:"payload" binding:"required"`
}

type InboundWebhookStats struct {
	TotalEndpoints   int64   `json:"total_endpoints"`
	ActiveEndpoints  int64   `json:"active_endpoints"`
	TotalInboundLogs int64   `json:"total_inbound_logs"`
	ProcessedLogs    int64   `json:"processed_logs"`
	RejectedLogs     int64   `json:"rejected_logs"`
	ErrorLogs        int64   `json:"error_logs"`
	ValidSigPercent  float64 `json:"valid_sig_percent"`
	AvgDurationMs    float64 `json:"avg_duration_ms"`
}

type InboundLogFilters struct {
	EndpointID    uint   `form:"endpoint_id"`
	ProcessStatus string `form:"process_status"`
	EndpointKey   string `form:"endpoint_key"`
	Search        string `form:"search"`
	Limit         int    `form:"limit"`
	Offset        int    `form:"offset"`
	CompanyID     uint
}

type AIInboundPayloadReport struct {
	DetectedProvider   string            `json:"detected_provider"`
	SuggestedModel     string            `json:"suggested_model"`
	SuggestedAction    string            `json:"suggested_action"`
	ExtractedEntityID  string            `json:"extracted_entity_id"`
	ExtractedAmount    float64           `json:"extracted_amount"`
	ExtractedStatus    string            `json:"extracted_status"`
	FieldMappings      map[string]string `json:"field_mappings"`
	SecurityAssessment string            `json:"security_assessment"`
	ExecutiveSummary   string            `json:"executive_summary"`
	AIProvider         string            `json:"ai_provider"`
}
