package project

import (
	"time"

	"erp-golang/internal/core/models"
	"erp-golang/internal/modules/base"
)

// Project corresponds to Odoo's project.project
type Project struct {
	models.ModelBase
	Name        string        `gorm:"size:255;not null" json:"name"`
	PartnerID   *uint         `json:"partner_id,omitempty"`
	Partner     *base.Partner `gorm:"foreignKey:PartnerID" json:"partner,omitempty"`
	UserID      *uint         `json:"user_id,omitempty"` // Project Manager
	User        *base.User    `gorm:"foreignKey:UserID" json:"user,omitempty"`
	DateStart   *time.Time    `json:"date_start,omitempty"`
	DateEnd     *time.Time    `json:"date_end,omitempty"`
	Status      string        `gorm:"size:32;default:'active'" json:"status"` // "active", "completed", "cancelled"
	Color       string        `gorm:"size:32;default:'#6366f1'" json:"color"`
	Description string        `gorm:"type:text" json:"description"`
	Tasks       []Task        `gorm:"foreignKey:ProjectID" json:"tasks,omitempty"`
}

// Task corresponds to Odoo's project.task
type Task struct {
	models.ModelBase
	ProjectID      uint            `gorm:"not null;index" json:"project_id"`
	Project        *Project        `gorm:"foreignKey:ProjectID" json:"project,omitempty"`
	Name           string          `gorm:"size:255;not null" json:"name"`
	PartnerID      *uint           `json:"partner_id,omitempty"`
	Partner        *base.Partner   `gorm:"foreignKey:PartnerID" json:"partner,omitempty"`
	UserID         *uint           `json:"user_id,omitempty"` // Assigned to
	User           *base.User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Stage          string          `gorm:"size:32;default:'todo';index" json:"stage"` // "todo", "in_progress", "review", "done", "cancelled"
	Priority       string          `gorm:"size:16;default:'1'" json:"priority"`      // "0" (Low), "1" (Normal), "2" (High), "3" (Urgent)
	PlannedHours   float64         `gorm:"default:0" json:"planned_hours"`
	EffectiveHours float64         `gorm:"default:0" json:"effective_hours"`
	Progress       float64         `gorm:"default:0" json:"progress"` // In %
	Deadline       *time.Time      `json:"deadline,omitempty"`
	Description    string          `gorm:"type:text" json:"description"`
	Timesheets     []TimesheetLine `gorm:"foreignKey:TaskID" json:"timesheets,omitempty"`
}

// TimesheetLine corresponds to Odoo's account.analytic.line for project timesheets
type TimesheetLine struct {
	models.ModelBase
	ProjectID  uint       `gorm:"not null;index" json:"project_id"`
	TaskID     uint       `gorm:"not null;index" json:"task_id"`
	UserID     uint       `gorm:"not null;index" json:"user_id"`
	User       *base.User `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Date       string     `gorm:"size:32;not null" json:"date"`
	UnitAmount float64    `gorm:"not null" json:"unit_amount"` // Hours spent e.g. 2.5
	Name       string     `gorm:"size:255;not null" json:"name"` // Description of work
}
