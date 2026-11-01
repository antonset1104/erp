package esignature

import (
	"time"

	"gorm.io/gorm"
)

// DocumentSignRequest represents an enterprise electronic signature & watermarking request
type DocumentSignRequest struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	Active    bool           `gorm:"default:true" json:"active"`

	RequestNumber string `gorm:"type:varchar(64);uniqueIndex;not null" json:"request_number"` // e.g. ESIGN/2026/00001
	CompanyID     uint   `gorm:"default:1;index" json:"company_id"`

	DocumentType   string `gorm:"type:varchar(32);not null;index" json:"document_type"`   // sale_order, purchase_order, invoice, delivery_order, payslip, etc.
	DocumentID     uint   `gorm:"not null;index" json:"document_id"`
	DocumentNumber string `gorm:"type:varchar(64);not null" json:"document_number"` // e.g. SO/2026/00001, INV/2026/00001
	Title          string `gorm:"type:varchar(255);not null" json:"title"`
	Description    string `gorm:"type:text" json:"description"`

	// Signer Details
	SignerName  string `gorm:"type:varchar(128);not null" json:"signer_name"`
	SignerEmail string `gorm:"type:varchar(128);not null" json:"signer_email"`
	SignerRole  string `gorm:"type:varchar(64);not null" json:"signer_role"` // Direktur Utama, Finance Manager, Vendor, etc.

	// Workflow State
	Status       string     `gorm:"type:varchar(32);default:'pending';index" json:"status"` // pending, signed, rejected, expired
	SecurityToken string    `gorm:"type:varchar(128);uniqueIndex;not null" json:"security_token"` // crypto random hex for public signing link
	PasscodeHash string     `gorm:"type:varchar(128)" json:"-"`                             // Optional 6-digit PIN protection
	ExpiryDate   *time.Time `json:"expiry_date"`

	// Signature Artifacts
	SignatureType           string     `gorm:"type:varchar(32);default:'draw'" json:"signature_type"` // draw, typed, upload, certificate_pkcs12
	SignatureData           string     `gorm:"type:text" json:"signature_data"`                       // Base64 PNG or PEM signature
	SignatureHash           string     `gorm:"type:varchar(128);index" json:"signature_hash"`         // SHA-256 document cryptographic hash
	CertificateSerialNumber string     `gorm:"type:varchar(128)" json:"certificate_serial_number"`
	SignedAt                *time.Time `json:"signed_at"`
	SignedIP                string     `gorm:"type:varchar(64)" json:"signed_ip"`
	SignedUserAgent         string     `gorm:"type:varchar(255)" json:"signed_user_agent"`

	// Watermarking & Visual Certification
	WatermarkEnabled bool   `gorm:"default:true" json:"watermark_enabled"`
	WatermarkText    string `gorm:"type:varchar(255)" json:"watermark_text"` // e.g. "DIGITALLY SIGNED & VERIFIED - PT MULTI HOLDING"
	WatermarkQRData  string `gorm:"type:text" json:"watermark_qr_data"`      // Verification URL + cryptographic proof

	// Rejection Reason
	RejectionReason string `gorm:"type:text" json:"rejection_reason"`

	// Audit Trail Log (JSON String Array of events)
	AuditTrailLog string `gorm:"type:text" json:"audit_trail_log"`
}

// SignerAuditLogEntry represents a single event in the tamper-evident audit trail
type SignerAuditLogEntry struct {
	Timestamp string `json:"timestamp"`
	Action    string `json:"action"` // created, viewed, signed, rejected, downloaded, verified
	Actor     string `json:"actor"`
	IP        string `json:"ip"`
	UserAgent string `json:"user_agent"`
	Details   string `json:"details"`
}

// DTOs
type CreateSignRequest struct {
	DocumentType     string `json:"document_type" binding:"required"`
	DocumentID       uint   `json:"document_id" binding:"required"`
	DocumentNumber   string `json:"document_number" binding:"required"`
	Title            string `json:"title" binding:"required"`
	Description      string `json:"description"`
	SignerName       string `json:"signer_name" binding:"required"`
	SignerEmail      string `json:"signer_email" binding:"required,email"`
	SignerRole       string `json:"signer_role" binding:"required"`
	WatermarkEnabled *bool  `json:"watermark_enabled"`
	WatermarkText    string `json:"watermark_text"`
	ExpiryDays       int    `json:"expiry_days"` // defaults to 7 days
}

type ExecuteSignRequest struct {
	SignatureType string `json:"signature_type" binding:"required"` // draw, typed, upload, certificate_pkcs12
	SignatureData string `json:"signature_data" binding:"required"` // base64 string
	WatermarkText string `json:"watermark_text"`
}

type RejectSignRequest struct {
	Reason string `json:"reason" binding:"required"`
}

type SignatureStats struct {
	TotalRequests    int64   `json:"total_requests"`
	PendingRequests  int64   `json:"pending_requests"`
	SignedRequests   int64   `json:"signed_requests"`
	RejectedRequests int64   `json:"rejected_requests"`
	AvgTurnaroundHrs float64 `json:"avg_turnaround_hrs"`
	ComplianceRate   float64 `json:"compliance_rate"`
}

// AIDigitalSignatureAuditReport represents UU ITE compliance evaluation
type AIDigitalSignatureAuditReport struct {
	AuditStatus         string   `json:"audit_status"` // VERIFIED_LEGAL, CAUTION, INVALID
	IntegrityScore      int      `json:"integrity_score"` // 0 - 100
	RequestNumber       string   `json:"request_number"`
	SignerIdentityMatch bool     `json:"signer_identity_match"`
	HashIntegrityValid  bool     `json:"hash_integrity_valid"`
	LegalBasis          string   `json:"legal_basis"` // e.g. "UU ITE No. 1/2024 & PP No. 71/2019"
	ExecutiveSummary    string   `json:"executive_summary"`
	EvidenceChecklist   []string `json:"evidence_checklist"`
	RiskRecommendations []string `json:"risk_recommendations"`
	AIProvider          string   `json:"ai_provider"`
}
