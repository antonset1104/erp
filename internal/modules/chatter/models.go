package chatter

import (
	"erp-golang/internal/core/models"
)

// MailMessage corresponds to standard enterprise mail.message (Audit Trail, Notes, and Notifications)
type MailMessage struct {
	models.ModelBase
	ModelName   string `gorm:"size:64;not null;index:idx_model_res" json:"model_name"` // e.g. "sale.order", "account.move"
	ResID       uint   `gorm:"not null;index:idx_model_res" json:"res_id"`             // ID of the target document
	AuthorID    uint   `json:"author_id"`
	AuthorName  string `gorm:"size:128" json:"author_name"`
	Body        string `gorm:"type:text;not null" json:"body"`                         // Comment / Message text
	MessageType string `gorm:"size:32;default:'comment'" json:"message_type"`          // "comment", "notification", "tracking"
	Tracking    string `gorm:"size:255" json:"tracking,omitempty"`                     // e.g. "Status: draft -> confirmed"
}
