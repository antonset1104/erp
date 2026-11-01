package attendance

import (
	"time"

	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// Shifts
func (r *Repository) CreateShift(shift *Shift) error {
	return r.db.Create(shift).Error
}

func (r *Repository) GetShiftByID(id uint) (*Shift, error) {
	var shift Shift
	if err := r.db.First(&shift, id).Error; err != nil {
		return nil, err
	}
	return &shift, nil
}

func (r *Repository) ListShifts(companyID uint) ([]Shift, error) {
	var shifts []Shift
	query := r.db.Model(&Shift{})
	if companyID > 0 {
		query = query.Where("company_id = ?", companyID)
	}
	if err := query.Order("id ASC").Find(&shifts).Error; err != nil {
		return nil, err
	}
	return shifts, nil
}

// Attendances
func (r *Repository) CreateAttendance(att *Attendance) error {
	return r.db.Create(att).Error
}

func (r *Repository) UpdateAttendance(att *Attendance) error {
	return r.db.Save(att).Error
}

func (r *Repository) DeleteAttendance(id uint) error {
	return r.db.Delete(&Attendance{}, id).Error
}

func (r *Repository) GetAttendanceByID(id uint) (*Attendance, error) {
	var att Attendance
	if err := r.db.Preload("Employee").Preload("Employee.Department").Preload("Shift").First(&att, id).Error; err != nil {
		return nil, err
	}
	return &att, nil
}

// GetActiveAttendanceByEmployee finds the currently open check-in session for an employee
func (r *Repository) GetActiveAttendanceByEmployee(employeeID uint) (*Attendance, error) {
	var att Attendance
	err := r.db.Preload("Employee").Preload("Shift").
		Where("employee_id = ? AND check_out IS NULL", employeeID).
		Order("id DESC").
		First(&att).Error
	if err != nil {
		return nil, err
	}
	return &att, nil
}

func (r *Repository) ListAttendances(companyID uint, filter AttendanceFilter) ([]Attendance, int64, error) {
	var attendances []Attendance
	var total int64

	query := r.db.Model(&Attendance{}).
		Preload("Employee").
		Preload("Employee.Department").
		Preload("Shift")

	if companyID > 0 {
		query = query.Where("company_id = ?", companyID)
	}
	if filter.EmployeeID > 0 {
		query = query.Where("employee_id = ?", filter.EmployeeID)
	}
	if filter.DepartmentID > 0 {
		query = query.Joins("JOIN hr_employees ON hr_employees.id = hr_attendances.employee_id").
			Where("hr_employees.department_id = ?", filter.DepartmentID)
	}
	if filter.DateFrom != "" {
		query = query.Where("date >= ?", filter.DateFrom)
	}
	if filter.DateTo != "" {
		query = query.Where("date <= ?", filter.DateTo)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.AnomalyOnly {
		query = query.Where("anomaly_flag = ?", true)
	}
	if filter.Search != "" {
		term := "%" + filter.Search + "%"
		query = query.Joins("LEFT JOIN hr_employees ON hr_employees.id = hr_attendances.employee_id").
			Where("hr_employees.name LIKE ? OR hr_attendances.in_address LIKE ? OR hr_attendances.ip_address LIKE ?", term, term, term)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	query = query.Order("check_in DESC").Limit(limit).Offset(filter.Offset)

	if err := query.Find(&attendances).Error; err != nil {
		return nil, 0, err
	}

	return attendances, total, nil
}

func (r *Repository) GetAttendancesByEmployeeAndDateRange(employeeID uint, startDate, endDate string) ([]Attendance, error) {
	var attendances []Attendance
	query := r.db.Model(&Attendance{}).
		Preload("Employee").
		Preload("Shift").
		Where("employee_id = ?", employeeID)

	if startDate != "" {
		query = query.Where("date >= ?", startDate)
	}
	if endDate != "" {
		query = query.Where("date <= ?", endDate)
	}

	if err := query.Order("date ASC, check_in ASC").Find(&attendances).Error; err != nil {
		return nil, err
	}
	return attendances, nil
}

func (r *Repository) GetStats(companyID uint) (*AttendanceStats, error) {
	todayStr := time.Now().Format("2006-01-02")
	stats := &AttendanceStats{}

	// Total records today
	r.db.Model(&Attendance{}).
		Where("company_id = ? AND date = ?", companyID, todayStr).
		Count(&stats.TotalToday)

	// Currently checked in (active session today or overall)
	r.db.Model(&Attendance{}).
		Where("company_id = ? AND check_out IS NULL", companyID).
		Count(&stats.CurrentlyCheckedIn)

	// Late today
	r.db.Model(&Attendance{}).
		Where("company_id = ? AND date = ? AND (late_minutes > 0 OR status = 'late')", companyID, todayStr).
		Count(&stats.LateToday)

	// Anomaly count total
	r.db.Model(&Attendance{}).
		Where("company_id = ? AND anomaly_flag = ?", companyID, true).
		Count(&stats.AnomalyCount)

	// Sum Overtime and Average Worked Hours (over recent 30 days)
	type AggResult struct {
		TotalOvertime float64
		AvgWorked     float64
	}
	var agg AggResult
	r.db.Model(&Attendance{}).
		Select("COALESCE(SUM(overtime_hours), 0) as total_overtime, COALESCE(AVG(worked_hours), 0) as avg_worked").
		Where("company_id = ? AND worked_hours > 0", companyID).
		Scan(&agg)

	stats.TotalOvertimeHours = agg.TotalOvertime
	stats.AverageWorkedHours = agg.AvgWorked

	return stats, nil
}
