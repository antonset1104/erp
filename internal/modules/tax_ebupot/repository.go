package tax_ebupot

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

type Repository interface {
	GeneratePeriodCode(month, year, revision int) string
	GenerateSlipNumber(year, month int, slipType string) (string, error)

	CreatePeriod(period *EbupotPeriod) error
	GetPeriod(id uint) (*EbupotPeriod, error)
	GetPeriodByMonthYear(month, year, rev int, companyID uint) (*EbupotPeriod, error)
	ListPeriods(companyID uint) ([]EbupotPeriod, error)
	UpdatePeriod(period *EbupotPeriod) error

	CreateWithholdingSlip(slip *WithholdingSlip) error
	GetWithholdingSlip(id uint) (*WithholdingSlip, error)
	ListWithholdingSlips(periodID uint, slipType string) ([]WithholdingSlip, error)
	UpdateWithholdingSlip(slip *WithholdingSlip) error
	DeleteWithholdingSlip(id uint) error

	CreateExportLog(log *EbupotExportLog) error
	ListExportLogs(periodID uint) ([]EbupotExportLog, error)

	GetStats(companyID uint) (*EbupotStats, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) GeneratePeriodCode(month, year, revision int) string {
	return fmt.Sprintf("SPT21/%04d/%02d/%d", year, month, revision)
}

func (r *repository) GenerateSlipNumber(year, month int, slipType string) (string, error) {
	prefix := "BP21"
	if strings.Contains(slipType, "pasal_26") {
		prefix = "BP26"
	} else if strings.Contains(slipType, "final") {
		prefix = "BPF21"
	}

	var count int64
	r.db.Model(&WithholdingSlip{}).
		Where("slip_number LIKE ?", fmt.Sprintf("%s/%04d/%02d/%%", prefix, year, month)).
		Count(&count)

	return fmt.Sprintf("%s/%04d/%02d/%05d", prefix, year, month, count+1), nil
}

func (r *repository) CreatePeriod(period *EbupotPeriod) error {
	return r.db.Create(period).Error
}

func (r *repository) GetPeriod(id uint) (*EbupotPeriod, error) {
	var period EbupotPeriod
	err := r.db.Preload("WithholdingSlips").First(&period, id).Error
	if err != nil {
		return nil, err
	}
	return &period, nil
}

func (r *repository) GetPeriodByMonthYear(month, year, rev int, companyID uint) (*EbupotPeriod, error) {
	var period EbupotPeriod
	err := r.db.Where("tax_month = ? AND tax_year = ? AND revision_number = ? AND company_id = ?",
		month, year, rev, companyID).First(&period).Error
	if err != nil {
		return nil, err
	}
	return &period, nil
}

func (r *repository) ListPeriods(companyID uint) ([]EbupotPeriod, error) {
	var periods []EbupotPeriod
	err := r.db.Where("company_id = ?", companyID).
		Order("tax_year DESC, tax_month DESC, revision_number DESC").
		Find(&periods).Error
	return periods, err
}

func (r *repository) UpdatePeriod(period *EbupotPeriod) error {
	return r.db.Save(period).Error
}

func (r *repository) CreateWithholdingSlip(slip *WithholdingSlip) error {
	return r.db.Create(slip).Error
}

func (r *repository) GetWithholdingSlip(id uint) (*WithholdingSlip, error) {
	var slip WithholdingSlip
	err := r.db.Preload("Employee").Preload("Partner").Preload("Payslip").First(&slip, id).Error
	if err != nil {
		return nil, err
	}
	return &slip, nil
}

func (r *repository) ListWithholdingSlips(periodID uint, slipType string) ([]WithholdingSlip, error) {
	var slips []WithholdingSlip
	q := r.db.Where("period_id = ?", periodID)
	if slipType != "" && slipType != "all" {
		q = q.Where("slip_type = ?", slipType)
	}
	err := q.Order("id ASC").Find(&slips).Error
	return slips, err
}

func (r *repository) UpdateWithholdingSlip(slip *WithholdingSlip) error {
	return r.db.Save(slip).Error
}

func (r *repository) DeleteWithholdingSlip(id uint) error {
	return r.db.Delete(&WithholdingSlip{}, id).Error
}

func (r *repository) CreateExportLog(log *EbupotExportLog) error {
	return r.db.Create(log).Error
}

func (r *repository) ListExportLogs(periodID uint) ([]EbupotExportLog, error) {
	var logs []EbupotExportLog
	err := r.db.Where("period_id = ?", periodID).Order("id DESC").Find(&logs).Error
	return logs, err
}

func (r *repository) GetStats(companyID uint) (*EbupotStats, error) {
	stats := &EbupotStats{}

	var periodCount int64
	r.db.Model(&EbupotPeriod{}).Where("company_id = ?", companyID).Count(&periodCount)
	stats.TotalPeriods = int(periodCount)

	var activePeriod EbupotPeriod
	if err := r.db.Where("company_id = ?", companyID).Order("tax_year DESC, tax_month DESC").First(&activePeriod).Error; err == nil {
		stats.ActivePeriodCode = activePeriod.PeriodCode
	}

	var totalSlips int64
	r.db.Model(&WithholdingSlip{}).Where("company_id = ?", companyID).Count(&totalSlips)
	stats.TotalWithholdingSlips = int(totalSlips)

	type SumResult struct {
		Gross float64
		Tax   float64
	}
	var sumRes SumResult
	r.db.Model(&WithholdingSlip{}).
		Where("company_id = ?", companyID).
		Select("COALESCE(SUM(gross_income), 0) as gross, COALESCE(SUM(tax_withheld), 0) as tax").
		Scan(&sumRes)

	stats.TotalGrossWages = sumRes.Gross
	stats.TotalTaxWithheld = sumRes.Tax

	var validNIK int64
	r.db.Model(&WithholdingSlip{}).
		Where("company_id = ? AND LENGTH(nik) = 16", companyID).
		Count(&validNIK)
	stats.ValidNIKCount = int(validNIK)

	var pendingCount int64
	r.db.Model(&WithholdingSlip{}).
		Where("company_id = ? AND status = 'draft'", companyID).
		Count(&pendingCount)
	stats.PendingReconciliation = int(pendingCount)

	return stats, nil
}
