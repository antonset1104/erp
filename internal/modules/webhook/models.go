package webhook

import (
	"time"

	"erp-golang/internal/core/models"
)

// Delivery statuses
const (
	StatusSuccess   = "success"
	StatusFailed    = "failed"
	StatusRetrying  = "retrying"
	StatusCancelled = "cancelled"
)

// Supported Event Topics
const (
	EventSaleConfirmed     = "sale.order.confirmed"
	EventInvoicePaid       = "account.invoice.paid"
	EventPickingValidated  = "stock.picking.done"
	EventPartnerCreated    = "res.partner.created"
	EventSubscriptionChurn = "subscription.churned"
	EventWebhookPing       = "webhook.ping"
	EventWildcard          = "*"
)

// WebhookSubscription represents a configured outbound webhook listener (Enterprise ERP webhook / webhook.subscription equivalent)
type WebhookSubscription struct {
	models.ModelBase
	Name          string     `gorm:"size:128;not null" json:"name"`          // e.g. "Zapier CRM Sync", "Logistics 3PL Dispatch"
	TargetURL     string     `gorm:"size:512;not null" json:"target_url"`    // HTTPS endpoint URL
	SecretToken   string     `gorm:"size:128;not null" json:"secret_token"`  // Shared secret for HMAC-SHA256 signing
	EventTopics   string     `gorm:"type:text;not null" json:"event_topics"` // Comma-separated topics or "*"
	CustomHeaders string     `gorm:"type:text" json:"custom_headers"`        // JSON key-value headers (e.g. {"Authorization":"Bearer xyz"})
	MaxRetries    int        `gorm:"default:3" json:"max_retries"`           // Maximum retry attempts
	RetryDelaySec int        `gorm:"default:60" json:"retry_delay_sec"`      // Initial retry delay in seconds
	TimeoutSec    int        `gorm:"default:10" json:"timeout_sec"`          // Request timeout in seconds
	IsActive      bool       `gorm:"default:true;index" json:"is_active"`
	LastFiredAt   *time.Time `json:"last_fired_at,omitempty"`
	LastStatus    string     `gorm:"size:32;default:'idle'" json:"last_status"` // idle, success, failed
	Description   string     `gorm:"type:text" json:"description"`
}

// WebhookDelivery represents an immutable outbound delivery execution log
type WebhookDelivery struct {
	models.ModelBase
	DeliveryCode   string               `gorm:"size:64;not null;uniqueIndex" json:"delivery_code"` // e.g. "DLV/2026/00001"
	SubscriptionID uint                 `gorm:"not null;index" json:"subscription_id"`
	Subscription   *WebhookSubscription `gorm:"foreignKey:SubscriptionID" json:"subscription,omitempty"`
	EventName      string               `gorm:"size:128;not null;index" json:"event_name"` // e.g. "sale.order.confirmed"
	ModelName      string               `gorm:"size:64;index" json:"model_name"`           // e.g. "sale.order"
	ResID          uint                 `gorm:"index" json:"res_id"`                       // Target entity ID
	TargetURL      string               `gorm:"size:512;not null" json:"target_url"`
	RequestHeaders string               `gorm:"type:text" json:"request_headers"`       // Snapshot of headers sent (includes HMAC signature)
	PayloadJSON    string               `gorm:"type:text;not null" json:"payload_json"` // Exact JSON payload sent
	HMACSignature  string               `gorm:"size:128" json:"hmac_signature"`         // hex-encoded HMAC-SHA256 signature
	StatusCode     int                  `gorm:"default:0;index" json:"status_code"`     // HTTP status code (200, 404, 500, etc.)
	ResponseBody   string               `gorm:"type:text" json:"response_body"`         // First 4KB of remote response
	DurationMs     int64                `gorm:"default:0" json:"duration_ms"`           // Roundtrip latency in milliseconds
	Status         string               `gorm:"size:32;not null;index" json:"status"`   // success, failed, retrying, cancelled
	AttemptCount   int                  `gorm:"default:1" json:"attempt_count"`         // 1, 2, 3...
	NextRetryAt    *time.Time           `gorm:"index" json:"next_retry_at,omitempty"`   // Timestamp for background worker
	ErrorMessage   string               `gorm:"type:text" json:"error_message"`         // Connection refused, timeout, etc.
}

// ---------------------------------------------------------------------------
// DTOs & Request / Response Types
// ---------------------------------------------------------------------------

type CreateWebhookRequest struct {
	Name          string `json:"name" binding:"required"`
	TargetURL     string `json:"target_url" binding:"required"`
	SecretToken   string `json:"secret_token"` // Auto-generated if blank
	EventTopics   string `json:"event_topics" binding:"required"`
	CustomHeaders string `json:"custom_headers"`
	MaxRetries    int    `json:"max_retries"`
	RetryDelaySec int    `json:"retry_delay_sec"`
	TimeoutSec    int    `json:"timeout_sec"`
	Description   string `json:"description"`
}

type UpdateWebhookRequest struct {
	Name          string `json:"name"`
	TargetURL     string `json:"target_url"`
	SecretToken   string `json:"secret_token"`
	EventTopics   string `json:"event_topics"`
	CustomHeaders string `json:"custom_headers"`
	MaxRetries    int    `json:"max_retries"`
	RetryDelaySec int    `json:"retry_delay_sec"`
	TimeoutSec    int    `json:"timeout_sec"`
	IsActive      *bool  `json:"is_active"`
	Description   string `json:"description"`
}

type PublishEventRequest struct {
	EventName string      `json:"event_name" binding:"required"`
	ModelName string      `json:"model_name"`
	ResID     uint        `json:"res_id"`
	Data      interface{} `json:"data" binding:"required"`
}

type WebhookStats struct {
	TotalSubscriptions  int64   `json:"total_subscriptions"`
	ActiveSubscriptions int64   `json:"active_subscriptions"`
	TotalDeliveries     int64   `json:"total_deliveries"`
	SuccessDeliveries   int64   `json:"success_deliveries"`
	FailedDeliveries    int64   `json:"failed_deliveries"`
	RetryingDeliveries  int64   `json:"retrying_deliveries"`
	SuccessRatePct      float64 `json:"success_rate_pct"`
	AvgDurationMs       float64 `json:"avg_duration_ms"`
}

type AIWebhookDiagnosticReport struct {
	TotalAnalyzed       int               `json:"total_analyzed"`
	FailedDeliveries    int               `json:"failed_deliveries"`
	HealthStatus        string            `json:"health_status"` // "HEALTHY", "DEGRADED", "CRITICAL"
	IdentifiedIssues    []string          `json:"identified_issues"`
	RemediationActions  []string          `json:"remediation_actions"`
	PayloadOptimization string            `json:"payload_optimization"`
	ExecutiveSummary    string            `json:"executive_summary"`
	AIProvider          string            `json:"ai_provider"`
	SampleErrors        []WebhookDelivery `json:"sample_errors"`
}
