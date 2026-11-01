package currency

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

func (r *Repository) ListCurrencies(activeOnly bool) ([]Currency, error) {
	var list []Currency
	query := r.db.Preload("Rates", func(db *gorm.DB) *gorm.DB {
		return db.Order("name desc").Limit(10)
	})
	if activeOnly {
		query = query.Where("active = ?", true)
	}
	err := query.Order("is_base desc, name asc").Find(&list).Error
	if err == nil {
		for i := range list {
			if len(list[i].Rates) > 0 {
				list[i].LatestSpotRate = list[i].Rates[0].DirectRate
				list[i].LatestRateDate = list[i].Rates[0].Name
			}
		}
	}
	return list, err
}

func (r *Repository) GetCurrencyByID(id uint) (*Currency, error) {
	var curr Currency
	err := r.db.Preload("Rates", func(db *gorm.DB) *gorm.DB {
		return db.Order("name desc")
	}).First(&curr, id).Error
	if err != nil {
		return nil, err
	}
	if len(curr.Rates) > 0 {
		curr.LatestSpotRate = curr.Rates[0].DirectRate
		curr.LatestRateDate = curr.Rates[0].Name
	}
	return &curr, nil
}

func (r *Repository) GetCurrencyByName(name string) (*Currency, error) {
	var curr Currency
	err := r.db.Preload("Rates", func(db *gorm.DB) *gorm.DB {
		return db.Order("name desc")
	}).Where("name = ?", name).First(&curr).Error
	if err != nil {
		return nil, err
	}
	return &curr, nil
}

func (r *Repository) GetBaseCurrency() (*Currency, error) {
	var curr Currency
	err := r.db.Where("is_base = ?", true).First(&curr).Error
	if err != nil {
		return nil, err
	}
	return &curr, nil
}

func (r *Repository) CreateCurrency(curr *Currency) error {
	return r.db.Create(curr).Error
}

func (r *Repository) UpdateCurrency(curr *Currency) error {
	return r.db.Save(curr).Error
}

func (r *Repository) CreateRate(rate *CurrencyRate) error {
	return r.db.Create(rate).Error
}

func (r *Repository) GetLatestRate(currencyID uint, date string, companyID uint) (*CurrencyRate, error) {
	var rate CurrencyRate
	query := r.db.Where("currency_id = ?", currencyID)
	if companyID > 0 {
		query = query.Where("company_id = ?", companyID)
	}
	if date != "" {
		query = query.Where("name <= ?", date)
	}
	err := query.Order("name desc, id desc").First(&rate).Error
	if err != nil {
		return nil, err
	}
	return &rate, nil
}

func (r *Repository) ListRates(currencyID uint, companyID uint) ([]CurrencyRate, error) {
	var list []CurrencyRate
	query := r.db.Where("currency_id = ?", currencyID)
	if companyID > 0 {
		query = query.Where("company_id = ?", companyID)
	}
	err := query.Order("name desc, id desc").Find(&list).Error
	return list, err
}

func (r *Repository) CreateRevaluation(rev *ForexRevaluation) error {
	return r.db.Create(rev).Error
}

func (r *Repository) GetRevaluationByID(id uint, companyID uint) (*ForexRevaluation, error) {
	var rev ForexRevaluation
	query := r.db.Preload("Currency").
		Preload("JournalEntry").
		Preload("Lines.Partner").
		Where("id = ?", id)
	if companyID > 0 {
		query = query.Where("company_id = ?", companyID)
	}
	err := query.First(&rev).Error
	if err != nil {
		return nil, err
	}
	return &rev, nil
}

func (r *Repository) ListRevaluations(companyID uint, state string) ([]ForexRevaluation, error) {
	var list []ForexRevaluation
	query := r.db.Preload("Currency").Preload("JournalEntry")
	if companyID > 0 {
		query = query.Where("company_id = ?", companyID)
	}
	if state != "" {
		query = query.Where("state = ?", state)
	}
	err := query.Order("revaluation_date desc, id desc").Find(&list).Error
	return list, err
}

func (r *Repository) UpdateRevaluation(rev *ForexRevaluation) error {
	return r.db.Save(rev).Error
}

func (r *Repository) GetNextRevaluationSequence(companyID uint) string {
	now := time.Now()
	prefix := fmt.Sprintf("REV/%d/%02d", now.Year(), now.Month())

	var count int64
	r.db.Model(&ForexRevaluation{}).
		Where("company_id = ? AND name LIKE ?", companyID, prefix+"%").
		Count(&count)

	return fmt.Sprintf("%s/%05d", prefix, count+1)
}

func (r *Repository) GetStats(companyID uint) (*CurrencyStats, error) {
	stats := &CurrencyStats{}

	r.db.Model(&Currency{}).Where("active = ?", true).Count(&stats.TotalCurrencies)
	r.db.Model(&CurrencyRate{}).Where("company_id = ?", companyID).Count(&stats.TotalRatesRecorded)
	r.db.Model(&ForexRevaluation{}).Where("company_id = ? AND state = ?", companyID, "posted").Count(&stats.TotalRevaluationsDone)

	// Sum YTD Unrealized Net Gain/Loss
	var netSum float64
	r.db.Model(&ForexRevaluation{}).
		Where("company_id = ? AND state = ?", companyID, "posted").
		Select("COALESCE(SUM(net_gain_loss), 0)").
		Scan(&netSum)
	stats.YTDUnrealizedGainLoss = netSum

	// Fetch latest USD and EUR spot rate
	var usdRate CurrencyRate
	if err := r.db.Joins("JOIN res_currencies ON res_currencies.id = res_currency_rates.currency_id").
		Where("res_currencies.name = ? AND res_currency_rates.company_id = ?", "USD", companyID).
		Order("res_currency_rates.name desc").First(&usdRate).Error; err == nil {
		stats.LatestUSDRate = usdRate.DirectRate
	}

	var eurRate CurrencyRate
	if err := r.db.Joins("JOIN res_currencies ON res_currencies.id = res_currency_rates.currency_id").
		Where("res_currencies.name = ? AND res_currency_rates.company_id = ?", "EUR", companyID).
		Order("res_currency_rates.name desc").First(&eurRate).Error; err == nil {
		stats.LatestEURRate = eurRate.DirectRate
	}

	return stats, nil
}
