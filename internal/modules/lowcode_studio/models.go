package lowcode_studio

import (
	"time"
)

// Tipe Data Field Dinamis yang didukung Low-Code Studio
const (
	FieldTypeString    = "string"    // Teks singkat (Char)
	FieldTypeText      = "text"      // Teks panjang (Text area)
	FieldTypeInteger   = "integer"   // Angka bulat
	FieldTypeFloat     = "float"     // Angka desimal
	FieldTypeMonetary  = "monetary"  // Nilai mata uang
	FieldTypeDate      = "date"      // Tanggal (YYYY-MM-DD)
	FieldTypeDateTime  = "datetime"  // Tanggal & Waktu
	FieldTypeBoolean   = "boolean"   // Checkbox (true/false)
	FieldTypeSelection = "selection" // Dropdown opsi pilihan
	FieldTypeMany2One  = "many2one"  // Relasi ke model lain (res.partner, product.product, dll)
	FieldTypeJSON      = "json"      // Dokumen / Objek terstruktur
)

// CustomModel mendefinisikan entitas bisnis dinamis (ir.model)
type CustomModel struct {
	ID               uint          `gorm:"primaryKey" json:"id"`
	CompanyID        uint          `gorm:"index;default:1" json:"company_id"`
	TechnicalName    string        `gorm:"size:128;uniqueIndex;not null" json:"technical_name"` // misal: x_medical_record, x_hotel_booking, x_mining_concession
	DisplayName      string        `gorm:"size:255;not null" json:"display_name"`               // misal: Rekam Medis Pasien, Reservasi Hotel
	Description      string        `gorm:"type:text" json:"description"`
	Icon             string        `gorm:"size:64;default:'fas fa-cube'" json:"icon"`          // FontAwesome icon class
	MenuCategory     string        `gorm:"size:64;default:'Custom Apps'" json:"menu_category"` // Kategori menu sidebar
	IsCoreExtension  bool          `gorm:"default:false" json:"is_core_extension"`             // True jika memperluas tabel standar (res.partner, sale.order, dll)
	CoreModelTarget  string        `gorm:"size:128" json:"core_model_target"`                  // misal: res.partner, sale.order, stock.picking
	AllowChatter     bool          `gorm:"default:true" json:"allow_chatter"`
	AllowAttachments bool          `gorm:"default:true" json:"allow_attachments"`
	IsActive         bool          `gorm:"default:true" json:"is_active"`
	Fields           []CustomField `gorm:"foreignKey:CustomModelID;constraint:OnDelete:CASCADE" json:"fields,omitempty"`
	CreatedAt        time.Time     `json:"created_at"`
	UpdatedAt        time.Time     `json:"updated_at"`
}

// CustomField mendefinisikan kolom atribut dinamis pada model (ir.model.fields)
type CustomField struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	CustomModelID    uint      `gorm:"index;not null" json:"custom_model_id"`
	FieldName        string    `gorm:"size:128;not null" json:"field_name"`  // misal: x_blood_type, x_doctor_license, x_room_capacity
	FieldLabel       string    `gorm:"size:255;not null" json:"field_label"` // Label UI, misal: Golongan Darah, No. SIP Dokter
	FieldType        string    `gorm:"size:32;not null" json:"field_type"`   // string, integer, float, monetary, date, boolean, selection, many2one
	IsRequired       bool      `gorm:"default:false" json:"is_required"`
	IsReadonly       bool      `gorm:"default:false" json:"is_readonly"`
	IsSearchable     bool      `gorm:"default:true" json:"is_searchable"`
	DefaultValue     string    `gorm:"size:255" json:"default_value"`
	SelectionOptions string    `gorm:"type:text" json:"selection_options"` // JSON array opsi, contoh: ["A", "B", "AB", "O"]
	RelationModel    string    `gorm:"size:128" json:"relation_model"`     // Untuk many2one: res.partner, product.product, atau custom model lain
	SequenceOrder    int       `gorm:"default:10" json:"sequence_order"`   // Urutan tampilan pada form/grid
	HelpTooltip      string    `gorm:"size:255" json:"help_tooltip"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// CustomRecord menyimpan baris data fisik untuk model dinamis
type CustomRecord struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	CompanyID     uint      `gorm:"index;default:1" json:"company_id"`
	CustomModelID uint      `gorm:"index;not null" json:"custom_model_id"`
	RecordCode    string    `gorm:"size:64;index;not null" json:"record_code"` // misal: REC/2026/00001
	DisplayName   string    `gorm:"size:255;not null" json:"display_name"`
	ValuesJSON    string    `gorm:"type:text;not null" json:"values_json"` // JSON map keyValue dari seluruh dynamic fields
	CreatedByID   uint      `gorm:"default:1" json:"created_by_id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// CoreModelExtensionValue menyimpan nilai field kustom yang disuntikkan ke entitas ERP standar (res.partner, sale.order, dll)
type CoreModelExtensionValue struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	CompanyID     uint      `gorm:"index;default:1" json:"company_id"`
	CoreModel     string    `gorm:"size:128;index;not null" json:"core_model"` // res.partner, sale.order, stock.picking, account.move
	RecordID      uint      `gorm:"index;not null" json:"record_id"`           // ID dari baris tabel standar
	ExtensionJSON string    `gorm:"type:text;not null" json:"extension_json"`  // JSON map dari field kustom
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// DTO Requests & Responses
type CreateCustomModelRequest struct {
	TechnicalName    string                     `json:"technical_name" binding:"required"`
	DisplayName      string                     `json:"display_name"`
	Name             string                     `json:"name"`
	Description      string                     `json:"description"`
	Icon             string                     `json:"icon"`
	MenuCategory     string                     `json:"menu_category"`
	Category         string                     `json:"category"`
	IsCoreExtension  bool                       `json:"is_core_extension"`
	CoreModelTarget  string                     `json:"core_model_target"`
	AllowChatter     bool                       `json:"allow_chatter"`
	AllowAttachments bool                       `json:"allow_attachments"`
	InitialFields    []CreateCustomFieldRequest `json:"initial_fields"`
}

type CreateCustomFieldRequest struct {
	FieldName        string `json:"field_name" binding:"required"`
	FieldLabel       string `json:"field_label"`
	Label            string `json:"label"`
	FieldType        string `json:"field_type" binding:"required"`
	IsRequired       bool   `json:"is_required"`
	Required         bool   `json:"required"`
	IsReadonly       bool   `json:"is_readonly"`
	IsSearchable     bool   `json:"is_searchable"`
	DefaultValue     string `json:"default_value"`
	SelectionOptions string `json:"selection_options"`
	OptionsJSON      string `json:"options_json"`
	RelationModel    string `json:"relation_model"`
	SequenceOrder    int    `json:"sequence_order"`
	HelpTooltip      string `json:"help_tooltip"`
}

type SaveCustomRecordRequest struct {
	DisplayName string                 `json:"display_name"`
	Values      map[string]interface{} `json:"values" binding:"required"`
}

type SaveCoreExtensionRequest struct {
	CoreModel string                 `json:"core_model" binding:"required"`
	RecordID  uint                   `json:"record_id" binding:"required"`
	Values    map[string]interface{} `json:"values" binding:"required"`
}

type LowCodeStudioStats struct {
	TotalCustomModels     int64   `json:"total_custom_models"`
	TotalCustomFields     int64   `json:"total_custom_fields"`
	TotalCustomRecords    int64   `json:"total_custom_records"`
	CoreExtendedModels    int64   `json:"core_extended_models"`
	ActiveBusinessSchemas int64   `json:"active_business_schemas"`
	SystemReadinessRate   float64 `json:"system_readiness_rate"`
}

type AISchemaGenerationRequest struct {
	IndustryPrompt string `json:"industry_prompt"` // misal: "Sistem Rekam Medis Rumah Sakit dengan Poli, Diagnosa ICD-10 & Resep Obat"
	Prompt         string `json:"prompt"`
	TargetCategory string `json:"target_category"` // Healthcare, Mining, Aviation, Agriculture, Construction, dll
}

type AISchemaGenerationResponse struct {
	ModelTechnicalName string                     `json:"model_technical_name"`
	ModelDisplayName   string                     `json:"model_display_name"`
	Description        string                     `json:"description"`
	Icon               string                     `json:"icon"`
	MenuCategory       string                     `json:"menu_category"`
	RecommendedFields  []CreateCustomFieldRequest `json:"recommended_fields"`
	ArchitectureNotes  string                     `json:"architecture_notes"`
	Provider           string                     `json:"provider"`
}
