package maintenance

import (
	"erp-golang/internal/core/models"
	"erp-golang/internal/modules/asset"
	"erp-golang/internal/modules/base"
	"erp-golang/internal/modules/hr"
)

// MaintenanceEquipmentCategory corresponds to standard enterprise maintenance.equipment.category
type MaintenanceEquipmentCategory struct {
	models.ModelBase
	Name             string     `gorm:"size:128;not null;uniqueIndex" json:"name"`
	TechnicianUserID *uint      `json:"technician_user_id,omitempty"`
	Technician       *base.User `gorm:"foreignKey:TechnicianUserID" json:"technician,omitempty"`
	Color            string     `gorm:"size:16;default:'#3b82f6'" json:"color"`
	Note             string     `gorm:"type:text" json:"note,omitempty"`
}

// MaintenanceEquipment corresponds to standard enterprise maintenance.equipment
type MaintenanceEquipment struct {
	models.ModelBase
	Name                  string                        `gorm:"size:255;not null" json:"name"`
	EquipmentNumber       string                        `gorm:"size:64;not null;uniqueIndex" json:"equipment_number"` // e.g. EQP/2026/00001
	SerialNumber          string                        `gorm:"size:128" json:"serial_number,omitempty"`
	Model                 string                        `gorm:"size:128" json:"model,omitempty"`
	CategoryID            *uint                         `gorm:"index" json:"category_id,omitempty"`
	Category              *MaintenanceEquipmentCategory `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	PartnerID             *uint                         `gorm:"index" json:"partner_id,omitempty"` // Vendor / Supplier
	Partner               *base.Partner                 `gorm:"foreignKey:PartnerID" json:"partner,omitempty"`
	AssetID               *uint                         `gorm:"index" json:"asset_id,omitempty"` // Fixed Asset link
	Asset                 *asset.Asset                  `gorm:"foreignKey:AssetID" json:"asset,omitempty"`
	DepartmentID          *uint                         `gorm:"index" json:"department_id,omitempty"`
	Department            *hr.Department                `gorm:"foreignKey:DepartmentID" json:"department,omitempty"`
	EmployeeID            *uint                         `gorm:"index" json:"employee_id,omitempty"`
	Employee              *hr.Employee                  `gorm:"foreignKey:EmployeeID" json:"employee,omitempty"`
	TechnicianUserID      *uint                         `gorm:"index" json:"technician_user_id,omitempty"`
	Technician            *base.User                    `gorm:"foreignKey:TechnicianUserID" json:"technician,omitempty"`
	Location              string                        `gorm:"size:255" json:"location,omitempty"`
	Cost                  float64                       `gorm:"default:0" json:"cost"`
	WarrantyDate          string                        `gorm:"size:32" json:"warranty_date,omitempty"`
	MaintenancePeriodDays int                           `gorm:"default:30" json:"maintenance_period_days"`
	NextActionDate        string                        `gorm:"size:32" json:"next_action_date,omitempty"`
	Status                string                        `gorm:"size:32;default:'operational';index" json:"status"` // "operational", "maintenance", "scrapped"
	Note                  string                        `gorm:"type:text" json:"note,omitempty"`
}

// MaintenanceRequest corresponds to standard enterprise maintenance.request
type MaintenanceRequest struct {
	models.ModelBase
	RequestNumber   string                `gorm:"size:64;not null;uniqueIndex" json:"request_number"` // e.g. REQ/MAINT/2026/00001
	Name            string                `gorm:"size:255;not null" json:"name"`                     // Subject / problem summary
	EquipmentID     uint                  `gorm:"not null;index" json:"equipment_id"`
	Equipment       *MaintenanceEquipment `gorm:"foreignKey:EquipmentID" json:"equipment,omitempty"`
	MaintenanceType string                `gorm:"size:32;default:'corrective';index" json:"maintenance_type"` // "preventive", "corrective"
	Priority        string                `gorm:"size:8;default:'1';index" json:"priority"`                   // "0" Low, "1" Medium, "2" High, "3" Urgent
	Stage           string                `gorm:"size:32;default:'new';index" json:"stage"`                   // "new", "in_progress", "repaired", "scrap"
	RequestDate     string                `gorm:"size:32" json:"request_date"`
	ScheduleDate    string                `gorm:"size:32" json:"schedule_date,omitempty"`
	CloseDate       string                `gorm:"size:32" json:"close_date,omitempty"`
	DurationHours   float64               `gorm:"default:0" json:"duration_hours"`
	UserID          *uint                 `gorm:"index" json:"user_id,omitempty"` // Technician in charge
	User            *base.User            `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Description     string                `gorm:"type:text" json:"description"`
	Resolution      string                `gorm:"type:text" json:"resolution,omitempty"`
}
