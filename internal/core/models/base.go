package models

import (
	"time"

	"gorm.io/gorm"
)

// ModelBase mimics standard enterprise BaseModel fields (id, create_date, write_date, create_uid, write_uid, active, company_id)
type ModelBase struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	Active    bool           `gorm:"default:true;index" json:"active"`
	CompanyID uint           `gorm:"index;default:1" json:"company_id"`
	CreateUID uint           `json:"create_uid,omitempty"`
	WriteUID  uint           `json:"write_uid,omitempty"`
}
