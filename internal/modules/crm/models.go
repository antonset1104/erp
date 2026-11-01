package crm

import (
	"erp-golang/internal/core/models"
	"erp-golang/internal/modules/base"
)

// CrmLead corresponds to Odoo's crm.lead (Leads & Opportunities Pipeline)
type CrmLead struct {
	models.ModelBase
	Name            string        `gorm:"size:255;not null" json:"name"` // Opportunity Title e.g. "Pengadaan Laptop Kantor"
	PartnerID       *uint         `json:"partner_id,omitempty"`
	Partner         *base.Partner `gorm:"foreignKey:PartnerID" json:"partner,omitempty"`
	PartnerName     string        `gorm:"size:255" json:"partner_name"`
	ContactName     string        `gorm:"size:128" json:"contact_name"`
	Email           string        `gorm:"size:128" json:"email"`
	Phone           string        `gorm:"size:64" json:"phone"`
	ExpectedRevenue float64       `gorm:"default:0" json:"expected_revenue"`
	Probability     float64       `gorm:"default:20" json:"probability"` // Probability in % (0 - 100)
	Stage           string        `gorm:"size:32;default:'new';index" json:"stage"` // "new", "qualified", "proposition", "won", "lost"
	Priority        string        `gorm:"size:16;default:'1'" json:"priority"`      // "0" (Low), "1" (Medium), "2" (High), "3" (Urgent)
	UserID          *uint         `json:"user_id,omitempty"`                        // Sales Representative
	SaleOrderID     *uint         `json:"sale_order_id,omitempty"`                  // Generated Quotation ID
	LostReason      string        `gorm:"size:255" json:"lost_reason,omitempty"`
	Description     string        `gorm:"type:text" json:"description"`
}
