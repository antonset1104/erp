package payroll

import (
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// --- Structures & Rules ---

func (r *Repository) CreateStructure(structure *PayrollStructure) error {
	return r.db.Create(structure).Error
}

func (r *Repository) GetStructureByID(id uint) (*PayrollStructure, error) {
	var s PayrollStructure
	err := r.db.Preload("Rules", func(db *gorm.DB) *gorm.DB {
		return db.Order("sequence asc")
	}).First(&s, id).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *Repository) GetStructureByCode(code string) (*PayrollStructure, error) {
	var s PayrollStructure
	err := r.db.Preload("Rules", func(db *gorm.DB) *gorm.DB {
		return db.Order("sequence asc")
	}).Where("code = ?", code).First(&s).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *Repository) ListStructures(companyID uint) ([]PayrollStructure, error) {
	var items []PayrollStructure
	err := r.db.Preload("Rules", func(db *gorm.DB) *gorm.DB {
		return db.Order("sequence asc")
	}).Where("company_id = ?", companyID).Order("id asc").Find(&items).Error
	return items, err
}

func (r *Repository) CreateSalaryRule(rule *SalaryRule) error {
	return r.db.Create(rule).Error
}

func (r *Repository) ListSalaryRulesByStructure(structureID uint) ([]SalaryRule, error) {
	var rules []SalaryRule
	err := r.db.Where("structure_id = ?", structureID).Order("sequence asc").Find(&rules).Error
	return rules, err
}

// --- Payslip Runs (Batches) ---

func (r *Repository) CreatePayslipRun(run *PayslipRun) error {
	return r.db.Create(run).Error
}

func (r *Repository) GetPayslipRunByID(id uint) (*PayslipRun, error) {
	var run PayslipRun
	err := r.db.Preload("Payslips.Employee").First(&run, id).Error
	if err != nil {
		return nil, err
	}
	return &run, nil
}

func (r *Repository) UpdatePayslipRun(run *PayslipRun) error {
	return r.db.Save(run).Error
}

func (r *Repository) ListPayslipRuns(companyID uint) ([]PayslipRun, error) {
	var items []PayslipRun
	err := r.db.Where("company_id = ?", companyID).Order("id desc").Find(&items).Error
	return items, err
}

// --- Payslips ---

func (r *Repository) CreatePayslip(slip *Payslip) error {
	return r.db.Create(slip).Error
}

func (r *Repository) GetPayslipByID(id uint) (*Payslip, error) {
	var slip Payslip
	err := r.db.Preload("Employee.Department").
		Preload("Employee.Job").
		Preload("Structure.Rules", func(db *gorm.DB) *gorm.DB {
			return db.Order("sequence asc")
		}).
		Preload("Lines", func(db *gorm.DB) *gorm.DB {
			return db.Order("sequence asc")
		}).
		First(&slip, id).Error
	if err != nil {
		return nil, err
	}
	return &slip, nil
}

func (r *Repository) UpdatePayslip(slip *Payslip) error {
	return r.db.Save(slip).Error
}

func (r *Repository) ListPayslips(companyID uint, employeeID uint, runID uint, period string, state string) ([]Payslip, error) {
	var slips []Payslip
	query := r.db.Preload("Employee.Department").
		Preload("Employee.Job").
		Preload("Structure").
		Where("company_id = ?", companyID)

	if employeeID > 0 {
		query = query.Where("employee_id = ?", employeeID)
	}
	if runID > 0 {
		query = query.Where("payslip_run_id = ?", runID)
	}
	if period != "" {
		query = query.Where("period = ?", period)
	}
	if state != "" {
		query = query.Where("state = ?", state)
	}

	err := query.Order("id desc").Find(&slips).Error
	return slips, err
}

func (r *Repository) SavePayslipLines(slipID uint, lines []PayslipLine) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Clear existing lines
		if err := tx.Where("payslip_id = ?", slipID).Delete(&PayslipLine{}).Error; err != nil {
			return err
		}
		// Insert new lines
		for i := range lines {
			lines[i].PayslipID = slipID
			if err := tx.Create(&lines[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// --- Statistics ---

func (r *Repository) GetStats(companyID uint) (*PayrollStats, error) {
	var stats PayrollStats

	// Count runs
	r.db.Model(&PayslipRun{}).Where("company_id = ?", companyID).Count(&stats.TotalRuns)

	// Count payslips
	r.db.Model(&Payslip{}).Where("company_id = ?", companyID).Count(&stats.TotalPayslips)
	r.db.Model(&Payslip{}).Where("company_id = ? AND state = 'paid'", companyID).Count(&stats.TotalPaidSlips)
	r.db.Model(&Payslip{}).Where("company_id = ? AND state = 'draft'", companyID).Count(&stats.TotalDraftSlips)
	r.db.Model(&Payslip{}).Where("company_id = ? AND state IN ('verify', 'done')", companyID).Count(&stats.TotalConfirmedSlips)
	r.db.Model(&Payslip{}).Where("company_id = ? AND ai_check_status IN ('pending', 'warning')", companyID).Count(&stats.PendingAuditCount)

	// Sum totals
	type AggResult struct {
		Gross     float64
		Deduction float64
		Net       float64
		Overtime  float64
	}
	var agg AggResult
	r.db.Model(&Payslip{}).
		Where("company_id = ?", companyID).
		Select("COALESCE(SUM(gross_salary), 0) as gross, COALESCE(SUM(total_deductions), 0) as deduction, COALESCE(SUM(net_salary), 0) as net, COALESCE(SUM(overtime_hours), 0) as overtime").
		Scan(&agg)

	stats.TotalGrossExpenditure = agg.Gross
	stats.TotalDeductionsWithheld = agg.Deduction
	stats.TotalNetPaid = agg.Net
	stats.TotalOvertimeHours = agg.Overtime

	return &stats, nil
}
