package maintenance

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

// ----------------- Category Operations -----------------

func (s *Service) CreateCategory(cat *MaintenanceEquipmentCategory) error {
	if cat.Name == "" {
		return errors.New("category name is required")
	}
	return s.db.Create(cat).Error
}

func (s *Service) ListCategories(companyID uint) ([]MaintenanceEquipmentCategory, error) {
	var cats []MaintenanceEquipmentCategory
	err := s.db.Where("company_id = ?", companyID).
		Preload("Technician").
		Order("id asc").
		Find(&cats).Error
	return cats, err
}

// ----------------- Equipment Operations -----------------

func (s *Service) CreateEquipment(eq *MaintenanceEquipment) error {
	if eq.Name == "" {
		return errors.New("equipment name is required")
	}

	if eq.Status == "" {
		eq.Status = "operational"
	}
	if eq.MaintenancePeriodDays <= 0 {
		eq.MaintenancePeriodDays = 30
	}

	year := time.Now().Year()
	if eq.EquipmentNumber == "" {
		var count int64
		s.db.Model(&MaintenanceEquipment{}).Count(&count)
		eq.EquipmentNumber = fmt.Sprintf("EQP/%d/%05d", year, count+1)
	}

	if eq.NextActionDate == "" {
		eq.NextActionDate = time.Now().AddDate(0, 0, eq.MaintenancePeriodDays).Format("2006-01-02")
	}

	return s.db.Create(eq).Error
}

func (s *Service) ListEquipments(companyID, categoryID uint, status string) ([]MaintenanceEquipment, error) {
	var list []MaintenanceEquipment
	q := s.db.Where("company_id = ?", companyID)

	if categoryID > 0 {
		q = q.Where("category_id = ?", categoryID)
	}
	if status != "" && status != "all" {
		q = q.Where("status = ?", status)
	}

	err := q.Preload("Category").
		Preload("Partner").
		Preload("Asset").
		Preload("Department").
		Preload("Employee").
		Preload("Technician").
		Order("id desc").
		Find(&list).Error
	return list, err
}

func (s *Service) GetEquipment(id, companyID uint) (*MaintenanceEquipment, error) {
	var eq MaintenanceEquipment
	err := s.db.Where("id = ? AND company_id = ?", id, companyID).
		Preload("Category").
		Preload("Partner").
		Preload("Asset").
		Preload("Department").
		Preload("Employee").
		Preload("Technician").
		First(&eq).Error
	if err != nil {
		return nil, err
	}
	return &eq, nil
}

func (s *Service) UpdateEquipment(eq *MaintenanceEquipment) error {
	return s.db.Save(eq).Error
}

// ----------------- Maintenance Request Operations -----------------

func (s *Service) CreateRequest(req *MaintenanceRequest) error {
	if req.Name == "" {
		return errors.New("request subject / problem summary is required")
	}
	if req.EquipmentID == 0 {
		return errors.New("equipment is required")
	}

	if req.MaintenanceType == "" {
		req.MaintenanceType = "corrective"
	}
	if req.Priority == "" {
		req.Priority = "1"
	}
	if req.Stage == "" {
		req.Stage = "new"
	}
	if req.RequestDate == "" {
		req.RequestDate = time.Now().Format("2006-01-02")
	}

	year := time.Now().Year()
	if req.RequestNumber == "" {
		var count int64
		s.db.Model(&MaintenanceRequest{}).Count(&count)
		req.RequestNumber = fmt.Sprintf("REQ/MAINT/%d/%05d", year, count+1)
	}

	// If corrective breakdown, update equipment status to maintenance
	if req.MaintenanceType == "corrective" {
		s.db.Model(&MaintenanceEquipment{}).Where("id = ?", req.EquipmentID).Update("status", "maintenance")
	}

	return s.db.Create(req).Error
}

func (s *Service) ListRequests(companyID, equipmentID uint, stage, mType, priority string) ([]MaintenanceRequest, error) {
	var list []MaintenanceRequest
	q := s.db.Where("company_id = ?", companyID)

	if equipmentID > 0 {
		q = q.Where("equipment_id = ?", equipmentID)
	}
	if stage != "" && stage != "all" {
		q = q.Where("stage = ?", stage)
	}
	if mType != "" && mType != "all" {
		q = q.Where("maintenance_type = ?", mType)
	}
	if priority != "" && priority != "all" {
		q = q.Where("priority = ?", priority)
	}

	err := q.Preload("Equipment").
		Preload("Equipment.Category").
		Preload("User").
		Order("id desc").
		Find(&list).Error
	return list, err
}

func (s *Service) GetRequest(id, companyID uint) (*MaintenanceRequest, error) {
	var req MaintenanceRequest
	err := s.db.Where("id = ? AND company_id = ?", id, companyID).
		Preload("Equipment").
		Preload("Equipment.Category").
		Preload("User").
		First(&req).Error
	if err != nil {
		return nil, err
	}
	return &req, nil
}

func (s *Service) UpdateRequestStage(id, companyID, userID uint, stage string) (*MaintenanceRequest, error) {
	req, err := s.GetRequest(id, companyID)
	if err != nil {
		return nil, errors.New("maintenance request not found")
	}

	req.Stage = stage
	req.WriteUID = userID
	if stage == "repaired" && req.CloseDate == "" {
		req.CloseDate = time.Now().Format("2006-01-02 15:04:05")
	}

	if err := s.db.Save(req).Error; err != nil {
		return nil, err
	}
	return req, nil
}

func (s *Service) AssignRequest(id, companyID, userID, technicianID uint) (*MaintenanceRequest, error) {
	req, err := s.GetRequest(id, companyID)
	if err != nil {
		return nil, errors.New("maintenance request not found")
	}

	req.UserID = &technicianID
	req.WriteUID = userID
	if req.Stage == "new" {
		req.Stage = "in_progress"
	}

	if err := s.db.Save(req).Error; err != nil {
		return nil, err
	}
	return s.GetRequest(id, companyID)
}

func (s *Service) CompleteRequest(id, companyID, userID uint, resolution string, durationHours float64) (*MaintenanceRequest, error) {
	req, err := s.GetRequest(id, companyID)
	if err != nil {
		return nil, errors.New("maintenance request not found")
	}

	req.Stage = "repaired"
	req.Resolution = resolution
	req.DurationHours = durationHours
	req.CloseDate = time.Now().Format("2006-01-02 15:04:05")
	req.WriteUID = userID

	if err := s.db.Save(req).Error; err != nil {
		return nil, err
	}

	// Restore equipment status to operational and advance next action date
	var eq MaintenanceEquipment
	if err := s.db.First(&eq, req.EquipmentID).Error; err == nil {
		eq.Status = "operational"
		days := eq.MaintenancePeriodDays
		if days <= 0 {
			days = 30
		}
		eq.NextActionDate = time.Now().AddDate(0, 0, days).Format("2006-01-02")
		eq.WriteUID = userID
		s.db.Save(&eq)
	}

	return s.GetRequest(id, companyID)
}

func (s *Service) AutoSchedulePreventive(companyID, userID uint) ([]MaintenanceRequest, error) {
	today := time.Now().Format("2006-01-02")
	var dueEquipments []MaintenanceEquipment
	err := s.db.Where("company_id = ? AND status != 'scrapped' AND (next_action_date <= ? OR next_action_date IS NULL OR next_action_date = '')", companyID, today).Find(&dueEquipments).Error
	if err != nil {
		return nil, err
	}

	var created []MaintenanceRequest
	for _, eq := range dueEquipments {
		req := MaintenanceRequest{
			Name:            fmt.Sprintf("Pemeriksaan & Servis Rutin Berkala (%s)", eq.Name),
			EquipmentID:     eq.ID,
			MaintenanceType: "preventive",
			Priority:        "1",
			Stage:           "new",
			RequestDate:     today,
			ScheduleDate:    today,
			UserID:          eq.TechnicianUserID,
			Description:     fmt.Sprintf("Pengecekan preventif berkala tiap %d hari sesuai SOP pabrik.", eq.MaintenancePeriodDays),
		}
		req.CompanyID = companyID
		req.CreateUID = userID
		req.WriteUID = userID

		if err := s.CreateRequest(&req); err == nil {
			created = append(created, req)

			// Advance equipment next action date
			days := eq.MaintenancePeriodDays
			if days <= 0 {
				days = 30
			}
			eq.NextActionDate = time.Now().AddDate(0, 0, days).Format("2006-01-02")
			eq.WriteUID = userID
			s.db.Save(&eq)
		}
	}

	return created, nil
}
