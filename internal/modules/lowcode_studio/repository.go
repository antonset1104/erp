package lowcode_studio

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

type Repository interface {
	AutoMigrate() error

	// Custom Models
	CreateModel(model *CustomModel) error
	GetModelByID(id uint) (*CustomModel, error)
	GetModelByTechnicalName(name string) (*CustomModel, error)
	ListModels(companyID uint) ([]CustomModel, error)
	UpdateModel(model *CustomModel) error
	DeleteModel(id uint) error

	// Custom Fields
	CreateField(field *CustomField) error
	GetFieldByID(id uint) (*CustomField, error)
	ListFieldsByModelID(modelID uint) ([]CustomField, error)
	UpdateField(field *CustomField) error
	DeleteField(id uint) error

	// Custom Records
	GenerateRecordCode(prefix string) string
	CreateRecord(record *CustomRecord) error
	GetRecordByID(id uint) (*CustomRecord, error)
	ListRecordsByModelID(modelID uint, companyID uint, limit int, offset int) ([]CustomRecord, int64, error)
	UpdateRecord(record *CustomRecord) error
	DeleteRecord(id uint) error

	// Core Extension Values
	GetCoreExtensionValue(coreModel string, recordID uint) (*CoreModelExtensionValue, error)
	SaveCoreExtensionValue(val *CoreModelExtensionValue) error

	// Stats
	GetStats(companyID uint) (*LowCodeStudioStats, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) AutoMigrate() error {
	return r.db.AutoMigrate(
		&CustomModel{},
		&CustomField{},
		&CustomRecord{},
		&CoreModelExtensionValue{},
	)
}

// Custom Models
func (r *repository) CreateModel(model *CustomModel) error {
	return r.db.Create(model).Error
}

func (r *repository) GetModelByID(id uint) (*CustomModel, error) {
	var model CustomModel
	err := r.db.Preload("Fields", func(db *gorm.DB) *gorm.DB {
		return db.Order("sequence_order ASC")
	}).First(&model, id).Error
	if err != nil {
		return nil, err
	}
	return &model, nil
}

func (r *repository) GetModelByTechnicalName(name string) (*CustomModel, error) {
	var model CustomModel
	err := r.db.Preload("Fields", func(db *gorm.DB) *gorm.DB {
		return db.Order("sequence_order ASC")
	}).Where("technical_name = ?", name).First(&model).Error
	if err != nil {
		return nil, err
	}
	return &model, nil
}

func (r *repository) ListModels(companyID uint) ([]CustomModel, error) {
	var models []CustomModel
	q := r.db.Preload("Fields", func(db *gorm.DB) *gorm.DB {
		return db.Order("sequence_order ASC")
	})
	if companyID > 0 {
		q = q.Where("company_id = ? OR company_id = 0", companyID)
	}
	err := q.Order("id DESC").Find(&models).Error
	return models, err
}

func (r *repository) UpdateModel(model *CustomModel) error {
	return r.db.Save(model).Error
}

func (r *repository) DeleteModel(id uint) error {
	return r.db.Delete(&CustomModel{}, id).Error
}

// Custom Fields
func (r *repository) CreateField(field *CustomField) error {
	return r.db.Create(field).Error
}

func (r *repository) GetFieldByID(id uint) (*CustomField, error) {
	var field CustomField
	err := r.db.First(&field, id).Error
	if err != nil {
		return nil, err
	}
	return &field, nil
}

func (r *repository) ListFieldsByModelID(modelID uint) ([]CustomField, error) {
	var fields []CustomField
	err := r.db.Where("custom_model_id = ?", modelID).Order("sequence_order ASC").Find(&fields).Error
	return fields, err
}

func (r *repository) UpdateField(field *CustomField) error {
	return r.db.Save(field).Error
}

func (r *repository) DeleteField(id uint) error {
	return r.db.Delete(&CustomField{}, id).Error
}

// Custom Records
func (r *repository) GenerateRecordCode(prefix string) string {
	if prefix == "" {
		prefix = "REC"
	}
	now := time.Now()
	var count int64
	r.db.Model(&CustomRecord{}).Count(&count)
	return fmt.Sprintf("%s/%d/%05d", prefix, now.Year(), count+1)
}

func (r *repository) CreateRecord(record *CustomRecord) error {
	return r.db.Create(record).Error
}

func (r *repository) GetRecordByID(id uint) (*CustomRecord, error) {
	var record CustomRecord
	err := r.db.First(&record, id).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *repository) ListRecordsByModelID(modelID uint, companyID uint, limit int, offset int) ([]CustomRecord, int64, error) {
	var records []CustomRecord
	var total int64

	q := r.db.Model(&CustomRecord{}).Where("custom_model_id = ?", modelID)
	if companyID > 0 {
		q = q.Where("company_id = ? OR company_id = 0", companyID)
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if limit <= 0 {
		limit = 50
	}
	err := q.Order("id DESC").Limit(limit).Offset(offset).Find(&records).Error
	return records, total, err
}

func (r *repository) UpdateRecord(record *CustomRecord) error {
	return r.db.Save(record).Error
}

func (r *repository) DeleteRecord(id uint) error {
	return r.db.Delete(&CustomRecord{}, id).Error
}

// Core Extension Values
func (r *repository) GetCoreExtensionValue(coreModel string, recordID uint) (*CoreModelExtensionValue, error) {
	var val CoreModelExtensionValue
	err := r.db.Where("core_model = ? AND record_id = ?", coreModel, recordID).First(&val).Error
	if err != nil {
		return nil, err
	}
	return &val, nil
}

func (r *repository) SaveCoreExtensionValue(val *CoreModelExtensionValue) error {
	var existing CoreModelExtensionValue
	err := r.db.Where("core_model = ? AND record_id = ?", val.CoreModel, val.RecordID).First(&existing).Error
	if err == nil {
		existing.ExtensionJSON = val.ExtensionJSON
		existing.CompanyID = val.CompanyID
		return r.db.Save(&existing).Error
	}
	return r.db.Create(val).Error
}

// Stats
func (r *repository) GetStats(companyID uint) (*LowCodeStudioStats, error) {
	stats := LowCodeStudioStats{
		SystemReadinessRate: 98.5,
	}

	r.db.Model(&CustomModel{}).Count(&stats.TotalCustomModels)
	r.db.Model(&CustomField{}).Count(&stats.TotalCustomFields)
	r.db.Model(&CustomRecord{}).Count(&stats.TotalCustomRecords)
	r.db.Model(&CustomModel{}).Where("is_core_extension = ?", true).Count(&stats.CoreExtendedModels)
	r.db.Model(&CustomModel{}).Where("is_active = ?", true).Count(&stats.ActiveBusinessSchemas)

	return &stats, nil
}
