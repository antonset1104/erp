package attachment

import (
	"time"

	"gorm.io/gorm"
)

// Attachment represents an uploaded document/file across any ERP model (based on standard enterprise ir.attachment)
type Attachment struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	Name          string         `gorm:"size:255;not null;index" json:"name"`
	ResModel      string         `gorm:"size:100;index" json:"res_model"` // e.g. "hr.expense", "account.move", "quality.alert", "approval.request", "document"
	ResID         uint           `gorm:"index" json:"res_id"`
	ResName       string         `gorm:"size:255" json:"res_name"` // e.g. "EXP/2026/00001", "INV/2026/00001"
	MimeType      string         `gorm:"size:128" json:"mimetype"`
	FileSize      int64          `json:"file_size"`
	StorageType   string         `gorm:"size:32" json:"storage_type"` // "r2" or "local"
	StoragePath   string         `gorm:"size:512" json:"storage_path"`
	FileURL       string         `gorm:"size:512" json:"file_url"`
	Checksum      string         `gorm:"size:64;index" json:"checksum"`
	Description   string         `gorm:"type:text" json:"description"`
	OCRStatus     string         `gorm:"size:32;default:'none'" json:"ocr_status"` // "none", "pending", "processed", "failed"
	OCRData       string         `gorm:"type:text" json:"ocr_data"`                // JSON string of ReceiptOCRResult
	CompanyID     uint           `gorm:"index;not null" json:"company_id"`
	CreatedByID   uint           `json:"created_by_id"`
	CreatedByName string         `gorm:"size:128" json:"created_by_name"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Attachment) TableName() string {
	return "ir_attachments"
}

// UploadAttachmentRequest represents input payload for uploading an attachment
type UploadAttachmentRequest struct {
	Name        string `json:"name" form:"name"`
	ResModel    string `json:"res_model" form:"res_model"`
	ResID       uint   `json:"res_id" form:"res_id"`
	ResName     string `json:"res_name" form:"res_name"`
	Description string `json:"description" form:"description"`
	AutoOCR     bool   `json:"auto_ocr" form:"auto_ocr"`
	TextHint    string `json:"text_hint" form:"text_hint"`
	FileBase64  string `json:"file_base64" form:"file_base64"` // For JSON uploads
}

// AttachmentFilter provides filtering options for listing attachments
type AttachmentFilter struct {
	ResModel  string
	ResID     uint
	OCRStatus string
	Search    string
	Limit     int
	Offset    int
}
