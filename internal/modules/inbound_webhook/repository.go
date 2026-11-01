package inbound_webhook

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// ---------------------------------------------------------------------------
// Sequence Number Generators
// ---------------------------------------------------------------------------

func (r *Repository) GenerateEndpointCode(companyID uint) (string, error) {
	year := time.Now().Year()
	prefix := fmt.Sprintf("INWH/%d/", year)

	var count int64
	err := r.db.Model(&InboundWebhookEndpoint{}).
		Where("company_id = ? AND endpoint_code LIKE ?", companyID, prefix+"%").
		Count(&count).Error
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s%05d", prefix, count+1), nil
}

func (r *Repository) GenerateLogCode(companyID uint) (string, error) {
	year := time.Now().Year()
	prefix := fmt.Sprintf("INLOG/%d/", year)

	var count int64
	err := r.db.Model(&InboundWebhookLog{}).
		Where("company_id = ? AND log_code LIKE ?", companyID, prefix+"%").
		Count(&count).Error
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s%05d", prefix, count+1), nil
}

// ---------------------------------------------------------------------------
// Inbound Endpoints CRUD
// ---------------------------------------------------------------------------

func (r *Repository) CreateEndpoint(endpoint *InboundWebhookEndpoint) error {
	return r.db.Create(endpoint).Error
}

func (r *Repository) GetEndpointByID(id uint, companyID uint) (*InboundWebhookEndpoint, error) {
	var endpoint InboundWebhookEndpoint
	err := r.db.Where("id = ? AND company_id = ?", id, companyID).First(&endpoint).Error
	if err != nil {
		return nil, err
	}
	return &endpoint, nil
}

func (r *Repository) GetEndpointByKey(key string) (*InboundWebhookEndpoint, error) {
	var endpoint InboundWebhookEndpoint
	err := r.db.Where("endpoint_key = ? AND is_active = ?", key, true).First(&endpoint).Error
	if err != nil {
		return nil, err
	}
	return &endpoint, nil
}

func (r *Repository) ListEndpoints(companyID uint) ([]InboundWebhookEndpoint, error) {
	var endpoints []InboundWebhookEndpoint
	err := r.db.Where("company_id = ?", companyID).Order("id DESC").Find(&endpoints).Error
	return endpoints, err
}

func (r *Repository) UpdateEndpoint(endpoint *InboundWebhookEndpoint) error {
	return r.db.Save(endpoint).Error
}

func (r *Repository) DeleteEndpoint(id uint, companyID uint) error {
	return r.db.Where("id = ? AND company_id = ?", id, companyID).Delete(&InboundWebhookEndpoint{}).Error
}

func (r *Repository) IncrementEndpointCall(id uint, status string) error {
	now := time.Now()
	return r.db.Model(&InboundWebhookEndpoint{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"total_calls":    gorm.Expr("total_calls + 1"),
			"last_called_at": &now,
			"last_status":    status,
		}).Error
}

// ---------------------------------------------------------------------------
// Inbound Logs
// ---------------------------------------------------------------------------

func (r *Repository) CreateLog(log *InboundWebhookLog) error {
	return r.db.Create(log).Error
}

func (r *Repository) GetLogByID(id uint, companyID uint) (*InboundWebhookLog, error) {
	var l InboundWebhookLog
	err := r.db.Preload("Endpoint").Where("id = ? AND company_id = ?", id, companyID).First(&l).Error
	if err != nil {
		return nil, err
	}
	return &l, nil
}

func (r *Repository) ListLogs(filters InboundLogFilters) ([]InboundWebhookLog, int64, error) {
	var logs []InboundWebhookLog
	var total int64

	q := r.db.Model(&InboundWebhookLog{}).Where("company_id = ?", filters.CompanyID)

	if filters.EndpointID > 0 {
		q = q.Where("endpoint_id = ?", filters.EndpointID)
	}
	if filters.ProcessStatus != "" {
		q = q.Where("process_status = ?", filters.ProcessStatus)
	}
	if filters.EndpointKey != "" {
		q = q.Where("endpoint_key = ?", filters.EndpointKey)
	}
	if filters.Search != "" {
		search := "%" + filters.Search + "%"
		q = q.Where("log_code LIKE ? OR payload_json LIKE ? OR action_taken LIKE ?", search, search, search)
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	limit := filters.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	err := q.Preload("Endpoint").Order("id DESC").Limit(limit).Offset(filters.Offset).Find(&logs).Error
	return logs, total, err
}

// ---------------------------------------------------------------------------
// Statistics & Analytics
// ---------------------------------------------------------------------------

func (r *Repository) GetStats(companyID uint) (*InboundWebhookStats, error) {
	stats := &InboundWebhookStats{}

	// Endpoints count
	r.db.Model(&InboundWebhookEndpoint{}).Where("company_id = ?", companyID).Count(&stats.TotalEndpoints)
	r.db.Model(&InboundWebhookEndpoint{}).Where("company_id = ? AND is_active = ?", companyID, true).Count(&stats.ActiveEndpoints)

	// Logs count
	r.db.Model(&InboundWebhookLog{}).Where("company_id = ?", companyID).Count(&stats.TotalInboundLogs)
	r.db.Model(&InboundWebhookLog{}).Where("company_id = ? AND process_status = ?", companyID, StatusProcessed).Count(&stats.ProcessedLogs)
	r.db.Model(&InboundWebhookLog{}).Where("company_id = ? AND process_status = ?", companyID, StatusRejected).Count(&stats.RejectedLogs)
	r.db.Model(&InboundWebhookLog{}).Where("company_id = ? AND process_status = ?", companyID, StatusError).Count(&stats.ErrorLogs)

	// Valid Signature %
	var validSigCount int64
	r.db.Model(&InboundWebhookLog{}).Where("company_id = ? AND signature_valid = ?", companyID, true).Count(&validSigCount)
	if stats.TotalInboundLogs > 0 {
		stats.ValidSigPercent = float64(validSigCount) / float64(stats.TotalInboundLogs) * 100.0
	} else {
		stats.ValidSigPercent = 100.0
	}

	// Average duration ms
	r.db.Model(&InboundWebhookLog{}).Where("company_id = ?", companyID).
		Select("COALESCE(AVG(duration_ms), 0)").Scan(&stats.AvgDurationMs)

	return stats, nil
}
