package approval

import (
	"time"

	"erp-golang/internal/core/models"
	"erp-golang/internal/modules/base"
)

// ApprovalCategory defines approval templates and governance rules (approval.category in Enterprise ERP)
type ApprovalCategory struct {
	models.ModelBase
	Name                 string  `gorm:"size:128;not null" json:"name"`
	Description          string  `gorm:"type:text" json:"description"`
	SequenceCode         string  `gorm:"size:32;default:'APPR'" json:"sequence_code"`
	ApprovalType         string  `gorm:"size:32;default:'general'" json:"approval_type"` // general, capex, purchase, discount, payment
	HasAmount            bool    `gorm:"default:false" json:"has_amount"`
	MinAmount            float64 `gorm:"type:decimal(16,2);default:0" json:"min_amount"` // Threshold to trigger high-level approvals
	HasDate              bool    `gorm:"default:false" json:"has_date"`
	MinApprovalsRequired int     `gorm:"default:1" json:"min_approvals_required"` // Number of mandatory approvals
	IsManagerApprover    bool    `gorm:"default:true" json:"is_manager_approver"`  // Direct manager is mandatory level 1
	Active               bool    `gorm:"default:true" json:"active"`
}

func (ApprovalCategory) TableName() string {
	return "approval_categories"
}

// ApprovalRequest represents a specific business approval workflow (approval.request in Enterprise ERP)
type ApprovalRequest struct {
	models.ModelBase
	Name           string             `gorm:"size:255;not null" json:"name"`
	Code           string             `gorm:"size:64;uniqueIndex" json:"code"`
	CategoryID     uint               `gorm:"not null" json:"category_id"`
	Category       ApprovalCategory   `gorm:"foreignKey:CategoryID" json:"category"`
	RequestOwnerID uint               `gorm:"not null" json:"request_owner_id"`
	RequestOwner   base.User          `gorm:"foreignKey:RequestOwnerID" json:"request_owner"`
	Reason         string             `gorm:"type:text" json:"reason"`
	Amount         float64            `gorm:"type:decimal(16,2);default:0" json:"amount"`
	CurrencyID     uint               `gorm:"default:1" json:"currency_id"`
	DateStart      *time.Time         `json:"date_start"`
	DateEnd        *time.Time         `json:"date_end"`
	Reference      string             `gorm:"size:128" json:"reference"` // Associated document e.g. PO00001, SO00002, EXP/2026/00001
	Stage          string             `gorm:"size:32;default:'new'" json:"stage"` // new, pending, approved, refused, cancelled
	RefusalReason  string             `gorm:"type:text" json:"refusal_reason"`
	Approvers      []ApprovalApprover `gorm:"foreignKey:RequestID;constraint:OnDelete:CASCADE" json:"approvers"`
}

func (ApprovalRequest) TableName() string {
	return "approval_requests"
}

// ApprovalApprover represents each approval tier in the matrix (approval.approver in Enterprise ERP)
type ApprovalApprover struct {
	models.ModelBase
	RequestID    uint       `gorm:"not null;index" json:"request_id"`
	UserID       uint       `gorm:"not null" json:"user_id"`
	User         base.User  `gorm:"foreignKey:UserID" json:"user"`
	Sequence     int        `gorm:"default:10" json:"sequence"` // Tier order: 1 = Manager, 2 = Finance, 3 = Executive Director
	Required     bool       `gorm:"default:true" json:"required"`
	Status       string     `gorm:"size:32;default:'new'" json:"status"` // new, waiting, approved, refused, cancelled
	Comment      string     `gorm:"type:text" json:"comment"`
	ApprovalDate *time.Time `json:"approval_date"`
}

func (ApprovalApprover) TableName() string {
	return "approval_approvers"
}
