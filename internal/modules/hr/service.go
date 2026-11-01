package hr

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

func (s *Service) ListDepartments(companyID uint) ([]Department, error) {
	var depts []Department
	if err := s.db.Where("company_id = ?", companyID).Order("name asc").Find(&depts).Error; err != nil {
		return nil, err
	}
	return depts, nil
}

func (s *Service) CreateDepartment(dept *Department) error {
	return s.db.Create(dept).Error
}

func (s *Service) ListJobs(companyID uint) ([]JobPosition, error) {
	var jobs []JobPosition
	if err := s.db.Preload("Department").Where("company_id = ?", companyID).Order("name asc").Find(&jobs).Error; err != nil {
		return nil, err
	}
	return jobs, nil
}

func (s *Service) CreateJob(job *JobPosition) error {
	return s.db.Create(job).Error
}

func (s *Service) ListEmployees(companyID uint) ([]Employee, error) {
	var employees []Employee
	if err := s.db.Preload("Department").Preload("Job").
		Where("company_id = ? AND status = 'active'", companyID).
		Order("name asc").Find(&employees).Error; err != nil {
		return nil, err
	}
	return employees, nil
}

func (s *Service) CreateEmployee(emp *Employee) error {
	return s.db.Create(emp).Error
}

func (s *Service) ListPayslips(companyID uint, period string) ([]Payslip, error) {
	var slips []Payslip
	query := s.db.Preload("Employee.Department").Preload("Employee.Job").Where("company_id = ?", companyID)
	if period != "" {
		query = query.Where("period = ?", period)
	}
	if err := query.Order("id desc").Find(&slips).Error; err != nil {
		return nil, err
	}
	return slips, nil
}

func (s *Service) GeneratePayslip(employeeID uint, period string, allowances, deductions float64, note string, companyID uint) (*Payslip, error) {
	var emp Employee
	if err := s.db.Where("id = ? AND company_id = ?", employeeID, companyID).First(&emp).Error; err != nil {
		return nil, errors.New("employee not found")
	}

	var count int64
	s.db.Model(&Payslip{}).Count(&count)
	name := fmt.Sprintf("SLIP/%s/%05d", time.Now().Format("2006/01"), count+1)

	net := emp.BasicSalary + allowances - deductions

	slip := &Payslip{
		Name:        name,
		EmployeeID:  emp.ID,
		Period:      period,
		Date:        time.Now().Format("2006-01-02"),
		BasicSalary: emp.BasicSalary,
		Allowances:  allowances,
		Deductions:  deductions,
		NetSalary:   net,
		State:       "draft",
		Note:        note,
	}
	slip.CompanyID = companyID

	if err := s.db.Create(slip).Error; err != nil {
		return nil, err
	}

	return slip, nil
}

func (s *Service) ConfirmPayslip(id uint, companyID uint) (*Payslip, error) {
	var slip Payslip
	if err := s.db.Where("id = ? AND company_id = ?", id, companyID).First(&slip).Error; err != nil {
		return nil, errors.New("payslip not found")
	}
	if slip.State != "draft" {
		return nil, errors.New("payslip already confirmed or paid")
	}

	slip.State = "confirmed"
	if err := s.db.Save(&slip).Error; err != nil {
		return nil, err
	}
	return &slip, nil
}

func (s *Service) PayPayslip(id uint, companyID uint) (*Payslip, error) {
	var slip Payslip
	if err := s.db.Where("id = ? AND company_id = ?", id, companyID).First(&slip).Error; err != nil {
		return nil, errors.New("payslip not found")
	}
	if slip.State == "paid" {
		return nil, errors.New("payslip already paid")
	}

	slip.State = "paid"
	if err := s.db.Save(&slip).Error; err != nil {
		return nil, err
	}
	return &slip, nil
}
