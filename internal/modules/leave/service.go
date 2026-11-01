package leave

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// ----------------- Leave Types -----------------

func (s *Service) ListLeaveTypes(companyID uint) ([]LeaveType, error) {
	var list []LeaveType
	err := s.db.Where("company_id = ?", companyID).Order("id asc").Find(&list).Error
	return list, err
}

func (s *Service) CreateLeaveType(lt *LeaveType) error {
	if lt.Name == "" || lt.Code == "" {
		return errors.New("leave type name and code are required")
	}
	return s.db.Create(lt).Error
}

// ----------------- Leave Allocations -----------------

func (s *Service) ListAllocations(companyID, employeeID uint, year int) ([]LeaveAllocation, error) {
	var list []LeaveAllocation
	q := s.db.Where("company_id = ?", companyID)
	if employeeID > 0 {
		q = q.Where("employee_id = ?", employeeID)
	}
	if year > 0 {
		q = q.Where("year = ?", year)
	}
	err := q.Preload("Employee").
		Preload("LeaveType").
		Order("year desc, employee_id asc").
		Find(&list).Error
	return list, err
}

func (s *Service) CreateAllocation(alloc *LeaveAllocation, companyID, userID uint) error {
	if alloc.EmployeeID == 0 {
		return errors.New("employee is required")
	}
	if alloc.LeaveTypeID == 0 {
		return errors.New("leave type is required")
	}
	if alloc.AllocatedDays <= 0 {
		return errors.New("allocated days must be greater than 0")
	}

	if alloc.Year == 0 {
		alloc.Year = time.Now().Year()
	}
	if alloc.RemainingDays == 0 {
		alloc.RemainingDays = alloc.AllocatedDays - alloc.UsedDays
	}
	if alloc.State == "" {
		alloc.State = "approved"
	}

	alloc.CompanyID = companyID
	alloc.CreateUID = userID
	alloc.WriteUID = userID

	return s.db.Create(alloc).Error
}

func (s *Service) ApproveAllocation(id, companyID, userID uint) (*LeaveAllocation, error) {
	var alloc LeaveAllocation
	err := s.db.Where("id = ? AND company_id = ?", id, companyID).First(&alloc).Error
	if err != nil {
		return nil, errors.New("leave allocation not found")
	}

	alloc.State = "approved"
	alloc.WriteUID = userID
	if err := s.db.Save(&alloc).Error; err != nil {
		return nil, err
	}
	return &alloc, nil
}

// ----------------- Leave Requests -----------------

func (s *Service) ListLeaveRequests(companyID, employeeID uint, state, typeCode string) ([]LeaveRequest, error) {
	var list []LeaveRequest
	q := s.db.Where("company_id = ?", companyID)
	if employeeID > 0 {
		q = q.Where("employee_id = ?", employeeID)
	}
	if state != "" && state != "all" {
		q = q.Where("state = ?", state)
	}
	if typeCode != "" && typeCode != "all" {
		q = q.Joins("JOIN leave_types ON leave_types.id = leave_requests.leave_type_id").
			Where("leave_types.code = ?", typeCode)
	}

	err := q.Preload("Employee").
		Preload("Employee.Department").
		Preload("LeaveType").
		Preload("Approver").
		Order("id desc").
		Find(&list).Error
	return list, err
}

func (s *Service) GetLeaveRequest(id, companyID uint) (*LeaveRequest, error) {
	var req LeaveRequest
	err := s.db.Where("id = ? AND company_id = ?", id, companyID).
		Preload("Employee").
		Preload("Employee.Department").
		Preload("LeaveType").
		Preload("Approver").
		First(&req).Error
	if err != nil {
		return nil, err
	}
	return &req, nil
}

func (s *Service) CreateLeaveRequest(req *LeaveRequest, companyID, userID uint) error {
	if req.EmployeeID == 0 {
		return errors.New("employee is required")
	}
	if req.LeaveTypeID == 0 {
		return errors.New("leave type is required")
	}
	if req.DateFrom == "" || req.DateTo == "" {
		return errors.New("date from and date to are required")
	}
	if req.NumberOfDays <= 0 {
		return errors.New("number of days must be greater than 0")
	}

	var lt LeaveType
	if err := s.db.First(&lt, req.LeaveTypeID).Error; err != nil {
		return errors.New("leave type not found")
	}

	// If leave type requires allocation, check available balance
	if lt.RequiresAllocation {
		year := time.Now().Year()
		var alloc LeaveAllocation
		err := s.db.Where("employee_id = ? AND leave_type_id = ? AND year = ? AND state = 'approved'",
			req.EmployeeID, req.LeaveTypeID, year).First(&alloc).Error
		if err != nil {
			return fmt.Errorf("tidak ada alokasi cuti aktif untuk tahun %d", year)
		}
		if alloc.RemainingDays < req.NumberOfDays {
			return fmt.Errorf("sisa jatah cuti tidak mencukupi (sisa: %.1f hari, diajukan: %.1f hari)", alloc.RemainingDays, req.NumberOfDays)
		}
	}

	year := time.Now().Year()
	if req.RequestNumber == "" {
		var count int64
		s.db.Model(&LeaveRequest{}).Count(&count)
		req.RequestNumber = fmt.Sprintf("LV/%d/%05d", year, count+1)
	}

	if req.State == "" {
		req.State = "confirm" // default submitted / awaiting validation
	}

	req.CompanyID = companyID
	req.CreateUID = userID
	req.WriteUID = userID

	return s.db.Create(req).Error
}

func (s *Service) ApproveLeaveRequest(id, companyID, approverUserID uint) (*LeaveRequest, error) {
	var req LeaveRequest
	err := s.db.Where("id = ? AND company_id = ?", id, companyID).
		Preload("LeaveType").
		First(&req).Error
	if err != nil {
		return nil, errors.New("leave request not found")
	}

	if req.State == "validate" {
		return nil, errors.New("leave request is already approved")
	}

	// Atomically deduct allocation if required
	if req.LeaveType != nil && req.LeaveType.RequiresAllocation {
		year := time.Now().Year()
		var alloc LeaveAllocation
		err := s.db.Where("employee_id = ? AND leave_type_id = ? AND year = ? AND state = 'approved'",
			req.EmployeeID, req.LeaveTypeID, year).First(&alloc).Error
		if err == nil {
			if alloc.RemainingDays < req.NumberOfDays {
				return nil, fmt.Errorf("sisa jatah cuti tidak mencukupi untuk persetujuan (sisa: %.1f hari)", alloc.RemainingDays)
			}
			alloc.UsedDays += req.NumberOfDays
			alloc.RemainingDays = alloc.AllocatedDays - alloc.UsedDays
			alloc.WriteUID = approverUserID
			s.db.Save(&alloc)
		}
	}

	nowStr := time.Now().Format("2006-01-02 15:04:05")
	req.State = "validate"
	req.ApproverID = &approverUserID
	req.ApprovalDate = nowStr
	req.WriteUID = approverUserID

	if err := s.db.Save(&req).Error; err != nil {
		return nil, err
	}

	return &req, nil
}

func (s *Service) RefuseLeaveRequest(id, companyID, approverUserID uint, reason string) (*LeaveRequest, error) {
	var req LeaveRequest
	err := s.db.Where("id = ? AND company_id = ?", id, companyID).
		Preload("LeaveType").
		First(&req).Error
	if err != nil {
		return nil, errors.New("leave request not found")
	}

	// If it was already approved, return back the deducted allocation
	if req.State == "validate" && req.LeaveType != nil && req.LeaveType.RequiresAllocation {
		year := time.Now().Year()
		var alloc LeaveAllocation
		err := s.db.Where("employee_id = ? AND leave_type_id = ? AND year = ?",
			req.EmployeeID, req.LeaveTypeID, year).First(&alloc).Error
		if err == nil {
			alloc.UsedDays -= req.NumberOfDays
			if alloc.UsedDays < 0 {
				alloc.UsedDays = 0
			}
			alloc.RemainingDays = alloc.AllocatedDays - alloc.UsedDays
			alloc.WriteUID = approverUserID
			s.db.Save(&alloc)
		}
	}

	req.State = "refuse"
	req.ApproverID = &approverUserID
	req.RefusalReason = reason
	req.WriteUID = approverUserID

	if err := s.db.Save(&req).Error; err != nil {
		return nil, err
	}

	return &req, nil
}

// GetUnpaidLeaveDays calculates total days of approved unpaid leaves for an employee in a given month (e.g. "2026-10")
func (s *Service) GetUnpaidLeaveDays(employeeID uint, monthPrefix string, companyID uint) (float64, error) {
	var total float64
	err := s.db.Table("leave_requests").
		Joins("JOIN leave_types ON leave_types.id = leave_requests.leave_type_id").
		Where("leave_requests.company_id = ? AND leave_requests.employee_id = ? AND leave_requests.state = 'validate' AND leave_types.is_unpaid = ?", companyID, employeeID, true).
		Where("leave_requests.date_from LIKE ?", monthPrefix+"%").
		Select("COALESCE(SUM(leave_requests.number_of_days), 0)").
		Scan(&total).Error
	return total, err
}

func (s *Service) GetLeaveSummary(companyID, employeeID uint) (map[string]interface{}, error) {
	year := time.Now().Year()
	var totalAllocated, totalUsed, totalRemaining float64

	var allocs []LeaveAllocation
	s.db.Where("company_id = ? AND year = ? AND state = 'approved'", companyID, year).
		Where(map[string]interface{}{"employee_id": employeeID}).Find(&allocs)

	for _, a := range allocs {
		totalAllocated += a.AllocatedDays
		totalUsed += a.UsedDays
		totalRemaining += a.RemainingDays
	}

	var pendingCount int64
	s.db.Model(&LeaveRequest{}).Where("company_id = ? AND state = 'confirm'", companyID).Count(&pendingCount)

	var approvedCount int64
	s.db.Model(&LeaveRequest{}).Where("company_id = ? AND state = 'validate'", companyID).Count(&approvedCount)

	return map[string]interface{}{
		"total_allocated": totalAllocated,
		"total_used":      totalUsed,
		"total_remaining": totalRemaining,
		"pending_count":   pendingCount,
		"approved_count":  approvedCount,
	}, nil
}
