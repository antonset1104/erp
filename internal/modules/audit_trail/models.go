package audit_trail

import (
	"erp-golang/internal/core/models"
)

// Actions tracked by Change Data Capture (CDC)
const (
	ActionCreate   = "CREATE"
	ActionUpdate   = "UPDATE"
	ActionDelete   = "DELETE"
	ActionRead     = "READ"
	ActionRollback = "ROLLBACK"
)

// AuditLogRule defines what models and operations are tracked (Enterprise ERP auditlog.rule equivalent)
type AuditLogRule struct {
	models.ModelBase
	Name          string `gorm:"size:128;not null" json:"name"`            // e.g. "Audit Jurnal & Faktur Keuangan"
	ModelName     string `gorm:"size:64;not null;index" json:"model_name"` // e.g. "account.move", "hr.employee", "sale.order"
	LogCreate     bool   `gorm:"default:true" json:"log_create"`
	LogWrite      bool   `gorm:"default:true" json:"log_write"`
	LogUnlink     bool   `gorm:"default:true" json:"log_unlink"`
	LogRead       bool   `gorm:"default:false" json:"log_read"`
	TrackFields   string `gorm:"type:text" json:"track_fields"`            // Comma separated list of fields or "*" for all
	CaptureDiff   bool   `gorm:"default:true" json:"capture_diff"`         // Capture before vs after JSON diff
	Active        bool   `gorm:"default:true" json:"active"`
	RetentionDays int    `gorm:"default:365" json:"retention_days"`        // SOX / ISO 27001 retention period in days (0=forever)
	Description   string `gorm:"type:text" json:"description"`
}

// AuditLogEntry is the immutable record of a database change event (CDC Entry)
type AuditLogEntry struct {
	models.ModelBase
	LogCode       string              `gorm:"size:64;not null;uniqueIndex" json:"log_code"` // e.g. "AUD/2026/00001"
	ModelName     string              `gorm:"size:64;not null;index" json:"model_name"`     // e.g. "account.move"
	ResID         uint                `gorm:"not null;index" json:"res_id"`                 // Target entity primary key
	ResName       string              `gorm:"size:255" json:"res_name"`                     // e.g. "INV/2026/00001"
	Action        string              `gorm:"size:32;not null;index" json:"action"`         // CREATE, UPDATE, DELETE, READ, ROLLBACK
	UserID        uint                `gorm:"index" json:"user_id"`
	UserName      string              `gorm:"size:128" json:"user_name"`
	UserEmail     string              `gorm:"size:128" json:"user_email"`
	IPAddress     string              `gorm:"size:64" json:"ip_address"`
	UserAgent     string              `gorm:"size:255" json:"user_agent"`
	Timestamp     string              `gorm:"size:32;not null;index" json:"timestamp"` // "YYYY-MM-DD HH:MM:SS"
	Summary       string              `gorm:"type:text" json:"summary"`
	OldValuesJSON string              `gorm:"type:text" json:"old_values_json"`
	NewValuesJSON string              `gorm:"type:text" json:"new_values_json"`
	Diffs         []AuditLogFieldDiff `gorm:"foreignKey:AuditLogID" json:"diffs"`
	IsSuspicious  bool                `gorm:"default:false;index" json:"is_suspicious"` // Flagged by anomaly detection
	AnomalyReason string              `gorm:"type:text" json:"anomaly_reason"`
	RiskScore     int                 `gorm:"default:0" json:"risk_score"`              // 0 to 100
}

// AuditLogFieldDiff contains granular field-level modification details
type AuditLogFieldDiff struct {
	models.ModelBase
	AuditLogID  uint   `gorm:"not null;index" json:"audit_log_id"`
	FieldName   string `gorm:"size:64;not null;index" json:"field_name"` // e.g. "amount_total", "wage", "state"
	OldValue    string `gorm:"type:text" json:"old_value"`
	NewValue    string `gorm:"type:text" json:"new_value"`
	ValueType   string `gorm:"size:32;default:'string'" json:"value_type"` // string, float, int, bool, date
	IsSensitive bool   `gorm:"default:false" json:"is_sensitive"`          // Sensitive financial / PII / salary field
}

// ---------------------------------------------------------------------------
// DTOs & Request / Response Types
// ---------------------------------------------------------------------------

// RecordChangeRequest payload for submitting changes to the CDC Engine
type RecordChangeRequest struct {
	ModelName string                 `json:"model_name" binding:"required"`
	ResID     uint                   `json:"res_id" binding:"required"`
	ResName   string                 `json:"res_name"`
	Action    string                 `json:"action" binding:"required"`
	OldValues map[string]interface{} `json:"old_values"`
	NewValues map[string]interface{} `json:"new_values"`
	IPAddress string                 `json:"ip_address"`
	UserAgent string                 `json:"user_agent"`
	Summary   string                 `json:"summary"`
}

// RollbackRequest payload to revert a past change
type RollbackRequest struct {
	Reason string `json:"reason" binding:"required"`
}

// CreateRuleRequest payload to configure audit rules
type CreateRuleRequest struct {
	Name          string `json:"name" binding:"required"`
	ModelName     string `json:"model_name" binding:"required"`
	LogCreate     *bool  `json:"log_create"`
	LogWrite      *bool  `json:"log_write"`
	LogUnlink     *bool  `json:"log_unlink"`
	LogRead       *bool  `json:"log_read"`
	TrackFields   string `json:"track_fields"`
	CaptureDiff   *bool  `json:"capture_diff"`
	RetentionDays int    `json:"retention_days"`
	Description   string `json:"description"`
}

// AuditStats aggregated metrics for security dashboard
type AuditStats struct {
	TotalLogs          int64            `json:"total_logs"`
	CreateCount        int64            `json:"create_count"`
	UpdateCount        int64            `json:"update_count"`
	DeleteCount        int64            `json:"delete_count"`
	RollbackCount      int64            `json:"rollback_count"`
	SuspiciousCount    int64            `json:"suspicious_count"`
	TrackedModelsCount int64            `json:"tracked_models_count"`
	TopModels          map[string]int64 `json:"top_models"`
	TopUsers           map[string]int64 `json:"top_users"`
}

// AIAuditAnomalyReport from Cloudflare Workers AI Security & Anomaly Audit Copilot
type AIAuditAnomalyReport struct {
	TotalAnalyzed        int             `json:"total_analyzed"`
	SuspiciousCount      int             `json:"suspicious_count"`
	OverallSecurityScore int             `json:"overall_security_score"` // 0-100 (100 = completely secure)
	ComplianceStatus     string          `json:"compliance_status"`      // "COMPLIANT", "AUDIT_WARNING", "NON_COMPLIANT"
	FlaggedEvents        []AuditLogEntry `json:"flagged_events"`
	KeyObservations      []string        `json:"key_observations"`
	RemediationActions   []string        `json:"remediation_actions"`
	ExecutiveSummary     string          `json:"executive_summary"`
	AIProvider           string          `json:"ai_provider"`
}
