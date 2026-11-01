package helpdesk

import (
	"erp-golang/internal/core/models"
	"erp-golang/internal/modules/base"
)

// HelpdeskTeam corresponds to Odoo's helpdesk.team
type HelpdeskTeam struct {
	models.ModelBase
	Name     string     `gorm:"size:128;not null" json:"name"`
	Code     string     `gorm:"size:16;not null;uniqueIndex" json:"code"` // e.g. CARE, TECH, BILL
	Email    string     `gorm:"size:128" json:"email,omitempty"`
	LeaderID *uint      `json:"leader_id,omitempty"`
	Leader   *base.User `gorm:"foreignKey:LeaderID" json:"leader,omitempty"`
	Color    string     `gorm:"size:16;default:'#3b82f6'" json:"color"`
	Active   bool       `gorm:"default:true" json:"active"`
}

// HelpdeskTicket corresponds to Odoo's helpdesk.ticket
type HelpdeskTicket struct {
	models.ModelBase
	TicketNumber string        `gorm:"size:64;not null;uniqueIndex" json:"ticket_number"` // e.g. TICKET/2026/00001
	Name         string        `gorm:"size:255;not null" json:"name"`                    // Subject / title
	TeamID       *uint         `gorm:"index" json:"team_id,omitempty"`
	Team         *HelpdeskTeam `gorm:"foreignKey:TeamID" json:"team,omitempty"`
	PartnerID    *uint         `gorm:"index" json:"partner_id,omitempty"`
	Partner      *base.Partner `gorm:"foreignKey:PartnerID" json:"partner,omitempty"`
	PartnerName  string        `gorm:"size:255" json:"partner_name,omitempty"`
	PartnerEmail string        `gorm:"size:128" json:"partner_email,omitempty"`
	PartnerPhone string        `gorm:"size:64" json:"partner_phone,omitempty"`
	UserID       *uint         `gorm:"index" json:"user_id,omitempty"` // Assigned agent
	User         *base.User    `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Priority     string        `gorm:"size:8;default:'1';index" json:"priority"`           // "0" Low, "1" Medium, "2" High, "3" Urgent
	Category     string        `gorm:"size:32;default:'technical';index" json:"category"`  // "technical", "billing", "inquiry", "warranty"
	Stage        string        `gorm:"size:32;default:'new';index" json:"stage"`           // "new", "in_progress", "waiting", "solved", "cancelled"
	Description  string        `gorm:"type:text" json:"description"`
	Resolution   string        `gorm:"type:text" json:"resolution,omitempty"`
	CloseDate    string        `gorm:"size:32" json:"close_date,omitempty"`
}
