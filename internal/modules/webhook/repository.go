package webhook

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

// DeliveryFilters parameter struct for flexible querying
type DeliveryFilters struct {
	SubscriptionID uint
	EventName      string
	Status         string
	StartDate      string
	EndDate        string
	Search         string
	CompanyID      uint
	Limit          int
	Offset         int
}

// ---------------------------------------------------------------------------
// Subscription Repository Methods
// ---------------------------------------------------------------------------

func (r *Repository) CreateSubscription(sub *WebhookSubscription) error {
	return r.db.Create(sub).Error
}

func (r *Repository) GetSubscriptionByID(id uint, companyID uint) (*WebhookSubscription, error) {
	var sub WebhookSubscription
	query := r.db.Where("id = ?", id)
	if companyID > 0 {
		query = query.Where("company_id = ?", companyID)
	}
	if err := query.First(&sub).Error; err != nil {
		return nil, err
	}
	return &sub, nil
}

func (r *Repository) ListSubscriptions(companyID uint) ([]WebhookSubscription, error) {
	var subs []WebhookSubscription
	query := r.db.Model(&WebhookSubscription{})
	if companyID > 0 {
		query = query.Where("company_id = ?", companyID)
	}
	err := query.Order("id desc").Find(&subs).Error
	return subs, err
}

func (r *Repository) UpdateSubscription(sub *WebhookSubscription) error {
	return r.db.Save(sub).Error
}

func (r *Repository) DeleteSubscription(id uint, companyID uint) error {
	query := r.db.Where("id = ?", id)
	if companyID > 0 {
		query = query.Where("company_id = ?", companyID)
	}
	return query.Delete(&WebhookSubscription{}).Error
}

// GetActiveSubscriptionsForTopic returns subscriptions listening to the specified topic or wildcard "*"
func (r *Repository) GetActiveSubscriptionsForTopic(topic string, companyID uint) ([]WebhookSubscription, error) {
	var allActive []WebhookSubscription
	query := r.db.Where("is_active = ?", true)
	if companyID > 0 {
		query = query.Where("company_id = ?", companyID)
	}
	if err := query.Find(&allActive).Error; err != nil {
		return nil, err
	}

	var matching []WebhookSubscription
	for _, sub := range allActive {
		topics := strings.Split(sub.EventTopics, ",")
		for _, t := range topics {
			clean := strings.TrimSpace(t)
			if clean == EventWildcard || clean == topic {
				matching = append(matching, sub)
				break
			}
		}
	}
	return matching, nil
}

// ---------------------------------------------------------------------------
// Delivery Execution Repository Methods
// ---------------------------------------------------------------------------

func (r *Repository) GenerateDeliveryCode(companyID uint) (string, error) {
	var count int64
	year := time.Now().Year()
	prefix := fmt.Sprintf("DLV/%d/", year)
	if err := r.db.Model(&WebhookDelivery{}).Where("delivery_code LIKE ?", prefix+"%").Count(&count).Error; err != nil {
		return "", err
	}
	return fmt.Sprintf("%s%05d", prefix, count+1), nil
}

func (r *Repository) CreateDelivery(delivery *WebhookDelivery) error {
	return r.db.Create(delivery).Error
}

func (r *Repository) GetDeliveryByID(id uint, companyID uint) (*WebhookDelivery, error) {
	var dlv WebhookDelivery
	query := r.db.Preload("Subscription").Where("id = ?", id)
	if companyID > 0 {
		query = query.Where("company_id = ?", companyID)
	}
	if err := query.First(&dlv).Error; err != nil {
		return nil, err
	}
	return &dlv, nil
}

func (r *Repository) UpdateDelivery(delivery *WebhookDelivery) error {
	return r.db.Save(delivery).Error
}

func (r *Repository) ListDeliveries(filters DeliveryFilters) ([]WebhookDelivery, int64, error) {
	var deliveries []WebhookDelivery
	var total int64

	query := r.db.Model(&WebhookDelivery{})
	if filters.CompanyID > 0 {
		query = query.Where("company_id = ?", filters.CompanyID)
	}
	if filters.SubscriptionID > 0 {
		query = query.Where("subscription_id = ?", filters.SubscriptionID)
	}
	if filters.EventName != "" {
		query = query.Where("event_name = ?", filters.EventName)
	}
	if filters.Status != "" {
		query = query.Where("status = ?", filters.Status)
	}
	if filters.StartDate != "" {
		query = query.Where("created_at >= ?", filters.StartDate)
	}
	if filters.EndDate != "" {
		query = query.Where("created_at <= ?", filters.EndDate)
	}
	if filters.Search != "" {
		s := "%" + strings.ToLower(filters.Search) + "%"
		query = query.Where("LOWER(delivery_code) LIKE ? OR LOWER(event_name) LIKE ? OR LOWER(target_url) LIKE ? OR LOWER(response_body) LIKE ?", s, s, s, s)
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

	err := query.Preload("Subscription").Order("id desc").Limit(limit).Offset(offset).Find(&deliveries).Error
	return deliveries, total, err
}

func (r *Repository) GetPendingRetries(companyID uint, limit int) ([]WebhookDelivery, error) {
	var pending []WebhookDelivery
	now := time.Now()
	query := r.db.Preload("Subscription").
		Where("status = ? AND next_retry_at IS NOT NULL AND next_retry_at <= ?", StatusRetrying, now)

	if companyID > 0 {
		query = query.Where("company_id = ?", companyID)
	}
	if limit <= 0 {
		limit = 50
	}
	err := query.Order("next_retry_at asc").Limit(limit).Find(&pending).Error
	return pending, err
}

func (r *Repository) GetStats(companyID uint) (*WebhookStats, error) {
	stats := &WebhookStats{}

	// Subscription counts
	qSub := r.db.Model(&WebhookSubscription{})
	if companyID > 0 {
		qSub = qSub.Where("company_id = ?", companyID)
	}
	_ = qSub.Count(&stats.TotalSubscriptions).Error

	qSubActive := r.db.Model(&WebhookSubscription{}).Where("is_active = ?", true)
	if companyID > 0 {
		qSubActive = qSubActive.Where("company_id = ?", companyID)
	}
	_ = qSubActive.Count(&stats.ActiveSubscriptions).Error

	// Delivery counts
	qDlv := r.db.Model(&WebhookDelivery{})
	if companyID > 0 {
		qDlv = qDlv.Where("company_id = ?", companyID)
	}
	_ = qDlv.Count(&stats.TotalDeliveries).Error

	qSuccess := r.db.Model(&WebhookDelivery{}).Where("status = ?", StatusSuccess)
	qFailed := r.db.Model(&WebhookDelivery{}).Where("status = ?", StatusFailed)
	qRetrying := r.db.Model(&WebhookDelivery{}).Where("status = ?", StatusRetrying)

	if companyID > 0 {
		qSuccess = qSuccess.Where("company_id = ?", companyID)
		qFailed = qFailed.Where("company_id = ?", companyID)
		qRetrying = qRetrying.Where("company_id = ?", companyID)
	}

	_ = qSuccess.Count(&stats.SuccessDeliveries).Error
	_ = qFailed.Count(&stats.FailedDeliveries).Error
	_ = qRetrying.Count(&stats.RetryingDeliveries).Error

	if stats.TotalDeliveries > 0 {
		stats.SuccessRatePct = float64(stats.SuccessDeliveries*100) / float64(stats.TotalDeliveries)
	} else {
		stats.SuccessRatePct = 100.0
	}

	// Average latency
	var avgDur float64
	dQuery := r.db.Model(&WebhookDelivery{}).Select("COALESCE(AVG(duration_ms), 0)")
	if companyID > 0 {
		dQuery = dQuery.Where("company_id = ?", companyID)
	}
	_ = dQuery.Scan(&avgDur).Error
	stats.AvgDurationMs = avgDur

	return stats, nil
}
