package audit_trail

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetDB() *gorm.DB {
	return r.db
}

// AuditLogFilters parameter struct for flexible querying
type AuditLogFilters struct {
	ModelName    string
	ResID        uint
	Action       string
	UserID       uint
	IsSuspicious *bool
	StartDate    string
	EndDate      string
	Search       string
	CompanyID    uint
	Limit        int
	Offset       int
}

// ---------------------------------------------------------------------------
// Rule Methods
// ---------------------------------------------------------------------------

func (r *Repository) CreateRule(rule *AuditLogRule) error {
	return r.db.Create(rule).Error
}

func (r *Repository) GetRuleByID(id uint, companyID uint) (*AuditLogRule, error) {
	var rule AuditLogRule
	query := r.db.Where("id = ?", id)
	if companyID > 0 {
		query = query.Where("company_id = ?", companyID)
	}
	if err := query.First(&rule).Error; err != nil {
		return nil, err
	}
	return &rule, nil
}

func (r *Repository) GetRuleByModel(modelName string, companyID uint) (*AuditLogRule, error) {
	var rule AuditLogRule
	query := r.db.Where("model_name = ? AND active = ?", modelName, true)
	if companyID > 0 {
		query = query.Where("company_id = ?", companyID)
	}
	if err := query.First(&rule).Error; err != nil {
		return nil, err
	}
	return &rule, nil
}

func (r *Repository) ListRules(companyID uint) ([]AuditLogRule, error) {
	var rules []AuditLogRule
	query := r.db.Model(&AuditLogRule{})
	if companyID > 0 {
		query = query.Where("company_id = ?", companyID)
	}
	err := query.Order("id asc").Find(&rules).Error
	return rules, err
}

func (r *Repository) UpdateRule(rule *AuditLogRule) error {
	return r.db.Save(rule).Error
}

func (r *Repository) DeleteRule(id uint, companyID uint) error {
	query := r.db.Where("id = ?", id)
	if companyID > 0 {
		query = query.Where("company_id = ?", companyID)
	}
	return query.Delete(&AuditLogRule{}).Error
}

// ---------------------------------------------------------------------------
// Entry & CDC Methods
// ---------------------------------------------------------------------------

func (r *Repository) GenerateLogCode(companyID uint) (string, error) {
	var count int64
	year := time.Now().Year()
	prefix := fmt.Sprintf("AUD/%d/", year)
	if err := r.db.Model(&AuditLogEntry{}).Where("log_code LIKE ?", prefix+"%").Count(&count).Error; err != nil {
		return "", err
	}
	return fmt.Sprintf("%s%05d", prefix, count+1), nil
}

func (r *Repository) CreateEntry(entry *AuditLogEntry) error {
	return r.db.Create(entry).Error
}

func (r *Repository) GetEntryByID(id uint, companyID uint) (*AuditLogEntry, error) {
	var entry AuditLogEntry
	query := r.db.Preload("Diffs").Where("id = ?", id)
	if companyID > 0 {
		query = query.Where("company_id = ?", companyID)
	}
	if err := query.First(&entry).Error; err != nil {
		return nil, err
	}
	return &entry, nil
}

func (r *Repository) ListEntries(filters AuditLogFilters) ([]AuditLogEntry, int64, error) {
	var entries []AuditLogEntry
	var total int64

	query := r.db.Model(&AuditLogEntry{})
	if filters.CompanyID > 0 {
		query = query.Where("company_id = ?", filters.CompanyID)
	}
	if filters.ModelName != "" {
		query = query.Where("model_name = ?", filters.ModelName)
	}
	if filters.ResID > 0 {
		query = query.Where("res_id = ?", filters.ResID)
	}
	if filters.Action != "" {
		query = query.Where("action = ?", filters.Action)
	}
	if filters.UserID > 0 {
		query = query.Where("user_id = ?", filters.UserID)
	}
	if filters.IsSuspicious != nil {
		query = query.Where("is_suspicious = ?", *filters.IsSuspicious)
	}
	if filters.StartDate != "" {
		query = query.Where("timestamp >= ?", filters.StartDate)
	}
	if filters.EndDate != "" {
		query = query.Where("timestamp <= ?", filters.EndDate)
	}
	if filters.Search != "" {
		s := "%" + strings.ToLower(filters.Search) + "%"
		query = query.Where("LOWER(log_code) LIKE ? OR LOWER(model_name) LIKE ? OR LOWER(res_name) LIKE ? OR LOWER(user_name) LIKE ? OR LOWER(summary) LIKE ?", s, s, s, s, s)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	limit := filters.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	offset := filters.Offset
	if offset < 0 {
		offset = 0
	}

	err := query.Preload("Diffs").Order("id desc").Limit(limit).Offset(offset).Find(&entries).Error
	return entries, total, err
}

func (r *Repository) ListRecentEntries(companyID uint, limit int) ([]AuditLogEntry, error) {
	var entries []AuditLogEntry
	query := r.db.Model(&AuditLogEntry{})
	if companyID > 0 {
		query = query.Where("company_id = ?", companyID)
	}
	if limit <= 0 {
		limit = 50
	}
	err := query.Preload("Diffs").Order("id desc").Limit(limit).Find(&entries).Error
	return entries, err
}

func (r *Repository) GetStats(companyID uint) (*AuditStats, error) {
	stats := &AuditStats{
		TopModels: make(map[string]int64),
		TopUsers:  make(map[string]int64),
	}

	baseQuery := r.db.Model(&AuditLogEntry{})
	if companyID > 0 {
		baseQuery = baseQuery.Where("company_id = ?", companyID)
	}

	if err := baseQuery.Count(&stats.TotalLogs).Error; err != nil {
		return nil, err
	}

	// Action counts
	qCreate := r.db.Model(&AuditLogEntry{}).Where("action = ?", ActionCreate)
	qUpdate := r.db.Model(&AuditLogEntry{}).Where("action = ?", ActionUpdate)
	qDelete := r.db.Model(&AuditLogEntry{}).Where("action = ?", ActionDelete)
	qRollback := r.db.Model(&AuditLogEntry{}).Where("action = ?", ActionRollback)
	qSuspicious := r.db.Model(&AuditLogEntry{}).Where("is_suspicious = ?", true)

	if companyID > 0 {
		qCreate = qCreate.Where("company_id = ?", companyID)
		qUpdate = qUpdate.Where("company_id = ?", companyID)
		qDelete = qDelete.Where("company_id = ?", companyID)
		qRollback = qRollback.Where("company_id = ?", companyID)
		qSuspicious = qSuspicious.Where("company_id = ?", companyID)
	}

	_ = qCreate.Count(&stats.CreateCount).Error
	_ = qUpdate.Count(&stats.UpdateCount).Error
	_ = qDelete.Count(&stats.DeleteCount).Error
	_ = qRollback.Count(&stats.RollbackCount).Error
	_ = qSuspicious.Count(&stats.SuspiciousCount).Error

	// Tracked models count
	rRule := r.db.Model(&AuditLogRule{}).Where("active = ?", true)
	if companyID > 0 {
		rRule = rRule.Where("company_id = ?", companyID)
	}
	_ = rRule.Count(&stats.TrackedModelsCount).Error

	// Top models
	type ModelCount struct {
		ModelName string
		Count     int64
	}
	var modelCounts []ModelCount
	mQuery := r.db.Model(&AuditLogEntry{}).Select("model_name, count(*) as count")
	if companyID > 0 {
		mQuery = mQuery.Where("company_id = ?", companyID)
	}
	if err := mQuery.Group("model_name").Order("count desc").Limit(5).Scan(&modelCounts).Error; err == nil {
		for _, mc := range modelCounts {
			stats.TopModels[mc.ModelName] = mc.Count
		}
	}

	// Top users
	type UserCount struct {
		UserName string
		Count    int64
	}
	var userCounts []UserCount
	uQuery := r.db.Model(&AuditLogEntry{}).Select("user_name, count(*) as count").Where("user_name != ''")
	if companyID > 0 {
		uQuery = uQuery.Where("company_id = ?", companyID)
	}
	if err := uQuery.Group("user_name").Order("count desc").Limit(5).Scan(&userCounts).Error; err == nil {
		for _, uc := range userCounts {
			stats.TopUsers[uc.UserName] = uc.Count
		}
	}

	return stats, nil
}
