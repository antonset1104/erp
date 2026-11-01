package subscription

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
// Template Methods
// ---------------------------------------------------------------------------

func (r *Repository) CreateTemplate(template *SubscriptionTemplate) error {
	return r.db.Create(template).Error
}

func (r *Repository) GetTemplateByID(id uint, companyID uint) (*SubscriptionTemplate, error) {
	var template SubscriptionTemplate
	if err := r.db.Where("id = ? AND company_id = ?", id, companyID).First(&template).Error; err != nil {
		return nil, err
	}
	return &template, nil
}

func (r *Repository) GetTemplateByCode(code string, companyID uint) (*SubscriptionTemplate, error) {
	var template SubscriptionTemplate
	if err := r.db.Where("code = ? AND company_id = ?", code, companyID).First(&template).Error; err != nil {
		return nil, err
	}
	return &template, nil
}

func (r *Repository) ListTemplates(companyID uint) ([]SubscriptionTemplate, error) {
	var templates []SubscriptionTemplate
	err := r.db.Where("company_id = ?", companyID).Order("id asc").Find(&templates).Error
	return templates, err
}

// ---------------------------------------------------------------------------
// Subscription Methods
// ---------------------------------------------------------------------------

func (r *Repository) CreateSubscription(sub *Subscription) error {
	return r.db.Create(sub).Error
}

func (r *Repository) GetSubscriptionByID(id uint, companyID uint) (*Subscription, error) {
	var sub Subscription
	err := r.db.Preload("Partner").
		Preload("Template").
		Preload("Lines.Product").
		Preload("Invoices").
		Where("id = ? AND company_id = ?", id, companyID).
		First(&sub).Error
	if err != nil {
		return nil, err
	}
	return &sub, nil
}

func (r *Repository) GetSubscriptionByCode(code string, companyID uint) (*Subscription, error) {
	var sub Subscription
	err := r.db.Preload("Partner").
		Preload("Template").
		Preload("Lines.Product").
		Preload("Invoices").
		Where("code = ? AND company_id = ?", code, companyID).
		First(&sub).Error
	if err != nil {
		return nil, err
	}
	return &sub, nil
}

func (r *Repository) UpdateSubscription(sub *Subscription) error {
	return r.db.Save(sub).Error
}

func (r *Repository) ListSubscriptions(companyID uint, state string, partnerID uint, templateID uint, search string) ([]Subscription, error) {
	var subs []Subscription
	query := r.db.Preload("Partner").
		Preload("Template").
		Preload("Lines.Product").
		Where("company_id = ?", companyID)

	if state != "" {
		query = query.Where("state = ?", state)
	}
	if partnerID > 0 {
		query = query.Where("partner_id = ?", partnerID)
	}
	if templateID > 0 {
		query = query.Where("template_id = ?", templateID)
	}
	if search != "" {
		pattern := "%" + search + "%"
		query = query.Where("code LIKE ? OR name LIKE ? OR churn_reason LIKE ?", pattern, pattern, pattern)
	}

	err := query.Order("id desc").Find(&subs).Error
	return subs, err
}

func (r *Repository) GetDueSubscriptions(companyID uint, targetDate string) ([]Subscription, error) {
	var subs []Subscription
	err := r.db.Preload("Partner").
		Preload("Template").
		Preload("Lines.Product").
		Where("company_id = ? AND state = ? AND next_invoice_date <= ?", companyID, StateInProgress, targetDate).
		Order("next_invoice_date asc").
		Find(&subs).Error
	return subs, err
}

// ---------------------------------------------------------------------------
// Lines & Invoices & Logs
// ---------------------------------------------------------------------------

func (r *Repository) CreateInvoiceLog(log *SubscriptionInvoiceLog) error {
	return r.db.Create(log).Error
}

func (r *Repository) ListInvoiceLogs(companyID uint, subscriptionID uint) ([]SubscriptionInvoiceLog, error) {
	var logs []SubscriptionInvoiceLog
	query := r.db.Where("company_id = ?", companyID)
	if subscriptionID > 0 {
		query = query.Where("subscription_id = ?", subscriptionID)
	}
	err := query.Order("id desc").Find(&logs).Error
	return logs, err
}

func (r *Repository) CreateStageLog(log *SubscriptionStageLog) error {
	return r.db.Create(log).Error
}

func (r *Repository) ListStageLogs(subscriptionID uint) ([]SubscriptionStageLog, error) {
	var logs []SubscriptionStageLog
	err := r.db.Where("subscription_id = ?", subscriptionID).Order("id asc").Find(&logs).Error
	return logs, err
}

// ---------------------------------------------------------------------------
// Helpers & Sequences
// ---------------------------------------------------------------------------

func (r *Repository) GetNextSubscriptionCode(companyID uint) (string, error) {
	year := time.Now().Format("2006")
	var count int64
	r.db.Model(&Subscription{}).Where("company_id = ?", companyID).Count(&count)

	for {
		count++
		candidate := fmt.Sprintf("SUB/%s/%05d", year, count)
		var exists int64
		r.db.Model(&Subscription{}).Where("code = ?", candidate).Count(&exists)
		if exists == 0 {
			return candidate, nil
		}
	}
}

// GetSaaSMetrics calculates aggregate SaaS metrics
func (r *Repository) GetSaaSMetrics(companyID uint) (*SaaSMetrics, error) {
	metrics := &SaaSMetrics{
		SubscriptionsByInterval: make(map[string]int64),
		IntervalMRRBreakdown:    make(map[string]float64),
	}

	var allSubs []Subscription
	if err := r.db.Where("company_id = ?", companyID).Find(&allSubs).Error; err != nil {
		return nil, err
	}

	metrics.TotalSubscriptions = int64(len(allSubs))

	var activeMRR, churnedMRR float64

	for _, s := range allSubs {
		switch s.State {
		case StateInProgress:
			metrics.ActiveSubscriptions++
			activeMRR += s.TotalMRR
			metrics.SubscriptionsByInterval[s.RecurringInterval]++
			metrics.IntervalMRRBreakdown[s.RecurringInterval] += s.TotalMRR
		case StatePaused:
			metrics.PausedSubscriptions++
		case StateChurned:
			metrics.ChurnedSubscriptions++
			churnedMRR += s.TotalMRR
		case StateDraft:
			metrics.DraftSubscriptions++
		}
		metrics.TotalInvoicedLifetime += s.TotalInvoicedAmount
	}

	metrics.TotalLiveMRR = activeMRR
	metrics.TotalLiveARR = activeMRR * 12.0

	// Count total invoices
	var invCount int64
	r.db.Model(&SubscriptionInvoiceLog{}).Where("company_id = ?", companyID).Count(&invCount)
	metrics.TotalInvoicesCount = invCount

	// Logo Churn Rate = Churned / (Active + Churned) * 100
	totalActiveOrChurned := metrics.ActiveSubscriptions + metrics.ChurnedSubscriptions
	if totalActiveOrChurned > 0 {
		metrics.LogoChurnRatePct = (float64(metrics.ChurnedSubscriptions) / float64(totalActiveOrChurned)) * 100.0
	}

	// Revenue Churn Rate = Lost MRR / (Active MRR + Lost MRR) * 100
	totalPotentialMRR := activeMRR + churnedMRR
	if totalPotentialMRR > 0 {
		metrics.RevenueChurnRatePct = (churnedMRR / totalPotentialMRR) * 100.0
	}

	// ARPU = Total MRR / Active Subscriptions
	if metrics.ActiveSubscriptions > 0 {
		metrics.ARPU = activeMRR / float64(metrics.ActiveSubscriptions)
	}

	// Estimated LTV = ARPU / (LogoChurnRate / 100) or ARPU * 24 if churn rate is 0
	if metrics.LogoChurnRatePct > 0 {
		metrics.EstimatedLTV = metrics.ARPU / (metrics.LogoChurnRatePct / 100.0)
	} else if metrics.ActiveSubscriptions > 0 {
		metrics.EstimatedLTV = metrics.ARPU * 24.0 // 2-year baseline when zero churn
	}

	return metrics, nil
}
