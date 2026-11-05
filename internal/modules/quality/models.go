package quality

import (
	"erp-golang/internal/core/models"
	"erp-golang/internal/modules/base"
	"erp-golang/internal/modules/mrp"
	"erp-golang/internal/modules/product"
	"erp-golang/internal/modules/stock"
)

// QualityPoint corresponds to standard enterprise quality.point (Inspection Rules & Quality Triggers)
type QualityPoint struct {
	models.ModelBase
	Title             string           `gorm:"size:128;not null;index" json:"title"`     // e.g. "Inspeksi Visual Segel & Fisik Barang Masuk"
	Code              string           `gorm:"size:32;not null;uniqueIndex" json:"code"` // e.g. "QP/WH-IN/01", "QP/MO/01"
	ProductID         *uint            `gorm:"index" json:"product_id,omitempty"`        // Null = applies to all products
	Product           *product.Product `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	ProductCategoryID *uint            `gorm:"index" json:"product_category_id,omitempty"`
	OperationType     string           `gorm:"size:32;default:'incoming';index" json:"operation_type"` // "incoming" (WH/IN), "outgoing" (WH/OUT), "mrp_operation" (MO)
	TestType          string           `gorm:"size:32;default:'passfail'" json:"test_type"`            // "passfail", "measure", "instructions"
	Norm              float64          `gorm:"default:0" json:"norm"`                                  // Target value (e.g. 12.0)
	ToleranceMin      float64          `gorm:"default:0" json:"tolerance_min"`                         // Min acceptable value (e.g. 11.5)
	ToleranceMax      float64          `gorm:"default:0" json:"tolerance_max"`                         // Max acceptable value (e.g. 12.5)
	NormUnit          string           `gorm:"size:32" json:"norm_unit,omitempty"`                     // e.g. "V", "mm", "gram", "°C", "Ohm"
	Instructions      string           `gorm:"type:text" json:"instructions,omitempty"`                // SOP guidance for operator
	Active            bool             `json:"active"`
}

// QualityCheck corresponds to standard enterprise quality.check (Executed Quality Inspection Item)
type QualityCheck struct {
	models.ModelBase
	Name          string              `gorm:"size:64;not null;uniqueIndex" json:"name"` // e.g. "QC/2026/00001"
	PointID       *uint               `gorm:"index" json:"point_id,omitempty"`
	Point         *QualityPoint       `gorm:"foreignKey:PointID" json:"point,omitempty"`
	Title         string              `gorm:"size:128;not null" json:"title"`
	ProductID     uint                `gorm:"not null;index" json:"product_id"`
	Product       *product.Product    `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	PickingID     *uint               `gorm:"index" json:"picking_id,omitempty"` // Stock Receipt WH/IN
	Picking       *stock.StockPicking `gorm:"foreignKey:PickingID" json:"picking,omitempty"`
	ProductionID  *uint               `gorm:"index" json:"production_id,omitempty"` // MRP MO
	Production    *mrp.MrpProduction  `gorm:"foreignKey:ProductionID" json:"production,omitempty"`
	LotNumber     string              `gorm:"size:64" json:"lot_number,omitempty"`
	TestType      string              `gorm:"size:32;default:'passfail'" json:"test_type"` // "passfail", "measure", "instructions"
	Norm          float64             `gorm:"default:0" json:"norm"`
	ToleranceMin  float64             `gorm:"default:0" json:"tolerance_min"`
	ToleranceMax  float64             `gorm:"default:0" json:"tolerance_max"`
	NormUnit      string              `gorm:"size:32" json:"norm_unit,omitempty"`
	Measure       float64             `gorm:"default:0" json:"measure"`                          // Actual quantitative measurement
	QualityState  string              `gorm:"size:32;default:'none';index" json:"quality_state"` // "none" (pending), "pass", "fail"
	InspectorID   *uint               `gorm:"index" json:"inspector_id,omitempty"`
	Inspector     *base.User          `gorm:"foreignKey:InspectorID" json:"inspector,omitempty"`
	CheckDate     string              `gorm:"size:32" json:"check_date,omitempty"`
	Notes         string              `gorm:"type:text" json:"notes,omitempty"`
	FailureReason string              `gorm:"type:text" json:"failure_reason,omitempty"`
}

// QualityAlert corresponds to standard enterprise quality.alert (Non-Conformance / Root Cause Investigation)
type QualityAlert struct {
	models.ModelBase
	Name             string              `gorm:"size:64;not null;uniqueIndex" json:"name"` // e.g. "QA/2026/00001"
	Title            string              `gorm:"size:255;not null" json:"title"`           // e.g. "Patah Pin Konektor & Fluktuasi Voltase pada WH/IN/00001"
	CheckID          *uint               `gorm:"index" json:"check_id,omitempty"`          // Linked failing QualityCheck
	Check            *QualityCheck       `gorm:"foreignKey:CheckID" json:"check,omitempty"`
	ProductID        *uint               `gorm:"index" json:"product_id,omitempty"`
	Product          *product.Product    `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	PickingID        *uint               `gorm:"index" json:"picking_id,omitempty"`
	Picking          *stock.StockPicking `gorm:"foreignKey:PickingID" json:"picking,omitempty"`
	ProductionID     *uint               `gorm:"index" json:"production_id,omitempty"`
	Production       *mrp.MrpProduction  `gorm:"foreignKey:ProductionID" json:"production,omitempty"`
	PartnerID        *uint               `gorm:"index" json:"partner_id,omitempty"` // Vendor / Customer
	Partner          *base.Partner       `gorm:"foreignKey:PartnerID" json:"partner,omitempty"`
	Priority         string              `gorm:"size:16;default:'1'" json:"priority"`             // "0" (Low), "1" (Medium), "2" (High), "3" (Critical)
	Stage            string              `gorm:"size:32;default:'new';index" json:"stage"`        // "new", "confirmed", "in_progress", "resolved", "closed"
	ActionType       string              `gorm:"size:32;default:'quarantine'" json:"action_type"` // "quarantine", "scrap", "return", "rework"
	RootCause        string              `gorm:"type:text" json:"root_cause,omitempty"`           // Root cause analysis (RCA)
	PreventiveAction string              `gorm:"type:text" json:"preventive_action,omitempty"`    // CAPA preventive action
	UserID           *uint               `gorm:"index" json:"user_id,omitempty"`                  // QA Specialist / Assigned Officer
	User             *base.User          `gorm:"foreignKey:UserID" json:"user,omitempty"`
	CloseDate        string              `gorm:"size:32" json:"close_date,omitempty"`
	Notes            string              `gorm:"type:text" json:"notes,omitempty"`
}
